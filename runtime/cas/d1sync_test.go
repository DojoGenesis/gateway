package cas

import (
	"context"
	"io"
	"testing"
	"time"
)

// mockLocalStore implements Store + deltaQuerier for testing.
type mockLocalStore struct {
	entries []DeltaEntry
	putLog  [][]byte
}

func (m *mockLocalStore) Put(_ context.Context, content []byte, meta ContentMeta) (Ref, error) {
	m.putLog = append(m.putLog, content)
	return computeHash(content), nil
}

func (m *mockLocalStore) Get(_ context.Context, _ Ref) ([]byte, ContentMeta, error) {
	return nil, ContentMeta{}, ErrNotFound
}

func (m *mockLocalStore) Has(_ context.Context, _ Ref) (bool, error) {
	return false, nil
}

func (m *mockLocalStore) Tag(_ context.Context, _, _ string, _ Ref) error {
	return nil
}

func (m *mockLocalStore) Resolve(_ context.Context, _, _ string) (Ref, error) {
	return "", ErrNotFound
}

func (m *mockLocalStore) Untag(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockLocalStore) List(_ context.Context, _ string) ([]TagEntry, error) {
	return nil, nil
}

func (m *mockLocalStore) GC(_ context.Context) (GCResult, error) {
	return GCResult{}, nil
}

func (m *mockLocalStore) Export(_ context.Context, _ []Ref, _ io.Writer) error {
	return nil
}

func (m *mockLocalStore) Import(_ context.Context, _ io.Reader) ([]Ref, error) {
	return nil, nil
}

func (m *mockLocalStore) Close() error {
	return nil
}

// deltaQuerier implementation for mockLocalStore.
func (m *mockLocalStore) Delta(_ context.Context, since int64, limit int) ([]DeltaEntry, error) {
	var result []DeltaEntry
	for _, e := range m.entries {
		if e.SyncCursor > since {
			result = append(result, e)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockLocalStore) MaxSyncCursor(_ context.Context) (int64, error) {
	var max int64
	for _, e := range m.entries {
		if e.SyncCursor > max {
			max = e.SyncCursor
		}
	}
	return max, nil
}

// mockRemoteStore captures puts for verification.
type mockRemoteStore struct {
	mockLocalStore
	received []Ref
}

func (m *mockRemoteStore) Put(_ context.Context, content []byte, meta ContentMeta) (Ref, error) {
	ref := computeHash(content)
	m.received = append(m.received, ref)
	return ref, nil
}

func TestD1SyncerSyncsEntries(t *testing.T) {
	local := &mockLocalStore{
		entries: []DeltaEntry{
			{Hash: "aaa", Data: []byte("hello"), SyncCursor: 1, Meta: ContentMeta{Type: ContentSkill}},
			{Hash: "bbb", Data: []byte("world"), SyncCursor: 2, Meta: ContentMeta{Type: ContentSkill}},
		},
	}
	remote := &mockRemoteStore{}

	cfg := D1SyncConfig{Interval: 50 * time.Millisecond, BatchSize: 100}
	syncer := NewD1Syncer(local, remote, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	syncer.Run(ctx)

	if len(remote.received) != 2 {
		t.Fatalf("expected 2 synced entries, got %d", len(remote.received))
	}

	status := syncer.Status()
	if status.LastCursor != 2 {
		t.Errorf("expected cursor=2, got %d", status.LastCursor)
	}
	if !status.Healthy {
		t.Error("expected healthy=true")
	}
}

func TestD1SyncerStatusWhenEmpty(t *testing.T) {
	local := &mockLocalStore{}
	remote := &mockRemoteStore{}

	cfg := D1SyncConfig{Interval: 50 * time.Millisecond, BatchSize: 100}
	syncer := NewD1Syncer(local, remote, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	syncer.Run(ctx)

	status := syncer.Status()
	if status.LastCursor != 0 {
		t.Errorf("expected cursor=0, got %d", status.LastCursor)
	}
	if !status.Healthy {
		t.Error("expected healthy=true when nothing to sync")
	}
}

func TestDefaultD1SyncConfig(t *testing.T) {
	cfg := DefaultD1SyncConfig()
	if cfg.Interval != 5*time.Second {
		t.Errorf("expected 5s interval, got %v", cfg.Interval)
	}
	if cfg.BatchSize != 500 {
		t.Errorf("expected 500 batch size, got %d", cfg.BatchSize)
	}
}
