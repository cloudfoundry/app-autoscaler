package aggregator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/model"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"time"
)

type Consumer func(map[string]*model.Policy, chan *model.AppMonitor)

type PolicyPoller struct {
	logger   lager.Logger
	interval time.Duration
	database db.PolicyDB
	appChan  chan *model.AppMonitor
	clock    clock.Clock
	tick     clock.Ticker
	doneChan chan bool
	consumer Consumer
}

func NewPolicyPoller(logger lager.Logger, clock clock.Clock, interval time.Duration, database db.PolicyDB, consumer Consumer, appChan chan *model.AppMonitor) *PolicyPoller {
	return &PolicyPoller{
		logger:   logger.Session("PolicyPoller"),
		clock:    clock,
		interval: interval,
		database: database,
		doneChan: make(chan bool),
		consumer: consumer,
		appChan:  appChan,
	}
}
func (p *PolicyPoller) Start() {
	p.tick = p.clock.NewTicker(p.interval)
	go p.startPolicyRetrieve()
	p.logger.Info("started", lager.Data{"interval": p.interval})
}

func (p *PolicyPoller) Stop() {
	if p.tick != nil {
		p.tick.Stop()
		close(p.doneChan)
	}
	p.logger.Info("stopped")
}
func (p *PolicyPoller) startPolicyRetrieve() {
	for {
		policyJsons, err := p.retrievePolicies()
		if err != nil {
			continue
		}
		policies := p.computePolicys(policyJsons)
		p.consumer(policies, p.appChan)
		select {
		case <-p.doneChan:
			return
		case <-p.tick.C():
		}
	}
}

func (p *PolicyPoller) retrievePolicies() ([]*model.PolicyJson, error) {
	policyJsons, err := p.database.RetrievePolicies()
	if err != nil {
		p.logger.Error("retrieve policyJsons", err)
		return nil, err
	}
	p.logger.Info("policy count", lager.Data{"count": len(policyJsons)})
	return policyJsons, nil
}
func (p *PolicyPoller) computePolicys(policyJsons []*model.PolicyJson) map[string]*model.Policy {
	policyMap := make(map[string]*model.Policy)
	for _, policyRow := range policyJsons {
		tmpPolicy := policyRow.GetPolicy()
		policyMap[policyRow.AppId] = tmpPolicy
	}
	p.logger.Info("policy count", lager.Data{"count": len(policyMap)})
	return policyMap
}
