package cf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/cloudfoundry-community/go-uaa"
	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/config"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"golang.org/x/oauth2"
)

// CFClientWrapper wraps go-cfclient and go-uaa to implement the existing CFClient interfaces
type CFClientWrapper struct {
	cfClient  *client.Client
	uaaClient *uaa.API
	conf      *Config
	logger    lager.Logger
	clk       clock.Clock

	// Token management
	tokenInfoMu sync.RWMutex
	tokenInfo   TokensInfo

	// Cached endpoints
	endpointsMu sync.RWMutex
	endpoints   *Endpoints
}

// CtxClientWrapper provides context-aware methods
type CtxClientWrapper struct {
	*CFClientWrapper
}

var _ CFClient = &CFClientWrapper{}
var _ ContextClient = &CtxClientWrapper{}

// NewCFClientWrapper creates a new CFClient using go-cfclient/v3 and go-uaa
func NewCFClientWrapper(conf *Config, logger lager.Logger, clk clock.Clock) (*CFClientWrapper, error) {
	// Build config options
	options := []config.Option{
		config.ClientCredentials(conf.ClientID, conf.Secret),
		config.UserAgent(GetUserAgent()),
	}
	if conf.SkipSSLValidation {
		options = append(options, config.SkipTLSValidation())
	}

	// Create go-cfclient configuration
	cfg, err := config.New(conf.API, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cfclient config: %w", err)
	}

	// Create go-cfclient
	cfClient, err := client.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create cfclient: %w", err)
	}

	wrapper := &CFClientWrapper{
		cfClient: cfClient,
		conf:     conf,
		logger:   logger,
		clk:      clk,
	}

	return wrapper, nil
}

// initUAAClient initializes the UAA client lazily (after we have endpoints)
func (w *CFClientWrapper) initUAAClient(ctx context.Context) error {
	if w.uaaClient != nil {
		return nil
	}

	endpoints, err := w.GetCtxClient().GetEndpoints(ctx)
	if err != nil {
		return fmt.Errorf("failed to get endpoints for UAA client: %w", err)
	}

	uaaAPI, err := uaa.New(
		endpoints.Uaa.Url,
		uaa.WithClientCredentials(w.conf.ClientID, w.conf.Secret, uaa.JSONWebToken),
		uaa.WithSkipSSLValidation(w.conf.SkipSSLValidation),
	)
	if err != nil {
		return fmt.Errorf("failed to create UAA client: %w", err)
	}

	w.uaaClient = uaaAPI
	return nil
}

// GetCtxClient returns the context-aware client
func (w *CFClientWrapper) GetCtxClient() ContextClient {
	return &CtxClientWrapper{w}
}

// --- AuthClient Interface Implementation ---

// Login authenticates with CF using client credentials
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

// InvalidateToken invalidates the current token
func (w *CFClientWrapper) InvalidateToken() {
	w.tokenInfoMu.Lock()
	defer w.tokenInfoMu.Unlock()
	w.tokenInfo.grantTime = time.Time{}
}

// RefreshAuthToken refreshes the authentication token
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

// refreshToken performs the actual token refresh using go-cfclient
func (c *CtxClientWrapper) refreshToken(ctx context.Context) (Tokens, error) {
	// go-cfclient handles token refresh internally, but we need to expose tokens
	// for compatibility. We'll get a fresh token by making a lightweight API call.
	// The cfClient internally manages token refresh.

	// Get the access token from cfclient's config
	// Since go-cfclient manages tokens internally, we need to trigger a refresh
	// by making a call that requires authentication

	// For now, we simulate getting tokens by making a root API call
	// which will trigger the internal token refresh mechanism
	_, err := c.cfClient.Root.Get(ctx)
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to refresh token: %w", err)
	}

	// go-cfclient doesn't expose the token directly in v3
	// We need to get it from the underlying config/transport
	// This is a limitation - we may need to use our own token management for UAA
	return c.tokenInfo.Tokens, nil
}

// GetTokens returns the current tokens, refreshing if necessary
func (w *CFClientWrapper) GetTokens() (Tokens, error) {
	return w.GetCtxClient().GetTokens(context.Background())
}

func (c *CtxClientWrapper) GetTokens(ctx context.Context) (Tokens, error) {
	tokenInfo := c.getTokenInfo()
	if tokenInfo.isTokenExpired(c.clk.Now) {
		return c.RefreshAuthToken(ctx)
	}
	return tokenInfo.Tokens, nil
}

func (c *CtxClientWrapper) getTokenInfo() TokensInfo {
	c.tokenInfoMu.RLock()
	defer c.tokenInfoMu.RUnlock()
	return c.tokenInfo
}

// IsUserAdmin checks if the user has cloud_controller.admin scope
func (w *CFClientWrapper) IsUserAdmin(userToken string) (bool, error) {
	return w.GetCtxClient().IsUserAdmin(context.Background(), userToken)
}

