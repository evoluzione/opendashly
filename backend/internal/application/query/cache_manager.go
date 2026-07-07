package query

import (
	"strings"
	"sync"
	"time"

	"opendashly/backend/internal/infrastructure/config"
)

var (
	servicesCacheTTL   = time.Duration(config.Auto().QueryServicesCacheTTLSec) * time.Second
	attributesCacheTTL = time.Duration(config.Auto().QueryAttributesCacheTTLSec) * time.Second
)

type cachedStringSlice struct {
	values    []string
	expiresAt time.Time
}

type cacheManager struct {
	mu              sync.RWMutex
	servicesTTL     time.Duration
	attributesTTL   time.Duration
	servicesCache   cachedStringSlice
	attributesCache map[string]cachedStringSlice
}

func newCacheManager(servicesTTL, attributesTTL time.Duration) *cacheManager {
	return &cacheManager{
		servicesTTL:     servicesTTL,
		attributesTTL:   attributesTTL,
		attributesCache: map[string]cachedStringSlice{},
	}
}

func (c *cacheManager) getServices() ([]string, bool) {
	now := time.Now()
	c.mu.RLock()
	entry := c.servicesCache
	c.mu.RUnlock()
	if now.After(entry.expiresAt) || len(entry.values) == 0 {
		return nil, false
	}
	return cloneStringSlice(entry.values), true
}

func (c *cacheManager) setServices(values []string) {
	c.mu.Lock()
	c.servicesCache = cachedStringSlice{
		values:    cloneStringSlice(values),
		expiresAt: time.Now().Add(c.servicesTTL),
	}
	c.mu.Unlock()
}

func (c *cacheManager) getAttributes(search string) ([]string, bool) {
	cacheKey := normalizeCacheKey(search)
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.attributesCache[cacheKey]
	c.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) {
		return nil, false
	}
	return cloneStringSlice(entry.values), true
}

func (c *cacheManager) setAttributes(search string, values []string) {
	cacheKey := normalizeCacheKey(search)
	entry := cachedStringSlice{
		values:    cloneStringSlice(values),
		expiresAt: time.Now().Add(c.attributesTTL),
	}

	c.mu.Lock()
	c.attributesCache[cacheKey] = entry
	c.mu.Unlock()
}

func normalizeCacheKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
