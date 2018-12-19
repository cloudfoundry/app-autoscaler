package collector

import (
	"autoscaler/cf"
	"autoscaler/collection"
	"autoscaler/metricscollector/noaa"
	"autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry/sonde-go/events"

	"time"
)

type appPoller struct {
	appId                         string
	collectInterval               time.Duration
	cache                         *collection.TSDCache
	logger                        lager.Logger
	cfc                           cf.CFClient
	noaaConsumer                  noaa.NoaaConsumer
	pclock                        clock.Clock
	doneChan                      chan bool
	dataChan                      chan *models.AppInstanceMetric
	isMetricsPersistencySupported bool
}

func NewAppPoller(logger lager.Logger, appId string, collectInterval time.Duration, cacheSize int, isMetricsPersistencySupported bool, cfc cf.CFClient, noaaConsumer noaa.NoaaConsumer, pclock clock.Clock, dataChan chan *models.AppInstanceMetric) AppCollector {
	return &appPoller{
		appId:                         appId,
		collectInterval:               collectInterval,
		cache:                         collection.NewTSDCache(cacheSize),
		logger:                        logger,
		cfc:                           cfc,
		noaaConsumer:                  noaaConsumer,
		pclock:                        pclock,
		doneChan:                      make(chan bool),
		dataChan:                      dataChan,
		isMetricsPersistencySupported: isMetricsPersistencySupported,
	}

}

func (ap *appPoller) Start() {
	go ap.startPollMetrics()

	ap.logger.Info("app-poller-started", lager.Data{"appid": ap.appId, "collect-interval": ap.collectInterval})
}

func (ap *appPoller) Stop() {
	ap.doneChan <- true
	ap.logger.Info("app-poller-stopped", lager.Data{"appid": ap.appId})
}

func (ap *appPoller) startPollMetrics() {
	for {
		ap.pollMetric()
		timer := ap.pclock.NewTimer(ap.collectInterval)
		select {
		case <-ap.doneChan:
			timer.Stop()
			return
		case <-timer.C():
		}
	}
}

func (ap *appPoller) pollMetric() {
	logger := ap.logger.WithData(lager.Data{"appId": ap.appId})
	logger.Debug("poll-metric")

	var containerEnvelopes []*events.Envelope
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		logger.Debug("poll-metric-from-noaa-retry", lager.Data{"attempt": attempt + 1})
		containerEnvelopes, err = ap.noaaConsumer.ContainerEnvelopes(ap.appId, cf.TokenTypeBearer+" "+ap.cfc.GetTokens().AccessToken)
		if err == nil {
			break
		}
	}
	if err != nil {
		logger.Error("poll-metric-from-noaa", err)
		return
	}

	logger.Debug("poll-metric-get-containerenvelopes", lager.Data{"envelops": containerEnvelopes})

	metrics := noaa.GetMetricsFromContainerEnvelopes(ap.pclock.Now().UnixNano(), ap.appId, containerEnvelopes)
	logger.Debug("poll-metric-get-metrics", lager.Data{"metrics": metrics})

	for _, metric := range metrics {
		ap.cache.Put(metric)
		if ap.isMetricsPersistencySupported {
			ap.dataChan <- metric
		}
	}
}

func (ap *appPoller) Query(start, end int64, labels map[string]string) ([]collection.TSD, bool) {
	return ap.cache.Query(start, end+1, labels)
}
