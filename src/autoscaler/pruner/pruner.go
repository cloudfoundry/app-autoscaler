package pruner

import (
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type Pruner interface {
	PruneOldData()
}

type DbPrunerRunner struct {
	pruner   Pruner
	name     string
	interval time.Duration
	clock    clock.Clock
	doneChan chan bool
	logger   lager.Logger
}

func NewDbPrunerRunner(pruner Pruner, name string, interval time.Duration, clock clock.Clock, logger lager.Logger) *DbPrunerRunner {
	return &DbPrunerRunner{
		pruner:   pruner,
		name:     name,
		interval: interval,
		clock:    clock,
		logger:   logger,
		doneChan: make(chan bool),
	}
}

func (dpr *DbPrunerRunner) Start() {
	go dpr.startPrune()

	dpr.logger.Info(dpr.name+"-pruner-started", lager.Data{"refresh_interval_in_hours": dpr.interval})
}

func (dpr *DbPrunerRunner) Stop() {
	close(dpr.doneChan)
	dpr.logger.Info(dpr.name + "-pruner-stopped")
}

func (dpr *DbPrunerRunner) startPrune() {
	ticker := dpr.clock.NewTicker(dpr.interval)

	for {
		dpr.pruner.PruneOldData()
		select {
		case <-dpr.doneChan:
			ticker.Stop()
			return
		case <-ticker.C():
		}
	}
}
