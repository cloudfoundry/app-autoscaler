package metricsgateway

import (
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
)

type Emitter interface {
	Accept(envelope *loggregator_v2.Envelope)
	Emit(*loggregator_v2.Envelope) error
}

type emitter struct {
	logger               lager.Logger
	metricsServerAddress string
	envelopChan          chan *loggregator_v2.Envelope
	bufferSize           int64
	doneChan             chan bool
}

func NewEmitter(logger lager.Logger, bufferSize int64, metricsServerAddress string) Emitter {
	return &emitter{
		logger:               logger.Session("Emitter"),
		metricsServerAddress: metricsServerAddress,
		envelopChan:          make(chan *loggregator_v2.Envelope, bufferSize),
	}
}
func (e *emitter) Start() {
	go e.startEmitEnvelope()
	e.logger.Info("started")
}

func (e *emitter) startEmitEnvelope() {
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
		}
	}
}

func (e *emitter) Stop() {
	e.doneChan <- true

}

func (e *emitter) Accept(envelope *loggregator_v2.Envelope) {
	e.envelopChan <- envelope
}
func (e *emitter) Emit(*loggregator_v2.Envelope) error {
	// to be done
	return nil
}
