package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/runtime/cas"
	"github.com/DojoGenesis/gateway/workflow"
	"github.com/DojoGenesis/gateway/workflow/api"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestServer returns a new httptest.Server backed by a temp SQLite CAS store.
func newTestServer(t *testing.T) (*httptest.Server, cas.Store) {
	t.Helper()
	store, err := cas.NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	mux := http.NewServeMux()
	h := api.NewWorkflowHandler(store)
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, store
}

// validDef returns a minimal valid WorkflowDefinition.
func validDef(name string) *workflow.WorkflowDefinition {
	return &workflow.WorkflowDefinition{
		Version:      "1.0.0",
		Name:         name,
		ArtifactType: workflow.WorkflowArtifactType,
		Steps: []workflow.Step{
			{ID: "step-a", Skill: "scout", Inputs: map[string]string{"topic": "testing"}},
			{ID: "step-b", Skill: "synthesize", Inputs: map[string]string{"src": "{{ steps.step-a.outputs.result }}"}, DependsOn: []string{"step-a"}},
		},
	}
}

// cycleDef returns a WorkflowDefinition with a cycle (a -> b -> a).
func cycleDef() *workflow.WorkflowDefinition {
	return &workflow.WorkflowDefinition{
		Version:      "1.0.0",
		Name:         "cycle-workflow",
		ArtifactType: workflow.WorkflowArtifactType,
		Steps: []workflow.Step{
			{ID: "a", Skill: "scout", Inputs: map[string]string{}, DependsOn: []string{"b"}},
			{ID: "b", Skill: "synthesize", Inputs: map[string]string{}, DependsOn: []string{"a"}},
		},
	}
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data)) //nolint:noctx
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func putJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("new PUT request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", url, err)
	}
	return resp
}

func getURL(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 1. TestCreateWorkflow_Valid
// ---------------------------------------------------------------------------

func TestCreateWorkflow_Valid(t *testing.T) {
	srv, _ := newTestServer(t)

	def := validDef("my-pipeline")
	resp := postJSON(t, srv.URL+"/api/workflows", def)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]string
	decodeJSON(t, resp, &result)

	if result["ref"] == "" {
		t.Error("expected non-empty ref")
	}
	if !strings.HasPrefix(result["ref"], "sha256:") {
		t.Errorf("ref should start with sha256:, got %q", result["ref"])
	}
	if result["name"] != "my-pipeline" {
		t.Errorf("name: got %q, want %q", result["name"], "my-pipeline")
	}
	if result["version"] != "1.0.0" {
		t.Errorf("version: got %q, want %q", result["version"], "1.0.0")
	}
}

// ---------------------------------------------------------------------------
// 2. TestCreateWorkflow_InvalidCycle
// ---------------------------------------------------------------------------

func TestCreateWorkflow_InvalidCycle(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := postJSON(t, srv.URL+"/api/workflows", cycleDef())

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var result map[string]string
	decodeJSON(t, resp, &result)

	if result["error"] == "" {
		t.Error("expected non-empty error message")
	}
	if !strings.Contains(result["error"], "cycle") {
		t.Errorf("expected cycle error, got %q", result["error"])
	}
}

// ---------------------------------------------------------------------------
// 3. TestCreateWorkflow_EmptyName
// ---------------------------------------------------------------------------

