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
	result, err := c.GetServicePlan(ccServicePlanGuid)
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
func (c *Client) GetServicePlan(servicePlanGuid string) (*ServicePlan, error) {
	theUrl := fmt.Sprintf("/v3/service_plans/%s", servicePlanGuid)
	plan, err := ResourceRetriever[*ServicePlan]{c}.Get(theUrl)
	if err != nil {
		return plan, fmt.Errorf("failed GetServicePlan(%s): %w", servicePlanGuid, err)
	}
	return plan, nil
}
