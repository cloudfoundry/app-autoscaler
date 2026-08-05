package cache

import (
	"maps"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// AllowedMetricCache wraps go-cache for the allowed-metric type sets.
// go-cache returns stored values by reference, so every read and write
// clones the map. This keeps the clone-on-read / clone-on-write invariant
// in one place: callers can never accidentally share a mutable map with
// the cache.
type AllowedMetricCache struct {
	c *gocache.Cache
}

func New(ttl, cleanupInterval time.Duration) *AllowedMetricCache {
	return &AllowedMetricCache{c: gocache.New(ttl, cleanupInterval)}
}

// Get returns a clone of the cached set, so callers may freely mutate it.
func (a *AllowedMetricCache) Get(appID string) (map[string]struct{}, bool) {
	v, found := a.c.Get(appID)
	if !found {
		return nil, false
	}
	m, ok := v.(map[string]struct{})
	if !ok {
		return nil, false
	}
	return maps.Clone(m), true
}

// Set stores a clone, so later mutation of the input does not affect the cache.
func (a *AllowedMetricCache) Set(appID string, m map[string]struct{}, ttl time.Duration) {
	a.c.Set(appID, maps.Clone(m), ttl)
}

// Replace stores a clone, so later mutation of the input does not affect the cache.
func (a *AllowedMetricCache) Replace(appID string, m map[string]struct{}, ttl time.Duration) error {
	return a.c.Replace(appID, maps.Clone(m), ttl)
}

func (a *AllowedMetricCache) Delete(appID string) {
	a.c.Delete(appID)
}

// AppIDs returns the cached app IDs. It fully encapsulates gocache.Item so it
// never leaks through the wrapper.
func (a *AllowedMetricCache) AppIDs() []string {
	items := a.c.Items()
	ids := make([]string, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	return ids
}

func (a *AllowedMetricCache) Flush() {
	a.c.Flush()
}
