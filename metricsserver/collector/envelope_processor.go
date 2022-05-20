package collector

import (
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock"
	loggregator_v2 "code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
)

type EnvelopeProcessor interface {
	Start()
	Stop()
	IsCacheEmpty() bool
}

type envelopeProcessor struct {
	logger                 lager.Logger
	collectInterval        time.Duration
	doneChan               chan bool
	clock                  clock.Clock
	processorIndex         int
	numProcessors          int
	httpStartStopEnvelopes map[string][]*loggregator_v2.Envelope // appID map of envelopes
	envelopeChan           <-chan *loggregator_v2.Envelope
	metricChan             chan<- *models.AppInstanceMetric
	getAppIDs              func() map[string]bool
	mu                     sync.RWMutex
}

func NewEnvelopeProcessor(logger lager.Logger, collectInterval time.Duration, clock clock.Clock, processsorIndex, numProcesssors int,
	envelopeChan <-chan *loggregator_v2.Envelope, metricChan chan<- *models.AppInstanceMetric, getAppIDs func() map[string]bool) *envelopeProcessor {
	return &envelopeProcessor{
		logger:                 logger,
		collectInterval:        collectInterval,
		doneChan:               make(chan bool),
		clock:                  clock,
		httpStartStopEnvelopes: map[string][]*loggregator_v2.Envelope{},
		processorIndex:         processsorIndex,
		numProcessors:          numProcesssors,
		envelopeChan:           envelopeChan,
		metricChan:             metricChan,
		getAppIDs:              getAppIDs,
	}
}

func (ep *envelopeProcessor) Start() {
	go ep.processEvents()
	ep.logger.Info("envelop-processor-started", lager.Data{"processor-index": ep.processorIndex, "processor-num": ep.numProcessors})
}

func (ep *envelopeProcessor) Stop() {
	ep.doneChan <- true
}

func (ep *envelopeProcessor) processEvents() {
	ticker := ep.clock.NewTicker(ep.collectInterval)
	for {
		select {
		case <-ep.doneChan:
			ticker.Stop()
			return

		case e := <-ep.envelopeChan:
			metrics := ep.getAppInstanceMetrics(e)
			for i, _ := range metrics {
				ep.metricChan <- &metrics[i]
			}

		case <-ticker.C():
			ep.processHttpStartStopMetrics()
		}
	}
}

func (ep *envelopeProcessor) IsCacheEmpty() bool {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	return len(ep.httpStartStopEnvelopes) == 0
}

func (ep *envelopeProcessor) getAppInstanceMetrics(e *loggregator_v2.Envelope) []models.AppInstanceMetric {
	switch e.GetMessage().(type) {
	case *loggregator_v2.Envelope_Gauge:
		appInstanceMetrics, _ := envelopeprocessor.GetGaugeInstanceMetrics([]*loggregator_v2.Envelope{e}, ep.clock.Now().UnixNano())
		return appInstanceMetrics
	case *loggregator_v2.Envelope_Timer:
		ep.cacheHttpStartStopEnvelop(e)
		return []models.AppInstanceMetric{}
	default:
		return []models.AppInstanceMetric{}
	}
}
func (ep *envelopeProcessor) isMetrissrvRespForApp(appID string) bool {
	return helpers.FNVHash(appID)%uint32(ep.numProcessors) == uint32(ep.processorIndex)
}

func (ep *envelopeProcessor) processHttpStartStopMetrics() {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.logger.Debug("compute-and-save-metrics", lager.Data{"message": "start to compute and save metrics"})

	for appID := range ep.getAppIDs() {
		if !ep.isMetrissrvRespForApp(appID) { // skip apps we are not responsible for
			continue
		}

		metrics := envelopeprocessor.GetHttpStartStopInstanceMetrics(ep.httpStartStopEnvelopes[appID], appID, ep.clock.Now().UnixNano(),
			ep.collectInterval)
		for i, _ := range metrics {
			ep.metricChan <- &metrics[i]
		}
	}

	ep.httpStartStopEnvelopes = map[string][]*loggregator_v2.Envelope{}
}

func (ep *envelopeProcessor) cacheHttpStartStopEnvelop(e *loggregator_v2.Envelope) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.httpStartStopEnvelopes[e.SourceId] = append(ep.httpStartStopEnvelopes[e.SourceId], e)
}
