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
	ticker     clock.Ticker
	doneChan   chan bool
}

func NewMetricsDBPruner(logger lager.Logger, metricsDB db.MetricsDB, interval int, cutoffDays int, clock clock.Clock) *MetricsDBPruner {
	return &MetricsDBPruner{
		logger:     logger,
		metricsDB:  metricsDB,
		interval:   time.Duration(interval) * time.Hour,
		cutoffDays: cutoffDays,
		clock:      clock,
		doneChan:   make(chan bool),
	}
}

func (mdp *MetricsDBPruner) Start() {
	mdp.ticker = mdp.clock.NewTicker(mdp.interval)
	go mdp.startMetricPrune()

	mdp.logger.Info("metrics-db-pruner-started", lager.Data{"interval_in_hours": mdp.interval})
}

func (mdp *MetricsDBPruner) Stop() {
	if mdp.ticker != nil {
		mdp.ticker.Stop()
		close(mdp.doneChan)
	}
	mdp.logger.Info("metrics-db-pruner-stopped")
}

func (mdp *MetricsDBPruner) startMetricPrune() {
	for {
		mdp.PruneOldMetrics()
		select {
		case <-mdp.doneChan:
			return
		case <-mdp.ticker.C():
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
