package channel

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter controls outbound message throughput per platform. Each adapter
// ships with a default limiter tuned to platform-specific limits (ADR-018).
type RateLimiter interface {
	// Allow returns true if a request for the given key is permitted
	// without blocking. Returns false if the rate limit is exceeded.
	Allow(ctx context.Context, key string) (bool, error)

	// Wait blocks until a request for the given key is permitted or the
	// context is cancelled.
	Wait(ctx context.Context, key string) error
}

// tokenBucket tracks the state for a single rate-limit key.
type tokenBucket struct {
	tokens   float64
	lastFill time.Time
}

// TokenBucketLimiter implements RateLimiter using the token bucket algorithm.
// Each key has an independent bucket with configurable rate and burst.
type TokenBucketLimiter struct {
	mu      sync.Mutex
	rate    float64       // tokens per second
	burst   int           // maximum tokens
	buckets map[string]*tokenBucket
	nowFunc func() time.Time // injectable clock for testing
}

// NewTokenBucketLimiter creates a limiter that allows rate tokens per second
// with a maximum burst capacity.
func NewTokenBucketLimiter(rate float64, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:    rate,
		burst:   burst,
		buckets: make(map[string]*tokenBucket),
		nowFunc: time.Now,
	}
}

// refill adds tokens to the bucket based on elapsed time since last fill.
// Must be called with mu held.
func (l *TokenBucketLimiter) refill(b *tokenBucket) {
	now := l.nowFunc()
	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	b.lastFill = now
}

// getBucket returns the bucket for key, creating it if absent.
// Must be called with mu held.
func (l *TokenBucketLimiter) getBucket(key string) *tokenBucket {
	b, ok := l.buckets[key]
	if !ok {
		b = &tokenBucket{
			tokens:   float64(l.burst),
			lastFill: l.nowFunc(),
		}
		l.buckets[key] = b
	}
	return b
}

// Allow checks whether a single token is available for key without blocking.
func (l *TokenBucketLimiter) Allow(_ context.Context, key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	b := l.getBucket(key)
	l.refill(b)

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true, nil
	}
	return false, nil
}

// Wait blocks until a token is available for key or ctx is cancelled.
func (l *TokenBucketLimiter) Wait(ctx context.Context, key string) error {
	for {
		ok, err := l.Allow(ctx, key)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		// Calculate how long until next token is available.
		waitDuration := time.Duration(float64(time.Second) / l.rate)
		if waitDuration < time.Millisecond {
			waitDuration = time.Millisecond
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("channel: rate limiter wait cancelled: %w", ctx.Err())
		case <-time.After(waitDuration):
			// Loop back and try again.
		}
	}
}

// NoOpLimiter always allows requests. Use for testing or platforms without
// meaningful rate limits.
type NoOpLimiter struct{}

// Allow always returns true.
func (NoOpLimiter) Allow(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// Wait always returns immediately.
func (NoOpLimiter) Wait(_ context.Context, _ string) error {
	return nil
}
