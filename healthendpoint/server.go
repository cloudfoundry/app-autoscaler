package healthendpoint

// package healthendpoint runs the health endpoint server for different autoscaler components
//
// The health endpoint server is used to expose the health
// and readiness check endpoints for the autoscaler components.
//
// It provides the following functionalities:
// - Expose the health and readiness check endpoints
// - Expose the prometheus metrics
// - Expose the pprof endpoints
// - Basic authentication for health and readiness check endpoints
// - Basic authentication for pprof endpoints
// - Basic authentication for prometheus metrics
//
// The healthendpoint package provide the following checkers:
// - NewHTTPChecker
// - NewTCPChecker
// - NewDatabaseChecker
//
// The health endpoint server is used by the following autoscaler components:
// - metricsforwarder
// - scalingengine
// - eventgenerator
// - scheduler

import (
	"net/http/pprof"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/server/common"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// basic authentication credentials struct

func NewHealthRouter(conf helpers.HealthConfig, healthCheckers []Checker, logger lager.Logger, gatherer prometheus.Gatherer, time func() time.Time) (*mux.Router, error) {
	var healthRouter *mux.Router
	var err error
	username := conf.BasicAuth.Username
	password := conf.BasicAuth.Password
	usernameHash := conf.BasicAuth.UsernameHash
	passwordHash := conf.BasicAuth.PasswordHash
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

func healthBasicAuthRouter(conf helpers.HealthConfig, healthCheckers []Checker, logger lager.Logger, gatherer prometheus.Gatherer, time func() time.Time) (*mux.Router, error) {
	ba, err := helpers.CreateBasicAuthMiddleware(logger, conf.BasicAuth)
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
	health.Use(ba.BasicAuthenticationMiddleware)
	pprofRouter := router.PathPrefix("/debug/pprof").Subrouter()
	pprofRouter.Use(ba.BasicAuthenticationMiddleware)

	pprofRouter.HandleFunc("/cmdline", pprof.Cmdline)
	pprofRouter.HandleFunc("/profile", pprof.Profile)
	pprofRouter.HandleFunc("/symbol", pprof.Symbol)
	pprofRouter.HandleFunc("/trace", pprof.Trace)
	pprofRouter.PathPrefix("").HandlerFunc(pprof.Index)

	everything := router.PathPrefix("").Subrouter()
	everything.Use(ba.BasicAuthenticationMiddleware)
	everything.PathPrefix("").Handler(promHandler)

	return router, nil
}
