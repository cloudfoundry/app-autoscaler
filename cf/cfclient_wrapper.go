package cf

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/config"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

const uaaRequestTimeout = 30 * time.Second

type CFClientWrapper struct {
	cfClient    *client.Client
	conf        *Config
	logger      lager.Logger
	clk         clock.Clock
	tokenInfoMu sync.RWMutex
	tokenInfo   TokensInfo
	endpointsMu sync.RWMutex
	endpoints   *Endpoints
}

type CtxClientWrapper struct {
	*CFClientWrapper
}

var _ CFClient = &CFClientWrapper{}
var _ ContextClient = &CtxClientWrapper{}

func NewCFClientWrapper(conf *Config, logger lager.Logger, clk clock.Clock) (*CFClientWrapper, error) {
	options := []config.Option{
		config.ClientCredentials(conf.ClientID, conf.Secret),
		config.UserAgent(GetUserAgent()),
	}
	if conf.SkipSSLValidation {
		options = append(options, config.SkipTLSValidation())
	}

	cfg, err := config.New(conf.API, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cfclient config: %w", err)
	}

	cfClient, err := client.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create cfclient: %w", err)
	}

	return &CFClientWrapper{
		cfClient: cfClient,
		conf:     conf,
		logger:   logger,
		clk:      clk,
	}, nil
}

func (w *CFClientWrapper) GetCtxClient() ContextClient {
	return &CtxClientWrapper{w}
}

func (w *CFClientWrapper) Login() error {
	return w.GetCtxClient().Login(context.Background())
}

func (c *CtxClientWrapper) Login(ctx context.Context) error {
	tokens, err := c.refreshToken(ctx)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	c.tokenInfoMu.Lock()
	defer c.tokenInfoMu.Unlock()
	c.tokenInfo.Tokens = tokens
	c.tokenInfo.grantTime = c.clk.Now()

	return nil
}

func (w *CFClientWrapper) InvalidateToken() {
	w.tokenInfoMu.Lock()
	defer w.tokenInfoMu.Unlock()
	w.tokenInfo.grantTime = time.Time{}
}

func (w *CFClientWrapper) RefreshAuthToken() (Tokens, error) {
	return w.GetCtxClient().RefreshAuthToken(context.Background())
}

func (c *CtxClientWrapper) RefreshAuthToken(ctx context.Context) (Tokens, error) {
	c.tokenInfoMu.Lock()
	defer c.tokenInfoMu.Unlock()

	if !c.tokenInfo.isTokenExpired(c.clk.Now) {
		return c.tokenInfo.Tokens, nil
	}

	tokens, err := c.refreshToken(ctx)
	if err != nil {
		return c.tokenInfo.Tokens, err
	}

	c.tokenInfo.Tokens = tokens
	c.tokenInfo.grantTime = c.clk.Now()

	return tokens, nil
}

func (c *CtxClientWrapper) refreshToken(ctx context.Context) (Tokens, error) {
	_, err := c.cfClient.Root.Get(ctx)
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to refresh token: %w", err)
	}
	return c.tokenInfo.Tokens, nil
}

func (w *CFClientWrapper) GetTokens() (Tokens, error) {
	return w.GetCtxClient().GetTokens(context.Background())
}

func (c *CtxClientWrapper) GetTokens(ctx context.Context) (Tokens, error) {
	info := c.getTokenInfo()
	if info.isTokenExpired(c.clk.Now) {
		return c.RefreshAuthToken(ctx)
	}
	return info.Tokens, nil
}

func (c *CtxClientWrapper) getTokenInfo() TokensInfo {
	c.tokenInfoMu.RLock()
	defer c.tokenInfoMu.RUnlock()
	return c.tokenInfo
}

func (w *CFClientWrapper) IsUserAdmin(userToken string) (bool, error) {
	return w.GetCtxClient().IsUserAdmin(context.Background(), userToken)
}

func (c *CtxClientWrapper) IsUserAdmin(ctx context.Context, userToken string) (bool, error) {
	resp, err := c.introspectToken(ctx, userToken)
	if err != nil {
		return false, err
	}
	if slices.Contains(resp.Scopes, CCAdminScope) {
		c.logger.Info("user is cc admin")
		return true, nil
	}
	return false, nil
}

