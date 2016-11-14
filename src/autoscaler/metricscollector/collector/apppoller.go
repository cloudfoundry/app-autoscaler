package collector

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/metricscollector/noaa"
	"autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

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
	timer        clock.Timer
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
	ap.timer = ap.pclock.NewTimer(ap.pollInterval)
	go ap.startPollMetrics()

	ap.logger.Info("app-poller-started", lager.Data{"appid": ap.appId, "poll-interval": ap.pollInterval})
}

func (ap *appPoller) Stop() {
	if ap.timer != nil {
		ap.timer.Stop()
		close(ap.doneChan)
	}
	ap.logger.Info("app-poller-stopped", lager.Data{"appid": ap.appId})
}

func (ap *appPoller) startPollMetrics() {
	for {
		ap.pollMetric()
		ap.timer.Reset(ap.pollInterval)
		select {
		case <-ap.doneChan:
			return
		case <-ap.timer.C():
		}
	}
}

func (ap *appPoller) pollMetric() {
	ap.logger.Debug("poll-metric", lager.Data{"appid": ap.appId})

	containerEnvelopes, err := ap.noaaConsumer.ContainerEnvelopes(ap.appId, "bearer"+" "+ap.cfc.GetTokens().AccessToken)
	if err != nil {
		ap.logger.Error("poll-metric-from-noaa", err)
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
