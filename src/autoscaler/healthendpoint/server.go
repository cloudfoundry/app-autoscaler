package healthendpoint

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

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

// basic authentication middleware functionality for healthcheck
func (bam *basicAuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, authOK := r.BasicAuth()

		if authOK == false || bcrypt.CompareHashAndPassword(bam.usernameHash, []byte(username)) != nil || bcrypt.CompareHashAndPassword(bam.passwordHash, []byte(password)) != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewServer(logger lager.Logger, port int, gatherer prometheus.Gatherer) (ifrit.Runner, error) {

	router := mux.NewRouter()
	r := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
	router.PathPrefix("/debug/pprof").Handler(http.DefaultServeMux)
	router.PathPrefix("").Handler(r)
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	logger.Info("new-health-server", lager.Data{"addr": addr})
	return http_server.New(addr, router), nil
}

// open the healthcheck port with basic authentication.
// Make sure that username and password is not empty
func NewServerWithBasicAuth(logger lager.Logger, port int, gatherer prometheus.Gatherer, username string, password string, usernameHash string, passwordHash string) (ifrit.Runner, error) {
	logger.Info("new-health-server", lager.Data{"####username": username, "####password": password})
	if username == "" && password == "" {
		//when username and password are not set then dont use basic authentication
		healthServer, err := NewServer(logger, port, gatherer)
		return healthServer, err

	} else {
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

		var passwordHashByte []byte
		if passwordHash == "" {
			passwordHashByte, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost) // use MinCost as the config already provided it as cleartext
			if err != nil {
				logger.Error("failed-new-server-password", err)
				return nil, err
			}
		} else {
			passwordHashByte = []byte(passwordHash)
		}

		basicAuthentication := &basicAuthenticationMiddleware{
			usernameHash: usernameHashByte,
			passwordHash: passwordHashByte,
		}

		r := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})

		// basic authentication middleware
		middleWareHandlerRouter := mux.NewRouter()
		middleWareHandlerRouter.Use(basicAuthentication.Middleware)

		// add router path and router handler
		middleWareHandlerRouter.Handle("/health", r)

		middleWareHandlerRouter.PathPrefix("/debug/pprof").Handler(http.DefaultServeMux)
		middleWareHandlerRouter.PathPrefix("").Handler(r)

		addr := fmt.Sprintf("0.0.0.0:%d", port)
		logger.Info("new-health-server-basic-auth", lager.Data{"addr": addr})

		return http_server.New(addr, middleWareHandlerRouter), nil
	}
}
