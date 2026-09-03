package server

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"

	"code.cloudfoundry.org/lager/v3"
)

// VarsFunc adapts a mux-vars handler to http.Handler.
type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vh(w, r, mux.Vars(r))
}

// Server hosts the activator's HTTP surfaces:
//   - CF (route-service) server: receives Gorouter-forwarded requests for
//     parked apps (Loop A);
//   - mTLS server: park/unpark control endpoints called by the scaling engine;
//   - health server.
type Server struct {
	logger       lager.Logger
	conf         *config.Config
	handler      *activator.Handler
	healthRouter *mux.Router
}

func NewServer(logger lager.Logger, conf *config.Config, handler *activator.Handler) *Server {
	return &Server{logger: logger, conf: conf, handler: handler}
}

// CreateCFServer serves BOTH the engine-facing park/unpark control API and the
// route-service catch-all on the activator's single public route.
//
// POC-ONLY: co-hosting these two surfaces on one route/server is a proof-of-
// concept shortcut. The control API needs XFCC auth (only the scaling engine may
// park an app); the route-service catch-all must NOT be XFCC-authed (it carries
// arbitrary end-user traffic with X-CF-Forwarded-Url / X-CF-Proxy-* headers). We
// achieve both with a gorilla/mux subrouter: XFCC middleware is applied ONLY to
// the control subrouter (registered first so its specific /v1/apps/{appid}/park
// paths match before the catch-all), leaving the catch-all unauthenticated.
//
// For a production implementation this should be split onto a dedicated cf-
// route (control, XFCC) vs the public route (route-service), mirroring the
// scalingengine HOST vs CF_HOST split, to avoid any path collision on
// /v1/apps/*/park with a real app's own routes. See docs/design/scale-to-zero.md.
func (s *Server) CreateCFServer(am auth.XFCCAuthMiddleware) (ifrit.Runner, error) {
	router := mux.NewRouter()

	// Control API — XFCC-authed, specific paths, registered before the catch-all.
	control := router.PathPrefix("/v1/apps").Subrouter()
	control.Use(am.XFCCAuthenticationMiddleware)
	control.Path("/{appid}/park").Methods(http.MethodPut).Handler(VarsFunc(s.handler.Park))
	control.Path("/{appid}/park").Methods(http.MethodDelete).Handler(VarsFunc(s.handler.Unpark))

	// Route-service catch-all — NOT XFCC-authed.
	router.PathPrefix("/").HandlerFunc(s.handler.HandleRouteService)

	return helpers.NewHTTPServer(s.logger.Session("cf-server"), s.conf.CFServer, router)
}

// CreateMtlsServer is vestigial (a leftover from the BOSH deployment model where
// components were reachable by stable IP). A CF-app activator is reached only via
// its public route, so the control API lives on the CF server (see
// CreateCFServer). Kept as a 404 stub so the standard server group is unchanged.
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
