package operator

import (
	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type InstanceMetricsDbPruner struct {
	instanceMetricsDb db.InstanceMetricsDB
	cutoffDays        int
	clock             clock.Clock
	logger            lager.Logger
}

func NewInstanceMetricsDbPruner(instanceMetricsDb db.InstanceMetricsDB, cutoffDays int, clock clock.Clock, logger lager.Logger) *InstanceMetricsDbPruner {
	return &InstanceMetricsDbPruner{
		instanceMetricsDb: instanceMetricsDb,
		cutoffDays:        cutoffDays,
		clock:             clock,
		logger:            logger,
	}
}

func (idp InstanceMetricsDbPruner) Operate() {
	idp.logger.Debug("Pruning instance metrics")

	timestamp := idp.clock.Now().AddDate(0, 0, -idp.cutoffDays).UnixNano()

	err := idp.instanceMetricsDb.PruneInstanceMetrics(timestamp)
	if err != nil {
		idp.logger.Error("failed-prune-metrics", err)
		return
	}
}
