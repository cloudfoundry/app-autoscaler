package aggregator

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type Consumer func(map[string]*models.AppPolicy, chan *models.AppMonitor)

type PolicyPoller struct {
	logger    lager.Logger
	interval  time.Duration
	nodeNum   int
	nodeIndex int
	database  db.PolicyDB
	clock     clock.Clock
	doneChan  chan bool
	policyMap map[string]*models.AppPolicy
	lock      sync.RWMutex
}

func NewPolicyPoller(logger lager.Logger, clock clock.Clock, interval time.Duration, nodeNum, nodeIndex int, database db.PolicyDB) *PolicyPoller {
	return &PolicyPoller{
		logger:    logger.Session("PolicyPoller"),
		clock:     clock,
		interval:  interval,
		nodeNum:   nodeNum,
		nodeIndex: nodeIndex,
		database:  database,
		doneChan:  make(chan bool),
		policyMap: make(map[string]*models.AppPolicy),
	}
}
func (p *PolicyPoller) GetPolicies() map[string]*models.AppPolicy {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.policyMap
}
func (p *PolicyPoller) Start() {
	go p.startPolicyRetrieve()
	p.logger.Info("started", lager.Data{"interval": p.interval})
}

func (p *PolicyPoller) Stop() {
	close(p.doneChan)
	p.logger.Info("stopped")
}

func (p *PolicyPoller) startPolicyRetrieve() {
	tick := p.clock.NewTicker(p.interval)
	defer tick.Stop()

	for {
		policyJsons, err := p.retrievePolicies()
		if err != nil {
			continue
		}
		policies := p.computePolicies(policyJsons)
		p.lock.Lock()
		p.policyMap = policies
		p.lock.Unlock()
		select {
		case <-p.doneChan:
			return
		case <-tick.C():

		}
	}
}

func (p *PolicyPoller) retrievePolicies() ([]*models.PolicyJson, error) {
	policyJsons, err := p.database.RetrievePolicies()
	if err != nil {
		p.logger.Error("retrieve policyJsons", err)
		return nil, err
	}
	p.logger.Debug("policy count", lager.Data{"count": len(policyJsons)})
	return policyJsons, nil
}

func (p *PolicyPoller) computePolicies(policyJsons []*models.PolicyJson) map[string]*models.AppPolicy {
	policyMap := make(map[string]*models.AppPolicy)
	for _, policyJSON := range policyJsons {
		if (p.nodeNum == 1) || (helpers.FNVHash(policyJSON.AppId)%uint32(p.nodeNum) == uint32(p.nodeIndex)) {
			appPolicy := policyJSON.GetAppPolicy()
			policyMap[policyJSON.AppId] = appPolicy
		}
	}
	return policyMap
}
