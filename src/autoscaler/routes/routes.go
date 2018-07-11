package routes

import (
	"github.com/gorilla/mux"

	"net/http"
)

const (
	MetricHistoriesPath         = "/v1/apps/{appid}/metric_histories/{metrictype}"
	GetMetricHistoriesRouteName = "GetMetricHistories"

	AggregatedMetricHistoriesPath         = "/v1/apps/{appid}/aggregated_metric_histories/{metrictype}"
	GetAggregatedMetricHistoriesRouteName = "GetAggregatedMetricHistories"

	ScalePath      = "/v1/apps/{appid}/scale"
	ScaleRouteName = "Scale"

	ScalingHistoriesPath         = "/v1/apps/{appid}/scaling_histories"
	GetScalingHistoriesRouteName = "GetScalingHistories"

	ActiveSchedulePath            = "/v1/apps/{appid}/active_schedules/{scheduleid}"
	SetActiveScheduleRouteName    = "SetActiveSchedule"
	DeleteActiveScheduleRouteName = "DeleteActiveSchedule"

	ActiveSchedulesPath         = "/v1/apps/{appid}/active_schedules"
	GetActiveSchedulesRouteName = "GetActiveSchedules"

	SyncActiveSchedulesPath      = "/v1/syncSchedules"
	SyncActiveSchedulesRouteName = "SyncActiveSchedules"

	BrokerCatalogPath      = "/sb/v2/catalog"
	BrokerCatalogRouteName = "GetCatalog"

	BrokerInstancePath            = "/sb/v2/service_instances/{instanceId}"
	BrokerCreateInstanceRouteName = "CreateInstance"
	BrokerDeleteInstanceRouteName = "DeleteInstance"

	BrokerBindingPath            = "/sb/v2/service_instances/{instanceId}/service_bindings/{bindingId}"
	BrokerCreateBindingRouteName = "CreateBinding"
	BrokerDeleteBindingRouteName = "DeleteBinding"
	CustomMetricsPath            = "/v1/{appid}/metrics"
	PostCustomMetricsRouteName   = "PostCustomMetrics"
)

type AutoScalerRoute struct {
	metricsCollectorRoutes *mux.Router
	eventGeneratorRoutes   *mux.Router
	scalingEngineRoutes    *mux.Router
	brokerRoutes           *mux.Router
	metricsForwarderRoutes *mux.Router
}

var autoScalerRouteInstance = newRouters()

func newRouters() *AutoScalerRoute {
	instance := &AutoScalerRoute{
		metricsCollectorRoutes: mux.NewRouter(),
		eventGeneratorRoutes:   mux.NewRouter(),
		scalingEngineRoutes:    mux.NewRouter(),
		brokerRoutes:           mux.NewRouter(),
		metricsForwarderRoutes: mux.NewRouter(),
	}

	instance.metricsCollectorRoutes.Path(MetricHistoriesPath).Methods(http.MethodGet).Name(GetMetricHistoriesRouteName)

	instance.eventGeneratorRoutes.Path(AggregatedMetricHistoriesPath).Methods(http.MethodGet).Name(GetAggregatedMetricHistoriesRouteName)

	instance.scalingEngineRoutes.Path(ScalePath).Methods(http.MethodPost).Name(ScaleRouteName)
	instance.scalingEngineRoutes.Path(ScalingHistoriesPath).Methods(http.MethodGet).Name(GetScalingHistoriesRouteName)
	instance.scalingEngineRoutes.Path(ActiveSchedulePath).Methods(http.MethodPut).Name(SetActiveScheduleRouteName)
	instance.scalingEngineRoutes.Path(ActiveSchedulePath).Methods(http.MethodDelete).Name(DeleteActiveScheduleRouteName)
	instance.scalingEngineRoutes.Path(ActiveSchedulesPath).Methods(http.MethodGet).Name(GetActiveSchedulesRouteName)
	instance.scalingEngineRoutes.Path(SyncActiveSchedulesPath).Methods(http.MethodPut).Name(SyncActiveSchedulesRouteName)

	instance.brokerRoutes.Path(BrokerCatalogPath).Methods(http.MethodGet).Name(BrokerCatalogRouteName)

	instance.brokerRoutes.Path(BrokerInstancePath).Methods(http.MethodPut).Name(BrokerCreateInstanceRouteName)
	instance.brokerRoutes.Path(BrokerInstancePath).Methods(http.MethodDelete).Name(BrokerDeleteInstanceRouteName)

	instance.brokerRoutes.Path(BrokerBindingPath).Methods(http.MethodPut).Name(BrokerCreateBindingRouteName)
	instance.brokerRoutes.Path(BrokerBindingPath).Methods(http.MethodDelete).Name(BrokerDeleteBindingRouteName)
	instance.metricsForwarderRoutes.Path(CustomMetricsPath).Methods(http.MethodPost).Name(PostCustomMetricsRouteName)

	return instance

}
func MetricsCollectorRoutes() *mux.Router {
	return autoScalerRouteInstance.metricsCollectorRoutes
}

func EventGeneratorRoutes() *mux.Router {
	return autoScalerRouteInstance.eventGeneratorRoutes
}

func ScalingEngineRoutes() *mux.Router {
	return autoScalerRouteInstance.scalingEngineRoutes
}

func BrokerRoutes() *mux.Router {
	return autoScalerRouteInstance.brokerRoutes
}

func MetricsForwarderRoutes() *mux.Router {
	return autoScalerRouteInstance.metricsForwarderRoutes
}
