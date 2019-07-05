package cf

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"autoscaler/models"
)

const (
	PathCFInfo                                   = "/v2/info"
	PathCFAuth                                   = "/oauth/token"
	GrantTypeClientCredentials                   = "client_credentials"
	GrantTypeRefreshToken                        = "refresh_token"
	TimeToRefreshBeforeTokenExpire time.Duration = 10 * time.Minute
)

type Tokens struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type Endpoints struct {
	AuthEndpoint    string `json:"authorization_endpoint"`
	TokenEndpoint   string `json:"token_endpoint"`
	DopplerEndpoint string `json:"doppler_logging_endpoint"`
}

type CFClient interface {
	Login() error
	RefreshAuthToken() (string, error)
	GetTokens() Tokens
	GetEndpoints() Endpoints
	GetApp(string) (*models.AppEntity, error)
	SetAppInstances(string, int) error
	IsUserAdmin(userToken string) (bool, error)
	IsUserSpaceDeveloper(userToken string, appId string) (bool, error)
}

type cfClient struct {
	logger     lager.Logger
	conf       *CFConfig
	clk        clock.Clock
	tokens     Tokens
	endpoints  Endpoints
	infoURL    string
	tokenURL   string
	loginForm  url.Values
	authHeader string
	httpClient *http.Client
	lock       *sync.Mutex
	grantTime  time.Time
}

func NewCFClient(conf *CFConfig, logger lager.Logger, clk clock.Clock) CFClient {
	c := &cfClient{}
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

	c.httpClient = cfhttp.NewClient()
	c.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: conf.SkipSSLValidation}
	c.httpClient.Transport.(*http.Transport).DialContext = (&net.Dialer{
		Timeout: 30 * time.Second,
	}).DialContext

	c.lock = &sync.Mutex{}

	return c
}

func (c *cfClient) retrieveEndpoints() error {
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
	return nil
}

func (c *cfClient) requestClientCredentialGrant(formData *url.Values) error {
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

func (c *cfClient) Login() error {
	c.logger.Info("login", lager.Data{"infoURL": c.infoURL})

	err := c.retrieveEndpoints()
	if err != nil {
		return err
	}

	return c.requestClientCredentialGrant(&c.loginForm)
}

func (c *cfClient) RefreshAuthToken() (string, error) {
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

func (c *cfClient) GetTokens() Tokens {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.isTokenToBeExpired() {
		c.RefreshAuthToken()
	}
	return c.tokens
}

func (c *cfClient) isTokenToBeExpired() bool {
	return c.clk.Now().Sub(c.grantTime) > (time.Duration(c.tokens.ExpiresIn)*time.Second - TimeToRefreshBeforeTokenExpire)
}

func (c *cfClient) GetEndpoints() Endpoints {
	return c.endpoints
}
