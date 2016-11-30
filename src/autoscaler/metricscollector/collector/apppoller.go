package collector

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/metricscollector/noaa"
	"autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry/sonde-go/events"

	"time"
)

type AppPoller interface {
	Start()
	Stop()
}

type appPoller struct {
	appId        string
	pollInterval time.Duration
	logger       lager.Logger
	cfc          cf.CfClient
	noaaConsumer noaa.NoaaConsumer
	database     db.InstanceMetricsDB
	pclock       clock.Clock
	doneChan     chan bool
}

func NewAppPoller(logger lager.Logger, appId string, pollInterval time.Duration, cfc cf.CfClient, noaaConsumer noaa.NoaaConsumer, database db.InstanceMetricsDB, pclock clock.Clock) AppPoller {
	return &appPoller{
		appId:        appId,
		pollInterval: pollInterval,
		logger:       logger,
		cfc:          cfc,
		noaaConsumer: noaaConsumer,
		database:     database,
		pclock:       pclock,
		doneChan:     make(chan bool),
	}

}

func (ap *appPoller) Start() {
	go ap.startPollMetrics()

	ap.logger.Info("app-poller-started", lager.Data{"appid": ap.appId, "poll-interval": ap.pollInterval})
}

func (ap *appPoller) Stop() {
	ap.doneChan <- true
	ap.logger.Info("app-poller-stopped", lager.Data{"appid": ap.appId})
}

func (ap *appPoller) startPollMetrics() {
	for {
		ap.pollMetric()
		timer := ap.pclock.NewTimer(ap.pollInterval)
		select {
		case <-ap.doneChan:
			return
		case <-timer.C():
		}
	}
}

func (ap *appPoller) pollMetric() {
	ap.logger.Debug("poll-metric", lager.Data{"appid": ap.appId})

	var containerEnvelopes []*events.Envelope
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		ap.logger.Debug("poll-metric-from-noaa-retry", lager.Data{"attempt": attempt + 1, "appid": ap.appId})

		containerEnvelopes, err = ap.noaaConsumer.ContainerEnvelopes(ap.appId, "bearer"+" "+ap.cfc.GetTokens().AccessToken)

		if err == nil {
			break
		}
	}

	if err != nil {
		ap.logger.Error("poll-metric-from-noaa", err, lager.Data{"appid": ap.appId})
		return
	}

	metrics := models.GetInstanceMemoryMetricFromContainerEnvelopes(ap.pclock.Now().UnixNano(), ap.appId, containerEnvelopes)
	ap.logger.Debug("poll-metric-get-memory-metric", lager.Data{"metrics": metrics})

	for _, metric := range metrics {
		err = ap.database.SaveMetric(metric)
		if err != nil {
			ap.logger.Error("poll-metric-save", err, lager.Data{"metric": metric})
		}
	}
}
