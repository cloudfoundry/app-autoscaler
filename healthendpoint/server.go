package healthendpoint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

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

// Middleware basic authentication middleware functionality for healthcheck
func (bam *basicAuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
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
func NewServerWithBasicAuth(healthCheckers []Checker, logger lager.Logger, port int, gatherer prometheus.Gatherer, username string, password string, usernameHash string, passwordHash string) (ifrit.Runner, error) {
	healthRouter, err := NewHealthRouter(healthCheckers, logger, gatherer, username, password, usernameHash, passwordHash)
	if err != nil {
		return nil, err
	}
	var addr string
	if os.Getenv("APP_AUTOSCALER_TEST_RUN") == "true" {
		addr = fmt.Sprintf("localhost:%d", port)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", port)
	}

	logger.Info("new-health-server-basic-auth", lager.Data{"addr": addr})
	return http_server.New(addr, healthRouter), nil
}

func NewHealthRouter(healthCheckers []Checker, logger lager.Logger, gatherer prometheus.Gatherer, username string, password string, usernameHash string, passwordHash string) (*mux.Router, error) {
	logger.Info("new-health-server", lager.Data{"####username": username, "####password": password})
	var healthRouter *mux.Router
	var err error
	if username == "" && password == "" {
		//when username and password are not set then don't use basic authentication
		healthRouter = mux.NewRouter()
		r := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
		healthRouter.PathPrefix("").Handler(r)
		healthRouter.Handle("/health/readiness", common.VarsFunc(readiness(healthCheckers)))
	} else {
		healthRouter, err = HealthBasicAuthRouter(healthCheckers, logger, gatherer, username, password, usernameHash, passwordHash)
		if err != nil {
			return nil, err
		}
	}
	return healthRouter, nil
}

func HealthBasicAuthRouter(
	healthCheckers []Checker,
	logger lager.Logger,
	gatherer prometheus.Gatherer,
	username string,
	password string,
	usernameHash string,
	passwordHash string) (*mux.Router, error) {
	basicAuthentication, err := createBasicAuthMiddleware(logger, usernameHash, username, passwordHash, password)
	if err != nil {
		return nil, err
	}
	promHandler := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})

	// /health
	router := mux.NewRouter()
	// unauthenticated paths
	router.Handle("/health/readiness", common.VarsFunc(readiness(healthCheckers)))

	//authenticated paths
	health := router.Path("/health").Subrouter()
	health.Use(basicAuthentication.Middleware)

	everything := router.PathPrefix("").Subrouter()
	everything.Use(basicAuthentication.Middleware)
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

type ReadinessCheck struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}
type readinessResponse struct {
	Status string           `json:"status"`
	Checks []ReadinessCheck `json:"checks"`
}

type Checker func() ReadinessCheck

func readiness(checkers []Checker) func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		w.Header().Set("Content-Type", "application/json")

		checks := make([]ReadinessCheck, 0, 8)

		for _, checker := range checkers {
			check := checker()
			checks = append(checks, check)
		}
		response, err := json.Marshal(readinessResponse{Status: "OK", Checks: checks})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"Internal error"}`))
		}
		_, _ = w.Write(response)
	}
}
