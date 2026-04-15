package metrics

import (
	"fmt"
	"sync"
	"time"
)

const (
	defaultDashboardFreshCacheTTL = 30 * time.Second
	defaultDashboardStaleCacheTTL = 15 * time.Minute
)

type dashboardCacheEntry struct {
	response   *DashboardResponse
	freshUntil time.Time
	staleUntil time.Time
}

type dashboardCache struct {
	mu       sync.RWMutex
	entries  map[string]*dashboardCacheEntry
	freshTTL time.Duration
	staleTTL time.Duration
}

func newDashboardCache(freshTTL, staleTTL time.Duration) *dashboardCache {
	if freshTTL <= 0 {
		freshTTL = defaultDashboardFreshCacheTTL
	}
	if staleTTL <= 0 {
		staleTTL = defaultDashboardStaleCacheTTL
	}
	if staleTTL < freshTTL {
		staleTTL = freshTTL
	}
	return &dashboardCache{
		entries:  make(map[string]*dashboardCacheEntry),
		freshTTL: freshTTL,
		staleTTL: staleTTL,
	}
}

func dashboardCacheKey(req DashboardRequest) string {
	// Truncate times to the minute so nearby requests share the cache entry
	from := req.From.Truncate(time.Minute)
	to := req.To.Truncate(time.Minute)
	return fmt.Sprintf("%s|%s|%s", from.Format(time.RFC3339), to.Format(time.RFC3339), req.ServiceName)
}

func (c *dashboardCache) getFresh(req DashboardRequest) (*DashboardResponse, bool) {
	key := dashboardCacheKey(req)
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok || now.After(entry.freshUntil) {
		return nil, false
	}
	return cloneDashboardResponse(entry.response), true
}

func (c *dashboardCache) getStale(req DashboardRequest) (*DashboardResponse, bool) {
	key := dashboardCacheKey(req)
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok || now.After(entry.staleUntil) {
		return nil, false
	}
	return cloneDashboardResponse(entry.response), true
}

func (c *dashboardCache) set(req DashboardRequest, resp *DashboardResponse) {
	key := dashboardCacheKey(req)
	now := time.Now()

	c.mu.Lock()
	// Evict expired entries
	for k, e := range c.entries {
		if now.After(e.staleUntil) {
			delete(c.entries, k)
		}
	}
	c.entries[key] = &dashboardCacheEntry{
		response:   cloneDashboardResponse(resp),
		freshUntil: now.Add(c.freshTTL),
		staleUntil: now.Add(c.staleTTL),
	}
	c.mu.Unlock()
}

func cloneDashboardResponse(src *DashboardResponse) *DashboardResponse {
	if src == nil {
		return nil
	}

	out := *src
	out.Hotspots = HotspotsData{
		LatencyDistribution: cloneOrEmptySlice(src.Hotspots.LatencyDistribution),
		SlowestEndpoints:    cloneOrEmptySlice(src.Hotspots.SlowestEndpoints),
		ErrorHotspots:       cloneOrEmptySlice(src.Hotspots.ErrorHotspots),
		TopEndpoints:        cloneOrEmptySlice(src.Hotspots.TopEndpoints),
		StatusCodes:         cloneOrEmptySlice(src.Hotspots.StatusCodes),
	}
	out.Satisfaction = SatisfactionData{
		Apdex:           src.Satisfaction.Apdex,
		ErrorRate:       src.Satisfaction.ErrorRate,
		Throughput:      src.Satisfaction.Throughput,
		TimeSeries:      cloneOrEmptySlice(src.Satisfaction.TimeSeries),
		LatencySeries:   cloneOrEmptySlice(src.Satisfaction.LatencySeries),
		ErrorRateSeries: cloneOrEmptySlice(src.Satisfaction.ErrorRateSeries),
	}
	out.Logs = LogsData{
		VolumeSeries: cloneOrEmptySlice(src.Logs.VolumeSeries),
		Levels:       cloneOrEmptySlice(src.Logs.Levels),
	}
	out.Warnings = cloneOrEmptySlice(src.Warnings)
	return &out
}

func cloneOrEmptySlice[T any](in []T) []T {
	if len(in) == 0 {
		return []T{}
	}
	out := make([]T, len(in))
	copy(out, in)
	return out
}
