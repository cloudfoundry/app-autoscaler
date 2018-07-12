package collector

import (
	"autoscaler/cf"
	"autoscaler/collection"
	"autoscaler/metricscollector/noaa"
	"autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"

	"fmt"
	"time"
)

type appStreamer struct {
	appId                    string
	logger                   lager.Logger
	collectInterval          time.Duration
	cfc                      cf.CfClient
	noaaConsumerForStreaming noaa.NoaaConsumer
	noaaConsumerForFirehose  noaa.NoaaConsumer
	doneChan                 chan bool
	sclock                   clock.Clock
	numRequests              map[int32]int64
	sumReponseTimes          map[int32]int64
	streamTicker             clock.Ticker
	firehoseTicker           clock.Ticker
	dataChan                 chan *models.AppInstanceMetric
}

const autoscalerFirehoseId = "autoscalerFirhoseId"
const autoscalerEventOrigin = "autoscaler_metrics_forwarder"

func NewAppStreamer(logger lager.Logger, appId string, interval time.Duration, cfc cf.CfClient, noaaConsumerForStreaming noaa.NoaaConsumer, noaaConsumerForFirehose noaa.NoaaConsumer, sclock clock.Clock, dataChan chan *models.AppInstanceMetric) AppCollector {
	return &appStreamer{
		appId:                    appId,
		logger:                   logger,
		collectInterval:          interval,
		cache:                    collection.NewTSDCache(cacheSize),
		cfc:                      cfc,
		noaaConsumerForStreaming: noaaConsumerForStreaming,
		noaaConsumerForFirehose:  noaaConsumerForFirehose,
		doneChan:                 make(chan bool),
		sclock:                   sclock,
		numRequests:              make(map[int32]int64),
		sumReponseTimes:          make(map[int32]int64),
		dataChan:                 dataChan,
	}
}

func (as *appStreamer) Start() {
	go as.streamMetrics()
	as.logger.Info("app-streamer-started", lager.Data{"appid": as.appId})
	go as.readFirehose()
	as.logger.Info("app-firehose-started")
}

func (as *appStreamer) Stop() {
	as.doneChan <- true
}

func (as *appStreamer) streamMetrics() {
	eventChan, errorChan := as.noaaConsumerForStreaming.Stream(as.appId, cf.TokenTypeBearer+" "+as.cfc.GetTokens().AccessToken)
	as.streamTicker = as.sclock.NewTicker(as.collectInterval)
	var err error
	for {
		select {
		case <-as.doneChan:
			as.streamTicker.Stop()
			err := as.noaaConsumerForStreaming.Close()
			if err == nil {
				as.logger.Info("noaa-connection-closed", lager.Data{"appid": as.appId})
			} else {
				as.logger.Error("close-noaa-connection", err, lager.Data{"appid": as.appId})
			}
			as.logger.Info("app-streamer-stopped", lager.Data{"appid": as.appId})
			return

		case err = <-errorChan:
			as.logger.Error("error-stream-metrics", err, lager.Data{"appid": as.appId})

		case event := <-eventChan:
			as.processEvent(event)

		case <-as.streamTicker.C():
			if err != nil {
				closeErr := as.noaaConsumerForStreaming.Close()
				if closeErr != nil {
					as.logger.Error("close-noaa-connection", err, lager.Data{"appid": as.appId})
				}
				eventChan, errorChan = as.noaaConsumerForStreaming.Stream(as.appId, cf.TokenTypeBearer+" "+as.cfc.GetTokens().AccessToken)
				as.logger.Info("noaa-reconnected", lager.Data{"appid": as.appId})
				err = nil
			} else {
				as.computeAndSaveMetrics()
			}
		}
	}
}

func (as *appStreamer) readFirehose() {
	var err error
	eventChan, errorChan := as.noaaConsumerForFirehose.FilteredFirehose(autoscalerFirehoseId, "", consumer.Metrics)
	as.firehoseTicker = as.sclock.NewTicker(as.collectInterval)
	for {
		select {
		case <-as.doneChan:
			as.firehoseTicker.Stop()
			err := as.noaaConsumerForFirehose.Close()
			if err == nil {
				as.logger.Info("noaa-connection-closed", lager.Data{"appid": as.appId})
			} else {
				as.logger.Error("close-noaa-connection", err, lager.Data{"appid": as.appId})
			}
			as.logger.Info("app-streamer-stopped", lager.Data{"appid": as.appId})
			return

		case err = <-errorChan:
			as.logger.Error("firehose-metrics-error", err, lager.Data{"appid": as.appId})

		case event := <-eventChan:
			if event.GetEventType() == events.Envelope_ValueMetric {
				as.processEvent(event)
			}

		case <-as.firehoseTicker.C():
			if err != nil {
				closeErr := as.noaaConsumerForFirehose.Close()
				if closeErr != nil {
					as.logger.Error("close-noaa-connection", err, lager.Data{"appid": as.appId})
				}
				eventChan, errorChan = as.noaaConsumerForFirehose.FilteredFirehose(autoscalerFirehoseId, "", consumer.Metrics)
				as.logger.Info("noaa-reconnected", lager.Data{"appid": as.appId})
				err = nil
			}
		}
	}
}

