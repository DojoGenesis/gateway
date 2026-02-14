package mcp

import (
	"context"
	"testing"
)

func TestAdaptMCPTool_SuccessPath(t *testing.T) {
	mcpTool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
		ServerName: "test_server",
	}

	// Create a mock connection
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
		client:  nil, // Not connected - will error
		healthy: false,
	}

	toolFunc := AdaptMCPTool(mcpTool, conn, "test:test_tool")

	// Verify it's a valid ToolFunc
	if toolFunc == nil {
		t.Fatal("AdaptMCPTool() returned nil")
	}

	// Call it - should error because not connected
	ctx := context.Background()
	_, err := toolFunc(ctx, map[string]interface{}{"test": "value"})
	if err == nil {
		t.Error("Expected error when calling disconnected tool")
	}
}

func TestCreateToolDefinition_FullyPopulated(t *testing.T) {
	mcpTool := &Tool{
		Name:        "my_tool",
		Description: "Tool description",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{"type": "string"},
			},
		},
		ServerName: "test_server",
	}

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

	// Per spec: namespace format uses colon separator
	if toolDef.Name != "test:my_tool" {
		t.Errorf("Name = %v, want test:my_tool", toolDef.Name)
	}

	if toolDef.Description != "Tool description" {
		t.Errorf("Description = %v, want 'Tool description'", toolDef.Description)
	}

	if toolDef.Parameters == nil {
		t.Error("Parameters should not be nil")
	}

	if toolDef.Function == nil {
		t.Error("Function should not be nil")
	}

	// Parameters should be the same as InputSchema
	if toolDef.Parameters["type"] != "object" {
		t.Error("Parameters should contain type=object")
	}
}

func TestConvertInputSchema_AllTypes(t *testing.T) {
	tests := []struct {
		name   string
		schema interface{}
	}{
		{
			name:   "nil schema",
			schema: nil,
		},
		{
			name: "complex map schema",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"foo": map[string]interface{}{
						"type":        "string",
						"description": "A string param",
					},
					"bar": map[string]interface{}{
						"type":    "integer",
						"minimum": 0,
					},
				},
				"required": []interface{}{"foo"},
			},
		},
		{
			name:   "string schema",
			schema: "not a map",
		},
		{
			name:   "number schema",
			schema: 123,
		},
		{
			name:   "boolean schema",
			schema: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertInputSchema(tt.schema)
			if result == nil {
				t.Error("convertInputSchema() should never return nil")
			}

			// nil schema returns empty map, others should have 'type'
			if tt.schema != nil {
				if _, ok := result["type"]; !ok {
					t.Error("Non-nil schema result should have a 'type' field")
				}
			}
		})
	}
}

func TestCalculateMapSize_Accuracy(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		min  int // minimum expected size
	}{
		{
			name: "nil map",
			m:    nil,
			min:  0,
		},
		{
			name: "empty map",
			m:    map[string]interface{}{},
			min:  2, // {}
		},
		{
			name: "single key-value",
			m: map[string]interface{}{
				"key": "value",
			},
			min: 10,
		},
		{
			name: "nested map",
			m: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
			min: 20,
		},
		{
			name: "with array",
			m: map[string]interface{}{
				"array": []interface{}{"a", "b", "c"},
			},
			min: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := calculateMapSize(tt.m)
			if size < tt.min {
				t.Errorf("calculateMapSize() = %d, want at least %d", size, tt.min)
			}
		})
	}
}
