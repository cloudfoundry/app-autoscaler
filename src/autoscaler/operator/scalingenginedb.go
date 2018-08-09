package operator

import (
	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type ScalingEngineDbPruner struct {
	scalingEngineDb db.ScalingEngineDB
	cutoffDays      int
	clock           clock.Clock
	logger          lager.Logger
}

func NewScalingEngineDbPruner(scalingEngineDb db.ScalingEngineDB, cutoffDays int, clock clock.Clock, logger lager.Logger) *ScalingEngineDbPruner {
	return &ScalingEngineDbPruner{
		scalingEngineDb: scalingEngineDb,
		cutoffDays:      cutoffDays,
		clock:           clock,
		logger:          logger,
	}
}

func (sdp ScalingEngineDbPruner) Operate() {
	sdp.logger.Debug("Pruning  scaling histories")

	timestamp := sdp.clock.Now().AddDate(0, 0, -sdp.cutoffDays).UnixNano()
	err := sdp.scalingEngineDb.PruneScalingHistories(timestamp)
	if err != nil {
		sdp.logger.Error("failed-prune-scaling-histories", err)
		return
	}
}
