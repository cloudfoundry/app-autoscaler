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
	database     db.MetricsDB
	pclock       clock.Clock
	ticker       clock.Ticker
	doneChan     chan bool
}

func NewAppPoller(logger lager.Logger, appId string, pollInterval time.Duration, cfc cf.CfClient, noaaConsumer noaa.NoaaConsumer, database db.MetricsDB, pclcok clock.Clock) AppPoller {
	return &appPoller{
		appId:        appId,
		pollInterval: pollInterval,
		logger:       logger,
		cfc:          cfc,
		noaaConsumer: noaaConsumer,
		database:     database,
		pclock:       pclcok,
		doneChan:     make(chan bool),
	}

}

func (ap *appPoller) Start() {
	ap.ticker = ap.pclock.NewTicker(ap.pollInterval)
	go ap.startPollMetrics()

	ap.logger.Info("app-poller-started", lager.Data{"appid": ap.appId, "poll-interval": ap.pollInterval})
}

func (ap *appPoller) Stop() {
	if ap.ticker != nil {
		ap.ticker.Stop()
		close(ap.doneChan)
	}
	ap.logger.Info("app-poller-stopped", lager.Data{"appid": ap.appId})
}

func (ap *appPoller) startPollMetrics() {
	for {
		ap.pollMetric()
		select {
		case <-ap.doneChan:
			return
		case <-ap.ticker.C():
		}
	}
}

func (ap *appPoller) pollMetric() {
	ap.logger.Debug("poll-metric", lager.Data{"appid": ap.appId})

	containerMetrics, err := ap.noaaConsumer.ContainerEnvelopes(ap.appId, "bearer"+" "+ap.cfc.GetTokens().AccessToken)
	if err != nil {
		ap.logger.Error("poll-metric-from-noaa", err)
		return
	}

	metric := models.GetMemoryMetricFromContainerMetrics(ap.appId, containerMetrics)
	ap.logger.Debug("poll-metric-get-memory-metric", lager.Data{"metric": *metric})

	if len(metric.Instances) == 0 {
		return
	}

	err = ap.database.SaveMetric(metric)
	if err != nil {
		ap.logger.Error("poll-metric-save", err)
	}
}
