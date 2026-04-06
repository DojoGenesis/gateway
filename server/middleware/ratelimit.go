package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds rate limiter configuration.
type RateLimitConfig struct {
	// RequestsPerMinute is the maximum sustained request rate per IP.
	RequestsPerMinute int
	// BurstSize is the maximum number of requests allowed in a short burst.
	BurstSize int
	// CleanupInterval is how often stale limiters are evicted.
	CleanupInterval time.Duration
	// MaxAge is how long an idle limiter is kept before eviction.
	MaxAge time.Duration
}

// DefaultRateLimitConfig returns sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 300,
		BurstSize:         50,
		CleanupInterval:   5 * time.Minute,
		MaxAge:            10 * time.Minute,
	}
}

// ipLimiter tracks a rate.Limiter and when it was last used.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimitStore manages per-IP rate limiters.
type rateLimitStore struct {
	mu       sync.RWMutex
	limiters map[string]*ipLimiter
	rate     rate.Limit
	burst    int
}

func newRateLimitStore(rps rate.Limit, burst int) *rateLimitStore {
	return &rateLimitStore{
		limiters: make(map[string]*ipLimiter),
		rate:     rps,
		burst:    burst,
	}
}

// getLimiter returns the rate limiter for a given IP, creating one if needed.
func (s *rateLimitStore) getLimiter(ip string) *rate.Limiter {
	s.mu.RLock()
	entry, exists := s.limiters[ip]
	s.mu.RUnlock()

	if exists {
		s.mu.Lock()
		entry.lastSeen = time.Now()
		s.mu.Unlock()
		return entry.limiter
	}

	limiter := rate.NewLimiter(s.rate, s.burst)
	s.mu.Lock()
	s.limiters[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}
	s.mu.Unlock()
	return limiter
}

// cleanup removes limiters that haven't been seen since maxAge.
func (s *rateLimitStore) cleanup(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for ip, entry := range s.limiters {
		if entry.lastSeen.Before(cutoff) {
			delete(s.limiters, ip)
		}
	}
}

// RateLimitMiddleware creates a per-IP token-bucket rate limiter.
func RateLimitMiddleware(cfg RateLimitConfig) gin.HandlerFunc {
	rps := rate.Limit(float64(cfg.RequestsPerMinute) / 60.0)
	store := newRateLimitStore(rps, cfg.BurstSize)

	// Background cleanup goroutine
	go func() {
		ticker := time.NewTicker(cfg.CleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			store.cleanup(cfg.MaxAge)
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := store.getLimiter(ip)

		if !limiter.Allow() {
			slog.Warn("rate limit exceeded", "client_ip", ip)
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
