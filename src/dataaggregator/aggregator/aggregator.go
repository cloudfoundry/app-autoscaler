package aggregator

import (
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"dataaggregator/appmetric"
	"dataaggregator/db"
	"dataaggregator/policy"
	"strconv"
	"time"
)

type Aggregator struct {
	logger             lager.Logger
	metricCollectorUrl string
	doneChan           chan bool
	appChan            chan *appmetric.AppMonitor
	metricChan         chan *appmetric.AppMetric
	policyPoller       *PolicyPoller
	metricPollerCount  int64
	metricPollerArray  []*MetricPoller
	database           db.DB
}

func NewAggregator(logger lager.Logger, clock clock.Clock, policyPollerInterval time.Duration, database db.DB, metricCollectorUrl string, metricPollerCount int64) *Aggregator {
	aggregator := &Aggregator{
		logger:             logger.Session("aggregator"),
		metricCollectorUrl: metricCollectorUrl,
		doneChan:           make(chan bool),
		appChan:            make(chan *appmetric.AppMonitor, 10),
		metricChan:         make(chan *appmetric.AppMetric, 10),
		metricPollerCount:  metricPollerCount,
		metricPollerArray:  []*MetricPoller{},
		database:           database,
	}
	aggregator.policyPoller = NewPolicyPoller(logger, clock, policyPollerInterval, database, aggregator.consumeTrigger)
	var i int64
	for i = 0; i < metricPollerCount; i++ {
		pollerLogger := lager.NewLogger("Metric-Poller-" + strconv.FormatInt(i, 10))
		poller := NewMetricPoller(metricCollectorUrl, pollerLogger, aggregator.appChan, aggregator.consumeAppMetric, cfhttp.NewClient())
		aggregator.metricPollerArray = append(aggregator.metricPollerArray, poller)
	}
	return aggregator
}
func (a *Aggregator) consumeTrigger(policyList []*policy.PolicyJson, triggerMap map[string]*policy.Trigger) {
	for appId, trigger := range triggerMap {
		for _, rule := range trigger.TriggerRecord.ScalingRules {
			a.appChan <- &appmetric.AppMonitor{
				AppId:          appId,
				MetricType:     rule.MetricType,
				StatWindowSecs: rule.StatWindowSecs,
			}
		}
	}
}
func (a *Aggregator) consumeAppMetric(appMetric *appmetric.AppMetric) {
	a.metricChan <- appMetric
	err := a.database.SaveAppMetric(appMetric)
	if err != nil {
		a.logger.Error("save appmetric to database failed", err, lager.Data{"appmetric": appMetric})
	}
}
func (a *Aggregator) Start() {
	a.policyPoller.Start()
	for _, metricPoller := range a.metricPollerArray {
		metricPoller.Start()
	}
	a.logger.Info("policy-poller-started")
}
func (a *Aggregator) Stop() {
	a.policyPoller.Stop()
	for _, metricPoller := range a.metricPollerArray {
		metricPoller.Stop()
	}
	// close(a.appChan)
	// close(a.metricChan)
	a.logger.Info("aggregator-stopped")
}
