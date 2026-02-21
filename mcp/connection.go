package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPServerConnection manages a single MCP server connection and provides
// methods for tool discovery and invocation.
type MCPServerConnection struct {
	name    string
	config  MCPServerConfig
	client  *client.Client
	mu      sync.RWMutex
	healthy bool
}

// NewMCPServerConnection creates a new connection instance for an MCP server.
// Does not establish the connection; call Connect() to initiate.
func NewMCPServerConnection(name string, cfg MCPServerConfig) (*MCPServerConnection, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server config: %w", err)
	}

	return &MCPServerConnection{
		name:    name,
		config:  cfg,
		healthy: false,
	}, nil
}

// Connect establishes a connection to the MCP server.
// For stdio transport, this starts the subprocess and initializes the client.
func (c *MCPServerConnection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.Transport.Type == "stdio" {
		return c.connectStdio(ctx)
	} else if c.config.Transport.Type == "sse" {
		return c.connectSSE(ctx)
	} else if c.config.Transport.Type == "streamable_http" {
		return c.connectStreamableHTTP(ctx)
	}

	return fmt.Errorf("unsupported transport: %s", c.config.Transport.Type)
}

// connectStdio establishes a stdio-based connection to an MCP server.
func (c *MCPServerConnection) connectStdio(ctx context.Context) error {
	// Prepare environment variables as a slice
	var envList []string
	for k, v := range c.config.Transport.Env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	// Create MCP client (automatically starts the connection)
	mcpClient, err := client.NewStdioMCPClient(
		c.config.Transport.Command,
		envList,
		c.config.Transport.Args...,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server %s: %w", c.name, err)
	}

	// Complete the MCP initialize handshake — required before ListTools or CallTool.
	// The MCP spec mandates: client sends initialize → server responds → client sends
	// initialized notification → normal operation. Skipping this step causes
	// "client not initialized" errors on all subsequent requests.
	startupTimeout := 10 * time.Second
	if c.config.Timeouts.Startup > 0 {
		startupTimeout = time.Duration(c.config.Timeouts.Startup) * time.Second
	}
	initCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	if _, err := mcpClient.Initialize(initCtx, mcp.InitializeRequest{}); err != nil {
		_ = mcpClient.Close()
		return fmt.Errorf("MCP initialize handshake failed for server %s: %w", c.name, err)
	}

	c.client = mcpClient
	c.healthy = true

	return nil
}

// connectSSE establishes an SSE-based connection to an MCP server.
func (c *MCPServerConnection) connectSSE(ctx context.Context) error {
	// Build client options from headers
	var options []transport.ClientOption
	if len(c.config.Transport.Headers) > 0 {
		options = append(options, transport.WithHeaders(c.config.Transport.Headers))
	}

	// Create MCP client using SSE transport
	mcpClient, err := client.NewSSEMCPClient(c.config.Transport.URL, options...)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server %s via SSE: %w", c.name, err)
	}

	c.client = mcpClient
	c.healthy = true

	return nil
}

// connectStreamableHTTP establishes a streamable HTTP connection to an MCP server.
func (c *MCPServerConnection) connectStreamableHTTP(ctx context.Context) error {
	// Build client options from headers
	var options []transport.StreamableHTTPCOption
	if len(c.config.Transport.Headers) > 0 {
		options = append(options, transport.WithHTTPHeaders(c.config.Transport.Headers))
	}

	// Create MCP client using streamable HTTP transport
	mcpClient, err := client.NewStreamableHttpClient(c.config.Transport.URL, options...)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server %s via streamable HTTP: %w", c.name, err)
	}

	c.client = mcpClient
	c.healthy = true

	return nil
}

// Disconnect closes the connection to the MCP server gracefully.
func (c *MCPServerConnection) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return nil
	}

	err := c.client.Close()
	c.client = nil
	c.healthy = false

	return err
}

