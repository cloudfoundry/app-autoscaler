package operator

import (
	"autoscaler/db"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
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

func (sdp ScalingEngineDbPruner) Operate() {
	sdp.logger.Debug("Pruning  scaling histories")

	timestamp := sdp.clock.Now().Add(-sdp.cutoffDuration).UnixNano()
	err := sdp.scalingEngineDb.PruneScalingHistories(timestamp)
	if err != nil {
		sdp.logger.Error("failed-prune-scaling-histories", err)
		return
	}
}
