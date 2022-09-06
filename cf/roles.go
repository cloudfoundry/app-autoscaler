package cf

import (
	"context"
	"fmt"
	"net/url"
)

const (
	RoleOrganisationUser           RoleType = "organization_user"
	RoleOrganizationAuditor        RoleType = "organization_auditor"
	RoleOrganizationManager        RoleType = "organization_manager"
	RoleOrganizationBillingManager RoleType = "organization_billing_manager"
	RoleSpaceAuditor               RoleType = "space_auditor"
	RoleSpaceDeveloper             RoleType = "space_developer"
	RoleSpaceManager               RoleType = "space_manager"
	RoleSpaceSupporter             RoleType = "space_supporter"
)

type (
	Role struct {
		Guid string   `json:"guid"`
		Type RoleType `json:"type"`
	}
	RoleType string
)

type Roles []Role

type SpaceId string
type UserId string

func (r Roles) HasRole(roleType RoleType) bool {
	for _, role := range r {
		if role.Type == roleType {
			return true
		}
	}
	return false
}

/*GetSpaceDeveloperRoles
 * Get role information given a set of filters
 * from the v3 api https://v3-apidocs.cloudfoundry.org/version/3.122.0/index.html#roles
 */
func (c *Client) GetSpaceDeveloperRoles(spaceId SpaceId, userId UserId) (Roles, error) {
	return c.CtxClient.GetSpaceDeveloperRoles(context.Background(), spaceId, userId)
}

func (c *CtxClient) GetSpaceDeveloperRoles(ctx context.Context, spaceId SpaceId, userId UserId) (Roles, error) {
	parameters := url.Values{}
	parameters.Add("types", "space_developer")
	parameters.Add("space_guids", string(spaceId))
	parameters.Add("user_guids", string(userId))
	params := parameters.Encode()
	theUrl := fmt.Sprintf("/v3/roles?%s", params)
	roles, err := PagedResourceRetriever[Role]{AuthenticatedClient{c}}.GetAllPages(ctx, theUrl)
	if err != nil {
		return nil, fmt.Errorf("failed GetSpaceDeveloperRoles spaceId(%s) userId(%s): %w", spaceId, userId, err)
	}
	return roles, err
}
