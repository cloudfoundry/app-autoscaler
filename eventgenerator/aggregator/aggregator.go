package aggregator

import (
	"autoscaler/db"
	"autoscaler/models"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type Aggregator struct {
	logger                    lager.Logger
	doneChan                  chan bool
	appChan                   chan *models.AppMonitor
	metricPollerArray         []*MetricPoller
	cclock                    clock.Clock
	aggregatorExecuteInterval time.Duration
	saveInterval              time.Duration
	getPolicies               models.GetPolicies
	defaultStatWindowSecs     int
	appMetricChan             chan *models.AppMetric
	appMetricDB               db.AppMetricDB
}

func NewAggregator(logger lager.Logger, clock clock.Clock, aggregatorExecuteInterval time.Duration, saveInterval time.Duration,
	appMonitorChan chan *models.AppMonitor, getPolicies models.GetPolicies, defaultStatWindowSecs int, appMetricChan chan *models.AppMetric, appMetricDB db.AppMetricDB) (*Aggregator, error) {
	aggregator := &Aggregator{
		logger:   logger.Session("Aggregator"),
		doneChan: make(chan bool),
		appChan:  appMonitorChan,
		cclock:   clock,
		aggregatorExecuteInterval: aggregatorExecuteInterval,
		saveInterval:              saveInterval,
		getPolicies:               getPolicies,
		defaultStatWindowSecs:     defaultStatWindowSecs,
		appMetricChan:             appMetricChan,
		appMetricDB:               appMetricDB,
	}
	return aggregator, nil
}

func (a *Aggregator) getAppMonitors(policyMap map[string]*models.AppPolicy) []*models.AppMonitor {
	if policyMap == nil {
		return nil
	}
	appMonitors := make([]*models.AppMonitor, 0, len(policyMap))
	for appId, appPolicy := range policyMap {
		for _, rule := range appPolicy.ScalingPolicy.ScalingRules {
			appMonitors = append(appMonitors, &models.AppMonitor{
				AppId:      appId,
				MetricType: rule.MetricType,
				StatWindow: rule.StatWindow(a.defaultStatWindowSecs),
			})
		}
	}

	return appMonitors
}

func (a *Aggregator) Start() {
	go a.startAggregating()
	go a.startSavingAppMetric()

	a.logger.Info("started")
}

func (a *Aggregator) Stop() {
	for _, metricPoller := range a.metricPollerArray {
		metricPoller.Stop()
	}
	close(a.doneChan)
	a.logger.Info("stopped")
}

func (a *Aggregator) startAggregating() {
	ticker := a.cclock.NewTicker(a.aggregatorExecuteInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.doneChan:
			return
		case <-ticker.C():
			appMonitors := a.getAppMonitors(a.getPolicies())
			for _, monitor := range appMonitors {
				a.appChan <- monitor
			}
		}
	}
}

func (a *Aggregator) startSavingAppMetric() {
	appMetricArray := []*models.AppMetric{}
	ticker := a.cclock.NewTicker(a.saveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.doneChan:
			return
		case appMetric := <-a.appMetricChan:
			appMetricArray = append(appMetricArray, appMetric)
		case <-ticker.C():
			go func(appMetricDB db.AppMetricDB, metrics []*models.AppMetric) {
				appMetricDB.SaveAppMetricsInBulk(metrics)
				return
			}(a.appMetricDB, appMetricArray)
			appMetricArray = []*models.AppMetric{}
		}
	}
}
