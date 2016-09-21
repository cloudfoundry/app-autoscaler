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
	evaluatorCount   int
	database         db.AppMetricDB
	scalingEngineUrl string
}

func NewAppEvaluationManager(evaluateInterval time.Duration, logger lager.Logger, cclock clock.Clock, triggerChan chan []*model.Trigger, evaluatorCount int, database db.AppMetricDB, scalingEngineUrl string) *AppEvaluationManager {
	manager := &AppEvaluationManager{
		evaluateInterval: evaluateInterval,
		logger:           logger.Session("AppEvaluationManager"),
		cclock:           cclock,
		doneChan:         make(chan bool),
		triggerChan:      triggerChan,
		triggers:         map[string][]*model.Trigger{},
		evaluatorArray:   []*Evaluator{},
		evaluatorCount:   evaluatorCount,
		database:         database,
		scalingEngineUrl: scalingEngineUrl,
	}
	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = evaluatorCount
	for i := 0; i < evaluatorCount; i++ {
		evaluator := NewEvaluator(logger, client, scalingEngineUrl, triggerChan, database)
		manager.evaluatorArray = append(manager.evaluatorArray, evaluator)
	}
	return manager
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
			a.evaluate()
		}
	}
	a.logger.Info("started")
}

func (a *AppEvaluationManager) SetTriggers(triggers map[string][]*model.Trigger) {
	a.lock.Lock()
	a.triggers = triggers
	a.lock.Unlock()
}

func (a *AppEvaluationManager) evaluate() {
	a.lock.Lock()
	triggers := a.triggers
	a.lock.Unlock()
	for _, triggerArray := range triggers {
		a.triggerChan <- triggerArray
	}
}
