package publicapiserver

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/apis/scalinghistory"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/brokerserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tedsuo/ifrit"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vh(w, r, mux.Vars(r))
}

type PublicApiServer struct {
	logger              lager.Logger
	conf                *config.Config
	policyDB            db.PolicyDB
	bindingDB           db.BindingDB
	credentials         cred_helper.Credentials
	checkBindingFunc    api.CheckBindingFunc
	cfClient            cf.CFClient
	httpStatusCollector healthendpoint.HTTPStatusCollector
	rateLimiter         ratelimiter.Limiter
	brokerServer        brokerserver.BrokerServer

	healthRouter *mux.Router
}

func NewPublicApiServer(logger lager.Logger, conf *config.Config, policyDB db.PolicyDB,
	bindingDB db.BindingDB, credentials cred_helper.Credentials, checkBindingFunc api.CheckBindingFunc,
	cfClient cf.CFClient, httpStatusCollector healthendpoint.HTTPStatusCollector,
	rateLimiter ratelimiter.Limiter, brokerServer brokerserver.BrokerServer) *PublicApiServer {
	return &PublicApiServer{
		logger:              logger,
		conf:                conf,
		policyDB:            policyDB,
		bindingDB:           bindingDB,
		credentials:         credentials,
		checkBindingFunc:    checkBindingFunc,
		cfClient:            cfClient,
		httpStatusCollector: httpStatusCollector,
		rateLimiter:         rateLimiter,
		brokerServer:        brokerServer,
	}
}

func (s *PublicApiServer) Setup() error {
	hr, err := s.createHealthRouter()
	if err != nil {
		return err
	}

	s.healthRouter = hr

	return nil
}

func (s *PublicApiServer) GetHealthServer() (ifrit.Runner, error) {
	return helpers.NewHTTPServer(s.logger, s.conf.Health.ServerConfig, s.healthRouter)
}

func (s *PublicApiServer) GetUnifiedServer() (ifrit.Runner, error) {
	pah := NewPublicApiHandler(s.logger, s.conf, s.policyDB, s.bindingDB, s.credentials)
	scalingHistoryHandler, err := s.newScalingHistoryHandler()
	if err != nil {
		return nil, err
	}

	r := s.setupApiRoutes(pah, scalingHistoryHandler)

	brokerRouter, err := s.brokerServer.GetRouter()
	if err != nil {
		return nil, err
	}

	return helpers.NewHTTPServer(s.logger, s.conf.VCAPServer, s.setupVCAPRouter(r, s.healthRouter, brokerRouter))
}

func (s *PublicApiServer) GetMtlsServer() (ifrit.Runner, error) {
	pah := NewPublicApiHandler(s.logger, s.conf, s.policyDB, s.bindingDB, s.credentials)
	scalingHistoryHandler, err := s.newScalingHistoryHandler()
	if err != nil {
		return nil, err
	}

	r := s.setupApiRoutes(pah, scalingHistoryHandler)

	return helpers.NewHTTPServer(s.logger, s.conf.Server, s.setupMainRouter(r, s.healthRouter))
}

func (s *PublicApiServer) setupApiRoutes(pah *PublicApiHandler, scalingHistoryHandler http.Handler) *mux.Router {
	r := routes.ApiOpenRoutes()
	r.Use(otelmux.Middleware("apiserver"))
	r.Use(healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector).Collect)

	r.Get(routes.PublicApiInfoRouteName).Handler(VarsFunc(pah.GetApiInfo))
	r.Get(routes.PublicApiHealthRouteName).Handler(VarsFunc(pah.GetHealth))

	rp := routes.ApiRoutes()
	rateLimiterMiddleware := ratelimiter.NewRateLimiterMiddleware("appId", s.rateLimiter, s.logger.Session("api-ratelimiter-middleware"))
	rp.Use(rateLimiterMiddleware.CheckRateLimit)
	rp.Use(NewMiddleware(s.logger, s.cfClient, s.checkBindingFunc, s.conf.APIClientId).HasClientToken)
	rp.Use(NewMiddleware(s.logger, s.cfClient, s.checkBindingFunc, s.conf.APIClientId).Oauth)
	rp.Use(NewMiddleware(s.logger, s.cfClient, s.checkBindingFunc, s.conf.APIClientId).CheckServiceBinding)
	rp.Use(healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector).Collect)

	rp.Get(routes.PublicApiScalingHistoryRouteName).Handler(scalingHistoryHandler)
	rp.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).Handler(VarsFunc(pah.GetAggregatedMetricsHistories))

	s.setupPolicyRoutes(rp, pah)

	return r
}

