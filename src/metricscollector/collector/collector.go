package collector

import (
	"metricscollector/cf"
	"metricscollector/config"
	"metricscollector/db"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"

	"sync"
	"time"
)

type NoaaConsumer interface {
	ContainerMetrics(appGuid string, authToken string) ([]*events.ContainerMetric, error)
}

type Collector struct {
	logger   lager.Logger
	conf     *config.CollectorConfig
	cfc      cf.CfClient
	noaa     NoaaConsumer
	database db.DB
	tick     *time.Ticker
	doneChan chan bool
	pollers  map[string]*AppPoller
	lock     *sync.Mutex
}

func NewCollector(logger lager.Logger, conf *config.CollectorConfig, cfc cf.CfClient, noaa NoaaConsumer, database db.DB) *Collector {
	return &Collector{
		logger:   logger,
		conf:     conf,
		cfc:      cfc,
		noaa:     noaa,
		database: database,
		doneChan: make(chan bool),
		pollers:  make(map[string]*AppPoller),
		lock:     &sync.Mutex{},
	}
}

func (c *Collector) Start() {
	c.tick = time.NewTicker(time.Duration(c.conf.RefreshInterval) * time.Second)
	go c.startAppRefresh()

	c.logger.Info("collector-started", lager.Data{"config": c.conf})
}

func (c *Collector) startAppRefresh() {
	c.refreshApps()
	for {
		select {
		case <-c.doneChan:
			return
		case <-c.tick.C:
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
	for id, poller := range c.pollers {
		_, exist := appIds[id]
		if !exist {
			c.logger.Debug("refresh-apps-remove", lager.Data{"appId": id})
			poller.Stop()
			delete(c.pollers, id)
		}
	}

	for id, _ := range appIds {
		_, exist := c.pollers[id]
		if !exist {
			c.logger.Debug("refresh-apps-add", lager.Data{"appId": id})
			ap := NewAppPoller(c.logger, id, time.Duration(c.conf.PollInterval)*time.Second, c.cfc, c.noaa, c.database)
			c.pollers[id] = ap
			ap.Start()
		}
	}
	c.lock.Unlock()
}

func (c *Collector) Stop() {
	if c.tick != nil {
		c.tick.Stop()
		c.doneChan <- true

		c.lock.Lock()
		for _, ap := range c.pollers {
			ap.Stop()
		}
		c.lock.Unlock()
	}
	c.logger.Info("collector-stopped")
}
