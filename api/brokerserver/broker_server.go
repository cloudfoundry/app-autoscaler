package brokerserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"

	"github.com/pivotal-cf/brokerapi/domain"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"golang.org/x/crypto/bcrypt"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

type MiddleWareBrokerCredentials struct {
	BrokerUsername     string
	BrokerUsernameHash []byte
	BrokerPassword     string
	BrokerPasswordHash []byte
}

type basicAuthenticationMiddleware struct {
	brokerCredentials []MiddleWareBrokerCredentials
}

func (bam *basicAuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		username, password, authOK := r.BasicAuth()

		crenditialFoundFlag := false
		for _, brokerCredential := range bam.brokerCredentials {
			usernameHashResult := bcrypt.CompareHashAndPassword(brokerCredential.BrokerUsernameHash, []byte(username))
			passwordHashResult := bcrypt.CompareHashAndPassword(brokerCredential.BrokerPasswordHash, []byte(password))
			if authOK && usernameHashResult == nil && passwordHashResult == nil {
				crenditialFoundFlag = true
				break
			}
		}

		if !crenditialFoundFlag {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewBrokerServer(logger lager.Logger, conf *config.Config, bindingdb db.BindingDB, policydb db.PolicyDB, httpStatusCollector healthendpoint.HTTPStatusCollector, cfClient cf.CFClient, credentials cred_helper.Credentials) (ifrit.Runner, error) {
	var middleWareBrokerCredentials []MiddleWareBrokerCredentials

	for _, brokerCredential := range conf.BrokerCredentials {
		if string(brokerCredential.BrokerUsernameHash) == "" {
			var err error
			brokerCredential.BrokerUsernameHash, err = bcrypt.GenerateFromPassword([]byte(brokerCredential.BrokerUsername), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
			if err != nil {
				logger.Error("failed-new-server-hashing-broker-username", err)
				return nil, err
			}
		}

		if string(brokerCredential.BrokerPasswordHash) == "" {
			var err error
			brokerCredential.BrokerPasswordHash, err = bcrypt.GenerateFromPassword([]byte(brokerCredential.BrokerPassword), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
			if err != nil {
				logger.Error("failed-new-server-hashing-broker-password", err)
				return nil, err
			}
		}

		var middleWareBrokerCredential MiddleWareBrokerCredentials
		middleWareBrokerCredential.BrokerUsername = brokerCredential.BrokerUsername
		middleWareBrokerCredential.BrokerUsernameHash = brokerCredential.BrokerUsernameHash
		middleWareBrokerCredential.BrokerPassword = brokerCredential.BrokerPassword
		middleWareBrokerCredential.BrokerPasswordHash = brokerCredential.BrokerPasswordHash

		middleWareBrokerCredentials = append(middleWareBrokerCredentials, middleWareBrokerCredential)
	}

	catalogBytes, err := ioutil.ReadFile(conf.CatalogPath)
	if err != nil {
		logger.Error("failed to read catalog file", err)
		return nil, err
	}
	catalog := &struct {
		Services []domain.Service `json:"services"`
	}{}
	err = json.Unmarshal(catalogBytes, catalog)
	if err != nil {
		logger.Error("failed to parse catalog", err)
		return nil, err
	}

	basicAuthentication := &basicAuthenticationMiddleware{
		brokerCredentials: middleWareBrokerCredentials,
	}
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	ah := NewBrokerHandler(logger, conf, bindingdb, policydb, catalog.Services, cfClient, credentials)

	r := routes.BrokerRoutes()

	r.Use(basicAuthentication.Middleware)
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.BrokerCatalogRouteName).Handler(VarsFunc(ah.GetBrokerCatalog))
	r.Get(routes.BrokerCreateInstanceRouteName).Handler(VarsFunc(ah.CreateServiceInstance))
	r.Get(routes.BrokerUpdateInstanceRouteName).Handler(VarsFunc(ah.UpdateServiceInstance))
	r.Get(routes.BrokerDeleteInstanceRouteName).Handler(VarsFunc(ah.DeleteServiceInstance))
	r.Get(routes.BrokerCreateBindingRouteName).Handler(VarsFunc(ah.BindServiceInstance))
	r.Get(routes.BrokerDeleteBindingRouteName).Handler(VarsFunc(ah.UnbindServiceInstance))
	r.Get(routes.BrokerHealthRouteName).Handler(VarsFunc(ah.GetHealth))

	var addr string
	if os.Getenv("APP_AUTOSCALER_TEST_RUN") == "true" {
		addr = fmt.Sprintf("localhost:%d", conf.BrokerServer.Port)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", conf.BrokerServer.Port)
	}

	var runner ifrit.Runner
	if (conf.BrokerServer.TLS.KeyFile == "") || (conf.BrokerServer.TLS.CertFile == "") {
		runner = http_server.New(addr, r)
	} else {
		//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
		tlsConfig, err := cfhttp.NewTLSConfig(conf.BrokerServer.TLS.CertFile, conf.BrokerServer.TLS.KeyFile, conf.BrokerServer.TLS.CACertFile)
		if err != nil {
			logger.Error("failed-new-server-new-tls-config", err, lager.Data{"tls": conf.BrokerServer.TLS})
			return nil, err
		}
		runner = http_server.NewTLSServer(addr, r, tlsConfig)
	}

	logger.Info("broker-http-server-created", lager.Data{"serverConfig": conf.BrokerServer})
	return runner, nil
}

func GetDashboardURL(conf *config.Config, instanceId string) string {
	return fmt.Sprintf("%s/manage/%s", conf.DashboardRedirectURI, instanceId)
}
