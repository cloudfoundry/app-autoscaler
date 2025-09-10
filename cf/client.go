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
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"code.cloudfoundry.org/cfhttp/v2"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

const (
	PathCFAuth                     = "/oauth/token"
	PathIntrospectToken            = "/introspect"
	GrantTypeClientCredentials     = "client_credentials"
	TimeToRefreshBeforeTokenExpire = 10 * time.Minute
	defaultPerPage                 = 100
)

type (
	Guid   string
	Tokens struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}

	TokensInfo struct {
		Tokens
		grantTime time.Time
	}

	IntrospectionResponse struct {
		Active   bool     `json:"active"`
		Email    string   `json:"email"`
		ClientId string   `json:"client_id"`
		Scopes   []string `json:"scope"`
	}

	AuthClient interface {
		Login() error
		InvalidateToken()
		RefreshAuthToken() (Tokens, error)
		GetTokens() (Tokens, error)
		IsUserAdmin(userToken string) (bool, error)
		IsUserSpaceDeveloper(userToken string, appId Guid) (bool, error)
		IsTokenAuthorized(token, clientId string) (bool, error)
	}

	AuthContextClient interface {
		Login(ctx context.Context) error
		InvalidateToken()
		RefreshAuthToken(ctx context.Context) (Tokens, error)
		GetTokens(ctx context.Context) (Tokens, error)
		IsUserAdmin(ctx context.Context, userToken string) (bool, error)
		IsUserSpaceDeveloper(ctx context.Context, userToken string, appId Guid) (bool, error)
		IsTokenAuthorized(ctx context.Context, token, clientId string) (bool, error)
	}

	CFClient interface {
		AuthClient
		ApiClient
		GetCtxClient() ContextClient
	}

	ContextClient interface {
		AuthContextClient
		ApiContextClient
	}

	ApiClient interface {
		GetEndpoints() (Endpoints, error)
		GetApp(appId Guid) (*App, error)
		GetAppProcesses(appId Guid, processTypes ...string) (Processes, error)
		GetAppAndProcesses(appId Guid) (*AppAndProcesses, error)
		ScaleAppWebProcess(appId Guid, numberOfProcesses int) error
		GetServiceInstance(serviceInstanceGuid string) (*ServiceInstance, error)
		GetServicePlan(servicePlanGuid string) (*ServicePlan, error)
	}

	ApiContextClient interface {
		GetEndpoints(ctx context.Context) (Endpoints, error)
		GetApp(ctx context.Context, appId Guid) (*App, error)
		GetAppProcesses(ctx context.Context, appId Guid, processTypes ...string) (Processes, error)
		GetAppAndProcesses(ctx context.Context, appId Guid) (*AppAndProcesses, error)
		ScaleAppWebProcess(ctx context.Context, appId Guid, numberOfProcesses int) error
		GetServiceInstance(ctx context.Context, serviceInstanceGuid string) (*ServiceInstance, error)
		GetServicePlan(ctx context.Context, servicePlanGuid string) (*ServicePlan, error)
	}

	Client struct {
		*CtxClient
	}

	CtxClient struct {
		logger lager.Logger
		conf   *Config
		clk    clock.Clock

		tokenInfoMu sync.RWMutex
		tokenInfo   TokensInfo

		endpoints   *Lazy[Endpoints]
		loginForm   url.Values
		authHeader  string
		Client      *http.Client
		retryClient *http.Client
	}
)

func (c *Client) GetCtxClient() ContextClient {
	return c.CtxClient
}

var _ fmt.Stringer = Guid("some_guid")

func (g Guid) String() string {
	return string(g)
}

var _ CFClient = &Client{}
var _ ContextClient = &CtxClient{}

func NewCFClient(conf *Config, logger lager.Logger, clk clock.Clock) *Client {
	c := &Client{&CtxClient{}}
	c.logger = logger
	c.conf = conf
	c.clk = clk

	c.loginForm = url.Values{
		"grant_type":    {GrantTypeClientCredentials},
		"client_id":     {conf.ClientID},
		"client_secret": {conf.Secret},
	}
	c.authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(conf.ClientID+":"+conf.Secret))
	//nolint:gosec // #nosec G402 -- due to https://github.com/securego/gosec/issues/1105
	c.Client = cfhttp.NewClient(
		cfhttp.WithTLSConfig(&tls.Config{InsecureSkipVerify: conf.SkipSSLValidation}),
		cfhttp.WithDialTimeout(10*time.Second),
		cfhttp.WithIdleConnTimeout(time.Duration(conf.IdleConnectionTimeoutMs)*time.Millisecond),
		cfhttp.WithMaxIdleConnsPerHost(conf.MaxIdleConnsPerHost),
	)
	c.Client.Transport = DrainingTransport{c.Client.Transport}
	c.retryClient = createRetryClient(conf, c.Client, logger)
	c.endpoints = NewLazy(c.CtxClient.GetEndpoints)
	if c.conf.PerPage == 0 {
		c.conf.PerPage = defaultPerPage
	}
	return c
}

func createRetryClient(conf *Config, client *http.Client, logger lager.Logger) *http.Client {
	return RetryClient(conf.ClientConfig, client, logger)
}

