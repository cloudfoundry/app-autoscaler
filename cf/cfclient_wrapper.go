package cf

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/config"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

const uaaRequestTimeout = 30 * time.Second
const defaultDialTimeout = 10 * time.Second

type CFClientWrapper struct {
	cfClient    *client.Client
	conf        *Config
	logger      lager.Logger
	httpClient  *http.Client
	endpointsMu sync.RWMutex
	endpoints   *Endpoints
}

type WrapperOption func(*wrapperOptions)

type wrapperOptions struct {
	httpClient *http.Client
}

func WithHTTPClient(client *http.Client) WrapperOption {
	return func(o *wrapperOptions) {
		o.httpClient = client
	}
}

var _ CFClient = &CFClientWrapper{}

func NewCFClientWrapper(conf *Config, logger lager.Logger, opts ...WrapperOption) (*CFClientWrapper, error) {
	wo := &wrapperOptions{}
	for _, opt := range opts {
		opt(wo)
	}

	httpClient := wo.httpClient
	if httpClient == nil {
		httpClient = createConfiguredHTTPClient(conf, logger)
	}

	options := []config.Option{
		config.ClientCredentials(conf.ClientID, conf.Secret),
		config.UserAgent(GetUserAgent()),
		config.HttpClient(httpClient),
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
		cfClient:   cfClient,
		conf:       conf,
		logger:     logger,
		httpClient: httpClient,
	}, nil
}

// createConfiguredHTTPClient creates an HTTP client with retry logic and connection pool settings.
func createConfiguredHTTPClient(conf *Config, logger lager.Logger) *http.Client {
	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: defaultDialTimeout}).DialContext,
		MaxIdleConnsPerHost: conf.MaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(conf.IdleConnectionTimeoutMs) * time.Millisecond,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: conf.SkipSSLValidation}, //nolint:gosec
	}

	baseClient := &http.Client{
		Transport: transport,
		Timeout:   uaaRequestTimeout,
	}

	return RetryClient(conf.ClientConfig, baseClient, logger)
}

func (w *CFClientWrapper) Login(ctx context.Context) error {
	// Verify credentials by making a test API call
	// go-cfclient handles token management internally
	_, err := w.cfClient.Root.Get(ctx)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	return nil
}

func (w *CFClientWrapper) IsUserAdmin(ctx context.Context, userToken string) (bool, error) {
	resp, err := w.introspectToken(ctx, userToken)
	if err != nil {
		return false, err
	}
	if slices.Contains(resp.Scopes, CCAdminScope) {
		w.logger.Info("user is cc admin")
		return true, nil
	}
	return false, nil
}

func (w *CFClientWrapper) IsUserSpaceDeveloper(ctx context.Context, userToken string, appId Guid) (bool, error) {
	userId, err := w.getUserId(ctx, userToken)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			w.logger.Error("getUserId: token not authorized", err)
			return false, nil
		}
		return false, fmt.Errorf("failed IsUserSpaceDeveloper for appId(%s): %w", appId, err)
	}

	spaceId, err := w.getSpaceId(ctx, appId)
	if err != nil {
		return false, fmt.Errorf("failed IsUserSpaceDeveloper for appId(%s): %w", appId, err)
	}

	roles, err := w.GetSpaceDeveloperRoles(ctx, spaceId, userId)
	if err != nil {
		if IsNotFound(err) {
			w.logger.Info("GetSpaceDeveloperRoles: not found", lager.Data{"userId": userId, "spaceId": spaceId})
			return false, nil
		}
		return false, fmt.Errorf("failed IsUserSpaceDeveloper userId(%s), spaceId(%s): %w", userId, spaceId, err)
	}

	if !roles.HasRole(RoleSpaceDeveloper) {
		w.logger.Info("user without SpaceDeveloper role tried to access API")
		return false, nil
	}
	return true, nil
}

func (w *CFClientWrapper) IsTokenAuthorized(ctx context.Context, token, clientId string) (bool, error) {
	resp, err := w.introspectToken(ctx, token)
	if err != nil {
		return false, err
	}
	return resp.Active && resp.ClientId == clientId, nil
}

func (w *CFClientWrapper) getUaaURL(ctx context.Context) (string, error) {
	endpoints, err := w.GetEndpoints(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get UAA endpoint: %w", err)
	}
	return strings.TrimSuffix(endpoints.Uaa.Url, "/"), nil
}

func (w *CFClientWrapper) doUaaRequest(req *http.Request, result any) error {
	req.Header.Set("User-Agent", GetUserAgent())
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}

func (w *CFClientWrapper) introspectToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	uaaURL, err := w.getUaaURL(ctx)
	if err != nil {
		return nil, err
	}

	body := "token=" + url.QueryEscape(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uaaURL+"/introspect", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create introspect request: %w", err)
	}

	credentials := base64.StdEncoding.EncodeToString([]byte(w.conf.ClientID + ":" + w.conf.Secret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+credentials)

	var result IntrospectionResponse
	if err := w.doUaaRequest(req, &result); err != nil {
		return nil, fmt.Errorf("introspect token failed: %w", err)
	}
	return &result, nil
}

