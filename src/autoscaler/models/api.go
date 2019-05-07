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
