package generator

import (
	"autoscaler/eventgenerator/config"
	"autoscaler/models"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cenk/backoff"
	"github.com/rubyist/circuitbreaker"
)

type ConsumeAppMonitorMap func(map[string][]*models.Trigger, chan []*models.Trigger)

type AppEvaluationManager struct {
	evaluateInterval time.Duration
	logger           lager.Logger
	emClock          clock.Clock
	doneChan         chan bool
	triggerChan      chan []*models.Trigger
	getPolicies      models.GetPolicies
	breakerConfig    config.CircuitBreakerConfig
	breakers         map[string]*circuit.Breaker
	cooldownExpired  map[string]int64
	breakerLock      *sync.RWMutex
	cooldownLock     *sync.Mutex
}

func NewAppEvaluationManager(logger lager.Logger, evaluateInterval time.Duration, emClock clock.Clock,
	triggerChan chan []*models.Trigger, getPolicies models.GetPolicies,
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
		cooldownLock:     &sync.Mutex{},
	}, nil
}

func (a *AppEvaluationManager) getTriggers(policyMap map[string]*models.AppPolicy) map[string][]*models.Trigger {
	if policyMap == nil {
		return nil
	}
	triggersByType := make(map[string][]*models.Trigger)
	now := a.emClock.Now().UnixNano()
	for appId, policy := range policyMap {
		for _, rule := range policy.ScalingPolicy.ScalingRules {
			a.cooldownLock.Lock()
			cooldownExpiredAt, found := a.cooldownExpired[appId]
			a.cooldownLock.Unlock()
			if found {
				if cooldownExpiredAt > now {
					continue
				}
			}
			triggerKey := appId + "#" + rule.MetricType
			triggers, exist := triggersByType[triggerKey]
			if !exist {
				triggers = []*models.Trigger{}
			}
			triggers = append(triggers, &models.Trigger{
				AppId:                 appId,
				MetricType:            rule.MetricType,
				BreachDurationSeconds: rule.BreachDurationSeconds,
				CoolDownSeconds:       rule.CoolDownSeconds,
				Threshold:             rule.Threshold,
				Operator:              rule.Operator,
				Adjustment:            rule.Adjustment,
			})
			triggersByType[triggerKey] = triggers
		}
	}
	return triggersByType
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
					bf.MaxElapsedTime = 0      // never stop retry
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
