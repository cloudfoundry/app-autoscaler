package routes

import (
	"net/http"

	"github.com/gorilla/mux"
)

const (
	MetricHistoriesPath         = "/v1/apps/{appid}/metric_histories/{metrictype}"
	GetMetricHistoriesRouteName = "GetMetricHistories"

	AggregatedMetricHistoriesPath         = "/v1/apps/{appid}/aggregated_metric_histories/{metrictype}"
	GetAggregatedMetricHistoriesRouteName = "GetAggregatedMetricHistories"

	ScalePath      = "/v1/apps/{appid}/scale"
	ScaleRouteName = "Scale"

	ScalingHistoriesPath         = "/v1/apps/{guid}/scaling_histories"
	GetScalingHistoriesRouteName = "GetScalingHistories"

	LivenessPath      = "/v1/liveness"
	LivenessRouteName = "Liveness"

	ActiveSchedulePath            = "/v1/apps/{appid}/active_schedules/{scheduleid}"
	SetActiveScheduleRouteName    = "SetActiveSchedule"
	DeleteActiveScheduleRouteName = "DeleteActiveSchedule"

	ActiveSchedulesPath         = "/v1/apps/{appid}/active_schedules"
	GetActiveSchedulesRouteName = "GetActiveSchedules"

	SyncActiveSchedulesPath      = "/v1/syncSchedules"
	SyncActiveSchedulesRouteName = "SyncActiveSchedules"

	HealthPath = "/health"

	EnvelopePath               = "/v1/envelopes"
	EnvelopeReportRouteName    = "ReportEnvelope"
	CustomMetricsPath          = "/v1/apps/{appid}/metrics"
	PostCustomMetricsRouteName = "PostCustomMetrics"
	SchedulePath               = "/v1/apps/{appId}/schedules"
	UpdateScheduleRouteName    = "UpdateSchedule"
	DeleteScheduleRouteName    = "DeleteSchedule"

	PublicApiScalingHistoryPath      = "/{appId}/scaling_histories"
	PublicApiScalingHistoryRouteName = "GetPublicApiScalingHistories"

	PublicApiMetricsHistoryPath = "/{appId}/metric_histories/{metricType}"

	PublicApiAggregatedMetricsHistoryPath      = "/{appId}/aggregated_metric_histories/{metricType}"
	PublicApiAggregatedMetricsHistoryRouteName = "GetPublicApiAggregatedMetricsHistories"

	PublicApiPolicyPath            = "/v1/apps/{appId:.+}/policy"
	PublicApiGetPolicyRouteName    = "GetPolicy"
	PublicApiAttachPolicyRouteName = "AttachPolicy"
	PublicApiDetachPolicyRouteName = "DetachPolicy"

	PublicApiInfoPath      = "/v1/info"
	PublicApiInfoRouteName = "GetPublicApiInfo"

	PublicApiHealthRouteName = "GetPublicApiHealth"
)

type Router struct {
	router *mux.Router
}

func NewRouter() *Router {
	r := mux.NewRouter()
	return &Router{router: r}
}

func (r *Router) RegisterRoutes() {
	r.registerMetricsCollectorRoutes()
	r.registerMetricsForwarderRoutes()
	r.registerSchedulerRoutes()

	r.CreateScalingEngineRoutes()

	r.CreateEventGeneratorSubrouter()
	r.CreateApiPublicSubrouter()
	r.CreateApiSubrouter()
	r.CreateApiPolicySubrouter()
}

func (r *Router) CreateScalingEngineRoutes() *mux.Router {
	r.router.Path(ScalePath).Methods(http.MethodPost).Name(ScaleRouteName)
	r.router.Path(ScalingHistoriesPath).Methods(http.MethodGet).Name(GetScalingHistoriesRouteName)
	r.router.Path(ActiveSchedulePath).Methods(http.MethodPut).Name(SetActiveScheduleRouteName)
	r.router.Path(ActiveSchedulePath).Methods(http.MethodDelete).Name(DeleteActiveScheduleRouteName)
	r.router.Path(ActiveSchedulesPath).Methods(http.MethodGet).Name(GetActiveSchedulesRouteName)
	r.router.Path(SyncActiveSchedulesPath).Methods(http.MethodPut).Name(SyncActiveSchedulesRouteName)
	r.router.Path(LivenessPath).Methods(http.MethodGet).Name(LivenessRouteName)

	return r.router
}

func (r *Router) registerMetricsCollectorRoutes() {
	r.router.Path(MetricHistoriesPath).Methods(http.MethodGet).Name(GetMetricHistoriesRouteName)
}

func (r *Router) registerMetricsForwarderRoutes() {
	r.router.Path(CustomMetricsPath).Methods(http.MethodPost).Name(PostCustomMetricsRouteName)
}

func (r *Router) registerSchedulerRoutes() {
	r.router.Path(SchedulePath).Methods(http.MethodPut).Name(UpdateScheduleRouteName)
	r.router.Path(SchedulePath).Methods(http.MethodDelete).Name(DeleteScheduleRouteName)
}

func (r *Router) CreateEventGeneratorSubrouter() *mux.Router {
	eventgeneratorRoutes := r.router.PathPrefix("").Subrouter()
	eventgeneratorRoutes.Path(AggregatedMetricHistoriesPath).Methods(http.MethodGet).Name(GetAggregatedMetricHistoriesRouteName)
	eventgeneratorRoutes.Path(LivenessPath).Methods(http.MethodGet).Name(LivenessRouteName)
	return eventgeneratorRoutes
}

func (r *Router) CreateApiPublicSubrouter() *mux.Router {
	publicApiRoutes := r.router.PathPrefix("").Subrouter()
	publicApiRoutes.Path(PublicApiInfoPath).Methods(http.MethodGet).Name(PublicApiInfoRouteName)
	publicApiRoutes.Path(HealthPath).Methods(http.MethodGet).Name(PublicApiHealthRouteName)

	return publicApiRoutes
}

func (r *Router) CreateApiSubrouter() *mux.Router {
	apiRoutes := r.router.PathPrefix("/v1/apps").Subrouter()
	apiRoutes.Path(PublicApiScalingHistoryPath).Methods(http.MethodGet).Name(PublicApiScalingHistoryRouteName)
	apiRoutes.Path(PublicApiAggregatedMetricsHistoryPath).Methods(http.MethodGet).Name(PublicApiAggregatedMetricsHistoryRouteName)
	return apiRoutes
}

func (r *Router) CreateApiPolicySubrouter() *mux.Router {
	apiPolicyRoutes := r.router.Path(PublicApiPolicyPath).Subrouter()
	apiPolicyRoutes.Path("").Methods(http.MethodGet).Name(PublicApiGetPolicyRouteName)
	apiPolicyRoutes.Path("").Methods(http.MethodPut).Name(PublicApiAttachPolicyRouteName)
	apiPolicyRoutes.Path("").Methods(http.MethodDelete).Name(PublicApiDetachPolicyRouteName)
	return apiPolicyRoutes
}

func (r *Router) GetRouter() *mux.Router {
	return r.router
}

var autoScalerRouteInstance = NewRouter()

func init() {
	autoScalerRouteInstance.RegisterRoutes()
}

func MetricsCollectorRoutes() *mux.Router {
	return autoScalerRouteInstance.GetRouter()
}

func MetricsForwarderRoutes() *mux.Router {
	return autoScalerRouteInstance.GetRouter()
}

func SchedulerRoutes() *mux.Router {
	return autoScalerRouteInstance.GetRouter()
}

func ApiPolicyRoutes() *mux.Router {
	return autoScalerRouteInstance.GetRouter()
}
