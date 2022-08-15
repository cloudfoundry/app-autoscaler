package cf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"code.cloudfoundry.org/lager"
)

const (
	ServicePlansPath = "v2/service_plans"
)

type ServicePlanEntity struct {
	UniqueId string `json:"unique_id"`
}

type ServicePlanResource struct {
	Entity ServicePlanEntity `json:"entity"`
}

func (c *Client) GetBrokerPlanGuid(ccServicePlanGuid string) (string, error) {
	return c.brokerPlanGuid.Func(ccServicePlanGuid)
}

func (c *Client) getBrokerPlanGuid(ccServicePlanGuid string) (string, error) {
	result, err := c.GetServicePlanResource(ccServicePlanGuid)
	if err != nil {
		return "", err
	}

	brokerPlanGuid := result.Entity.UniqueId
	c.logger.Info("found-guid", lager.Data{"brokerPlanGuid": brokerPlanGuid})
	return brokerPlanGuid, nil
}

func (c *Client) GetServicePlanResource(ccServicePlanGuid string) (*ServicePlanResource, error) {
	servicePlansUrl, err := url.Parse(c.conf.API)
	if err != nil {
		return nil, fmt.Errorf("cf-client-get-broker-plan-guid: failed to parse CF API URL: %w", err)
	}
	servicePlansUrl.Path = servicePlansUrl.Path + path.Join(ServicePlansPath, ccServicePlanGuid)

	req, err := http.NewRequest("GET", servicePlansUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("cf-client-get-broker-plan-guid: failed to create request to CF API: %w", err)
	}

	tokens, _ := c.GetTokens()
	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)

	var resp *http.Response
	resp, err = c.httpClient.Do(req)

	if err != nil {
		c.logger.Error("do-request", err)
		return nil, fmt.Errorf("cf-client-get-broker-plan-guid: failed to execute request to CF API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("cf-client-get-broker-plan-guid: failed to get service plan: %s [%d] %s", servicePlansUrl.String(), resp.StatusCode, resp.Status)
		c.logger.Error("get-response", err)
		return nil, err
	}

	result := &ServicePlanResource{}

	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		c.logger.Error("decode", err)
		return nil, fmt.Errorf("cf-client-get-broker-plan-guid: failed to decode response from CF API: %w", err)
	}
	return result, nil
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
