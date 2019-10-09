package collector

import (
	"autoscaler/collection"
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"fmt"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type MetricCollector interface {
	Start()
	Stop()
	GetAppIDs() map[string]bool
	QueryMetrics(string, int, string, int64, int64, db.OrderType) ([]*models.AppInstanceMetric, error)
	QueryMetricsWithLabels(string, int64, int64, db.OrderType, map[string]string) ([]*models.AppInstanceMetric, bool)
}

type metricCollector struct {
	logger                lager.Logger
	refreshInterval       time.Duration
	collectInterval       time.Duration
	PersistMetrics        bool
	saveInterval          time.Duration
	nodeNum               int
	nodeIndex             int
	metricCacheSizePerApp int
	policyDb              db.PolicyDB
	instancemetricsDb     db.InstanceMetricsDB
	cclock                clock.Clock
	doneChan              chan bool
	doneSaveChan          chan bool
	ticker                clock.Ticker
	lock                  *sync.RWMutex
	metricsChan           <-chan *models.AppInstanceMetric
	appIDs                map[string]bool
	metricCache           map[string]*collection.TSDCache
	mLock                 *sync.RWMutex
}

func NewCollector(logger lager.Logger, refreshInterval time.Duration, collectInterval time.Duration, PersistMetrics bool, saveInterval time.Duration,
	nodeIndex, nodeNum int, metricCacheSizePerApp int, policyDb db.PolicyDB, instancemetricsDb db.InstanceMetricsDB,
	cclock clock.Clock, metricsChan <-chan *models.AppInstanceMetric) *metricCollector {
	return &metricCollector{
		refreshInterval:       refreshInterval,
		collectInterval:       collectInterval,
		saveInterval:          saveInterval,
		PersistMetrics:        PersistMetrics,
		nodeIndex:             nodeIndex,
		nodeNum:               nodeNum,
		metricCacheSizePerApp: metricCacheSizePerApp,
		logger:                logger,
		policyDb:              policyDb,
		instancemetricsDb:     instancemetricsDb,
		cclock:                cclock,
		doneChan:              make(chan bool),
		doneSaveChan:          make(chan bool),
		lock:                  &sync.RWMutex{},
		metricsChan:           metricsChan,
		appIDs:                map[string]bool{},
		metricCache:           make(map[string]*collection.TSDCache),
		mLock:                 &sync.RWMutex{},
	}
}

func (c *metricCollector) Start() {
	c.ticker = c.cclock.NewTicker(c.refreshInterval)
	go c.startAppRefresh()
	go c.saveMetrics()

	c.logger.Info("collector-started")
}

func (c *metricCollector) startAppRefresh() {
	for {
		c.refreshApps()
		select {
		case <-c.doneChan:
			return
		case <-c.ticker.C():
		}
	}
}

func (c *metricCollector) refreshApps() {
	apps, err := c.policyDb.GetAppIds()
	if err != nil {
		c.logger.Error("refresh-apps", err)
		return
	}

	appIDs := map[string]bool{}
	for id := range apps {
		if helpers.FNVHash(id)%uint32(c.nodeNum) == uint32(c.nodeIndex) {
			appIDs[id] = true
		}
	}

	c.lock.Lock()
	c.appIDs = appIDs
	c.lock.Unlock()

	c.mLock.Lock()
	for id := range c.metricCache {
		if _, exist := appIDs[id]; !exist {
			delete(c.metricCache, id)
		}
	}
	for id := range appIDs {
		if _, exist := c.metricCache[id]; !exist {
			c.metricCache[id] = collection.NewTSDCache(c.metricCacheSizePerApp)
		}
	}
	c.mLock.Unlock()

}

func (c *metricCollector) Stop() {
	if c.ticker != nil {
		c.ticker.Stop()
		c.doneChan <- true
		c.doneSaveChan <- true
	}
	c.logger.Info("collector-stopped")
}

func (c *metricCollector) GetAppIDs() map[string]bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.appIDs
}

func (c *metricCollector) QueryMetrics(appID string, instanceIndex int, name string, start, end int64, order db.OrderType) ([]*models.AppInstanceMetric, error) {
	if end == -1 {
		end = c.cclock.Now().UnixNano()
	}

	c.mLock.RLock()
	appCache, exist := c.metricCache[appID]
	c.mLock.RUnlock()

	if exist {
		labels := map[string]string{models.MetricLabelName: name}
		if instanceIndex >= 0 {
			labels[models.MetricLabelInstanceIndex] = fmt.Sprintf("%d", instanceIndex)
		}
		result, hit := appCache.Query(start, end+1, labels)
		if hit || !c.PersistMetrics {
			metrics := make([]*models.AppInstanceMetric, len(result))
			if order == db.ASC {
				for index, tsd := range result {
					metrics[index] = tsd.(*models.AppInstanceMetric)
				}
			} else {
				for index, tsd := range result {
					metrics[len(result)-1-index] = tsd.(*models.AppInstanceMetric)
				}
			}
			return metrics, nil
		}
	}

	if c.PersistMetrics {
		return c.instancemetricsDb.RetrieveInstanceMetrics(appID, instanceIndex, name, start, end, order)
	}
	return []*models.AppInstanceMetric{}, nil
}

func (c *metricCollector) QueryMetricsWithLabels(appID string, start, end int64, order db.OrderType, labels map[string]string) ([]*models.AppInstanceMetric, bool) {
	if end == -1 {
		end = c.cclock.Now().UnixNano()
	}

	c.mLock.RLock()
	appCache, exist := c.metricCache[appID]
	c.mLock.RUnlock()

	if exist {
		result, hit := appCache.Query(start, end+1, labels)
		if hit || !c.PersistMetrics {
			metrics := make([]*models.AppInstanceMetric, len(result))
			if order == db.ASC {
				for index, tsd := range result {
					metrics[index] = tsd.(*models.AppInstanceMetric)
				}
			} else {
				for index, tsd := range result {
					metrics[len(result)-1-index] = tsd.(*models.AppInstanceMetric)
				}
			}
			return metrics, true
		}
	}
	return nil, false
}

func (c *metricCollector) saveMetrics() {
	ticker := c.cclock.NewTicker(c.saveInterval)
	metrics := []*models.AppInstanceMetric{}
	for {
		select {
		case m := <-c.metricsChan:
			c.saveMetricToCache(m)
			if c.PersistMetrics {
				metrics = append(metrics, m)
			}
		case <-ticker.C():
			if c.PersistMetrics && len(metrics) > 0 {
				go func(instancemetricsDb db.InstanceMetricsDB, metrics []*models.AppInstanceMetric) {
					instancemetricsDb.SaveMetricsInBulk(metrics)
					metrics = nil
					return
				}(c.instancemetricsDb, metrics)
				metrics = nil
			}
		case <-c.doneSaveChan:
			return
		}
	}
}

func (c *metricCollector) saveMetricToCache(m *models.AppInstanceMetric) bool {
	c.mLock.RLock()
	appCache := c.metricCache[m.AppId]
	c.mLock.RUnlock()

	if appCache != nil {
		appCache.Put(m)
		return true
	}
	return false
}
