package generator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/model"
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"net/http"
	"sync"
	"time"
)

type ConsumeAppMonitorMap func(map[string][]*model.Trigger, chan []*model.Trigger)
type AppEvaluationManager struct {
	evaluateInterval time.Duration
	logger           lager.Logger
	cclock           clock.Clock
	lock             sync.Mutex
	doneChan         chan bool
	triggerChan      chan []*model.Trigger
	triggers         map[string][]*model.Trigger
	evaluatorArray   []*Evaluator
	getPolicies      model.GetPolicies
}

func NewAppEvaluationManager(evaluateInterval time.Duration, logger lager.Logger, cclock clock.Clock, triggerChan chan []*model.Trigger,
	evaluatorCount int, database db.AppMetricDB, scalingEngineUrl string, getPolicies model.GetPolicies) *AppEvaluationManager {
	manager := &AppEvaluationManager{
		evaluateInterval: evaluateInterval,
		logger:           logger.Session("AppEvaluationManager"),
		cclock:           cclock,
		doneChan:         make(chan bool),
		triggerChan:      triggerChan,
		triggers:         map[string][]*model.Trigger{},
		evaluatorArray:   []*Evaluator{},
		getPolicies:      getPolicies,
	}
	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = evaluatorCount
	for i := 0; i < evaluatorCount; i++ {
		evaluator := NewEvaluator(logger, client, scalingEngineUrl, triggerChan, database)
		manager.evaluatorArray = append(manager.evaluatorArray, evaluator)
	}
	return manager
}
func (a *AppEvaluationManager) getTriggers(policyMap map[string]*model.Policy) map[string][]*model.Trigger {
	if policyMap == nil {
		return nil
	}
	var triggerArrayMap map[string][]*model.Trigger = make(map[string][]*model.Trigger)
	for appId, policy := range policyMap {
		for _, rule := range policy.TriggerRecord.ScalingRules {
			triggerKey := appId + "#" + rule.MetricType
			triggerArray, exist := triggerArrayMap[triggerKey]
			if !exist {
				triggerArray = []*model.Trigger{}
			}
			triggerArray = append(triggerArray, &model.Trigger{
				AppId:            appId,
				MetricType:       rule.MetricType,
				BreachDuration:   rule.BreachDuration,
				CoolDownDuration: rule.CoolDownDuration,
				Threshold:        rule.Threshold,
				Operator:         rule.Operator,
				Adjustment:       rule.Adjustment,
			})
			triggerArrayMap[triggerKey] = triggerArray
		}
	}
	return triggerArrayMap
}
func (a *AppEvaluationManager) Start() {
	for _, evaluator := range a.evaluatorArray {
		evaluator.Start()
	}
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
			a.lock.Lock()
			a.triggers = a.getTriggers(a.getPolicies())
			a.lock.Unlock()
			for _, triggerArray := range a.triggers {
				a.triggerChan <- triggerArray
			}
		}
	}
	a.logger.Info("started")
}
