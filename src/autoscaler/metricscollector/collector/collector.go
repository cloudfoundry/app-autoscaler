package collector

import (
	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"sync"
	"time"
)

type Collector struct {
	refreshInterval time.Duration
	logger          lager.Logger
	database        db.PolicyDB
	cclock          clock.Clock
	createPoller    func(string) AppPoller
	doneChan        chan bool
	pollers         map[string]AppPoller
	timer           clock.Timer
	lock            *sync.Mutex
}

func NewCollector(refreshInterval time.Duration, logger lager.Logger, database db.PolicyDB, cclock clock.Clock, createPoller func(string) AppPoller) *Collector {
	return &Collector{
		refreshInterval: refreshInterval,
		logger:          logger,
		database:        database,
		cclock:          cclock,
		createPoller:    createPoller,
		doneChan:        make(chan bool),
		pollers:         make(map[string]AppPoller),
		lock:            &sync.Mutex{},
	}
}

func (c *Collector) Start() {
	c.timer = c.cclock.NewTimer(c.refreshInterval)
	go c.startAppRefresh()
	c.logger.Info("collector-started")
}

func (c *Collector) startAppRefresh() {
	for {
		c.refreshApps()
		c.timer.Reset(c.refreshInterval)
		select {
		case <-c.doneChan:
			return
		case <-c.timer.C():
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
			ap := c.createPoller(id)
			ap.Start()
			c.pollers[id] = ap
		}
	}
	c.lock.Unlock()
}

func (c *Collector) Stop() {
	if c.timer != nil {
		c.timer.Stop()
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
