package cf

import (
	"fmt"
)

type (
	Endpoints struct {
		CloudControllerV3 Href `json:"cloud_controller_v3"`
		NetworkPolicyV0   Href `json:"network_policy_v0"`
		NetworkPolicyV1   Href `json:"network_policy_v1"`
		Login             Href `json:"login"`
		Uaa               Href `json:"uaa"`
		Routing           Href `json:"routing"`
		Logging           Href `json:"logging"`
		LogCache          Href `json:"log_cache"`
		LogStream         Href `json:"log_stream"`
		AppSsh            Href `json:"app_ssh"`
	}
	EndpointsResponse struct {
		Links Endpoints `json:"links"`
	}
)

func (c *Client) GetEndpoints() (Endpoints, error) {
	return c.endpoints.Get()
}

func (c *Client) getEndpoints() (Endpoints, error) {
	endpoints, err := ResourceRetriever[EndpointsResponse]{&ResourceRetriever[EndpointsResponse]{c}}.Get("/")
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed GetEndpoints: %w", err)
	}
	return endpoints.Links, err
}
