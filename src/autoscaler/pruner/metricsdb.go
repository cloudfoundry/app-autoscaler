package pruner

import (
	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type MetricsDbPruner struct {
	metricsDb  db.MetricsDB
	cutoffDays int
	clock      clock.Clock
	logger     lager.Logger
}

func NewMetricsDbPruner(metricsDb db.MetricsDB, cutoffDays int, clock clock.Clock, logger lager.Logger) *MetricsDbPruner {
	return &MetricsDbPruner{
		metricsDb:  metricsDb,
		cutoffDays: cutoffDays,
		clock:      clock,
		logger:     logger.Session("metricsdbpruner"),
	}
}

func (mdp MetricsDbPruner) Prune() {
	mdp.logger.Debug("Pruning metrics", lager.Data{"cutoff_days": mdp.cutoffDays})

	timestamp := mdp.clock.Now().AddDate(0, 0, -mdp.cutoffDays).UnixNano()

	err := mdp.metricsDb.PruneMetrics(timestamp)
	if err != nil {
		mdp.logger.Error("failed-prune-metrics", err)
		return
	}
}
