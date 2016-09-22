package pruner

import (
	"time"

	"autoscaler/db"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type MetricsDBPruner struct {
	logger     lager.Logger
	metricsDB  db.MetricsDB
	interval   time.Duration
	cutoffDays int
	clock      clock.Clock
	doneChan   chan bool
}

func NewMetricsDBPruner(logger lager.Logger, metricsDB db.MetricsDB, interval time.Duration, cutoffDays int, clock clock.Clock) *MetricsDBPruner {
	return &MetricsDBPruner{
		logger:     logger,
		metricsDB:  metricsDB,
		interval:   interval,
		cutoffDays: cutoffDays,
		clock:      clock,
		doneChan:   make(chan bool),
	}
}

func (mdp *MetricsDBPruner) Start() {
	go mdp.startMetricPrune()

	mdp.logger.Info("metrics-db-pruner-started", lager.Data{"interval_in_hours": mdp.interval})
}

func (mdp *MetricsDBPruner) Stop() {
	close(mdp.doneChan)
	mdp.logger.Info("metrics-db-pruner-stopped")
}

func (mdp *MetricsDBPruner) startMetricPrune() {
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

func (mdp *MetricsDBPruner) PruneOldMetrics() {
	mdp.logger.Debug("Prune metrics db data older than", lager.Data{"cutoff_days": mdp.cutoffDays})

	timestamp := mdp.clock.Now().AddDate(0, 0, -mdp.cutoffDays).UnixNano()

	err := mdp.metricsDB.PruneMetrics(timestamp)
	if err != nil {
		mdp.logger.Error("prune-metricsdb", err)
		return
	}
}
