package metricsgateway

import (
	"autoscaler/helpers"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
)

type Dispatcher struct {
	logger      lager.Logger
	envelopChan chan *loggregator_v2.Envelope
	doneChan    chan bool
	emitters    []Emitter
}

func NewDispatcher(logger lager.Logger, envelopChan chan *loggregator_v2.Envelope, emitters []Emitter) *Dispatcher {
	return &Dispatcher{
		logger:      logger.Session("Dispather"),
		envelopChan: envelopChan,
		emitters:    emitters,
		doneChan:    make(chan bool),
	}
}
func (d *Dispatcher) Start() {
	go d.dispatch()
	d.logger.Info("dispatcher-started")
}

func (d *Dispatcher) Stop() {
	d.doneChan <- true
}
func (d *Dispatcher) dispatch() {
	for {
		select {
		case <-d.doneChan:
			d.logger.Info("dispatcher-stopped")
			return
		case e := <-d.envelopChan:
			appID := e.SourceId
			emmitter := d.getEmitter(appID)
			emmitter.Accept(e)
		}
	}
}

func (d *Dispatcher) getEmitter(appID string) Emitter {
	return d.emitters[helpers.FNVHash(appID)%(uint32)(len(d.emitters))]
}
