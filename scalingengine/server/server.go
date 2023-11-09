package server

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/apis/scalinghistory"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/schedule"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
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

	return helpers.NewHTTPServer(logger, conf.Server, r)
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
