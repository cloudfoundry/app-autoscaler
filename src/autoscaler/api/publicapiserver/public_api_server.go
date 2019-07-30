package publicapiserver

import (
	"fmt"
	"net/http"

	"autoscaler/api/config"
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/routes"

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

func NewPublicApiServer(logger lager.Logger, conf *config.Config, policydb db.PolicyDB, bindingDB db.BindingDB, cfclient cf.CFClient) (ifrit.Runner, error) {

	pah := NewPublicApiHandler(logger, conf, policydb)
	oam := NewOauthMiddleware(logger, cfclient)
	cbm := NewCheckBindingMiddleware(logger, bindingDB)

	r := routes.PublicApiRoutes()
	r.Get(routes.PublicApiInfoRouteName).Handler(VarsFunc(pah.GetApiInfo))
	r.Get(routes.PublicApiHealthRouteName).Handler(VarsFunc(pah.GetHealth))

	rp := routes.PublicApiProtectedRoutes()
	rp.Use(oam.Middleware)
	rp.Get(routes.PublicApiScalingHistoryRouteName).Handler(VarsFunc(pah.GetScalingHistories))
	rp.Get(routes.PublicApiMetricsHistoryRouteName).Handler(VarsFunc(pah.GetInstanceMetricsHistories))
	rp.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).Handler(VarsFunc(pah.GetAggregatedMetricsHistories))

	rpolicy := routes.PublicApiPolicyRoutes()
	rpolicy.Use(oam.Middleware)
	if !conf.UseBuildInMode {
		rpolicy.Use(cbm.CheckServiceBinding)
	}
	rpolicy.Get(routes.PublicApiGetPolicyRouteName).Handler(VarsFunc(pah.GetScalingPolicy))
	rpolicy.Get(routes.PublicApiAttachPolicyRouteName).Handler(VarsFunc(pah.AttachScalingPolicy))
	rpolicy.Get(routes.PublicApiDetachPolicyRouteName).Handler(VarsFunc(pah.DetachScalingPolicy))

	rcredential := routes.PublicApiCustomMetricsCredentialRoutes()
	rcredential.Use(oam.Middleware)
	if !conf.UseBuildInMode {
		rcredential.Use(cbm.CheckServiceBinding)
	}
	rcredential.Get(routes.PublicApiGetCustomMetricsCredentialRouteName).Handler(VarsFunc(pah.GetCustomMetricsCredential))
	rcredential.Get(routes.PublicApiCreateCustomMetricsCredentialRouteName).Handler(VarsFunc(pah.CreateCustomMetricsCredential))
	rcredential.Get(routes.PublicApiDeleteCustomMetricsCredentialRouteName).Handler(VarsFunc(pah.DeleteCustomMetricsCredential))

	addr := fmt.Sprintf("0.0.0.0:%d", conf.PublicApiServer.Port)

	var runner ifrit.Runner
	if (conf.PublicApiServer.TLS.KeyFile == "") || (conf.PublicApiServer.TLS.CertFile == "") {
		logger.Info("creating-public-api-http-server")
		runner = http_server.New(addr, r)
	} else {
		logger.Info("creating-public-api-https-server")
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