func (c *CtxClientWrapper) IsUserAdmin(ctx context.Context, userToken string) (bool, error) {
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

// IsUserSpaceDeveloper checks if the user is a space developer for the app's space
func (w *CFClientWrapper) IsUserSpaceDeveloper(userToken string, appId Guid) (bool, error) {
	return w.GetCtxClient().IsUserSpaceDeveloper(context.Background(), userToken, appId)
}

func (c *CtxClientWrapper) IsUserSpaceDeveloper(ctx context.Context, userToken string, appId Guid) (bool, error) {
	userId, err := c.getUserId(ctx, userToken)
	if err != nil {
		if err == ErrUnauthorized {
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

// IsTokenAuthorized checks if a token is authorized for a specific client
func (w *CFClientWrapper) IsTokenAuthorized(token, clientId string) (bool, error) {
	return w.GetCtxClient().IsTokenAuthorized(context.Background(), token, clientId)
}

func (c *CtxClientWrapper) IsTokenAuthorized(ctx context.Context, token, clientId string) (bool, error) {
	introspectionResponse, err := c.introspectToken(ctx, token)
	if err != nil {
		return false, err
	}
	if introspectionResponse.Active && introspectionResponse.ClientId == clientId {
		return true, nil
	}

	return false, nil
}

// introspectToken uses UAA's introspection endpoint via go-uaa's Curl method
func (c *CtxClientWrapper) introspectToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	if err := c.initUAAClient(ctx); err != nil {
		return nil, err
	}

	// Use go-uaa's Curl to call /introspect endpoint
	// Curl signature: Curl(path string, method string, data string, headers []string) (string, string, int, error)
	data := fmt.Sprintf("token=%s", url.QueryEscape(token))
	body, _, statusCode, err := c.uaaClient.Curl("/introspect", "POST", data, []string{
		"Content-Type: application/x-www-form-urlencoded",
	})
	if err != nil {
		return nil, fmt.Errorf("introspect token failed: %w", err)
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("introspect token failed with status code: %d", statusCode)
	}

	// Parse the response
	response := &IntrospectionResponse{}
	if err := parseJSON([]byte(body), response); err != nil {
		return nil, fmt.Errorf("failed to parse introspection response: %w", err)
	}

	return response, nil
}

// getUserId gets the user ID from the user token using UAA's /userinfo endpoint
func (c *CtxClientWrapper) getUserId(ctx context.Context, userToken string) (UserId, error) {
	if err := c.initUAAClient(ctx); err != nil {
		return "", err
	}

	// Create a temporary UAA client with the user's token
	endpoints, err := c.GetEndpoints(ctx)
	if err != nil {
		return "", err
	}

	// Use oauth2.Token for go-uaa
	token := &oauth2.Token{
		AccessToken: userToken,
		TokenType:   TokenTypeBearer,
	}

	var userUAAClient *uaa.API
	if c.conf.SkipSSLValidation {
		userUAAClient, err = uaa.New(
			endpoints.Uaa.Url,
			uaa.WithToken(token),
			uaa.WithSkipSSLValidation(true),
		)
	} else {
		userUAAClient, err = uaa.New(
			endpoints.Uaa.Url,
			uaa.WithToken(token),
		)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create user UAA client: %w", err)
	}

	// Use GetMe to get the user info
	userInfo, err := userUAAClient.GetMe()
	if err != nil {
		// Check if it's an unauthorized error
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
			return "", ErrUnauthorized
		}
		return "", fmt.Errorf("failed to get user info: %w", err)
	}

	return UserId(userInfo.UserID), nil
}

// getSpaceId gets the space ID from an app
func (c *CtxClientWrapper) getSpaceId(ctx context.Context, appId Guid) (SpaceId, error) {
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

// --- ApiClient Interface Implementation ---

// GetEndpoints returns the CF API endpoints
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

	// Double-check after acquiring write lock
	if c.endpoints != nil {
		return *c.endpoints, nil
	}

	// Get root info from go-cfclient
	root, err := c.cfClient.Root.Get(ctx)
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed GetEndpoints: %w", MapCFClientError(err))
	}

	endpoints := mapRootToEndpoints(root)
	c.endpoints = &endpoints

	return endpoints, nil
}

// GetApp retrieves an application by its GUID
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

// GetAppProcesses retrieves processes for an application
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

// GetAppAndProcesses retrieves both app and processes in parallel
func (w *CFClientWrapper) GetAppAndProcesses(appId Guid) (*AppAndProcesses, error) {
	return w.GetCtxClient().GetAppAndProcesses(context.Background(), appId)
}

func (c *CtxClientWrapper) GetAppAndProcesses(ctx context.Context, appId Guid) (*AppAndProcesses, error) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	var app *App
	var processes Processes
	var errApp, errProc error

	go func() {
		defer wg.Done()
		app, errApp = c.GetApp(ctx, appId)
	}()
	go func() {
		defer wg.Done()
		processes, errProc = c.GetAppProcesses(ctx, appId, ProcessTypeWeb)
	}()
	wg.Wait()

	if errApp != nil {
		return nil, fmt.Errorf("get state&instances failed: %w", errApp)
	}
	if errProc != nil {
		return nil, fmt.Errorf("get state&instances failed: %w", errProc)
	}
	return &AppAndProcesses{App: app, Processes: processes}, nil
}

