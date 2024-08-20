package manager

import (
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/patrickmn/go-cache"
)

type Consumer func(map[string]*models.AppPolicy, chan *models.AppMonitor)

type PolicyManager struct {
	logger             lager.Logger
	interval           time.Duration
	cacheTTL           time.Duration
	allowedMetricCache cache.Cache
	database           db.PolicyDB
	clock              clock.Clock
	doneChan           chan bool
	policyMap          map[string]*models.AppPolicy
	pLock              sync.RWMutex
	mLock              sync.RWMutex
}

func NewPolicyManager(logger lager.Logger, clock clock.Clock, interval time.Duration,
	database db.PolicyDB, allowedMetricCache cache.Cache, cacheTTL time.Duration) *PolicyManager {
	return &PolicyManager{
		logger:             logger.Session("PolicyManager"),
		clock:              clock,
		interval:           interval,
		cacheTTL:           cacheTTL,
		allowedMetricCache: allowedMetricCache,
		database:           database,
		doneChan:           make(chan bool),
		policyMap:          make(map[string]*models.AppPolicy),
	}
}
func (pm *PolicyManager) Start() {
	go pm.startPolicyRetrieve()
	pm.logger.Info("started", lager.Data{"interval": pm.interval})
}

func (pm *PolicyManager) Stop() {
	close(pm.doneChan)
	pm.logger.Info("stopped")
}

func (pm *PolicyManager) startPolicyRetrieve() {
	tick := pm.clock.NewTicker(pm.interval)
	defer tick.Stop()

	for {
		policyJsons, err := pm.retrievePolicies()
		if err != nil {
			continue
		}
		policies := pm.computePolicies(policyJsons)

		pm.pLock.Lock()
		pm.policyMap = policies
		pm.pLock.Unlock()

		cacheRefresheErr := pm.RefreshAllowedMetricCache(policies)

		if cacheRefresheErr != nil {
			continue
		}
		select {
		case <-pm.doneChan:
			return
		case <-tick.C():
		}
	}
}

func (pm *PolicyManager) retrievePolicies() ([]*models.PolicyJson, error) {
	policyJsons, err := pm.database.RetrievePolicies()
	if err != nil {
		pm.logger.Error("retrieve policyJsons", err)
		return nil, err
	}
	pm.logger.Debug("policycount", lager.Data{"count": len(policyJsons)})
	return policyJsons, nil
}

func (pm *PolicyManager) computePolicies(policyJsons []*models.PolicyJson) map[string]*models.AppPolicy {
	policyMap := make(map[string]*models.AppPolicy)
	for _, policyJSON := range policyJsons {
		appPolicy, err := policyJSON.GetAppPolicy()
		if err != nil {
			pm.logger.Error("get-app-policy", err)
			continue
		}
		policyMap[policyJSON.AppId] = appPolicy
	}
	return policyMap
}

func (pm *PolicyManager) RefreshAllowedMetricCache(policies map[string]*models.AppPolicy) error {
	pm.mLock.Lock()
	defer pm.mLock.Unlock()
	allowedMetricTypeSet := make(map[string]struct{})
	allowedMetricMap := pm.allowedMetricCache.Items()
	//Iterating over the cache and replace the allowed metrics for existing policy
	for applicationId := range allowedMetricMap {
		if policy, ok := policies[applicationId]; ok {
			scalingPolicy := policy.ScalingPolicy
			for _, metrictype := range scalingPolicy.ScalingRules {
				allowedMetricTypeSet[metrictype.MetricType] = struct{}{}
			}
			err := pm.allowedMetricCache.Replace(applicationId, allowedMetricTypeSet, pm.cacheTTL)
			if err != nil {
				pm.logger.Error("Error updating allowedMetricCache", err)
				return err
			}
		} else {
			//If the policy is not present in the cache, remove the entry from the cache
			pm.allowedMetricCache.Delete(applicationId)
		}
	}
	return nil
}
