package activator

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.cloudfoundry.org/lager/v3"
)

// Handler implements the activator's two request-facing surfaces:
//   - HandleRouteService: Loop A — a Gorouter-forwarded request for a parked
//     app; wake it, wait for readiness, forward the held request.
//   - Park / Unpark: the engine-facing control calls.
type Handler struct {
	logger       lager.Logger
	parker       Parker
	engineClient ScalingEngineClient
	watcher      ReadinessWatcher
	registry     Registry
	httpClient   *http.Client
	readyTimeout time.Duration
}

func NewHandler(
	logger lager.Logger,
	parker Parker,
	engineClient ScalingEngineClient,
	watcher ReadinessWatcher,
	registry Registry,
	httpClient *http.Client,
	readyTimeout time.Duration,
) *Handler {
	return &Handler{
		logger:       logger.Session("handler"),
		parker:       parker,
		engineClient: engineClient,
		watcher:      watcher,
		registry:     registry,
		httpClient:   httpClient,
		readyTimeout: readyTimeout,
	}
}

// Park binds the app's routes to the activator route-service. Called by the
// scaling engine before it scales the app to zero.
func (h *Handler) Park(w http.ResponseWriter, _ *http.Request, vars map[string]string) {
	appID := vars["appid"]
	if err := h.parker.Park(context.Background(), appID); err != nil {
		h.logger.Error("park-failed", err, lager.Data{"appID": appID})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Unpark unbinds the app's routes from the activator route-service.
func (h *Handler) Unpark(w http.ResponseWriter, _ *http.Request, vars map[string]string) {
	appID := vars["appid"]
	if err := h.parker.Unpark(context.Background(), appID); err != nil {
		h.logger.Error("unpark-failed", err, lager.Data{"appID": appID})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleRouteService is Loop A: hold the request, wake the app, wait for
// readiness (routing-api Upsert), then forward to X-CF-Forwarded-Url so
// Gorouter delivers it to the now-live app. On timeout: 503 + Retry-After.
func (h *Handler) HandleRouteService(w http.ResponseWriter, r *http.Request) {
	forwardedURL := r.Header.Get("X-CF-Forwarded-Url")
	appID := h.appForForwardedURL(forwardedURL)
	if appID == "" {
		// Not a parked app we know about; nothing we can do but signal retry.
		h.logger.Info("no-parked-app-for-request", lager.Data{"forwardedURL": forwardedURL})
		writeRetry(w)
		return
	}

	logger := h.logger.Session("wake", lager.Data{"appID": appID})
	logger.Info("waking")

	// Register the readiness waiter BEFORE triggering the scale-up, so an Upsert
	// that arrives quickly cannot be missed between ScaleUp and WaitForReady.
	ctx, cancel := context.WithTimeout(r.Context(), h.readyTimeout)
	defer cancel()
	ready := h.watcher.WaitForReady(ctx, appID)

	if err := h.engineClient.ScaleUp(r.Context(), appID); err != nil {
		logger.Error("scale-up-failed", err)
		writeRetry(w)
		return
	}

	if err := <-ready; err != nil {
		logger.Error("app-not-ready-in-time", err)
		writeRetry(w)
		return
	}

	logger.Info("ready-forwarding")
	h.forward(w, r, forwardedURL, logger)
}

// forward proxies the held request to forwardedURL, preserving the CF proxy
// headers so Gorouter delivers straight to the live app.
func (h *Handler) forward(w http.ResponseWriter, r *http.Request, forwardedURL string, logger lager.Logger) {
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, forwardedURL, r.Body)
	if err != nil {
		logger.Error("build-forward-request-failed", err)
		writeRetry(w)
		return
	}
	outReq.Header = r.Header.Clone()

	resp, err := h.httpClient.Do(outReq)
	if err != nil {
		logger.Error("forward-failed", err)
		writeRetry(w)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// appForForwardedURL matches the forwarded request URL to a parked app by host.
func (h *Handler) appForForwardedURL(forwardedURL string) string {
	if forwardedURL == "" {
		return ""
	}
	reqHost := hostOf(forwardedURL)
	if reqHost == "" {
		return ""
	}
	for _, appID := range h.registry.ParkedApps() {
		for _, routeURL := range h.registry.RoutesFor(appID) {
			if hostOf(routeURL) == reqHost {
				return appID
			}
		}
	}
	return ""
}

func hostOf(raw string) string {
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host
}

func writeRetry(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "5")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = fmt.Fprint(w, "app is waking up, please retry")
}
