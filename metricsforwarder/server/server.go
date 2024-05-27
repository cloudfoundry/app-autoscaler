package server

import (
	"fmt"
	"os"

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
	"github.com/patrickmn/go-cache"
	"github.com/tedsuo/ifrit"
)

func NewServer(logger lager.Logger, conf *config.Config, policyDB db.PolicyDB, credentials cred_helper.Credentials, allowedMetricCache cache.Cache, httpStatusCollector healthendpoint.HTTPStatusCollector, rateLimiter ratelimiter.Limiter) (ifrit.Runner, error) {
	metricForwarder, err := forwarder.NewMetricForwarder(logger, conf)
	if err != nil {
		logger.Error("failed-to-create-metricforwarder-server", err)
		os.Exit(1)
	}

	mh := NewCustomMetricsHandler(logger, metricForwarder, policyDB, allowedMetricCache)
	authenticator, err := auth.New(logger, credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to add auth middleware: %w", err)
	}
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	rateLimiterMiddleware := ratelimiter.NewRateLimiterMiddleware("appid", rateLimiter, logger.Session("metricforwarder-ratelimiter-middleware"))

	r := routes.MetricsForwarderRoutes()
	r.Use(otelmux.Middleware("metricsforwarder"))
	r.Use(rateLimiterMiddleware.CheckRateLimit)
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Use(authenticator.Authenticate)
	r.Get(routes.PostCustomMetricsRouteName).Handler(common.VarsFunc(mh.VerifyCredentialsAndPublishMetrics))

	return helpers.NewHTTPServer(logger, conf.Server, r)
}
