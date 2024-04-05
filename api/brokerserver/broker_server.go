package brokerserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"github.com/pivotal-cf/brokerapi/v11"
	"github.com/pivotal-cf/brokerapi/v11/domain"

	"code.cloudfoundry.org/lager/v3"
	"github.com/go-chi/chi/v5"
	"github.com/tedsuo/ifrit"
	"golang.org/x/crypto/bcrypt"
)

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
		brokerCredential = restrictToMaxBcryptLength(logger, brokerCredential)
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

	catalogBytes, err := os.ReadFile(conf.CatalogPath)
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
	autoscalerBroker := broker.New(logger.Session("broker"), conf, bindingdb, policydb, catalog.Services, credentials)

	r := chi.NewRouter()

	r.Use(basicAuthentication.Middleware)
	r.Use(httpStatusCollectMiddleware.Collect)
	brokerapi.AttachRoutes(r, autoscalerBroker, slog.New(lager.NewHandler(logger.Session("broker_handler"))))

	r.HandleFunc(routes.BrokerHealthPath, GetHealth)

	return helpers.NewHTTPServer(logger, conf.BrokerServer, r)
}

func restrictToMaxBcryptLength(logger lager.Logger, brokerCredential config.BrokerCredentialsConfig) config.BrokerCredentialsConfig {
	if len(brokerCredential.BrokerUsername) > 72 {
		logger.Error("warning-configured-username-too-long-using-only-first-72-characters", bcrypt.ErrPasswordTooLong, lager.Data{"username-length": len(brokerCredential.BrokerUsername)})
		brokerCredential.BrokerUsername = brokerCredential.BrokerUsername[:72]
	}
	if len(brokerCredential.BrokerPassword) > 72 {
		logger.Error("warning-configured-password-too-long-using-only-first-72-characters", bcrypt.ErrPasswordTooLong, lager.Data{"password-length": len(brokerCredential.BrokerPassword)})
		brokerCredential.BrokerPassword = brokerCredential.BrokerPassword[:72]
	}

	return brokerCredential
}

func GetHealth(w http.ResponseWriter, _ *http.Request) {
	handlers.WriteJSONResponse(w, http.StatusOK, []byte(`{"alive":"true"}`))
}
