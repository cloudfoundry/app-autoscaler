package collector

import (
	"metricscollector/cf"
	"metricscollector/db"
	"metricscollector/metrics"

	"code.cloudfoundry.org/lager"
	"time"
)

type AppPoller struct {
	logger   lager.Logger
	appId    string
	interval time.Duration
	cfc      cf.CfClient
	noaa     NoaaConsumer
	database db.DB
	ticker   *time.Ticker
	doneChan chan bool
}

func NewAppPoller(logger lager.Logger, appId string, interval time.Duration, cfc cf.CfClient, noaa NoaaConsumer, database db.DB) *AppPoller {
	return &AppPoller{
		logger:   logger,
		appId:    appId,
		interval: interval,
		cfc:      cfc,
		noaa:     noaa,
		database: database,
		doneChan: make(chan bool),
	}

}

func (ap *AppPoller) Start() {
	ap.ticker = time.NewTicker(ap.interval)
	go ap.startPollMetrics()

	ap.logger.Info("app-poller-started", lager.Data{"appid": ap.appId})
}

func (ap *AppPoller) Stop() {
	if ap.ticker != nil {
		ap.ticker.Stop()
		ap.doneChan <- true
	}
	ap.logger.Info("app-poller-stopped", lager.Data{"appid": ap.appId})
}

func (ap *AppPoller) startPollMetrics() {
	ap.pollMetric()
	for {
		select {
		case <-ap.doneChan:
			return
		case <-ap.ticker.C:
			ap.pollMetric()
		}
	}
}

func (ap *AppPoller) pollMetric() {
	ap.logger.Debug("poll-metric", lager.Data{"appid": ap.appId})

	containerMetrics, err := ap.noaa.ContainerMetrics(ap.appId, "bearer"+" "+ap.cfc.GetTokens().AccessToken)
	if err != nil {
		ap.logger.Error("poll-metric-from-noaa", err)
		return
	}

	metric := metrics.GetMemoryMetricFromContainerMetrics(ap.appId, containerMetrics)
	ap.logger.Debug("poll-metric-get-memory-metric", lager.Data{"metric": *metric})

	if len(metric.Instances) == 0 {
		return
	}

	err = ap.database.SaveMetric(metric)
	if err != nil {
		ap.logger.Error("poll-metric-save", err)
	}
}
