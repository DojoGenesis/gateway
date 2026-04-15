package mcpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mcpserver "github.com/DojoGenesis/gateway/protocol/mcpserver"
)

// extractFirstItem safely extracts the first map element from a slice that
// may be typed as []interface{} or []map[string]interface{}.
func extractFirstItem(t *testing.T, label string, raw interface{}) map[string]interface{} {
	t.Helper()
	switch v := raw.(type) {
	case []map[string]interface{}:
		if len(v) == 0 {
			t.Fatalf("%s: array is empty", label)
		}
		return v[0]
	case []interface{}:
		if len(v) == 0 {
			t.Fatalf("%s: array is empty", label)
		}
		item, ok := v[0].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: first element is not a map: %T", label, v[0])
		}
		return item
	default:
		t.Fatalf("%s: not an array: %T", label, raw)
		return nil
	}
}

// extractSliceOfMaps safely converts a slice that may be typed as
// []interface{} or []map[string]interface{} into []map[string]interface{}.
func extractSliceOfMaps(t *testing.T, label string, raw interface{}) []map[string]interface{} {
	t.Helper()
	switch v := raw.(type) {
	case []map[string]interface{}:
		return v
	case []interface{}:
		result := make([]map[string]interface{}, 0, len(v))
		for i, item := range v {
			m, ok := item.(map[string]interface{})
			if !ok {
				t.Fatalf("%s[%d]: not a map: %T", label, i, item)
			}
			result = append(result, m)
		}
		return result
	default:
		t.Fatalf("%s: not an array: %T", label, raw)
		return nil
	}
}

// mockDIPAPI creates an httptest.Server that mimics DIP REST API endpoints.
func mockDIPAPI(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// POST /api/v1/query
	mux.HandleFunc("POST /api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-ID") != "mcp-server" {
			t.Errorf("query: X-User-ID header missing or wrong: %q", r.Header.Get("X-User-ID"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("query: Content-Type header: %q", r.Header.Get("Content-Type"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "empty body", http.StatusBadRequest)
			return
		}
		var queryReq map[string]interface{}
		if err := json.Unmarshal(body, &queryReq); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if _, ok := queryReq["query"]; !ok {
			http.Error(w, "missing query field", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"nodes": []map[string]interface{}{
				{"id": "aaa-bbb", "name": "Button", "type": "component"},
			},
		})
	})

	// GET /api/v1/nodes/{id}
	mux.HandleFunc("GET /api/v1/nodes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   id,
			"name": "Button",
			"type": "component",
		})
	})

	// POST /api/v1/measurements
	mux.HandleFunc("POST /api/v1/measurements", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "empty body", http.StatusBadRequest)
			return
		}
		var measReq map[string]interface{}
		if err := json.Unmarshal(body, &measReq); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if _, ok := measReq["component_id"]; !ok {
			http.Error(w, "missing component_id field", http.StatusBadRequest)
			return
		}
		if _, ok := measReq["score"]; !ok {
			http.Error(w, "missing score field", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "meas-001",
			"score": 0.85,
		})
	})

	// GET /api/v1/lenses
	mux.HandleFunc("GET /api/v1/lenses", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"lenses": []map[string]interface{}{
				{"id": "lens-001", "name": "visual-harmony", "license_type": "guided"},
			},
		})
	})

	// GET /api/v1/components/{id}/profile
	mux.HandleFunc("GET /api/v1/components/{id}/profile", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"component_id": id,
			"measurements": []map[string]interface{}{
				{"score": 0.9, "lens": "visual-harmony"},
			},
		})
	})

	// GET /api/v1/components/{id}/deviations
	mux.HandleFunc("GET /api/v1/components/{id}/deviations", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"component_id": id,
			"patterns":     []map[string]interface{}{},
		})
	})

	// GET /api/v1/nodes (for resources)
	mux.HandleFunc("GET /api/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"nodes": []map[string]interface{}{
				{"id": "comp-001", "name": "Button", "type": "component"},
			},
		})
	})

	return httptest.NewServer(mux)
}

