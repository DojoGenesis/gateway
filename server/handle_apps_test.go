package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/apps"
	"github.com/DojoGenesis/gateway/tools"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// testToolRegistry implements gateway.ToolRegistry for handler tests.
type testToolRegistry struct {
	tools map[string]*tools.ToolDefinition
}

func newTestToolRegistry() *testToolRegistry {
	return &testToolRegistry{tools: make(map[string]*tools.ToolDefinition)}
}

func (r *testToolRegistry) Register(_ context.Context, def *tools.ToolDefinition) error {
	r.tools[def.Name] = def
	return nil
}

func (r *testToolRegistry) Get(_ context.Context, name string) (*tools.ToolDefinition, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, &toolNotFoundError{name}
	}
	return t, nil
}

func (r *testToolRegistry) List(_ context.Context) ([]*tools.ToolDefinition, error) {
	var result []*tools.ToolDefinition
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result, nil
}

func (r *testToolRegistry) ListByNamespace(_ context.Context, _ string) ([]*tools.ToolDefinition, error) {
	return nil, nil
}

type toolNotFoundError struct {
	name string
}

func (e *toolNotFoundError) Error() string {
	return "tool not found: " + e.name
}

func newAppsTestServer() *Server {
	reg := newTestToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name:        "echo",
		Description: "Echoes input",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"echo": args["message"]}, nil
		},
	})

	mgr := apps.NewAppManager(apps.AppManagerConfig{
		AllowedOrigins:     []string{"http://localhost:3000"},
		DefaultToolTimeout: 5 * time.Second,
	}, reg)

	return &Server{
		appManager: mgr,
	}
}

func TestHandleGetResource_Success(t *testing.T) {
	s := newAppsTestServer()
	s.appManager.RegisterResource(&apps.ResourceMeta{
		URI:      "ui://test/app.html",
		MimeType: "text/html",
		Content:  []byte("<html>test</html>"),
		CacheKey: "test-key-1",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/gateway/resources?uri=ui://test/app.html", nil)

	s.handleGetResource(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "<html>test</html>" {
		t.Errorf("body = %q, want %q", w.Body.String(), "<html>test</html>")
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Error("missing Content-Security-Policy header")
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", w.Header().Get("X-Content-Type-Options"))
	}
	if w.Header().Get("ETag") == "" {
		t.Error("missing ETag header")
	}
}

func TestHandleGetResource_NotFound(t *testing.T) {
	s := newAppsTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/gateway/resources?uri=ui://nonexistent/app.html", nil)

	s.handleGetResource(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGetResource_InvalidURI(t *testing.T) {
	s := newAppsTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/gateway/resources?uri=https://example.com", nil)

	s.handleGetResource(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleGetResource_MissingURI(t *testing.T) {
	s := newAppsTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/gateway/resources", nil)

	s.handleGetResource(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleLaunchApp_Success(t *testing.T) {
	s := newAppsTestServer()
	s.appManager.RegisterResource(&apps.ResourceMeta{
		URI:     "ui://test/app.html",
		Content: []byte("app"),
	})

	body, _ := json.Marshal(map[string]string{
		"resource_uri": "ui://test/app.html",
		"session_id":   "session-1",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/gateway/apps/launch", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	s.handleLaunchApp(c)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["instance_id"] == nil || resp["instance_id"] == "" {
		t.Error("missing instance_id in response")
	}
}

func TestHandleCloseApp_Success(t *testing.T) {
	s := newAppsTestServer()
	s.appManager.RegisterResource(&apps.ResourceMeta{
		URI:     "ui://test/app.html",
		Content: []byte("app"),
	})

	inst, _ := s.appManager.LaunchApp("ui://test/app.html", "session-1")

	body, _ := json.Marshal(map[string]string{
		"instance_id": inst.ID,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/gateway/apps/close", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	s.handleCloseApp(c)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestHandleProxyToolCall_Authorized(t *testing.T) {
	s := newAppsTestServer()
	s.appManager.RegisterResource(&apps.ResourceMeta{
		URI:     "ui://test/app.html",
		Content: []byte("app"),
	})

	inst, _ := s.appManager.LaunchApp("ui://test/app.html", "session-1")

	body, _ := json.Marshal(map[string]interface{}{
		"app_id":    inst.ID,
		"tool_name": "echo",
		"arguments": map[string]interface{}{"message": "hello"},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/gateway/apps/tool-call", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	s.handleProxyToolCall(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp apps.ToolCallResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.IsError {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestHandleProxyToolCall_Unauthorized(t *testing.T) {
	s := newAppsTestServer()

	body, _ := json.Marshal(map[string]interface{}{
		"app_id":    "fake-app-id",
		"tool_name": "echo",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/gateway/apps/tool-call", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	s.handleProxyToolCall(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp apps.ToolCallResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.IsError {
		t.Fatal("expected error response for unauthorized call")
	}
}

func TestHandleListApps(t *testing.T) {
	s := newAppsTestServer()
	s.appManager.RegisterResource(&apps.ResourceMeta{
		URI:     "ui://test/app1.html",
		Content: []byte("a"),
	})
	s.appManager.RegisterResource(&apps.ResourceMeta{
		URI:     "ui://test/app2.html",
		Content: []byte("b"),
	})

	s.appManager.LaunchApp("ui://test/app1.html", "session-1")
	s.appManager.LaunchApp("ui://test/app2.html", "session-1")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/gateway/apps?session_id=session-1", nil)

	s.handleListApps(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Instances []interface{} `json:"instances"`
		Count     int           `json:"count"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
}

func TestHandleAppStatus(t *testing.T) {
	s := newAppsTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/gateway/apps/status", nil)

	s.handleAppStatus(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp apps.AppManagerStatus
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Healthy {
		t.Error("expected healthy status")
	}
}

func TestHandleGetResource_Disabled(t *testing.T) {
	s := &Server{appManager: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/gateway/resources?uri=ui://test/app.html", nil)

	s.handleGetResource(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleGetResource_304NotModified(t *testing.T) {
	s := newAppsTestServer()
	s.appManager.RegisterResource(&apps.ResourceMeta{
		URI:      "ui://test/cached.html",
		MimeType: "text/html",
		Content:  []byte("<html>cached</html>"),
		CacheKey: "cached-key",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/gateway/resources?uri=ui://test/cached.html", nil)
	c.Request.Header.Set("If-None-Match", `"cached-key"`)

	s.handleGetResource(c)

	if w.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotModified)
	}
}
