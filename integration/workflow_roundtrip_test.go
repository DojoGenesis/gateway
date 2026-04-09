package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
	"github.com/DojoGenesis/gateway/runtime/cas"
	"github.com/DojoGenesis/gateway/workflow"
)

// ---------------------------------------------------------------------------
// Test 3: Workflow round-trip integrity
//
// Build a minimal 2-step workflow definition (JSON).
// POST to /api/workflows -> get cas_hash back.
// Execute the workflow using the same CAS hash.
// Verify workflow loads from CAS by hash and executes.
// Verify reply delivered (via mock adapter.Send).
// ---------------------------------------------------------------------------

// workflowCRUDHandler provides a minimal HTTP server for workflow CRUD
// operations backed by CAS. This is a test-only implementation that mirrors
// the production handler in workflow/api/handler.go but without importing
// the heavy server module.
type workflowCRUDHandler struct {
	store cas.Store
}

func (h *workflowCRUDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/api/workflows" && r.Method == http.MethodPost:
		h.createWorkflow(w, r)
	case path == "/api/workflows" && r.Method == http.MethodGet:
		h.listWorkflows(w, r)
	case strings.HasPrefix(path, "/api/workflows/") && r.Method == http.MethodGet:
		name := strings.TrimPrefix(path, "/api/workflows/")
		name = strings.TrimSuffix(name, "/")
		h.getWorkflow(w, r, name)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *workflowCRUDHandler) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var def workflow.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if err := workflow.Validate(&def); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	data, err := workflow.Marshal(&def)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	ref, err := h.store.Put(ctx, data, cas.ContentMeta{
		Type:      cas.ContentConfig,
		CreatedAt: time.Now().UTC(),
		Labels: map[string]string{
			"kind":    "workflow",
			"name":    def.Name,
			"version": def.Version,
		},
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "store error: "+err.Error())
		return
	}

	tagName := "workflow/" + def.Name
	version := def.Version
	if version == "" {
		version = "latest"
	}
	if err := h.store.Tag(ctx, tagName, version, ref); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "tag error: "+err.Error())
		return
	}
	if version != "latest" {
		if err := h.store.Tag(ctx, tagName, "latest", ref); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "tag latest error: "+err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"ref":     "sha256:" + string(ref),
		"name":    def.Name,
		"version": version,
	})
}

func (h *workflowCRUDHandler) listWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	entries, err := h.store.List(ctx, "workflow/")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "list error: "+err.Error())
		return
	}

	type entry struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Ref     string `json:"ref"`
	}
	var items []entry
	for _, e := range entries {
		if e.Version == "canvas" {
			continue
		}
		items = append(items, entry{
			Name:    strings.TrimPrefix(e.Name, "workflow/"),
			Version: e.Version,
			Ref:     "sha256:" + string(e.Ref),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"workflows": items})
}