func (s *PublicApiServer) setupPolicyRoutes(rp *mux.Router, pah *PublicApiHandler) {
	rpolicy := routes.ApiPolicyRoutes()
	rlm := ratelimiter.NewRateLimiterMiddleware("appId", s.rateLimiter, s.logger.Session("api-ratelimiter-middleware"))
	pasm := NewMiddleware(s.logger, s.cfClient, s.checkBindingFunc, s.conf.APIClientId)
	rpolicy.Use(rlm.CheckRateLimit)
	rpolicy.Use(pasm.HasClientToken)
	rpolicy.Use(pasm.Oauth)
	rpolicy.Use(pasm.CheckServiceBinding)
	rpolicy.Use(healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector).Collect)

	rpolicy.Get(routes.PublicApiGetPolicyRouteName).Handler(VarsFunc(pah.GetScalingPolicy))
	rpolicy.Get(routes.PublicApiAttachPolicyRouteName).Handler(VarsFunc(pah.AttachScalingPolicy))
	rpolicy.Get(routes.PublicApiDetachPolicyRouteName).Handler(VarsFunc(pah.DetachScalingPolicy))
}

func (s *PublicApiServer) createHealthRouter() (*mux.Router, error) {
	checkers := []healthendpoint.Checker{}
	gatherer := s.createPrometheusRegistry()
	healthRouter, err := healthendpoint.NewHealthRouter(s.conf.Health, checkers, s.logger.Session("health-server"), gatherer, time.Now)
	if err != nil {
		return nil, fmt.Errorf("failed to create health router: %w", err)
	}

	s.logger.Debug("Successfully created health server")
	return healthRouter, nil
}

func (s *PublicApiServer) createPrometheusRegistry() *prometheus.Registry {
	promRegistry := prometheus.NewRegistry()
	healthendpoint.RegisterCollectors(promRegistry,
		[]prometheus.Collector{
			healthendpoint.NewDatabaseStatusCollector("autoscaler", "golangapiserver", "policyDB", s.policyDB),
			healthendpoint.NewDatabaseStatusCollector("autoscaler", "golangapiserver", "bindingDB", s.bindingDB),
			s.httpStatusCollector,
		},
		true, s.logger.Session("golangapiserver-prometheus"))
	return promRegistry
}

func (s *PublicApiServer) newScalingHistoryHandler() (http.Handler, error) {
	ss := SecuritySource{}
	scalingHistoryHandler, err := NewScalingHistoryHandler(s.logger, s.conf)
	if err != nil {
		return nil, fmt.Errorf("error creating scaling history handler: %w", err)
	}
	return scalinghistory.NewServer(scalingHistoryHandler, ss)
}

func (s *PublicApiServer) setupVCAPRouter(r *mux.Router, healthRouter *mux.Router, brokerRouter *chi.Mux) *mux.Router {
	mainRouter := mux.NewRouter()

	mainRouter.PathPrefix("/v2").Handler(brokerRouter)
	mainRouter.PathPrefix("/v1").Handler(r)
	mainRouter.PathPrefix("/health").Handler(healthRouter)
	mainRouter.PathPrefix("/").Handler(healthRouter)

	return mainRouter
}

func (s *PublicApiServer) setupMainRouter(r *mux.Router, healthRouter *mux.Router) *mux.Router {
	mainRouter := mux.NewRouter()
	mainRouter.PathPrefix("/v1").Handler(r)
	mainRouter.PathPrefix("/health").Handler(healthRouter)
	mainRouter.PathPrefix("/").Handler(healthRouter)
	return mainRouter
}
