package metrics

import (
	"fmt"
	"sync"
	"time"
)

const dashboardCacheTTL = 30 * time.Second

type dashboardCacheEntry struct {
	response  *DashboardResponse
	expiresAt time.Time
}

type dashboardCache struct {
	mu      sync.RWMutex
	entries map[string]*dashboardCacheEntry
}

func newDashboardCache() *dashboardCache {
	return &dashboardCache{
		entries: make(map[string]*dashboardCacheEntry),
	}
}

func dashboardCacheKey(req DashboardRequest) string {
	// Truncate times to the minute so nearby requests share the cache entry
	from := req.From.Truncate(time.Minute)
	to := req.To.Truncate(time.Minute)
	return fmt.Sprintf("%s|%s|%s", from.Format(time.RFC3339), to.Format(time.RFC3339), req.ServiceName)
}

func (c *dashboardCache) get(req DashboardRequest) (*DashboardResponse, bool) {
	key := dashboardCacheKey(req)
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok || now.After(entry.expiresAt) {
		return nil, false
	}
	return entry.response, true
}

func (c *dashboardCache) set(req DashboardRequest, resp *DashboardResponse) {
	key := dashboardCacheKey(req)
	now := time.Now()

	c.mu.Lock()
	// Evict expired entries
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
	c.entries[key] = &dashboardCacheEntry{
		response:  resp,
		expiresAt: now.Add(dashboardCacheTTL),
	}
	c.mu.Unlock()
}