func (as *appStreamer) processEvent(event *events.Envelope) {
	if event.GetEventType() == events.Envelope_ContainerMetric {
		as.logger.Debug("process-event-get-containermetric-event", lager.Data{"event": event})
		metrics := noaa.GetMetricsFromContainerEnvelope(as.sclock.Now().UnixNano(), as.appId, event)
		for _, metric := range metrics {
			as.cache.Put(metric)
			as.dataChan <- metric
		}
	} else if event.GetEventType() == events.Envelope_HttpStartStop {
		as.logger.Debug("process-event-get-httpstartstop-event", lager.Data{"event": event})
		ss := event.GetHttpStartStop()
		if ss != nil {
			as.numRequests[ss.GetInstanceIndex()]++
			as.sumReponseTimes[ss.GetInstanceIndex()] += (ss.GetStopTimestamp() - ss.GetStartTimestamp())
		}
	} else if event.GetEventType() == events.Envelope_ValueMetric {
		as.logger.Debug("process-event-get-value-metric-event", lager.Data{"event": event})
		ss := event.GetValueMetric()
		if ss != nil && event.GetOrigin() == autoscalerEventOrigin {
			valuemetric := noaa.GetCustomMetricFromValueMetricEvent(as.sclock.Now().UnixNano(), event)
			as.logger.Debug("process-event-get-custom-value-metric", lager.Data{"metric": valuemetric})
			if valuemetric != nil {
				as.dataChan <- valuemetric
			}
		}
	}
}

func (as *appStreamer) computeAndSaveMetrics() {
	as.logger.Debug("compute-and-save-metrics", lager.Data{"message": "start to compute and save metrics"})
	if len(as.numRequests) == 0 {
		throughput := &models.AppInstanceMetric{
			AppId:         as.appId,
			InstanceIndex: 0,
			CollectedAt:   as.sclock.Now().UnixNano(),
			Name:          models.MetricNameThroughput,
			Unit:          models.UnitRPS,
			Value:         "0",
			Timestamp:     as.sclock.Now().UnixNano(),
		}
		as.logger.Debug("compute-throughput", lager.Data{"message": "write 0 throughput due to no requests"})
		as.cache.Put(throughput)
		as.dataChan <- throughput
		return
	}

	for instanceIdx, numReq := range as.numRequests {
		throughput := &models.AppInstanceMetric{
			AppId:         as.appId,
			InstanceIndex: uint32(instanceIdx),
			CollectedAt:   as.sclock.Now().UnixNano(),
			Name:          models.MetricNameThroughput,
			Unit:          models.UnitRPS,
			Value:         fmt.Sprintf("%d", int(float64(numReq)/as.collectInterval.Seconds()+0.5)),
			Timestamp:     as.sclock.Now().UnixNano(),
		}
		as.logger.Debug("compute-throughput", lager.Data{"throughput": throughput})
		as.cache.Put(throughput)
		as.dataChan <- throughput

		responseTime := &models.AppInstanceMetric{
			AppId:         as.appId,
			InstanceIndex: uint32(instanceIdx),
			CollectedAt:   as.sclock.Now().UnixNano(),
			Name:          models.MetricNameResponseTime,
			Unit:          models.UnitMilliseconds,
			Value:         fmt.Sprintf("%d", as.sumReponseTimes[instanceIdx]/(numReq*1000*1000)),
			Timestamp:     as.sclock.Now().UnixNano(),
		}
		as.logger.Debug("compute-responsetime", lager.Data{"responsetime": responseTime})
		as.cache.Put(responseTime)
		as.dataChan <- responseTime
	}

	as.numRequests = make(map[int32]int64)
	as.sumReponseTimes = make(map[int32]int64)
}

func (as *appStreamer) Query(start, end int64, labels map[string]string) ([]collection.TSD, bool) {
	return as.cache.Query(start, end, labels)
}
