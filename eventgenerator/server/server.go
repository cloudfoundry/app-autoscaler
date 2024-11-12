package server

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
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
func createEventGeneratorRouter(logger lager.Logger, queryAppMetric aggregator.QueryAppMetricsFunc, httpStatusCollector healthendpoint.HTTPStatusCollector, serverConfig config.ServerConfig) (*mux.Router, error) {
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	eh := NewEventGenHandler(logger, queryAppMetric)
	r := routes.EventGeneratorRoutes()
	r.Use(otelmux.Middleware("eventgenerator"))
	r.Use(httpStatusCollectMiddleware.Collect)
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
}

func (s *Server) GetMtlsServer() (ifrit.Runner, error) {
	eventGeneratorRouter, err := createEventGeneratorRouter(s.logger, s.queryAppMetric, s.httpStatusCollector, s.conf.Server)
	if err != nil {
		return nil, fmt.Errorf("failed to create event generator router: %w", err)
	}

	return helpers.NewHTTPServer(s.logger, serverConfigFrom(s.conf), eventGeneratorRouter)
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

func (s *Server) GetHealthServer() (ifrit.Runner, error) {
	healthRouter, err := createHealthRouter(s.appMetricDB, s.policyDb, s.logger, s.conf, s.httpStatusCollector)
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}
	return helpers.NewHTTPServer(s.logger, s.conf.Health.ServerConfig, healthRouter)
}

func createHealthRouter(appMetricDB db.AppMetricDB, policyDb db.PolicyDB, logger lager.Logger, conf *config.Config, httpStatusCollector healthendpoint.HTTPStatusCollector) (*mux.Router, error) {
	checkers := []healthendpoint.Checker{}
	gatherer := CreatePrometheusRegistry(appMetricDB, policyDb, httpStatusCollector, logger)
	healthRouter, err := healthendpoint.NewHealthRouter(conf.Health, checkers, logger.Session("health-server"), gatherer, time.Now)
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}

	return healthRouter, nil
}

func CreatePrometheusRegistry(appMetricDB db.AppMetricDB, policyDb db.PolicyDB, httpStatusCollector healthendpoint.HTTPStatusCollector, logger lager.Logger) *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "eventgenerator", "appMetricDB", appMetricDB),
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "eventgenerator", "policyDB", policyDb),
		httpStatusCollector,
	}, true, logger.Session("eventgenerator-prometheus"))
	return promRegistry
}
