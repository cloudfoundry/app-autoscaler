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

	BrokerCatalogPath      = "/v2/catalog"
	BrokerCatalogRouteName = "GetCatalog"

	BrokerInstancePath            = "/v2/service_instances/{instanceId}"
	BrokerCreateInstanceRouteName = "CreateInstance"
	BrokerUpdateInstanceRouteName = "UpdateInstance"
	BrokerDeleteInstanceRouteName = "DeleteInstance"

	BrokerBindingPath            = "/v2/service_instances/{instanceId}/service_bindings/{bindingId}"
	BrokerCreateBindingRouteName = "CreateBinding"
	BrokerDeleteBindingRouteName = "DeleteBinding"

	EnvelopePath               = "/v1/envelopes"
	EnvelopeReportRouteName    = "ReportEnvelope"
	CustomMetricsPath          = "/v1/apps/{appid}/metrics"
	PostCustomMetricsRouteName = "PostCustomMetrics"
	SchedulePath               = "/v1/apps/{appId}/schedules"
	UpdateScheduleRouteName    = "UpdateSchedule"
	DeleteScheduleRouteName    = "DeleteSchedule"

	PublicApiScalingHistoryPath      = "/{appId}/scaling_histories"
	PublicApiScalingHistoryRouteName = "GetPublicApiScalingHistories"

	PublicApiMetricsHistoryPath      = "/{appId}/metric_histories/{metricType}"
	PublicApiMetricsHistoryRouteName = "GetPublicApiMetricsHistories"

	PublicApiAggregatedMetricsHistoryPath      = "/{appId}/aggregated_metric_histories/{metricType}"
	PublicApiAggregatedMetricsHistoryRouteName = "GetPublicApiAggregatedMetricsHistories"

	PublicApiPolicyPath            = "/v1/apps/{appId:.+}/policy"
	PublicApiGetPolicyRouteName    = "GetPolicy"
	PublicApiAttachPolicyRouteName = "AttachPolicy"
	PublicApiDetachPolicyRouteName = "DetachPolicy"

	PublicApiCredentialPath            = "/v1/apps/{appId:.+}/credential"
	PublicApiCreateCredentialRouteName = "CreateCredential"
	PublicApiDeleteCredentialRouteName = "DeleteCredential"

	PublicApiInfoPath      = "/v1/info"
	PublicApiInfoRouteName = "GetPublicApiInfo"

	PublicApiHealthPath      = "/health"
	PublicApiHealthRouteName = "GetPublicApiHealth"
)

type AutoScalerRoute struct {
	schedulerRoutes        *mux.Router
	metricsCollectorRoutes *mux.Router
	eventGeneratorRoutes   *mux.Router
	scalingEngineRoutes    *mux.Router
	brokerRoutes           *mux.Router
	metricServerRoutes     *mux.Router
	metricsForwarderRoutes *mux.Router
	apiOpenRoutes          *mux.Router
	apiRoutes              *mux.Router
	apiPolicyRoutes        *mux.Router
	apiCredentialRoutes    *mux.Router
}

var autoScalerRouteInstance = newRouters()

func newRouters() *AutoScalerRoute {
	instance := &AutoScalerRoute{
		schedulerRoutes:        mux.NewRouter(),
		metricsCollectorRoutes: mux.NewRouter(),
		eventGeneratorRoutes:   mux.NewRouter(),
		scalingEngineRoutes:    mux.NewRouter(),
		brokerRoutes:           mux.NewRouter(),
		metricServerRoutes:     mux.NewRouter(),
		metricsForwarderRoutes: mux.NewRouter(),
		apiOpenRoutes:          mux.NewRouter(),
		apiRoutes:              mux.NewRouter(),
		apiPolicyRoutes:        mux.NewRouter(),
		apiCredentialRoutes:    mux.NewRouter(),
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
	instance.brokerRoutes.Path(BrokerInstancePath).Methods(http.MethodPatch).Name(BrokerUpdateInstanceRouteName)
	instance.brokerRoutes.Path(BrokerInstancePath).Methods(http.MethodDelete).Name(BrokerDeleteInstanceRouteName)

	instance.brokerRoutes.Path(BrokerBindingPath).Methods(http.MethodPut).Name(BrokerCreateBindingRouteName)
	instance.brokerRoutes.Path(BrokerBindingPath).Methods(http.MethodDelete).Name(BrokerDeleteBindingRouteName)
	instance.metricsForwarderRoutes.Path(CustomMetricsPath).Methods(http.MethodPost).Name(PostCustomMetricsRouteName)

	instance.metricServerRoutes.Path(EnvelopePath).Name(EnvelopeReportRouteName)

	instance.schedulerRoutes.Path(SchedulePath).Methods(http.MethodPut).Name(UpdateScheduleRouteName)
	instance.schedulerRoutes.Path(SchedulePath).Methods(http.MethodDelete).Name(DeleteScheduleRouteName)
	instance.apiOpenRoutes.Path(PublicApiInfoPath).Methods(http.MethodGet).Name(PublicApiInfoRouteName)
	instance.apiOpenRoutes.Path(PublicApiHealthPath).Methods(http.MethodGet).Name(PublicApiHealthRouteName)

	instance.apiRoutes = instance.apiOpenRoutes.PathPrefix("/v1/apps").Subrouter()
	instance.apiRoutes.Path(PublicApiScalingHistoryPath).Methods(http.MethodGet).Name(PublicApiScalingHistoryRouteName)
	instance.apiRoutes.Path(PublicApiMetricsHistoryPath).Methods(http.MethodGet).Name(PublicApiMetricsHistoryRouteName)
	instance.apiRoutes.Path(PublicApiAggregatedMetricsHistoryPath).Methods(http.MethodGet).Name(PublicApiAggregatedMetricsHistoryRouteName)

	instance.apiPolicyRoutes = instance.apiOpenRoutes.Path(PublicApiPolicyPath).Subrouter()
	instance.apiPolicyRoutes.Path("").Methods(http.MethodGet).Name(PublicApiGetPolicyRouteName)
	instance.apiPolicyRoutes.Path("").Methods(http.MethodPut).Name(PublicApiAttachPolicyRouteName)
	instance.apiPolicyRoutes.Path("").Methods(http.MethodDelete).Name(PublicApiDetachPolicyRouteName)

	instance.apiCredentialRoutes = instance.apiOpenRoutes.Path(PublicApiCredentialPath).Subrouter()
	instance.apiCredentialRoutes.Path("").Methods(http.MethodPut).Name(PublicApiCreateCredentialRouteName)
	instance.apiCredentialRoutes.Path("").Methods(http.MethodDelete).Name(PublicApiDeleteCredentialRouteName)

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

func MetricServerRoutes() *mux.Router {
	return autoScalerRouteInstance.metricServerRoutes
}

func MetricsForwarderRoutes() *mux.Router {
	return autoScalerRouteInstance.metricsForwarderRoutes
}

func SchedulerRoutes() *mux.Router {
	return autoScalerRouteInstance.schedulerRoutes
}

func ApiOpenRoutes() *mux.Router {
	return autoScalerRouteInstance.apiOpenRoutes
}

func ApiRoutes() *mux.Router {
	return autoScalerRouteInstance.apiRoutes
}
func ApiPolicyRoutes() *mux.Router {
	return autoScalerRouteInstance.apiPolicyRoutes
}
func ApiCredentialRoutes() *mux.Router {
	return autoScalerRouteInstance.apiCredentialRoutes
}
