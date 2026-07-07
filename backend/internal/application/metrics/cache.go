package metrics

import (
	"fmt"
	"sync"
	"time"

	"opendashly/backend/internal/infrastructure/config"
)

var (
	defaultDashboardFreshCacheTTL = time.Duration(config.Auto().DashboardFreshCacheTTLSec) * time.Second
	defaultDashboardStaleCacheTTL = time.Duration(config.Auto().DashboardStaleCacheTTLSec) * time.Second
)

type dashboardCacheEntry struct {
	response   *DashboardResponse
	freshUntil time.Time
	staleUntil time.Time
}

type dashboardCache struct {
	mu       sync.RWMutex
	entries  map[string]*dashboardCacheEntry
	lastGood map[string]*dashboardCacheEntry
	freshTTL time.Duration
	staleTTL time.Duration
	lastTTL  time.Duration
}

func newDashboardCache(freshTTL, staleTTL, lastGoodTTL time.Duration) *dashboardCache {
	if freshTTL <= 0 {
		freshTTL = defaultDashboardFreshCacheTTL
	}
	if staleTTL <= 0 {
		staleTTL = defaultDashboardStaleCacheTTL
	}
	if lastGoodTTL <= 0 {
		lastGoodTTL = time.Duration(config.Auto().DashboardLastGoodTTLSec) * time.Second
	}
	if staleTTL < freshTTL {
		staleTTL = freshTTL
	}
	return &dashboardCache{
		entries:  make(map[string]*dashboardCacheEntry),
		lastGood: make(map[string]*dashboardCacheEntry),
		freshTTL: freshTTL,
		staleTTL: staleTTL,
		lastTTL:  lastGoodTTL,
	}
}

func dashboardCacheKey(req DashboardRequest) string {
	// Truncate times to the minute so nearby requests share the cache entry
	from := req.From.Truncate(time.Minute)
	to := req.To.Truncate(time.Minute)
	return fmt.Sprintf("%s|%s|%s", from.Format(time.RFC3339), to.Format(time.RFC3339), req.ServiceName)
}

func dashboardLastGoodKey(req DashboardRequest) string {
	duration := req.To.Sub(req.From).Round(time.Minute)
	if duration <= 0 {
		duration = 0
	}
	return fmt.Sprintf("%s|%s", duration.String(), req.ServiceName)
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

	if ok && !now.After(entry.staleUntil) {
		return cloneDashboardResponse(entry.response), true
	}

	lastKey := dashboardLastGoodKey(req)
	c.mu.RLock()
	entry, ok = c.lastGood[lastKey]
	c.mu.RUnlock()
	if ok && !now.After(entry.staleUntil) {
		return cloneDashboardResponse(entry.response), true
	}
	return nil, false
}

func (c *dashboardCache) set(req DashboardRequest, resp *DashboardResponse) {
	key := dashboardCacheKey(req)
	lastKey := dashboardLastGoodKey(req)
	now := time.Now()

	c.mu.Lock()
	// Evict expired entries
	for k, e := range c.entries {
		if now.After(e.staleUntil) {
			delete(c.entries, k)
		}
	}
	for k, e := range c.lastGood {
		if now.After(e.staleUntil) {
			delete(c.lastGood, k)
		}
	}
	c.entries[key] = &dashboardCacheEntry{
		response:   cloneDashboardResponse(resp),
		freshUntil: now.Add(c.freshTTL),
		staleUntil: now.Add(c.staleTTL),
	}
	c.lastGood[lastKey] = &dashboardCacheEntry{
		response:   cloneDashboardResponse(resp),
		freshUntil: now.Add(c.freshTTL),
		staleUntil: now.Add(c.lastTTL),
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
	out.Health = src.Health
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
