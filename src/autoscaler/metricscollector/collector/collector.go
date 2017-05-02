package collector

import (
	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"sync"
	"time"
)

type AppCollector interface {
	Start()
	Stop()
}

type Collector struct {
	refreshInterval    time.Duration
	logger             lager.Logger
	database           db.PolicyDB
	cclock             clock.Clock
	createAppCollector func(string) AppCollector
	doneChan           chan bool
	appCollectors      map[string]AppCollector
	ticker             clock.Ticker
	lock               *sync.Mutex
}

func NewCollector(refreshInterval time.Duration, logger lager.Logger, database db.PolicyDB, cclock clock.Clock, createAppCollector func(string) AppCollector) *Collector {
	return &Collector{
		refreshInterval:    refreshInterval,
		logger:             logger,
		database:           database,
		cclock:             cclock,
		createAppCollector: createAppCollector,
		doneChan:           make(chan bool),
		appCollectors:      make(map[string]AppCollector),
		lock:               &sync.Mutex{},
	}
}

func (c *Collector) Start() {
	c.ticker = c.cclock.NewTicker(c.refreshInterval)
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
	for id, ap := range c.appCollectors {
		_, exist := appIds[id]
		if !exist {
			c.logger.Debug("refresh-apps-remove", lager.Data{"appId": id})
			ap.Stop()
			delete(c.appCollectors, id)
		}
	}

	for id, _ := range appIds {
		_, exist := c.appCollectors[id]
		if !exist {
			c.logger.Debug("refresh-apps-add", lager.Data{"appId": id})
			ap := c.createAppCollector(id)
			ap.Start()
			c.appCollectors[id] = ap
		}
	}
	c.lock.Unlock()
}

func (c *Collector) Stop() {
	if c.ticker != nil {
		c.ticker.Stop()
		close(c.doneChan)

		c.lock.Lock()
		for _, ap := range c.appCollectors {
			ap.Stop()
		}
		c.lock.Unlock()
	}
	c.logger.Info("collector-stopped")
}

func (c *Collector) GetCollectorAppIds() []string {
	var appIds []string
	c.lock.Lock()
	for id, _ := range c.appCollectors {
		appIds = append(appIds, id)
	}
	c.lock.Unlock()
	return appIds
}
