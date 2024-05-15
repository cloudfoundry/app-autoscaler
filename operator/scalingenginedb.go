package operator

import (
	"context"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type ScalingEngineDbPruner struct {
	scalingEngineDb                db.ScalingEngineDB
	scalingHistoriesCutoffDuration time.Duration
	clock                          clock.Clock
	logger                         lager.Logger
}

func NewScalingEngineDbPruner(scalingEngineDb db.ScalingEngineDB, scalingHistoriesCutoffDuration time.Duration, clock clock.Clock, logger lager.Logger) *ScalingEngineDbPruner {
	return &ScalingEngineDbPruner{
		scalingEngineDb:                scalingEngineDb,
		scalingHistoriesCutoffDuration: scalingHistoriesCutoffDuration,
		clock:                          clock,
		logger:                         logger.Session("scaling_engine_db_pruner"),
	}
}

func (sdp ScalingEngineDbPruner) Operate(ctx context.Context) {
	historyCutoffTimestamp := sdp.clock.Now().Add(-sdp.scalingHistoriesCutoffDuration).UnixNano()

	logger := sdp.logger.Session("pruning-scaling-histories-and-cooldowns", lager.Data{"history-cutoff-time": historyCutoffTimestamp})
	logger.Info("starting")
	defer logger.Info("completed")

	err := sdp.scalingEngineDb.PruneScalingHistories(ctx, historyCutoffTimestamp)
	if err != nil {
		sdp.logger.Error("failed-prune-scaling-histories", err)
	}

	err = sdp.scalingEngineDb.PruneCooldowns(ctx, sdp.clock.Now().UnixNano())
	if err != nil {
		sdp.logger.Error("failed-prune-scaling-cooldowns", err)
	}
}