func (w *CFClientWrapper) IsUserSpaceDeveloper(userToken string, appId Guid) (bool, error) {
	return w.GetCtxClient().IsUserSpaceDeveloper(context.Background(), userToken, appId)
}

func (c *CtxClientWrapper) IsUserSpaceDeveloper(ctx context.Context, userToken string, appId Guid) (bool, error) {
	userId, err := c.getUserId(ctx, userToken)
	if err != nil {
		if err == ErrUnauthorized {
			c.logger.Error("getUserId: token not authorized", err)
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
			c.logger.Info("GetSpaceDeveloperRoles: not found", lager.Data{"userId": userId, "spaceId": spaceId})
			return false, nil
		}
		return false, fmt.Errorf("failed IsUserSpaceDeveloper userId(%s), spaceId(%s): %w", userId, spaceId, err)
	}

	isSpaceDev := roles.HasRole(RoleSpaceDeveloper)
	if !isSpaceDev {
		c.logger.Error("user without SpaceDeveloper role tried to access API", nil)
	}
	return isSpaceDev, nil
}

func (w *CFClientWrapper) IsTokenAuthorized(token, clientId string) (bool, error) {
	return w.GetCtxClient().IsTokenAuthorized(context.Background(), token, clientId)
}

func (c *CtxClientWrapper) IsTokenAuthorized(ctx context.Context, token, clientId string) (bool, error) {
	resp, err := c.introspectToken(ctx, token)
	if err != nil {
		return false, err
	}
	return resp.Active && resp.ClientId == clientId, nil
}

func (c *CtxClientWrapper) newHTTPClient() *http.Client {
	httpClient := &http.Client{Timeout: uaaRequestTimeout}
	if c.conf.SkipSSLValidation {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}
	return httpClient
}

func (c *CtxClientWrapper) introspectToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	endpoints, err := c.GetEndpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints for introspection: %w", err)
	}

	introspectURL := strings.TrimSuffix(endpoints.Uaa.Url, "/") + "/introspect"
	body := "token=" + url.QueryEscape(token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, introspectURL, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create introspect request: %w", err)
	}

	credentials := base64.StdEncoding.EncodeToString([]byte(c.conf.ClientID + ":" + c.conf.Secret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := c.newHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspect token failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read introspect response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspect token failed with status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result IntrospectionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse introspection response: %w", err)
	}

	return &result, nil
}

func (c *CtxClientWrapper) getUserId(ctx context.Context, userToken string) (UserId, error) {
	endpoints, err := c.GetEndpoints(ctx)
	if err != nil {
		return "", err
	}

	userinfoURL := strings.TrimSuffix(endpoints.Uaa.Url, "/") + "/userinfo"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+userToken)

	resp, err := c.newHTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", ErrUnauthorized
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo failed with status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var userInfo struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(respBody, &userInfo); err != nil {
		return "", fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	return UserId(userInfo.UserID), nil
}

func (c *CtxClientWrapper) getSpaceId(ctx context.Context, appId Guid) (SpaceId, error) {
	app, err := c.GetApp(ctx, appId)
	if err != nil {
		return "", fmt.Errorf("getSpaceId failed: %w", err)
	}

	spaceId := app.Relationships.Space.Data.Guid
	if spaceId == "" {
		return "", fmt.Errorf("empty space-guid for app %s", appId)
	}

	return spaceId, nil
}

func (w *CFClientWrapper) GetEndpoints() (Endpoints, error) {
	return w.GetCtxClient().GetEndpoints(context.Background())
}

func (c *CtxClientWrapper) GetEndpoints(ctx context.Context) (Endpoints, error) {
	c.endpointsMu.RLock()
	if c.endpoints != nil {
		defer c.endpointsMu.RUnlock()
		return *c.endpoints, nil
	}
	c.endpointsMu.RUnlock()

	c.endpointsMu.Lock()
	defer c.endpointsMu.Unlock()

	if c.endpoints != nil {
		return *c.endpoints, nil
	}

	root, err := c.cfClient.Root.Get(ctx)
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed GetEndpoints: %w", MapCFClientError(err))
	}

	endpoints := mapRootToEndpoints(root)
	c.endpoints = &endpoints

	return endpoints, nil
}

