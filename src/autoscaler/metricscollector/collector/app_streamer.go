package collector

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/metricscollector/noaa"
	"autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"

	"fmt"
	"time"
)

type appStreamer struct {
	appId           string
	logger          lager.Logger
	collectInterval time.Duration
	cfc             cf.CfClient
	noaaConsumer    noaa.NoaaConsumer
	database        db.InstanceMetricsDB
	doneChan        chan bool
	sclock          clock.Clock
	numRequests     map[int32]int64
	sumReponseTimes map[int32]int64
	ticker          clock.Ticker
}

func NewAppStreamer(logger lager.Logger, appId string, interval time.Duration, cfc cf.CfClient, noaaConsumer noaa.NoaaConsumer, database db.InstanceMetricsDB, sclock clock.Clock) AppCollector {
	return &appStreamer{
		appId:           appId,
		logger:          logger,
		collectInterval: interval,
		cfc:             cfc,
		noaaConsumer:    noaaConsumer,
		database:        database,
		doneChan:        make(chan bool),
		sclock:          sclock,
		numRequests:     make(map[int32]int64),
		sumReponseTimes: make(map[int32]int64),
	}
}

func (as *appStreamer) Start() {
	go as.streamMetrics()
	as.logger.Info("app-streamer-started", lager.Data{"appid": as.appId})
}

func (as *appStreamer) Stop() {
	as.doneChan <- true
}

func (as *appStreamer) streamMetrics() {
	eventChan, errorChan := as.noaaConsumer.Stream(as.appId, cf.TokenTypeBearer+" "+as.cfc.GetTokens().AccessToken)
	as.ticker = as.sclock.NewTicker(as.collectInterval)
	var err error
	for {
		select {
		case <-as.doneChan:
			as.ticker.Stop()
			err := as.noaaConsumer.Close()
			if err == nil {
				as.logger.Info("noaa-connection-closed", lager.Data{"appid": as.appId})
			} else {
				as.logger.Error("close-noaa-connection", err, lager.Data{"appid": as.appId})
			}
			as.logger.Info("app-streamer-stopped", lager.Data{"appid": as.appId})
			return

		case err = <-errorChan:
			as.logger.Error("stream-metrics", err, lager.Data{"appid": as.appId})

		case event := <-eventChan:
			as.processEvent(event)

		case <-as.ticker.C():
			if err != nil {
				closeErr := as.noaaConsumer.Close()
				if closeErr != nil {
					as.logger.Error("close-noaa-connection", err, lager.Data{"appid": as.appId})
				}
				eventChan, errorChan = as.noaaConsumer.Stream(as.appId, cf.TokenTypeBearer+" "+as.cfc.GetTokens().AccessToken)
				as.logger.Info("noaa-reconnected", lager.Data{"appid": as.appId})
				err = nil
			} else {
				as.computeAndSaveMetrics()
			}
		}
	}
}

func (as *appStreamer) processEvent(event *events.Envelope) {
	if event.GetEventType() == events.Envelope_ContainerMetric {
		as.logger.Debug("process-event-get-containermetric-event", lager.Data{"event": event})

		metric := noaa.GetInstanceMemoryUsedMetricFromContainerMetricEvent(as.sclock.Now().UnixNano(), as.appId, event)
		as.logger.Debug("process-event-get-memoryused-metric", lager.Data{"metric": metric})
		if metric != nil {
			err := as.database.SaveMetric(metric)
			if err != nil {
				as.logger.Error("process-event-save-metric", err, lager.Data{"metric": metric})
			}
		}
		metric = noaa.GetInstanceMemoryUtilMetricFromContainerMetricEvent(as.sclock.Now().UnixNano(), as.appId, event)
		as.logger.Debug("process-event-get-memoryutil-metric", lager.Data{"metric": metric})
		if metric != nil {
			err := as.database.SaveMetric(metric)
			if err != nil {
				as.logger.Error("process-event-save-metric", err, lager.Data{"metric": metric})
			}
		}
		metric = noaa.GetInstanceCpuPercentageMetricFromContainerMetricEvent(as.sclock.Now().UnixNano(), as.appId, event)
		as.logger.Debug("process-event-get-cpupercentage-metric", lager.Data{"metric": metric})
		if metric != nil {
			err := as.database.SaveMetric(metric)
			if err != nil {
				as.logger.Error("process-event-save-metric", err, lager.Data{"metric": metric})
			}
		}
	} else if event.GetEventType() == events.Envelope_HttpStartStop {
		as.logger.Debug("process-event-get-httpstartstop-event", lager.Data{"event": event})
		ss := event.GetHttpStartStop()
		if ss != nil {
			as.numRequests[ss.GetInstanceIndex()]++
			as.sumReponseTimes[ss.GetInstanceIndex()] += (ss.GetStopTimestamp() - ss.GetStartTimestamp())
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
		err := as.database.SaveMetric(throughput)
		if err != nil {
			as.logger.Error("save-metric-to-database", err, lager.Data{"throughput": throughput})
		}
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

		err := as.database.SaveMetric(throughput)
		if err != nil {
			as.logger.Error("save-metric-to-database", err, lager.Data{"throughput": throughput})
		}

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

		err = as.database.SaveMetric(responseTime)
		if err != nil {
			as.logger.Error("save-metric-to-database", err, lager.Data{"responsetime": responseTime})
		}
	}

	as.numRequests = make(map[int32]int64)
	as.sumReponseTimes = make(map[int32]int64)
}
