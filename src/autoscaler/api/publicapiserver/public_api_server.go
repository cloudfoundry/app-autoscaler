package publicapiserver

import (
	"fmt"
	"net/http"

	"autoscaler/api/config"
	"autoscaler/db"
	"autoscaler/routes"

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

func NewPublicApiServer(logger lager.Logger, conf *config.Config, policydb db.PolicyDB) (ifrit.Runner, error) {

	pah := NewPublicApiHandler(logger, conf, policydb)
	oam := NewOauthMiddleware(logger, conf)

	r := routes.PublicApiRoutes()
	r.Get(routes.PublicApiInfoRouteName).Handler(VarsFunc(pah.GetApiInfo))
	r.Get(routes.PublicApiHealthRouteName).Handler(VarsFunc(pah.GetHealth))

	rp := routes.PublicApiProtectedRoutes()
	rp.Use(oam.Middleware)
	rp.Get(routes.PublicApiScalingHistoryRouteName).Handler(VarsFunc(pah.GetScalingHistories))
	rp.Get(routes.PublicApiMetricsHistoryRouteName).Handler(VarsFunc(pah.GetInstanceMetricsHistories))
	rp.Get(routes.PublicApiAggregatedMetricsHistoryRouteName).Handler(VarsFunc(pah.GetAggregatedMetricsHistories))

	addr := fmt.Sprintf("0.0.0.0:%d", conf.PublicApiServer.Port)

	var runner ifrit.Runner
	runner = http_server.New(addr, r)

	logger.Info("public-api-http-server-created", lager.Data{"serverConfig": conf.PublicApiServer})
	return runner, nil
}
