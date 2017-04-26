package collector

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/metricscollector/noaa"
	"autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"
)

type AppStreamer interface {
	Start()
	Stop()
}

type appStreamer struct {
	appId        string
	logger       lager.Logger
	cfc          cf.CfClient
	noaaConsumer noaa.NoaaConsumer
	database     db.InstanceMetricsDB
	doneChan     chan bool
	sclock       clock.Clock
}

func NewAppStreamer(logger lager.Logger, appId string, cfc cf.CfClient, noaaConsumer noaa.NoaaConsumer, database db.InstanceMetricsDB, sclock clock.Clock) AppStreamer {
	return &appStreamer{
		appId:        appId,
		logger:       logger,
		cfc:          cfc,
		noaaConsumer: noaaConsumer,
		database:     database,
		doneChan:     make(chan bool),
		sclock:       sclock,
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
	for {
		select {
		case <-as.doneChan:
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
		}
	}
}

func (as *appStreamer) processEvent(event *events.Envelope) {
	if event.GetEventType() == events.Envelope_ContainerMetric {
		metric := models.GetInstanceMemoryMetricFromContainerMetricEvent(as.sclock.Now().UnixNano(), as.appId, event)
		as.logger.Debug("process-event-get-memory-metric", lager.Data{"metric": metric})
		if metric != nil {
			err := as.database.SaveMetric(metric)
			if err != nil {
				as.logger.Error("process-event-save-metric", err, lager.Data{"metric": metric})
			}
		}
	}
}
