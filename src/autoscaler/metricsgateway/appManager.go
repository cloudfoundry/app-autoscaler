package metricsgateway

import (
	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"sync"
	"time"
)

type GetAppIDsFunc func() map[string]bool

type AppManager struct {
	logger   lager.Logger
	interval time.Duration
	policyDB db.PolicyDB
	clock    clock.Clock
	doneChan chan bool
	appIDMap map[string]bool
	pLock    sync.RWMutex
}

func NewAppManager(logger lager.Logger, clock clock.Clock, interval time.Duration, policyDB db.PolicyDB) *AppManager {
	return &AppManager{
		logger:   logger.Session("AppManager"),
		clock:    clock,
		interval: interval,
		policyDB: policyDB,
		doneChan: make(chan bool),
		appIDMap: make(map[string]bool),
	}
}
func (am *AppManager) GetAppIDs() map[string]bool {
	am.pLock.RLock()
	defer am.pLock.RUnlock()
	return am.appIDMap
}
func (am *AppManager) Start() {
	go am.startAppIDsRetrieve()
	am.logger.Info("started", lager.Data{"interval": am.interval})
}

func (am *AppManager) Stop() {
	close(am.doneChan)
	am.logger.Info("stopped")
}

func (am *AppManager) startAppIDsRetrieve() {
	tick := am.clock.NewTicker(am.interval)
	defer tick.Stop()
	for {
		appIDMap, err := am.retrieveAppIDs()
		if err != nil {
			continue
		}
		am.pLock.Lock()
		am.appIDMap = appIDMap
		am.pLock.Unlock()
		select {
		case <-am.doneChan:
			return
		case <-tick.C():

		}
	}
}

func (am *AppManager) retrieveAppIDs() (map[string]bool, error) {
	appIDMap, err := am.policyDB.GetAppIds()
	if err != nil {
		am.logger.Error("retrieve-app-ids", err)
		return nil, err
	}
	am.logger.Debug("retrieve-app-ids", lager.Data{"count": len(appIDMap)})
	return appIDMap, nil
}
