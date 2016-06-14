package cf

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"metrics-collector/config"
	"net/http"
	"net/url"
	"strings"
)

const (
	PATH_CF_INFO                  = "/v2/info"
	PATH_CF_AUTH                  = "/oauth/token"
	GRANT_TYPE_PASSWORD           = "password"
	GRANT_TYPE_CLIENT_CREDENTIALS = "client_credentials"
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
	tokens    Tokens
	endpoints Endpoints
	config    *config.CfConfig
}

func NewCfClient(conf *config.CfConfig) CfClient {
	return &cfClient{
		config: conf,
	}
}

func (c *cfClient) retrieveEndpoints() error {
	url := c.config.Api + PATH_CF_INFO
	resp, err := DoRequest("GET", url, "", nil, nil)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error retrieving cf endpoints: %s [%d] %s", url, resp.StatusCode, resp.Status)
	}

	d := json.NewDecoder(resp.Body)
	return d.Decode(&c.endpoints)
}

//
// Get Access/Refresh Tokens from login server
//

func (c *cfClient) Login() error {
	err := c.retrieveEndpoints()
	if err != nil {
		return err
	}

	authURL := c.endpoints.AuthEndpoint + PATH_CF_AUTH
	grantType := strings.ToLower(c.config.GrantType)

	var form url.Values
	if grantType == GRANT_TYPE_PASSWORD {
		form = url.Values{
			"grant_type": {GRANT_TYPE_PASSWORD},
			"username":   {c.config.User},
			"password":   {c.config.Pass},
		}
	} else if grantType == GRANT_TYPE_CLIENT_CREDENTIALS {
		form = url.Values{
			"grant_type":    {GRANT_TYPE_CLIENT_CREDENTIALS},
			"client_id":     {c.config.ClientId},
			"client_secret": {c.config.Secret},
		}
	} else {
		return fmt.Errorf("Not supported grant type: %s", grantType)
	}

	headers := map[string]string{}
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["charset"] = "utf-8"

	var token string
	if grantType == GRANT_TYPE_PASSWORD {
		token = "Basic Y2Y6"
	} else {
		token = c.config.ClientId + ":" + c.config.Secret
		token = "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
	}

	var resp *http.Response
	resp, err = DoRequest("POST", authURL, token, headers, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Login failed: %s [%d] %s", authURL, resp.StatusCode, resp.Status)
	}

	d := json.NewDecoder(resp.Body)
	return d.Decode(&c.endpoints)
}

func (c *cfClient) GetTokens() Tokens {
	return c.tokens
}

func (c *cfClient) GetEndpoints() Endpoints {
	return c.endpoints
}
