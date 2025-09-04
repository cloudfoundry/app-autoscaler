package cf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"code.cloudfoundry.org/lager/v3"
)

const (
	CCAdminScope = "cloud_controller.admin"
)

var (
	ErrUnauthorized       = errors.New(http.StatusText(http.StatusUnauthorized))
	ErrInvalidTokenFormat = errors.New("invalid token format")
)

func (c *Client) IsUserSpaceDeveloper(userToken string, appId Guid) (bool, error) {
	return c.CtxClient.IsUserSpaceDeveloper(context.Background(), userToken, appId)
}

func (c *CtxClient) IsUserSpaceDeveloper(ctx context.Context, userToken string, appId Guid) (bool, error) {
	userId, err := c.getUserId(ctx, userToken)
	if err != nil {
		if errors.Is(ErrUnauthorized, err) {
			c.logger.Error("getUserId: token Not authorized", err)
			return false, nil
		}
		return false, fmt.Errorf("failed IsUserSpaceDeveloper for appId(%s): %w", appId, err)
	}

	spaceId, err := c.getSpaceId(ctx, appId)
	if err != nil {
		return false, fmt.Errorf("failed IsUserSpaceDeveloper for appId(%s): %w", appId, err)
	}

	roles, err := c.GetSpaceDeveloperRoles(ctx, spaceId, userId)
	if err != nil {
		if IsNotFound(err) {
			c.logger.Info("GetSpaceDeveloperRoles: Not not found", lager.Data{"userId": userId, "spaceid": spaceId})
			return false, nil
		}
		return false, fmt.Errorf("failed IsUserSpaceDeveloper userId(%s), spaceId(%s): %w", userId, spaceId, err)
	}

	isSpaceDeveloperOnAppSpace := roles.HasRole(RoleSpaceDeveloper)
	if !isSpaceDeveloperOnAppSpace {
		c.logger.Error("User without SpaceDeveloper role in the apps space tried to access API", nil)
	}
	return isSpaceDeveloperOnAppSpace, nil
}

func (c *Client) IsUserAdmin(userToken string) (bool, error) {
	return c.CtxClient.IsUserAdmin(context.Background(), userToken)
}
func (c *CtxClient) IsUserAdmin(ctx context.Context, userToken string) (bool, error) {
	introspectionResponse, err := c.introspectToken(ctx, userToken)
	if err != nil {
		return false, err
	}

	for _, scope := range introspectionResponse.Scopes {
		if scope == CCAdminScope {
			c.logger.Info("user is cc admin")
			return true, nil
		}
	}

	return false, nil
}

func (c *CtxClient) getUserId(ctx context.Context, userToken string) (UserId, error) {
	endpoints, err := c.GetEndpoints(ctx)
	if err != nil {
		return "", err
	}
	userInfoEndpoint := endpoints.Uaa.Url + "/userinfo"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoEndpoint, nil)
	if err != nil {
		c.logger.Error("Failed to get user info, create request failed", err, lager.Data{"userInfoEndpoint": userInfoEndpoint})
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set("Content-Type", "application/json")
	c.setUserAgent(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		c.logger.Error("Failed to get user info, request failed", err, lager.Data{"userInfoEndpoint": userInfoEndpoint})
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		userInfo := struct {
			UserId UserId `json:"user_id"`
		}{}
		err = json.NewDecoder(resp.Body).Decode(&userInfo)
		if err != nil {
			c.logger.Error("Failed to parse user info response body", err, lager.Data{"userInfoEndpoint": userInfoEndpoint})
			return "", err
		}
		return userInfo.UserId, nil
	case http.StatusNotFound, http.StatusUnauthorized:
		c.logger.Error("Failed to get user info, token invalid", nil, lager.Data{"userInfoEndpoint": userInfoEndpoint, "statusCode": resp.StatusCode})
		return "", ErrUnauthorized
	default:
		c.logger.Error("Failed to get user info", nil, lager.Data{"userInfoEndpoint": userInfoEndpoint, "statusCode": resp.StatusCode})
		return "", fmt.Errorf("Failed to get user info, statuscode :%v", resp.StatusCode)
	}
}

func (c *CtxClient) getSpaceId(ctx context.Context, appId Guid) (SpaceId, error) {
	app, err := c.GetApp(ctx, appId)
	if err != nil {
		return "", fmt.Errorf("getSpaceId failed: %w", err)
	}

	spaceId := app.Relationships.Space.Data.Guid

	if spaceId == "" {
		return "", fmt.Errorf("empty space-guid: failed to retrieve it for app with id %s", appId)
	}

	return spaceId, nil
}
