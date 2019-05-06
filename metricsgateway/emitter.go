package metricsgateway

import (
	"crypto/tls"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"

	"autoscaler/metricsgateway/helpers"
)

type Emitter interface {
	Accept(envelope *loggregator_v2.Envelope)
	Emit(*loggregator_v2.Envelope) error
}

type EnvelopeEmitter struct {
	logger               lager.Logger
	metricsServerAddress string
	handshakeTimeout     time.Duration
	envelopChan          chan *loggregator_v2.Envelope
	bufferSize           int64
	doneChan             chan bool
	keepAliveInterval    time.Duration
	eclock               clock.Clock
	ticker               clock.Ticker
	wsHelper             helpers.WSHelper
}

func NewEnvelopeEmitter(logger lager.Logger, bufferSize int64, metricsServerAddress string, tlsConfig *tls.Config, handshakeTimeout time.Duration, eclock clock.Clock, keepAliveInterval time.Duration, wsHelper helpers.WSHelper) *EnvelopeEmitter {
	return &EnvelopeEmitter{
		logger:               logger.Session("EnvelopeEmitter"),
		metricsServerAddress: metricsServerAddress,
		handshakeTimeout:     handshakeTimeout,
		envelopChan:          make(chan *loggregator_v2.Envelope, bufferSize),
		doneChan:             make(chan bool),
		eclock:               eclock,
		keepAliveInterval:    keepAliveInterval,
		wsHelper:             wsHelper,
	}
}
func (e *EnvelopeEmitter) Start() {
	err := e.wsHelper.SetupConn()
	if err != nil {
		e.logger.Error("failed-to-start-emimtter", err)
		return
	}
	go e.startEmitEnvelope()
	e.logger.Info("started")
}

func (e *EnvelopeEmitter) startEmitEnvelope() {
	e.ticker = e.eclock.NewTicker(e.keepAliveInterval)
	for {
		select {
		case <-e.doneChan:
			e.logger.Info("stopped")
			return
		case envelope := <-e.envelopChan:
			err := e.Emit(envelope)
			if err != nil {
				e.logger.Error("failed-to-emit-envelope", err, lager.Data{"message": envelope})
			}
		case <-e.ticker.C():
			e.wsHelper.Ping()
		}
	}
}

func (e *EnvelopeEmitter) Stop() {
	e.wsHelper.CloseConn()
	e.doneChan <- true

}

func (e *EnvelopeEmitter) Accept(envelope *loggregator_v2.Envelope) {
	e.logger.Debug("accept-envelope", lager.Data{"envelope": envelope})
	e.envelopChan <- envelope
}
func (e *EnvelopeEmitter) Emit(envelope *loggregator_v2.Envelope) error {
	e.logger.Debug("emit-envelope", lager.Data{"envelope": envelope})
	err := e.wsHelper.Write(envelope)
	return err
}
