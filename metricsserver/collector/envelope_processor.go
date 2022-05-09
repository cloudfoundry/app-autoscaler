package collector

import (
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
	HttpStartStopEnvelopes map[string][]*loggregator_v2.Envelope // appID map of envelopes
	envelopeChan           <-chan *loggregator_v2.Envelope
	metricChan             chan<- *models.AppInstanceMetric
	getAppIDs              func() map[string]bool
}

func NewEnvelopeProcessor(logger lager.Logger, collectInterval time.Duration, clock clock.Clock, processsorIndex, numProcesssors int,
	envelopeChan <-chan *loggregator_v2.Envelope, metricChan chan<- *models.AppInstanceMetric, getAppIDs func() map[string]bool) *envelopeProcessor {
	return &envelopeProcessor{
		logger:          logger,
		collectInterval: collectInterval,
		doneChan:        make(chan bool),
		clock:           clock,
		processorIndex:  processsorIndex,
		numProcessors:   numProcesssors,
		envelopeChan:    envelopeChan,
		metricChan:      metricChan,
		getAppIDs:       getAppIDs,
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
			for _, metric := range metrics {
				ep.metricChan <- metric
			}

		case <-ticker.C():
			// replace processHttpStartStopMetrics with:
			//metrics := envelopeprocessor.ComputeHttpStartStopFrom(ep.HttpStartStopEnvelopes)
			//for _, metric := range metrics {
			//	ep.metricChan <- metric
			//}
			ep.processHttpStartStopMetrics()
		}
	}
}

func (ep *envelopeProcessor) IsCacheEmpty() bool {
	return ep.HttpStartStopEnvelopes == nil
}

func (ep *envelopeProcessor) getAppInstanceMetrics(e *loggregator_v2.Envelope) []*models.AppInstanceMetric {
	switch e.GetMessage().(type) {
	case *loggregator_v2.Envelope_Gauge:
		return envelopeprocessor.GetGaugeInstanceMetrics(e, ep.clock.Now().UnixNano())
	case *loggregator_v2.Envelope_Timer:
		ep.cacheHttpStartStopEnvelop(e)
		return []*models.AppInstanceMetric{}
	default:
		return []*models.AppInstanceMetric{}
	}
}
func (ep *envelopeProcessor) isMetrissrvRespForApp(appID string) bool {
	return helpers.FNVHash(appID)%uint32(ep.numProcessors) == uint32(ep.processorIndex)
}

func (ep *envelopeProcessor) processHttpStartStopMetrics() {
	ep.logger.Debug("compute-and-save-metrics", lager.Data{"message": "start to compute and save metrics"})
	for appID := range ep.getAppIDs() {
		if !ep.isMetrissrvRespForApp(appID) { // skip apps we are not responsible for
			continue
		}

		metrics := envelopeprocessor.ComputeHttpStartStop(ep.HttpStartStopEnvelopes[appID], appID, ep.clock.Now().UnixNano(),
			ep.collectInterval)
		for _, metric := range metrics {
			ep.metricChan <- metric
		}
	}

	ep.HttpStartStopEnvelopes = nil
}

func (ep *envelopeProcessor) cacheHttpStartStopEnvelop(e *loggregator_v2.Envelope) {
	if ep.HttpStartStopEnvelopes == nil {
		ep.HttpStartStopEnvelopes = map[string][]*loggregator_v2.Envelope{}
	}
	if ep.HttpStartStopEnvelopes[e.SourceId] == nil {
		ep.HttpStartStopEnvelopes[e.SourceId] = []*loggregator_v2.Envelope{}
	}

	ep.HttpStartStopEnvelopes[e.SourceId] = append(ep.HttpStartStopEnvelopes[e.SourceId], e)
}
