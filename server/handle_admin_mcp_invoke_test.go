package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/mcp"
	"github.com/DojoGenesis/gateway/tools"
)

// stubMCPInvokeRegistry is a minimal in-memory gateway.ToolRegistry test
// double scoped to this test file. server/ has no pre-existing mock of
// gateway.ToolRegistry (mcp/host_test.go's mockToolRegistry lives in a
// different module/package and cannot be imported here).
type stubMCPInvokeRegistry struct {
	tools map[string]*tools.ToolDefinition
}

func newStubMCPInvokeRegistry() *stubMCPInvokeRegistry {
	return &stubMCPInvokeRegistry{tools: make(map[string]*tools.ToolDefinition)}
}

func (s *stubMCPInvokeRegistry) Register(ctx context.Context, def *tools.ToolDefinition) error {
	s.tools[def.Name] = def
	return nil
}

func (s *stubMCPInvokeRegistry) Get(ctx context.Context, name string) (*tools.ToolDefinition, error) {
	tool, ok := s.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

func (s *stubMCPInvokeRegistry) List(ctx context.Context) ([]*tools.ToolDefinition, error) {
	result := make([]*tools.ToolDefinition, 0, len(s.tools))
	for _, t := range s.tools {
		result = append(result, t)
	}
	return result, nil
}

func (s *stubMCPInvokeRegistry) ListByNamespace(ctx context.Context, prefix string) ([]*tools.ToolDefinition, error) {
	result := make([]*tools.ToolDefinition, 0)
	for name, t := range s.tools {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			result = append(result, t)
		}
	}
	return result, nil
}

// stubMCPInvokeHost is a test double for MCPStatusProvider scoped to this
// test file, with a configurable CallTool behavior.
type stubMCPInvokeHost struct {
	callToolFunc func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (map[string]interface{}, error)
}

func (h *stubMCPInvokeHost) Status() map[string]mcp.ServerStatus {
	return map[string]mcp.ServerStatus{}
}

func (h *stubMCPInvokeHost) CallTool(ctx context.Context, serverName string, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	if h.callToolFunc == nil {
		return map[string]interface{}{}, nil
	}
	return h.callToolFunc(ctx, serverName, toolName, args)
}

// newMCPInvokeTestServer builds a Server with a registered "demo:echo" tool
// (requiring a "message" string argument) and the given CallTool behavior.
func newMCPInvokeTestServer(callToolFunc func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (map[string]interface{}, error)) (*Server, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	registry := newStubMCPInvokeRegistry()
	registry.tools["demo:echo"] = &tools.ToolDefinition{
		Name:        "demo:echo",
		Description: "Echoes a message",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"message"},
		},
	}

	s := &Server{
		toolRegistry:   registry,
		mcpHostManager: &stubMCPInvokeHost{callToolFunc: callToolFunc},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})
	router.POST("/admin/mcp/tools/invoke", s.handleAdminInvokeMCPTool)

	return s, router
}

