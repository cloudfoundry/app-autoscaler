package cf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"code.cloudfoundry.org/lager"
)

type ServicePlanEntity struct {
	UniqueId string `json:"unique_id"`
}

type ServicePlanResource struct {
	Entity ServicePlanEntity `json:"entity"`
}

func (c *Client) getBrokerPlanGuid(ccServicePlanGuid string) (string, error) {
	logger := c.logger.Session("cf-client-get-broker-plan-guid", lager.Data{"ccServicePlanGuid": ccServicePlanGuid})
	logger.Debug("start")
	defer logger.Debug("end")

	c.planMapsLock.Lock()
	defer c.planMapsLock.Unlock()

	if g, ok := c.ccServicePlanToBrokerPlanGuid[ccServicePlanGuid]; ok {
		return g, nil
	}

	servicePlansUrl, err := url.Parse(c.conf.API)
	if err != nil {
		return "", fmt.Errorf("cf-client-get-broker-plan-guid: failed to parse CF API URL: %w", err)
	}
	servicePlansUrl.Path = servicePlansUrl.Path + path.Join(ServicePlansPath, ccServicePlanGuid)

	logger.Info("created-url", lager.Data{"url": servicePlansUrl.String()})

	req, err := http.NewRequest("GET", servicePlansUrl.String(), nil)
	if err != nil {
		logger.Error("new-request", err)
		return "", fmt.Errorf("cf-client-get-broker-plan-guid: failed to create request to CF API: %w", err)
	}

	tokens, _ := c.GetTokens()
	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)

	var resp *http.Response
	resp, err = c.httpClient.Do(req)

	if err != nil {
		logger.Error("do-request", err)
		return "", fmt.Errorf("cf-client-get-broker-plan-guid: failed to execute request to CF API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("cf-client-get-broker-plan-guid: failed to get service plan: %s [%d] %s", servicePlansUrl.String(), resp.StatusCode, resp.Status)
		logger.Error("get-response", err)
		return "", err
	}

	result := &ServicePlanResource{}

	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		logger.Error("decode", err)
		return "", fmt.Errorf("cf-client-get-broker-plan-guid: failed to decode response from CF API: %w", err)
	}

	brokerPlanGuid := result.Entity.UniqueId
	logger.Info("found-guid", lager.Data{"brokerPlanGuid": brokerPlanGuid})
	c.ccServicePlanToBrokerPlanGuid[ccServicePlanGuid] = brokerPlanGuid
	c.brokerPlanGuidToCCServicePlanGuid[brokerPlanGuid] = ccServicePlanGuid

	return brokerPlanGuid, nil
}

func (c *Client) GetServicePlan(serviceInstanceGuid string) (string, error) {
	logger := c.logger.Session("cf-client-get-service-plan", lager.Data{"serviceInstanceGuid": serviceInstanceGuid})
	logger.Debug("start")
	defer logger.Debug("end")

	c.instanceMapLock.Lock()
	defer c.instanceMapLock.Unlock()

	if g, ok := c.serviceInstanceGuidToBrokerPlanGuid[serviceInstanceGuid]; ok {
		return g, nil
	}

	result, err := c.GetServiceInstance(serviceInstanceGuid)
	if err != nil {
		return "", err
	}

	servicePlanGuid := result.Relationships.ServicePlan.Data.Guid
	logger.Info("found-guid", lager.Data{"servicePlanGuid": servicePlanGuid})
	brokerPlanGuid, err := c.getBrokerPlanGuid(servicePlanGuid)
	if err != nil {
		logger.Error("cc-plan-to-broker-plan", err)
		return "", fmt.Errorf("cf-client-get-service-plan: failed to translate Cloud Controller service plan to broker service plan: %w", err)
	}

	c.serviceInstanceGuidToBrokerPlanGuid[serviceInstanceGuid] = brokerPlanGuid

	return c.serviceInstanceGuidToBrokerPlanGuid[serviceInstanceGuid], nil
}
