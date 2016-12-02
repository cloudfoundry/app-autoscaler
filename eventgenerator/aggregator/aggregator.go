package aggregator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/generator"
	"autoscaler/eventgenerator/model"
	"autoscaler/models"
	"net/http"
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
	getPolicies               model.GetPolicies
}

func NewAggregator(logger lager.Logger, clock clock.Clock, aggregatorExecuteInterval time.Duration,
	policyPollerInterval time.Duration, policyDatabase db.PolicyDB, appMetricDatabase db.AppMetricDB,
	metricCollectorUrl string, metricPollerCount int, evaluationManager *generator.AppEvaluationManager,
	appMonitorChan chan *model.AppMonitor, getPolicies model.GetPolicies, tlsCerts *models.TLSCerts) (*Aggregator, error) {
	aggregator := &Aggregator{
		logger:            logger.Session("Aggregator"),
		doneChan:          make(chan bool),
		appChan:           appMonitorChan,
		metricPollerArray: []*MetricPoller{},
		appMetricDatabase: appMetricDatabase,
		evaluationManager: evaluationManager,
		cclock:            clock,
		aggregatorExecuteInterval: aggregatorExecuteInterval,
		getPolicies:               getPolicies,
	}
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

func (a *Aggregator) getAppMonitors(policyMap map[string]*model.Policy) []*model.AppMonitor {
	if policyMap == nil {
		return nil
	}
	appMonitors := make([]*model.AppMonitor, 0, len(policyMap))
	for appId, policy := range policyMap {
		for _, rule := range policy.TriggerRecord.ScalingRules {

			appMonitors = append(appMonitors, &model.AppMonitor{
				AppId:      appId,
				MetricType: rule.MetricType,
				StatWindow: rule.StatWindow,
			})
		}
	}

	return appMonitors
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
			appMonitors := a.getAppMonitors(a.getPolicies())
			for _, appMonitorTmp := range appMonitors {
				a.appChan <- appMonitorTmp
			}
		}
	}
}
