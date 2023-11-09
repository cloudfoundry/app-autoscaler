package collector

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
)

func NewWSServer(logger lager.Logger, tls models.TLSCerts, port int, keepAlive time.Duration, envelopeChannels []chan *loggregator_v2.Envelope, httpStatusCollector healthendpoint.HTTPStatusCollector) (ifrit.Runner, error) {
	wsHandler := NewWSMessageHandler(logger.Session("ws_handler"), envelopeChannels, keepAlive)

	httpServerConfig := helpers.ServerConfig{
		Port: port,
		TLS:  tls,
	}

	return helpers.NewHTTPServer(logger, httpServerConfig, wsHandler)
}
