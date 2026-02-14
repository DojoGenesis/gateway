package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type CachedResponse struct {
	Content   string
	Timestamp time.Time
	HitCount  int
}

type ResponseCache struct {
	mu      sync.RWMutex
	cache   map[string]*CachedResponse
	ttl     time.Duration
	maxSize int
	enabled bool
	hits    int64
	misses  int64
}

func NewResponseCache(ttl time.Duration, maxSize int) *ResponseCache {
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	if maxSize == 0 {
		maxSize = 1000
	}

	rc := &ResponseCache{
		cache:   make(map[string]*CachedResponse),
		ttl:     ttl,
		maxSize: maxSize,
		enabled: true,
	}

	go rc.cleanupExpired()

	return rc
}

func (rc *ResponseCache) Get(query string) (string, bool) {
	if !rc.enabled {
		return "", false
	}

	rc.mu.RLock()
	key := rc.hashQuery(query)
	cached, exists := rc.cache[key]
	rc.mu.RUnlock()

	if !exists {
		rc.mu.Lock()
		rc.misses++
		rc.mu.Unlock()
		return "", false
	}

	if time.Since(cached.Timestamp) > rc.ttl {
		rc.mu.Lock()
		delete(rc.cache, key)
		rc.misses++
		rc.mu.Unlock()
		return "", false
	}

	rc.mu.Lock()
	cached.HitCount++
	rc.hits++
	rc.mu.Unlock()

	return cached.Content, true
}

func (rc *ResponseCache) Set(query string, response string) {
	if !rc.enabled {
		return
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	if len(rc.cache) >= rc.maxSize {
		rc.evictLRU()
	}

	key := rc.hashQuery(query)
	rc.cache[key] = &CachedResponse{
		Content:   response,
		Timestamp: time.Now(),
		HitCount:  0,
	}
}

func (rc *ResponseCache) hashQuery(query string) string {
	hasher := sha256.New()
	hasher.Write([]byte(query))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (rc *ResponseCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time = time.Now()
	var lowestHits int = int(^uint(0) >> 1)

	for key, cached := range rc.cache {
		if cached.HitCount < lowestHits ||
			(cached.HitCount == lowestHits && cached.Timestamp.Before(oldestTime)) {
			oldestKey = key
			oldestTime = cached.Timestamp
			lowestHits = cached.HitCount
		}
	}

	if oldestKey != "" {
		delete(rc.cache, oldestKey)
	}
}

func (rc *ResponseCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for key, cached := range rc.cache {
			if now.Sub(cached.Timestamp) > rc.ttl {
				delete(rc.cache, key)
			}
		}
		rc.mu.Unlock()
	}
}

func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cache = make(map[string]*CachedResponse)
	rc.hits = 0
	rc.misses = 0
}

func (rc *ResponseCache) Stats() (hits, misses int64, hitRate float64) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	hits = rc.hits
	misses = rc.misses
	total := float64(hits + misses)

	if total > 0 {
		hitRate = float64(hits) / total
	}

	return
}

func (rc *ResponseCache) Enable() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.enabled = true
}

func (rc *ResponseCache) Disable() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.enabled = false
}

func (rc *ResponseCache) Size() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.cache)
}
