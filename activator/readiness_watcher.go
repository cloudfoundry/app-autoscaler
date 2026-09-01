package activator

import (
	"context"
	"sync"

	"code.cloudfoundry.org/lager/v3"
)

// RouteEvent is a route-registration event from the routing-api stream,
// decoupled from the concrete client type for testability.
type RouteEvent struct {
	RouteURL string
	Action   string // "Upsert" or "Delete"
}

// RouteEventStream is the subset of routing-api's EventSource we consume.
type RouteEventStream interface {
	Next() (RouteEvent, error)
	Close() error
}

// RouteEventSubscriber opens a route-registration event stream. The real
// implementation wraps routing_api.Client.SubscribeToEvents (see
// routing_events.go); tests supply a fake.
type RouteEventSubscriber interface {
	Subscribe() (RouteEventStream, error)
}

const actionUpsert = "Upsert"

// readinessWatcher is Loop B: it consumes route-registration events and, when a
// parked app's route is (re)registered by route-emitter (an Upsert), treats the
// app as ready — releasing held requests and un-parking it. See
// docs/design/scale-to-zero.md §5.3.
type readinessWatcher struct {
	logger     lager.Logger
	subscriber RouteEventSubscriber
	registry   Registry
	parker     Parker

	mu      sync.Mutex
	waiters map[string][]chan struct{} // appID -> readiness waiters
}

func NewReadinessWatcher(logger lager.Logger, subscriber RouteEventSubscriber, registry Registry, parker Parker) ReadinessWatcher {
	return &readinessWatcher{
		logger:     logger.Session("readiness-watcher"),
		subscriber: subscriber,
		registry:   registry,
		parker:     parker,
		waiters:    make(map[string][]chan struct{}),
	}
}

// WaitForReady returns a channel that receives nil when the app's route is
// re-registered (Upsert observed by Run), or ctx.Err() if ctx is done first.
func (w *readinessWatcher) WaitForReady(ctx context.Context, appID string) <-chan error {
	result := make(chan error, 1)
	notify := make(chan struct{})

	w.mu.Lock()
	w.waiters[appID] = append(w.waiters[appID], notify)
	w.mu.Unlock()

	go func() {
		select {
		case <-notify:
			result <- nil
		case <-ctx.Done():
			w.removeWaiter(appID, notify)
			result <- ctx.Err()
		}
	}()
	return result
}

func (w *readinessWatcher) removeWaiter(appID string, target chan struct{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	waiters := w.waiters[appID]
	for i, ch := range waiters {
		if ch == target {
			w.waiters[appID] = append(waiters[:i], waiters[i+1:]...)
			break
		}
	}
	if len(w.waiters[appID]) == 0 {
		delete(w.waiters, appID)
	}
}

// Run consumes the event stream until ctx is done. On each Upsert whose route
// belongs to a parked app, it un-parks the app and wakes any waiters.
func (w *readinessWatcher) Run(ctx context.Context) error {
	stream, err := w.subscriber.Subscribe()
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()

	go func() {
		<-ctx.Done()
		_ = stream.Close()
	}()

	for {
		event, err := stream.Next()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if event.Action != actionUpsert {
			continue
		}
		w.handleUpsert(ctx, event.RouteURL)
	}
}

func (w *readinessWatcher) handleUpsert(ctx context.Context, routeURL string) {
	appID := w.appForRoute(routeURL)
	if appID == "" {
		return // not a route we parked
	}
	w.logger.Info("app-ready-upsert", lager.Data{"appID": appID, "route": routeURL})

	// Wake held requests first, then un-park (unbind route service) lazily so a
	// slow unbind never drops traffic.
	w.wakeWaiters(appID)
	if err := w.parker.Unpark(ctx, appID); err != nil {
		w.logger.Error("failed-to-unpark", err, lager.Data{"appID": appID})
	}
}

func (w *readinessWatcher) appForRoute(routeURL string) string {
	for _, appID := range w.registry.ParkedApps() {
		for _, u := range w.registry.RoutesFor(appID) {
			if u == routeURL {
				return appID
			}
		}
	}
	return ""
}

func (w *readinessWatcher) wakeWaiters(appID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, ch := range w.waiters[appID] {
		close(ch)
	}
	delete(w.waiters, appID)
}
