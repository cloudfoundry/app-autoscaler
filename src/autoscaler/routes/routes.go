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

	PublicApiCustomMetricsCredentialPath            = "/v1/apps/{appId:.+}/custom_metrics_credential"
	PublicApiGetCustomMetricsCredentialRouteName    = "GetCustomMetricsCredential"
	PublicApiCreateCustomMetricsCredentialRouteName = "CreateCustomMetricsCredential"
	PublicApiDeleteCustomMetricsCredentialRouteName = "DeleteCustomMetricsCredential"

	PublicApiInfoPath      = "/v1/info"
	PublicApiInfoRouteName = "GetPublicApiInfo"

	PublicApiHealthPath      = "/health"
	PublicApiHealthRouteName = "GetPublicApiHealth"
)

type AutoScalerRoute struct {
	schedulerRoutes                         *mux.Router
	metricsCollectorRoutes                  *mux.Router
	eventGeneratorRoutes                    *mux.Router
	scalingEngineRoutes                     *mux.Router
	brokerRoutes                            *mux.Router
	metricServerRoutes                      *mux.Router
	metricsForwarderRoutes                  *mux.Router
	publicApiRoutes                         *mux.Router
	publicApiProtectedRoutes                *mux.Router
	publicApiPolicyRouters                  *mux.Router
	publicApiCustomMetricsCredentialRouters *mux.Router
}

var autoScalerRouteInstance = newRouters()

func newRouters() *AutoScalerRoute {
	instance := &AutoScalerRoute{
		schedulerRoutes:                         mux.NewRouter(),
		metricsCollectorRoutes:                  mux.NewRouter(),
		eventGeneratorRoutes:                    mux.NewRouter(),
		scalingEngineRoutes:                     mux.NewRouter(),
		brokerRoutes:                            mux.NewRouter(),
		metricServerRoutes:                      mux.NewRouter(),
		metricsForwarderRoutes:                  mux.NewRouter(),
		publicApiRoutes:                         mux.NewRouter(),
		publicApiProtectedRoutes:                mux.NewRouter(),
		publicApiPolicyRouters:                  mux.NewRouter(),
		publicApiCustomMetricsCredentialRouters: mux.NewRouter(),
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

	instance.metricServerRoutes.Path(EnvelopePath).Name(EnvelopeReportRouteName)

	instance.schedulerRoutes.Path(SchedulePath).Methods(http.MethodPut).Name(UpdateScheduleRouteName)
	instance.schedulerRoutes.Path(SchedulePath).Methods(http.MethodDelete).Name(DeleteScheduleRouteName)
	instance.publicApiRoutes.Path(PublicApiInfoPath).Methods(http.MethodGet).Name(PublicApiInfoRouteName)
	instance.publicApiRoutes.Path(PublicApiHealthPath).Methods(http.MethodGet).Name(PublicApiHealthRouteName)

	instance.publicApiProtectedRoutes = instance.publicApiRoutes.PathPrefix("/v1/apps").Subrouter()
	instance.publicApiProtectedRoutes.Path(PublicApiScalingHistoryPath).Methods(http.MethodGet).Name(PublicApiScalingHistoryRouteName)
	instance.publicApiProtectedRoutes.Path(PublicApiMetricsHistoryPath).Methods(http.MethodGet).Name(PublicApiMetricsHistoryRouteName)
	instance.publicApiProtectedRoutes.Path(PublicApiAggregatedMetricsHistoryPath).Methods(http.MethodGet).Name(PublicApiAggregatedMetricsHistoryRouteName)

	instance.publicApiPolicyRouters = instance.publicApiRoutes.Path(PublicApiPolicyPath).Subrouter()
	instance.publicApiPolicyRouters.Path("").Methods(http.MethodGet).Name(PublicApiGetPolicyRouteName)
	instance.publicApiPolicyRouters.Path("").Methods(http.MethodPut).Name(PublicApiAttachPolicyRouteName)
	instance.publicApiPolicyRouters.Path("").Methods(http.MethodDelete).Name(PublicApiDetachPolicyRouteName)

	instance.publicApiCustomMetricsCredentialRouters = instance.publicApiRoutes.Path(PublicApiCustomMetricsCredentialPath).Subrouter()
	instance.publicApiCustomMetricsCredentialRouters.Path("").Methods(http.MethodGet).Name(PublicApiGetCustomMetricsCredentialRouteName)
	instance.publicApiCustomMetricsCredentialRouters.Path("").Methods(http.MethodPut).Name(PublicApiCreateCustomMetricsCredentialRouteName)
	instance.publicApiCustomMetricsCredentialRouters.Path("").Methods(http.MethodDelete).Name(PublicApiDeleteCustomMetricsCredentialRouteName)

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

func PublicApiRoutes() *mux.Router {
	return autoScalerRouteInstance.publicApiRoutes
}

func PublicApiProtectedRoutes() *mux.Router {
	return autoScalerRouteInstance.publicApiProtectedRoutes
}
func PublicApiPolicyRoutes() *mux.Router {
	return autoScalerRouteInstance.publicApiPolicyRouters
}
func PublicApiCustomMetricsCredentialRoutes() *mux.Router {
	return autoScalerRouteInstance.publicApiCustomMetricsCredentialRouters
}
