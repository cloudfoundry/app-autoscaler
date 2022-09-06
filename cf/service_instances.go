package cf

import (
	"context"
	"fmt"
)

type (
	ServicePlanData struct {
		Guid string `json:"guid"`
	}
	ServicePlanRelation struct {
		Data ServicePlanData `json:"data"`
	}
	ServiceInstanceRelationships struct {
		ServicePlan ServicePlanRelation `json:"service_plan"`
	}
	ServiceInstance struct {
		Guid          string                       `json:"guid"`
		Type          string                       `json:"type"`
		Relationships ServiceInstanceRelationships `json:"relationships"`
	}
)

func (c *Client) GetServiceInstance(serviceInstanceGuid string) (*ServiceInstance, error) {
	return c.CtxClient.GetServiceInstance(context.Background(), serviceInstanceGuid)
}

func (c *CtxClient) GetServiceInstance(ctx context.Context, serviceInstanceGuid string) (*ServiceInstance, error) {
	theUrl := fmt.Sprintf("/v3/service_instances/%s", serviceInstanceGuid)
	serviceInstance, err := ResourceRetriever[ServiceInstance]{AuthenticatedClient{c}}.Get(ctx, theUrl)
	if err != nil {
		return nil, fmt.Errorf("failed GetServiceInstance guid(%s): %w", serviceInstanceGuid, err)
	}
	return &serviceInstance, err
}
