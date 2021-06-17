package collector

import (
	"autoscaler/helpers"
	"autoscaler/models"
	"fmt"
	"strconv"
	"time"
	"math"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
)

type EnvelopeProcessor interface {
	Start()
	Stop()
}

type envelopeProcessor struct {
	logger          lager.Logger
	collectInterval time.Duration
	doneChan        chan bool
	clock           clock.Clock
	numRequests     map[string]map[uint32]int64
	sumReponseTimes map[string]map[uint32]int64
	processorIndex  int
	numProcessors   int
	envelopeChan    <-chan *loggregator_v2.Envelope
	metricChan      chan<- *models.AppInstanceMetric
	getAppIDs       func() map[string]bool
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
			ep.processEnvelope(e)

		case <-ticker.C():
			ep.computeAndSaveMetrics()
		}
	}
}

func (ep *envelopeProcessor) processEnvelope(e *loggregator_v2.Envelope) {
	instanceIndex, _ := strconv.ParseInt(e.InstanceId, 10, 32)
	switch e.GetMessage().(type) {
	case *loggregator_v2.Envelope_Gauge:
		ep.logger.Debug("process-envelope", lager.Data{"index": ep.processorIndex, "appID": e.SourceId, "message": e.Message})
		g := e.GetGauge()
		_, exist := g.GetMetrics()["memory_quota"]
		if exist {
			ep.processContainerMetrics(e.SourceId, uint32(instanceIndex), e.Timestamp, g)
		} else {
			ep.processCustomMetrics(e.SourceId, uint32(instanceIndex), e.Timestamp, g)
		}
	case *loggregator_v2.Envelope_Timer:
		ep.logger.Debug("filter-envelopes-get-httpstartstop", lager.Data{"index": ep.processorIndex, "appID": e.SourceId, "message": e.Message})
		t := e.GetTimer()
		ep.processHttpStartStop(e.SourceId, uint32(instanceIndex), t)
	}

}

func (ep *envelopeProcessor) processContainerMetrics(appID string, instanceIndex uint32, timestamp int64, g *loggregator_v2.Gauge) {
	memory, exist := g.GetMetrics()["memory"]
	if exist {
		memoryUsedMetric := &models.AppInstanceMetric{
			AppId:         appID,
			InstanceIndex: instanceIndex,
			CollectedAt:   ep.clock.Now().UnixNano(),
			Name:          models.MetricNameMemoryUsed,
			Unit:          models.UnitMegaBytes,
			Value:         fmt.Sprintf("%d", int(math.Ceil(memory.GetValue()/(1024*1024)))),
			Timestamp:     timestamp,
		}
		ep.metricChan <- memoryUsedMetric

		memoryQuota, exist := g.GetMetrics()["memory_quota"]
		if exist && memoryQuota.GetValue() != 0 {
			memoryUtilMetric := &models.AppInstanceMetric{
				AppId:         appID,
				InstanceIndex: instanceIndex,
				CollectedAt:   ep.clock.Now().UnixNano(),
				Name:          models.MetricNameMemoryUtil,
				Unit:          models.UnitPercentage,
				Value:         fmt.Sprintf("%d", int(math.Ceil(memory.GetValue()/memoryQuota.GetValue()*100))),
				Timestamp:     timestamp,
			}
			ep.metricChan <- memoryUtilMetric
		}
	}

	cpu, exist := g.GetMetrics()["cpu"]
	if exist {
		cpuMetric := &models.AppInstanceMetric{
			AppId:         appID,
			InstanceIndex: instanceIndex,
			CollectedAt:   ep.clock.Now().UnixNano(),
			Name:          models.MetricNameCPUUtil,
			Unit:          models.UnitPercentage,
			Value:         fmt.Sprintf("%d", int64(math.Ceil(cpu.GetValue()))),
			Timestamp:     timestamp,
		}
		ep.metricChan <- cpuMetric
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
	ep.sumReponseTimes[appID][instanceIndex] += (t.Stop - t.Start)
}

func (ep *envelopeProcessor) processCustomMetrics(appID string, instanceIndex uint32, timestamp int64, g *loggregator_v2.Gauge) {
	for n, v := range g.GetMetrics() {
		customMetric := &models.AppInstanceMetric{
			AppId:         appID,
			InstanceIndex: instanceIndex,
			CollectedAt:   ep.clock.Now().UnixNano(),
			Name:          n,
			Unit:          v.Unit,
			Value:         fmt.Sprintf("%d", int64(math.Ceil(v.Value))),
			Timestamp:     timestamp,
		}
		ep.metricChan <- customMetric
	}
}

func (ep *envelopeProcessor) computeAndSaveMetrics() {
	ep.logger.Debug("compute-and-save-metrics", lager.Data{"message": "start to compute and save metrics"})
	for appID := range ep.getAppIDs() {
		im := ep.numRequests[appID]
		if im == nil || len(im) == 0 {
			if helpers.FNVHash(appID)%uint32(ep.numProcessors) == uint32(ep.processorIndex) {
				throughputMetric := &models.AppInstanceMetric{
					AppId:         appID,
					InstanceIndex: 0,
					CollectedAt:   ep.clock.Now().UnixNano(),
					Name:          models.MetricNameThroughput,
					Unit:          models.UnitRPS,
					Value:         "0",
					Timestamp:     ep.clock.Now().UnixNano(),
				}

				responseTimeMetric := &models.AppInstanceMetric{
					AppId:         appID,
					InstanceIndex: 0,
					CollectedAt:   ep.clock.Now().UnixNano(),
					Name:          models.MetricNameResponseTime,
					Unit:          models.UnitMilliseconds,
					Value:         "0",
					Timestamp:     ep.clock.Now().UnixNano(),
				}

				ep.metricChan <- throughputMetric
				ep.metricChan <- responseTimeMetric
			}
			continue
		}
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
				Value:         fmt.Sprintf("%d", int64(math.Ceil(float64(ep.sumReponseTimes[appID][instanceIdx])/float64((numReq*1000*1000))))),
				Timestamp:     ep.clock.Now().UnixNano(),
			}
			ep.metricChan <- responseTimeMetric
		}
	}
	ep.numRequests = map[string]map[uint32]int64{}
	ep.sumReponseTimes = map[string]map[uint32]int64{}

}

