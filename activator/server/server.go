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
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

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

// CreateCFServer serves the route-service endpoint (catch-all: any path/method
// the original app would serve). See docs/design/scale-to-zero.md §5.2.
func (s *Server) CreateCFServer(am auth.XFCCAuthMiddleware) (ifrit.Runner, error) {
	router := mux.NewRouter()
	router.Use(am.XFCCAuthenticationMiddleware)
	router.PathPrefix("/").HandlerFunc(s.handler.HandleRouteService)
	return helpers.NewHTTPServer(s.logger.Session("cf-server"), s.conf.CFServer, router)
}

// CreateMtlsServer serves the park/unpark control endpoints for the engine.
func (s *Server) CreateMtlsServer() (ifrit.Runner, error) {
	router := routes.NewRouter()
	activatorRouter := router.CreateActivatorRoutes()
	activatorRouter.Get(routes.ActivatorParkRouteName).Handler(VarsFunc(s.handler.Park))
	activatorRouter.Get(routes.ActivatorUnparkRouteName).Handler(VarsFunc(s.handler.Unpark))
	return helpers.NewHTTPServer(s.logger.Session("mtls-server"), s.conf.Server, activatorRouter)
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
