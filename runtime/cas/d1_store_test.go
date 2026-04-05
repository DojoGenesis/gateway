package cas

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock D1 server
// ---------------------------------------------------------------------------

// d1MockDB is an in-memory store that backs the mock D1 HTTP server.
type d1MockDB struct {
	content map[string]d1MockContent
	tags    map[string]d1MockTag // key: name+"@"+version
}

type d1MockContent struct {
	Hash      string
	Data      string // base64-encoded
	Meta      string
	CreatedAt string
}

type d1MockTag struct {
	Name      string
	Version   string
	Ref       string
	CreatedAt string
}

func newD1MockDB() *d1MockDB {
	return &d1MockDB{
		content: make(map[string]d1MockContent),
		tags:    make(map[string]d1MockTag),
	}
}

// d1Response wraps any result in the D1 response envelope.
func d1Response(rows []map[string]any) []byte {
	resp := map[string]any{
		"success": true,
		"errors":  []any{},
		"result": []map[string]any{
			{
				"results": rows,
				"success": true,
			},
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// newD1MockServer starts an httptest.Server that handles D1 /query requests
// by dispatching to the provided mock DB.
func newD1MockServer(db *d1MockDB) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			SQL    string `json:"sql"`
			Params []any  `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		rows := dispatchD1(db, req.SQL, req.Params)
		w.Write(d1Response(rows))
	}))
}

// dispatchD1 is a very small SQL dispatcher for the mock server.
// It handles the exact SQL patterns emitted by d1_store.go.
func dispatchD1(db *d1MockDB, sql string, params []any) []map[string]any {
	str := func(i int) string {
		if i >= len(params) {
			return ""
		}
		s, _ := params[i].(string)
		return s
	}

	switch {
	// PUT: INSERT INTO content ...
	case hasPrefix(sql, "INSERT INTO content"):
		hash := str(0)
		db.content[hash] = d1MockContent{
			Hash:      hash,
			Data:      str(1),
			Meta:      str(2),
			CreatedAt: str(3),
		}
		return nil

	// GET: SELECT data, meta FROM content WHERE hash = ?
	case hasPrefix(sql, "SELECT data, meta FROM content"):
		hash := str(0)
		c, ok := db.content[hash]
		if !ok {
			return nil
		}
		return []map[string]any{{"data": c.Data, "meta": c.Meta}}

	// HAS: SELECT COUNT(*) AS n FROM content WHERE hash = ?
	case hasPrefix(sql, "SELECT COUNT(*) AS n FROM content"):
		hash := str(0)
		_, ok := db.content[hash]
		n := 0
		if ok {
			n = 1
		}
		return []map[string]any{{"n": float64(n)}}

	// TAG: INSERT INTO tags ...
	case hasPrefix(sql, "INSERT INTO tags"):
		name, version, ref := str(0), str(1), str(2)
		db.tags[name+"@"+version] = d1MockTag{
			Name: name, Version: version, Ref: ref, CreatedAt: str(3),
		}
		return nil

	// RESOLVE: SELECT ref FROM tags WHERE name = ? AND version = ?
	case hasPrefix(sql, "SELECT ref FROM tags"):
		name, version := str(0), str(1)
		t, ok := db.tags[name+"@"+version]
		if !ok {
			return nil
		}
		return []map[string]any{{"ref": t.Ref}}

	// LIST: SELECT t.name, t.version, t.ref, c.meta FROM tags t JOIN content c ...
	case hasPrefix(sql, "SELECT t.name"):
		prefix := str(0)
		prefix = prefix[:len(prefix)-1] // strip trailing %
		var rows []map[string]any
		for _, t := range db.tags {
			if len(t.Name) >= len(prefix) && t.Name[:len(prefix)] == prefix {
				c := db.content[t.Ref]
				rows = append(rows, map[string]any{
					"name": t.Name, "version": t.Version,
					"ref": t.Ref, "meta": c.Meta,
				})
			}
		}
		return rows

	// GC query: SELECT c.hash, LENGTH(c.data) AS sz FROM content c LEFT JOIN tags ...
	case hasPrefix(sql, "SELECT c.hash"):
		var rows []map[string]any
		for _, c := range db.content {
			referenced := false
			for _, t := range db.tags {
				if t.Ref == c.Hash {
					referenced = true
					break
				}
			}
			if !referenced {
				rows = append(rows, map[string]any{
					"hash": c.Hash,
					"sz":   float64(len(c.Data)),
				})
			}
		}
		return rows

	// GC delete: DELETE FROM content WHERE hash IN (...)
	case hasPrefix(sql, "DELETE FROM content"):
		for _, p := range params {
			h, _ := p.(string)
			delete(db.content, h)
		}
		return nil
	}

	return nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func newD1TestStore(t *testing.T) (Store, *d1MockDB, *httptest.Server) {
	t.Helper()
	db := newD1MockDB()
	srv := newD1MockServer(db)
	store, err := NewD1Store(D1Config{
		AccountID:  "test-account",
		DatabaseID: "test-db",
		APIToken:   "test-token",
		BaseURL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("NewD1Store: %v", err)
	}
	t.Cleanup(srv.Close)
	return store, db, srv
}

func TestD1Store_PutGet(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	ctx := context.Background()

	content := []byte("hello d1 world")
	meta := ContentMeta{Type: ContentSkill, CreatedAt: time.Now().UTC()}

	ref, err := store.Put(ctx, content, meta)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if ref == "" {
		t.Fatal("Put: empty ref")
	}

	got, gotMeta, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("Get: got %q, want %q", got, content)
	}
	if gotMeta.Type != meta.Type {
		t.Errorf("Get meta.Type: got %q, want %q", gotMeta.Type, meta.Type)
	}
}

func TestD1Store_GetNotFound(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	_, _, err := store.Get(context.Background(), Ref("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("Get nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestD1Store_Has(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	ctx := context.Background()

	ok, err := store.Has(ctx, Ref("missing"))
	if err != nil {
		t.Fatalf("Has missing: %v", err)
	}
	if ok {
		t.Error("Has missing: expected false")
	}

	ref, _ := store.Put(ctx, []byte("data"), ContentMeta{Type: ContentConfig})
	ok, err = store.Has(ctx, ref)
	if err != nil {
		t.Fatalf("Has present: %v", err)
	}
	if !ok {
		t.Error("Has present: expected true")
	}
}

func TestD1Store_TagResolve(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	ctx := context.Background()

	content := []byte("skill content")
	ref, _ := store.Put(ctx, content, ContentMeta{Type: ContentSkill})

	if err := store.Tag(ctx, "skill/echo", "1.0.0", ref); err != nil {
		t.Fatalf("Tag: %v", err)
	}

	got, err := store.Resolve(ctx, "skill/echo", "1.0.0")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != ref {
		t.Errorf("Resolve: got %q, want %q", got, ref)
	}
}

func TestD1Store_TagNotFound(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	_, err := store.Resolve(context.Background(), "skill/missing", "1.0.0")
	if err != ErrNotFound {
		t.Errorf("Resolve missing: got %v, want ErrNotFound", err)
	}
}

func TestD1Store_TagRefNotFound(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	err := store.Tag(context.Background(), "skill/echo", "1.0.0", Ref("nonexistent"))
	if err == nil {
		t.Error("Tag nonexistent ref: expected error")
	}
}

func TestD1Store_List(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	ctx := context.Background()

	ref1, _ := store.Put(ctx, []byte("skill-a"), ContentMeta{Type: ContentSkill})
	ref2, _ := store.Put(ctx, []byte("skill-b"), ContentMeta{Type: ContentSkill})
	store.Tag(ctx, "skill/alpha", "1.0.0", ref1)
	store.Tag(ctx, "skill/beta", "1.0.0", ref2)
	store.Tag(ctx, "workflow/main", "1.0.0", ref1)

	entries, err := store.List(ctx, "skill/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("List skill/: got %d entries, want 2", len(entries))
	}

	entries, err = store.List(ctx, "workflow/")
	if err != nil {
		t.Fatalf("List workflow/: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("List workflow/: got %d entries, want 1", len(entries))
	}
}

func TestD1Store_GC(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	ctx := context.Background()

	// Put two items; tag only one.
	ref1, _ := store.Put(ctx, []byte("referenced"), ContentMeta{Type: ContentSkill})
	store.Put(ctx, []byte("orphan"), ContentMeta{Type: ContentSkill})
	store.Tag(ctx, "skill/used", "1.0.0", ref1)

	result, err := store.GC(ctx)
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	if result.Removed != 1 {
		t.Errorf("GC.Removed: got %d, want 1", result.Removed)
	}

	// Referenced item should still be retrievable.
	_, _, err = store.Get(ctx, ref1)
	if err != nil {
		t.Errorf("Get after GC: %v", err)
	}
}

func TestD1Store_PutIdempotent(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	ctx := context.Background()

	content := []byte("same content")
	meta := ContentMeta{Type: ContentConfig}

	ref1, err := store.Put(ctx, content, meta)
	if err != nil {
		t.Fatalf("Put #1: %v", err)
	}
	ref2, err := store.Put(ctx, content, meta)
	if err != nil {
		t.Fatalf("Put #2: %v", err)
	}
	if ref1 != ref2 {
		t.Errorf("idempotent Put: got different refs %q vs %q", ref1, ref2)
	}
}

func TestD1Store_Base64RoundTrip(t *testing.T) {
	// Verify raw binary content survives the base64 round-trip through the mock.
	store, db, _ := newD1TestStore(t)
	ctx := context.Background()

	binary := []byte{0x00, 0xFF, 0x01, 0xFE, 0x80, 0x7F}
	ref, err := store.Put(ctx, binary, ContentMeta{Type: ContentSkill})
	if err != nil {
		t.Fatalf("Put binary: %v", err)
	}

	// Verify the stored value in the mock DB is valid base64.
	c := db.content[string(ref)]
	if _, err := base64.StdEncoding.DecodeString(c.Data); err != nil {
		t.Errorf("stored data is not valid base64: %v", err)
	}

	got, _, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get binary: %v", err)
	}
	if string(got) != string(binary) {
		t.Errorf("binary round-trip: got %v, want %v", got, binary)
	}
}

func TestD1Store_Close(t *testing.T) {
	store, _, _ := newD1TestStore(t)
	if err := store.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestNewD1Store_MissingConfig(t *testing.T) {
	_, err := NewD1Store(D1Config{})
	if err == nil {
		t.Error("NewD1Store with empty config: expected error")
	}
}
