package collector

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type AppCollector interface {
	Start()
	Stop()
}

type Collector struct {
	refreshInterval    time.Duration
	collectInterval    time.Duration
	saveInterval       time.Duration
	nodeNum            int
	nodeIndex          int
	logger             lager.Logger
	policyDb           db.PolicyDB
	instancemetricsDb  db.InstanceMetricsDB
	cclock             clock.Clock
	createAppCollector func(string, chan *models.AppInstanceMetric) AppCollector
	doneChan           chan bool
	doneSaveChan       chan bool
	appCollectors      map[string]AppCollector
	ticker             clock.Ticker
	lock               *sync.Mutex
	dataChan           chan *models.AppInstanceMetric
}

func NewCollector(refreshInterval time.Duration, collectInterval time.Duration, saveInterval time.Duration,
	nodeIndex, nodeNum int, logger lager.Logger, policyDb db.PolicyDB, instancemetricsDb db.InstanceMetricsDB,
	cclock clock.Clock, createAppCollector func(string, chan *models.AppInstanceMetric) AppCollector) *Collector {
	return &Collector{
		refreshInterval:    refreshInterval,
		collectInterval:    collectInterval,
		saveInterval:       saveInterval,
		nodeIndex:          nodeIndex,
		nodeNum:            nodeNum,
		logger:             logger,
		policyDb:           policyDb,
		instancemetricsDb:  instancemetricsDb,
		cclock:             cclock,
		createAppCollector: createAppCollector,
		doneChan:           make(chan bool),
		doneSaveChan:       make(chan bool),
		appCollectors:      make(map[string]AppCollector),
		lock:               &sync.Mutex{},
		dataChan:           make(chan *models.AppInstanceMetric),
	}
}

func (c *Collector) Start() {
	c.ticker = c.cclock.NewTicker(c.refreshInterval)
	go c.startAppRefresh()
	go c.SaveMetricsInDB()
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
	appIds, err := c.policyDb.GetAppIds()
	if err != nil {
		c.logger.Error("refresh-apps", err)
		return
	}

	appShard := map[string]bool{}
	if c.nodeNum == 1 {
		appShard = appIds
	} else {
		for id := range appIds {
			if helpers.FNVHash(id)%uint32(c.nodeNum) == uint32(c.nodeIndex) {
				appShard[id] = true
			}
		}
	}

	c.lock.Lock()
	for id, ap := range c.appCollectors {
		_, exist := appShard[id]
		if !exist {
			c.logger.Debug("refresh-apps-remove", lager.Data{"appId": id})
			ap.Stop()
			delete(c.appCollectors, id)
		}
	}

	for id := range appShard {
		_, exist := c.appCollectors[id]
		if !exist {
			c.logger.Debug("refresh-apps-add", lager.Data{"appId": id})
			ap := c.createAppCollector(id, c.dataChan)
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
		close(c.doneSaveChan)

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
	for id := range c.appCollectors {
		appIds = append(appIds, id)
	}
	c.lock.Unlock()
	return appIds
}

func (c *Collector) SaveMetricsInDB() {
	ticker := c.cclock.NewTicker(c.saveInterval)
	metrics := make([]*models.AppInstanceMetric, 0)
	for {
		select {
		case metric := <-c.dataChan:
			metrics = append(metrics, metric)
		case <-ticker.C():
			go func(instancemetricsDb db.InstanceMetricsDB, metrics []*models.AppInstanceMetric) {
				instancemetricsDb.SaveMetricsInBulk(metrics)
				metrics = nil
				return
			}(c.instancemetricsDb, metrics)
			metrics = nil
		case <-c.doneSaveChan:
			return
		}
	}
}
