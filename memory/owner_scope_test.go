package memory

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/pkg/gateway"
	_ "modernc.org/sqlite"
)

func newScopeTestManager(t *testing.T) *MemoryManager {
	t.Helper()
	mm, err := NewMemoryManager(filepath.Join(t.TempDir(), "scope.db"))
	if err != nil {
		t.Fatalf("NewMemoryManager: %v", err)
	}
	t.Cleanup(func() { mm.Close() })
	return mm
}

func storeOwned(t *testing.T, mm *MemoryManager, id, owner, content string) {
	t.Helper()
	now := time.Now()
	if err := mm.Store(context.Background(), Memory{
		ID: id, Type: "note", Content: content, OwnerID: owner,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("store %s: %v", id, err)
	}
}

func TestOwnerScope_CrossOwnerAccessIsRefused(t *testing.T) {
	ctx := context.Background()
	mm := newScopeTestManager(t)
	storeOwned(t, mm, "a1", "alice", "alice zebra")
	storeOwned(t, mm, "b1", "bob", "bob zebra")
	bob := ForOwner("bob")

	if _, err := mm.GetMemoryScoped(ctx, bob, "a1"); !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("get: want ErrMemoryNotFound, got %v", err)
	}
	if _, err := mm.UpdateScoped(ctx, bob, "a1", "overwritten", nil); !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("update: want ErrMemoryNotFound, got %v", err)
	}
	if err := mm.DeleteMemoryScoped(ctx, bob, "a1"); !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("delete: want ErrMemoryNotFound, got %v", err)
	}

	results, total, err := mm.SearchMemoriesPageScoped(ctx, bob, "zebra", 10, 0)
	if err != nil || total != 1 || len(results) != 1 || results[0].Memory.ID != "b1" {
		t.Errorf("search: want only b1, got total=%d results=%v err=%v", total, results, err)
	}
	results, total, err = mm.SearchMemoriesPageScoped(ctx, bob, "", 10, 0)
	if err != nil || total != 1 || results[0].Memory.ID != "b1" {
		t.Errorf("empty-query search: want only b1, got total=%d err=%v", total, err)
	}

	listed, err := mm.ListMemoriesScoped(ctx, bob, MemoryFilter{})
	if err != nil || len(listed) != 1 || listed[0].ID != "b1" {
		t.Errorf("list: want only b1, got %v err=%v", listed, err)
	}
	if n, err := mm.CountMemoriesScoped(ctx, bob, MemoryFilter{}); err != nil || n != 1 {
		t.Errorf("count: want 1, got %d err=%v", n, err)
	}

	got, err := mm.GetMemoryScoped(ctx, ForOwner("alice"), "a1")
	if err != nil || got.Content != "alice zebra" || got.OwnerID != "alice" {
		t.Errorf("alice's memory changed or vanished: %+v err=%v", got, err)
	}
}

func TestOwnerScope_ZeroAndEmptyScopesFailClosed(t *testing.T) {
	ctx := context.Background()
	mm := newScopeTestManager(t)
	storeOwned(t, mm, "legacy", LegacyOwnerID, "legacy row")

	for name, scope := range map[string]OwnerScope{"zero": {}, "empty owner": ForOwner("")} {
		if _, err := mm.GetMemoryScoped(ctx, scope, "legacy"); !errors.Is(err, ErrInvalidScope) {
			t.Errorf("%s get: want ErrInvalidScope, got %v", name, err)
		}
		if _, _, err := mm.SearchMemoriesPageScoped(ctx, scope, "", 10, 0); !errors.Is(err, ErrInvalidScope) {
			t.Errorf("%s search: want ErrInvalidScope, got %v", name, err)
		}
		if _, err := mm.ListMemoriesScoped(ctx, scope, MemoryFilter{}); !errors.Is(err, ErrInvalidScope) {
			t.Errorf("%s list: want ErrInvalidScope, got %v", name, err)
		}
		if err := mm.DeleteMemoryScoped(ctx, scope, "legacy"); !errors.Is(err, ErrInvalidScope) {
			t.Errorf("%s delete: want ErrInvalidScope, got %v", name, err)
		}
	}

	if _, err := mm.GetMemoryScoped(ctx, AllOwners(), "legacy"); err != nil {
		t.Errorf("AllOwners should reach legacy rows: %v", err)
	}
}

