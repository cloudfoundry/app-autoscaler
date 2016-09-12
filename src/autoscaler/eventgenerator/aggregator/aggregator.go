package aggregator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/model"
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"net/http"
	"time"
)

type Aggregator struct {
	logger             lager.Logger
	metricCollectorUrl string
	doneChan           chan bool
	appChan            chan *model.AppMonitor
	policyPoller       *PolicyPoller
	metricPollerCount  int
	metricPollerArray  []*MetricPoller
	policyDatabase     db.PolicyDB
	appMetricDatabase  db.AppMetricDB
	evaluationManager  *AppEvaluationManager
}

func NewAggregator(logger lager.Logger, clock clock.Clock, policyPollerInterval time.Duration, policyDatabase db.PolicyDB, appMetricDatabase db.AppMetricDB, metricCollectorUrl string, metricPollerCount int, evaluationManager *AppEvaluationManager) *Aggregator {
	aggregator := &Aggregator{
		logger:             logger.Session("Aggregator"),
		metricCollectorUrl: metricCollectorUrl,
		doneChan:           make(chan bool),
		appChan:            make(chan *model.AppMonitor, 1024),
		metricPollerCount:  metricPollerCount,
		metricPollerArray:  []*MetricPoller{},
		policyDatabase:     policyDatabase,
		appMetricDatabase:  appMetricDatabase,
		evaluationManager:  evaluationManager,
	}
	aggregator.policyPoller = NewPolicyPoller(logger, clock, policyPollerInterval, policyDatabase, aggregator.ConsumePolicy, aggregator.appChan)
	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = metricPollerCount

	var i int
	for i = 0; i < metricPollerCount; i++ {
		poller := NewMetricPoller(metricCollectorUrl, logger, aggregator.appChan, aggregator.ConsumeAppMetric, client)
		aggregator.metricPollerArray = append(aggregator.metricPollerArray, poller)
	}
	return aggregator
}
func (a *Aggregator) ConsumePolicy(policyMap map[string]*model.Policy, appChan chan *model.AppMonitor) {
	if policyMap == nil {
		return
	}
	var triggerArrayMap map[string][]*model.Trigger = make(map[string][]*model.Trigger)
	for appId, policy := range policyMap {
		for _, rule := range policy.TriggerRecord.ScalingRules {
			appChan <- &model.AppMonitor{
				AppId:      appId,
				MetricType: rule.MetricType,
				StatWindow: rule.StatWindow,
			}
			_, exist := triggerArrayMap[appId+"#"+rule.MetricType]
			if !exist {
				triggerArrayMap[appId+"#"+rule.MetricType] = []*model.Trigger{}
			}
			triggerArrayMap[appId+"#"+rule.MetricType] = append(triggerArrayMap[appId+"#"+rule.MetricType], &model.Trigger{
				AppId:            appId,
				MetricType:       rule.MetricType,
				BreachDuration:   rule.BreachDuration,
				CoolDownDuration: rule.CoolDownDuration,
				Threshold:        rule.Threshold,
				Operator:         rule.Operator,
				Adjustment:       rule.Adjustment,
			})

		}
	}
	a.evaluationManager.SetTriggers(triggerArrayMap)
}
func (a *Aggregator) ConsumeAppMetric(appMetric *model.AppMetric) {
	if appMetric == nil {
		return
	}
	err := a.appMetricDatabase.SaveAppMetric(appMetric)
	if err != nil {
		a.logger.Error("save appmetric to database failed", err, lager.Data{"appmetric": appMetric})
	}
}
func (a *Aggregator) Start() {
	a.policyPoller.Start()
	for _, metricPoller := range a.metricPollerArray {
		metricPoller.Start()
	}
	a.logger.Info("started")
}
func (a *Aggregator) Stop() {
	a.policyPoller.Stop()
	for _, metricPoller := range a.metricPollerArray {
		metricPoller.Stop()
	}
	a.logger.Info("stopped")
}
