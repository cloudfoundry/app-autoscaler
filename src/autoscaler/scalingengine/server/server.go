package server

import (
	"autoscaler/db"
	"autoscaler/routes"
	"autoscaler/scalingengine"
	"autoscaler/scalingengine/config"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"

	"fmt"
	"net/http"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, conf *config.Config, scalingEngineDB db.ScalingEngineDB, scalingEngine scalingengine.ScalingEngine) (ifrit.Runner, error) {
	handler := NewScalingHandler(logger, scalingEngineDB, scalingEngine)

	r := routes.ScalingEngineRoutes()
	r.Get(routes.ScaleRouteName).Handler(VarsFunc(handler.Scale))
	r.Get(routes.GetScalingHistoriesRouteName).Handler(VarsFunc(handler.GetScalingHistories))

	r.Get(routes.SetActiveScheduleRouteName).Handler(VarsFunc(handler.StartActiveSchedule))
	r.Get(routes.DeleteActiveScheduleRouteName).Handler(VarsFunc(handler.RemoveActiveSchedule))
	r.Get(routes.GetActiveSchedulesRouteName).Handler(VarsFunc(handler.GetActiveSchedule))

	addr := fmt.Sprintf("0.0.0.0:%d", conf.Server.Port)
	logger.Info("new-http-server", lager.Data{"serverConfig": conf.Server})

	if (conf.Server.TLS.KeyFile != "") && (conf.Server.TLS.CertFile != "") {
		tlsConfig, err := cfhttp.NewTLSConfig(conf.Server.TLS.CertFile, conf.Server.TLS.KeyFile, conf.Server.TLS.CACertFile)
		if err != nil {
			logger.Error("failed-new-server-new-tls-config", err, lager.Data{"tls": conf.Server.TLS})
			return nil, err
		}
		return http_server.NewTLSServer(addr, r, tlsConfig), nil
	}

	return http_server.New(addr, r), nil
}
