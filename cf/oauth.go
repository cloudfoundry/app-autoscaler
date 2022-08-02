package cf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.cloudfoundry.org/lager"
)

const (
	CCAdminScope = "cloud_controller.admin"
)

var (
	ErrUnauthrorized      = fmt.Errorf(http.StatusText(http.StatusUnauthorized))
	ErrInvalidTokenFormat = fmt.Errorf("Invalid token format")
)

func (c *Client) IsUserSpaceDeveloper(userToken string, appId string) (bool, error) {
	userId, err := c.getUserId(userToken)
	if err != nil {
		return false, err
	}

	spaceId, err := c.getSpaceId(userToken, appId)
	if err != nil {
		return false, err
	}

	rolesEndpoint := c.getSpaceDeveloperRolesEndpoint(userId, spaceId)

	req, err := http.NewRequest("GET", rolesEndpoint, nil)
	if err != nil {
		c.logger.Error("Failed to create get roles request", err, lager.Data{"rolesEndpoint": rolesEndpoint})
		return false, err
	}
	req.Header.Set("Authorization", userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get roles, request failed", err, lager.Data{"rolesEndpoint": rolesEndpoint})
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		c.logger.Error("Failed to get roles, token invalid", nil, lager.Data{"rolesEndpoint": rolesEndpoint, "statusCode": resp.StatusCode})
		return false, ErrUnauthrorized
	} else if resp.StatusCode != http.StatusOK {
		c.logger.Error("Failed to get roles", nil, lager.Data{"rolesEndpoint": rolesEndpoint, "statusCode": resp.StatusCode})
		return false, fmt.Errorf("Failed to get roles, statusCode : %v", resp.StatusCode)
	}

	roles := struct {
		Pagination struct {
			Total int `json:"total_results"`
		} `json:"pagination"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&roles)
	if err != nil {
		c.logger.Error("Failed to parse roles response body", err, lager.Data{"rolesEndpoint": rolesEndpoint})
		return false, err
	}

	isSpaceDeveloperOnAppSapce := roles.Pagination.Total > 0
	if !isSpaceDeveloperOnAppSapce {
		c.logger.Error("User without SpaceDeveloper role in the apps space tried to access API", nil, lager.Data{"rolesEndpoint": rolesEndpoint})
	}
	return isSpaceDeveloperOnAppSapce, nil
}

func (c *Client) IsUserAdmin(userToken string) (bool, error) {
	scopes, err := c.getUserScope(userToken)
	if err != nil {
		return false, err
	}

	for _, scope := range scopes {
		if scope == CCAdminScope {
			c.logger.Info("user is cc admin")
			return true, nil
		}
	}

	return false, nil
}

func (c *Client) getUserScope(userToken string) ([]string, error) {
	userScopeEndpoint, err := c.getUserScopeEndpoint(userToken)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", userScopeEndpoint, nil)
	if err != nil {
		c.logger.Error("Failed to create getuserscope request", err, lager.Data{"userScopeEndpoint": userScopeEndpoint})
		return nil, err
	}
	req.SetBasicAuth(c.conf.ClientID, c.conf.Secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to getuserscope, request failed", err, lager.Data{"userScopeEndpoint": userScopeEndpoint})
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Failed to get user scope", nil, lager.Data{"userScopeEndpoint": userScopeEndpoint, "statusCode": resp.StatusCode})
		return nil, fmt.Errorf("Failed to get user scope, statusCode : %v", resp.StatusCode)
	}

	userScope := struct {
		Scope []string `json:"scope"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&userScope)
	if err != nil {
		c.logger.Error("Failed to parse user scope response body", err, lager.Data{"userScopeEndpoint": userScopeEndpoint})
		return nil, err
	}
	return userScope.Scope, nil
}

func (c *Client) getUserId(userToken string) (string, error) {
	userInfoEndpoint := c.getUserInfoEndpoint()

	req, err := http.NewRequest("GET", userInfoEndpoint, nil)
	if err != nil {
		c.logger.Error("Failed to get user info, create request failed", err, lager.Data{"userInfoEndpoint": userInfoEndpoint})
		return "", err
	}
	req.Header.Set("Authorization", userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get user info, request failed", err, lager.Data{"userInfoEndpoint": userInfoEndpoint})
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		c.logger.Error("Failed to get user info, token invalid", nil, lager.Data{"userInfoEndpoint": userInfoEndpoint, "statusCode": resp.StatusCode})
		return "", ErrUnauthrorized
	} else if resp.StatusCode != http.StatusOK {
		c.logger.Error("Failed to get user info", nil, lager.Data{"userInfoEndpoint": userInfoEndpoint, "statusCode": resp.StatusCode})
		return "", fmt.Errorf("Failed to get user info, statuscode :%v", resp.StatusCode)
	}

	userInfo := struct {
		UserId string `json:"user_id"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		c.logger.Error("Failed to parse user info response body", err, lager.Data{"userInfoEndpoint": userInfoEndpoint})
		return "", err
	}

	return userInfo.UserId, nil
}

func (c *Client) getSpaceId(userToken string, appId string) (string, error) {
	appsEndpoint := c.getAppsEndpoint(appId)

	req, err := http.NewRequest("GET", appsEndpoint, nil)
	if err != nil {
		c.logger.Error("Failed to create apps request", err, lager.Data{"appsEndpoint": appsEndpoint})
		return "", err
	}
	req.Header.Set("Authorization", userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get app, request failed", err, lager.Data{"appsEndpoint": appsEndpoint})
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound {
		c.logger.Error("Failed to get app info, token invalid", nil, lager.Data{"appsEndpoint": appsEndpoint, "statusCode": resp.StatusCode})
		return "", ErrUnauthrorized
	} else if resp.StatusCode != http.StatusOK {
		c.logger.Error("Failed to get app info", nil, lager.Data{"appsEndpoint": appsEndpoint, "statusCode": resp.StatusCode})
		return "", fmt.Errorf("Failed to get app, statusCode : %v", resp.StatusCode)
	}

	app := struct {
		Relationships struct {
			Space struct {
				Data struct {
					GUID string `json:"guid"`
				} `json:"data"`
			} `json:"space"`
		} `json:"relationships"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&app)
	if err != nil {
		c.logger.Error("Failed to parse app response body", err, lager.Data{"appsEndpoint": appsEndpoint})
		return "", err
	}

	spaceId := app.Relationships.Space.Data.GUID

	if spaceId == "" {
		c.logger.Error("Failed to retrieve space guid", nil, lager.Data{"appsEndpoint": appsEndpoint, "appObject": app})
		return "", fmt.Errorf("Failed to retrieve space guid")
	}

	return spaceId, nil
}

func (c *Client) getUserScopeEndpoint(userToken string) (string, error) {
	parameters := url.Values{}
	parameters.Add("token", strings.Split(userToken, " ")[1])

	userScopeEndpoint := c.endpoints.TokenEndpoint + "/check_token?" + parameters.Encode()
	return userScopeEndpoint, nil
}

func (c *Client) getUserInfoEndpoint() string {
	return c.endpoints.TokenEndpoint + "/userinfo"
}

func (c *Client) getAppsEndpoint(appId string) string {
	return c.conf.API + "/v3/apps/" + appId
}

func (c *Client) getSpaceDeveloperRolesEndpoint(userId string, spaceId string) string {
	parameters := url.Values{}
	parameters.Add("types", "space_developer")
	parameters.Add("space_guids", spaceId)
	parameters.Add("user_guids", userId)
	spaceDeveloperRolesEndpoint := c.conf.API + "/v3/roles?" + parameters.Encode()
	return spaceDeveloperRolesEndpoint
}
