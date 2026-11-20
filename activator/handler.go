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
	gorouterURL  string
}

func NewHandler(
	logger lager.Logger,
	parker Parker,
	engineClient ScalingEngineClient,
	watcher ReadinessWatcher,
	registry Registry,
	httpClient *http.Client,
	readyTimeout time.Duration,
	gorouterURL string,
) *Handler {
	client := *httpClient
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Handler{
		logger:       logger.Session("handler"),
		parker:       parker,
		engineClient: engineClient,
		watcher:      watcher,
		registry:     registry,
		httpClient:   &client,
		readyTimeout: readyTimeout,
		gorouterURL:  gorouterURL,
	}
}

// Park registers the app's routes with the activator as their NATS backend.
// Called by the scaling engine before it scales the app to zero.
func (h *Handler) Park(w http.ResponseWriter, _ *http.Request, vars map[string]string) {
	appID := vars["appid"]
	if err := h.parker.Park(context.Background(), appID); err != nil {
		h.logger.Error("park-failed", err, lager.Data{"appID": appID})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Unpark deregisters the app's routes from the activator's NATS backend.
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
	appID, target := h.routeForRequest(r)
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
	logger.Info("ready-forwarding", lager.Data{"target": target.String()})
	h.forward(w, r, target, logger)
}

// forward proxies the held request to targetURL (the woken app's route).
func (h *Handler) forward(w http.ResponseWriter, r *http.Request, targetURL *url.URL, logger lager.Logger) {
	// Dial only the operator-configured Gorouter ingress, not the app's DNS
	// name: even a CF-registered custom domain can resolve to an internal IP.
	// #nosec G704 -- The destination is operator-configured; request-derived values cannot change its authority.
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, h.gorouterURL, r.Body)
	if err != nil {
		logger.Error("build-forward-request-failed", err)
		writeRetry(w)
		return
	}
	outReq.Header = r.Header.Clone()
	outReq.Host = targetURL.Host
	// Copy only the path and query. The scheme and authority remain those of
	// the configured Gorouter even for absolute-form requests or forged headers.
	outReq.URL.Path = r.URL.Path
	outReq.URL.RawPath = r.URL.RawPath
	outReq.URL.RawQuery = r.URL.RawQuery

	// #nosec G704 -- Dial only the configured Gorouter; select a CF-registered route via Host, and never follow redirects.
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

// routeForRequest selects a route Host exclusively from routes discovered
// through CF, never using the request's authority as an outbound destination.
func (h *Handler) routeForRequest(r *http.Request) (string, *url.URL) {
	host := hostOnly(r.Host)
	if host == "" {
		return "", nil
	}
	for _, appID := range h.registry.ParkedApps() {
		for _, routeURL := range h.registry.RoutesFor(appID) {
			if !strings.Contains(routeURL, "://") {
				routeURL = "https://" + routeURL
			}
			target, err := url.Parse(routeURL)
			if err != nil || target.Scheme != "https" || target.User != nil || target.Host == "" {
				continue
			}
			if !strings.EqualFold(target.Hostname(), host) || !strings.HasPrefix(r.URL.Path, target.Path) {
				continue
			}
			return appID, target
		}
	}
	return "", nil
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

func writeRetry(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "5")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = fmt.Fprint(w, "app is waking up, please retry")
}