func TestStore_IDCollisionNeverMovesOwnership(t *testing.T) {
	ctx := context.Background()
	mm := newScopeTestManager(t)
	storeOwned(t, mm, "shared-id", "alice", "alice original")

	now := time.Now()
	err := mm.Store(ctx, Memory{ID: "shared-id", Type: "note", Content: "bob takeover", OwnerID: "bob", CreatedAt: now, UpdatedAt: now})
	if !errors.Is(err, ErrMemoryOwnerConflict) {
		t.Fatalf("want ErrMemoryOwnerConflict, got %v", err)
	}
	got, err := mm.GetMemory(ctx, "shared-id")
	if err != nil || got.OwnerID != "alice" || got.Content != "alice original" {
		t.Errorf("collision changed the row: %+v err=%v", got, err)
	}

	// Same owner re-storing the same ID is still an upsert.
	storeOwned(t, mm, "shared-id", "alice", "alice edited")
	got, _ = mm.GetMemory(ctx, "shared-id")
	if got.Content != "alice edited" {
		t.Errorf("same-owner upsert did not apply: %q", got.Content)
	}
}

func TestDeleteAllForOwner(t *testing.T) {
	ctx := context.Background()
	mm := newScopeTestManager(t)
	storeOwned(t, mm, "a1", "alice", "one")
	storeOwned(t, mm, "a2", "alice", "two")
	storeOwned(t, mm, "b1", "bob", "three")
	storeOwned(t, mm, "l1", LegacyOwnerID, "four")

	if _, err := mm.DeleteAllForOwner(ctx, LegacyOwnerID); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("empty owner must be refused, got %v", err)
	}

	n, err := mm.DeleteAllForOwner(ctx, "alice")
	if err != nil || n != 2 {
		t.Fatalf("want 2 deleted, got %d err=%v", n, err)
	}
	if left, _ := mm.CountMemories(ctx, MemoryFilter{}); left != 2 {
		t.Errorf("want bob's and the legacy row left, got %d rows", left)
	}
	if n, _ := mm.CountMemoriesScoped(ctx, ForOwner("alice"), MemoryFilter{}); n != 0 {
		t.Errorf("alice still has %d memories", n)
	}
}

func TestMigrateOwnerColumn_PreDGS163Database(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	// The schema exactly as it was before DGS-163.
	if _, err := db.Exec(`
		CREATE TABLE memories (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			metadata TEXT,
			embedding BLOB,
			context_type TEXT DEFAULT 'conversation',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		INSERT INTO memories (id, type, content, metadata, context_type, created_at, updated_at)
		VALUES ('old-1', 'note', 'written before owners existed', '{}', 'private', datetime('now'), datetime('now'));
	`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Opening twice proves the migration is idempotent.
	for i := 0; i < 2; i++ {
		mm, err := NewMemoryManager(path)
		if err != nil {
			t.Fatalf("open %d: %v", i, err)
		}
		got, err := mm.GetMemoryScoped(context.Background(), AllOwners(), "old-1")
		if err != nil || got.OwnerID != LegacyOwnerID {
			t.Fatalf("open %d: legacy row: %+v err=%v", i, got, err)
		}
		if _, err := mm.GetMemoryScoped(context.Background(), ForOwner("anyone"), "old-1"); !errors.Is(err, ErrMemoryNotFound) {
			t.Errorf("open %d: a subject must not see legacy rows, got %v", i, err)
		}
		var idx int
		if err := mm.DB().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_memories_owner_updated'`).Scan(&idx); err != nil || idx != 1 {
			t.Errorf("open %d: owner index missing (n=%d err=%v)", i, idx, err)
		}
		mm.Close()
	}
}

func TestGatewayMemoryStore_ScopesByContextOwner(t *testing.T) {
	mm := newScopeTestManager(t)
	store := NewGatewayMemoryStore(mm)

	if err := store.Store(context.Background(), &gateway.MemoryEntry{EntryType: "note", Content: "x"}); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("store without an owner on ctx: want ErrInvalidScope, got %v", err)
	}

	alice := WithOwner(context.Background(), "alice")
	bob := WithOwner(context.Background(), "bob")
	entry := &gateway.MemoryEntry{EntryType: "note", Content: "alice zebra"}
	if err := store.Store(alice, entry); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(bob, entry.ID); err == nil {
		t.Error("bob read alice's entry")
	}
	if got, _ := store.Search(bob, &gateway.SearchQuery{Text: "zebra"}, 10); len(got) != 0 {
		t.Errorf("bob's search returned %d of alice's entries", len(got))
	}
	if err := store.Delete(bob, entry.ID); err == nil {
		t.Error("bob deleted alice's entry")
	}
	if _, err := store.Get(alice, entry.ID); err != nil {
		t.Errorf("alice lost her entry: %v", err)
	}
}
