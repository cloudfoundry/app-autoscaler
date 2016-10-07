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
	name     string
	interval time.Duration
	clock    clock.Clock
	logger   lager.Logger
}

func NewDbPrunerRunner(dbPruner DbPruner, name string, interval time.Duration, clock clock.Clock, logger lager.Logger) *DbPrunerRunner {
	return &DbPrunerRunner{
		dbPruner: dbPruner,
		name:     name,
		interval: interval,
		clock:    clock,
		logger:   logger,
	}
}

func (dpr *DbPrunerRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	ticker := dpr.clock.NewTicker(dpr.interval)

	dpr.logger.Info(dpr.name+"-started", lager.Data{"refresh_interval": dpr.interval})

	for {
		go dpr.dbPruner.Prune()
		select {
		case <-signals:
			ticker.Stop()
			dpr.logger.Info(dpr.name + "-stopped")
			return nil
		case <-ticker.C():
		}
	}
}
