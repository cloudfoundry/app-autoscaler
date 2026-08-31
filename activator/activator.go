// Package activator implements scale-from-zero for Cloud Foundry apps parked by
// the autoscaler. It is the counterpart to the scaling engine's Activator seam
// (see scalingengine/activator.go and docs/design/scale-to-zero.md).
//
// The activator is a CF app bound as a route service. While an app is parked
// (scaled to zero), its routes are bound to the activator route-service so
// Gorouter forwards requests here instead of to the zero-instance app. The
// activator then runs two decoupled loops:
//
//   - Loop A (request handling): hold the incoming request, ask the scaling
//     engine to scale the app to 1, wait for the app's readiness signal, then
//     forward the held request back to X-CF-Forwarded-Url so Gorouter delivers
//     it to the now-live app. On timeout, return 503 + Retry-After.
//
//   - Loop B (unpark): driven by the routing-api Upsert event stream. When an
//     app the activator has parked registers its real endpoint (from any cause
//     — request-driven, scheduled, or manual `cf scale`), unbind its route
//     service. A periodic reconcile against CF is the safety net for missed
//     events / activator restarts.
//
// This file is the PoC scaffold: the loops are defined as seams with stub
// implementations so the service compiles and runs. Real routing-api and CF
// route-service integration lands as the loops are implemented.
package activator

import (
	"context"

	"code.cloudfoundry.org/lager/v3"
)

// Parker binds and unbinds an app's routes to the activator route-service.
// Park is the operation the scaling engine's Activator.Park ultimately drives
// (bind -> confirm -> scale-to-0). Unpark is invoked by Loop B once an app is
// observed to be up again.
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
}

// ReadinessWatcher is Loop B: it consumes route-registration (Upsert) events
// and signals when a parked app's real endpoint has come up.
type ReadinessWatcher interface {
	// WaitForReady blocks until the app is observed ready or ctx is done.
	WaitForReady(ctx context.Context, appID string) error
	// Run consumes the event stream until ctx is done, un-parking apps whose
	// Upsert arrives regardless of what caused the scale-up.
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
