package collector

import (
	"metricscollector/cf"
	"metricscollector/config"
	"metricscollector/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"
	"sync"
	"time"
)

type NoaaConsumer interface {
	ContainerMetrics(appGuid string, authToken string) ([]*events.ContainerMetric, error)
}

type Collector struct {
	conf         *config.CollectorConfig
	logger       lager.Logger
	cfc          cf.CfClient
	noaa         NoaaConsumer
	database     db.DB
	cclock       clock.Clock
	createPoller func(string, time.Duration, lager.Logger, cf.CfClient, NoaaConsumer, db.DB, clock.Clock) AppPoller
	doneChan     chan bool
	pollers      map[string]AppPoller
	ticker       clock.Ticker
	lock         *sync.Mutex
}

var createAppPollerFunc = func(appId string, pollInterval time.Duration, logger lager.Logger, cfc cf.CfClient, noaa NoaaConsumer, database db.DB, pclcok clock.Clock) AppPoller {
	return NewAppPoller(appId, pollInterval, logger, cfc, noaa, database, pclcok)
}

func NewCollector(conf *config.CollectorConfig, logger lager.Logger, cfc cf.CfClient, noaa NoaaConsumer, database db.DB, cclock clock.Clock,
	createPoller func(string, time.Duration, lager.Logger, cf.CfClient, NoaaConsumer, db.DB, clock.Clock) AppPoller) *Collector {
	return &Collector{
		conf:         conf,
		logger:       logger,
		cfc:          cfc,
		noaa:         noaa,
		database:     database,
		cclock:       cclock,
		createPoller: createPoller,
		doneChan:     make(chan bool),
		pollers:      make(map[string]AppPoller),
		lock:         &sync.Mutex{},
	}
}

func (c *Collector) Start() {
	c.ticker = c.cclock.NewTicker(c.conf.RefreshInterval)
	go c.startAppRefresh()
	c.logger.Info("collector-started")
}

func (c *Collector) startAppRefresh() {
	for {
		c.refreshApps()
		select {
		case <-c.doneChan:
			return
		case <-c.ticker.C():
		}
	}
}

func (c *Collector) refreshApps() {
	appIds, err := c.database.GetAppIds()
	if err != nil {
		c.logger.Error("refresh-apps", err)
		return
	}

	c.logger.Debug("refresh-apps", lager.Data{"appIds": appIds})

	c.lock.Lock()
	for id, ap := range c.pollers {
		_, exist := appIds[id]
		if !exist {
			c.logger.Debug("refresh-apps-remove", lager.Data{"appId": id})
			ap.Stop()
			delete(c.pollers, id)
		}
	}

	for id, _ := range appIds {
		_, exist := c.pollers[id]
		if !exist {
			c.logger.Debug("refresh-apps-add", lager.Data{"appId": id})
			ap := c.createPoller(id, c.conf.PollInterval, c.logger, c.cfc, c.noaa, c.database, c.cclock)
			ap.Start()
			c.pollers[id] = ap
		}
	}
	c.lock.Unlock()
}

func (c *Collector) Stop() {
	if c.ticker != nil {
		c.ticker.Stop()
		close(c.doneChan)

		c.lock.Lock()
		for _, ap := range c.pollers {
			ap.Stop()
		}
		c.lock.Unlock()
	}
	c.logger.Info("collector-stopped")
}

func (c *Collector) GetPollerAppIds() []string {
	var appIds []string
	c.lock.Lock()
	for id, _ := range c.pollers {
		appIds = append(appIds, id)
	}
	c.lock.Unlock()
	return appIds
}
