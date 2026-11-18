package appstate

import (
	"context"
	"os"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
)

// Runner supervises a recoverable polling task. Shutdown waits for cancellation
// so the task cannot outlive its database or transport.
func Runner(logger lager.Logger, interval time.Duration, task func(context.Context) error) ifrit.Runner {
	return ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				if err := task(ctx); err != nil && ctx.Err() == nil {
					logger.Error("poll-failed-retrying", err)
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
		close(ready)
		<-signals
		cancel()
		<-done
		return nil
	})
}
