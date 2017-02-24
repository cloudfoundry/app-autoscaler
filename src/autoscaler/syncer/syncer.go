package syncer

import (
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

type Syncer interface {
	Synchronize() error
}
type SyncerRunner struct {
	activeScheduleSyncer Syncer
	interval             time.Duration
	clock                clock.Clock
	logger               lager.Logger
}

func NewSyncerRunner(activeScheduleSyncer Syncer, interval time.Duration, clock clock.Clock, logger lager.Logger) *SyncerRunner {
	return &SyncerRunner{
		activeScheduleSyncer: activeScheduleSyncer,
		interval:             interval,
		clock:                clock,
		logger:               logger,
	}
}

func (sr *SyncerRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	ticker := sr.clock.NewTicker(sr.interval)

	sr.logger.Info("started", lager.Data{"synchronize_interval": sr.interval})

	for {
		go sr.activeScheduleSyncer.Synchronize()
		select {
		case <-signals:
			ticker.Stop()
			sr.logger.Info("stopped")
			return nil
		case <-ticker.C():
		}
	}
}
