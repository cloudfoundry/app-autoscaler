package routes

import (
	"github.com/gorilla/mux"

	"net/http"
)

const (
	MetricHistoriesPath         = "/v1/apps/{appid}/metric_histories/{metrictype}"
	GetMetricHistoriesRouteName = "GetMetricHistories"

	MemoryMetricPath         = "/v1/apps/{appid}/metrics/memoryused"
	GetMemoryMetricRouteName = "GetMemoryMetric"

	ScalePath      = "/v1/apps/{appid}/scale"
	ScaleRouteName = "Scale"

	ScalingHistoriesPath         = "/v1/apps/{appid}/scaling_histories"
	GetScalingHistoriesRouteName = "GetScalingHistories"

	ActiveSchedulePath            = "/v1/apps/{appid}/active_schedules/{scheduleid}"
	SetActiveScheduleRouteName    = "SetActiveSchedule"
	DeleteActiveScheduleRouteName = "DeleteActiveSchedule"

	ActiveSchedulesPath         = "/v1/apps/{appid}/active_schedules"
	GetActiveSchedulesRouteName = "GetActiveSchedules"
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

	instance.metricsCollectorRoutes.Path(MemoryMetricPath).Methods(http.MethodGet).Name(GetMemoryMetricRouteName)
	instance.metricsCollectorRoutes.Path(MetricHistoriesPath).Methods(http.MethodGet).Name(GetMetricHistoriesRouteName)

	instance.scalingEngineRoutes.Path(ScalePath).Methods(http.MethodPost).Name(ScaleRouteName)
	instance.scalingEngineRoutes.Path(ScalingHistoriesPath).Methods(http.MethodGet).Name(GetScalingHistoriesRouteName)
	instance.scalingEngineRoutes.Path(ActiveSchedulePath).Methods(http.MethodPut).Name(SetActiveScheduleRouteName)
	instance.scalingEngineRoutes.Path(ActiveSchedulePath).Methods(http.MethodDelete).Name(DeleteActiveScheduleRouteName)
	instance.scalingEngineRoutes.Path(ActiveSchedulesPath).Methods(http.MethodGet).Name(GetActiveSchedulesRouteName)

	return instance

}
func MetricsCollectorRoutes() *mux.Router {
	return autoScalerRouteInstance.metricsCollectorRoutes
}
func ScalingEngineRoutes() *mux.Router {
	return autoScalerRouteInstance.scalingEngineRoutes
}
