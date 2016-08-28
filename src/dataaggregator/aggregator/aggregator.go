package aggregator

import (
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"dataaggregator/appmetric"
	"dataaggregator/db"
	"dataaggregator/policy"
	"net/http"
	"strconv"
	"time"
)

type Aggregator struct {
	logger             lager.Logger
	metricCollectorUrl string
	doneChan           chan bool
	appChan            chan *appmetric.AppMonitor
	policyPoller       *PolicyPoller
	metricPollerCount  int64
	metricPollerArray  []*MetricPoller
	database           db.DB
}

func NewAggregator(logger lager.Logger, clock clock.Clock, policyPollerInterval time.Duration, database db.DB, metricCollectorUrl string, metricPollerCount int64) *Aggregator {
	aggregator := &Aggregator{
		logger:             logger,
		metricCollectorUrl: metricCollectorUrl,
		doneChan:           make(chan bool),
		appChan:            make(chan *appmetric.AppMonitor, 10),
		metricPollerCount:  metricPollerCount,
		metricPollerArray:  []*MetricPoller{},
		database:           database,
	}
	aggregator.policyPoller = NewPolicyPoller(lager.NewLogger("policy-poller"), clock, policyPollerInterval, database, aggregator.ConsumeTrigger, aggregator.appChan)

	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = (int)(metricPollerCount)

	var i int64
	for i = 0; i < metricPollerCount; i++ {
		pollerLogger := lager.NewLogger("metric-poller-" + strconv.FormatInt(i, 10))
		poller := NewMetricPoller(metricCollectorUrl, pollerLogger, aggregator.appChan, aggregator.ConsumeAppMetric, client)
		aggregator.metricPollerArray = append(aggregator.metricPollerArray, poller)
	}
	return aggregator
}
func (a *Aggregator) ConsumeTrigger(triggerMap map[string]*policy.Trigger, appChan chan *appmetric.AppMonitor) {
	for appId, trigger := range triggerMap {
		for _, rule := range trigger.TriggerRecord.ScalingRules {
			appChan <- &appmetric.AppMonitor{
				AppId:          appId,
				MetricType:     rule.MetricType,
				StatWindowSecs: rule.StatWindowSecs,
			}
		}
	}
}
func (a *Aggregator) ConsumeAppMetric(appMetric *appmetric.AppMetric) {
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
	a.logger.Info("aggregator-started")
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
