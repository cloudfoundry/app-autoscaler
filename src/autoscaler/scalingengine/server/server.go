package server

import (
	"autoscaler/db"
	"autoscaler/scalingengine"
	"autoscaler/scalingengine/config"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"

	"fmt"
	"net/http"
)

const (
	PathScale                = "/v1/apps/{appid}/scale"
	PathScalingHistories     = "/v1/apps/{appid}/scaling_histories"
	PathActiveSchedule       = "/v1/apps/{appid}/active_schedules/{scheduleid}"
	RouteNameScale           = "scale"
	RouteNameHistoreis       = "histories"
	RouteNameActiveSchedules = "activeSchedules"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, conf config.ServerConfig, historyDB db.HistoryDB, scalingEngine scalingengine.ScalingEngine) ifrit.Runner {
	handler := NewScalingHandler(logger, historyDB, scalingEngine)

	r := mux.NewRouter()
	r.Methods("POST").Path(PathScale).Handler(VarsFunc(handler.Scale)).Name(RouteNameScale)
	r.Methods("GET").Path(PathScalingHistories).Handler(VarsFunc(handler.GetScalingHistories)).Name(RouteNameHistoreis)
	r.Methods("PUT").Path(PathActiveSchedule).Handler(VarsFunc(handler.StartActiveSchedule)).Name(RouteNameActiveSchedules)
	r.Methods("DELETE").Path(PathActiveSchedule).Handler(VarsFunc(handler.RemoveActiveSchedule)).Name(RouteNameActiveSchedules)

	addr := fmt.Sprintf("0.0.0.0:%d", conf.Port)
	return http_server.New(addr, r)
}
