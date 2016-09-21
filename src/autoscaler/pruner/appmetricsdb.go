package pruner

import (
	"time"

	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type AppMetricsDbPruner struct {
	logger       lager.Logger
	appMetricsDb db.AppMetricDB
	interval     time.Duration
	cutoffDays   int
	clock        clock.Clock
	doneChan     chan bool
}

func NewAppMetricsDbPruner(logger lager.Logger, appMetricsDb db.AppMetricDB, interval time.Duration, cutoffDays int, clock clock.Clock) *AppMetricsDbPruner {
	return &AppMetricsDbPruner{
		logger:       logger,
		appMetricsDb: appMetricsDb,
		interval:     interval,
		cutoffDays:   cutoffDays,
		clock:        clock,
		doneChan:     make(chan bool),
	}
}

func (amdp *AppMetricsDbPruner) Start() {
	go amdp.startMetricPrune()

	amdp.logger.Info("app-metrics-db-pruner-started", lager.Data{"refresh_interval_in_hours": amdp.interval})
}

func (amdp *AppMetricsDbPruner) Stop() {
	close(amdp.doneChan)
	amdp.logger.Info("app-metrics-db-pruner-stopped")
}

func (amdp *AppMetricsDbPruner) startMetricPrune() {
	ticker := amdp.clock.NewTicker(amdp.interval)
	for {
		amdp.PruneOldMetrics()
		select {
		case <-amdp.doneChan:
			ticker.Stop()
			return
		case <-ticker.C():
		}
	}
}

func (amdp *AppMetricsDbPruner) PruneOldMetrics() {
	amdp.logger.Debug("Prune app metric db data older than", lager.Data{"cutoff_days": amdp.cutoffDays})

	timestamp := amdp.clock.Now().AddDate(0, 0, -amdp.cutoffDays).UnixNano()

	err := amdp.appMetricsDb.PruneAppMetrics(timestamp)
	if err != nil {
		amdp.logger.Error("prune-appmetricsdb", err)
		return
	}
}
