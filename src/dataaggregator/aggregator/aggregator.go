package aggregator

import (
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"dataaggregator/appmetric"
	"dataaggregator/db"
	"dataaggregator/policy"
	"errors"
	"net/http"
	"time"
)

type Aggregator struct {
	logger             lager.Logger
	metricCollectorUrl string
	doneChan           chan bool
	appChan            chan *appmetric.AppMonitor
	policyPoller       *PolicyPoller
	metricPollerCount  int64
	MetricPollerArray  []*MetricPoller
	database           db.DB
}

func NewAggregator(logger lager.Logger, clock clock.Clock, policyPollerInterval time.Duration, database db.DB, metricCollectorUrl string, metricPollerCount int64) *Aggregator {
	aggregator := &Aggregator{
		logger:             logger.Session("Aggregator"),
		metricCollectorUrl: metricCollectorUrl,
		doneChan:           make(chan bool),
		appChan:            make(chan *appmetric.AppMonitor, 10),
		metricPollerCount:  metricPollerCount,
		MetricPollerArray:  []*MetricPoller{},
		database:           database,
	}
	aggregator.policyPoller = NewPolicyPoller(logger, clock, policyPollerInterval, database, aggregator.ConsumeTrigger, aggregator.appChan)

	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = (int)(metricPollerCount)

	var i int64
	for i = 0; i < metricPollerCount; i++ {
		poller := NewMetricPoller(metricCollectorUrl, logger, aggregator.appChan, aggregator.ConsumeAppMetric, client)
		aggregator.MetricPollerArray = append(aggregator.MetricPollerArray, poller)
	}
	return aggregator
}
func (a *Aggregator) ConsumeTrigger(triggerMap map[string]*policy.Trigger, appChan chan *appmetric.AppMonitor) {
	if triggerMap == nil {
		return
	}
	for appId, trigger := range triggerMap {
		for _, rule := range trigger.TriggerRecord.ScalingRules {
			appChan <- &appmetric.AppMonitor{
				AppId:      appId,
				MetricType: rule.MetricType,
				StatWindow: rule.StatWindow,
			}
		}
	}
}
func (a *Aggregator) ConsumeAppMetric(appMetric *appmetric.AppMetric) {
	if appMetric == nil {
		a.logger.Error("appmetric is nil", errors.New("Should not save a nil appmetric to database"))
		return
	}
	err := a.database.SaveAppMetric(appMetric)
	if err != nil {
		a.logger.Error("save appmetric to database failed", err, lager.Data{"appmetric": appMetric})
	}
}
func (a *Aggregator) Start() {
	a.policyPoller.Start()
	for _, metricPoller := range a.MetricPollerArray {
		metricPoller.Start()
	}
	a.logger.Info("started")
}
func (a *Aggregator) Stop() {
	a.policyPoller.Stop()
	for _, metricPoller := range a.MetricPollerArray {
		metricPoller.Stop()
	}
	a.logger.Info("stopped")
}
