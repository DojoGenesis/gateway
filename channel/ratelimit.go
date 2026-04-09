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

// ---------------------------------------------------------------------------
// Per-adapter default rate limiters (ADR-018 Q15)
// ---------------------------------------------------------------------------

// NewSlackLimiter returns a rate limiter tuned to Slack's documented limits:
// 1 message per second per channel. Burst of 1 (no bursting above rate).
func NewSlackLimiter() *TokenBucketLimiter {
	return NewTokenBucketLimiter(1.0, 1) // 1 msg/sec/channel
}

// NewDiscordLimiter returns a rate limiter tuned to Discord's per-guild rate
// limits. Discord allows 5 requests per 5 seconds per channel (1 msg/sec
// sustained, burst 5). This is the "normal" tier; bots in very large guilds
// may have higher limits.
func NewDiscordLimiter() *TokenBucketLimiter {
	return NewTokenBucketLimiter(1.0, 5) // 1 msg/sec sustained, burst 5
}

// TelegramDualLimiter applies different rate limits for group chats vs direct
// messages. Telegram allows 30 msg/sec to groups but only 1 msg/sec to DMs.
// Keys should be prefixed: "group:{chat_id}" or "dm:{chat_id}".
type TelegramDualLimiter struct {
	groupLimiter *TokenBucketLimiter
	dmLimiter    *TokenBucketLimiter
}

// NewTelegramLimiter returns a dual-mode limiter: 30 msg/sec for groups,
// 1 msg/sec for DMs.
func NewTelegramLimiter() *TelegramDualLimiter {
	return &TelegramDualLimiter{
		groupLimiter: NewTokenBucketLimiter(30.0, 30), // 30 msg/sec groups
		dmLimiter:    NewTokenBucketLimiter(1.0, 1),   // 1 msg/sec DM
	}
}

// Allow checks rate limits based on key prefix. Keys starting with "dm:" use
// the DM limiter; all others use the group limiter.
func (t *TelegramDualLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if len(key) > 3 && key[:3] == "dm:" {
		return t.dmLimiter.Allow(ctx, key)
	}
	return t.groupLimiter.Allow(ctx, key)
}

// Wait blocks until allowed, respecting the correct limiter for the key prefix.
func (t *TelegramDualLimiter) Wait(ctx context.Context, key string) error {
	if len(key) > 3 && key[:3] == "dm:" {
		return t.dmLimiter.Wait(ctx, key)
	}
	return t.groupLimiter.Wait(ctx, key)
}
