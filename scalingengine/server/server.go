package server

import (
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/apis/scalinghistory"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/schedule"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"fmt"
	"net/http"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, conf *config.Config, scalingEngineDB db.ScalingEngineDB, scalingEngine scalingengine.ScalingEngine, synchronizer schedule.ActiveScheduleSychronizer, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {
	handler := NewScalingHandler(logger, scalingEngineDB, scalingEngine)
	syncHandler := NewSyncHandler(logger, synchronizer)
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	r := routes.ScalingEngineRoutes()
	r.Use(otelmux.Middleware("scalingengine"))

	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.ScaleRouteName).Handler(VarsFunc(handler.Scale))

	scalingHistoryHandler, err := newScalingHistoryHandler(logger, scalingEngineDB)
	if err != nil {
		return nil, err
	}
	r.Get(routes.GetScalingHistoriesRouteName).Handler(scalingHistoryHandler)

	r.Get(routes.SetActiveScheduleRouteName).Handler(VarsFunc(handler.StartActiveSchedule))
	r.Get(routes.DeleteActiveScheduleRouteName).Handler(VarsFunc(handler.RemoveActiveSchedule))
	r.Get(routes.GetActiveSchedulesRouteName).Handler(VarsFunc(handler.GetActiveSchedule))

	r.Get(routes.SyncActiveSchedulesRouteName).Handler(VarsFunc(syncHandler.Sync))

	var addr string
	if os.Getenv("APP_AUTOSCALER_TEST_RUN") == "true" {
		addr = fmt.Sprintf("localhost:%d", conf.Server.Port)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", conf.Server.Port)
	}

	logger.Info("new-http-server", lager.Data{"serverConfig": conf.Server})

	if (conf.Server.TLS.KeyFile != "") && (conf.Server.TLS.CertFile != "") {
		tlsConfig, err := conf.Server.TLS.CreateServerConfig()
		if err != nil {
			logger.Error("failed-new-server-new-tls-config", err, lager.Data{"tls": conf.Server.TLS})
			return nil, fmt.Errorf("scalingengine tls config error: %w", err)
		}
		return http_server.NewTLSServer(addr, r, tlsConfig), nil
	}

	return http_server.New(addr, r), nil
}

func newScalingHistoryHandler(logger lager.Logger, scalingEngineDB db.ScalingEngineDB) (http.Handler, error) {
	scalingHistoryHandler, err := NewScalingHistoryHandler(logger, scalingEngineDB)
	if err != nil {
		return nil, fmt.Errorf("error creating scaling history handler: %w", err)
	}
	server, err := scalinghistory.NewServer(scalingHistoryHandler, scalingHistoryHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating ogen scaling history server: %w", err)
	}
	return server, err
}
