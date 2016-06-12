package cf

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/config"
	. "github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/util"
	"io/ioutil"
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
	tokens    *Tokens
	endpoints *Endpoints
	config    *config.CfConfig
}

func NewCfClient(conf config.CfConfig) CfClient {
	return &cfClient{
		config:    &conf,
		tokens:    &Tokens{},
		endpoints: &Endpoints{},
	}
}

func (c *cfClient) retrieveEndpoints() error {

	url := c.config.Api + PATH_CF_INFO
	resp, err := DoRequest("GET", url, "", nil, nil)

	if err == nil {
		defer resp.Body.Close()

		if (resp.StatusCode >= 200) && (resp.StatusCode < 300) {
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(body, c.endpoints)
			}
		} else {
			err = errors.New("Error retrive cf endpoints from " + url + " : " + resp.Status)
		}
	}
	return err
}

//
// Get Access/Refresh Tokens from login server
//

func (c *cfClient) Login() error {

	err := c.retrieveEndpoints()
	if err != nil {
		return err
	}

	authUrl := c.endpoints.AuthEndpoint + PATH_CF_AUTH
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
		return errors.New("Not supported grant type :" + grantType)
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
	resp, err = DoRequest("POST", authUrl, token, headers, strings.NewReader(form.Encode()))

	if err == nil {
		defer resp.Body.Close()

		if (resp.StatusCode >= 200) && (resp.StatusCode < 300) {
			var body []byte
			body, err = ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(body, c.tokens)
			}
		} else {
			err = errors.New("Error login cf with " + authUrl + " : " + resp.Status)
		}
	}
	return err

}

func (c *cfClient) GetTokens() Tokens {
	return *c.tokens
}

func (c *cfClient) GetEndpoints() Endpoints {
	return *c.endpoints
}
