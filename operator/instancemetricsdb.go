package operator

import (
	"context"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type InstanceMetricsDbPruner struct {
	instanceMetricsDb db.InstanceMetricsDB
	cutoffDuration    time.Duration
	clock             clock.Clock
	logger            lager.Logger
}

func NewInstanceMetricsDbPruner(instanceMetricsDb db.InstanceMetricsDB, cutoffDuration time.Duration, clock clock.Clock, logger lager.Logger) *InstanceMetricsDbPruner {
	return &InstanceMetricsDbPruner{
		instanceMetricsDb: instanceMetricsDb,
		cutoffDuration:    cutoffDuration,
		clock:             clock,
		logger:            logger,
	}
}

func (idp InstanceMetricsDbPruner) Operate(ctx context.Context) {
	idp.logger.Debug("Pruning instance metrics")

	timestamp := idp.clock.Now().Add(-idp.cutoffDuration).UnixNano()
	err := idp.instanceMetricsDb.PruneInstanceMetrics(ctx, timestamp)
	if err != nil {
		idp.logger.Error("failed-prune-metrics", err)
		return
	}
}
