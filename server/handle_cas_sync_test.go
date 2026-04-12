package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// setupSyncTestRouter creates a minimal router wired to the three CAS sync handlers.
// It returns the router and the underlying in-memory CAS store for test setup.
func setupSyncTestRouter(t *testing.T) (*gin.Engine, cas.Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store, err := cas.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	s := &Server{
		workflowCAS: store,
		cfg: &ServerConfig{
			Port:        "7340",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	r := gin.New()
	// Inject a request ID so errorResponse includes it.
	r.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})
	r.GET("/api/cas/delta", s.handleCASDelta)
	r.PUT("/api/cas/batch", s.handleCASBatch)
	r.GET("/api/cas/status", s.handleCASSyncStatus)
	return r, store
}

// ─── handleCASDelta ──────────────────────────────────────────────────────────

// TestHandleCASDelta_Empty verifies that GET /api/cas/delta on an empty store
// returns HTTP 200 with an empty entries array.
func TestHandleCASDelta_Empty(t *testing.T) {
	r, _ := setupSyncTestRouter(t)

	req, _ := http.NewRequest(http.MethodGet, "/api/cas/delta", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	entries, ok := body["entries"]
	if !ok {
		t.Fatal("response missing 'entries' field")
	}
	// entries should be null or an empty array — count must be 0.
	count, _ := body["count"].(float64)
	if count != 0 {
		t.Errorf("expected count=0, got %v", count)
	}
	if entries != nil {
		arr, ok := entries.([]interface{})
		if !ok {
			t.Fatalf("entries is not an array: %T", entries)
		}
		if len(arr) != 0 {
			t.Errorf("expected empty entries array, got %d items", len(arr))
		}
	}
}

// TestHandleCASDelta_WithData puts blobs into the store and verifies the delta
// endpoint returns them when queried with since=0.
func TestHandleCASDelta_WithData(t *testing.T) {
	r, store := setupSyncTestRouter(t)

	ctx := t.Context()
	blobs := [][]byte{
		[]byte("delta-blob-one"),
		[]byte("delta-blob-two"),
		[]byte("delta-blob-three"),
	}
	for _, b := range blobs {
		if _, err := store.Put(ctx, b, cas.ContentMeta{Type: cas.ContentSkill}); err != nil {
			t.Fatalf("store.Put: %v", err)
		}
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/cas/delta?since=0&limit=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	count, _ := body["count"].(float64)
	if int(count) != len(blobs) {
		t.Errorf("expected count=%d, got %v", len(blobs), count)
	}

	entries, ok := body["entries"].([]interface{})
	if !ok {
		t.Fatalf("entries is not an array")
	}
	if len(entries) != len(blobs) {
		t.Errorf("expected %d entries, got %d", len(blobs), len(entries))
	}
}

// TestHandleCASDelta_Pagination puts 10 blobs in the store, requests limit=5,
// and expects exactly 5 entries returned.
func TestHandleCASDelta_Pagination(t *testing.T) {
	r, store := setupSyncTestRouter(t)

	ctx := t.Context()
	for i := 0; i < 10; i++ {
		data := []byte(fmt.Sprintf("pagination-blob-%02d", i))
		if _, err := store.Put(ctx, data, cas.ContentMeta{Type: cas.ContentSkill}); err != nil {
			t.Fatalf("store.Put: %v", err)
		}
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/cas/delta?since=0&limit=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	entries, ok := body["entries"].([]interface{})
	if !ok {
		t.Fatalf("entries is not an array")
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries with limit=5, got %d", len(entries))
	}
}

// TestHandleCASDelta_InvalidParams verifies that malformed since/limit values
// return HTTP 400.
func TestHandleCASDelta_InvalidParams(t *testing.T) {
	r, _ := setupSyncTestRouter(t)

	cases := []struct {
		name  string
		query string
	}{
		{"bad since", "/api/cas/delta?since=notanint"},
		{"zero limit", "/api/cas/delta?since=0&limit=0"},
		{"limit too large", "/api/cas/delta?since=0&limit=9999"},
		{"negative limit", "/api/cas/delta?since=0&limit=-1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

// ─── handleCASBatch ──────────────────────────────────────────────────────────

// TestHandleCASBatch_Success sends a valid batch and expects HTTP 200 with stored hashes.
func TestHandleCASBatch_Success(t *testing.T) {
	r, _ := setupSyncTestRouter(t)

	payloads := []string{"batch-blob-a", "batch-blob-b"}
	entries := make([]map[string]string, len(payloads))
	for i, p := range payloads {
		entries[i] = map[string]string{
			"hash":         fmt.Sprintf("placeholder-%d", i),
			"content_type": "skill",
			"data_base64":  base64.StdEncoding.EncodeToString([]byte(p)),
		}
	}

	body, _ := json.Marshal(map[string]interface{}{"entries": entries})
	req, _ := http.NewRequest(http.MethodPut, "/api/cas/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	success, _ := resp["success"].(float64)
	if int(success) != len(payloads) {
		t.Errorf("expected success=%d, got %v", len(payloads), success)
	}

	stored, ok := resp["stored"].([]interface{})
	if !ok {
		t.Fatalf("stored is not an array")
	}
	if len(stored) != len(payloads) {
		t.Errorf("expected %d stored hashes, got %d", len(payloads), len(stored))
	}
}

// TestHandleCASBatch_EmptyEntries verifies that an empty entries array returns 400.
func TestHandleCASBatch_EmptyEntries(t *testing.T) {
	r, _ := setupSyncTestRouter(t)

	body := []byte(`{"entries": []}`)
	req, _ := http.NewRequest(http.MethodPut, "/api/cas/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestHandleCASBatch_InvalidBase64 sends an entry with invalid base64 data and expects
// a partial-content response (the bad entry appears in errors, not stored).
func TestHandleCASBatch_InvalidBase64(t *testing.T) {
	r, _ := setupSyncTestRouter(t)

	entries := []map[string]string{
		{
			"hash":         "valid-hash",
			"content_type": "skill",
			"data_base64":  base64.StdEncoding.EncodeToString([]byte("valid-blob")),
		},
		{
			"hash":         "bad-hash",
			"content_type": "skill",
			"data_base64":  "!!!not-valid-base64!!!",
		},
	}
	body, _ := json.Marshal(map[string]interface{}{"entries": entries})
	req, _ := http.NewRequest(http.MethodPut, "/api/cas/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// One succeeded, one failed: expect 206 Partial Content.
	if w.Code != http.StatusPartialContent {
		t.Errorf("expected 206 Partial Content, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	failed, _ := resp["failed"].(float64)
	if failed != 1 {
		t.Errorf("expected failed=1, got %v", failed)
	}
	success, _ := resp["success"].(float64)
	if success != 1 {
		t.Errorf("expected success=1, got %v", success)
	}
}

// TestHandleCASBatch_TooMany sends 501 entries and expects 400.
func TestHandleCASBatch_TooMany(t *testing.T) {
	r, _ := setupSyncTestRouter(t)

	entries := make([]map[string]string, 501)
	for i := range entries {
		entries[i] = map[string]string{
			"hash":         fmt.Sprintf("hash-%d", i),
			"content_type": "skill",
			"data_base64":  base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("blob-%d", i))),
		}
	}
	body, _ := json.Marshal(map[string]interface{}{"entries": entries})
	req, _ := http.NewRequest(http.MethodPut, "/api/cas/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── handleCASSyncStatus ─────────────────────────────────────────────────────

// TestHandleCASSyncStatus_NoSyncer verifies that when d1Syncer is nil,
// the status endpoint returns HTTP 200 with sync_enabled=false.
func TestHandleCASSyncStatus_NoSyncer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store, err := cas.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	// d1Syncer intentionally nil.
	s := &Server{
		workflowCAS: store,
		cfg: &ServerConfig{
			Port:        "7340",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}
	r := gin.New()
	r.GET("/api/cas/status", s.handleCASSyncStatus)

	req, _ := http.NewRequest(http.MethodGet, "/api/cas/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	syncEnabled, ok := resp["sync_enabled"].(bool)
	if !ok || syncEnabled {
		t.Errorf("expected sync_enabled=false, got %v", resp["sync_enabled"])
	}
}

// TestHandleCASSyncStatus_WithSyncer verifies that when d1Syncer is set,
// the status endpoint returns HTTP 200 with sync_enabled=true and health fields.
func TestHandleCASSyncStatus_WithSyncer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	local, err := cas.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { local.Close() })

	remote, err := cas.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { remote.Close() })

	syncer := cas.NewD1Syncer(local, remote, cas.D1SyncConfig{
		Interval:  5 * time.Second,
		BatchSize: 500,
	})

	s := &Server{
		workflowCAS: local,
		d1Syncer:    syncer,
		cfg: &ServerConfig{
			Port:        "7340",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}
	r := gin.New()
	r.GET("/api/cas/status", s.handleCASSyncStatus)

	req, _ := http.NewRequest(http.MethodGet, "/api/cas/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	syncEnabled, ok := resp["sync_enabled"].(bool)
	if !ok || !syncEnabled {
		t.Errorf("expected sync_enabled=true, got %v", resp["sync_enabled"])
	}

	// Verify that the known health fields are present.
	for _, field := range []string{"last_cursor", "healthy", "lag_seconds"} {
		if _, exists := resp[field]; !exists {
			t.Errorf("expected field %q in sync status response", field)
		}
	}
}

// Ensure the strings import is used (only in invalid-params test via string building).
var _ = strings.Contains
