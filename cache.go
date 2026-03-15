package pinergy

import (
	"sync"
	"time"
)

// defaultTTLs maps API endpoint paths to their default cache TTLs.
var defaultTTLs = map[string]time.Duration{
	"/api/balance/":       60 * time.Second,
	"/api/usage/":         5 * time.Minute,
	"/api/levelpayusage/": 5 * time.Minute,
	"/api/compare/":       15 * time.Minute,
	"/api/configinfo/":    30 * time.Minute,
	"/api/defaultsinfo/":  30 * time.Minute,
	"/api/activetopups/":  2 * time.Minute,
	"/api/getnotif/":      5 * time.Minute,
	"/version.json":       10 * time.Minute,
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// ttlCache is a thread-safe in-memory cache that stores raw JSON response bytes
// with per-endpoint TTLs. Storing raw bytes avoids a generic type parameter
// and allows all endpoint methods to share a single cache instance.
type ttlCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttls    map[string]time.Duration
	enabled bool
}

func newTTLCache(overrides map[string]time.Duration) *ttlCache {
	ttls := make(map[string]time.Duration, len(defaultTTLs))
	for k, v := range defaultTTLs {
		ttls[k] = v
	}
	for k, v := range overrides {
		ttls[k] = v
	}
	return &ttlCache{
		entries: make(map[string]cacheEntry),
		ttls:    ttls,
		enabled: true,
	}
}

func newDisabledCache() *ttlCache {
	return &ttlCache{
		entries: make(map[string]cacheEntry),
		ttls:    make(map[string]time.Duration),
		enabled: false,
	}
}

// Get returns the cached bytes for key if present and not expired.
func (c *ttlCache) Get(key string) ([]byte, bool) {
	if !c.enabled {
		return nil, false
	}
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.data, true
}

// Set stores data for key using the TTL configured for endpoint.
// If no TTL is configured for the endpoint the entry is not stored.
func (c *ttlCache) Set(key, endpoint string, data []byte) {
	if !c.enabled {
		return
	}
	c.mu.RLock()
	ttl, ok := c.ttls[endpoint]
	c.mu.RUnlock()
	if !ok || ttl == 0 {
		return
	}
	c.mu.Lock()
	c.entries[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// Invalidate removes a specific cache entry.
func (c *ttlCache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// Flush clears all cache entries.
func (c *ttlCache) Flush() {
	c.mu.Lock()
	c.entries = make(map[string]cacheEntry)
	c.mu.Unlock()
}

// SetTTL updates the TTL for a specific endpoint.
func (c *ttlCache) SetTTL(endpoint string, ttl time.Duration) {
	c.mu.Lock()
	c.ttls[endpoint] = ttl
	c.mu.Unlock()
}
