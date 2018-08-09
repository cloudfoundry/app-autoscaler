package operator

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

func (amdp AppMetricsDbPruner) Operate() {
	amdp.logger.Debug("Pruning app metrics")

	timestamp := amdp.clock.Now().AddDate(0, 0, -amdp.cutoffDays).UnixNano()

	err := amdp.appMetricsDb.PruneAppMetrics(timestamp)
	if err != nil {
		amdp.logger.Error("failed-prune-appmetrics", err)
		return
	}

}
