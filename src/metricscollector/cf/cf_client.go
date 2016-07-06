package cf

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"metricscollector/config"
	"net/http"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
)

const (
	PathCfInfo = "/v2/info"
	PathCfAuth = "/oauth/token"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Endpoints struct {
	AuthEndpoint    string `json:"authorization_endpoint"`
	TokenEndpoint   string `json:"token_endpoint"`
	DopplerEndpoint string `json:"doppler_logging_endpoint"`
}

type CfClient interface {
	Login() error
	GetTokens() Tokens
	GetEndpoints() Endpoints
}

type cfClient struct {
	logger     lager.Logger
	tokens     Tokens
	endpoints  Endpoints
	infoUrl    string
	form       url.Values
	token      string
	headers    map[string]string
	httpClient *http.Client
}

func NewCfClient(conf *config.CfConfig, logger lager.Logger) CfClient {
	c := &cfClient{}

	c.logger = logger
	c.infoUrl = conf.Api + PathCfInfo

	if conf.GrantType == config.GrantTypePassword {
		c.form = url.Values{
			"grant_type": {config.GrantTypePassword},
			"username":   {conf.Username},
			"password":   {conf.Password},
		}
		c.token = "Basic Y2Y6"
	} else {
		c.form = url.Values{
			"grant_type":    {config.GrantTypeClientCredentials},
			"client_id":     {conf.ClientId},
			"client_secret": {conf.Secret},
		}
		c.token = "Basic " + base64.StdEncoding.EncodeToString([]byte(conf.ClientId+":"+conf.Secret))
	}

	c.headers = map[string]string{}
	c.headers["Content-Type"] = "application/x-www-form-urlencoded"
	c.headers["charset"] = "utf-8"

	c.httpClient = cfhttp.NewClient()
	c.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return c
}

func (c *cfClient) retrieveEndpoints() error {
	c.logger.Info("retrieve-endpoints", lager.Data{"infoUrl": c.infoUrl})

	resp, err := c.httpClient.Get(c.infoUrl)
	if err != nil {
		c.logger.Error("request-endpoints", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Error requesting endpoints: %s [%d] %s", c.infoUrl, resp.StatusCode, resp.Status)
		c.logger.Error("request-endpoints", err)
		return err
	}

	d := json.NewDecoder(resp.Body)
	err = d.Decode(&c.endpoints)
	if err != nil {
		c.logger.Error("decode-json-endpoints", err)
	}
	return err
}

func (c *cfClient) Login() error {
	err := c.retrieveEndpoints()
	if err != nil {
		return err
	}

	authURL := c.endpoints.AuthEndpoint + PathCfAuth
	c.logger.Info("login", lager.Data{"authURL": authURL, "form": c.form})

	var req *http.Request
	req, err = http.NewRequest("POST", authURL, strings.NewReader(c.form.Encode()))
	if err != nil {
		c.logger.Error("requst-login", err)
		return err
	}
	req.Header.Set("Authorization", c.token)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	var resp *http.Response
	resp, err = c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("request-login", err)
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Login failed: %s [%d] %s", authURL, resp.StatusCode, resp.Status)
		c.logger.Error("request-login", err)
		return err
	}

	d := json.NewDecoder(resp.Body)
	err = d.Decode(&c.tokens)
	if err != nil {
		c.logger.Error("decode-json-tokens", err)
	}
	return err

}

func (c *cfClient) GetTokens() Tokens {
	return c.tokens
}

func (c *cfClient) GetEndpoints() Endpoints {
	return c.endpoints
}
