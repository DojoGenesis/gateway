package disposition

import (
	"sync"
	"time"
)

// DispositionCache is a simple TTL-based cache for DispositionConfig instances.
// It prevents repeated disk I/O when loading the same agent configuration multiple times.
// The cache is thread-safe and automatically evicts expired entries.
type DispositionCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

// cacheEntry represents a single cached disposition with its expiration time.
type cacheEntry struct {
	config    *DispositionConfig
	expiresAt time.Time
}

// NewDispositionCache creates a new cache with the specified TTL (time-to-live).
// A TTL of 0 means entries never expire (not recommended for production).
func NewDispositionCache(ttl time.Duration) *DispositionCache {
	cache := &DispositionCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}

	// Start background cleanup goroutine if TTL is set
	if ttl > 0 {
		go cache.cleanupExpired()
	}

	return cache
}

// Get retrieves a cached DispositionConfig by key.
// Returns the config and true if found and not expired, or nil and false otherwise.
//
// The key format is: "${workspaceRoot}:${activeMode}"
// This ensures that the same workspace with different modes are cached separately.
func (c *DispositionCache) Get(key string) (*DispositionConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if c.ttl > 0 && time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.config, true
}

// Set stores a DispositionConfig in the cache with the specified key.
// The entry will expire after the cache's TTL duration.
//
// The key format is: "${workspaceRoot}:${activeMode}"
func (c *DispositionCache) Set(key string, cfg *DispositionConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Now().Add(c.ttl)
	if c.ttl == 0 {
		// Never expire
		expiresAt = time.Time{}
	}

	c.entries[key] = &cacheEntry{
		config:    cfg,
		expiresAt: expiresAt,
	}
}

// Clear removes all entries from the cache.
func (c *DispositionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
}

// Delete removes a specific entry from the cache.
// Returns true if the entry was found and deleted, false otherwise.
func (c *DispositionCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.entries[key]
	if exists {
		delete(c.entries, key)
		return true
	}
	return false
}

// Size returns the number of entries currently in the cache (including expired ones).
func (c *DispositionCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// cleanupExpired is a background goroutine that periodically removes expired entries.
// It runs every TTL/2 interval to keep memory usage bounded.
func (c *DispositionCache) cleanupExpired() {
	cleanupInterval := c.ttl / 2
	if cleanupInterval < time.Minute {
		cleanupInterval = time.Minute
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.removeExpiredEntries()
	}
}

// removeExpiredEntries removes all entries that have passed their expiration time.
func (c *DispositionCache) removeExpiredEntries() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if c.ttl > 0 && now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// makeCacheKey creates a cache key from workspace root and active mode.
func makeCacheKey(workspaceRoot, activeMode string) string {
	if activeMode == "" {
		return workspaceRoot + ":base"
	}
	return workspaceRoot + ":" + activeMode
}