func RetryClient(config ClientConfig, client *http.Client, logger lager.Logger) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 0
	if config.MaxRetries != 0 {
		retryClient.RetryMax = config.MaxRetries
	}
	if config.MaxRetryWaitMs != 0 {
		retryClient.RetryWaitMax = time.Duration(config.MaxRetryWaitMs) * time.Millisecond
	}
	retryClient.Logger = LeveledLoggerAdapter{logger.Session("retryablehttp")}
	retryClient.HTTPClient = client
	retryClient.ErrorHandler = func(resp *http.Response, err error, numTries int) (*http.Response, error) {
		return resp, err
	}
	return retryClient.StandardClient()
}

func (c *CtxClient) requestClientCredentialGrant(ctx context.Context, formData *url.Values) (Tokens, error) {
	tokens := Tokens{}
	endpoints, err := c.GetEndpoints(ctx)
	if err != nil {
		return tokens, err
	}

	c.tokenInfoMu.Lock()
	defer c.tokenInfoMu.Unlock()

	if !c.tokenInfo.isTokenExpired(c.clk.Now) {
		return c.tokenInfo.Tokens, nil
	}

	tokens, err = c.doRequestCredGrant(ctx, formData, endpoints.Uaa.Url)
	if err != nil {
		return c.tokenInfo.Tokens, err
	}

	c.tokenInfo.Tokens = tokens
	c.tokenInfo.grantTime = c.clk.Now()

	return tokens, nil
}

func (c *CtxClient) doRequestCredGrant(ctx context.Context, formData *url.Values, credUrl string) (Tokens, error) {
	tokens := Tokens{}
	tokenUrl := credUrl + PathCFAuth
	c.logger.Info("request-client-credential-grant", lager.Data{"tokenURL": tokenUrl, "form": *formData})

	req, err := http.NewRequestWithContext(ctx, "POST", tokenUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		c.logger.Error("request-client-credential-grant-new-request", err)
		return tokens, err
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	c.setUserAgent(req)

	var resp *http.Response
	resp, err = c.Client.Do(req)
	if err != nil {
		c.logger.Error("request-client-credential-grant-do-request", err)
		return tokens, err
	}

	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request client credential grant failed: %s [%d] %s", tokenUrl, resp.StatusCode, resp.Status)
		c.logger.Error("request-client-credential-grant-response", err)
		return tokens, err
	}

	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		c.logger.Error("request-client-credential-grant-decode", err)
		return tokens, err
	}
	return tokens, nil
}

func (c *Client) Login() error {
	return c.CtxClient.Login(context.Background())
}

func (c *CtxClient) Login(ctx context.Context) error {
	_, err := c.requestClientCredentialGrant(ctx, &c.loginForm)
	return err
}

func (c *Client) InvalidateToken() {
	c.CtxClient.InvalidateToken()
}
func (c *CtxClient) InvalidateToken() {
	c.tokenInfoMu.Lock()
	defer c.tokenInfoMu.Unlock()
	c.tokenInfo.grantTime = time.Time{}
}

func (c *Client) RefreshAuthToken() (Tokens, error) {
	return c.CtxClient.RefreshAuthToken(context.Background())
}

func (c *CtxClient) RefreshAuthToken(ctx context.Context) (Tokens, error) {
	return c.requestClientCredentialGrant(ctx, &c.loginForm)
}

func (c *Client) GetTokens() (Tokens, error) {
	return c.CtxClient.GetTokens(context.Background())
}

func (c *CtxClient) GetTokens(ctx context.Context) (Tokens, error) {
	tokenInfo := c.getTokenInfo()
	token := tokenInfo.Tokens
	var err error
	if tokenInfo.isTokenExpired(c.clk.Now) {
		token, err = c.RefreshAuthToken(ctx)
		if err != nil {
			return token, err
		}
	}
	return token, nil
}

func (c *CtxClient) getTokenInfo() TokensInfo {
	c.tokenInfoMu.RLock()
	defer c.tokenInfoMu.RUnlock()
	//Note this is a copy not a pointer
	return c.tokenInfo
}

func (t TokensInfo) isTokenExpired(now func() time.Time) bool {
	return now().Sub(t.grantTime) > (time.Duration(t.ExpiresIn)*time.Second - TimeToRefreshBeforeTokenExpire)
}

func (c *Client) IsTokenAuthorized(token, clientId string) (bool, error) {
	return c.CtxClient.IsTokenAuthorized(context.Background(), token, clientId)
}
func (c *CtxClient) IsTokenAuthorized(ctx context.Context, token, clientId string) (bool, error) {
	introspectionResponse, err := c.introspectToken(ctx, token)
	if err != nil {
		return false, err
	}
	if introspectionResponse.Active && introspectionResponse.ClientId == clientId {
		return true, nil
	}

	return false, nil
}

func (c *CtxClient) introspectToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	endpoints, err := c.GetEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	formData := url.Values{"token": {token}}
	tokenURL := endpoints.Uaa.Url + PathIntrospectToken
	request, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	err = c.addAuth(ctx, request)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	c.setUserAgent(request)

	resp, err := c.Client.Do(request)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %v while calling /introspect endpoint", resp.Status)
	}
	defer func() { _ = resp.Body.Close() }()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	introspectionResponse := &IntrospectionResponse{}
	err = json.Unmarshal(responseBody, introspectionResponse)
	if err != nil {
		return nil, err
	}

	return introspectionResponse, nil
}

func (c *CtxClient) addAuth(ctx context.Context, req *http.Request) error {
	tokens, err := c.GetTokens(ctx)
	if err != nil {
		return fmt.Errorf("get token failed: %w", err)
	}

	req.Header.Set("Authorization", TokenTypeBearer+" "+tokens.AccessToken)
	return nil
}

func (c *CtxClient) setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", GetUserAgent())
}
