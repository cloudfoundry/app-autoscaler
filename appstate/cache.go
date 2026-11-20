package appstate

import (
	"context"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
)

type cacheEntry struct {
	instances int
	expires   time.Time
}

// Cache only supports no-op decisions. A target-changing operation always
// verifies CF. Tokens prevent an in-flight read from undoing an invalidation.
type Cache struct {
	mu         sync.Mutex
	entries    map[string]cacheEntry
	generation uint64
	maxAge     time.Duration
	checkedAt  time.Time
	cursor     int64
}

func NewCache(maxAge time.Duration) *Cache {
	return &Cache{entries: make(map[string]cacheEntry), maxAge: maxAge}
}

func (c *Cache) Token() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

func (c *Cache) Remember(appID string, app *cf.AppAndProcesses, started time.Time, token uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if token != c.generation || app == nil || app.App == nil {
		return
	}
	if app.App.State != "STARTED" || app.App.DisableAutoscaling != nil {
		delete(c.entries, appID)
		return
	}
	c.entries[appID] = cacheEntry{instances: app.Processes.GetInstances(), expires: started.Add(c.maxAge)}
}

func (c *Cache) Instances(appID string, now time.Time) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[appID]
	if !ok || !now.Before(e.expires) || c.checkedAt.IsZero() || now.Sub(c.checkedAt) > c.maxAge {
		delete(c.entries, appID)
		return 0, false
	}
	return e.instances, true
}

func (c *Cache) Invalidate(appID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	delete(c.entries, appID)
}

func (c *Cache) Poll(ctx context.Context, source Source) error {
	c.mu.Lock()
	cursor := c.cursor
	c.mu.Unlock()
	for {
		changes, err := source.Changes(ctx, cursor)
		if err != nil {
			c.mu.Lock()
			c.entries = make(map[string]cacheEntry)
			c.generation++
			c.checkedAt = time.Time{}
			c.mu.Unlock()
			return err
		}
		c.mu.Lock()
		if changes.Reset {
			c.entries = make(map[string]cacheEntry)
			c.generation++
		}
		for _, item := range changes.Items {
			delete(c.entries, item.AppID)
		}
		if len(changes.Items) > 0 {
			c.generation++
		}
		c.cursor = changes.Cursor
		// Only enable cache decisions once the entire change backlog is drained.
		if !changes.More {
			c.checkedAt = changes.CheckedAt
		} else {
			c.checkedAt = time.Time{}
		}
		c.mu.Unlock()
		cursor = changes.Cursor
		if !changes.More {
			return nil
		}
	}
}
