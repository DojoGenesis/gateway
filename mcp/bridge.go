package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/tools"
)

// AdaptMCPTool creates a Gateway ToolFunc that bridges to an MCP tool on a remote server.
// The returned function can be registered in the Gateway's tool registry.
//
// The adapter:
//   - Translates Gateway tool call format to MCP tool call format
//   - Invokes the tool on the remote MCP server via the connection
//   - Translates MCP results back to Gateway format
//   - Emits OTEL spans with server_id, server_display_name, tool_name, tool_namespaced, latency, and success attributes
//
// namespacedToolName should be the full tool name (e.g., "mcp_server:tool_name")
func AdaptMCPTool(mcpTool *Tool, bridge *MCPServerConnection, namespacedToolName string) tools.ToolFunc {
	return func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		startTime := time.Now()

		// Start OTEL span with spec-compliant attributes
		serverID := bridge.config.ID
		serverDisplayName := bridge.config.DisplayName
		ctx, span := StartToolCallSpan(ctx, serverID, serverDisplayName, mcpTool.Name, namespacedToolName)
		defer FinishSpan(span)

		// Calculate input size for telemetry
		inputSize := calculateMapSize(input)

		// Call the MCP tool on the remote server
		result, err := bridge.CallTool(ctx, mcpTool.Name, input)

		// Calculate latency
		latency := time.Since(startTime)

		if err != nil {
			// Record error in OTEL span
			RecordToolCallError(span, err, 0)
			return nil, fmt.Errorf("MCP tool %s on server %s failed: %w", mcpTool.Name, bridge.GetName(), err)
		}

		// Calculate output size for telemetry
		outputSize := calculateMapSize(result)

		// Record success in OTEL span
		RecordToolCallSuccess(span, latency, inputSize, outputSize)

		return result, nil
	}
}

// CreateToolDefinition creates a Gateway ToolDefinition from an MCP tool.
// The tool name is prefixed with the namespace prefix from the server config.
// Per spec: namespace format is "namespace_prefix:tool_name" (with colon separator).
func CreateToolDefinition(mcpTool *Tool, bridge *MCPServerConnection, namespacePrefix string) *tools.ToolDefinition {
	// Apply namespace prefix to tool name with colon separator
	// Strip any trailing punctuation from namespace prefix before adding colon
	cleanPrefix := strings.TrimSuffix(strings.TrimSuffix(namespacePrefix, "."), "_")
	fullName := cleanPrefix + ":" + mcpTool.Name

	// Convert MCP input schema to Gateway parameters format
	parameters := convertInputSchema(mcpTool.InputSchema)

	// Create the adapted tool function (pass the namespaced tool name for OTEL)
	toolFunc := AdaptMCPTool(mcpTool, bridge, fullName)

	return &tools.ToolDefinition{
		Name:        fullName,
		Description: mcpTool.Description,
		Parameters:  parameters,
		Function:    toolFunc,
		Timeout:     0, // Use global default timeout
	}
}

// convertInputSchema converts an MCP input schema to Gateway parameters format.
// The input schema is expected to be a JSON Schema object.
func convertInputSchema(schema interface{}) map[string]interface{} {
	if schema == nil {
		return map[string]interface{}{}
	}

	// If schema is already a map, return it
	if m, ok := schema.(map[string]interface{}); ok {
		return m
	}

	// Otherwise, wrap it in a generic object schema
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "object",
				"description": "Tool input parameters",
			},
		},
	}
}

// calculateMapSize estimates the size of a map in bytes by marshaling to JSON.
// Used for telemetry purposes.
func calculateMapSize(m map[string]interface{}) int {
	if m == nil {
		return 0
	}

	data, err := json.Marshal(m)
	if err != nil {
		return 0
	}

	return len(data)
}