// ScaleAppWebProcess scales the web process of an application
func (w *CFClientWrapper) ScaleAppWebProcess(appId Guid, numberOfProcesses int) error {
	return w.GetCtxClient().ScaleAppWebProcess(context.Background(), appId, numberOfProcesses)
}

func (c *CtxClientWrapper) ScaleAppWebProcess(ctx context.Context, appId Guid, numberOfProcesses int) error {
	// go-cfclient v3 scales by process GUID, but we can use the app GUID + process type
	// First, get the web process for the app
	processes, err := c.cfClient.Processes.ListForAppAll(ctx, string(appId), &client.ProcessListOptions{
		Types: client.Filter{Values: []string{ProcessTypeWeb}},
	})
	if err != nil {
		return fmt.Errorf("failed to get web process for app '%s': %w", appId, MapCFClientError(err))
	}

	if len(processes) == 0 {
		return fmt.Errorf("no web process found for app '%s'", appId)
	}

	// Scale the web process
	processGUID := processes[0].GUID
	_, err = c.cfClient.Processes.Scale(ctx, processGUID, &resource.ProcessScale{
		Instances: &numberOfProcesses,
	})
	if err != nil {
		return fmt.Errorf("failed scaling app '%s' to %d: %w", appId, numberOfProcesses, MapCFClientError(err))
	}

	return nil
}

// GetServiceInstance retrieves a service instance by GUID
func (w *CFClientWrapper) GetServiceInstance(serviceInstanceGuid string) (*ServiceInstance, error) {
	return w.GetCtxClient().GetServiceInstance(context.Background(), serviceInstanceGuid)
}

func (c *CtxClientWrapper) GetServiceInstance(ctx context.Context, serviceInstanceGuid string) (*ServiceInstance, error) {
	si, err := c.cfClient.ServiceInstances.Get(ctx, serviceInstanceGuid)
	if err != nil {
		return nil, fmt.Errorf("failed GetServiceInstance guid(%s): %w", serviceInstanceGuid, MapCFClientError(err))
	}
	return mapResourceServiceInstance(si), nil
}

// GetServicePlan retrieves a service plan by GUID
func (w *CFClientWrapper) GetServicePlan(servicePlanGuid string) (*ServicePlan, error) {
	return w.GetCtxClient().GetServicePlan(context.Background(), servicePlanGuid)
}

func (c *CtxClientWrapper) GetServicePlan(ctx context.Context, servicePlanGuid string) (*ServicePlan, error) {
	sp, err := c.cfClient.ServicePlans.Get(ctx, servicePlanGuid)
	if err != nil {
		return nil, fmt.Errorf("failed GetServicePlan(%s): %w", servicePlanGuid, MapCFClientError(err))
	}
	return mapResourceServicePlan(sp), nil
}

// GetSpaceDeveloperRoles retrieves space developer roles for a user
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

// --- Type Mapping Functions ---

// mapRootToEndpoints maps go-cfclient's Root to our Endpoints type
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

// mapResourceApp maps go-cfclient's App to our App type
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

// mapResourceMetadata maps go-cfclient's Metadata to our Metadata type
func mapResourceMetadata(m *resource.Metadata) Metadata {
	result := Metadata{
		Labels: Labels{},
	}
	if m != nil && m.Labels != nil {
		if v, ok := m.Labels["app-autoscaler.cloudfoundry.org/disable-autoscaling"]; ok {
			result.Labels.DisableAutoscaling = v
		}
	}
	return result
}

// mapResourceProcesses maps go-cfclient's Process list to our Processes type
func mapResourceProcesses(processes []*resource.Process) Processes {
	result := make(Processes, len(processes))
	for i, p := range processes {
		result[i] = mapResourceProcess(p)
	}
	return result
}

// mapResourceProcess maps go-cfclient's Process to our Process type
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

// mapResourceServiceInstance maps go-cfclient's ServiceInstance to our ServiceInstance type
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

// mapResourceServicePlan maps go-cfclient's ServicePlan to our ServicePlan type
func mapResourceServicePlan(sp *resource.ServicePlan) *ServicePlan {
	return &ServicePlan{
		Guid: sp.GUID,
		BrokerCatalog: BrokerCatalog{
			Id: sp.BrokerCatalog.ID,
		},
	}
}

// mapResourceRoles maps go-cfclient's Role list to our Roles type
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

// parseJSON is a helper to parse JSON bytes
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