func (w *CFClientWrapper) GetApp(appId Guid) (*App, error) {
	return w.GetCtxClient().GetApp(context.Background(), appId)
}

func (c *CtxClientWrapper) GetApp(ctx context.Context, appId Guid) (*App, error) {
	app, err := c.cfClient.Applications.Get(ctx, string(appId))
	if err != nil {
		return nil, fmt.Errorf("failed getting app '%s': %w", appId, MapCFClientError(err))
	}
	return mapResourceApp(app), nil
}

func (w *CFClientWrapper) GetAppProcesses(appId Guid, processTypes ...string) (Processes, error) {
	return w.GetCtxClient().GetAppProcesses(context.Background(), appId, processTypes...)
}

func (c *CtxClientWrapper) GetAppProcesses(ctx context.Context, appId Guid, processTypes ...string) (Processes, error) {
	opts := &client.ProcessListOptions{}
	if len(processTypes) > 0 {
		opts.Types = client.Filter{Values: processTypes}
	}

	processes, err := c.cfClient.Processes.ListForAppAll(ctx, string(appId), opts)
	if err != nil {
		return nil, fmt.Errorf("failed GetAppProcesses '%s': %w", appId, MapCFClientError(err))
	}

	return mapResourceProcesses(processes), nil
}

func (w *CFClientWrapper) GetAppAndProcesses(appId Guid) (*AppAndProcesses, error) {
	return w.GetCtxClient().GetAppAndProcesses(context.Background(), appId)
}

func (c *CtxClientWrapper) GetAppAndProcesses(ctx context.Context, appId Guid) (*AppAndProcesses, error) {
	var wg sync.WaitGroup
	var app *App
	var processes Processes
	var appErr, procErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		app, appErr = c.GetApp(ctx, appId)
	}()
	go func() {
		defer wg.Done()
		processes, procErr = c.GetAppProcesses(ctx, appId, ProcessTypeWeb)
	}()
	wg.Wait()

	if appErr != nil {
		return nil, fmt.Errorf("get state&instances failed: %w", appErr)
	}
	if procErr != nil {
		return nil, fmt.Errorf("get state&instances failed: %w", procErr)
	}
	return &AppAndProcesses{App: app, Processes: processes}, nil
}

func (w *CFClientWrapper) ScaleAppWebProcess(appId Guid, numberOfProcesses int) error {
	return w.GetCtxClient().ScaleAppWebProcess(context.Background(), appId, numberOfProcesses)
}

func (c *CtxClientWrapper) ScaleAppWebProcess(ctx context.Context, appId Guid, instances int) error {
	processes, err := c.cfClient.Processes.ListForAppAll(ctx, string(appId), &client.ProcessListOptions{
		Types: client.Filter{Values: []string{ProcessTypeWeb}},
	})
	if err != nil {
		return fmt.Errorf("failed to get web process for app '%s': %w", appId, MapCFClientError(err))
	}

	if len(processes) == 0 {
		return fmt.Errorf("no web process found for app '%s'", appId)
	}

	_, err = c.cfClient.Processes.Scale(ctx, processes[0].GUID, &resource.ProcessScale{
		Instances: &instances,
	})
	if err != nil {
		return fmt.Errorf("failed scaling app '%s' to %d: %w", appId, instances, MapCFClientError(err))
	}

	return nil
}

func (w *CFClientWrapper) GetServiceInstance(guid string) (*ServiceInstance, error) {
	return w.GetCtxClient().GetServiceInstance(context.Background(), guid)
}

func (c *CtxClientWrapper) GetServiceInstance(ctx context.Context, guid string) (*ServiceInstance, error) {
	si, err := c.cfClient.ServiceInstances.Get(ctx, guid)
	if err != nil {
		return nil, fmt.Errorf("failed GetServiceInstance(%s): %w", guid, MapCFClientError(err))
	}
	return mapResourceServiceInstance(si), nil
}

func (w *CFClientWrapper) GetServicePlan(guid string) (*ServicePlan, error) {
	return w.GetCtxClient().GetServicePlan(context.Background(), guid)
}

