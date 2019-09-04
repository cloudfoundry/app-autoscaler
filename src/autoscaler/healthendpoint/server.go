package healthendpoint

import (
	"code.cloudfoundry.org/lager"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"net/http"
	_ "net/http/pprof"
)

func NewServer(logger lager.Logger, port int, gatherer prometheus.Gatherer) (ifrit.Runner, error) {
	router := mux.NewRouter()
	r := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
	router.PathPrefix("/debug/pprof").Handler(http.DefaultServeMux)
	router.PathPrefix("").Handler(r)
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	logger.Info("new-health-server", lager.Data{"addr": addr})
	return http_server.New(addr, router), nil
}
