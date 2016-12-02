package aggregator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/generator"
	"autoscaler/eventgenerator/model"
	"autoscaler/models"
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
	policyPoller              *PolicyPoller
	metricPollerArray         []*MetricPoller
	appMetricDatabase         db.AppMetricDB
	evaluationManager         *generator.AppEvaluationManager
	cclock                    clock.Clock
	aggregatorExecuteInterval time.Duration
	appMonitorArray           []*model.AppMonitor
	lock                      sync.Mutex
}

func NewAggregator(logger lager.Logger, clock clock.Clock, aggregatorExecuteInterval time.Duration,
	policyPollerInterval time.Duration, policyDatabase db.PolicyDB, appMetricDatabase db.AppMetricDB,
	metricCollectorUrl string, metricPollerCount int, evaluationManager *generator.AppEvaluationManager,
	appMonitorChan chan *model.AppMonitor, tlsCerts *models.TLSCerts) (*Aggregator, error) {

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
	}
	aggregator.policyPoller = NewPolicyPoller(logger, clock, policyPollerInterval, policyDatabase, aggregator.ConsumePolicy, aggregator.appChan)
	client := cfhttp.NewClient()
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = metricPollerCount
	if tlsCerts != nil {
		tlsConfig, err := cfhttp.NewTLSConfig(tlsCerts.CertFile, tlsCerts.KeyFile, tlsCerts.CACertFile)
		if err != nil {
			return nil, err
		}
		client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
	}

	var i int
	for i = 0; i < metricPollerCount; i++ {
		poller := NewMetricPoller(metricCollectorUrl, logger, aggregator.appChan, aggregator.ConsumeAppMetric, client)
		aggregator.metricPollerArray = append(aggregator.metricPollerArray, poller)
	}
	return aggregator, nil
}

func (a *Aggregator) ConsumePolicy(policyMap map[string]*model.Policy, appChan chan *model.AppMonitor) {
	if policyMap == nil {
		return
	}
	var triggerArrayMap map[string][]*model.Trigger = make(map[string][]*model.Trigger)
	var appMonitorArrayTmp = []*model.AppMonitor{}

	for appId, policy := range policyMap {
		for _, rule := range policy.TriggerRecord.ScalingRules {

			appMonitorArrayTmp = append(appMonitorArrayTmp, &model.AppMonitor{
				AppId:      appId,
				MetricType: rule.MetricType,
				StatWindow: rule.StatWindow,
			})

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

	a.setAppMonitors(appMonitorArrayTmp)
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
	go a.startWork()

	a.logger.Info("started")
}

func (a *Aggregator) Stop() {
	a.policyPoller.Stop()
	for _, metricPoller := range a.metricPollerArray {
		metricPoller.Stop()
	}
	close(a.doneChan)
	a.logger.Info("stopped")
}

func (a *Aggregator) setAppMonitors(appMonitors []*model.AppMonitor) {
	a.lock.Lock()
	a.appMonitorArray = appMonitors
	a.lock.Unlock()
}

func (a *Aggregator) startWork() {
	ticker := a.cclock.NewTicker(a.aggregatorExecuteInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.doneChan:
			return
		case <-ticker.C():
			a.addToAggregateChannel()
		}
	}
}

func (a *Aggregator) addToAggregateChannel() {
	a.lock.Lock()
	appMonitors := a.appMonitorArray
	a.lock.Unlock()

	for _, appMonitorTmp := range appMonitors {
		a.appChan <- appMonitorTmp
	}
}
