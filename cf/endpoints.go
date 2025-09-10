package cf

import (
	"context"
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

/*GetEndpoints
 * Gets the root information from the v3 api. This consists mostly of the endpoints needed to use the cf environment.
 */
func (c *Client) GetEndpoints() (Endpoints, error) {
	//nolint:staticcheck // QF1008: embedded field access is intentional for API design
	return c.CtxClient.endpoints.Get(context.Background())
}

func (c *CtxClient) GetEndpoints(ctx context.Context) (Endpoints, error) {
	endpoints, err := ResourceRetriever[EndpointsResponse]{&ResourceRetriever[EndpointsResponse]{c}}.Get(ctx, "/")
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed GetEndpoints: %w", err)
	}
	return endpoints.Links, err
}
