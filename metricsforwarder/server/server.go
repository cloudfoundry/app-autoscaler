package server

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/common"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"
)

func NewServer(logger lager.Logger, conf *config.Config, policyDb db.PolicyDB, bindingDB db.BindingDB, credentials cred_helper.Credentials, allowedMetricCache cache.Cache, httpStatusCollector healthendpoint.HTTPStatusCollector, rateLimiter ratelimiter.Limiter) (ifrit.Runner, error) {
	metricForwarder, err := createMetricForwarder(logger, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric forwarder: %w", err)
	}

	mh := NewCustomMetricsHandler(logger, *metricForwarder, policyDb, bindingDB, allowedMetricCache)
	authenticator, err := auth.New(logger, credentials, bindingDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth middleware: %w", err)
	}

	mfRouter := setupMetricForwarderRouter(logger, mh, authenticator, httpStatusCollector, rateLimiter)
	healthRouter, err := createHealthRouter(policyDb, credentials, logger, conf, httpStatusCollector)
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}

	mainRouter := setupMainRouter(mfRouter, healthRouter)
	return helpers.NewHTTPServer(logger, conf.Server, mainRouter)
}

func createMetricForwarder(logger lager.Logger, conf *config.Config) (*forwarder.MetricForwarder, error) {
	metricForwarder, err := forwarder.NewMetricForwarder(logger, conf)
	if err != nil {
		logger.Error("failed-to-create-metricforwarder-server", err)
		return nil, err
	}
	return &metricForwarder, nil
}

func setupMetricForwarderRouter(logger lager.Logger, mh *CustomMetricsHandler, authenticator *auth.Auth, httpStatusCollector healthendpoint.HTTPStatusCollector, rateLimiter ratelimiter.Limiter) *mux.Router {
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	rateLimiterMiddleware := ratelimiter.NewRateLimiterMiddleware("appid", rateLimiter, logger.Session("metricforwarder-ratelimiter-middleware"))

	mfRouter := routes.MetricsForwarderRoutes()
	mfRouter.Use(otelmux.Middleware("metricsforwarder"))
	mfRouter.Use(rateLimiterMiddleware.CheckRateLimit)
	mfRouter.Use(httpStatusCollectMiddleware.Collect)
	mfRouter.Use(authenticator.Authenticate)
	mfRouter.Get(routes.PostCustomMetricsRouteName).Handler(common.VarsFunc(mh.VerifyCredentialsAndPublishMetrics))

	return mfRouter
}

func createHealthRouter(policyDb db.PolicyDB, credDb cred_helper.Credentials, logger lager.Logger, conf *config.Config, httpStatusCollector healthendpoint.HTTPStatusCollector) (*mux.Router, error) {
	checkers := []healthendpoint.Checker{
		healthendpoint.DbChecker(db.PolicyDb, policyDb),
		healthendpoint.DbChecker(db.StoredProcedureDb, credDb),
	}

	gatherer := createPrometheusRegistry(policyDb, httpStatusCollector, logger)
	healthRouter, err := healthendpoint.NewHealthRouter(conf.Health, checkers, logger.Session("health-server"), gatherer, time.Now)
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}
	return healthRouter, nil
}

func createPrometheusRegistry(policyDb db.PolicyDB, httpStatusCollector healthendpoint.HTTPStatusCollector, logger lager.Logger) *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry, []prometheus.Collector{
		healthendpoint.NewDatabaseStatusCollector("autoscaler", "metricsforwarder", "policyDB", policyDb),
		httpStatusCollector,
	}, true, logger.Session("metricsforwarder-prometheus"))
	return promRegistry
}

func setupMainRouter(mfRouter, healthRouter *mux.Router) *mux.Router {
	mainRouter := mux.NewRouter()
	mainRouter.PathPrefix("/v1").Handler(mfRouter)
	mainRouter.PathPrefix("/health").Handler(healthRouter)
	mainRouter.PathPrefix("/").Handler(healthRouter)
	return mainRouter
}
