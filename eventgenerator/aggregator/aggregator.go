package aggregator

import (
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
	getPolicies               models.GetPolicies
}

func NewAggregator(logger lager.Logger, clock clock.Clock, aggregatorExecuteInterval time.Duration,
	appMonitorChan chan *models.AppMonitor, getPolicies models.GetPolicies) (*Aggregator, error) {
	aggregator := &Aggregator{
		logger:   logger.Session("Aggregator"),
		doneChan: make(chan bool),
		appChan:  appMonitorChan,
		cclock:   clock,
		aggregatorExecuteInterval: aggregatorExecuteInterval,
		getPolicies:               getPolicies,
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
				StatWindow: rule.StatWindow(),
			})
		}
	}

	return appMonitors
}

func (a *Aggregator) Start() {
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
			for _, monitor := range appMonitors {
				a.appChan <- monitor
			}
		}
	}
}
