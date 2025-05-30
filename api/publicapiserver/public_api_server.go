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

	brokerServer brokerserver.BrokerServer

	autoscalerRouter *routes.Router

	healthRouter              *mux.Router
	publicApiServerMiddleware *Middleware
	rateLimiterMiddleware     *ratelimiter.RateLimiterMiddleware
}

func NewPublicApiServer(logger lager.Logger, conf *config.Config, policyDB db.PolicyDB,
	bindingDB db.BindingDB, credentials cred_helper.Credentials, checkBindingFunc api.CheckBindingFunc,
	cfClient cf.CFClient, httpStatusCollector healthendpoint.HTTPStatusCollector,
	rateLimiter ratelimiter.Limiter, brokerServer brokerserver.BrokerServer) *PublicApiServer {
	return &PublicApiServer{
		logger:                    logger,
		conf:                      conf,
		policyDB:                  policyDB,
		bindingDB:                 bindingDB,
		credentials:               credentials,
		checkBindingFunc:          checkBindingFunc,
		cfClient:                  cfClient,
		httpStatusCollector:       httpStatusCollector,
		brokerServer:              brokerServer,
		autoscalerRouter:          routes.NewRouter(),
		publicApiServerMiddleware: NewMiddleware(logger, cfClient, checkBindingFunc, conf.APIClientId),
		rateLimiterMiddleware:     ratelimiter.NewRateLimiterMiddleware("appId", rateLimiter, logger.Session("api-ratelimiter-middleware")),
	}
}

func (s *PublicApiServer) CreateHealthServer() (ifrit.Runner, error) {
	if err := s.setupHealthRouter(); err != nil {
		return nil, err
	}

	return helpers.NewHTTPServer(s.logger.Session("HealthServer"), s.conf.Health.ServerConfig, s.healthRouter)
}

func (s *PublicApiServer) setupBrokerRouter() error {
	brokerRouter, err := s.brokerServer.GetRouter()
	if err != nil {
		return err
	}

	s.autoscalerRouter.GetRouter().PathPrefix("/v2").Handler(brokerRouter)

	return nil
}

func (s *PublicApiServer) CreateCFServer() (ifrit.Runner, error) {
	if err := s.setupBrokerRouter(); err != nil {
		return nil, err
	}

	if err := s.setupHealthRouter(); err != nil {
		return nil, err
	}

	if err := s.setupApiRoutes(); err != nil {
		return nil, err
	}

	r := s.autoscalerRouter.GetRouter()

	return helpers.NewHTTPServer(s.logger.Session("CfServer"), s.conf.CFServer, r)
}

func (s *PublicApiServer) CreateMtlsServer() (ifrit.Runner, error) {
	if err := s.setupApiRoutes(); err != nil {
		return nil, err
	}

	return helpers.NewHTTPServer(s.logger.Session("MtlsServer"), s.conf.Server, s.autoscalerRouter.GetRouter())
}

func (s *PublicApiServer) setupApiProtectedRoutes(pah *PublicApiHandler, scalingHistoryHandler http.Handler) {
	apiProtectedRouter := s.autoscalerRouter.CreateApiSubrouter()
	apiProtectedRouter.Use(otelmux.Middleware("apiserver"))
	apiProtectedRouter.Use(healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector).Collect)
	apiProtectedRouter.Use(s.rateLimiterMiddleware.CheckRateLimit)
	apiProtectedRouter.Use(s.publicApiServerMiddleware.HasClientToken)
	apiProtectedRouter.Use(s.publicApiServerMiddleware.Oauth)
	apiProtectedRouter.Use(s.publicApiServerMiddleware.CheckServiceBinding)
	apiProtectedRouter.Use(healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector).Collect)
	apiProtectedRouter.Get(routes.PublicApiScalingHistoryRouteName).Handler(scalingHistoryHandler)
	apiProtectedRouter.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).Handler(VarsFunc(pah.GetAggregatedMetricsHistories))
}

func (s *PublicApiServer) setupPolicyRoutes(pah *PublicApiHandler) {
	rpolicy := s.autoscalerRouter.CreateApiPolicySubrouter()
	rpolicy.Use(s.rateLimiterMiddleware.CheckRateLimit)
	rpolicy.Use(s.publicApiServerMiddleware.HasClientToken)
	rpolicy.Use(s.publicApiServerMiddleware.Oauth)
	rpolicy.Use(s.publicApiServerMiddleware.CheckServiceBinding)
	rpolicy.Use(healthendpoint.NewHTTPStatusCollectMiddleware(s.httpStatusCollector).Collect)
	rpolicy.Get(routes.PublicApiGetPolicyRouteName).Handler(VarsFunc(pah.GetScalingPolicy))
	rpolicy.Get(routes.PublicApiAttachPolicyRouteName).Handler(VarsFunc(pah.AttachScalingPolicy))
	rpolicy.Get(routes.PublicApiDetachPolicyRouteName).Handler(VarsFunc(pah.DetachScalingPolicy))
}

func (s *PublicApiServer) setupPublicApiRoutes(pah *PublicApiHandler) {
	apiPublicRouter := s.autoscalerRouter.CreateApiPublicSubrouter()
	apiPublicRouter.Get(routes.PublicApiInfoRouteName).Handler(VarsFunc(pah.GetApiInfo))
	apiPublicRouter.Get(routes.PublicApiHealthRouteName).Handler(VarsFunc(pah.GetHealth))
}

func (s *PublicApiServer) setupApiRoutes() error {
	publicApiHandler := NewPublicApiHandler(s.logger, s.conf, s.policyDB, s.bindingDB, s.credentials)
	scalingHistoryHandler, err := s.newScalingHistoryHandler()
	if err != nil {
		return err
	}
	s.setupApiProtectedRoutes(publicApiHandler, scalingHistoryHandler)
	s.setupPublicApiRoutes(publicApiHandler)
	s.setupPolicyRoutes(publicApiHandler)

	return nil
}

func (s *PublicApiServer) setupHealthRouter() error {
	checkers := []healthendpoint.Checker{}
	gatherer := s.createPrometheusRegistry()

	healthRouter, err := healthendpoint.NewHealthRouter(s.conf.Health, checkers, s.logger.Session("health-server"), gatherer, time.Now)
	if err != nil {
		return fmt.Errorf("failed to create health router: %w", err)
	}

	s.healthRouter = healthRouter

	return nil
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
