package models

import (
	"encoding/json"
	"reflect"
)

type XFCCAuth struct {
	ValidOrgGuid   string `yaml:"valid_org_guid" json:"valid_org_guid"`
	ValidSpaceGuid string `yaml:"valid_space_guid" json:"valid_space_guid"`
}

type BrokerContext struct {
	OrgGUID   string `json:"organization_guid"`
	SpaceGUID string `json:"space_guid"`
}

type PreviousValues struct {
	PlanID string `json:"plan_id"`
}
type BrokerCommonRequestBody struct {
	ServiceID      string         `json:"service_id"`
	PlanID         string         `json:"plan_id,omitempty"`
	BrokerContext  BrokerContext  `json:"context"`
	PreviousValues PreviousValues `json:"previous_values"`
}

type InstanceParameters struct {
	DefaultPolicy json.RawMessage `json:"default_policy,omitempty"`
}

type InstanceCreationRequestBody struct {
	BrokerCommonRequestBody
	OrgGUID    string             `json:"organization_guid"`
	SpaceGUID  string             `json:"space_guid"`
	Parameters InstanceParameters `json:"parameters,omitempty"`
}

type InstanceUpdateRequestBody struct {
	BrokerCommonRequestBody
	Parameters *InstanceParameters `json:"parameters,omitempty"`
}

type ServiceInstance struct {
	ServiceInstanceId string `db:"service_instance_id"`
	OrgId             string `db:"org_id"`
	SpaceId           string `db:"space_id"`
	DefaultPolicy     string `db:"default_policy"`
	DefaultPolicyGuid string `db:"default_policy_guid"`
}

func (s *ServiceInstance) Equals(other *ServiceInstance) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}

	if s.ServiceInstanceId != other.ServiceInstanceId ||
		s.OrgId != other.OrgId ||
		s.SpaceId != other.SpaceId ||
		s.DefaultPolicyGuid != other.DefaultPolicyGuid {
		return false
	}

	return s.policiesEqual(other)
}

func (s *ServiceInstance) policiesEqual(other *ServiceInstance) bool {
	if (s.DefaultPolicy == "" || s.DefaultPolicy == "null") &&
		(other.DefaultPolicy == "" || other.DefaultPolicy == "null") {
		return true
	}

	var policy1, policy2 PolicyDefinition

	if err := json.Unmarshal([]byte(s.DefaultPolicy), &policy1); err != nil {
		return false
	}

	if err := json.Unmarshal([]byte(other.DefaultPolicy), &policy2); err != nil {
		return false
	}

	return reflect.DeepEqual(policy1, policy2)
}

type ServiceBinding struct {
	ServiceBindingID      string `db:"binding_id"`
	ServiceInstanceID     string `db:"service_instance_id"`
	AppID                 string `db:"app_id"`
	CustomMetricsStrategy string `db:"custom_metrics_strategy"`
}

type BindingRequestBody struct {
	BrokerCommonRequestBody
	AppID string `json:"app_guid"`

	// 🚧 To-do: Support `bind_resource` as described in
	// <https://github.com/openservicebrokerapi/servicebroker/blob/v2.17/spec.md#request-creating-a-service-binding>;
	// Afterwards make use of it in tests;
	Policy json.RawMessage `json:"parameters,omitempty"`
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