func TestCreateWorkflow_EmptyName(t *testing.T) {
	srv, _ := newTestServer(t)

	def := validDef("")
	def.Name = ""
	resp := postJSON(t, srv.URL+"/api/workflows", def)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var result map[string]string
	decodeJSON(t, resp, &result)

	if result["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

// ---------------------------------------------------------------------------
// 4. TestListWorkflows
// ---------------------------------------------------------------------------

func TestListWorkflows(t *testing.T) {
	srv, _ := newTestServer(t)

	// Create two workflows.
	resp1 := postJSON(t, srv.URL+"/api/workflows", validDef("pipeline-alpha"))
	if resp1.StatusCode != http.StatusCreated {
		resp1.Body.Close()
		t.Fatalf("create pipeline-alpha: expected 201, got %d", resp1.StatusCode)
	}
	resp1.Body.Close()

	resp2 := postJSON(t, srv.URL+"/api/workflows", validDef("pipeline-beta"))
	if resp2.StatusCode != http.StatusCreated {
		resp2.Body.Close()
		t.Fatalf("create pipeline-beta: expected 201, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()

	// List.
	resp := getURL(t, srv.URL+"/api/workflows")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Workflows []map[string]string `json:"workflows"`
	}
	decodeJSON(t, resp, &result)

	if len(result.Workflows) != 2 {
		t.Fatalf("expected 2 workflows, got %d", len(result.Workflows))
	}

	names := make(map[string]bool)
	for _, w := range result.Workflows {
		names[w["name"]] = true
	}
	if !names["pipeline-alpha"] {
		t.Error("expected pipeline-alpha in list")
	}
	if !names["pipeline-beta"] {
		t.Error("expected pipeline-beta in list")
	}
}

// ---------------------------------------------------------------------------
// 5. TestGetWorkflow
// ---------------------------------------------------------------------------

func TestGetWorkflow(t *testing.T) {
	srv, _ := newTestServer(t)

	def := validDef("fetch-me")
	resp := postJSON(t, srv.URL+"/api/workflows", def)
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// GET by name.
	getResp := getURL(t, srv.URL+"/api/workflows/fetch-me")
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getResp.StatusCode)
	}

	var got workflow.WorkflowDefinition
	decodeJSON(t, getResp, &got)

	if got.Name != "fetch-me" {
		t.Errorf("name: got %q, want %q", got.Name, "fetch-me")
	}
	if len(got.Steps) != 2 {
		t.Errorf("steps: got %d, want 2", len(got.Steps))
	}
}

// ---------------------------------------------------------------------------
// 6. TestGetWorkflow_NotFound
// ---------------------------------------------------------------------------

func TestGetWorkflow_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := getURL(t, srv.URL+"/api/workflows/does-not-exist")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 7. TestSaveCanvas
// ---------------------------------------------------------------------------

func TestSaveCanvas(t *testing.T) {
	srv, _ := newTestServer(t)

	// Create workflow first.
	def := validDef("canvas-workflow")
	createResp := postJSON(t, srv.URL+"/api/workflows", def)
	if createResp.StatusCode != http.StatusCreated {
		createResp.Body.Close()
		t.Fatalf("create workflow: expected 201, got %d", createResp.StatusCode)
	}
	var createResult map[string]string
	decodeJSON(t, createResp, &createResult)
	wfRef := createResult["ref"]

	// PUT canvas state.
	canvas := workflow.CanvasState{
		WorkflowRef: wfRef,
		Viewport:    workflow.Viewport{X: 10, Y: 20, Zoom: 1.5},
		NodePositions: map[string]workflow.Position{
			"step-a": {X: 100, Y: 200},
			"step-b": {X: 400, Y: 200},
		},
	}

	putResp := putJSON(t, srv.URL+"/api/workflows/canvas-workflow/canvas", canvas)
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("put canvas: expected 200, got %d", putResp.StatusCode)
	}

	var putResult map[string]string
	decodeJSON(t, putResp, &putResult)
	if putResult["ref"] == "" {
		t.Error("expected non-empty ref from PUT canvas")
	}

	// GET canvas back.
	getResp := getURL(t, srv.URL+"/api/workflows/canvas-workflow/canvas")
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get canvas: expected 200, got %d", getResp.StatusCode)
	}

	var gotCanvas workflow.CanvasState
	decodeJSON(t, getResp, &gotCanvas)

	if gotCanvas.WorkflowRef != wfRef {
		t.Errorf("workflow_ref: got %q, want %q", gotCanvas.WorkflowRef, wfRef)
	}
	if gotCanvas.Viewport.Zoom != 1.5 {
		t.Errorf("zoom: got %v, want 1.5", gotCanvas.Viewport.Zoom)
	}
	if len(gotCanvas.NodePositions) != 2 {
		t.Errorf("node_positions: got %d, want 2", len(gotCanvas.NodePositions))
	}
	if pos, ok := gotCanvas.NodePositions["step-a"]; !ok || pos.X != 100 || pos.Y != 200 {
		t.Errorf("node_positions[step-a]: got %+v, want {X:100 Y:200}", pos)
	}
}

// ---------------------------------------------------------------------------
// 8. TestValidateWorkflow_Valid
// ---------------------------------------------------------------------------

func TestValidateWorkflow_Valid(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := postJSON(t, srv.URL+"/api/workflows/my-wf/validate", validDef("my-wf"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	decodeJSON(t, resp, &result)

	if result["valid"] != true {
		t.Errorf("expected valid=true, got %v", result["valid"])
	}
	if _, hasErr := result["error"]; hasErr {
		t.Errorf("expected no error key, got error=%v", result["error"])
	}
}

// ---------------------------------------------------------------------------
// 9. TestValidateWorkflow_Invalid
// ---------------------------------------------------------------------------

func TestValidateWorkflow_Invalid(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := postJSON(t, srv.URL+"/api/workflows/cycle-wf/validate", cycleDef())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	decodeJSON(t, resp, &result)

	if result["valid"] != false {
		t.Errorf("expected valid=false, got %v", result["valid"])
	}
	errMsg, ok := result["error"].(string)
	if !ok || errMsg == "" {
		t.Errorf("expected non-empty error string, got %v", result["error"])
	}
	if !strings.Contains(errMsg, "cycle") {
		t.Errorf("expected cycle in error, got %q", errMsg)
	}
}

// ---------------------------------------------------------------------------
// 10. TestListSkills
// ---------------------------------------------------------------------------

func TestListSkills(t *testing.T) {
	srv, _ := newTestServer(t)

	// No skills installed — should return empty list.
	resp := getURL(t, srv.URL+"/api/skills")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	decodeJSON(t, resp, &result)

	skills, ok := result["skills"]
	if !ok {
		t.Fatal("expected 'skills' key in response")
	}

	// json.Decoder decodes arrays as []any.
	skillList, ok := skills.([]any)
	if !ok {
		// null/nil is also acceptable for empty.
		if skills != nil {
			t.Fatalf("expected skills to be a list or null, got %T: %v", skills, skills)
		}
	} else if len(skillList) != 0 {
		t.Errorf("expected empty skills list, got %d items", len(skillList))
	}
}
