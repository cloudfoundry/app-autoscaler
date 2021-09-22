package server

import (
	"autoscaler/db"
	"autoscaler/healthendpoint"
	"autoscaler/metricsforwarder/config"
	"autoscaler/metricsforwarder/forwarder"
	"autoscaler/metricsforwarder/server/auth"
	"autoscaler/metricsforwarder/server/common"
	"autoscaler/ratelimiter"
	"autoscaler/routes"
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/patrickmn/go-cache"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

func NewServer(logger lager.Logger, conf *config.Config, policyDB db.PolicyDB, credentialCache cache.Cache, allowedMetricCache cache.Cache, httpStatusCollector healthendpoint.HTTPStatusCollector, rateLimiter ratelimiter.Limiter) (ifrit.Runner, error) {
	metricForwarder, err := forwarder.NewMetricForwarder(logger, conf)
	if err != nil {
		logger.Error("failed-to-create-metricforwarder-server", err)
		os.Exit(1)
	}

	mh := NewCustomMetricsHandler(logger, metricForwarder, policyDB, allowedMetricCache)
	authenticator := auth.New(logger, policyDB, credentialCache, conf.CacheTTL, conf.MetricsForwarderConfig.TLS.CACertFile)

	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	rateLimiterMiddleware := ratelimiter.NewRateLimiterMiddleware("appid", rateLimiter, logger.Session("metricforwarder-ratelimiter-middleware"))

	r := routes.MetricsForwarderRoutes()
	r.Use(rateLimiterMiddleware.CheckRateLimit)
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Use(authenticator.Authenticate)
	r.Get(routes.PostCustomMetricsRouteName).Handler(common.VarsFunc(mh.VerifyCredentialsAndPublishMetrics))

	var addr string
	if os.Getenv("APP_AUTOSCALER_TEST_RUN") == "true" {
		addr = fmt.Sprintf("localhost:%d", conf.Server.Port)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", conf.Server.Port)
	}

	runner := http_server.New(addr, r)

	logger.Info("metrics-forwarder-http-server-created", lager.Data{"config": conf})
	return runner, nil
}
