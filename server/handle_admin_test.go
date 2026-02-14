package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/mcp"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMCPHostManager is a test double for MCPHostManager
type mockMCPHostManager struct {
	status map[string]mcp.ServerStatus
}

func (m *mockMCPHostManager) Status() map[string]mcp.ServerStatus {
	return m.status
}

func (m *mockMCPHostManager) Start(ctx context.Context) error {
	return nil
}

func (m *mockMCPHostManager) Stop(ctx context.Context) error {
	return nil
}

func TestHandleAdminMCPStatus_NoMCPHost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create server without MCP host manager
	server := &Server{
		mcpHostManager: nil,
	}

	// Setup router
	router := gin.New()
	router.GET("/admin/mcp/status", server.handleAdminMCPStatus)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/admin/mcp/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should return empty status when MCP is disabled
	assert.Equal(t, float64(0), response["total_servers"])
	assert.Equal(t, float64(0), response["total_tools"])
	assert.Equal(t, false, response["healthy"])

	servers, ok := response["servers"].(map[string]interface{})
	assert.True(t, ok)
	assert.Empty(t, servers)
}

func TestHandleAdminMCPStatus_SingleConnectedServer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create mock MCP host manager with one connected server
	mockHost := &mockMCPHostManager{
		status: map[string]mcp.ServerStatus{
			"mcp_by_dojo": {
				Name:        "mcp_by_dojo",
				Connected:   true,
				ToolCount:   14,
				LastError:   "",
				LastChecked: time.Now(),
			},
		},
	}

	// Create server with mock MCP host
	server := &Server{
		mcpHostManager: mockHost,
	}

	// Setup router
	router := gin.New()
	router.GET("/admin/mcp/status", server.handleAdminMCPStatus)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/admin/mcp/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify overall status
	assert.Equal(t, float64(1), response["total_servers"])
	assert.Equal(t, float64(14), response["total_tools"])
	assert.Equal(t, true, response["healthy"])

	// Verify server details
	servers, ok := response["servers"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, servers, "mcp_by_dojo")

	serverStatus, ok := servers["mcp_by_dojo"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "mcp_by_dojo", serverStatus["server_id"])
	assert.Equal(t, "connected", serverStatus["state"])
	assert.Equal(t, float64(14), serverStatus["tool_count"])
	assert.Equal(t, "", serverStatus["last_error"])
}

func TestHandleAdminMCPStatus_MultipleServers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create mock MCP host manager with multiple servers
	mockHost := &mockMCPHostManager{
		status: map[string]mcp.ServerStatus{
			"mcp_by_dojo": {
				Name:        "mcp_by_dojo",
				Connected:   true,
				ToolCount:   14,
				LastError:   "",
				LastChecked: time.Now(),
			},
			"composio": {
				Name:        "composio",
				Connected:   true,
				ToolCount:   25,
				LastError:   "",
				LastChecked: time.Now(),
			},
		},
	}

	// Create server with mock MCP host
	server := &Server{
		mcpHostManager: mockHost,
	}

	// Setup router
	router := gin.New()
	router.GET("/admin/mcp/status", server.handleAdminMCPStatus)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/admin/mcp/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify overall status
	assert.Equal(t, float64(2), response["total_servers"])
	assert.Equal(t, float64(39), response["total_tools"]) // 14 + 25
	assert.Equal(t, true, response["healthy"])

	// Verify both servers are present
	servers, ok := response["servers"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, servers, "mcp_by_dojo")
	assert.Contains(t, servers, "composio")
}

func TestHandleAdminMCPStatus_DisconnectedServer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create mock MCP host manager with disconnected server
	mockHost := &mockMCPHostManager{
		status: map[string]mcp.ServerStatus{
			"mcp_by_dojo": {
				Name:        "mcp_by_dojo",
				Connected:   false,
				ToolCount:   0,
				LastError:   "connection timeout",
				LastChecked: time.Now(),
			},
		},
	}

	// Create server with mock MCP host
	server := &Server{
		mcpHostManager: mockHost,
	}

	// Setup router
	router := gin.New()
	router.GET("/admin/mcp/status", server.handleAdminMCPStatus)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/admin/mcp/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Overall health should be false when ANY server is disconnected
	assert.Equal(t, false, response["healthy"])

	// Verify server details
	servers, ok := response["servers"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, servers, "mcp_by_dojo")

	serverStatus, ok := servers["mcp_by_dojo"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "disconnected", serverStatus["state"])
	assert.Equal(t, "connection timeout", serverStatus["last_error"])
}

func TestHandleAdminMCPStatus_MixedHealthServers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create mock MCP host manager with mixed health states
	mockHost := &mockMCPHostManager{
		status: map[string]mcp.ServerStatus{
			"mcp_by_dojo": {
				Name:        "mcp_by_dojo",
				Connected:   true,
				ToolCount:   14,
				LastError:   "",
				LastChecked: time.Now(),
			},
			"composio": {
				Name:        "composio",
				Connected:   false,
				ToolCount:   0,
				LastError:   "authentication failed",
				LastChecked: time.Now(),
			},
		},
	}

	// Create server with mock MCP host
	server := &Server{
		mcpHostManager: mockHost,
	}

	// Setup router
	router := gin.New()
	router.GET("/admin/mcp/status", server.handleAdminMCPStatus)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/admin/mcp/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Overall health should be false when ANY server is disconnected
	assert.Equal(t, false, response["healthy"])

	// Verify tool count only includes connected servers
	assert.Equal(t, float64(14), response["total_tools"])

	// Verify both servers are present
	servers, ok := response["servers"].(map[string]interface{})
	require.True(t, ok)
	assert.Len(t, servers, 2)

	// Check healthy server
	dojoStatus, ok := servers["mcp_by_dojo"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "connected", dojoStatus["state"])

	// Check unhealthy server
	composioStatus, ok := servers["composio"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "disconnected", composioStatus["state"])
	assert.Equal(t, "authentication failed", composioStatus["last_error"])
}
