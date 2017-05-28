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

type AppStreamer interface {
	Start()
	Stop()
}

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

func NewAppStreamer(logger lager.Logger, appId string, interval time.Duration, cfc cf.CfClient, noaaConsumer noaa.NoaaConsumer, database db.InstanceMetricsDB, sclock clock.Clock) AppStreamer {
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
	eventChan, errorChan := as.noaaConsumer.Stream(as.appId, as.cfc.GetTokens().AccessToken)
	as.ticker = as.sclock.NewTicker(as.collectInterval)
	for {
		select {
		case <-as.doneChan:
			as.ticker.Stop()
			err := as.noaaConsumer.Close()
			if err == nil {
				as.logger.Info("noaa-connections-closed", lager.Data{"appid": as.appId})
			} else {
				as.logger.Error("close-noaa-connections", err, lager.Data{"appid": as.appId})
			}
			as.logger.Info("app-streamer-stopped", lager.Data{"appid": as.appId})
			return

		case err := <-errorChan:
			as.logger.Error("stream-metrics", err, lager.Data{"appid": as.appId})

		case event := <-eventChan:
			as.processEvent(event)

		case <-as.ticker.C():
			as.computeAndSaveMetrics()
		}
	}
}

func (as *appStreamer) processEvent(event *events.Envelope) {
	if event.GetEventType() == events.Envelope_ContainerMetric {
		metric := noaa.GetInstanceMemoryMetricFromContainerMetricEvent(as.sclock.Now().UnixNano(), as.appId, event)
		as.logger.Debug("process-event-get-memory-metric", lager.Data{"metric": metric})
		if metric != nil {
			err := as.database.SaveMetric(metric)
			if err != nil {
				as.logger.Error("process-event-save-metric", err, lager.Data{"metric": metric})
			}
		}
	} else if event.GetEventType() == events.Envelope_HttpStartStop {
		ss := event.GetHttpStartStop()
		if ss != nil {
			as.numRequests[ss.GetInstanceIndex()]++
			as.sumReponseTimes[ss.GetInstanceIndex()] += (ss.GetStopTimestamp() - ss.GetStartTimestamp())
		}
	}
}

func (as *appStreamer) computeAndSaveMetrics() {
	for instanceIdx, numReq := range as.numRequests {
		if numReq != 0 {
			througput := &models.AppInstanceMetric{
				AppId:         as.appId,
				InstanceIndex: uint32(instanceIdx),
				CollectedAt:   as.sclock.Now().UnixNano(),
				Name:          models.MetricNameThroughput,
				Unit:          models.UnitRPS,
				Value:         fmt.Sprintf("%d", int(float64(numReq)/as.collectInterval.Seconds()+0.5)),
				Timestamp:     as.sclock.Now().UnixNano(),
			}
			as.logger.Debug("compute-throughput", lager.Data{"throughput": througput})

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

			err := as.database.SaveMetric(througput)
			if err != nil {
				as.logger.Error("save-metric-to-database", err, lager.Data{"throughput": througput})
			}

			err = as.database.SaveMetric(responseTime)
			if err != nil {
				as.logger.Error("save-metric-to-database", err, lager.Data{"responsetime": responseTime})
			}

		}
	}

	as.numRequests = make(map[int32]int64)
	as.sumReponseTimes = make(map[int32]int64)
}
