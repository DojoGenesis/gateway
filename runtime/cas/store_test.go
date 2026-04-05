package cas_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/runtime/cas"
)

func newTestStore(t *testing.T) cas.Store {
	t.Helper()
	store, err := cas.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewSQLiteStore(t *testing.T) {
	store, err := cas.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore returned error: %v", err)
	}
	if store == nil {
		t.Fatal("NewSQLiteStore returned nil")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestPutGetRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	content := []byte("hello world")
	meta := cas.ContentMeta{Type: cas.ContentSkill, CreatedBy: "test"}

	ref, err := store.Put(ctx, content, meta)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if ref == "" {
		t.Fatal("Put returned empty ref")
	}

	got, gotMeta, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("Get content: got %q, want %q", got, content)
	}
	if gotMeta.Type != cas.ContentSkill {
		t.Errorf("Get meta type: got %q, want %q", gotMeta.Type, cas.ContentSkill)
	}
	if gotMeta.Size != int64(len(content)) {
		t.Errorf("Get meta size: got %d, want %d", gotMeta.Size, len(content))
	}
}

func TestDeduplication(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	content := []byte("same content")
	ref1, err := store.Put(ctx, content, cas.ContentMeta{Type: cas.ContentSkill})
	if err != nil {
		t.Fatalf("Put 1: %v", err)
	}
	ref2, err := store.Put(ctx, content, cas.ContentMeta{Type: cas.ContentConfig})
	if err != nil {
		t.Fatalf("Put 2: %v", err)
	}
	if ref1 != ref2 {
		t.Errorf("Deduplication failed: %s != %s", ref1, ref2)
	}
}

func TestHas(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	exists, err := store.Has(ctx, cas.Ref("nonexistent"))
	if err != nil {
		t.Fatalf("Has: %v", err)
	}
	if exists {
		t.Error("Has returned true for nonexistent ref")
	}

	ref, _ := store.Put(ctx, []byte("data"), cas.ContentMeta{Type: cas.ContentSkill})
	exists, err = store.Has(ctx, ref)
	if err != nil {
		t.Fatalf("Has: %v", err)
	}
	if !exists {
		t.Error("Has returned false for existing ref")
	}
}

func TestGetNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, _, err := store.Get(ctx, cas.Ref("nonexistent"))
	if !errors.Is(err, cas.ErrNotFound) {
		t.Errorf("Get nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestTagAndResolve(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	ref, _ := store.Put(ctx, []byte("skill content"), cas.ContentMeta{Type: cas.ContentSkill})

	if err := store.Tag(ctx, "skill/analyze", "1.0.0", ref); err != nil {
		t.Fatalf("Tag: %v", err)
	}

	resolved, err := store.Resolve(ctx, "skill/analyze", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved != ref {
		t.Errorf("Resolve: got %s, want %s", resolved, ref)
	}
}

func TestResolveNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Resolve(ctx, "nonexistent", "1.0.0")
	if !errors.Is(err, cas.ErrNotFound) {
		t.Errorf("Resolve nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	ref1, _ := store.Put(ctx, []byte("skill a"), cas.ContentMeta{Type: cas.ContentSkill})
	ref2, _ := store.Put(ctx, []byte("skill b"), cas.ContentMeta{Type: cas.ContentSkill})
	ref3, _ := store.Put(ctx, []byte("config c"), cas.ContentMeta{Type: cas.ContentConfig})

	store.Tag(ctx, "skill/a", "1.0.0", ref1)
	store.Tag(ctx, "skill/b", "1.0.0", ref2)
	store.Tag(ctx, "config/c", "1.0.0", ref3)

	entries, err := store.List(ctx, "skill/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List: got %d entries, want 2", len(entries))
	}
}

func TestGC(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Untagged content should be collected
	store.Put(ctx, []byte("orphan"), cas.ContentMeta{Type: cas.ContentSkill})

	// Tagged content should be preserved
	ref, _ := store.Put(ctx, []byte("tagged"), cas.ContentMeta{Type: cas.ContentSkill})
	store.Tag(ctx, "skill/keep", "1.0.0", ref)

	result, err := store.GC(ctx)
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	if result.Removed != 1 {
		t.Errorf("GC removed: got %d, want 1", result.Removed)
	}

	// Tagged content still accessible
	exists, _ := store.Has(ctx, ref)
	if !exists {
		t.Error("GC removed tagged content")
	}
}

func TestExportImport(t *testing.T) {
	store1 := newTestStore(t)
	ctx := context.Background()

	ref, _ := store1.Put(ctx, []byte("export me"), cas.ContentMeta{Type: cas.ContentSkill, CreatedBy: "test"})

	var buf bytes.Buffer
	if err := store1.Export(ctx, []cas.Ref{ref}, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	store2 := newTestStore(t)
	refs, err := store2.Import(ctx, &buf)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("Import: got %d refs, want 1", len(refs))
	}

	got, meta, err := store2.Get(ctx, refs[0])
	if err != nil {
		t.Fatalf("Get after import: %v", err)
	}
	if !bytes.Equal(got, []byte("export me")) {
		t.Errorf("Content after import: got %q, want %q", got, "export me")
	}
	if meta.Type != cas.ContentSkill {
		t.Errorf("Meta type after import: got %q, want %q", meta.Type, cas.ContentSkill)
	}
}

func TestConcurrentPut(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte("concurrent data " + string(rune('A'+i%26)))
			_, err := store.Put(ctx, data, cas.ContentMeta{Type: cas.ContentSkill})
			if err != nil {
				t.Errorf("concurrent Put %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
}
