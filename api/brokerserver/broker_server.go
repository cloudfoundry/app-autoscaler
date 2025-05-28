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
	"code.cloudfoundry.org/brokerapi/v13"
	"code.cloudfoundry.org/brokerapi/v13/domain"
	"code.cloudfoundry.org/lager/v3"
	"github.com/go-chi/chi/v5"
	"github.com/tedsuo/ifrit"
	"golang.org/x/crypto/bcrypt"
)

type BrokerCredentials struct {
	Username     string
	UsernameHash []byte
	Password     string
	PasswordHash []byte
}

type AuthMiddleware struct {
	credentials []BrokerCredentials
}

func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		if !am.authenticate(r) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (am *AuthMiddleware) authenticate(r *http.Request) bool {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}
	for _, cred := range am.credentials {
		if bcrypt.CompareHashAndPassword(cred.UsernameHash, []byte(username)) == nil &&
			bcrypt.CompareHashAndPassword(cred.PasswordHash, []byte(password)) == nil {
			return true
		}
	}
	return false
}

type BrokerServer interface {
	CreateServer() (ifrit.Runner, error)
	GetRouter() (*chi.Mux, error)
}

type brokerServer struct {
	logger              lager.Logger
	conf                *config.Config
	bindingDB           db.BindingDB
	policyDB            db.PolicyDB
	httpStatusCollector healthendpoint.HTTPStatusCollector
	cfClient            cf.CFClient
	credentials         cred_helper.Credentials
}

func NewBrokerServer(logger lager.Logger, conf *config.Config, bindingDB db.BindingDB, policyDB db.PolicyDB, httpStatusCollector healthendpoint.HTTPStatusCollector, cfClient cf.CFClient, credentials cred_helper.Credentials) *brokerServer {
	return &brokerServer{
		logger:              logger,
		conf:                conf,
		bindingDB:           bindingDB,
		policyDB:            policyDB,
		httpStatusCollector: httpStatusCollector,
		cfClient:            cfClient,
		credentials:         credentials,
	}
}

func (s *brokerServer) CreateServer() (ifrit.Runner, error) {
	router, err := s.GetRouter()
	if err != nil {
		return nil, err
	}

	return helpers.NewHTTPServer(s.logger, s.conf.BrokerServer, router)
}

func prepareCredentials(logger lager.Logger, conf *config.Config) ([]BrokerCredentials, error) {
	var credentialsList []BrokerCredentials
	for _, cred := range conf.BrokerCredentials {
		cred = restrictToMaxBcryptLength(logger, cred)
		usernameHash, err := hashPassword(cred.BrokerUsername)
		if err != nil {
			return nil, err
		}
		passwordHash, err := hashPassword(cred.BrokerPassword)
		if err != nil {
			return nil, err
		}
		credentialsList = append(credentialsList, BrokerCredentials{
			Username:     cred.BrokerUsername,
			UsernameHash: usernameHash,
			Password:     cred.BrokerPassword,
			PasswordHash: passwordHash,
		})
	}
	return credentialsList, nil
}

func hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
}

func loadCatalog(path string, logger lager.Logger) ([]domain.Service, error) {
	catalogBytes, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read catalog file", err)
		return nil, err
	}
	var catalog struct {
		Services []domain.Service `json:"services"`
	}
	if err := json.Unmarshal(catalogBytes, &catalog); err != nil {
		logger.Error("failed to parse catalog", err)
		return nil, err
	}
	return catalog.Services, nil
}

func (s *brokerServer) GetRouter() (*chi.Mux, error) {
	credentialsList, err := prepareCredentials(s.logger, s.conf)
	if err != nil {
		return nil, err
	}

	catalog, err := loadCatalog(s.conf.CatalogPath, s.logger)
	if err != nil {
		return nil, err
	}

	authMiddleware := &AuthMiddleware{credentials: credentialsList}
	httpStatusMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector)

	autoscalerBroker := broker.New(s.logger.Session("broker"), s.conf, s.bindingDB, s.policyDB, catalog, s.credentials)

	router := chi.NewRouter()

	brokerAPI := brokerapi.NewWithOptions(autoscalerBroker, slog.New(lager.NewHandler(s.logger.Session("broker_handler"))), brokerapi.WithCustomAuth(authMiddleware.Middleware), brokerapi.WithAdditionalMiddleware(httpStatusMiddleware.Collect))
	router.Handle(("/*"), brokerAPI)

	router.HandleFunc(routes.HealthPath, GetHealth)

	return router, nil
}

func restrictToMaxBcryptLength(logger lager.Logger, cred config.BrokerCredentialsConfig) config.BrokerCredentialsConfig {
	if len(cred.BrokerUsername) > 72 {
		logger.Error("warning-configured-username-too-long-using-only-first-72-characters", bcrypt.ErrPasswordTooLong, lager.Data{"username-length": len(cred.BrokerUsername)})
		cred.BrokerUsername = cred.BrokerUsername[:72]
	}
	if len(cred.BrokerPassword) > 72 {
		logger.Error("warning-configured-password-too-long-using-only-first-72-characters", bcrypt.ErrPasswordTooLong, lager.Data{"password-length": len(cred.BrokerPassword)})
		cred.BrokerPassword = cred.BrokerPassword[:72]
	}
	return cred
}

func GetHealth(w http.ResponseWriter, _ *http.Request) {
	handlers.WriteJSONResponse(w, http.StatusOK, []byte(`{"alive":"true"}`))
}
