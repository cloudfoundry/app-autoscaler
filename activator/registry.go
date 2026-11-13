package activator

import "sync"

// inMemoryRegistry is a concurrency-safe Registry. It holds the activator's
// view of which apps are parked and which routes it bound for them. This state
// is authoritative only as a cache; it must be reconcilable from CF (the
// ReadinessWatcher's reconcile loop) so an activator restart can recover.
type inMemoryRegistry struct {
	mu     sync.RWMutex
	parked map[string][]string // appID -> bound route URLs
}

func NewInMemoryRegistry() Registry {
	return &inMemoryRegistry{parked: make(map[string][]string)}
}

func (r *inMemoryRegistry) IsParked(appID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.parked[appID]
	return ok
}

func (r *inMemoryRegistry) MarkParked(appID string, routes []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parked[appID] = routes
}

func (r *inMemoryRegistry) MarkUnparked(appID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.parked, appID)
}

func (r *inMemoryRegistry) ParkedApps() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	apps := make([]string, 0, len(r.parked))
	for appID := range r.parked {
		apps = append(apps, appID)
	}
	return apps
}

func (r *inMemoryRegistry) RoutesFor(appID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.parked[appID]
}