func (c *CtxClientWrapper) GetServicePlan(ctx context.Context, guid string) (*ServicePlan, error) {
	sp, err := c.cfClient.ServicePlans.Get(ctx, guid)
	if err != nil {
		return nil, fmt.Errorf("failed GetServicePlan(%s): %w", guid, MapCFClientError(err))
	}
	return mapResourceServicePlan(sp), nil
}

func (c *CtxClientWrapper) GetSpaceDeveloperRoles(ctx context.Context, spaceId SpaceId, userId UserId) (Roles, error) {
	opts := &client.RoleListOptions{
		Types:      client.Filter{Values: []string{string(RoleSpaceDeveloper)}},
		SpaceGUIDs: client.Filter{Values: []string{string(spaceId)}},
		UserGUIDs:  client.Filter{Values: []string{string(userId)}},
	}

	roles, err := c.cfClient.Roles.ListAll(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed GetSpaceDeveloperRoles spaceId(%s) userId(%s): %w", spaceId, userId, MapCFClientError(err))
	}

	return mapResourceRoles(roles), nil
}

func mapRootToEndpoints(root *resource.Root) Endpoints {
	return Endpoints{
		CloudControllerV3: Href{Url: root.Links.CloudControllerV3.Href},
		NetworkPolicyV0:   Href{Url: root.Links.NetworkPolicyV0.Href},
		NetworkPolicyV1:   Href{Url: root.Links.NetworkPolicyV1.Href},
		Login:             Href{Url: root.Links.Login.Href},
		Uaa:               Href{Url: root.Links.Uaa.Href},
		Routing:           Href{Url: root.Links.Routing.Href},
		Logging:           Href{Url: root.Links.Logging.Href},
		LogCache:          Href{Url: root.Links.LogCache.Href},
		LogStream:         Href{Url: root.Links.LogStream.Href},
		AppSsh:            Href{Url: root.Links.AppSSH.Href},
	}
}

func mapResourceApp(app *resource.App) *App {
	return &App{
		Guid:      app.GUID,
		Name:      app.Name,
		State:     string(app.State),
		CreatedAt: app.CreatedAt,
		UpdatedAt: app.UpdatedAt,
		Relationships: Relationships{
			Space: &Space{
				Data: SpaceData{
					Guid: SpaceId(app.Relationships.Space.Data.GUID),
				},
			},
		},
		Metadata: mapResourceMetadata(app.Metadata),
	}
}

func mapResourceMetadata(m *resource.Metadata) Metadata {
	result := Metadata{Labels: Labels{}}
	if m != nil && m.Labels != nil {
		if v, ok := m.Labels["app-autoscaler.cloudfoundry.org/disable-autoscaling"]; ok {
			result.DisableAutoscaling = v
		}
	}
	return result
}

func mapResourceProcesses(processes []*resource.Process) Processes {
	result := make(Processes, len(processes))
	for i, p := range processes {
		result[i] = mapResourceProcess(p)
	}
	return result
}

func mapResourceProcess(p *resource.Process) Process {
	return Process{
		Guid:       p.GUID,
		Type:       p.Type,
		Instances:  p.Instances,
		MemoryInMb: p.MemoryInMB,
		DiskInMb:   p.DiskInMB,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

func mapResourceServiceInstance(si *resource.ServiceInstance) *ServiceInstance {
	return &ServiceInstance{
		Guid: si.GUID,
		Type: string(si.Type),
		Relationships: ServiceInstanceRelationships{
			ServicePlan: ServicePlanRelation{
				Data: ServicePlanData{
					Guid: si.Relationships.ServicePlan.Data.GUID,
				},
			},
		},
	}
}

func mapResourceServicePlan(sp *resource.ServicePlan) *ServicePlan {
	return &ServicePlan{
		Guid: sp.GUID,
		BrokerCatalog: BrokerCatalog{
			Id: sp.BrokerCatalog.ID,
		},
	}
}

func mapResourceRoles(roles []*resource.Role) Roles {
	result := make(Roles, len(roles))
	for i, r := range roles {
		result[i] = Role{
			Guid: r.GUID,
			Type: RoleType(r.Type),
		}
	}
	return result
}
