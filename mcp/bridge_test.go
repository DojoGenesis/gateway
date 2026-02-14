package mcp

import (
	"context"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

func TestCreateToolDefinition(t *testing.T) {
	mcpTool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
		ServerName: "test_server",
	}

	// Create a mock connection (won't actually connect)
	conn := &MCPServerConnection{
		name: "test_server",
		config: MCPServerConfig{
			ID:              "test_server",
			DisplayName:     "Test Server",
			NamespacePrefix: "test",
			Transport: TransportConfig{
				Type:    "stdio",
				Command: "echo",
			},
		},
	}

	toolDef := CreateToolDefinition(mcpTool, conn, "test.")

	// Per spec: namespace format is "namespace_prefix:tool_name" (colon separator)
	if toolDef.Name != "test:test_tool" {
		t.Errorf("CreateToolDefinition() Name = %v, want test:test_tool", toolDef.Name)
	}

	if toolDef.Description != "A test tool" {
		t.Errorf("CreateToolDefinition() Description = %v, want 'A test tool'", toolDef.Description)
	}

	if toolDef.Function == nil {
		t.Error("CreateToolDefinition() Function is nil")
	}

	if toolDef.Parameters == nil {
		t.Error("CreateToolDefinition() Parameters is nil")
	}
}

func TestConvertInputSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema interface{}
		want   map[string]interface{}
	}{
		{
			name:   "nil schema",
			schema: nil,
			want:   map[string]interface{}{},
		},
		{
			name: "map schema",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{"type": "string"},
				},
			},
			want: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			name:   "non-map schema",
			schema: "invalid",
			want: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type":        "object",
						"description": "Tool input parameters",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertInputSchema(tt.schema)
			if got == nil {
				t.Error("convertInputSchema() returned nil")
			}
		})
	}
}

func TestCalculateMapSize(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		want int
	}{
		{
			name: "nil map",
			m:    nil,
			want: 0,
		},
		{
			name: "empty map",
			m:    map[string]interface{}{},
			want: 2, // "{}"
		},
		{
			name: "simple map",
			m: map[string]interface{}{
				"key": "value",
			},
			want: 15, // {"key":"value"}
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateMapSize(tt.m)
			if got != tt.want {
				t.Errorf("calculateMapSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdaptMCPTool_ErrorHandling(t *testing.T) {
	mcpTool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{},
		ServerName:  "test_server",
	}

	// Create a mock connection that's not connected
	conn := &MCPServerConnection{
		name: "test_server",
		config: MCPServerConfig{
			ID:              "test_server",
			DisplayName:     "Test Server",
			NamespacePrefix: "test",
			Transport: TransportConfig{
				Type:    "stdio",
				Command: "echo",
			},
		},
		client:  nil, // Not connected
		healthy: false,
	}

	toolFunc := AdaptMCPTool(mcpTool, conn, "test:test_tool")

	// Call the function - should fail because not connected
	ctx := context.Background()
	input := map[string]interface{}{"test": "input"}

	_, err := toolFunc(ctx, input)
	if err == nil {
		t.Error("AdaptMCPTool() expected error when not connected, got nil")
	}
}

func TestToolDefinitionIntegration(t *testing.T) {
	// This test verifies that a created ToolDefinition has the correct structure
	// for Gateway tool registry integration
	mcpTool := &Tool{
		Name:        "composio_search",
		Description: "Search for information",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
			},
			"required": []interface{}{"query"},
		},
		ServerName: "composio",
	}

	conn := &MCPServerConnection{
		name: "composio",
		config: MCPServerConfig{
			ID:              "composio",
			DisplayName:     "Composio Integration",
			NamespacePrefix: "composio",
			Transport: TransportConfig{
				Type:    "stdio",
				Command: "python",
				Args:    []string{"-m", "composio.client"},
			},
		},
	}

	toolDef := CreateToolDefinition(mcpTool, conn, "composio.")

	// Verify ToolDefinition structure (per spec: colon separator)
	if toolDef.Name != "composio:composio_search" {
		t.Errorf("Expected name 'composio:composio_search', got '%s'", toolDef.Name)
	}

	if toolDef.Description != "Search for information" {
		t.Errorf("Expected description 'Search for information', got '%s'", toolDef.Description)
	}

	if toolDef.Function == nil {
		t.Error("Expected Function to be non-nil")
	}

	if toolDef.Parameters == nil {
		t.Error("Expected Parameters to be non-nil")
	}

	// Verify that ToolDefinition matches the tools.ToolDefinition structure
	var _ *tools.ToolDefinition = toolDef
}
