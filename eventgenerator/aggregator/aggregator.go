package aggregator

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type Aggregator struct {
	logger                    lager.Logger
	doneChan                  chan bool
	appMonitorChan            chan *models.AppMonitor
	metricPollerArray         []*MetricPoller
	cclock                    clock.Clock
	aggregatorExecuteInterval time.Duration
	saveInterval              time.Duration
	getPolicies               GetPoliciesFunc
	saveAppMetricToCache      SaveAppMetricToCacheFunc
	defaultStatWindowSecs     int
	appMetricChan             chan *models.AppMetric
	appMetricDB               db.AppMetricDB
}

func NewAggregator(logger lager.Logger, clock clock.Clock, aggregatorExecuteInterval time.Duration, saveInterval time.Duration,
	appMonitorChan chan *models.AppMonitor, getPolicies GetPoliciesFunc, saveAppMetricToCache SaveAppMetricToCacheFunc, defaultStatWindowSecs int, appMetricChan chan *models.AppMetric, appMetricDB db.AppMetricDB) (*Aggregator, error) {
	aggregator := &Aggregator{
		logger:                    logger.Session("Aggregator"),
		doneChan:                  make(chan bool),
		appMonitorChan:            appMonitorChan,
		cclock:                    clock,
		aggregatorExecuteInterval: aggregatorExecuteInterval,
		saveInterval:              saveInterval,
		getPolicies:               getPolicies,
		saveAppMetricToCache:      saveAppMetricToCache,
		defaultStatWindowSecs:     defaultStatWindowSecs,
		appMetricChan:             appMetricChan,
		appMetricDB:               appMetricDB,
	}
	return aggregator, nil
}

func (a *Aggregator) getAppMonitors(policyMap map[string]*models.AppPolicy) map[string]*models.AppMonitor {
	if policyMap == nil {
		return nil
	}
	appMonitors := map[string]*models.AppMonitor{}
	for appID, appPolicy := range policyMap {
		for _, rule := range appPolicy.ScalingPolicy.ScalingRules {
			appMonitors[fmt.Sprintf("%s-%s", appID, rule.MetricType)] = &models.AppMonitor{
				AppId:      appID,
				MetricType: rule.MetricType,
				StatWindow: time.Second * time.Duration(a.defaultStatWindowSecs),
			}
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
			for _, ap := range appMonitors {
				a.appMonitorChan <- ap
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
			a.saveAppMetricToCache(appMetric)
		case <-ticker.C():
			if len(appMetricArray) > 0 {
				go func(appMetricDB db.AppMetricDB, metrics []*models.AppMetric) {
					_ = appMetricDB.SaveAppMetricsInBulk(metrics)
				}(a.appMetricDB, appMetricArray)
				appMetricArray = []*models.AppMetric{}
			}
		}
	}
}
