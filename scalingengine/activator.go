package scalingengine

import (
	"context"

	"code.cloudfoundry.org/lager/v3"
)

// Activator parks an application behind the autoscaler activator while it is
// scaled to zero and un-parks it once it is scaled back up. Parking binds the
// app's routes to the activator route-service so requests to a zero-instance
// app can trigger a scale-from-zero instead of failing.
//
// The scaling engine calls Park BEFORE scaling an app to zero so there is no
// window in which the app has no instances and no route coverage
// (bind -> confirm -> scale-to-0). See docs/design/scale-to-zero.md.
type Activator interface {
	Park(ctx context.Context, appId string) error
	Unpark(ctx context.Context, appId string) error
}

// NoopActivator is the default Activator. It does nothing, allowing the scaling
// engine to run standalone before the activator microservice is wired in.
type NoopActivator struct {
	logger lager.Logger
}

func NewNoopActivator(logger lager.Logger) *NoopActivator {
	return &NoopActivator{logger: logger.Session("noop-activator")}
}

func (a *NoopActivator) Park(_ context.Context, appId string) error {
	a.logger.Info("park-noop", lager.Data{"appId": appId})
	return nil
}

func (a *NoopActivator) Unpark(_ context.Context, appId string) error {
	a.logger.Info("unpark-noop", lager.Data{"appId": appId})
	return nil
}