func TestHandleAdminInvokeMCPTool(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		callToolFunc    func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (map[string]interface{}, error)
		expectedHTTP    int
		expectedCode    string // empty means expect 200 success, not an error envelope
		assertOnSuccess func(t *testing.T, body []byte)
		assertOnError   func(t *testing.T, env errorEnvelope)
	}{
		{
			name: "happy path returns 200 with result and duration_ms",
			body: `{"server":"demo","tool":"echo","arguments":{"message":"hi"}}`,
			callToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
				assert.Equal(t, "demo", serverName)
				assert.Equal(t, "echo", toolName)
				assert.Equal(t, "hi", args["message"])
				return map[string]interface{}{"echoed": "hi"}, nil
			},
			expectedHTTP: http.StatusOK,
			assertOnSuccess: func(t *testing.T, body []byte) {
				var resp MCPToolInvokeResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "demo", resp.Server)
				assert.Equal(t, "echo", resp.Tool)
				assert.Equal(t, "hi", resp.Result["echoed"])
				assert.GreaterOrEqual(t, resp.DurationMs, int64(0))
			},
		},
		{
			name:         "server containing colon rejected before any lookup",
			body:         `{"server":"demo:other","tool":"echo","arguments":{}}`,
			expectedHTTP: http.StatusBadRequest,
			expectedCode: "invalid_request",
		},
		{
			name:         "tool containing colon rejected before any lookup",
			body:         `{"server":"demo","tool":"other:sensitive_tool","arguments":{}}`,
			expectedHTTP: http.StatusBadRequest,
			expectedCode: "invalid_request",
		},
		{
			name:         "unknown server:tool combination returns 404",
			body:         `{"server":"unknown_server","tool":"unknown_tool","arguments":{}}`,
			expectedHTTP: http.StatusNotFound,
			expectedCode: "mcp_tool_not_found",
		},
		{
			name:         "arguments failing schema validation return 400",
			body:         `{"server":"demo","tool":"echo","arguments":{}}`,
			expectedHTTP: http.StatusBadRequest,
			expectedCode: "invalid_arguments",
		},
		{
			name: "CallTool ErrServerNotFound maps to 404 mcp_server_not_found",
			body: `{"server":"demo","tool":"echo","arguments":{"message":"hi"}}`,
			callToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
				return nil, fmt.Errorf("%w: %s", mcp.ErrServerNotFound, serverName)
			},
			expectedHTTP: http.StatusNotFound,
			expectedCode: "mcp_server_not_found",
		},
		{
			name: "CallTool ErrServerUnhealthy maps to 503 mcp_server_unavailable",
			body: `{"server":"demo","tool":"echo","arguments":{"message":"hi"}}`,
			callToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
				return nil, fmt.Errorf("%w: %s", mcp.ErrServerUnhealthy, serverName)
			},
			expectedHTTP: http.StatusServiceUnavailable,
			expectedCode: "mcp_server_unavailable",
		},
		{
			name: "CallTool generic error maps to 502 mcp_tool_execution_failed",
			body: `{"server":"demo","tool":"echo","arguments":{"message":"hi"}}`,
			callToolFunc: func(ctx context.Context, serverName, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
				return nil, errors.New("upstream exploded")
			},
			expectedHTTP: http.StatusBadGateway,
			expectedCode: "mcp_tool_execution_failed",
		},
		{
			name:         "malformed JSON body returns 400 invalid_request",
			body:         `{not json}`,
			expectedHTTP: http.StatusBadRequest,
			expectedCode: "invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, router := newMCPInvokeTestServer(tt.callToolFunc)

			req, _ := http.NewRequest(http.MethodPost, "/admin/mcp/tools/invoke", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.expectedCode == "" {
				require.Equal(t, tt.expectedHTTP, w.Code)
				if tt.assertOnSuccess != nil {
					tt.assertOnSuccess(t, w.Body.Bytes())
				}
				return
			}

			env := assertErrorShape(t, w, tt.expectedHTTP, tt.expectedCode)
			if tt.assertOnError != nil {
				tt.assertOnError(t, env)
			}
		})
	}
}

// TestHandleAdminInvokeMCPTool_ColonGuardBeforeLookup verifies that the colon
// guard runs before the registry lookup: with a registry that would panic or
// be observably called on lookup, we assert no lookup occurs for rejected
// input by using a nil-safe registry and confirming the 400 still comes from
// the guard, not from a not-found lookup error.
func TestHandleAdminInvokeMCPTool_ColonGuardBeforeLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	lookupCalled := false
	registry := newStubMCPInvokeRegistry()
	s := &Server{
		toolRegistry:   registryWithLookupTracking{stubMCPInvokeRegistry: registry, called: &lookupCalled},
		mcpHostManager: &stubMCPInvokeHost{},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})
	router.POST("/admin/mcp/tools/invoke", s.handleAdminInvokeMCPTool)

	body := `{"server":"demo:bad","tool":"echo","arguments":{}}`
	req, _ := http.NewRequest(http.MethodPost, "/admin/mcp/tools/invoke", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertErrorShape(t, w, http.StatusBadRequest, "invalid_request")
	assert.False(t, lookupCalled, "registry.Get must not be called when the colon guard rejects the request")
}

// registryWithLookupTracking wraps stubMCPInvokeRegistry to record whether
// Get was invoked, so the colon-guard-before-lookup ordering can be asserted.
type registryWithLookupTracking struct {
	*stubMCPInvokeRegistry
	called *bool
}

func (r registryWithLookupTracking) Get(ctx context.Context, name string) (*tools.ToolDefinition, error) {
	*r.called = true
	return r.stubMCPInvokeRegistry.Get(ctx, name)
}
