package operator

import (
	"context"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type OperatorRunner struct {
	operator Operator
	interval time.Duration
	clock    clock.Clock
	logger   lager.Logger
}

func NewOperatorRunner(operator Operator, interval time.Duration, clock clock.Clock, logger lager.Logger) *OperatorRunner {
	return &OperatorRunner{
		operator: operator,
		interval: interval,
		clock:    clock,
		logger:   logger,
	}
}

func (opr *OperatorRunner) Run(ctx context.Context, ready chan<- struct{}) error {
	close(ready)
	ticker := opr.clock.NewTicker(opr.interval)

	opr.logger.Info("started", lager.Data{"refresh_interval": opr.interval})

	for {
		go opr.operator.Operate(context.Background()) //nolint:gosec // Intentionally not tied to runner ctx; operations should complete independently
		select {
		case <-ctx.Done():
			ticker.Stop()
			opr.logger.Info("stopped")
			return nil
		case <-ticker.C():
		}
	}
}
