package healthendpoint

import (
	"fmt"

	"code.cloudfoundry.org/lager"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

func NewServer(logger lager.Logger, port int, gatherer prometheus.Gatherer) (ifrit.Runner, error) {

	r := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	logger.Info("new-health-server", lager.Data{"addr": addr})
	return http_server.New(addr, r), nil
}
