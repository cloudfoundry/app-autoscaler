package server

import (
	"fmt"
	"net/http"

	"autoscaler/db"
	"autoscaler/healthendpoint"
	"autoscaler/metricscollector/collector"
	"autoscaler/metricscollector/config"
	"autoscaler/routes"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, conf *config.Config, query collector.MetricQueryFunc, database db.InstanceMetricsDB, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {
	mh := NewMetricHandler(logger, conf.Server.NodeIndex, conf.Server.NodeAddrs, query, database)
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)

	r := routes.MetricsCollectorRoutes()
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.GetMetricHistoriesRouteName).Handler(VarsFunc(mh.GetMetricHistories))

	addr := fmt.Sprintf("0.0.0.0:%d", conf.Server.Port)

	var runner ifrit.Runner
	if (conf.Server.TLS.KeyFile == "") || (conf.Server.TLS.CertFile == "") {
		runner = http_server.New(addr, r)
	} else {
		tlsConfig, err := cfhttp.NewTLSConfig(conf.Server.TLS.CertFile, conf.Server.TLS.KeyFile, conf.Server.TLS.CACertFile)
		if err != nil {
			logger.Error("failed-new-server-new-tls-config", err, lager.Data{"tls": conf.Server.TLS})
			return nil, err
		}
		runner = http_server.NewTLSServer(addr, r, tlsConfig)
	}

	logger.Info("http-server-created", lager.Data{"serverConfig": conf.Server})
	return runner, nil
}
