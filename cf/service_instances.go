package cf

import (
	"fmt"
)

const (
	ServicePlansPath = "v2/service_plans"
)

type ServiceInstanceEntity struct {
	ServicePlanGuid string `json:"service_plan_guid"`
}

type ServiceInstanceResource struct {
	Entity ServiceInstanceEntity `json:"entity"`
}
type ServicePlanData struct {
	Guid string `json:"guid"`
}
type ServicePlan struct {
	Data ServicePlanData `json:"data"`
}
type ServiceInstanceRelationships struct {
	ServicePlan ServicePlan `json:"service_plan"`
}
type ServiceInstance struct {
	Guid          string                       `json:"guid"`
	Type          string                       `json:"type"`
	Relationships ServiceInstanceRelationships `json:"relationships"`
}

func (c *Client) GetServiceInstance(serviceInstanceGuid string) (*ServiceInstance, error) {

	theUrl := fmt.Sprintf("/v3/service_instances/%s?fields[service_plan]=name,guid", serviceInstanceGuid)
	serviceInstance, err := ResourceRetriever[ServiceInstance]{c}.Get(theUrl)
	if err != nil {
		return nil, fmt.Errorf("failed GetServiceInstance guid(%s): %w", serviceInstanceGuid, err)
	}
	return &serviceInstance, err
}
