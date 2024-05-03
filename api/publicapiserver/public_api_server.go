package publicapiserver

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/apis/scalinghistory"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/v3"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewPublicApiServer(logger lager.Logger, conf *config.Config, policydb db.PolicyDB, credentials cred_helper.Credentials,
	checkBindingFunc api.CheckBindingFunc, cfclient cf.CFClient, httpStatusCollector healthendpoint.HTTPStatusCollector,
	rateLimiter ratelimiter.Limiter, bindingdb db.BindingDB) (ifrit.Runner, error) {
	pah := NewPublicApiHandler(logger, conf, policydb, bindingdb, credentials)

	scalingHistoryHandler, err := newScalingHistoryHandler(logger, conf)
	if err != nil {
		return nil, err
	}

	mw := NewMiddleware(logger, cfclient, checkBindingFunc, conf.APIClientId)
	rateLimiterMiddleware := ratelimiter.NewRateLimiterMiddleware("appId", rateLimiter, logger.Session("api-ratelimiter-middleware"))
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	r := routes.ApiOpenRoutes()
	r.Use(otelmux.Middleware("apiserver"))
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.PublicApiInfoRouteName).Handler(VarsFunc(pah.GetApiInfo))
	r.Get(routes.PublicApiHealthRouteName).Handler(VarsFunc(pah.GetHealth))

	rp := routes.ApiRoutes()
	rp.Use(rateLimiterMiddleware.CheckRateLimit)
	rp.Use(mw.HasClientToken)
	rp.Use(mw.Oauth)
	rp.Use(httpStatusCollectMiddleware.Collect)

	rp.Get(routes.PublicApiScalingHistoryRouteName).Handler(scalingHistoryHandler)
	rp.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).Handler(VarsFunc(pah.GetAggregatedMetricsHistories))

	rpolicy := routes.ApiPolicyRoutes()
	rpolicy.Use(rateLimiterMiddleware.CheckRateLimit)
	rpolicy.Use(mw.HasClientToken)
	rpolicy.Use(mw.Oauth)
	rpolicy.Use(mw.CheckServiceBinding)

	rpolicy.Use(httpStatusCollectMiddleware.Collect)
	rpolicy.Get(routes.PublicApiGetPolicyRouteName).Handler(VarsFunc(pah.GetScalingPolicy))
	rpolicy.Get(routes.PublicApiAttachPolicyRouteName).Handler(VarsFunc(pah.AttachScalingPolicy))
	rpolicy.Get(routes.PublicApiDetachPolicyRouteName).Handler(VarsFunc(pah.DetachScalingPolicy))

	rcredential := routes.ApiCredentialRoutes()
	rcredential.Use(rateLimiterMiddleware.CheckRateLimit)

	rcredential.Use(httpStatusCollectMiddleware.Collect)
	rcredential.Use(mw.HasClientToken)
	rcredential.Use(mw.Oauth)

	return helpers.NewHTTPServer(logger, conf.PublicApiServer, r)
}

func newScalingHistoryHandler(logger lager.Logger, conf *config.Config) (http.Handler, error) {
	scalingHistoryHandler, err := NewScalingHistoryHandler(logger, conf)
	if err != nil {
		return nil, fmt.Errorf("error creating scaling history handler: %w", err)
	}
	scalingHistoryServer, err := scalinghistory.NewServer(scalingHistoryHandler, scalingHistoryHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating ogen scaling history server: %w", err)
	}
	return scalingHistoryServer, nil
}
