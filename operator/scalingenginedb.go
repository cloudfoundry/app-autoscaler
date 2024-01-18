package operator

import (
	"context"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type ScalingEngineDbPruner struct {
	scalingEngineDb db.ScalingEngineDB
	cutoffDuration  time.Duration
	clock           clock.Clock
	logger          lager.Logger
}

func NewScalingEngineDbPruner(scalingEngineDb db.ScalingEngineDB, cutoffDuration time.Duration, clock clock.Clock, logger lager.Logger) *ScalingEngineDbPruner {
	return &ScalingEngineDbPruner{
		scalingEngineDb: scalingEngineDb,
		cutoffDuration:  cutoffDuration,
		clock:           clock,
		logger:          logger,
	}
}

func (sdp ScalingEngineDbPruner) Operate(ctx context.Context) {
	sdp.logger.Debug("Pruning  scaling histories")

	timestamp := sdp.clock.Now().Add(-sdp.cutoffDuration).UnixNano()
	err := sdp.scalingEngineDb.PruneScalingHistories(ctx, timestamp)
	if err != nil {
		sdp.logger.Error("failed-prune-scaling-histories", err)
		return
	}
}
