package models

type BrokerCommonRequestBody struct {
	ServiceID string `json:"service_id"`
	PlanID    string `json:"plan_id"`
}

type InstanceCreationRequestBody struct {
	BrokerCommonRequestBody
	OrgGUID   string `json:"organization_guid"`
	SpaceGUID string `json:"space_guid"`
}

type BindingRequestBody struct {
	BrokerCommonRequestBody
	AppID  string `json:"app_guid"`
	Policy string `json:"parameters"`
}

type UnbindingRequestBody struct {
	BrokerCommonRequestBody
	AppID string `json:"app_guid"`
}

type PublicApiResponseBase struct {
	TotalResults int           `json:"total_results"`
	TotalPages   int           `json:"total_pages"`
	Page         int           `json:"page"`
	PrevUrl      string        `json:"prev_url"`
	NextUrl      string        `json:"next_url"`
	Resources    []interface{} `json:"resources"`
}
type InstanceMetricResponse struct {
	PublicApiResponseBase
	Resources []AppInstanceMetric `json:"resources"`
}
type AppMetricResponse struct {
	PublicApiResponseBase
	Resources []AppMetric `json:"resources"`
}
type AppScalingHistoryResponse struct {
	PublicApiResponseBase
	Resources []AppScalingHistory `json:"resources"`
}
