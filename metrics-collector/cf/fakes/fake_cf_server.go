package fakes

import (
	. "github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/cf"
	"github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/config"
	. "github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/util"

	"bytes"
	"encoding/base64"
	"github.com/gorilla/mux"
	"net/http"
)

const (
	FAKE_ACCESS_TOKEN     = "fake-access-token"
	FAKE_REFRESH_TOKEN    = "fake-refresh-token"
	FAKE_TOKEN_ENDPOINT   = "https://uaa.fake.com"
	FAKE_DOPPLER_ENDPOINT = "wss://doppler.fake.com:4443"
)

var FakeCfConfig = config.CfConfig{
	GrantType: "password",
	User:      "fake-user",
	Pass:      "fake-pass",
	ClientId:  "fake-client",
	Secret:    "fake-secret",
}

var authEndpoint string

func NewFakeApiServerHandler(auth string) http.Handler {
	authEndpoint = auth
	r := mux.NewRouter()
	r.Methods("GET").Path(PATH_CF_INFO).HandlerFunc(handleInfo)
	return r
}

func NewFakeAuthServerHandler() http.Handler {
	r := mux.NewRouter()
	r.Methods("POST").Path(PATH_CF_AUTH).HandlerFunc(handleLogin)
	return r
}

var infoBody = []byte(`
{
   "name": "",
   "build": "",
   "support": "http://support.cloudfoundry.com",
   "version": 0,
   "description": "",
   "authorization_endpoint": "{AUTH_ENDPOINT}",
   "token_endpoint": "{TOKEN_ENDPOINT}",
   "min_cli_version": null,
   "min_recommended_cli_version": null,
   "api_version": "2.48.0",
   "app_ssh_endpoint": "ssh.bosh-lite.com:2222",
   "app_ssh_host_key_fingerprint": "a6:d1:08:0b:b0:cb:9b:5f:c4:ba:44:2a:97:26:19:8a",
   "app_ssh_oauth_client": "ssh-proxy",
   "routing_endpoint": "https://api.bosh-lite.com/routing",
   "logging_endpoint": "wss://loggregator.bosh-lite.com:443",
   "doppler_logging_endpoint": "{DOPPLER_ENDPOINT}",
   "user": "38b2f682-04bf-48af-9e08-0325aa5c4ea9"
}
`)

var authBody = []byte(`
{
	"access_token":"{OAUTH_TOKEN}",
	"token_type":"bearer",
	"refresh_token":"{REFRESH_TOKEN}",
	"expires_in":43199,
	"scope":"openid cloud_controller.read password.write cloud_controller.write",
	"jti":"a735f90f-0b49-447d-8f9d-ae2fbc1491dd"}				
`)

func handleInfo(w http.ResponseWriter, r *http.Request) {

	b := bytes.Replace(infoBody, []byte("{AUTH_ENDPOINT}"), []byte(authEndpoint), -1)
	b = bytes.Replace(b, []byte("{TOKEN_ENDPOINT}"), []byte(FAKE_TOKEN_ENDPOINT), -1)
	b = bytes.Replace(b, []byte("{DOPPLER_ENDPOINT}"), []byte(FAKE_DOPPLER_ENDPOINT), -1)

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	grantType := r.FormValue("grant_type")
	if grantType != "password" && grantType != "client_credentials" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(CreateJsonErrorResponse("Error-login-CF", "invalid grant_type"))
		return
	}

	authHeader := r.Header.Get("Authorization")

	if grantType == "password" {
		if authHeader != "Basic Y2Y6" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			w.Write(CreateJsonErrorResponse("Error-Get-login-CF", "invalid authorization header"))
			return
		}

		user := r.FormValue("username")
		pass := r.FormValue("password")

		if user != FakeCfConfig.User || pass != FakeCfConfig.Pass {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			w.Write(CreateJsonErrorResponse("Error-Get-login-CF", "invalid login credentials"))
			return
		}
	} else {
		token := "Basic " + base64.StdEncoding.EncodeToString([]byte(FakeCfConfig.ClientId+":"+FakeCfConfig.Secret))
		if authHeader != token {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			w.Write(CreateJsonErrorResponse("Error-Get-login-CF", "invalid authorization header"))
			return
		}

		clientId := r.FormValue("client_id")
		secret := r.FormValue("client_secret")

		if clientId != FakeCfConfig.ClientId || secret != FakeCfConfig.Secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			w.Write(CreateJsonErrorResponse("Error-Get-login-CF", "invalid client credentials"))
			return
		}

	}

	b := bytes.Replace(authBody, []byte("{OAUTH_TOKEN}"), []byte(FAKE_ACCESS_TOKEN), -1)
	b = bytes.Replace(b, []byte("{REFRESH_TOKEN}"), []byte(FAKE_REFRESH_TOKEN), -1)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