func newDIPTestServer(t *testing.T, dipURL string) mcpserver.Server {
	t.Helper()
	srv, err := mcpserver.NewServer(mcpserver.DefaultConfig())
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := mcpserver.RegisterDIPTools(srv, dipURL); err != nil {
		t.Fatalf("RegisterDIPTools: %v", err)
	}
	if err := mcpserver.RegisterDIPResources(srv, dipURL); err != nil {
		t.Fatalf("RegisterDIPResources: %v", err)
	}
	return srv
}

func callTool(t *testing.T, srv mcpserver.Server, toolName string, args map[string]interface{}) map[string]interface{} {
	t.Helper()
	params, _ := json.Marshal(map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	})
	result, err := srv.HandleMessage(context.Background(), "tools/call", params)
	if err != nil {
		t.Fatalf("HandleMessage tools/call %s: %v", toolName, err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("tool %s: result is not a map: %T", toolName, result)
	}
	if isErr, ok := m["isError"]; ok {
		isErrBool, ok := isErr.(bool)
		if !ok {
			t.Fatalf("tool %s: isError is not a bool: %T", toolName, isErr)
		}
		if isErrBool {
			contentRaw, ok := m["content"]
			if !ok {
				t.Fatalf("tool %s returned error but missing 'content' key", toolName)
			}
			first := extractFirstItem(t, toolName+" error content", contentRaw)
			t.Fatalf("tool %s returned error: %s", toolName, first["text"])
		}
	}
	// Parse the JSON text from the content.
	contentRaw, ok := m["content"]
	if !ok {
		t.Fatalf("tool %s: missing 'content' key in response", toolName)
	}
	first := extractFirstItem(t, toolName+" content", contentRaw)
	text, ok := first["text"].(string)
	if !ok {
		t.Fatalf("tool %s: content[0][\"text\"] is not a string: %T", toolName, first["text"])
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("tool %s result is not valid JSON: %v (text: %q)", toolName, err, text)
	}
	return parsed
}

func readResource(t *testing.T, srv mcpserver.Server, uri string) map[string]interface{} {
	t.Helper()
	params, _ := json.Marshal(map[string]interface{}{"uri": uri})
	result, err := srv.HandleMessage(context.Background(), "resources/read", params)
	if err != nil {
		t.Fatalf("HandleMessage resources/read %s: %v", uri, err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("resource %s: result is not a map: %T", uri, result)
	}
	contentsRaw, ok := m["contents"]
	if !ok {
		t.Fatalf("resource %s: missing 'contents' key in response", uri)
	}
	first := extractFirstItem(t, "resource "+uri+" contents", contentsRaw)
	text, ok := first["text"].(string)
	if !ok {
		t.Fatalf("resource %s: contents[0][\"text\"] is not a string: %T", uri, first["text"])
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("resource %s result is not valid JSON: %v", uri, err)
	}
	return parsed
}

// --- Tool tests ---

func TestDIPToolsRegistration(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()

	srv := newDIPTestServer(t, mock.URL)

	// Verify all 6 tools are listed.
	result, err := srv.HandleMessage(context.Background(), "tools/list", nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("tools/list result is not a map: %T", result)
	}
	toolsRaw, ok := m["tools"]
	if !ok {
		t.Fatalf("missing 'tools' key in tools/list response")
	}
	toolItems := extractSliceOfMaps(t, "tools", toolsRaw)

	expected := []string{
		"dojo.dip.query",
		"dojo.dip.component",
		"dojo.dip.score",
		"dojo.dip.lenses",
		"dojo.dip.profile",
		"dojo.dip.deviations",
	}
	// Build a set of registered tool names.
	registered := make(map[string]bool)
	for _, tool := range toolItems {
		name, ok := tool["name"].(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(name, "dojo.dip.") {
			registered[name] = true
		}
	}
	for _, name := range expected {
		if !registered[name] {
			t.Errorf("tool %q not registered", name)
		}
	}
}

func TestDIPToolQuery(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result := callTool(t, srv, "dojo.dip.query", map[string]interface{}{
		"query": "buttons with low scores",
	})
	nodes, ok := result["nodes"].([]interface{})
	if !ok {
		t.Fatalf("expected nodes array, got %T", result["nodes"])
	}
	if len(nodes) == 0 {
		t.Error("expected at least one node")
	}
}

func TestDIPToolComponent(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result := callTool(t, srv, "dojo.dip.component", map[string]interface{}{
		"id": "test-uuid-123",
	})
	if result["id"] != "test-uuid-123" {
		t.Errorf("id: got %v, want test-uuid-123", result["id"])
	}
	if result["name"] != "Button" {
		t.Errorf("name: got %v, want Button", result["name"])
	}
}

func TestDIPToolComponentMissingID(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	// Call without id -- should return error.
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "dojo.dip.component",
		"arguments": map[string]interface{}{},
	})
	result, err := srv.HandleMessage(context.Background(), "tools/call", params)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %T", result)
	}
	if m["isError"] != true {
		t.Error("expected isError=true for missing id")
	}
}