// ListTools discovers all tools available on the MCP server.
// Returns a list of tool metadata that can be registered in the Gateway.
func (c *MCPServerConnection) ListTools(ctx context.Context) ([]*Tool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.client == nil {
		return nil, fmt.Errorf("not connected to MCP server %s", c.name)
	}

	// Call MCP server's list_tools method
	toolsResult, err := c.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools from %s: %w", c.name, err)
	}

	// Convert MCP tools to our internal Tool representation
	var tools []*Tool
	for _, mcpTool := range toolsResult.Tools {
		// Check if tool is allowed based on allowlist/blocklist
		if !c.config.IsToolAllowed(mcpTool.Name) {
			continue
		}

		tool := &Tool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			InputSchema: mcpTool.InputSchema,
			ServerName:  c.name,
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

// CallTool invokes a tool on the MCP server with the given arguments.
// Returns the result or an error if the invocation fails.
func (c *MCPServerConnection) CallTool(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
	c.mu.RLock()
	mcpClient := c.client
	c.mu.RUnlock()

	if mcpClient == nil {
		return nil, fmt.Errorf("not connected to MCP server %s", c.name)
	}

	// Check if tool is allowed
	if !c.config.IsToolAllowed(name) {
		return nil, fmt.Errorf("tool %s is not allowed on server %s", name, c.name)
	}

	// Set timeout for tool call
	timeout := 60 * time.Second
	if c.config.Timeouts.ToolDefault > 0 {
		timeout = time.Duration(c.config.Timeouts.ToolDefault) * time.Second
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Invoke the tool with retries
	maxRetries := 3
	if c.config.RetryPolicy.MaxAttempts > 0 {
		maxRetries = c.config.RetryPolicy.MaxAttempts
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      name,
				Arguments: args,
			},
		}

		result, err := mcpClient.CallTool(callCtx, request)
		if err == nil {
			// Success - convert result to map
			return convertToolResult(result), nil
		}

		lastErr = err

		// Don't retry if context was cancelled
		if callCtx.Err() != nil {
			break
		}

		// Wait before retrying (exponential backoff)
		if attempt < maxRetries {
			backoff := time.Duration(attempt+1) * 500 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("tool call failed after %d attempts: %w", maxRetries+1, lastErr)
}

// IsHealthy returns true if the connection is currently healthy and active.
func (c *MCPServerConnection) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthy && c.client != nil
}

// GetName returns the name of this MCP server connection.
func (c *MCPServerConnection) GetName() string {
	return c.name
}

// ResourceContent holds the content returned from an MCP server resource read.
type ResourceContent struct {
	URI      string
	MimeType string
	Text     *string // Non-nil for text resources
	Blob     *string // Non-nil for binary (base64) resources
}

// ReadResource reads a specific resource from the MCP server by URI.
func (c *MCPServerConnection) ReadResource(ctx context.Context, uri string) ([]ResourceContent, error) {
	c.mu.RLock()
	mcpClient := c.client
	c.mu.RUnlock()

	if mcpClient == nil {
		return nil, fmt.Errorf("not connected to MCP server %s", c.name)
	}

	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: uri,
		},
	}

	result, err := mcpClient.ReadResource(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource %s from %s: %w", uri, c.name, err)
	}

	var contents []ResourceContent
	for _, rc := range result.Contents {
		switch v := rc.(type) {
		case mcp.TextResourceContents:
			text := v.Text
			contents = append(contents, ResourceContent{
				URI:      v.URI,
				MimeType: v.MIMEType,
				Text:     &text,
			})
		case mcp.BlobResourceContents:
			blob := v.Blob
			contents = append(contents, ResourceContent{
				URI:      v.URI,
				MimeType: v.MIMEType,
				Blob:     &blob,
			})
		}
	}

	return contents, nil
}

// Tool represents an MCP tool discovered from a server.
type Tool struct {
	Name        string
	Description string
	InputSchema interface{}
	ServerName  string
}

// convertToolResult converts an MCP CallToolResult to a map format.
func convertToolResult(result *mcp.CallToolResult) map[string]interface{} {
	if result == nil {
		return map[string]interface{}{}
	}

	// Build the result map
	resultMap := map[string]interface{}{
		"isError": result.IsError,
	}

	// Add content if present
	if len(result.Content) > 0 {
		resultMap["content"] = result.Content
	}

	// Add structured content if present
	if result.StructuredContent != nil {
		resultMap["structuredContent"] = result.StructuredContent
	}

	return resultMap
}
