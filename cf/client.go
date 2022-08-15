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
	PathCFInfo                                   = "/v2/info"
	PathCFAuth                                   = "/oauth/token"
	PathIntrospectToken                          = "/introspect"
	GrantTypeClientCredentials                   = "client_credentials"
	GrantTypeRefreshToken                        = "refresh_token"
	TimeToRefreshBeforeTokenExpire time.Duration = 10 * time.Minute
	defaultPerPage                               = 100
)

type Tokens struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type IntrospectionResponse struct {
	Active   bool   `json:"active"`
	Email    string `json:"email"`
	ClientId string `json:"client_id"`
}

type Endpoints struct {
	AuthEndpoint    string `json:"authorization_endpoint"`
	TokenEndpoint   string `json:"token_endpoint"`
	DopplerEndpoint string `json:"doppler_logging_endpoint"`
}

var _ CFClient = &Client{}

type CFClient interface {
	Login() error
	RefreshAuthToken() (string, error)
	GetTokens() (Tokens, error)
	GetEndpoints() Endpoints
	GetApp(string) (*App, error)
	GetAppProcesses(string) (Processes, error)
	GetAppAndProcesses(string) (*AppAndProcesses, error)
	ScaleAppWebProcess(string, int) error
	IsUserAdmin(userToken string) (bool, error)
	IsUserSpaceDeveloper(userToken string, appId string) (bool, error)
	IsTokenAuthorized(token, clientId string) (bool, error)
	GetServicePlan(serviceInstanceGuid string) (string, error)
}

type Client struct {
	logger             lager.Logger
	conf               *Config
	clk                clock.Clock
	tokens             Tokens
	endpoints          Endpoints
	infoURL            string
	tokenURL           string
	introspectTokenURL string
	loginForm          url.Values
	authHeader         string
	httpClient         *http.Client
	lock               *sync.Mutex
	grantTime          time.Time
	retryClient        *http.Client
	servicePlan        *Memoizer[string, string]
	brokerPlanGuid     *Memoizer[string, string]
}

func NewCFClient(conf *Config, logger lager.Logger, clk clock.Clock) *Client {
	c := &Client{}
	c.logger = logger
	c.conf = conf
	c.clk = clk
	c.infoURL = conf.API + PathCFInfo

	c.loginForm = url.Values{
		"grant_type":    {GrantTypeClientCredentials},
		"client_id":     {conf.ClientID},
		"client_secret": {conf.Secret},
	}
	c.authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(conf.ClientID+":"+conf.Secret))
	// #nosec G402 - this is intentionally configurable
	c.httpClient = cfhttp.NewClient(
		cfhttp.WithTLSConfig(&tls.Config{InsecureSkipVerify: conf.SkipSSLValidation}),
		cfhttp.WithDialTimeout(30*time.Second),
	)
	c.httpClient.Transport = DrainingTransport{c.httpClient.Transport}
	c.retryClient = createRetryClient(conf, c.httpClient, logger)
	c.lock = &sync.Mutex{}

	c.servicePlan = NewMemoizer(c.getServicePlan)
	c.brokerPlanGuid = NewMemoizer(c.getBrokerPlanGuid)

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
	retryClient.Logger = LeveledLoggerAdapter{logger}
	retryClient.HTTPClient = client
	retryClient.ErrorHandler = func(resp *http.Response, err error, numTries int) (*http.Response, error) {
		return resp, err
	}
	return retryClient.StandardClient()
}

func (c *Client) retrieveEndpoints() error {
	c.logger.Info("retrieve-endpoints", lager.Data{"infoURL": c.infoURL})

	resp, err := c.httpClient.Get(c.infoURL)
	if err != nil {
		c.logger.Error("retrieve-endpoints-get", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Error requesting endpoints: %s [%d] %s", c.infoURL, resp.StatusCode, resp.Status)
		c.logger.Error("retrieve-endpoints-response", err)
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(&c.endpoints)
	if err != nil {
		c.logger.Error("retrieve-endpoints-decode", err)
		return err
	}

	c.tokenURL = c.endpoints.TokenEndpoint + PathCFAuth
	c.introspectTokenURL = c.endpoints.TokenEndpoint + PathIntrospectToken
	return nil
}

func (c *Client) requestClientCredentialGrant(formData *url.Values) error {
	c.logger.Info("request-client-credential-grant", lager.Data{"tokenURL": c.tokenURL, "form": *formData})

	req, err := http.NewRequest("POST", c.tokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		c.logger.Error("request-client-credential-grant-new-request", err)
		return err
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	var resp *http.Response
	resp, err = c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("request-client-credential-grant-do-request", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request client credential grant failed: %s [%d] %s", c.tokenURL, resp.StatusCode, resp.Status)
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
	c.logger.Info("login", lager.Data{"infoURL": c.infoURL})

	err := c.retrieveEndpoints()
	if err != nil {
		return err
	}

	return c.requestClientCredentialGrant(&c.loginForm)
}

func (c *Client) RefreshAuthToken() (string, error) {
	c.logger.Info("refresh-auth-token", lager.Data{"tokenURL": c.tokenURL})

	var err error
	if c.tokenURL == "" {
		err = c.retrieveEndpoints()
		if err != nil {
			return "", err
		}
	}
	err = c.requestClientCredentialGrant(&c.loginForm)
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

func (c *Client) GetEndpoints() Endpoints {
	return c.endpoints
}

func (c *Client) IsTokenAuthorized(token, clientId string) (bool, error) {
	formData := url.Values{"token": {token}}
	request, err := http.NewRequest("POST", c.introspectTokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return false, err
	}
	request.SetBasicAuth(c.conf.ClientID, c.conf.Secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	resp, err := c.httpClient.Do(request)
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
