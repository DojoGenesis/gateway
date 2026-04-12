package cas

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// D1SyncConfig holds configuration for the D1 sync loop.
type D1SyncConfig struct {
	// Interval between sync cycles. Default: 5s.
	// Override with DOJO_D1_SYNC_INTERVAL env var (seconds).
	Interval time.Duration

	// BatchSize is the max number of entries per sync cycle. Default: 500.
	// Override with DOJO_D1_SYNC_BATCH_SIZE env var.
	BatchSize int
}

// DefaultD1SyncConfig returns the default sync config, reading env overrides.
func DefaultD1SyncConfig() D1SyncConfig {
	cfg := D1SyncConfig{
		Interval:  5 * time.Second,
		BatchSize: 500,
	}

	if v := os.Getenv("DOJO_D1_SYNC_INTERVAL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			cfg.Interval = time.Duration(secs) * time.Second
		}
	}
	if v := os.Getenv("DOJO_D1_SYNC_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.BatchSize = n
		}
	}

	return cfg
}

// DeltaEntry is a single content entry returned by a delta query.
type DeltaEntry struct {
	Hash       string      `json:"hash"`
	Data       []byte      `json:"data"`
	Meta       ContentMeta `json:"meta"`
	SyncCursor int64       `json:"sync_cursor"`
}

// SyncStatus reports the current state of the D1 sync loop.
type SyncStatus struct {
	LastCursor int64     `json:"last_cursor"`
	LastSyncAt time.Time `json:"last_sync_at"`
	LagSeconds float64   `json:"lag_seconds"`
	Healthy    bool      `json:"healthy"`
}

// D1Syncer manages the background sync loop from local CAS to D1.
type D1Syncer struct {
	local  Store // must implement deltaQuerier
	remote Store // D1-backed store
	cfg    D1SyncConfig

	mu         sync.RWMutex
	lastCursor int64
	lastSyncAt time.Time
	lastErr    error
	healthy    bool
}

// NewD1Syncer creates a new syncer. The local store must be a sqliteStore
// (implements deltaQuerier). The remote store should be a D1-backed store.
func NewD1Syncer(local Store, remote Store, cfg D1SyncConfig) *D1Syncer {
	return &D1Syncer{
		local:   local,
		remote:  remote,
		cfg:     cfg,
		healthy: false,
	}
}

// Run starts the sync loop. It blocks until ctx is cancelled.
// Call this in a goroutine: go syncer.Run(ctx)
func (s *D1Syncer) Run(ctx context.Context) {
	log.Printf("[d1sync] starting sync loop (interval=%s, batch=%d)", s.cfg.Interval, s.cfg.BatchSize)

	// Initial sync immediately
	s.syncOnce(ctx)

	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[d1sync] sync loop stopped")
			return
		case <-ticker.C:
			s.syncOnce(ctx)
		}
	}
}

// syncOnce performs a single sync cycle.
func (s *D1Syncer) syncOnce(ctx context.Context) {
	dq, ok := s.local.(deltaQuerier)
	if !ok {
		s.setError(fmt.Errorf("local store does not support delta queries"))
		return
	}

	entries, err := dq.Delta(ctx, s.getLastCursor(), s.cfg.BatchSize)
	if err != nil {
		s.setError(fmt.Errorf("delta query: %w", err))
		return
	}

	if len(entries) == 0 {
		s.setHealthy()
		return
	}

	var maxCursor int64
	var syncErrors int

	for _, entry := range entries {
		// Push each entry to the remote D1 store
		_, err := s.remote.Put(ctx, entry.Data, entry.Meta)
		if err != nil {
			log.Printf("[d1sync] error pushing hash %s: %v", entry.Hash, err)
			syncErrors++
			// Continue with other entries — don't stop on a single failure
			continue
		}
		if entry.SyncCursor > maxCursor {
			maxCursor = entry.SyncCursor
		}
	}

	if syncErrors > 0 && syncErrors == len(entries) {
		// All entries failed — mark unhealthy but don't advance cursor
		s.setError(fmt.Errorf("all %d entries failed to sync", syncErrors))
		return
	}

	// Advance cursor past successfully synced entries
	if maxCursor > 0 {
		s.advanceCursor(maxCursor)
	}

	synced := len(entries) - syncErrors
	log.Printf("[d1sync] synced %d/%d entries (cursor=%d)", synced, len(entries), maxCursor)

	if syncErrors > 0 {
		s.setError(fmt.Errorf("%d/%d entries failed", syncErrors, len(entries)))
	} else {
		s.setHealthy()
	}
}

// Status returns the current sync status.
func (s *D1Syncer) Status() SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var lag float64
	if !s.lastSyncAt.IsZero() {
		lag = time.Since(s.lastSyncAt).Seconds()
	}

	return SyncStatus{
		LastCursor: s.lastCursor,
		LastSyncAt: s.lastSyncAt,
		LagSeconds: lag,
		Healthy:    s.healthy,
	}
}

func (s *D1Syncer) getLastCursor() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastCursor
}

func (s *D1Syncer) advanceCursor(cursor int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCursor = cursor
	s.lastSyncAt = time.Now().UTC()
}

func (s *D1Syncer) setHealthy() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = true
	s.lastErr = nil
	s.lastSyncAt = time.Now().UTC()
}

func (s *D1Syncer) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = false
	s.lastErr = err
	log.Printf("[d1sync] error: %v", err)
}

// deltaQuerier is the interface the local store must implement.
type deltaQuerier interface {
	Delta(ctx context.Context, since int64, limit int) ([]DeltaEntry, error)
	MaxSyncCursor(ctx context.Context) (int64, error)
}
