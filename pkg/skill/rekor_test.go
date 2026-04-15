package skill

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock Rekor server
// ---------------------------------------------------------------------------

type rekorMockServer struct {
	// uuids maps sha256:<hash> → []uuid
	uuids map[string][]string
	// entries maps uuid → raw entry map
	entries map[string]map[string]any
}

func newRekorMock() *rekorMockServer {
	return &rekorMockServer{
		uuids:   make(map[string][]string),
		entries: make(map[string]map[string]any),
	}
}

// addEntry registers a fake Rekor entry.
func (m *rekorMockServer) addEntry(hash, uuid string, logIndex int64, integratedTime int64) {
	m.uuids["sha256:"+hash] = append(m.uuids["sha256:"+hash], uuid)
	m.entries[uuid] = map[string]any{
		uuid: map[string]any{
			"body":           "eyJhcGlWZXJzaW9uIjoiMC4wLjEifQ==", // placeholder base64
			"integratedTime": integratedTime,
			"logIndex":       logIndex,
			"logID":          "c0d23d6ad406973f",
		},
	}
}

func (m *rekorMockServer) start() *httptest.Server {
	mux := http.NewServeMux()

	// POST /api/v1/index/retrieve
	mux.HandleFunc("/api/v1/index/retrieve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		uuids := m.uuids[req["hash"]]
		if uuids == nil {
			uuids = []string{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(uuids)
	})

	// GET /api/v1/log/entries/{uuid}
	mux.HandleFunc("/api/v1/log/entries/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		uuid := r.URL.Path[len("/api/v1/log/entries/"):]
		entry, ok := m.entries[uuid]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entry)
	})

	return httptest.NewServer(mux)
}

// ---------------------------------------------------------------------------
// RekorClient tests
// ---------------------------------------------------------------------------

func TestRekorClient_LookupByHash_Found(t *testing.T) {
	mock := newRekorMock()
	mock.addEntry("abc123def456", "uuid-aaa", 987654, 1720000000)
	srv := mock.start()
	defer srv.Close()

	client := NewRekorClient(srv.URL)
	uuids, err := client.LookupByHash(context.Background(), "sha256:abc123def456")
	if err != nil {
		t.Fatalf("LookupByHash: %v", err)
	}
	if len(uuids) != 1 || uuids[0] != "uuid-aaa" {
		t.Errorf("LookupByHash: got %v, want [uuid-aaa]", uuids)
	}
}

func TestRekorClient_LookupByHash_NotFound(t *testing.T) {
	mock := newRekorMock()
	srv := mock.start()
	defer srv.Close()

	client := NewRekorClient(srv.URL)
	uuids, err := client.LookupByHash(context.Background(), "sha256:notpresent")
	if err != nil {
		t.Fatalf("LookupByHash missing: %v", err)
	}
	if len(uuids) != 0 {
		t.Errorf("LookupByHash missing: got %v, want empty", uuids)
	}
}

func TestRekorClient_GetEntry(t *testing.T) {
	const (
		testUUID      = "uuid-bbb"
		testHash      = "deadbeef"
		testLogIndex  = int64(42000)
		testTimestamp = int64(1720100000)
	)

	mock := newRekorMock()
	mock.addEntry(testHash, testUUID, testLogIndex, testTimestamp)
	srv := mock.start()
	defer srv.Close()

	client := NewRekorClient(srv.URL)
	entry, err := client.GetEntry(context.Background(), testUUID)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if entry.UUID != testUUID {
		t.Errorf("UUID: got %q, want %q", entry.UUID, testUUID)
	}
	if entry.LogIndex != testLogIndex {
		t.Errorf("LogIndex: got %d, want %d", entry.LogIndex, testLogIndex)
	}
	wantTime := time.Unix(testTimestamp, 0).UTC()
	if !entry.IntegratedTime.Equal(wantTime) {
		t.Errorf("IntegratedTime: got %v, want %v", entry.IntegratedTime, wantTime)
	}
	if entry.EntryURL == "" {
		t.Error("EntryURL: expected non-empty")
	}
}

func TestRekorClient_GetEntry_NotFound(t *testing.T) {
	mock := newRekorMock()
	srv := mock.start()
	defer srv.Close()

	client := NewRekorClient(srv.URL)
	_, err := client.GetEntry(context.Background(), "nonexistent-uuid")
	if err == nil {
		t.Error("GetEntry missing: expected error")
	}
}

func TestRekorClient_LookupByHash_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewRekorClient(srv.URL)
	_, err := client.LookupByHash(context.Background(), "sha256:anything")
	if err == nil {
		t.Error("LookupByHash HTTP 500: expected error")
	}
}

func TestMonitorCASArtifact_Found(t *testing.T) {
	const hexHash = "cafebabe"
	mock := newRekorMock()
	mock.addEntry(hexHash, "uuid-ccc", 11111, 1720200000)
	srv := mock.start()
	defer srv.Close()

	client := NewRekorClient(srv.URL)
	entry, err := MonitorCASArtifact(context.Background(), client, hexHash)
	if err != nil {
		t.Fatalf("MonitorCASArtifact: %v", err)
	}
	if entry == nil {
		t.Fatal("MonitorCASArtifact: got nil entry, want non-nil")
	}
	if entry.LogIndex != 11111 {
		t.Errorf("LogIndex: got %d, want 11111", entry.LogIndex)
	}
}

func TestMonitorCASArtifact_NotFound(t *testing.T) {
	mock := newRekorMock()
	srv := mock.start()
	defer srv.Close()

	client := NewRekorClient(srv.URL)
	entry, err := MonitorCASArtifact(context.Background(), client, "nosuchblob")
	if err != nil {
		t.Fatalf("MonitorCASArtifact: %v", err)
	}
	if entry != nil {
		t.Errorf("MonitorCASArtifact: got %+v, want nil", entry)
	}
}

func TestNewRekorClient_DefaultURL(t *testing.T) {
	c := NewRekorClient("")
	if c.baseURL != defaultRekorBaseURL {
		t.Errorf("baseURL: got %q, want %q", c.baseURL, defaultRekorBaseURL)
	}
}
