package collector

import (
	"autoscaler/healthendpoint"
	"autoscaler/metricsserver/config"
	"fmt"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

func NewWSServer(logger lager.Logger, conf *config.Config, envelopeChannels []chan *loggregator_v2.Envelope, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {
	wsHandler := NewWSMessageHandler(logger.Session("ws_handler"), envelopeChannels, conf.Collector.KeepAliveTime)
	addr := fmt.Sprintf("0.0.0.0:%d", conf.Server.Port)

	var runner ifrit.Runner
	if (conf.Server.TLS.KeyFile == "") || (conf.Server.TLS.CertFile == "") {
		runner = http_server.New(addr, wsHandler)

	} else {
		tlsConfig, err := cfhttp.NewTLSConfig(conf.Server.TLS.CertFile, conf.Server.TLS.KeyFile, conf.Server.TLS.CACertFile)
		if err != nil {
			logger.Error("failed-new-websocket-server-new-tls-config", err, lager.Data{"tls": conf.Server.TLS})
			return nil, err
		}
		runner = http_server.NewTLSServer(addr, wsHandler, tlsConfig)
	}
	logger.Info("websocket-server-created", lager.Data{"port": conf.Server.Port})
	return runner, nil
}
