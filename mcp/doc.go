// Package mcp implements the Model Context Protocol (MCP) host functionality for the Agentic Gateway.
// It enables the Gateway to connect to external MCP servers, discover their tools, and bridge those tools
// into the Gateway's unified tool registry with namespace prefixing and filtering capabilities.
//
// The package supports:
//   - YAML-based configuration with environment variable expansion
//   - Multiple concurrent MCP server connections via stdio transport
//   - Tool discovery and automatic registration with namespace prefixes
//   - Health monitoring and automatic reconnection
//   - Tool allowlist/blocklist filtering per server
//   - OpenTelemetry span emission for tool invocations
//
// Example usage:
//
//	cfg := &mcp.MCPConfig{
//	    Servers: []mcp.MCPServerConfig{
//	        {
//	            ID:              "composio",
//	            DisplayName:     "Composio Integration",
//	            NamespacePrefix: "composio",
//	            Transport: mcp.TransportConfig{
//	                Type:    "stdio",
//	                Command: "python",
//	                Args:    []string{"-m", "composio.client"},
//	            },
//	        },
//	    },
//	}
//	manager, err := mcp.NewMCPHostManager(cfg, toolRegistry)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if err := manager.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer manager.Stop(ctx)
package mcp
