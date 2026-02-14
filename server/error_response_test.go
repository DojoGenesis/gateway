package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errorEnvelope is the expected shape of all error responses from the gateway.
//
//	{"error": {"code": "string", "message": "string", "details": {}, "request_id": "..."}}
type errorEnvelope struct {
	Error struct {
		Code      string                 `json:"code"`
		Message   string                 `json:"message"`
		Details   map[string]interface{} `json:"details"`
		RequestID interface{}            `json:"request_id"`
	} `json:"error"`
}

// assertErrorShape validates that the response body matches the canonical error shape.
func assertErrorShape(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedCode string) errorEnvelope {
	t.Helper()
	assert.Equal(t, expectedStatus, w.Code, "unexpected HTTP status")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json", "error response must be JSON")

	var env errorEnvelope
	err := json.Unmarshal(w.Body.Bytes(), &env)
	require.NoError(t, err, "response body must be valid JSON")
	assert.Equal(t, expectedCode, env.Error.Code, "error code mismatch")
	assert.NotEmpty(t, env.Error.Message, "error message must not be empty")
	assert.NotNil(t, env.Error.Details, "details must be present (even if empty)")
	return env
}

// newTestServer creates a minimal Server suitable for testing error responses.
func newTestServer() (*Server, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add request ID middleware so error responses include it
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})
	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*gateway.AgentConfig),
	}
	return s, router
}

// TestErrorResponseShape_ChatCompletions verifies error responses from POST /v1/chat/completions.
func TestErrorResponseShape_ChatCompletions(t *testing.T) {
	s, router := newTestServer()
	router.POST("/v1/chat/completions", s.handleChatCompletions)

	tests := []struct {
		name         string
		body         string
		expectedCode string
		expectedHTTP int
	}{
		{
			name:         "invalid JSON",
			body:         `{not json}`,
			expectedCode: "invalid_request",
			expectedHTTP: http.StatusBadRequest,
		},
		{
			name:         "missing model",
			body:         `{"messages": [{"role": "user", "content": "hi"}]}`,
			expectedCode: "invalid_request",
			expectedHTTP: http.StatusBadRequest,
		},
		{
			name:         "empty messages",
			body:         `{"model": "gpt-4", "messages": []}`,
			expectedCode: "invalid_request",
			expectedHTTP: http.StatusBadRequest,
		},
		{
			name:         "no provider configured",
			body:         `{"model": "gpt-4", "messages": [{"role": "user", "content": "hi"}]}`,
			expectedCode: "server_error",
			expectedHTTP: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assertErrorShape(t, w, tt.expectedHTTP, tt.expectedCode)
		})
	}
}

// TestErrorResponseShape_Tools verifies error responses from tool endpoints.
func TestErrorResponseShape_Tools(t *testing.T) {
	s, router := newTestServer()
	router.GET("/v1/tools/:name", s.handleGetTool)
	router.POST("/v1/tools/:name/invoke", s.handleInvokeTool)

	t.Run("GET /v1/tools/:name - not found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/v1/tools/nonexistent_tool", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertErrorShape(t, w, http.StatusNotFound, "not_found")
	})

	t.Run("POST /v1/tools/:name/invoke - invalid body", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/v1/tools/test/invoke", bytes.NewBufferString(`{not json}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertErrorShape(t, w, http.StatusBadRequest, "invalid_request")
	})

	t.Run("POST /v1/tools/:name/invoke - execution failure returns canonical error with details", func(t *testing.T) {
		// Invoke a tool that doesn't exist — should return canonical error shape
		body := `{"inputs": {"query": "test"}}`
		req, _ := http.NewRequest(http.MethodPost, "/v1/tools/nonexistent_tool_xyz/invoke", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		env := assertErrorShape(t, w, http.StatusInternalServerError, "tool_execution_failed")
		// Verify details contains tool metadata
		assert.NotEmpty(t, env.Error.Details, "tool execution error should include details with tool_name, inputs, duration_ms")
	})
}

// TestErrorResponseShape_Memory verifies error responses from memory endpoints.
func TestErrorResponseShape_Memory(t *testing.T) {
	s, router := newTestServer()
	router.POST("/v1/memory", s.handleStoreMemory)
	router.GET("/v1/memory", s.handleListMemories)
	router.GET("/v1/memory/:id", s.handleGetMemory)
	router.PUT("/v1/memory/:id", s.handleUpdateMemory)
	router.DELETE("/v1/memory/:id", s.handleDeleteMemory)
	router.POST("/v1/memory/search", s.handleSearchMemory)

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		expectedCode string
		expectedHTTP int
	}{
		{
			name:         "POST /v1/memory - no manager",
			method:       http.MethodPost,
			path:         "/v1/memory",
			body:         `{"type": "fact", "content": "test"}`,
			expectedCode: "server_error",
			expectedHTTP: http.StatusServiceUnavailable,
		},
		{
			name:         "POST /v1/memory - invalid body",
			method:       http.MethodPost,
			path:         "/v1/memory",
			body:         `{not json}`,
			expectedCode: "server_error",
			expectedHTTP: http.StatusServiceUnavailable, // manager nil checked first
		},
		{
			name:         "GET /v1/memory - no manager",
			method:       http.MethodGet,
			path:         "/v1/memory",
			expectedCode: "server_error",
			expectedHTTP: http.StatusServiceUnavailable,
		},
		{
			name:         "GET /v1/memory/:id - no manager",
			method:       http.MethodGet,
			path:         "/v1/memory/some-id",
			expectedCode: "server_error",
			expectedHTTP: http.StatusServiceUnavailable,
		},
		{
			name:         "PUT /v1/memory/:id - no manager",
			method:       http.MethodPut,
			path:         "/v1/memory/some-id",
			body:         `{"content": "updated"}`,
			expectedCode: "server_error",
			expectedHTTP: http.StatusServiceUnavailable,
		},
		{
			name:         "DELETE /v1/memory/:id - no manager",
			method:       http.MethodDelete,
			path:         "/v1/memory/some-id",
			expectedCode: "server_error",
			expectedHTTP: http.StatusServiceUnavailable,
		},
		{
			name:         "POST /v1/memory/search - no manager",
			method:       http.MethodPost,
			path:         "/v1/memory/search",
			body:         `{"query": "test"}`,
			expectedCode: "server_error",
			expectedHTTP: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			} else {
				body = &bytes.Buffer{}
			}
			req, _ := http.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assertErrorShape(t, w, tt.expectedHTTP, tt.expectedCode)
		})
	}
}

