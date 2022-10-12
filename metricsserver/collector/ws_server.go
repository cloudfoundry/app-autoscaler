package collector

import (
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

func NewWSServer(logger lager.Logger, tls models.TLSCerts, port int, keepAlive time.Duration, envelopeChannels []chan *loggregator_v2.Envelope, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {
	wsHandler := NewWSMessageHandler(logger.Session("ws_handler"), envelopeChannels, keepAlive)
	var addr string
	if os.Getenv("APP_AUTOSCALER_TEST_RUN") == "true" {
		addr = fmt.Sprintf("localhost:%d", port)
	} else {
		addr = fmt.Sprintf("0.0.0.0:%d", port)
	}

	var runner ifrit.Runner
	if (tls.KeyFile == "") || (tls.CertFile == "") {
		runner = http_server.New(addr, wsHandler)
	} else {
		tlsConfig, err := tls.CreateServerConfig()
		if err != nil {
			logger.Error("failed-new-websocket-server-new-tls-config", err)
			return nil, fmt.Errorf("ws server tls config error:%w", err)
		}
		runner = http_server.NewTLSServer(addr, wsHandler, tlsConfig)
	}
	logger.Info("websocket-server-created", lager.Data{"port": port})
	return runner, nil
}
