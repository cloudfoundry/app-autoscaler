package routes

import (
	"github.com/gorilla/mux"
)

const (
	memoryMetricPath          = "/v1/apps/{appid}/metrics/memoryused"
	memoryMetricHistoriesPath = "/v1/apps/{appid}/metric_histories/memoryused"

	MemoryMetricRoute        = "memory-metric"
	MemoryMetricHistoryRoute = "memory-metric-histories"

	scalePath             = "/v1/apps/{appid}/scale"
	scalingHistoriesPath  = "/v1/apps/{appid}/scaling_histories"
	activeSchedulePath    = "/v1/apps/{appid}/active_schedules/{scheduleid}"
	appActiveSchedulePath = "/v1/apps/{appid}/active_schedules"

	ScaleRoute                 = "scale"
	HistoriesRoute             = "histories"
	UpdateActiveSchedulesRoute = "updateActiveSchedules"
	DeleteActiveSchedulesRoute = "deleteActiveSchedules"
	GetActiveScheduleRoute     = "getActiveSchedule"
)

type AutoScalerRoute struct {
	metricsCollectorRoutes *mux.Router
	scalingEngineRoutes    *mux.Router
}

var autoScalerRouteInstance *AutoScalerRoute = newRouters()

func newRouters() *AutoScalerRoute {
	instance := &AutoScalerRoute{
		metricsCollectorRoutes: mux.NewRouter(),
		scalingEngineRoutes:    mux.NewRouter(),
	}

	instance.metricsCollectorRoutes.Path(memoryMetricPath).Name(MemoryMetricRoute)
	instance.metricsCollectorRoutes.Path(memoryMetricHistoriesPath).Name(MemoryMetricHistoryRoute)

	instance.scalingEngineRoutes.Path(scalePath).Name(ScaleRoute)
	instance.scalingEngineRoutes.Path(scalingHistoriesPath).Name(HistoriesRoute)
	instance.scalingEngineRoutes.Path(activeSchedulePath).Name(UpdateActiveSchedulesRoute)
	instance.scalingEngineRoutes.Path(activeSchedulePath).Name(DeleteActiveSchedulesRoute)
	instance.scalingEngineRoutes.Path(appActiveSchedulePath).Name(GetActiveScheduleRoute)

	return instance

}
func MetricsCollectorRoutes() *mux.Router {
	return autoScalerRouteInstance.metricsCollectorRoutes
}
func ScalingEngineRoutes() *mux.Router {
	return autoScalerRouteInstance.scalingEngineRoutes
}
