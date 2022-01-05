package publicapiserver

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cred_helper"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/ratelimiter"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
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
	mw := NewMiddleware(logger, cfclient, checkBindingFunc, conf.APIClientId)
	rateLimiterMiddleware := ratelimiter.NewRateLimiterMiddleware("appId", rateLimiter, logger.Session("api-ratelimiter-middleware"))
	httpStatusCollectMiddleware := healthendpoint.NewHTTPStatusCollectMiddleware(httpStatusCollector)
	r := routes.ApiOpenRoutes()
	r.Use(httpStatusCollectMiddleware.Collect)
	r.Get(routes.PublicApiInfoRouteName).Handler(VarsFunc(pah.GetApiInfo))
	r.Get(routes.PublicApiHealthRouteName).Handler(VarsFunc(pah.GetHealth))

	rp := routes.ApiRoutes()
	rp.Use(rateLimiterMiddleware.CheckRateLimit)
	rp.Use(mw.HasClientToken)
	rp.Use(mw.Oauth)
	rp.Use(httpStatusCollectMiddleware.Collect)
	rp.Get(routes.PublicApiScalingHistoryRouteName).Handler(VarsFunc(pah.GetScalingHistories))
	rp.Get(routes.PublicApiMetricsHistoryRouteName).Handler(VarsFunc(pah.GetInstanceMetricsHistories))
	rp.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).Handler(VarsFunc(pah.GetAggregatedMetricsHistories))

	rpolicy := routes.ApiPolicyRoutes()
	rpolicy.Use(rateLimiterMiddleware.CheckRateLimit)
	rpolicy.Use(mw.HasClientToken)
	rpolicy.Use(mw.Oauth)
	if !conf.UseBuildInMode {
		rpolicy.Use(mw.CheckServiceBinding)
	}
	rpolicy.Use(httpStatusCollectMiddleware.Collect)
	rpolicy.Get(routes.PublicApiGetPolicyRouteName).Handler(VarsFunc(pah.GetScalingPolicy))
	rpolicy.Get(routes.PublicApiAttachPolicyRouteName).Handler(VarsFunc(pah.AttachScalingPolicy))
	rpolicy.Get(routes.PublicApiDetachPolicyRouteName).Handler(VarsFunc(pah.DetachScalingPolicy))

	rcredential := routes.ApiCredentialRoutes()
	rcredential.Use(rateLimiterMiddleware.CheckRateLimit)
	if !conf.UseBuildInMode {
		rcredential.Use(mw.RejectCredentialOperationInServiceOffering)
	}
	rcredential.Use(httpStatusCollectMiddleware.Collect)
	rcredential.Use(mw.HasClientToken)
	rcredential.Use(mw.Oauth)
	rcredential.Get(routes.PublicApiCreateCredentialRouteName).Handler(VarsFunc(pah.CreateCredential))
	rcredential.Get(routes.PublicApiDeleteCredentialRouteName).Handler(VarsFunc(pah.DeleteCredential))

	var addr string
	if os.Getenv("APP_AUTOSCALER_TEST_RUN") == "true" {
		addr = fmt.Sprintf("localhost:%d", conf.PublicApiServer.Port)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", conf.PublicApiServer.Port)
	}

	var runner ifrit.Runner
	if (conf.PublicApiServer.TLS.KeyFile == "") || (conf.PublicApiServer.TLS.CertFile == "") {
		logger.Info("creating-public-api-http-server")
		runner = http_server.New(addr, r)
	} else {
		logger.Info("creating-public-api-https-server")
		//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
		tlsConfig, err := cfhttp.NewTLSConfig(conf.PublicApiServer.TLS.CertFile, conf.PublicApiServer.TLS.KeyFile, conf.PublicApiServer.TLS.CACertFile)
		if err != nil {
			logger.Error("failed-new-server-new-tls-config", err, lager.Data{"tls": conf.PublicApiServer.TLS})
			return nil, err
		}
		runner = http_server.NewTLSServer(addr, r, tlsConfig)
	}

	logger.Info("public-api-http-server-created", lager.Data{"serverConfig": conf.PublicApiServer})
	return runner, nil
}
