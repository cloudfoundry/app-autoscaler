package pruner

import (
	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type AppMetricsDbPruner struct {
	appMetricsDb db.AppMetricDB
	cutoffDays   int
	clock        clock.Clock
	logger       lager.Logger
}

func NewAppMetricsDbPruner(appMetricsDb db.AppMetricDB, cutoffDays int, clock clock.Clock, logger lager.Logger) *AppMetricsDbPruner {
	return &AppMetricsDbPruner{

		appMetricsDb: appMetricsDb,
		cutoffDays:   cutoffDays,
		clock:        clock,
		logger:       logger,
	}
}

func (amdp AppMetricsDbPruner) PruneOldData() {
	amdp.logger.Debug("Prune app metric db data older than", lager.Data{"cutoff_days": amdp.cutoffDays})

	timestamp := amdp.clock.Now().AddDate(0, 0, -amdp.cutoffDays).UnixNano()

	err := amdp.appMetricsDb.PruneAppMetrics(timestamp)
	if err != nil {
		amdp.logger.Error("prune-appmetricsdb", err)
		return
	}

}
