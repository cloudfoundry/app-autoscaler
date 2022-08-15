package cf

import (
	"fmt"
)

type (
	ServicePlanData struct {
		Guid string `json:"guid"`
	}
	ServicePlan struct {
		Data ServicePlanData `json:"data"`
	}
	ServiceInstanceRelationships struct {
		ServicePlan ServicePlan `json:"service_plan"`
	}
	ServiceInstance struct {
		Guid          string                       `json:"guid"`
		Type          string                       `json:"type"`
		Relationships ServiceInstanceRelationships `json:"relationships"`
	}
)

func (c *Client) GetServiceInstance(serviceInstanceGuid string) (*ServiceInstance, error) {
	theUrl := fmt.Sprintf("/v3/service_instances/%s", serviceInstanceGuid)
	serviceInstance, err := ResourceRetriever[ServiceInstance]{c}.Get(theUrl)
	if err != nil {
		return nil, fmt.Errorf("failed GetServiceInstance guid(%s): %w", serviceInstanceGuid, err)
	}
	return &serviceInstance, err
}
