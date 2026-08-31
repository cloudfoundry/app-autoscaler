package server

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"

	"code.cloudfoundry.org/lager/v3"
)

// Server hosts the activator's HTTP surfaces:
//   - the CF (route-service) server, which receives Gorouter-forwarded requests
//     for parked apps (Loop A entry point);
//   - the mTLS server, reserved for internal control endpoints;
//   - the health server.
type Server struct {
	logger       lager.Logger
	conf         *config.Config
	healthRouter *mux.Router
}

func NewServer(logger lager.Logger, conf *config.Config) *Server {
	return &Server{logger: logger, conf: conf}
}

// CreateCFServer serves the route-service endpoint. Gorouter forwards requests
// for parked (zero-instance) apps here, carrying X-CF-Forwarded-Url and the
// X-CF-Proxy-* headers. See docs/design/scale-to-zero.md §5.2 (Loop A).
func (s *Server) CreateCFServer(am auth.XFCCAuthMiddleware) (ifrit.Runner, error) {
	router := mux.NewRouter()
	router.Use(am.XFCCAuthenticationMiddleware)
	// Route service is a catch-all: it must accept any path/method the original
	// app would have served.
	router.PathPrefix("/").HandlerFunc(s.handleRouteService)
	return helpers.NewHTTPServer(s.logger.Session("cf-server"), s.conf.CFServer, router)
}

// CreateMtlsServer is reserved for internal control endpoints (none yet).
func (s *Server) CreateMtlsServer() (ifrit.Runner, error) {
	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	return helpers.NewHTTPServer(s.logger.Session("mtls-server"), s.conf.Server, router)
}

func (s *Server) CreateHealthServer() (ifrit.Runner, error) {
	if err := s.setupHealthRouter(); err != nil {
		return nil, err
	}
	return helpers.NewHTTPServer(s.logger.Session("health-server"), s.conf.Health.ServerConfig, s.healthRouter)
}

func (s *Server) setupHealthRouter() error {
	checkers := []healthendpoint.Checker{}
	gatherer := prometheus.NewRegistry()
	healthRouter, err := healthendpoint.NewHealthRouter(s.conf.Health, checkers, s.logger.Session("health"), gatherer, time.Now)
	if err != nil {
		return fmt.Errorf("failed to create health router: %w", err)
	}
	s.healthRouter = healthRouter
	return nil
}

// handleRouteService is the Loop A entry point. For the PoC scaffold it returns
// 503 + Retry-After (the documented cold-start-timeout response). Wiring the
// scale-up call + readiness wait + forward-to-X-CF-Forwarded-Url follows.
func (s *Server) handleRouteService(w http.ResponseWriter, r *http.Request) {
	forwardedURL := r.Header.Get("X-CF-Forwarded-Url")
	s.logger.Info("route-service-request", lager.Data{
		"forwardedURL": forwardedURL,
		"method":       r.Method,
	})
	// TODO(scale-to-zero): identify the parked app, hold the request, trigger a
	// scale-up via the scaling engine, wait for the readiness Upsert, then
	// forward to forwardedURL with the X-CF-Proxy-* headers intact.
	w.Header().Set("Retry-After", "5")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("app is waking up, please retry"))
}
