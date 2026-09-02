package activator

import (
	"context"
	"fmt"
	"io"
	"net"
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

// HandleRouteService is Loop A. The activator is registered (via NATS) as a
// backend of the parked app's route, so a request for a zero-instance app lands
// here directly (on the app's Host header — there is NO X-CF-Forwarded-Url,
// unlike a route service). We hold the request, wake the app, wait for
// readiness, then forward to the app's real route. On timeout: 503 + Retry-After.
//
// Loop-break: while parked, the activator is still a registered backend for the
// app's host, so forwarding to https://<host> could loop back here. The
// ReadinessWatcher (Loop B) Unparks the app on the readiness Upsert — which
// deregisters the activator — so by the time `ready` fires, Gorouter has only
// the real app as a backend and the forward reaches it.
func (h *Handler) HandleRouteService(w http.ResponseWriter, r *http.Request) {
	appID := h.appForHost(r.Host)
	if appID == "" {
		h.logger.Info("no-parked-app-for-request", lager.Data{"host": r.Host})
		writeRetry(w)
		return
	}

	logger := h.logger.Session("wake", lager.Data{"appID": appID, "host": r.Host})
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

	// The app is up and Loop B has deregistered the activator for this host, so
	// forwarding to the app's own URL now reaches the real app, not the activator.
	target := requestURL(r)
	logger.Info("ready-forwarding", lager.Data{"target": target})
	h.forward(w, r, target, logger)
}

// requestURL reconstructs the absolute URL for the incoming (route-served)
// request from its Host and path.
func requestURL(r *http.Request) string {
	scheme := "https"
	if fproto := r.Header.Get("X-Forwarded-Proto"); fproto != "" {
		scheme = fproto
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.RequestURI())
}

// forward proxies the held request to targetURL (the woken app's route).
func (h *Handler) forward(w http.ResponseWriter, r *http.Request, targetURL string, logger lager.Logger) {
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
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

// appForHost matches the incoming request Host to a parked app by route host.
func (h *Handler) appForHost(reqHost string) string {
	host := hostOnly(reqHost)
	if host == "" {
		return ""
	}
	for _, appID := range h.registry.ParkedApps() {
		for _, routeURL := range h.registry.RoutesFor(appID) {
			if hostOf(routeURL) == host {
				return appID
			}
		}
	}
	return ""
}

// hostOnly strips any port from a request Host header.
func hostOnly(hostport string) string {
	if hostport == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
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
