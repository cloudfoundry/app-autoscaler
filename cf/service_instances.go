package cf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

const (
	ServiceInstancesPath = "/v2/service_instances"
	ServicePlansPath     = "v2/service_plans"
)

type ServiceInstanceEntity struct {
	ServicePlanGuid string `json:"service_plan_guid"`
}

type ServiceInstanceResource struct {
	Entity ServiceInstanceEntity `json:"entity"`
}

func (c *Client) GetServiceInstance(serviceInstanceGuid string) (*ServiceInstanceResource, error) {
	serviceInstancesUrl, err := url.Parse(c.conf.API)
	if err != nil {
		return nil, fmt.Errorf("cf-client-get-service-plan: failed to parse CF API URL: %w", err)
	}
	serviceInstancesUrl.Path = path.Join(ServiceInstancesPath, serviceInstanceGuid)

	req, err := http.NewRequest("GET", serviceInstancesUrl.String(), nil)
	if err != nil {
		c.logger.Error("new-request", err)
		return nil, fmt.Errorf("cf-client-get-service-plan: failed to create request to CF API: %w", err)
	}

	tokens, _ := c.GetTokens()
	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)

	var resp *http.Response
	resp, err = c.httpClient.Do(req)

	if err != nil {
		c.logger.Error("do-request", err)
		return nil, fmt.Errorf("cf-client-get-service-plan: failed to execute request to CF API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("cf-client-get-service-plan: failed to get service plan: %s [%d] %s", serviceInstancesUrl.String(), resp.StatusCode, resp.Status)
		c.logger.Error("get-response", err)
		return nil, err
	}

	result := &ServiceInstanceResource{}

	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		c.logger.Error("decode", err)
		return nil, fmt.Errorf("cf-client-get-service-plan: failed to decode response from CF API: %w", err)
	}
	return result, nil
}
