package collector

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, serverConfig *ServerConfig, query MetricQueryFunc, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {
	mh := NewMetricHandler(logger, serverConfig.NodeIndex, serverConfig.NodeAddrs, query)
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)

	r := routes.MetricsCollectorRoutes()
	r.Use(otelmux.Middleware("metricsserver"))
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.GetMetricHistoriesRouteName).Handler(VarsFunc(mh.GetMetricHistories))

	httpServerConfig := helpers.ServerConfig{
		Port: serverConfig.Port,
		TLS:  serverConfig.TLS,
	}

	return helpers.NewHTTPServer(logger, httpServerConfig, r)
}