func (h *workflowCRUDHandler) getWorkflow(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	version := r.URL.Query().Get("version")
	if version == "" {
		version = "latest"
	}

	ref, err := h.store.Resolve(ctx, "workflow/"+name, version)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "workflow not found: "+name)
		return
	}

	data, _, err := h.store.Get(ctx, ref)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "get error: "+err.Error())
		return
	}

	def, err := workflow.Unmarshal(data)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unmarshal error: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(def)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestWorkflow_RoundTrip_BuildSaveExecuteReply(t *testing.T) {
	// --- 1. Create CAS store ---
	store := createTestCAS(t)

	// --- 2. Build a minimal 2-step workflow JSON ---
	wfJSON, err := buildMinimalWorkflowJSON("roundtrip-test")
	if err != nil {
		t.Fatalf("build workflow JSON: %v", err)
	}

	// --- 3. Start HTTP server with workflow CRUD handler ---
	handler := &workflowCRUDHandler{store: store}
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// --- 4. POST to /api/workflows -> get cas_hash back ---
	resp, err := http.Post(srv.URL+"/api/workflows", "application/json",
		bytes.NewReader(wfJSON))
	if err != nil {
		t.Fatalf("POST /api/workflows: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create workflow status = %d, want 201; body = %s", resp.StatusCode, string(body))
	}

	var createResp struct {
		Ref     string `json:"ref"`
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	if !strings.HasPrefix(createResp.Ref, "sha256:") {
		t.Fatalf("ref = %q, want sha256:... prefix", createResp.Ref)
	}
	if createResp.Name != "roundtrip-test" {
		t.Errorf("name = %q, want %q", createResp.Name, "roundtrip-test")
	}

	casHash := createResp.Ref
	t.Logf("workflow stored with ref=%s", casHash)

	// --- 5. GET the workflow back and verify it matches ---
	getResp, err := http.Get(srv.URL + "/api/workflows/roundtrip-test")
	if err != nil {
		t.Fatalf("GET /api/workflows/roundtrip-test: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getResp.Body)
		t.Fatalf("get workflow status = %d; body = %s", getResp.StatusCode, string(body))
	}

	var fetchedDef workflow.WorkflowDefinition
	if err := json.NewDecoder(getResp.Body).Decode(&fetchedDef); err != nil {
		t.Fatalf("decode fetched workflow: %v", err)
	}
	if fetchedDef.Name != "roundtrip-test" {
		t.Errorf("fetched name = %q, want %q", fetchedDef.Name, "roundtrip-test")
	}
	if len(fetchedDef.Steps) != 2 {
		t.Fatalf("fetched steps = %d, want 2", len(fetchedDef.Steps))
	}

	// --- 6. Execute the workflow via WorkflowExecutor ---
	executor := workflow.NewWorkflowExecutor(store, func(wfID, stepID, status string) {
		t.Logf("execution event: wf=%s step=%s status=%s", wfID, stepID, status)
	})

	result, err := executor.Execute(context.Background(), "roundtrip-test")
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("execution status = %q, want %q", result.Status, "completed")
	}
	if len(result.StepResults) != 2 {
		t.Errorf("step results = %d, want 2", len(result.StepResults))
	}

	// --- 7. Verify reply delivery via mock adapter ---
	pub := &memPublisher{}
	bus := channel.NewNATSBus(pub)
	adapter := &channel.StubAdapter{}

	runner := &stubWorkflowRunner{
		result: &channel.WorkflowRunResult{
			Status:    result.Status,
			StepCount: len(result.StepResults),
		},
	}
	bridge := channel.NewChannelBridge(runner)
	bridge.Register("stub", adapter)
	bridge.AddTrigger(channel.TriggerSpec{Platform: "stub", Workflow: "roundtrip-test"})
	bus.Subscribe(bridge.BusHandler(context.Background()))

	// Simulate an inbound trigger message.
	msg := &channel.ChannelMessage{
		ID:        "roundtrip-trigger",
		Platform:  "stub",
		ChannelID: "C_RT",
		UserID:    "U_RT",
		Text:      "trigger roundtrip",
		Timestamp: time.Now().UTC(),
	}
	evt, _ := channel.ToCloudEvent(msg)
	bus.Publish("dojo.channel.message.stub", evt)

	sent := adapter.Sent()
	if len(sent) != 1 {
		t.Fatalf("adapter sent %d messages, want 1", len(sent))
	}
	if !strings.Contains(sent[0].Text, "completed") {
		t.Errorf("reply text = %q, should contain 'completed'", sent[0].Text)
	}
}

func TestWorkflow_RoundTrip_CASHashIntegrity(t *testing.T) {
	store := createTestCAS(t)

	// Store the same workflow twice — CAS should return the same hash.
	wfJSON1, _ := buildMinimalWorkflowJSON("hash-integrity-test")
	wfJSON2, _ := buildMinimalWorkflowJSON("hash-integrity-test")

	ctx := context.Background()
	ref1, err := store.Put(ctx, wfJSON1, cas.ContentMeta{
		Type:      cas.ContentConfig,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("put 1: %v", err)
	}

	ref2, err := store.Put(ctx, wfJSON2, cas.ContentMeta{
		Type:      cas.ContentConfig,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("put 2: %v", err)
	}

	if ref1 != ref2 {
		t.Errorf("CAS refs differ for identical content: %s != %s", ref1, ref2)
	}

	// Fetch and verify content matches.
	data, _, err := store.Get(ctx, ref1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	var def workflow.WorkflowDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if def.Name != "hash-integrity-test" {
		t.Errorf("name = %q, want %q", def.Name, "hash-integrity-test")
	}
}

func TestWorkflow_RoundTrip_ValidationReject(t *testing.T) {
	store := createTestCAS(t)
	handler := &workflowCRUDHandler{store: store}
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Workflow with empty name should be rejected.
	badDef := `{"name": "", "version": "1.0.0", "steps": [{"id": "s1", "skill": "test"}]}`
	resp, err := http.Post(srv.URL+"/api/workflows", "application/json",
		bytes.NewReader([]byte(badDef)))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for invalid workflow", resp.StatusCode)
	}

	// Workflow with cycle should be rejected.
	cyclicDef := `{
		"name": "cyclic",
		"version": "1.0.0",
		"steps": [
			{"id": "a", "skill": "s1", "depends_on": ["b"]},
			{"id": "b", "skill": "s2", "depends_on": ["a"]}
		]
	}`
	resp, err = http.Post(srv.URL+"/api/workflows", "application/json",
		bytes.NewReader([]byte(cyclicDef)))
	if err != nil {
		t.Fatalf("POST cyclic: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("cyclic workflow status = %d, want 400", resp.StatusCode)
	}
}
