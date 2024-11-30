package server

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func Liveness(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) createEventGeneratorRouter() (*mux.Router, error) {
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector)
	eh := NewEventGenHandler(s.logger, s.queryAppMetric)
	autoscalerRouter := routes.NewRouter()

	r := autoscalerRouter.CreateEventGeneratorRoutes()
	r.Use(otelmux.Middleware("eventgenerator"))
	r.Use(httpStatusCollectMiddleware.Collect)

	r.Get(routes.LivenessRouteName).Handler(VarsFunc(Liveness))
	r.Get(routes.GetAggregatedMetricHistoriesRouteName).Handler(VarsFunc(eh.GetAggregatedMetricHistories))
	return r, nil
}

type Server struct {
	logger              lager.Logger
	conf                *config.Config
	appMetricDB         db.AppMetricDB
	policyDb            db.PolicyDB
	queryAppMetric      aggregator.QueryAppMetricsFunc
	httpStatusCollector healthendpoint.HTTPStatusCollector

	healthRouter *mux.Router
}

func NewServer(logger lager.Logger, conf *config.Config, appMetricDB db.AppMetricDB, policyDb db.PolicyDB, queryAppMetric aggregator.QueryAppMetricsFunc, httpStatusCollector healthendpoint.HTTPStatusCollector) *Server {
	return &Server{
		logger:              logger,
		conf:                conf,
		appMetricDB:         appMetricDB,
		policyDb:            policyDb,
		queryAppMetric:      queryAppMetric,
		httpStatusCollector: httpStatusCollector,
	}
}

func serverConfigFrom(conf *config.Config) helpers.ServerConfig {
	return helpers.ServerConfig{
		TLS:  conf.Server.TLS,
		Port: conf.Server.Port,
	}
}

func (s *Server) CreateHealthServer() (ifrit.Runner, error) {
	err := s.createHealthRouter()
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}
	return helpers.NewHTTPServer(s.logger, s.conf.Health.ServerConfig, s.healthRouter)
}

func (s *Server) createHealthRouter() error {
	checkers := []healthendpoint.Checker{}
	gatherer := createPrometheusRegistry(s.appMetricDB, s.policyDb, s.httpStatusCollector, s.logger)
	healthRouter, err := healthendpoint.NewHealthRouter(s.conf.Health, checkers, s.logger.Session("health-server"), gatherer, time.Now)
	if err != nil {
		return fmt.Errorf("failed to create health router: %w", err)
	}

	s.healthRouter = healthRouter
	return nil
}

func (s *Server) CreateCFServer(am auth.XFCCAuthMiddleware) (ifrit.Runner, error) {
	eventGeneratorRouter, err := s.createEventGeneratorRouter()
	if err != nil {
		return nil, fmt.Errorf("failed to create event generator router: %w", err)
	}

	eventGeneratorRouter.Use(am.XFCCAuthenticationMiddleware)
	if err := s.createHealthRouter(); err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}

	eventGeneratorRouter.PathPrefix("/health").Handler(s.healthRouter)

	return helpers.NewHTTPServer(s.logger, s.conf.CFServer, eventGeneratorRouter)
}

func (s *Server) CreateMtlsServer() (ifrit.Runner, error) {
	eventGeneratorRouter, err := s.createEventGeneratorRouter()
	if err != nil {
		return nil, fmt.Errorf("failed to create event generator router: %w", err)
	}

	return helpers.NewHTTPServer(s.logger, serverConfigFrom(s.conf), eventGeneratorRouter)
}

func createPrometheusRegistry(appMetricDB db.AppMetricDB, policyDb db.PolicyDB, httpStatusCollector healthendpoint.HTTPStatusCollector, logger lager.Logger) *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "eventgenerator", "appMetricDB", appMetricDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "eventgenerator", "policyDB", policyDb),
		httpStatusCollector,
	}, true, logger.Session("eventgenerator-prometheus"))
	return promRegistry
}
