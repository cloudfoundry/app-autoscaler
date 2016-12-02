package generator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/model"
	"autoscaler/models"
	"net/http"
	"time"

	"code.cloudfoundry.org/cfhttp"
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
	evaluatorArray   []*Evaluator
	getPolicies      model.GetPolicies
}

func NewAppEvaluationManager(evaluateInterval time.Duration, logger lager.Logger, cclock clock.Clock,
	triggerChan chan []*model.Trigger, evaluatorCount int, database db.AppMetricDB,
	scalingEngineUrl string, getPolicies model.GetPolicies, tlsCerts *models.TLSCerts) (*AppEvaluationManager, error) {
	manager := &AppEvaluationManager{
		evaluateInterval: evaluateInterval,
		logger:           logger.Session("AppEvaluationManager"),
		cclock:           cclock,
		doneChan:         make(chan bool),
		triggerChan:      triggerChan,
		evaluatorArray:   []*Evaluator{},
		getPolicies:      getPolicies,
	}

	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = evaluatorCount
	if tlsCerts != nil {
		tlsConfig, err := cfhttp.NewTLSConfig(tlsCerts.CertFile, tlsCerts.KeyFile, tlsCerts.CACertFile)
		if err != nil {
			return nil, err
		}
		client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
	}

	for i := 0; i < evaluatorCount; i++ {
		evaluator := NewEvaluator(logger, client, scalingEngineUrl, triggerChan, database)
		manager.evaluatorArray = append(manager.evaluatorArray, evaluator)
	}
	return manager, nil
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
			triggers := a.getTriggers(a.getPolicies())
			for _, triggerArray := range triggers {
				a.triggerChan <- triggerArray
			}
		}
	}
	a.logger.Info("started")
}
