package mcp

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPByDojoGenesis_ToolRegistration tests that all 14 tools from MCPByDojoGenesis
// register correctly with the expected namespace prefix.
// This is an integration test that requires the real mcp-by-dojo-genesis binary in PATH.
func TestMCPByDojoGenesis_ToolRegistration(t *testing.T) {
	// Skip if binary not in PATH
	if _, err := exec.LookPath("mcp-by-dojo-genesis"); err != nil {
		t.Skip("MCPByDojoGenesis binary not found in PATH, skipping integration test")
	}

	// Load test configuration
	cfg, err := LoadConfig("testdata/config-real-mcp.yaml")
	require.NoError(t, err, "Failed to load test config")

	// Create tool registry
	registry := newMockToolRegistry()

	// Create MCP host manager
	hostManager, err := NewMCPHostManager(&cfg.MCP, registry)
	require.NoError(t, err, "Failed to create MCP host manager")

	// Start MCP host (connects to MCPByDojoGenesis)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = hostManager.Start(ctx)
	require.NoError(t, err, "Failed to start MCP host")

	// Ensure cleanup
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		hostManager.Stop(stopCtx)
	}()

	// List all registered tools
	tools, err := registry.List(context.Background())
	require.NoError(t, err, "Failed to list tools")

	// Extract tool names
	var toolNames []string
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name)
	}

	// Verify we have exactly 14 tools (per spec §4.1)
	assert.Len(t, toolNames, 14, "Expected 14 tools from MCPByDojoGenesis")

	// Verify all tools have the correct namespace prefix
	expectedTools := []string{
		"mcp_by_dojo:create_artifact",
		"mcp_by_dojo:search_wisdom",
		"mcp_by_dojo:apply_seed",
		"mcp_by_dojo:list_seeds",
		"mcp_by_dojo:get_seed",
		"mcp_by_dojo:reflect",
		"mcp_by_dojo:check_pace",
		"mcp_by_dojo:explore_radical_freedom",
		"mcp_by_dojo:practice_inter_acceptance",
		"mcp_by_dojo:trace_lineage",
		"mcp_by_dojo:create_thinking_room",
		"mcp_by_dojo:get_skill",
		"mcp_by_dojo:list_skills",
		"mcp_by_dojo:search_skills",
	}

	// Check that specific tools exist
	for _, expectedTool := range expectedTools {
		assert.Contains(t, toolNames, expectedTool, "Expected tool %s to be registered", expectedTool)
	}

	// Verify namespace filtering works
	dojoTools, err := registry.ListByNamespace(context.Background(), "mcp_by_dojo")
	require.NoError(t, err, "Failed to list tools by namespace")
	assert.Len(t, dojoTools, 14, "Expected all 14 tools to match namespace prefix")
}

// TestMCPByDojoGenesis_ToolInvocation tests that we can successfully invoke
// a tool from MCPByDojoGenesis and get a non-empty result.
func TestMCPByDojoGenesis_ToolInvocation(t *testing.T) {
	// Skip if binary not in PATH
	if _, err := exec.LookPath("mcp-by-dojo-genesis"); err != nil {
		t.Skip("MCPByDojoGenesis binary not found in PATH, skipping integration test")
	}

	// Load test configuration
	cfg, err := LoadConfig("testdata/config-real-mcp.yaml")
	require.NoError(t, err, "Failed to load test config")

	// Create tool registry
	registry := newMockToolRegistry()

	// Create MCP host manager
	hostManager, err := NewMCPHostManager(&cfg.MCP, registry)
	require.NoError(t, err, "Failed to create MCP host manager")

	// Start MCP host
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = hostManager.Start(ctx)
	require.NoError(t, err, "Failed to start MCP host")

	// Ensure cleanup
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		hostManager.Stop(stopCtx)
	}()

	// Get the search_wisdom tool
	tool, err := registry.Get(context.Background(), "mcp_by_dojo:search_wisdom")
	require.NoError(t, err, "Failed to get search_wisdom tool")
	require.NotNil(t, tool, "Tool is nil")
	require.NotNil(t, tool.Function, "Tool function is nil")

	// Invoke the tool
	invokeCtx, invokeCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer invokeCancel()

	result, err := tool.Function(invokeCtx, map[string]interface{}{
		"query": "test search for integration test",
	})

	// Verify invocation succeeded
	require.NoError(t, err, "Tool invocation failed")
	assert.NotNil(t, result, "Tool result is nil")
	assert.NotEmpty(t, result, "Tool result is empty")

	// Log result for inspection (type can vary based on MCP implementation)
	t.Logf("Tool invocation result type: %T, value: %+v", result, result)
}

// TestMCPHostManager_StatusIntegration tests that the Status() method returns accurate
// server connection information with real MCP server.
func TestMCPHostManager_StatusIntegration(t *testing.T) {
	// Skip if binary not in PATH
	if _, err := exec.LookPath("mcp-by-dojo-genesis"); err != nil {
		t.Skip("MCPByDojoGenesis binary not found in PATH, skipping integration test")
	}

	// Load test configuration
	cfg, err := LoadConfig("testdata/config-real-mcp.yaml")
	require.NoError(t, err, "Failed to load test config")

	// Create tool registry
	registry := newMockToolRegistry()

	// Create MCP host manager
	hostManager, err := NewMCPHostManager(&cfg.MCP, registry)
	require.NoError(t, err, "Failed to create MCP host manager")

	// Start MCP host
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = hostManager.Start(ctx)
	require.NoError(t, err, "Failed to start MCP host")

	// Ensure cleanup
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		hostManager.Stop(stopCtx)
	}()

	// Get status
	status := hostManager.Status()

	// Verify status map contains our server
	assert.Contains(t, status, "mcp_by_dojo", "Expected status to contain mcp_by_dojo server")

	// Check server status details
	serverStatus, ok := status["mcp_by_dojo"]
	require.True(t, ok, "Server status not found")

	// Server should be connected
	assert.True(t, serverStatus.Connected, "Expected server to be connected")

	// Should have 14 tools
	assert.Equal(t, 14, serverStatus.ToolCount, "Expected 14 tools")

	// Server name should match
	assert.Equal(t, "mcp_by_dojo", serverStatus.Name, "Server name mismatch")

	// LastChecked should be recent
	assert.WithinDuration(t, time.Now(), serverStatus.LastChecked, 5*time.Second,
		"LastChecked timestamp is too old")

	// LastError should be empty for successful connection
	assert.Empty(t, serverStatus.LastError, "Expected no errors")
}

// TestMCPReconnection tests that the MCP host can recover from a lost connection.
// NOTE: This test is manual/documentary because killing a subprocess reliably in tests
// is platform-specific and complex. This test documents the expected behavior.
func TestMCPReconnection(t *testing.T) {
	t.Skip("Reconnection test requires manual process management - documented for future implementation")

	// FUTURE IMPLEMENTATION OUTLINE:
	// 1. Start MCP host with real server
	// 2. Verify connected state (GetStatus()["mcp_by_dojo"].Connected == true)
	// 3. Kill the MCP server process (platform-specific)
	// 4. Wait for health check cycle to detect failure
	// 5. Verify state transitions to unhealthy (Connected == false)
	// 6. Restart MCP server
	// 7. Wait for reconnection attempt
	// 8. Verify state returns to connected and tools re-register
	// 9. Invoke a tool to verify end-to-end functionality restored
}
