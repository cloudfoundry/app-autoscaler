package aggregator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/generator"
	"autoscaler/eventgenerator/model"
	"net/http"
	"sync"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type Aggregator struct {
	logger                    lager.Logger
	doneChan                  chan bool
	appChan                   chan *model.AppMonitor
	metricPollerArray         []*MetricPoller
	appMetricDatabase         db.AppMetricDB
	evaluationManager         *generator.AppEvaluationManager
	cclock                    clock.Clock
	aggregatorExecuteInterval time.Duration
	appMonitorArray           []*model.AppMonitor
	lock                      sync.Mutex
	getPolicies               model.GetPolicies
}

func NewAggregator(logger lager.Logger, clock clock.Clock, aggregatorExecuteInterval time.Duration, policyPollerInterval time.Duration,
	policyDatabase db.PolicyDB, appMetricDatabase db.AppMetricDB, metricCollectorUrl string, metricPollerCount int,
	evaluationManager *generator.AppEvaluationManager, appMonitorChan chan *model.AppMonitor, getPolicies model.GetPolicies) *Aggregator {
	aggregator := &Aggregator{
		logger:            logger.Session("Aggregator"),
		doneChan:          make(chan bool),
		appChan:           appMonitorChan,
		metricPollerArray: []*MetricPoller{},
		appMetricDatabase: appMetricDatabase,
		evaluationManager: evaluationManager,
		cclock:            clock,
		aggregatorExecuteInterval: aggregatorExecuteInterval,
		appMonitorArray:           []*model.AppMonitor{},
		getPolicies:               getPolicies,
	}
	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = metricPollerCount

	var i int
	for i = 0; i < metricPollerCount; i++ {
		poller := NewMetricPoller(metricCollectorUrl, logger, aggregator.appChan, aggregator.ConsumeAppMetric, client)
		aggregator.metricPollerArray = append(aggregator.metricPollerArray, poller)
	}
	return aggregator
}

func (a *Aggregator) getAppMonitors(policyMap map[string]*model.Policy) []*model.AppMonitor {
	if policyMap == nil {
		return nil
	}
	var appMonitorArrayTmp = []*model.AppMonitor{}
	for appId, policy := range policyMap {
		for _, rule := range policy.TriggerRecord.ScalingRules {

			appMonitorArrayTmp = append(appMonitorArrayTmp, &model.AppMonitor{
				AppId:      appId,
				MetricType: rule.MetricType,
				StatWindow: rule.StatWindow,
			})
		}
	}
	return appMonitorArrayTmp
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
	for _, metricPoller := range a.metricPollerArray {
		metricPoller.Start()
	}
	go a.startWork()

	a.logger.Info("started")
}

func (a *Aggregator) Stop() {
	for _, metricPoller := range a.metricPollerArray {
		metricPoller.Stop()
	}
	close(a.doneChan)
	a.logger.Info("stopped")
}

func (a *Aggregator) startWork() {
	ticker := a.cclock.NewTicker(a.aggregatorExecuteInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.doneChan:
			return
		case <-ticker.C():
			a.lock.Lock()
			a.appMonitorArray = a.getAppMonitors(a.getPolicies())
			a.lock.Unlock()
			for _, appMonitorTmp := range a.appMonitorArray {
				a.appChan <- appMonitorTmp
			}
		}
	}
}