func (w *CFClientWrapper) getUserId(ctx context.Context, userToken string) (UserId, error) {
	uaaURL, err := w.getUaaURL(ctx)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uaaURL+"/userinfo", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+userToken)

	var userInfo struct {
		UserID string `json:"user_id"`
	}
	if err := w.doUaaRequest(req, &userInfo); err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	return UserId(userInfo.UserID), nil
}

func (w *CFClientWrapper) getSpaceId(ctx context.Context, appId Guid) (SpaceId, error) {
	app, err := w.GetApp(ctx, appId)
	if err != nil {
		return "", fmt.Errorf("getSpaceId failed: %w", err)
	}
	if app.Relationships.Space == nil || app.Relationships.Space.Data.Guid == "" {
		return "", fmt.Errorf("empty space-guid for app %s", appId)
	}
	return app.Relationships.Space.Data.Guid, nil
}

func (w *CFClientWrapper) GetEndpoints(ctx context.Context) (Endpoints, error) {
	w.endpointsMu.RLock()
	if w.endpoints != nil {
		defer w.endpointsMu.RUnlock()
		return *w.endpoints, nil
	}
	w.endpointsMu.RUnlock()

	w.endpointsMu.Lock()
	defer w.endpointsMu.Unlock()

	if w.endpoints != nil {
		return *w.endpoints, nil
	}

	root, err := w.cfClient.Root.Get(ctx)
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed GetEndpoints: %w", MapCFClientError(err))
	}

	endpoints := mapRootToEndpoints(root)
	w.endpoints = &endpoints

	return endpoints, nil
}

func (w *CFClientWrapper) GetApp(ctx context.Context, appId Guid) (*App, error) {
	app, err := w.cfClient.Applications.Get(ctx, string(appId))
	if err != nil {
		return nil, fmt.Errorf("failed getting app '%s': %w", appId, MapCFClientError(err))
	}
	return mapResourceApp(app), nil
}

func (w *CFClientWrapper) GetAppProcesses(ctx context.Context, appId Guid, processTypes ...string) (Processes, error) {
	opts := &client.ProcessListOptions{}
	if len(processTypes) > 0 {
		opts.Types = client.Filter{Values: processTypes}
	}

	processes, err := w.cfClient.Processes.ListForAppAll(ctx, string(appId), opts)
	if err != nil {
		return nil, fmt.Errorf("failed GetAppProcesses '%s': %w", appId, MapCFClientError(err))
	}

	return mapResourceProcesses(processes), nil
}

func (w *CFClientWrapper) GetAppAndProcesses(ctx context.Context, appId Guid) (*AppAndProcesses, error) {
	var wg sync.WaitGroup
	var app *App
	var processes Processes
	var appErr, procErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		app, appErr = w.GetApp(ctx, appId)
	}()
	go func() {
		defer wg.Done()
		processes, procErr = w.GetAppProcesses(ctx, appId, ProcessTypeWeb)
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

func (w *CFClientWrapper) ScaleAppWebProcess(ctx context.Context, appId Guid, instances int) error {
	processes, err := w.cfClient.Processes.ListForAppAll(ctx, string(appId), &client.ProcessListOptions{
		Types: client.Filter{Values: []string{ProcessTypeWeb}},
	})
	if err != nil {
		return fmt.Errorf("failed to get web process for app '%s': %w", appId, MapCFClientError(err))
	}

	if len(processes) == 0 {
		return fmt.Errorf("no web process found for app '%s'", appId)
	}

	_, err = w.cfClient.Processes.Scale(ctx, processes[0].GUID, &resource.ProcessScale{
		Instances: &instances,
	})
	if err != nil {
		return fmt.Errorf("failed scaling app '%s' to %d: %w", appId, instances, MapCFClientError(err))
	}

	return nil
}

func (w *CFClientWrapper) GetServiceInstance(ctx context.Context, guid string) (*ServiceInstance, error) {
	si, err := w.cfClient.ServiceInstances.Get(ctx, guid)
	if err != nil {
		return nil, fmt.Errorf("failed GetServiceInstance(%s): %w", guid, MapCFClientError(err))
	}
	return mapResourceServiceInstance(si), nil
}

func (w *CFClientWrapper) GetServicePlan(ctx context.Context, guid string) (*ServicePlan, error) {
	sp, err := w.cfClient.ServicePlans.Get(ctx, guid)
	if err != nil {
		return nil, fmt.Errorf("failed GetServicePlan(%s): %w", guid, MapCFClientError(err))
	}
	return mapResourceServicePlan(sp), nil
}

func (w *CFClientWrapper) GetSpaceDeveloperRoles(ctx context.Context, spaceId SpaceId, userId UserId) (Roles, error) {
	opts := &client.RoleListOptions{
		Types:      client.Filter{Values: []string{string(RoleSpaceDeveloper)}},
		SpaceGUIDs: client.Filter{Values: []string{string(spaceId)}},
		UserGUIDs:  client.Filter{Values: []string{string(userId)}},
	}

	roles, err := w.cfClient.Roles.ListAll(ctx, opts)
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
	if m == nil || m.Labels == nil {
		return result
	}
	if v, ok := m.Labels["app-autoscaler.cloudfoundry.org/disable-autoscaling"]; ok {
		result.DisableAutoscaling = v
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
