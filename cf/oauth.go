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

func (c *cfClient) IsUserSpaceDeveloper(userToken string, appId string) (bool, error) {
	userId, err := c.getUserId(userToken)
	if err != nil {
		return false, err
	}

	spaceDeveloperPermissionEndpoint := c.getSpaceDeveloperPermissionEndpoint(userId, appId)

	req, err := http.NewRequest("GET", spaceDeveloperPermissionEndpoint, nil)
	if err != nil {
		c.logger.Error("Failed to create check space dev permission request", err, lager.Data{"spaceDeveloperPermissionEndpoint": spaceDeveloperPermissionEndpoint})
		return false, err
	}
	req.Header.Set("Authorization", userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get user space dev permission, request failed", err, lager.Data{"spaceDeveloperPermissionEndpoint": spaceDeveloperPermissionEndpoint})
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Failed to get user space dev permission", nil, lager.Data{"spaceDeveloperPermissionEndpoint": spaceDeveloperPermissionEndpoint, "statusCode": resp.StatusCode})
		return false, fmt.Errorf("Failed to get space developer permission, statusCode : %v", resp.StatusCode)
	}

	spaces := struct {
		Total int `json:"total_results"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&spaces)
	if err != nil {
		c.logger.Error("Failed to parse user space dev permission response body", err, lager.Data{"spaceDeveloperPermissionEndpoint": spaceDeveloperPermissionEndpoint})
		return false, err
	}
	return spaces.Total > 0, nil
}

func (c *cfClient) IsUserAdmin(userToken string) (bool, error) {
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

func (c *cfClient) getUserScope(userToken string) ([]string, error) {
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

func (c *cfClient) getUserId(userToken string) (string, error) {
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

func (c *cfClient) getUserScopeEndpoint(userToken string) (string, error) {

	parameters := url.Values{}
	parameters.Add("token", strings.Split(userToken, " ")[1])

	userScopeEndpoint := c.endpoints.TokenEndpoint + "/check_token?" + parameters.Encode()
	return userScopeEndpoint, nil
}

func (c *cfClient) getSpaceDeveloperPermissionEndpoint(userId string, appId string) string {
	parameters := url.Values{}
	parameters.Add("app_guid", appId)
	parameters.Add("developer_guid", userId)
	spaceDeveloperPermissionEndpoint := c.conf.API + "/v2/users/" + userId + "/spaces?" + parameters.Encode()
	return spaceDeveloperPermissionEndpoint
}

func (c *cfClient) getUserInfoEndpoint() string {
	return c.endpoints.TokenEndpoint + "/userinfo"
}
