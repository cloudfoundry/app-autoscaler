package collector

import (
	"fmt"
	"math"
	"strconv"
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
}

type envelopeProcessor struct {
	logger                 lager.Logger
	collectInterval        time.Duration
	doneChan               chan bool
	clock                  clock.Clock
	numRequests            map[string]map[uint32]int64 // to be depreacted
	sumReponseTimes        map[string]map[uint32]int64 // to be depreacted
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
		numRequests:     map[string]map[uint32]int64{},
		sumReponseTimes: map[string]map[uint32]int64{},
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
			// replace computeAndSaveMetrics with:
			//metrics := envelopeprocessor.ComputeHttpStartStopFrom(ep.HttpStartStopEnvelopes)
			//for _, metric := range metrics {
			//	ep.metricChan <- metric
			//}
			ep.computeAndSaveMetrics()
		}
	}
}

func (ep *envelopeProcessor) getAppInstanceMetrics(e *loggregator_v2.Envelope) []*models.AppInstanceMetric {
	instanceIndex, _ := strconv.ParseInt(e.InstanceId, 10, 32)
	switch e.GetMessage().(type) {
	case *loggregator_v2.Envelope_Gauge:
		return envelopeprocessor.GetGaugeInstanceMetrics(e, ep.clock.Now().UnixNano())
	case *loggregator_v2.Envelope_Timer:
		ep.cacheHttpStartStopEnvelop(e)
		ep.processHttpStartStop(e.SourceId, uint32(instanceIndex), e.GetTimer())
		return []*models.AppInstanceMetric{}
	default:
		return []*models.AppInstanceMetric{}
	}
}

func (ep *envelopeProcessor) processHttpStartStop(appID string, instanceIndex uint32, t *loggregator_v2.Timer) {
	if ep.numRequests[appID] == nil {
		ep.numRequests[appID] = map[uint32]int64{}
	}
	if ep.sumReponseTimes[appID] == nil {
		ep.sumReponseTimes[appID] = map[uint32]int64{}
	}

	ep.numRequests[appID][instanceIndex]++
	ep.sumReponseTimes[appID][instanceIndex] += t.Stop - t.Start
}

func (ep *envelopeProcessor) computeAndSaveMetrics() {
	ep.logger.Debug("compute-and-save-metrics", lager.Data{"message": "start to compute and save metrics"})
	for appID := range ep.getAppIDs() {
		im := ep.numRequests[appID]
		if len(im) == 0 {
			if helpers.FNVHash(appID)%uint32(ep.numProcessors) == uint32(ep.processorIndex) {
				metrics := envelopeprocessor.ComputeHttpStartStop(ep.HttpStartStopEnvelopes[appID], appID, ep.clock.Now().UnixNano())
				for _, metric := range metrics {
					ep.metricChan <- metric
				}
			}
			continue
		}

		//metrics := envelopeprocessor.ComputeHttpStartStop(ep.HttpStartStopEnvelopes[appID], appID, ep.clock.Now().UnixNano())
		//for _, metric := range metrics {
		//	ep.metricChan <- metric
		//}
		for instanceIdx, numReq := range im {
			throughputMetric := &models.AppInstanceMetric{
				AppId:         appID,
				InstanceIndex: instanceIdx,
				CollectedAt:   ep.clock.Now().UnixNano(),
				Name:          models.MetricNameThroughput,
				Unit:          models.UnitRPS,
				Value:         fmt.Sprintf("%d", int(math.Ceil(float64(numReq)/ep.collectInterval.Seconds()))),
				Timestamp:     ep.clock.Now().UnixNano(),
			}
			ep.metricChan <- throughputMetric

			responseTimeMetric := &models.AppInstanceMetric{
				AppId:         appID,
				InstanceIndex: instanceIdx,
				CollectedAt:   ep.clock.Now().UnixNano(),
				Name:          models.MetricNameResponseTime,
				Unit:          models.UnitMilliseconds,
				Value:         fmt.Sprintf("%d", int64(math.Ceil(float64(ep.sumReponseTimes[appID][instanceIdx])/float64(numReq*1000*1000)))),
				Timestamp:     ep.clock.Now().UnixNano(),
			}
			ep.metricChan <- responseTimeMetric
		}
	}
	// clean ep.HttpStartStopEnvelopes
	ep.numRequests = map[string]map[uint32]int64{}
	ep.sumReponseTimes = map[string]map[uint32]int64{}
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
