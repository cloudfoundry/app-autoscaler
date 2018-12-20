package aggregator

import (
	"autoscaler/collection"
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type Consumer func(map[string]*models.AppPolicy, chan *models.AppMonitor)
type GetPoliciesFunc func() map[string]*models.AppPolicy
type SaveAppMetricToCacheFunc func(*models.AppMetric) bool
type QueryAppMetricFromCacheFunc func(string, int64, int64, db.OrderType, map[string]string) ([]*models.AppMetric, bool)

type AppManager struct {
	logger                lager.Logger
	interval              time.Duration
	nodeNum               int
	nodeIndex             int
	metricCacheSizePerApp int
	metricCache           map[string]*collection.TSDCache
	database              db.PolicyDB
	clock                 clock.Clock
	doneChan              chan bool
	policyMap             map[string]*models.AppPolicy
	pLock                 sync.RWMutex
	mLock                 sync.RWMutex
}

func NewAppManager(logger lager.Logger, clock clock.Clock, interval time.Duration, nodeNum, nodeIndex int,
	metricCacheSizePerApp int, database db.PolicyDB) *AppManager {
	return &AppManager{
		logger:                logger.Session("AppManager"),
		clock:                 clock,
		interval:              interval,
		nodeNum:               nodeNum,
		nodeIndex:             nodeIndex,
		metricCacheSizePerApp: metricCacheSizePerApp,
		metricCache:           make(map[string]*collection.TSDCache),
		database:              database,
		doneChan:              make(chan bool),
		policyMap:             make(map[string]*models.AppPolicy),
	}
}
func (am *AppManager) GetPolicies() map[string]*models.AppPolicy {
	am.pLock.RLock()
	defer am.pLock.RUnlock()
	return am.policyMap
}
func (am *AppManager) Start() {
	go am.startPolicyRetrieve()
	am.logger.Info("started", lager.Data{"interval": am.interval})
}

func (am *AppManager) Stop() {
	close(am.doneChan)
	am.logger.Info("stopped")
}

func (am *AppManager) startPolicyRetrieve() {
	tick := am.clock.NewTicker(am.interval)
	defer tick.Stop()

	for {
		policyJsons, err := am.retrievePolicies()
		if err != nil {
			continue
		}
		policies := am.computePolicies(policyJsons)

		am.pLock.Lock()
		am.policyMap = policies
		am.pLock.Unlock()

		am.refreshMetricCache(policies)

		select {
		case <-am.doneChan:
			return
		case <-tick.C():

		}
	}
}

func (am *AppManager) retrievePolicies() ([]*models.PolicyJson, error) {
	policyJsons, err := am.database.RetrievePolicies()
	if err != nil {
		am.logger.Error("retrieve policyJsons", err)
		return nil, err
	}
	am.logger.Debug("policycount", lager.Data{"count": len(policyJsons)})
	return policyJsons, nil
}

func (am *AppManager) computePolicies(policyJsons []*models.PolicyJson) map[string]*models.AppPolicy {
	policyMap := make(map[string]*models.AppPolicy)
	for _, policyJSON := range policyJsons {
		if (am.nodeNum == 1) || (helpers.FNVHash(policyJSON.AppId)%uint32(am.nodeNum) == uint32(am.nodeIndex)) {
			appPolicy := policyJSON.GetAppPolicy()
			policyMap[policyJSON.AppId] = appPolicy
		}
	}
	return policyMap
}

func (am *AppManager) refreshMetricCache(policies map[string]*models.AppPolicy) {
	am.mLock.Lock()
	defer am.mLock.Unlock()
	for id := range am.metricCache {
		_, exist := policies[id]
		if !exist {
			delete(am.metricCache, id)
		}
	}

	for id := range policies {
		_, exist := am.metricCache[id]
		if !exist {
			am.metricCache[id] = collection.NewTSDCache(am.metricCacheSizePerApp)
		}
	}
}

func (am *AppManager) SaveMetricToCache(metric *models.AppMetric) bool {
	am.mLock.RLock()
	appCache := am.metricCache[metric.AppId]
	am.mLock.RUnlock()

	if appCache != nil {
		appCache.Put(metric)
		return true
	}
	return false
}

func (am *AppManager) QueryMetricsFromCache(appID string, start, end int64, order db.OrderType, labels map[string]string) ([]*models.AppMetric, bool) {
	am.mLock.RLock()
	appCache := am.metricCache[appID]
	am.mLock.RUnlock()

	if appCache == nil {
		return nil, false
	}

	result, ok := appCache.Query(start, end, labels)
	if !ok {
		return nil, false
	}

	metrics := make([]*models.AppMetric, len(result))
	if order == db.ASC {
		for index, tsd := range result {
			metrics[index] = tsd.(*models.AppMetric)
		}
	} else {
		for index, tsd := range result {
			metrics[len(result)-1-index] = tsd.(*models.AppMetric)
		}
	}
	return metrics, true
}
