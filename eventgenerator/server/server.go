package server

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, conf *config.Config, queryAppMetric aggregator.QueryAppMetricsFunc, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {
	eh := NewEventGenHandler(logger, queryAppMetric)
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	r := routes.EventGeneratorRoutes()
	r.Use(otelmux.Middleware("eventgenerator"))
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.GetAggregatedMetricHistoriesRouteName).Handler(VarsFunc(eh.GetAggregatedMetricHistories))

	httpServerConfig := helpers.ServerConfig{
		Port: conf.Server.Port,
		TLS:  conf.Server.TLS,
	}

	return helpers.NewHTTPServer(logger, httpServerConfig, r)
}
