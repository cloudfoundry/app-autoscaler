package cf

import (
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
	"code.cloudfoundry.org/lager"
)

const (
	PathCFAuth                                   = "/oauth/token"
	PathIntrospectToken                          = "/introspect"
	GrantTypeClientCredentials                   = "client_credentials"
	GrantTypeRefreshToken                        = "refresh_token"
	TimeToRefreshBeforeTokenExpire time.Duration = 10 * time.Minute
	defaultPerPage                               = 100
)

type (
	Guid   string
	Tokens struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}

	IntrospectionResponse struct {
		Active   bool   `json:"active"`
		Email    string `json:"email"`
		ClientId string `json:"client_id"`
	}

	CFClient interface {
		Login() error
		RefreshAuthToken() (string, error)
		GetTokens() (Tokens, error)
		GetEndpoints() (Endpoints, error)
		GetApp(appId Guid) (*App, error)
		GetAppProcesses(appId Guid, processTypes ...string) (Processes, error)
		GetAppAndProcesses(appId Guid) (*AppAndProcesses, error)
		ScaleAppWebProcess(appId Guid, numberOfProcesses int) error
		IsUserAdmin(userToken string) (bool, error)
		IsUserSpaceDeveloper(userToken string, appId Guid) (bool, error)
		IsTokenAuthorized(token, clientId string) (bool, error)
		GetServiceInstance(serviceInstanceGuid string) (*ServiceInstance, error)
		GetServicePlan(servicePlanGuid string) (*ServicePlan, error)
	}

	Client struct {
		logger      lager.Logger
		conf        *Config
		clk         clock.Clock
		tokens      Tokens
		endpoints   *Lazy[Endpoints]
		loginForm   url.Values
		authHeader  string
		Client      *http.Client
		lock        *sync.Mutex
		grantTime   time.Time
		retryClient *http.Client
	}
)

var _ fmt.Stringer = Guid("some_guid")

func (g Guid) String() string {
	return string(g)
}

var _ CFClient = &Client{}

func NewCFClient(conf *Config, logger lager.Logger, clk clock.Clock) *Client {
	c := &Client{}
	c.logger = logger
	c.conf = conf
	c.clk = clk

	c.loginForm = url.Values{
		"grant_type":    {GrantTypeClientCredentials},
		"client_id":     {conf.ClientID},
		"client_secret": {conf.Secret},
	}
	c.authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(conf.ClientID+":"+conf.Secret))
	// #nosec G402 - this is intentionally configurable
	c.Client = cfhttp.NewClient(
		cfhttp.WithTLSConfig(&tls.Config{InsecureSkipVerify: conf.SkipSSLValidation}),
		cfhttp.WithDialTimeout(10*time.Second),
		cfhttp.WithIdleConnTimeout(time.Duration(conf.IdleTimeoutMs)*time.Millisecond),
		cfhttp.WithMaxIdleConnsPerHost(conf.MaxIdleConnsPerHost),
	)
	c.Client.Transport = DrainingTransport{c.Client.Transport}
	c.retryClient = createRetryClient(conf, c.Client, logger)
	c.lock = &sync.Mutex{}
	c.endpoints = NewLazy(c.getEndpoints)
	if c.conf.PerPage == 0 {
		c.conf.PerPage = defaultPerPage
	}
	return c
}

func createRetryClient(conf *Config, client *http.Client, logger lager.Logger) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 0
	if conf.MaxRetries != 0 {
		retryClient.RetryMax = conf.MaxRetries
	}
	if conf.MaxRetryWaitMs != 0 {
		retryClient.RetryWaitMax = time.Duration(conf.MaxRetryWaitMs) * time.Millisecond
	}
	retryClient.Logger = LeveledLoggerAdapter{logger.Session("retryablehttp")}
	retryClient.HTTPClient = client
	retryClient.ErrorHandler = func(resp *http.Response, err error, numTries int) (*http.Response, error) {
		return resp, err
	}
	return retryClient.StandardClient()
}

func (c *Client) requestClientCredentialGrant(formData *url.Values) error {
	endpoints, err := c.GetEndpoints()
	if err != nil {
		return err
	}
	tokenUrl := endpoints.Uaa.Url + PathCFAuth
	c.logger.Info("request-client-credential-grant", lager.Data{"tokenURL": tokenUrl, "form": *formData})

	req, err := http.NewRequest("POST", tokenUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		c.logger.Error("request-client-credential-grant-new-request", err)
		return err
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	var resp *http.Response
	resp, err = c.Client.Do(req)
	if err != nil {
		c.logger.Error("request-client-credential-grant-do-request", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request client credential grant failed: %s [%d] %s", tokenUrl, resp.StatusCode, resp.Status)
		c.logger.Error("request-client-credential-grant-response", err)
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(&c.tokens)
	if err != nil {
		c.logger.Error("request-client-credential-grant-decode", err)
		return err
	}
	c.grantTime = time.Now()
	return nil
}

func (c *Client) Login() error {
	return c.requestClientCredentialGrant(&c.loginForm)
}

func (c *Client) RefreshAuthToken() (string, error) {
	err := c.requestClientCredentialGrant(&c.loginForm)
	if err != nil {
		return "", err
	}
	return TokenTypeBearer + " " + c.tokens.AccessToken, nil
}

func (c *Client) GetTokens() (Tokens, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.isTokenToBeExpired() {
		_, err := c.RefreshAuthToken()
		if err != nil {
			return c.tokens, err
		}
	}
	return c.tokens, nil
}

func (c *Client) isTokenToBeExpired() bool {
	return c.clk.Now().Sub(c.grantTime) > (time.Duration(c.tokens.ExpiresIn)*time.Second - TimeToRefreshBeforeTokenExpire)
}

func (c *Client) IsTokenAuthorized(token, clientId string) (bool, error) {
	endpoints, err := c.GetEndpoints()
	if err != nil {
		return false, err
	}
	formData := url.Values{"token": {token}}
	tokenURL := endpoints.Uaa.Url + PathIntrospectToken
	request, err := http.NewRequest("POST", tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return false, err
	}
	request.SetBasicAuth(c.conf.ClientID, c.conf.Secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	resp, err := c.Client.Do(request)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("received status code %v while calling /introspect endpoint", resp.Status)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	introspectionResponse := &IntrospectionResponse{}
	err = json.Unmarshal(responseBody, introspectionResponse)
	if err != nil {
		return false, err
	}

	if introspectionResponse.Active && introspectionResponse.ClientId == clientId {
		return true, nil
	}

	return false, nil
}
