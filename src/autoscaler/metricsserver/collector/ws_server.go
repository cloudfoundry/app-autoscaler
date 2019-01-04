package collector

import (
	"autoscaler/healthendpoint"
	"autoscaler/models"
	"fmt"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

func NewWSServer(logger lager.Logger, tls models.TLSCerts, port int, keepAlive time.Duration, envelopeChannels []chan *loggregator_v2.Envelope, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {
	wsHandler := NewWSMessageHandler(logger.Session("ws_handler"), envelopeChannels, keepAlive)
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	var runner ifrit.Runner
	if (tls.KeyFile == "") || (tls.CertFile == "") {
		runner = http_server.New(addr, wsHandler)

	} else {
		tlsConfig, err := cfhttp.NewTLSConfig(tls.CertFile, tls.KeyFile, tls.CACertFile)
		if err != nil {
			logger.Error("failed-new-websocket-server-new-tls-config", err)
			return nil, err
		}
		runner = http_server.NewTLSServer(addr, wsHandler, tlsConfig)
	}
	logger.Info("websocket-server-created", lager.Data{"port": port})
	return runner, nil
}
