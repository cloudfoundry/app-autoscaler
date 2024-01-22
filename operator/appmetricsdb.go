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
		logger:         logger.Session("app_metrics_db_pruner"),
	}
}

func (amdp AppMetricsDbPruner) Operate(ctx context.Context) {
	timestamp := amdp.clock.Now().Add(-amdp.cutoffDuration).UnixNano()

	logger := amdp.logger.Session("pruning-app-metrics", lager.Data{"cutoff-time": timestamp})
	logger.Info("starting")
	defer logger.Info("completed")

	err := amdp.appMetricsDb.PruneAppMetrics(ctx, timestamp)
	if err != nil {
		amdp.logger.Error("failed-prune-appmetrics", err)
		return
	}
}
