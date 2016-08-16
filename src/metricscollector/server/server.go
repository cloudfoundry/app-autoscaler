package server

import (
	"fmt"
	"net/http"

	"cf"
	"db"
	"metricscollector/config"
	"metricscollector/noaa"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

const (
	PathMemoryMetric             = "/v1/apps/{appid}/metrics/memory"
	PathMemoryMetricHistory      = "/v1/apps/{appid}/metrics_history/memory"
	RouteNameMemoryMetric        = "memory-metric"
	RouteNameMemoryMetricHistory = "memory-metric-history"
)

type VarsFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func (vh VarsFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vh(w, r, vars)
}

func NewServer(logger lager.Logger, conf config.ServerConfig, cfc cf.CfClient, consumer noaa.NoaaConsumer, database db.MetricsDB) ifrit.Runner {
	mmh := NewMemoryMetricHandler(logger, cfc, consumer, database)

	r := mux.NewRouter()
	r.Methods("GET").Path(PathMemoryMetric).Handler(VarsFunc(mmh.GetMemoryMetric)).Name(RouteNameMemoryMetric)
	r.Methods("GET").Path(PathMemoryMetricHistory).Handler(VarsFunc(mmh.GetMemoryMetricHistory)).Name(RouteNameMemoryMetricHistory)

	addr := fmt.Sprintf("0.0.0.0:%d", conf.Port)
	return http_server.New(addr, r)
}
