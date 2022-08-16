package cf

import (
	"fmt"

	"code.cloudfoundry.org/lager"
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

func (c *Client) GetBrokerPlanGuid(ccServicePlanGuid string) (string, error) {
	return c.brokerPlanGuid.Func(ccServicePlanGuid)
}

func (c *Client) getBrokerPlanGuid(ccServicePlanGuid string) (string, error) {
	result, err := c.GetServicePlanResource(ccServicePlanGuid)
	if err != nil {
		return "", err
	}

	brokerPlanGuid := result.BrokerCatalog.Id
	c.logger.Info("found-guid", lager.Data{"brokerPlanGuid": brokerPlanGuid})
	return brokerPlanGuid, nil
}

/*GetServicePlanResource
 *  v3 api docs https://v3-apidocs.cloudfoundry.org/version/3.123.0/index.html#service-plans
 */
func (c *Client) GetServicePlanResource(servicePlanGuid string) (*ServicePlan, error) {
	theUrl := fmt.Sprintf("/v3/service_plans/%s", servicePlanGuid)
	plan, err := ResourceRetriever[*ServicePlan]{c}.Get(theUrl)
	if err != nil {
		return plan, fmt.Errorf("failed GetServicePlanResource(%s): %w", servicePlanGuid, err)
	}
	return plan, nil
}

// GetServicePlan
// This function does not really get a service plan but the service plan's broker catalog id
func (c *Client) GetServicePlan(serviceInstanceGuid string) (string, error) {
	return c.servicePlan.Func(serviceInstanceGuid)
}

func (c *Client) getServicePlan(serviceInstanceGuid string) (string, error) {
	result, err := c.GetServiceInstance(serviceInstanceGuid)
	if err != nil {
		return "", err
	}

	servicePlanGuid := result.Relationships.ServicePlan.Data.Guid
	c.logger.Info("found-guid", lager.Data{"servicePlanGuid": servicePlanGuid})
	brokerPlanGuid, err := c.GetBrokerPlanGuid(servicePlanGuid)
	if err != nil {
		c.logger.Error("cc-plan-to-broker-plan", err)
		return "", fmt.Errorf("cf-client-get-service-plan: failed to translate Cloud Controller service plan to broker service plan: %w", err)
	}
	return brokerPlanGuid, nil
}
