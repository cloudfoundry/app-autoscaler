package cf

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pivotal-golang/lager"
	"metrics-collector/config"
	"metrics-collector/mhttp"
	"net/http"
	"net/url"
	"strings"
)

const (
	PATH_CF_INFO = "/v2/info"
	PATH_CF_AUTH = "/oauth/token"
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
	logger      lager.Logger
	tokens      Tokens
	endpoints   Endpoints
	infoUrl     string
	formEncoded string
	token       string
	headers     map[string]string
}

func NewCfClient(conf *config.CfConfig, logger lager.Logger) CfClient {
	c := &cfClient{}
	c.logger = logger
	c.infoUrl = conf.Api + PATH_CF_INFO

	if conf.GrantType == config.GRANT_TYPE_PASSWORD {
		c.formEncoded = url.Values{
			"grant_type": {config.GRANT_TYPE_PASSWORD},
			"username":   {conf.User},
			"password":   {conf.Pass},
		}.Encode()
		c.token = "Basic Y2Y6"
	} else {
		c.formEncoded = url.Values{
			"grant_type":    {config.GRANT_TYPE_CLIENT_CREDENTIALS},
			"client_id":     {conf.ClientId},
			"client_secret": {conf.Secret},
		}.Encode()
		c.token = "Basic " + base64.StdEncoding.EncodeToString([]byte(conf.ClientId+":"+conf.Secret))
	}
	c.headers = map[string]string{}
	c.headers["Content-Type"] = "application/x-www-form-urlencoded"
	c.headers["charset"] = "utf-8"
	return c
}

func (c *cfClient) retrieveEndpoints() error {
	c.logger.Info("retrieve-endpoints", lager.Data{"infoUrl": c.infoUrl, "formEncoded": c.formEncoded})

	resp, err := mhttp.DoRequest("GET", c.infoUrl, "", nil, nil)
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

	authURL := c.endpoints.AuthEndpoint + PATH_CF_AUTH
	c.logger.Info("login", lager.Data{"authURL": authURL})

	var resp *http.Response
	resp, err = mhttp.DoRequest("POST", authURL, c.token, c.headers, strings.NewReader(c.formEncoded))
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
