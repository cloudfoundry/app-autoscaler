// Package activator implements scale-from-zero for Cloud Foundry apps parked by
// the autoscaler. It is the counterpart to the scaling engine's Activator seam
// (see scalingengine/activator.go and docs/design/scale-to-zero.md).
//
// While an app is parked (scaled to zero), the activator registers itself as an
// mTLS backend for the app's routes on the Gorouter NATS bus (router.register),
// keeping the routes alive so Gorouter forwards requests to the activator
// instead of returning 404 for the zero-instance app. It then runs two decoupled
// loops:
//
//   - Loop A (request handling): hold the incoming request, ask the scaling
//     engine to scale the app to 1, wait for the app's readiness signal, then
//     forward the held request to the app's real route so Gorouter delivers it
//     to the now-live app. On timeout, return 503 + Retry-After.
//
//   - Loop B (unpark): driven by the same NATS bus. When route-emitter registers
//     an app's real backend for a route the activator has parked (from any cause
//     — request-driven, scheduled, or manual `cf scale`), the activator
//     deregisters its own backend for that route and releases held requests. A
//     periodic reconcile against CF is the safety net for missed events /
//     activator restarts.
//
// Park/Unpark drive the NATS registration; the readiness signal is a
// router.register for a parked route from a backend other than the activator.
package activator

import (
	"context"

	"code.cloudfoundry.org/lager/v3"
)

// Parker registers and unregisters an app's routes with the activator as their
// mTLS backend on the NATS bus. Park is the operation the scaling engine's
// Activator.Park ultimately drives (register -> confirm -> scale-to-0). Unpark
// is invoked by Loop B once an app is observed to be up again.
type Parker interface {
	Park(ctx context.Context, appID string) error
	Unpark(ctx context.Context, appID string) error
}

// Registry tracks which apps the activator currently has parked, and their
// bound routes. It must be reconstructable from CF (via reconcile) rather than
// relying solely on in-memory state, so an activator restart mid-park recovers.
type Registry interface {
	IsParked(appID string) bool
	MarkParked(appID string, routes []string)
	MarkUnparked(appID string)
	ParkedApps() []string
	// RoutesFor returns the route URLs the activator bound for a parked app.
	RoutesFor(appID string) []string
}

// ReadinessWatcher is Loop B: it consumes route-registration (Upsert) events
// from the NATS bus and signals when a parked app's real endpoint has come up
// (a router.register from a backend other than the activator).
type ReadinessWatcher interface {
	// WaitForReady returns a channel that receives nil when the app is observed
	// ready, or ctx.Err() if ctx is done first. Register the waiter before
	// triggering the scale-up to avoid missing a fast registration.
	WaitForReady(ctx context.Context, appID string) <-chan error
	// Run consumes the event stream until ctx is done, un-parking apps whose
	// registration arrives regardless of what caused the scale-up.
	Run(ctx context.Context) error
}

// noopParker is the scaffold Parker. It logs and succeeds so the service runs
// end-to-end before route-service binding is implemented.
type noopParker struct {
	logger lager.Logger
}

func NewNoopParker(logger lager.Logger) Parker {
	return &noopParker{logger: logger.Session("noop-parker")}
}

func (p *noopParker) Park(_ context.Context, appID string) error {
	p.logger.Info("park-noop", lager.Data{"appID": appID})
	return nil
}

func (p *noopParker) Unpark(_ context.Context, appID string) error {
	p.logger.Info("unpark-noop", lager.Data{"appID": appID})
	return nil
}
