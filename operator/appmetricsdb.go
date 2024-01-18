package operator

import (
	"context"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type AppMetricsDbPruner struct {
	appMetricsDb   db.AppMetricDB
	cutoffDuration time.Duration
	clock          clock.Clock
	logger         lager.Logger
}

func NewAppMetricsDbPruner(appMetricsDb db.AppMetricDB, cutoffDuration time.Duration, clock clock.Clock, logger lager.Logger) *AppMetricsDbPruner {
	return &AppMetricsDbPruner{
		appMetricsDb:   appMetricsDb,
		cutoffDuration: cutoffDuration,
		clock:          clock,
		logger:         logger,
	}
}

func (amdp AppMetricsDbPruner) Operate(ctx context.Context) {
	amdp.logger.Debug("Pruning app metrics")

	timestamp := amdp.clock.Now().Add(-amdp.cutoffDuration).UnixNano()

	err := amdp.appMetricsDb.PruneAppMetrics(ctx, timestamp)
	if err != nil {
		amdp.logger.Error("failed-prune-appmetrics", err)
		return
	}
}
