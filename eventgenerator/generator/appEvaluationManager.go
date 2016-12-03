package generator

import (
	"autoscaler/eventgenerator/model"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type ConsumeAppMonitorMap func(map[string][]*model.Trigger, chan []*model.Trigger)
type AppEvaluationManager struct {
	evaluateInterval time.Duration
	logger           lager.Logger
	cclock           clock.Clock
	doneChan         chan bool
	triggerChan      chan []*model.Trigger
	getPolicies      model.GetPolicies
}

func NewAppEvaluationManager(logger lager.Logger, evaluateInterval time.Duration, cclock clock.Clock,
	triggerChan chan []*model.Trigger, getPolicies model.GetPolicies) (*AppEvaluationManager, error) {
	return &AppEvaluationManager{
		evaluateInterval: evaluateInterval,
		logger:           logger.Session("AppEvaluationManager"),
		cclock:           cclock,
		doneChan:         make(chan bool),
		triggerChan:      triggerChan,
		getPolicies:      getPolicies,
	}, nil
}

func (a *AppEvaluationManager) getTriggers(policyMap map[string]*model.Policy) map[string][]*model.Trigger {
	if policyMap == nil {
		return nil
	}

	triggersByType := make(map[string][]*model.Trigger)
	for appId, policy := range policyMap {
		for _, rule := range policy.TriggerRecord.ScalingRules {
			triggerKey := appId + "#" + rule.MetricType
			triggers, exist := triggersByType[triggerKey]
			if !exist {
				triggers = []*model.Trigger{}
			}
			triggers = append(triggers, &model.Trigger{
				AppId:            appId,
				MetricType:       rule.MetricType,
				BreachDuration:   rule.BreachDuration,
				CoolDownDuration: rule.CoolDownDuration,
				Threshold:        rule.Threshold,
				Operator:         rule.Operator,
				Adjustment:       rule.Adjustment,
			})
			triggersByType[triggerKey] = triggers
		}
	}
	return triggersByType
}

func (a *AppEvaluationManager) Start() {
	go a.doEvaluate()
}

func (a *AppEvaluationManager) Stop() {
	close(a.doneChan)
	a.logger.Info("stopped")
}

func (a *AppEvaluationManager) doEvaluate() {
	ticker := a.cclock.NewTicker(a.evaluateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.doneChan:
			return
		case <-ticker.C():
			triggers := a.getTriggers(a.getPolicies())
			for _, triggerArray := range triggers {
				a.triggerChan <- triggerArray
			}
		}
	}
	a.logger.Info("started")
}
