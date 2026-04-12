package cas

import (
	"context"
	"testing"
	"time"
)

// newTestStore creates an in-memory SQLite store for testing.
// The store implements both Store and deltaQuerier.
func newTestStore(t *testing.T) Store {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// TestDefaultD1SyncConfig verifies that the default config has interval=5s and batch=500.
func TestDefaultD1SyncConfig(t *testing.T) {
	cfg := DefaultD1SyncConfig()
	if cfg.Interval != 5*time.Second {
		t.Errorf("expected Interval=5s, got %v", cfg.Interval)
	}
	if cfg.BatchSize != 500 {
		t.Errorf("expected BatchSize=500, got %d", cfg.BatchSize)
	}
}

// TestNewD1Syncer verifies that NewD1Syncer returns a non-nil syncer
// and that its initial Status has Healthy=false.
func TestNewD1Syncer(t *testing.T) {
	local := newTestStore(t)
	remote := newTestStore(t)
	cfg := D1SyncConfig{Interval: 5 * time.Second, BatchSize: 500}

	syncer := NewD1Syncer(local, remote, cfg)
	if syncer == nil {
		t.Fatal("NewD1Syncer returned nil")
	}

	status := syncer.Status()
	if status.Healthy {
		t.Error("expected initial Healthy=false, got true")
	}
}

// TestD1SyncerSyncOnce puts 3 blobs in local, calls syncOnce, and verifies
// that the remote store received all 3 blobs and that the cursor advanced.
func TestD1SyncerSyncOnce(t *testing.T) {
	local := newTestStore(t)
	remote := newTestStore(t)
	cfg := D1SyncConfig{Interval: 5 * time.Second, BatchSize: 500}
	syncer := NewD1Syncer(local, remote, cfg)

	ctx := context.Background()
	blobs := [][]byte{
		[]byte("blob-alpha"),
		[]byte("blob-beta"),
		[]byte("blob-gamma"),
	}
	var refs []Ref
	for _, b := range blobs {
		ref, err := local.Put(ctx, b, ContentMeta{Type: ContentSkill})
		if err != nil {
			t.Fatalf("local.Put: %v", err)
		}
		refs = append(refs, ref)
	}

	syncer.syncOnce(ctx)

	// All 3 blobs should now exist in the remote store.
	for i, ref := range refs {
		ok, err := remote.Has(ctx, ref)
		if err != nil {
			t.Fatalf("remote.Has(%s): %v", ref, err)
		}
		if !ok {
			t.Errorf("blob %d (ref=%s) not found in remote store", i, ref)
		}
	}

	// Cursor should have advanced past 0.
	status := syncer.Status()
	if status.LastCursor <= 0 {
		t.Errorf("expected LastCursor > 0, got %d", status.LastCursor)
	}
}

// TestD1SyncerSyncEmpty verifies that syncOnce on an empty store
// sets Healthy=true and leaves cursor at 0.
func TestD1SyncerSyncEmpty(t *testing.T) {
	local := newTestStore(t)
	remote := newTestStore(t)
	cfg := D1SyncConfig{Interval: 5 * time.Second, BatchSize: 500}
	syncer := NewD1Syncer(local, remote, cfg)

	syncer.syncOnce(context.Background())

	status := syncer.Status()
	if !status.Healthy {
		t.Error("expected Healthy=true after syncing empty store")
	}
	if status.LastCursor != 0 {
		t.Errorf("expected LastCursor=0, got %d", status.LastCursor)
	}
}

// TestD1SyncerStatus verifies that after a successful sync, Status reports
// Healthy=true and LastCursor > 0.
func TestD1SyncerStatus(t *testing.T) {
	local := newTestStore(t)
	remote := newTestStore(t)
	cfg := D1SyncConfig{Interval: 5 * time.Second, BatchSize: 500}
	syncer := NewD1Syncer(local, remote, cfg)

	ctx := context.Background()
	_, err := local.Put(ctx, []byte("status-test-blob"), ContentMeta{Type: ContentConfig})
	if err != nil {
		t.Fatalf("local.Put: %v", err)
	}

	syncer.syncOnce(ctx)

	status := syncer.Status()
	if !status.Healthy {
		t.Error("expected Healthy=true after successful sync")
	}
	if status.LastCursor <= 0 {
		t.Errorf("expected LastCursor > 0, got %d", status.LastCursor)
	}
}

// TestD1SyncerRunCancellation starts Run in a goroutine, cancels the context,
// and verifies the loop exits cleanly within a reasonable timeout.
func TestD1SyncerRunCancellation(t *testing.T) {
	local := newTestStore(t)
	remote := newTestStore(t)
	cfg := D1SyncConfig{Interval: 50 * time.Millisecond, BatchSize: 500}
	syncer := NewD1Syncer(local, remote, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		syncer.Run(ctx)
		close(done)
	}()

	// Let one tick fire, then cancel.
	time.Sleep(80 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Exited cleanly.
	case <-time.After(2 * time.Second):
		t.Error("Run did not exit within 2s after context cancellation")
	}
}
