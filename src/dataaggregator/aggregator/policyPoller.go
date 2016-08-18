package aggregator

import (
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"dataaggregator/db"
	"dataaggregator/policy"
	"time"
)

type Consumer func([]*policy.PolicyJson, map[string]*policy.Trigger)

type PolicyPoller struct {
	logger   lager.Logger
	interval time.Duration
	database db.DB
	clock    clock.Clock
	tick     clock.Ticker
	doneChan chan bool
	consumer Consumer
}

func NewPolicyPoller(logger lager.Logger, clock clock.Clock, interval time.Duration, database db.DB, consumer Consumer) *PolicyPoller {
	return &PolicyPoller{
		logger:   logger.Session("policy-poller"),
		clock:    clock,
		interval: interval,
		database: database,
		doneChan: make(chan bool),
		consumer: consumer,
	}
}
func (p *PolicyPoller) Start() {
	p.tick = p.clock.NewTicker(p.interval)
	go p.startPolicyRetrieve()
	p.logger.Info("policy-poller-started", lager.Data{"interval": p.interval})
}

func (p *PolicyPoller) Stop() {
	if p.tick != nil {
		p.tick.Stop()
		close(p.doneChan)
	}
	p.logger.Info("policy-poller-stopped")
}
func (p *PolicyPoller) startPolicyRetrieve() {
	for {
		policies, err := p.retrievePolicies()
		if err != nil {
			continue
		}
		triggers := p.computeTriggers(policies)
		p.consumer(policies, triggers)
		select {
		case <-p.doneChan:
			return
		case <-p.tick.C():
		}
	}
}

func (p *PolicyPoller) retrievePolicies() ([]*policy.PolicyJson, error) {
	policies, err := p.database.RetrievePolicies()
	if err != nil {
		p.logger.Error("retrieve policies", err)
		return nil, err
	}
	p.logger.Info("policy count", lager.Data{"count": len(policies)})
	return policies, nil
}
func (p *PolicyPoller) computeTriggers(policies []*policy.PolicyJson) map[string]*policy.Trigger {
	triggerMap := make(map[string]*policy.Trigger)
	for _, policyRow := range policies {
		tmpTrigger := policyRow.GetTrigger()
		triggerMap[policyRow.AppId] = tmpTrigger
	}
	p.logger.Info("trigger count", lager.Data{"count": len(triggerMap)})
	return triggerMap
}
