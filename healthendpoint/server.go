package healthendpoint

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/common"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"golang.org/x/crypto/bcrypt"
)

// basic authentication credentials struct
type basicAuthenticationMiddleware struct {
	usernameHash []byte
	passwordHash []byte
}

// middleware basic authentication middleware functionality for healthcheck
func (bam *basicAuthenticationMiddleware) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, authOK := r.BasicAuth()

		if !authOK || bcrypt.CompareHashAndPassword(bam.usernameHash, []byte(username)) != nil || bcrypt.CompareHashAndPassword(bam.passwordHash, []byte(password)) != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NewServerWithBasicAuth open the healthcheck port with basic authentication.
// Make sure that username and password is not empty
func NewServerWithBasicAuth(conf models.HealthConfig, healthCheckers []Checker, logger lager.Logger, gatherer prometheus.Gatherer, time func() time.Time) (ifrit.Runner, error) {
	healthRouter, err := NewHealthRouter(conf, healthCheckers, logger, gatherer, time)
	if err != nil {
		return nil, err
	}
	var addr string
	if os.Getenv("APP_AUTOSCALER_TEST_RUN") == "true" {
		addr = fmt.Sprintf("localhost:%d", conf.Port)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", conf.Port)
	}

	logger.Info("new-health-server-basic-auth", lager.Data{"addr": addr})
	return http_server.New(addr, healthRouter), nil
}

func NewHealthRouter(conf models.HealthConfig, healthCheckers []Checker, logger lager.Logger, gatherer prometheus.Gatherer, time func() time.Time) (*mux.Router, error) {
	var healthRouter *mux.Router
	var err error
	username := conf.HealthCheckUsername
	password := conf.HealthCheckPassword
	usernameHash := conf.HealthCheckUsernameHash
	passwordHash := conf.HealthCheckPasswordHash
	if username == "" && password == "" && usernameHash == "" && passwordHash == "" {
		//when username and password are not set then don't use basic authentication
		healthRouter = mux.NewRouter()
		if conf.ReadinessCheckEnabled {
			healthRouter.Handle("/health/readiness", common.VarsFunc(readiness(healthCheckers, time)))
		}
		healthRouter.PathPrefix("").Handler(promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	} else {
		healthRouter, err = healthBasicAuthRouter(conf, healthCheckers, logger, gatherer, time)
		if err != nil {
			return nil, err
		}
	}
	return healthRouter, nil
}

func healthBasicAuthRouter(conf models.HealthConfig, healthCheckers []Checker, logger lager.Logger, gatherer prometheus.Gatherer, time func() time.Time) (*mux.Router, error) {
	basicAuthentication, err := createBasicAuthMiddleware(logger, conf.HealthCheckUsernameHash, conf.HealthCheckUsername, conf.HealthCheckPasswordHash, conf.HealthCheckPassword)
	if err != nil {
		return nil, err
	}
	promHandler := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})

	// /health
	router := mux.NewRouter()
	// unauthenticated paths
	if conf.ReadinessCheckEnabled {
		router.Handle("/health/readiness", common.VarsFunc(readiness(healthCheckers, time)))
	}
	//authenticated paths
	health := router.Path("/health").Subrouter()
	health.Use(basicAuthentication.middleware)

	everything := router.PathPrefix("").Subrouter()
	everything.Use(basicAuthentication.middleware)
	everything.PathPrefix("").Handler(promHandler)

	return router, nil
}

func createBasicAuthMiddleware(logger lager.Logger, usernameHash string, username string, passwordHash string, password string) (*basicAuthenticationMiddleware, error) {
	usernameHashByte, err := getUserHashBytes(logger, usernameHash, username)
	if err != nil {
		return nil, err
	}

	passwordHashByte, err := getPasswordHashBytes(logger, passwordHash, password)
	if err != nil {
		return nil, err
	}

	basicAuthentication := &basicAuthenticationMiddleware{
		usernameHash: usernameHashByte,
		passwordHash: passwordHashByte,
	}
	return basicAuthentication, nil
}

func getPasswordHashBytes(logger lager.Logger, passwordHash string, password string) ([]byte, error) {
	var passwordHashByte []byte
	var err error
	if passwordHash == "" {
		passwordHashByte, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
		if err != nil {
			logger.Error("failed-new-server-password", err)
			return nil, err
		}
	} else {
		passwordHashByte = []byte(passwordHash)
	}
	return passwordHashByte, nil
}

func getUserHashBytes(logger lager.Logger, usernameHash string, username string) ([]byte, error) {
	var usernameHashByte []byte
	var err error
	if usernameHash == "" {
		// when username and password are set for health check
		usernameHashByte, err = bcrypt.GenerateFromPassword([]byte(username), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
		if err != nil {
			logger.Error("failed-new-server-username", err)
			return nil, err
		}
	} else {
		usernameHashByte = []byte(usernameHash)
	}
	return usernameHashByte, err
}
