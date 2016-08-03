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
	conf     *config.CollectorConfig
	logger   lager.Logger
	cfc      cf.CfClient
	noaa     NoaaConsumer
	database db.DB
	cclock   clock.Clock
	doneChan chan bool
	pollers  map[string]*AppPoller
	ticker   clock.Ticker
	lock     *sync.Mutex
}

func NewCollector(conf *config.CollectorConfig, logger lager.Logger, cfc cf.CfClient, noaa NoaaConsumer, database db.DB, cclock clock.Clock) *Collector {
	return &Collector{
		conf:     conf,
		logger:   logger,
		cfc:      cfc,
		noaa:     noaa,
		database: database,
		cclock:   cclock,
		doneChan: make(chan bool),
		pollers:  make(map[string]*AppPoller),
		lock:     &sync.Mutex{},
	}
}

func (c *Collector) Start() {
	c.ticker = c.cclock.NewTicker(time.Duration(c.conf.RefreshInterval) * time.Second)
	go c.startAppRefresh()
	c.logger.Info("collector-started")
}

func (c *Collector) startAppRefresh() {
	for {
		select {
		case <-c.doneChan:
			return
		case <-c.ticker.C():
			c.refreshApps()
		}
	}
}

func (c *Collector) refreshApps() {
	appIds, err := c.database.GetAppIds()
	if err != nil {
		c.logger.Error("refresh-app", err)
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
			ap := NewAppPoller(id, c.conf.PollInterval, c.logger, c.cfc, c.noaa, c.database, c.cclock)
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
