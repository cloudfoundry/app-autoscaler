package cf

import (
	"context"
	"fmt"
)

type ServicePlanEntity struct {
	UniqueId string `json:"unique_id"`
}

type ServicePlanResource struct {
	Entity ServicePlanEntity `json:"entity"`
}

type (
	ServicePlan struct {
		Guid          string        `json:"guid"`
		BrokerCatalog BrokerCatalog `json:"broker_catalog"`
	}
	BrokerCatalog struct {
		Id string `json:"id"`
	}
)

/*GetServicePlan
 * Gets the service plan for the guid given
 *  v3 api docs https://v3-apidocs.cloudfoundry.org/version/3.123.0/index.html#service-plans
 */
func (c *Client) GetServicePlan(servicePlanGuid string) (*ServicePlan, error) {
	return c.CtxClient.GetServicePlan(context.Background(), servicePlanGuid)
}

func (c *CtxClient) GetServicePlan(ctx context.Context, servicePlanGuid string) (*ServicePlan, error) {
	theUrl := fmt.Sprintf("/v3/service_plans/%s", servicePlanGuid)
	plan, err := ResourceRetriever[*ServicePlan]{AuthenticatedClient{c}}.Get(ctx, theUrl)
	if err != nil {
		return plan, fmt.Errorf("failed GetServicePlan(%s): %w", servicePlanGuid, err)
	}
	return plan, nil
}