func TestDIPToolScore(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result := callTool(t, srv, "dojo.dip.score", map[string]interface{}{
		"component_id":  "comp-001",
		"philosophy_id": "phil-001",
		"lens_id":       "lens-001",
		"score":         0.85,
		"method":        "llm",
	})
	if result["id"] != "meas-001" {
		t.Errorf("id: got %v, want meas-001", result["id"])
	}
}

func TestDIPToolLenses(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result := callTool(t, srv, "dojo.dip.lenses", map[string]interface{}{})
	lenses, ok := result["lenses"].([]interface{})
	if !ok {
		t.Fatalf("expected lenses array, got %T", result["lenses"])
	}
	if len(lenses) == 0 {
		t.Error("expected at least one lens")
	}
}

func TestDIPToolProfile(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result := callTool(t, srv, "dojo.dip.profile", map[string]interface{}{
		"component_id": "comp-001",
	})
	if result["component_id"] != "comp-001" {
		t.Errorf("component_id: got %v, want comp-001", result["component_id"])
	}
}

func TestDIPToolDeviations(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result := callTool(t, srv, "dojo.dip.deviations", map[string]interface{}{
		"component_id": "comp-001",
	})
	if result["component_id"] != "comp-001" {
		t.Errorf("component_id: got %v, want comp-001", result["component_id"])
	}
}

// --- Resource tests ---

func TestDIPResourcesRegistration(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result, err := srv.HandleMessage(context.Background(), "resources/list", nil)
	if err != nil {
		t.Fatalf("resources/list: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("resources/list result is not a map: %T", result)
	}
	resourcesRaw, ok := m["resources"]
	if !ok {
		t.Fatalf("missing 'resources' key in resources/list response")
	}
	resourceItems := extractSliceOfMaps(t, "resources", resourcesRaw)

	expected := map[string]bool{
		"dojo://dip/lenses":     false,
		"dojo://dip/components": false,
	}
	for _, r := range resourceItems {
		uri, ok := r["uri"].(string)
		if !ok {
			continue
		}
		if _, exists := expected[uri]; exists {
			expected[uri] = true
		}
	}
	for uri, found := range expected {
		if !found {
			t.Errorf("resource %q not registered", uri)
		}
	}
}

func TestDIPResourceLenses(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result := readResource(t, srv, "dojo://dip/lenses")
	lenses, ok := result["lenses"].([]interface{})
	if !ok {
		t.Fatalf("expected lenses array, got %T", result["lenses"])
	}
	if len(lenses) == 0 {
		t.Error("expected at least one lens")
	}
}

func TestDIPResourceComponents(t *testing.T) {
	mock := mockDIPAPI(t)
	defer mock.Close()
	srv := newDIPTestServer(t, mock.URL)

	result := readResource(t, srv, "dojo://dip/components")
	nodes, ok := result["nodes"].([]interface{})
	if !ok {
		t.Fatalf("expected nodes array, got %T", result["nodes"])
	}
	if len(nodes) == 0 {
		t.Error("expected at least one node")
	}
}

// --- Error handling ---

func TestDIPToolWithServerError(t *testing.T) {
	// Create a mock that returns 500 for everything.
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer errServer.Close()

	srv := newDIPTestServer(t, errServer.URL)

	// Call any tool -- should return an MCP isError response, not a Go error.
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "dojo.dip.lenses",
		"arguments": map[string]interface{}{},
	})
	result, err := srv.HandleMessage(context.Background(), "tools/call", params)
	if err != nil {
		t.Fatalf("HandleMessage should not return Go error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %T", result)
	}
	if m["isError"] != true {
		t.Error("expected isError=true when DIP API returns 500")
	}
}
