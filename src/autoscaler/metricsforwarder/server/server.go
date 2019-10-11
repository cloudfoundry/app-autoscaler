package server

import (
	"fmt"
	"net/http"
	"os"

	"autoscaler/db"
	"autoscaler/healthendpoint"
	"autoscaler/metricsforwarder/config"
	"autoscaler/metricsforwarder/forwarder"
	"autoscaler/ratelimiter"
	"autoscaler/routes"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	cache "github.com/patrickmn/go-cache"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, conf *config.Config, policyDB db.PolicyDB, credentialCache cache.Cache, allowedMetricCache cache.Cache, httpStatusCollector healthendpoint.HTTPStatusCollector, rateLimiter ratelimiter.Limiter) (ifrit.Runner, error) {

	metricForwarder, err := forwarder.NewMetricForwarder(logger, conf)
	if err != nil {
		logger.Error("failed-to-create-metricforwarder-server", err)
		os.Exit(1)
	}

	mh := NewCustomMetricsHandler(logger, metricForwarder, policyDB, credentialCache, allowedMetricCache, conf.CacheTTL)

	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	rateLimiterMiddleware := ratelimiter.NewRateLimiterMiddleware("appid", rateLimiter, logger.Session("metricforwarder-ratelimiter-middleware"))

	r := routes.MetricsForwarderRoutes()
	r.Use(rateLimiterMiddleware.CheckRateLimit)
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.PostCustomMetricsRouteName).Handler(VarsFunc(mh.PublishMetrics))

	addr := fmt.Sprintf("0.0.0.0:%d", conf.Server.Port)

	var runner ifrit.Runner
	runner = http_server.New(addr, r)

	logger.Info("metrics-forwarder-http-server-created", lager.Data{"config": conf})
	return runner, nil
}