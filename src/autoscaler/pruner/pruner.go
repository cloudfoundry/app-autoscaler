package pruner

import (
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type DbPruner interface {
	Prune()
}

type DbPrunerRunner struct {
	dbPruner DbPruner
	interval time.Duration
	clock    clock.Clock
	logger   lager.Logger
}

func NewDbPrunerRunner(dbPruner DbPruner, interval time.Duration, clock clock.Clock, logger lager.Logger) *DbPrunerRunner {
	return &DbPrunerRunner{
		dbPruner: dbPruner,
		interval: interval,
		clock:    clock,
		logger:   logger,
	}
}

func (dpr *DbPrunerRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	ticker := dpr.clock.NewTicker(dpr.interval)

	dpr.logger.Info("started", lager.Data{"refresh_interval": dpr.interval})

	for {
		go dpr.dbPruner.Prune()
		select {
		case <-signals:
			ticker.Stop()
			dpr.logger.Info("stopped")
			return nil
		case <-ticker.C():
		}
	}
}
