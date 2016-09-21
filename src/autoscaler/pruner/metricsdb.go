package pruner

import (
	"time"

	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type MetricsDbPruner struct {
	logger     lager.Logger
	metricsDb  db.MetricsDB
	interval   time.Duration
	cutoffDays int
	clock      clock.Clock
	doneChan   chan bool
}

func NewMetricsDbPruner(logger lager.Logger, metricsDb db.MetricsDB, interval time.Duration, cutoffDays int, clock clock.Clock) *MetricsDbPruner {
	return &MetricsDbPruner{
		logger:     logger,
		metricsDb:  metricsDb,
		interval:   interval,
		cutoffDays: cutoffDays,
		clock:      clock,
		doneChan:   make(chan bool),
	}
}

func (mdp *MetricsDbPruner) Start() {
	go mdp.startMetricPrune()

	mdp.logger.Info("metrics-db-pruner-started", lager.Data{"refresh_interval_in_hours": mdp.interval})
}

func (mdp *MetricsDbPruner) Stop() {
	close(mdp.doneChan)
	mdp.logger.Info("metrics-db-pruner-stopped")
}

func (mdp *MetricsDbPruner) startMetricPrune() {
	ticker := mdp.clock.NewTicker(mdp.interval)

	for {
		mdp.PruneOldMetrics()
		select {
		case <-mdp.doneChan:
			ticker.Stop()
			return
		case <-ticker.C():
		}
	}
}

func (mdp *MetricsDbPruner) PruneOldMetrics() {
	mdp.logger.Debug("Prune metrics db data older than", lager.Data{"cutoff_days": mdp.cutoffDays})

	timestamp := mdp.clock.Now().AddDate(0, 0, -mdp.cutoffDays).UnixNano()

	err := mdp.metricsDb.PruneMetrics(timestamp)
	if err != nil {
		mdp.logger.Error("prune-metricsdb", err)
		return
	}
}
