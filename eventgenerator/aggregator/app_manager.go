package aggregator

import (
	"sync"
	"time"

	"golang.org/x/exp/maps"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/collection"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type Consumer func(map[string]*models.AppPolicy, chan *models.AppMonitor)
type GetPoliciesFunc func() map[string]*models.AppPolicy
type SaveAppMetricToCacheFunc func(*models.AppMetric) bool
type QueryAppMetricsFunc func(appID string, metricType string, start int64, end int64, orderType db.OrderType) ([]*models.AppMetric, error)

type AppManager struct {
	logger                lager.Logger
	interval              time.Duration
	nodeNum               int
	nodeIndex             int
	metricCacheSizePerApp int
	metricCache           map[string]*collection.TSDCache
	policyDB              db.PolicyDB
	appMetricDB           db.AppMetricDB
	clock                 clock.Clock
	doneChan              chan bool
	policyMap             map[string]*models.AppPolicy
	pLock                 sync.RWMutex
	mLock                 sync.RWMutex
}

func NewAppManager(logger lager.Logger, clock clock.Clock, aggregator config.AggregatorConfig, pool config.PoolConfig, policyDB db.PolicyDB, appMetricDB db.AppMetricDB) *AppManager {
	return &AppManager{
		logger:                logger.Session("AppManager"),
		clock:                 clock,
		interval:              aggregator.PolicyPollerInterval,
		nodeNum:               pool.TotalInstances,
		nodeIndex:             pool.InstanceIndex,
		metricCacheSizePerApp: aggregator.MetricCacheSizePerApp,
		metricCache:           make(map[string]*collection.TSDCache),
		policyDB:              policyDB,
		appMetricDB:           appMetricDB,
		doneChan:              make(chan bool),
		policyMap:             make(map[string]*models.AppPolicy),
	}
}
func (am *AppManager) GetPolicies() map[string]*models.AppPolicy {
	am.pLock.RLock()
	defer am.pLock.RUnlock()
	return maps.Clone(am.policyMap)
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
	policyJsons, err := am.policyDB.RetrievePolicies()
	if err != nil {
		am.logger.Error("retrieve policyJsons", err)
		return nil, err
	}
	am.logger.Debug("policycount", lager.Data{"count": len(policyJsons)})
	return policyJsons, nil
}

func (am *AppManager) isEventgeneratorRespForApp(appID string) bool {
	// #nosec G115 -- nodes will be in the range of a dozen max - no need to worry about integer overflow
	return helpers.FNVHash(appID)%uint32(am.nodeNum) == uint32(am.nodeIndex)
}

func (am *AppManager) computePolicies(policyJsons []*models.PolicyJson) map[string]*models.AppPolicy {
	policyMap := make(map[string]*models.AppPolicy)
	for _, policyJSON := range policyJsons {
		if (am.nodeNum == 1) || am.isEventgeneratorRespForApp(policyJSON.AppId) {
			appPolicy, err := policyJSON.GetAppPolicy()
			if err != nil {
				am.logger.Error("get-app-policy", err)
				continue
			}
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

func (am *AppManager) QueryAppMetrics(appID string, metricType string, start int64, end int64, order db.OrderType) ([]*models.AppMetric, error) {
	am.mLock.RLock()
	appCache := am.metricCache[appID]
	am.mLock.RUnlock()

	if end == -1 {
		end = time.Now().UnixNano()
	}

	if appCache != nil {
		labels := map[string]string{models.MetricLabelName: metricType}
		result, hit := appCache.Query(start, end+1, labels)
		if hit {
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
			return metrics, nil
		}
	}
	return am.appMetricDB.RetrieveAppMetrics(appID, metricType, start, end, order)
}