// TestErrorResponseShape_Orchestrate verifies error responses from orchestration endpoints.
func TestErrorResponseShape_Orchestrate(t *testing.T) {
	s, router := newTestServer()
	router.POST("/v1/orchestrate", s.handleOrchestrate)
	router.GET("/v1/orchestrate/:id/events", s.handleOrchestrationEvents)

	t.Run("POST /v1/orchestrate - no engine", func(t *testing.T) {
		body := `{"task_description": "test task"}`
		req, _ := http.NewRequest(http.MethodPost, "/v1/orchestrate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertErrorShape(t, w, http.StatusServiceUnavailable, "server_error")
	})

	t.Run("POST /v1/orchestrate - invalid body", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/v1/orchestrate", bytes.NewBufferString(`{bad}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Engine nil check happens before JSON bind, so it returns service_unavailable
		assertErrorShape(t, w, http.StatusServiceUnavailable, "server_error")
	})

	t.Run("GET /v1/orchestrate/:id/events - not found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/v1/orchestrate/nonexistent/events", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertErrorShape(t, w, http.StatusNotFound, "not_found")
	})
}

// TestErrorResponseShape_GatewayEndpoints verifies error responses from /v1/gateway/* endpoints.
func TestErrorResponseShape_GatewayEndpoints(t *testing.T) {
	s, router := newTestServer()
	router.GET("/v1/gateway/tools", s.handleGatewayListTools)
	router.POST("/v1/gateway/agents", s.handleGatewayCreateAgent)
	router.GET("/v1/gateway/agents/:id", s.handleGatewayGetAgent)
	router.POST("/v1/gateway/agents/:id/chat", s.handleGatewayAgentChat)
	router.POST("/v1/gateway/orchestrate", s.handleGatewayOrchestrate)
	router.GET("/v1/gateway/orchestrate/:id/dag", s.handleGatewayOrchestrationDAG)
	router.GET("/v1/gateway/traces/:id", s.handleGatewayGetTrace)

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		expectedCode string
		expectedHTTP int
	}{
		{
			name:         "GET /v1/gateway/tools - no registry",
			method:       http.MethodGet,
			path:         "/v1/gateway/tools",
			expectedCode: "service_unavailable",
			expectedHTTP: http.StatusServiceUnavailable,
		},
		{
			name:         "POST /v1/gateway/agents - invalid body",
			method:       http.MethodPost,
			path:         "/v1/gateway/agents",
			body:         `{bad}`,
			expectedCode: "invalid_request",
			expectedHTTP: http.StatusBadRequest,
		},
		{
			name:         "POST /v1/gateway/agents - no initializer",
			method:       http.MethodPost,
			path:         "/v1/gateway/agents",
			body:         `{"workspace_root": "/tmp"}`,
			expectedCode: "service_unavailable",
			expectedHTTP: http.StatusServiceUnavailable,
		},
		{
			name:         "GET /v1/gateway/agents/:id - not found",
			method:       http.MethodGet,
			path:         "/v1/gateway/agents/nonexistent",
			expectedCode: "not_found",
			expectedHTTP: http.StatusNotFound,
		},
		{
			name:         "POST /v1/gateway/agents/:id/chat - not found",
			method:       http.MethodPost,
			path:         "/v1/gateway/agents/nonexistent/chat",
			body:         `{"message": "hello"}`,
			expectedCode: "not_found",
			expectedHTTP: http.StatusNotFound,
		},
		{
			name:         "POST /v1/gateway/orchestrate - invalid body",
			method:       http.MethodPost,
			path:         "/v1/gateway/orchestrate",
			body:         `{bad}`,
			expectedCode: "invalid_request",
			expectedHTTP: http.StatusBadRequest,
		},
		{
			name:         "GET /v1/gateway/orchestrate/:id/dag - not found",
			method:       http.MethodGet,
			path:         "/v1/gateway/orchestrate/nonexistent/dag",
			expectedCode: "not_found",
			expectedHTTP: http.StatusNotFound,
		},
		{
			name:         "GET /v1/gateway/traces/:id - no logger",
			method:       http.MethodGet,
			path:         "/v1/gateway/traces/some-trace",
			expectedCode: "service_unavailable",
			expectedHTTP: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			} else {
				body = &bytes.Buffer{}
			}
			req, _ := http.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assertErrorShape(t, w, tt.expectedHTTP, tt.expectedCode)
		})
	}
}

// TestErrorResponseShape_AdminEndpoints verifies error responses from /admin/* endpoints.
func TestErrorResponseShape_AdminEndpoints(t *testing.T) {
	s, router := newTestServer()
	router.POST("/admin/config/reload", s.handleAdminConfigReload)

	t.Run("POST /admin/config/reload - not implemented", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/admin/config/reload", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assertErrorShape(t, w, http.StatusNotImplemented, "not_implemented")
	})
}
