package generator

import (
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
	"github.com/cenkalti/backoff/v5"
	circuit "github.com/rubyist/circuitbreaker"
)

type ConsumeAppMonitorMap func(map[string][]*models.Trigger, chan []*models.Trigger)

type AppEvaluationManager struct {
	evaluateInterval time.Duration
	logger           lager.Logger
	emClock          clock.Clock
	doneChan         chan bool
	triggerChan      chan []*models.Trigger
	getPolicies      aggregator.GetPoliciesFunc
	breakerConfig    config.CircuitBreakerConfig
	breakers         map[string]*circuit.Breaker
	cooldownExpired  map[string]int64
	breakerLock      *sync.RWMutex
	cooldownLock     *sync.RWMutex
}

func NewAppEvaluationManager(logger lager.Logger, evaluateInterval time.Duration, emClock clock.Clock,
	triggerChan chan []*models.Trigger, getPolicies aggregator.GetPoliciesFunc,
	breakerConfig config.CircuitBreakerConfig) (*AppEvaluationManager, error) {
	return &AppEvaluationManager{
		evaluateInterval: evaluateInterval,
		logger:           logger.Session("AppEvaluationManager"),
		emClock:          emClock,
		doneChan:         make(chan bool),
		triggerChan:      triggerChan,
		getPolicies:      getPolicies,
		breakerConfig:    breakerConfig,
		cooldownExpired:  map[string]int64{},
		breakerLock:      &sync.RWMutex{},
		cooldownLock:     &sync.RWMutex{},
	}, nil
}

func (a *AppEvaluationManager) getTriggers(policyMap map[string]*models.AppPolicy) map[string][]*models.Trigger {
	if policyMap == nil {
		return nil
	}
	triggersByApp := make(map[string][]*models.Trigger)
	for appID, policy := range policyMap {
		now := a.emClock.Now().UnixNano()
		a.cooldownLock.RLock()
		cooldownExpiredAt, found := a.cooldownExpired[appID]
		a.cooldownLock.RUnlock()
		if found {
			if cooldownExpiredAt > now {
				continue
			}
		}
		triggers := []*models.Trigger{}
		for _, rule := range policy.ScalingPolicy.ScalingRules {
			triggers = append(triggers, &models.Trigger{
				AppId:                 appID,
				MetricType:            rule.MetricType,
				BreachDurationSeconds: rule.BreachDurationSeconds,
				CoolDownSeconds:       rule.CoolDownSeconds,
				Threshold:             rule.Threshold,
				Operator:              rule.Operator,
				Adjustment:            rule.Adjustment,
			})
		}
		triggersByApp[appID] = triggers
	}
	return triggersByApp
}

func (a *AppEvaluationManager) Start() {
	go a.doEvaluate()
	a.logger.Info("started")
}

func (a *AppEvaluationManager) Stop() {
	close(a.doneChan)
	a.logger.Info("stopped")
}

func (a *AppEvaluationManager) doEvaluate() {
	ticker := a.emClock.NewTicker(a.evaluateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.doneChan:
			return
		case <-ticker.C():
			policies := a.getPolicies()
			newBreakers := map[string]*circuit.Breaker{}
			for appID := range policies {
				cb, found := a.breakers[appID]
				if found {
					newBreakers[appID] = cb
				} else {
					bf := backoff.NewExponentialBackOff()
					bf.InitialInterval = a.breakerConfig.BackOffInitialInterval
					bf.MaxInterval = a.breakerConfig.BackOffMaxInterval
					bf.RandomizationFactor = 0 // do not randomize
					bf.Multiplier = 2
					bf.Reset()
					newBreakers[appID] = circuit.NewBreakerWithOptions(&circuit.Options{
						BackOff:    bf,
						ShouldTrip: circuit.ConsecutiveTripFunc(a.breakerConfig.ConsecutiveFailureCount),
					})
				}
			}

			a.breakerLock.Lock()
			a.breakers = newBreakers
			a.breakerLock.Unlock()

			triggers := a.getTriggers(policies)
			for _, triggerArray := range triggers {
				a.triggerChan <- triggerArray
			}
		}
	}
}

func (a *AppEvaluationManager) GetBreaker(appID string) *circuit.Breaker {
	a.breakerLock.RLock()
	defer a.breakerLock.RUnlock()
	return a.breakers[appID]
}

func (a *AppEvaluationManager) SetCoolDownExpired(appID string, expiredAt int64) {
	a.cooldownLock.Lock()
	defer a.cooldownLock.Unlock()
	a.cooldownExpired[appID] = expiredAt
}
