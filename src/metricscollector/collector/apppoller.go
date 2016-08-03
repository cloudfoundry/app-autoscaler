package collector

import (
	"metricscollector/cf"
	"metricscollector/db"
	"metricscollector/metrics"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"time"
)

type AppPoller struct {
	appId        string
	pollInterval int
	logger       lager.Logger
	cfc          cf.CfClient
	noaa         NoaaConsumer
	database     db.DB
	pclock       clock.Clock
	ticker       clock.Ticker
	doneChan     chan bool
}

func NewAppPoller(appId string, pollInterval int, logger lager.Logger, cfc cf.CfClient, noaa NoaaConsumer, database db.DB, pclcok clock.Clock) *AppPoller {
	return &AppPoller{
		appId:        appId,
		pollInterval: pollInterval,
		logger:       logger,
		cfc:          cfc,
		noaa:         noaa,
		database:     database,
		pclock:       pclcok,
		doneChan:     make(chan bool),
	}

}

func (ap *AppPoller) Start() {
	ap.ticker = ap.pclock.NewTicker(time.Duration(ap.pollInterval) * time.Second)
	go ap.startPollMetrics()

	ap.logger.Info("app-poller-started", lager.Data{"appid": ap.appId, "poll-interval": ap.pollInterval})
}

func (ap *AppPoller) Stop() {
	if ap.ticker != nil {
		ap.ticker.Stop()
		close(ap.doneChan)
	}
	ap.logger.Info("app-poller-stopped", lager.Data{"appid": ap.appId})
}

func (ap *AppPoller) startPollMetrics() {
	for {
		select {
		case <-ap.doneChan:
			return
		case <-ap.ticker.C():
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
