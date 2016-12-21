package routes

import (
	"github.com/gorilla/mux"
)

const (
	memoryMetricPath          = "/v1/apps/{appid}/metrics/memory"
	memoryMetricHistoriesPath = "/v1/apps/{appid}/metric_histories/memory"

	MemoryMetricRoute        = "memory-metric"
	MemoryMetricHistoryRoute = "memory-metric-histories"

	scalePath            = "/v1/apps/{appid}/scale"
	scalingHistoriesPath = "/v1/apps/{appid}/scaling_histories"
	activeSchedulePath   = "/v1/apps/{appid}/active_schedules/{scheduleid}"

	ScaleRoute                 = "scale"
	HistoreisRoute             = "histories"
	UpdateActiveSchedulesRoute = "updateActiveSchedules"
	DeleteActiveSchedulesRoute = "deleteActiveSchedules"
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
	instance.scalingEngineRoutes.Path(scalingHistoriesPath).Name(HistoreisRoute)
	instance.scalingEngineRoutes.Path(activeSchedulePath).Name(UpdateActiveSchedulesRoute)
	instance.scalingEngineRoutes.Path(activeSchedulePath).Name(DeleteActiveSchedulesRoute)

	return instance

}
func MetricsCollectorRoutes() *mux.Router {
	return autoScalerRouteInstance.metricsCollectorRoutes
}
func ScalingEngineRoutes() *mux.Router {
	return autoScalerRouteInstance.scalingEngineRoutes
}
