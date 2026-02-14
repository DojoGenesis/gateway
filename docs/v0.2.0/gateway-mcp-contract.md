# Gateway-MCP Integration Contract Specification

**Contract Version:** 1.0.0
**Effective Date:** 2026-02-13
**Layer Boundary:** Runtime (AgenticGateway) ↔ Protocol (MCP Servers)
**Grounded In:** ADR-002, ADR-003, ADR-005
**Status:** DRAFT (Ready for Implementation)

---

## 1. Purpose

This specification defines the binding contract between the AgenticGateway runtime and MCP (Model Context Protocol) servers. It establishes:

- **How the Gateway becomes an MCP host**, embedding MCP protocol handling into the gateway binary
- **How MCP servers are declared, connected, and managed** via declarative YAML configuration
- **How tools from external MCP servers integrate into the Gateway's unified tool namespace** and become available to the orchestration DAG
- **The lifecycle and error handling** of MCP server connections
- **OTEL observability** for all MCP tool invocations
- **The bridge between MCP's native tool model and the Gateway's ToolFunc/ToolDefinition system**

This contract ensures that MCP servers (MCPByDojoGenesis, Composio, user-built, third-party) can be plugged into the Gateway without modifying the orchestration engine, tool registry, or HTTP API.

---

## 2. Contract Parties

| Party | Role | Responsibility |
|-------|------|-----------------|
| **MCP Servers** | Provider | Expose tools via MCP protocol (stdio, SSE, or HTTP); report errors; respect timeouts |
| **Gateway Runtime (mcp/ module)** | Consumer & Mediator | Connect to servers; manage lifecycle; bridge tool models; register in unified namespace |
| **Orchestration Engine** | Client | Request tool resolution; invoke tools; receive results |

---

## 3. MCP Host Architecture

### 3.1 System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                      AgenticGateway Binary                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  pkg/gateway/                                                │  │
│  │  ├─ ToolRegistry (interface from ADR-001)                   │  │
│  │  ├─ Orchestration Engine (DAG resolver)                     │  │
│  │  └─ HTTP API (Server interface)                             │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                             ▲                                       │
│                             │                                       │
│  ┌──────────────────────────┴──────────────────────────────────┐  │
│  │  mcp/ (NEW MODULE)                                          │  │
│  │  ├─ MCPHostManager (config, lifecycle, health checks)       │  │
│  │  ├─ MCPServerConnection (per-server connection state)       │  │
│  │  ├─ MCPToolBridge (ToolFunc adapters)                       │  │
│  │  └─ RegistryAdapter (implements ToolRegistry)              │  │
│  └────────┬─────────────────────────┬──────────────┬───────────┘  │
│           │                         │              │               │
└───────────┼─────────────────────────┼──────────────┼───────────────┘
            │                         │              │
    ┌───────▼────┐          ┌────────▼─────┐   ┌──┴──────────┐
    │MCPByDojo    │          │  Composio    │   │ Custom MCP  │
    │(stdio)      │          │  (SSE over   │   │ Servers     │
    │14 tools,    │          │   HTTP)      │   │ (any)       │
    │20 seeds     │          └──────────────┘   └─────────────┘
    │32 skills    │
    └────────────┘
```

**Data Flow:**

1. **Startup:** Gateway reads YAML config → MCPHostManager initializes → connects to each server → MCPToolBridge adapts tools → registers in ToolRegistry with namespace prefix
2. **Invocation:** Orchestration resolves `mcp_by_dojo:create_artifact` → RegistryAdapter returns MCPToolFunc → tool invocation routed to MCP server → result mapped back to ToolResult
3. **Observability:** Each MCP tool call emits OTEL span with server name, tool name, latency, error details

### 3.2 YAML Configuration Schema

**File Location:** `gateway-config.yaml` (or environment variable override)

```yaml
# Gateway MCP Configuration
version: "1.0"
mcp:
  # Global MCP settings
  global:
    # Default timeout for all MCP tool calls (seconds)
    default_tool_timeout: 30
    # Reconnection strategy
    reconnect:
      max_retries: 3
      initial_backoff_ms: 1000
      max_backoff_ms: 30000
      backoff_multiplier: 2.0
    # Health check interval (seconds)
    health_check_interval: 60
    # Buffer size for streamed responses
    response_buffer_size: 1048576  # 1 MB

  # Individual MCP server definitions
  servers:
    # MCPByDojoGenesis: local stdio server
    - id: "mcp_by_dojo"
      display_name: "MCP By Dojo Genesis"
      namespace_prefix: "mcp_by_dojo"

      # Transport: stdio (local binary)
      transport:
        type: "stdio"
        # Command to start the server
        command: "/opt/mcp-servers/mcp-by-dojo-genesis"
        # Command line arguments
        args:
          - "--port=0"           # auto-select port for stdio handshake
          - "--seed-mode=full"   # enable all 20 seeds
        # Environment variables
        env:
          LOG_LEVEL: "info"
          MCPByDojo_WORKSPACE: "/var/mcp/dojo-workspace"

      # Tool filtering (optional)
      tools:
        # Allowlist (if set, only these tools are registered)
        allowlist: []
        # Blocklist (tools to exclude)
        blocklist: []

      # Connection timeouts (seconds)
      timeouts:
        startup: 10      # time to establish connection
        tool_default: 30 # per-tool invocation timeout
        health_check: 5  # health check timeout

      # Health checks
      health_check:
        enabled: true
        path: "/health"       # MCP health endpoint (if applicable)
        interval_sec: 60

      # Retry policy for failed tool invocations
      retry_policy:
        max_attempts: 2
        backoff_multiplier: 2.0
        max_backoff_ms: 5000

    # Composio: remote SSE server (wrapped REST API)
    - id: "composio"
      display_name: "Composio Integration"
      namespace_prefix: "composio"

      transport:
        type: "sse"
        # HTTP endpoint that serves SSE stream
        url: "https://composio.api.example.com/mcp/sse"
        # Authentication header
        headers:
          Authorization: "Bearer ${COMPOSIO_API_KEY}"  # env var expansion
          X-Client-ID: "agentic-gateway-prod"

      tools:
        allowlist: []   # empty = allow all
        blocklist:
          - "composio:admin:delete_org"  # dangerous operations
          - "composio:admin:modify_config"

      timeouts:
        startup: 15
        tool_default: 30
        health_check: 10

      health_check:
        enabled: true
        interval_sec: 120  # less frequent for remote service

      retry_policy:
        max_attempts: 3
        backoff_multiplier: 1.5
        max_backoff_ms: 10000

    # Custom user-built MCP server
    - id: "user_agents"
      display_name: "User-Built Agent Tools"
      namespace_prefix: "user_agents"

      transport:
        type: "stdio"
        command: "/home/deployment/agents/custom-server"
        args:
          - "--mode=agent"
        env:
          API_KEY: "${USER_AGENTS_API_KEY}"
          DEBUG: "false"

      timeouts:
        startup: 5
        tool_default: 60    # longer timeout for agent operations
        health_check: 5

      health_check:
        enabled: true
        interval_sec: 30

      retry_policy:
        max_attempts: 2
        backoff_multiplier: 2.0
        max_backoff_ms: 3000

  # Enable MCP module observability
  observability:
    enabled: true
    trace_provider: "otel"  # OpenTelemetry
    # Attributes to always include
    attributes:
      service.name: "agentic-gateway"
      service.version: "1.0.0"
      mcp.host.enabled: true
    # Log level for MCP events
    log_level: "info"
    # Sample rate for tool invocation spans (0.0-1.0)
    tool_span_sample_rate: 1.0
```

### 3.3 mcp/ Module Design

**Module Path:** `github.com/TresPies-source/AgenticGatewayByDojoGenesis/mcp`

**Primary Interfaces & Types:**

#### MCPHostConfig

```go
package mcp

import (
	"time"
)

// MCPHostConfig represents the parsed YAML configuration for the MCP host.
type MCPHostConfig struct {
	// Version of the config schema
	Version string `yaml:"version"`

	// Global MCP settings
	Global GlobalMCPConfig `yaml:"global"`

	// Individual server configurations
	Servers []MCPServerConfig `yaml:"servers"`

	// Observability settings
	Observability ObservabilityConfig `yaml:"observability"`
}

// GlobalMCPConfig defines default values and global behavior
type GlobalMCPConfig struct {
	// DefaultToolTimeout is the default timeout for MCP tool calls (seconds)
	DefaultToolTimeout int `yaml:"default_tool_timeout"`

	// ReconnectPolicy governs reconnection behavior
	ReconnectPolicy ReconnectPolicy `yaml:"reconnect"`

	// HealthCheckInterval is how often to check server health (seconds)
	HealthCheckInterval int `yaml:"health_check_interval"`

	// ResponseBufferSize for streamed responses
	ResponseBufferSize int `yaml:"response_buffer_size"`
}

// ReconnectPolicy controls reconnection attempts
type ReconnectPolicy struct {
	MaxRetries         int   `yaml:"max_retries"`
	InitialBackoffMs   int   `yaml:"initial_backoff_ms"`
	MaxBackoffMs       int   `yaml:"max_backoff_ms"`
	BackoffMultiplier  float64 `yaml:"backoff_multiplier"`
}

// MCPServerConfig describes a single MCP server
type MCPServerConfig struct {
	// ID is the unique identifier for this server (alphanumeric + underscore)
	ID string `yaml:"id"`

	// DisplayName is the human-readable name
	DisplayName string `yaml:"display_name"`

	// NamespacePrefix is prepended to all tool names from this server
	// Example: "mcp_by_dojo" → tools become "mcp_by_dojo:create_artifact"
	NamespacePrefix string `yaml:"namespace_prefix"`

	// Transport configuration (stdio, SSE, or HTTP)
	Transport TransportConfig `yaml:"transport"`

	// Tool filtering
	Tools ToolFilterConfig `yaml:"tools"`

	// Connection timeouts
	Timeouts TimeoutConfig `yaml:"timeouts"`

	// Health check settings
	HealthCheck HealthCheckConfig `yaml:"health_check"`

	// Retry policy for failed invocations
	RetryPolicy RetryPolicy `yaml:"retry_policy"`
}

// TransportConfig describes how to connect to the MCP server
type TransportConfig struct {
	// Type: "stdio", "sse", or "streamable_http"
	Type string `yaml:"type"`

	// For stdio: absolute path to executable
	Command string `yaml:"command"`

	// For stdio: command-line arguments (subject to env var expansion)
	Args []string `yaml:"args"`

	// For stdio: environment variables (supports ${VAR} expansion)
	Env map[string]string `yaml:"env"`

	// For SSE/HTTP: server URL
	URL string `yaml:"url"`

	// For SSE/HTTP: HTTP headers (e.g., Authorization, supports ${VAR} expansion)
	Headers map[string]string `yaml:"headers"`
}

// ToolFilterConfig controls which tools are registered
type ToolFilterConfig struct {
	// Allowlist: if non-empty, only these tools are registered
	Allowlist []string `yaml:"allowlist"`

	// Blocklist: tools to exclude from registration
	Blocklist []string `yaml:"blocklist"`
}

// TimeoutConfig specifies timeouts for various operations
type TimeoutConfig struct {
	// Startup is the time to establish the initial connection (seconds)
	Startup int `yaml:"startup"`

	// ToolDefault is the default timeout per tool invocation (seconds)
	ToolDefault int `yaml:"tool_default"`

	// HealthCheck is the timeout for health checks (seconds)
	HealthCheck int `yaml:"health_check"`
}

// HealthCheckConfig controls health monitoring
type HealthCheckConfig struct {
	// Enabled toggles health checks
	Enabled bool `yaml:"enabled"`

	// Path is the endpoint to check (if applicable)
	Path string `yaml:"path"`

	// IntervalSec is how often to run health checks
	IntervalSec int `yaml:"interval_sec"`
}

// RetryPolicy controls retry behavior for tool invocations
type RetryPolicy struct {
	// MaxAttempts is the maximum number of attempts (including initial)
	MaxAttempts int `yaml:"max_attempts"`

	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64 `yaml:"backoff_multiplier"`

	// MaxBackoffMs is the maximum backoff duration
	MaxBackoffMs int `yaml:"max_backoff_ms"`
}

// ObservabilityConfig controls tracing and logging
type ObservabilityConfig struct {
	// Enabled toggles observability
	Enabled bool `yaml:"enabled"`

	// TraceProvider (e.g., "otel" for OpenTelemetry)
	TraceProvider string `yaml:"trace_provider"`

	// Attributes added to all spans
	Attributes map[string]string `yaml:"attributes"`

	// LogLevel for MCP events
	LogLevel string `yaml:"log_level"`

	// ToolSpanSampleRate is the fraction of tool invocations to trace (0.0-1.0)
	ToolSpanSampleRate float64 `yaml:"tool_span_sample_rate"`
}
```

#### MCPServerConnection

```go
package mcp

import (
	"context"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
)

// MCPServerConnection manages the lifecycle and state of a single MCP server connection.
type MCPServerConnection struct {
	// Config is the server configuration
	Config MCPServerConfig

	// Client is the underlying MCP client connection (from mcp-go)
	Client *client.Client

	// State of the connection
	state ConnectionState

	// Mutex protects concurrent access
	mu sync.RWMutex

	// LastHealthCheck is when we last verified the server was healthy
	LastHealthCheck time.Time

	// ConnectedAt is when the connection was established
	ConnectedAt time.Time

	// FailureCount tracks consecutive failures
	FailureCount int

	// Tools cached from the server's ListTools call
	Tools map[string]*ToolDefinitionMCP

	// Channel for health check ticker
	healthCheckDone chan struct{}

	// Logger for this connection
	logger Logger
}

// ConnectionState represents the current state of a server connection
type ConnectionState string

const (
	// StateInitializing: connection being established
	StateInitializing ConnectionState = "initializing"

	// StateConnected: successfully connected and healthy
	StateConnected ConnectionState = "connected"

	// StateUnhealthy: connected but health check failed
	StateUnhealthy ConnectionState = "unhealthy"

	// StateReconnecting: attempting to re-establish connection
	StateReconnecting ConnectionState = "reconnecting"

	// StateDisconnected: not currently connected
	StateDisconnected ConnectionState = "disconnected"

	// StateTerminated: permanently shut down
	StateTerminated ConnectionState = "terminated"
)

// NewMCPServerConnection creates a new connection wrapper for an MCP server.
func NewMCPServerConnection(config MCPServerConfig, logger Logger) *MCPServerConnection {
	return &MCPServerConnection{
		Config:          config,
		state:           StateInitializing,
		Tools:           make(map[string]*ToolDefinitionMCP),
		healthCheckDone: make(chan struct{}),
		logger:          logger,
	}
}

// Connect establishes the MCP connection based on transport type.
// This respects the startup timeout from config.
func (c *MCPServerConnection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	startupDeadline := time.Duration(c.Config.Timeouts.Startup) * time.Second
	ctx, cancel := context.WithTimeout(ctx, startupDeadline)
	defer cancel()

	switch c.Config.Transport.Type {
	case "stdio":
		return c.connectStdio(ctx)
	case "sse":
		return c.connectSSE(ctx)
	case "streamable_http":
		return c.connectStreamableHTTP(ctx)
	default:
		return ErrInvalidTransportType
	}
}

// connectStdio establishes a stdio-based MCP connection.
func (c *MCPServerConnection) connectStdio(ctx context.Context) error {
	// Expand environment variables in command and args
	expandedEnv := expandEnvVars(c.Config.Transport.Env)
	expandedArgs := expandEnvVarsSlice(c.Config.Transport.Args)
	expandedCommand := expandEnvVar(c.Config.Transport.Command)

	// Use mcp-go's StdioTransport
	transport := client.NewStdioTransport(expandedCommand, expandedArgs, expandedEnv)

	// Create MCP client
	mcpClient, err := client.NewClient(ctx, transport)
	if err != nil {
		c.state = StateDisconnected
		return err
	}

	c.Client = mcpClient
	c.state = StateConnected
	c.ConnectedAt = time.Now()
	c.FailureCount = 0

	// Discover tools from the server
	if err := c.discoverTools(ctx); err != nil {
		c.logger.Warnf("Failed to discover tools from %s: %v", c.Config.ID, err)
		// Non-fatal; tools may be available later
	}

	return nil
}

// connectSSE establishes an SSE-based MCP connection.
func (c *MCPServerConnection) connectSSE(ctx context.Context) error {
	// Expand environment variables in URL and headers
	expandedURL := expandEnvVar(c.Config.Transport.URL)
	expandedHeaders := expandEnvVarsMap(c.Config.Transport.Headers)

	// Use mcp-go's SSETransport
	transport := client.NewSSETransport(expandedURL, expandedHeaders)

	mcpClient, err := client.NewClient(ctx, transport)
	if err != nil {
		c.state = StateDisconnected
		return err
	}

	c.Client = mcpClient
	c.state = StateConnected
	c.ConnectedAt = time.Now()
	c.FailureCount = 0

	if err := c.discoverTools(ctx); err != nil {
		c.logger.Warnf("Failed to discover tools from %s: %v", c.Config.ID, err)
	}

	return nil
}

// connectStreamableHTTP establishes an HTTP-based MCP connection (future).
func (c *MCPServerConnection) connectStreamableHTTP(ctx context.Context) error {
	// Placeholder for future streamable HTTP transport
	return ErrTransportNotYetImplemented
}

// discoverTools queries the server for available tools and caches them.
func (c *MCPServerConnection) discoverTools(ctx context.Context) error {
	toolTimeout := time.Duration(c.Config.Timeouts.ToolDefault) * time.Second
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	// Call ListTools on the MCP server
	// This is framework-dependent; assumes mcp-go provides this
	tools, err := c.Client.ListTools(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.Tools = make(map[string]*ToolDefinitionMCP)
	for _, tool := range tools {
		// Apply allowlist/blocklist filters
		if !c.shouldIncludeTool(tool.Name) {
			continue
		}

		c.Tools[tool.Name] = &ToolDefinitionMCP{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}

	return nil
}

// shouldIncludeTool checks if a tool passes the allowlist/blocklist filters.
func (c *MCPServerConnection) shouldIncludeTool(toolName string) bool {
	// If allowlist is non-empty, only include if in allowlist
	if len(c.Config.Tools.Allowlist) > 0 {
		found := false
		for _, allowed := range c.Config.Tools.Allowlist {
			if allowed == toolName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// If in blocklist, exclude
	for _, blocked := range c.Config.Tools.Blocklist {
		if blocked == toolName {
			return false
		}
	}

	return true
}

// InvokeTool calls a tool on the MCP server and returns the result.
// This respects the per-tool timeout and retry policy.
func (c *MCPServerConnection) InvokeTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	toolTimeout := time.Duration(c.Config.Timeouts.ToolDefault) * time.Second

	var lastErr error
	for attempt := 0; attempt < c.Config.RetryPolicy.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff between retries
			backoff := time.Duration(float64(c.Config.RetryPolicy.MaxBackoffMs) * math.Pow(
				c.Config.RetryPolicy.BackoffMultiplier,
				float64(attempt-1),
			)) * time.Millisecond

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		ctx, cancel := context.WithTimeout(ctx, toolTimeout)
		defer cancel()

		// Call the tool via MCP protocol
		result, err := c.Client.CallTool(ctx, toolName, arguments)
		if err == nil {
			c.FailureCount = 0
			return result, nil
		}

		lastErr = err
	}

	c.FailureCount++
	if c.FailureCount > 3 {
		c.state = StateUnhealthy
	}

	return nil, lastErr
}

// GetState returns the current connection state.
func (c *MCPServerConnection) GetState() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Close gracefully closes the MCP connection.
func (c *MCPServerConnection) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Client != nil {
		close(c.healthCheckDone)
		return c.Client.Close(ctx)
	}

	c.state = StateTerminated
	return nil
}
```

#### ToolDefinitionMCP

```go
package mcp

// ToolDefinitionMCP represents a tool from an MCP server.
// This is intermediate representation; MCPToolBridge converts to gateway's ToolDefinition.
type ToolDefinitionMCP struct {
	// Name of the tool (without namespace prefix)
	Name string

	// Description of what the tool does
	Description string

	// InputSchema is the JSON Schema defining input parameters
	// Typically a JSON-serialized schema
	InputSchema interface{}
}
```

#### MCPToolBridge

```go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MCPToolBridge adapts MCP tools into gateway ToolFunc and ToolDefinition.
type MCPToolBridge struct {
	// Connection to the MCP server
	conn *MCPServerConnection

	// Namespace prefix for this server's tools
	namespacePrefix string

	// OTEL tracer for span creation
	tracer trace.Tracer

	// Logger
	logger Logger
}

// NewMCPToolBridge creates a new tool bridge for an MCP server connection.
func NewMCPToolBridge(
	conn *MCPServerConnection,
	namespacePrefix string,
	tracer trace.Tracer,
	logger Logger,
) *MCPToolBridge {
	return &MCPToolBridge{
		conn:               conn,
		namespacePrefix:    namespacePrefix,
		tracer:             tracer,
		logger:             logger,
	}
}

// BridgeTools converts all discovered MCP tools into gateway ToolDefinitions.
// Returns a map of namespaced tool names to gateway ToolDefinitions.
func (b *MCPToolBridge) BridgeTools(ctx context.Context) (map[string]*gateway.ToolDefinition, error) {
	b.conn.mu.RLock()
	tools := b.conn.Tools
	b.conn.mu.RUnlock()

	bridged := make(map[string]*gateway.ToolDefinition)

	for toolName, mcpTool := range tools {
		namespacedName := fmt.Sprintf("%s:%s", b.namespacePrefix, toolName)

		// Create a gateway ToolDefinition
		toolDef := &gateway.ToolDefinition{
			Name:        namespacedName,
			Description: mcpTool.Description,
			Parameters:  mcpTool.InputSchema, // Passed as-is for JSON schema compatibility
			Function:    b.createToolFunc(toolName, mcpTool),
			Timeout:     time.Duration(b.conn.Config.Timeouts.ToolDefault) * time.Second,
		}

		bridged[namespacedName] = toolDef
	}

	return bridged, nil
}

// createToolFunc creates a ToolFunc closure that invokes the MCP tool.
func (b *MCPToolBridge) createToolFunc(mcpToolName string, mcpTool *ToolDefinitionMCP) gateway.ToolFunc {
	return func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Create OTEL span for this tool invocation
		spanCtx, span := b.tracer.Start(ctx, fmt.Sprintf("mcp.tool.invoke"),
			trace.WithAttributes(
				attribute.String("mcp.server_id", b.conn.Config.ID),
				attribute.String("mcp.server_display_name", b.conn.Config.DisplayName),
				attribute.String("mcp.tool_name", mcpToolName),
				attribute.String("mcp.tool_namespaced", fmt.Sprintf("%s:%s", b.namespacePrefix, mcpToolName)),
			),
		)
		defer span.End()

		startTime := time.Now()

		// Invoke the tool via MCP protocol
		result, err := b.conn.InvokeTool(spanCtx, mcpToolName, args)
		latency := time.Since(startTime)

		// Record span attributes
		span.SetAttributes(
			attribute.Int64("mcp.tool_latency_ms", latency.Milliseconds()),
		)

		if err != nil {
			span.RecordError(err)
			span.SetAttributes(
				attribute.String("mcp.tool_error", err.Error()),
				attribute.Bool("mcp.tool_success", false),
			)
			b.logger.Errorf("MCP tool %s:%s failed: %v", b.namespacePrefix, mcpToolName, err)
			return nil, err
		}

		span.SetAttributes(
			attribute.Bool("mcp.tool_success", true),
		)

		// Convert result to map[string]interface{}
		resultMap, err := b.normalizeResult(result)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}

		return resultMap, nil
	}
}

// normalizeResult converts MCP result (which may be various types) to map[string]interface{}.
func (b *MCPToolBridge) normalizeResult(result interface{}) (map[string]interface{}, error) {
	switch v := result.(type) {
	case map[string]interface{}:
		return v, nil
	case string:
		// If result is a string, wrap it
		return map[string]interface{}{"result": v}, nil
	case []byte:
		// Try to parse as JSON
		var m map[string]interface{}
		if err := json.Unmarshal(v, &m); err != nil {
			return map[string]interface{}{"result": string(v)}, nil
		}
		return m, nil
	default:
		// Serialize to JSON and parse back
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return map[string]interface{}{"result": string(data)}, nil
		}
		return m, nil
	}
}
```

#### MCPHostManager

```go
package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"go.opentelemetry.io/otel"
)

// MCPHostManager orchestrates the lifecycle of all MCP server connections
// and manages tool registration into the Gateway's ToolRegistry.
type MCPHostManager struct {
	// Configuration
	config MCPHostConfig

	// Map of server ID to connection
	connections map[string]*MCPServerConnection
	mu          sync.RWMutex

	// Map of server ID to tool bridge
	bridges map[string]*MCPToolBridge

	// Reference to the gateway's tool registry (from ADR-001)
	registry gateway.ToolRegistry

	// OTEL tracer
	tracer interface{} // Will be otel.Tracer in real implementation

	// Logger
	logger Logger

	// Channel for graceful shutdown
	done chan struct{}
}

// NewMCPHostManager creates a new MCP host manager.
func NewMCPHostManager(
	config MCPHostConfig,
	registry gateway.ToolRegistry,
	logger Logger,
) *MCPHostManager {
	return &MCPHostManager{
		config:      config,
		connections: make(map[string]*MCPServerConnection),
		bridges:     make(map[string]*MCPToolBridge),
		registry:    registry,
		tracer:      otel.Tracer("mcp"),
		logger:      logger,
		done:        make(chan struct{}),
	}
}

// Start initializes all configured MCP servers and registers their tools.
func (m *MCPHostManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var startupErrors []error

	for _, serverConfig := range m.config.Servers {
		if err := m.startServer(ctx, serverConfig); err != nil {
			m.logger.Errorf("Failed to start MCP server %s: %v", serverConfig.ID, err)
			startupErrors = append(startupErrors, err)
			// Continue with other servers
		}
	}

	if len(startupErrors) > 0 {
		return fmt.Errorf("failed to start some MCP servers: %v", startupErrors)
	}

	// Start background health checks
	go m.runHealthChecks(ctx)

	return nil
}

// startServer starts a single MCP server and registers its tools.
func (m *MCPHostManager) startServer(ctx context.Context, serverConfig MCPServerConfig) error {
	// Create connection
	conn := NewMCPServerConnection(serverConfig, m.logger)

	// Attempt to connect
	if err := conn.Connect(ctx); err != nil {
		return err
	}

	m.logger.Infof("Connected to MCP server %s (%s)", serverConfig.ID, serverConfig.DisplayName)

	// Create tool bridge
	bridge := NewMCPToolBridge(conn, serverConfig.NamespacePrefix, m.tracer.(interface{}), m.logger)

	// Bridge tools to gateway format
	bridgedTools, err := bridge.BridgeTools(ctx)
	if err != nil {
		conn.Close(ctx)
		return err
	}

	// Register tools in the gateway's registry
	for toolName, toolDef := range bridgedTools {
		if err := m.registry.RegisterTool(toolName, toolDef); err != nil {
			m.logger.Warnf("Failed to register tool %s: %v", toolName, err)
		} else {
			m.logger.Infof("Registered tool %s from server %s", toolName, serverConfig.ID)
		}
	}

	// Store connection and bridge
	m.connections[serverConfig.ID] = conn
	m.bridges[serverConfig.ID] = bridge

	return nil
}

// runHealthChecks periodically checks the health of all MCP servers.
func (m *MCPHostManager) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.config.Global.HealthCheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkServerHealth(ctx)
		case <-m.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkServerHealth verifies that all servers are still responding.
func (m *MCPHostManager) checkServerHealth(ctx context.Context) {
	m.mu.RLock()
	conns := make([]*MCPServerConnection, 0, len(m.connections))
	for _, conn := range m.connections {
		conns = append(conns, conn)
	}
	m.mu.RUnlock()

	for _, conn := range conns {
		healthTimeout := time.Duration(conn.Config.Timeouts.HealthCheck) * time.Second
		healthCtx, cancel := context.WithTimeout(ctx, healthTimeout)

		// Attempt a simple health check (e.g., ListTools)
		err := conn.discoverTools(healthCtx)
		cancel()

		if err != nil {
			m.logger.Warnf("Health check failed for server %s: %v", conn.Config.ID, err)
			// Mark as unhealthy and potentially attempt reconnection
			// (See ADR-002 for reconnection strategy)
		} else {
			conn.LastHealthCheck = time.Now()
		}
	}
}

// Stop gracefully shuts down all MCP server connections.
func (m *MCPHostManager) Stop(ctx context.Context) error {
	close(m.done)

	m.mu.Lock()
	defer m.mu.Unlock()

	var shutdownErrors []error
	for serverID, conn := range m.connections {
		if err := conn.Close(ctx); err != nil {
			m.logger.Errorf("Error closing server %s: %v", serverID, err)
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	if len(shutdownErrors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", shutdownErrors)
	}

	return nil
}

// GetConnection returns the connection for a given server ID (for testing/inspection).
func (m *MCPHostManager) GetConnection(serverID string) (*MCPServerConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connections[serverID]
	return conn, ok
}

// GetStatus returns the status of all MCP servers.
func (m *MCPHostManager) GetStatus() map[string]ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]ServerStatus)
	for serverID, conn := range m.connections {
		status[serverID] = ServerStatus{
			ServerID:        serverID,
			DisplayName:     conn.Config.DisplayName,
			State:           string(conn.GetState()),
			ConnectedAt:     conn.ConnectedAt,
			LastHealthCheck: conn.LastHealthCheck,
			FailureCount:    conn.FailureCount,
			ToolCount:       len(conn.Tools),
		}
	}
	return status
}

// ServerStatus represents the current status of an MCP server.
type ServerStatus struct {
	ServerID        string
	DisplayName     string
	State           string
	ConnectedAt     time.Time
	LastHealthCheck time.Time
	FailureCount    int
	ToolCount       int
}
```

---

## 4. Tool Integration Contract

### 4.1 Namespace Prefixing

**Goal:** Avoid tool name collisions while maintaining clear lineage.

**Rule 1: Namespace Format**
- Format: `<namespace_prefix>:<tool_name>`
- Example: `mcp_by_dojo:create_artifact`, `composio:execute_workflow`, `user_agents:fetch_data`

**Rule 2: Prefix Uniqueness**
- Each MCP server configuration must specify a unique `namespace_prefix`
- Namespace prefixes must be alphanumeric + underscore (regex: `^[a-zA-Z_][a-zA-Z0-9_]*$`)
- Gateway validates all prefixes are unique at startup

**Rule 3: Built-in Tool Namespace**
- Gateway's built-in tools use no prefix or the prefix `gateway`
- Built-in tools: `gateway:execute_node`, `gateway:orchestrate_subgraph`, etc. (if any)
- MCP prefixes cannot use `gateway` or `mcp` (reserved)

**Rule 4: Collision Handling**
- If two servers define tools with identical names after prefixing, the startup fails with a detailed error
- Configuration must be corrected before retry
- Example error: `namespace collision: "composio:execute_tool" already registered; adjust namespace_prefix in config`

**Rule 5: Namespace Visibility**
- The orchestration engine sees namespaced tool names in the unified registry
- DAG node references must use fully qualified names: `tool: "mcp_by_dojo:create_artifact"`
- HTTP API exposes namespaced tools in tool discovery endpoints

### 4.2 Tool Registration Flow

**Step-by-Step Registration Process:**

```
1. Gateway starts → reads YAML config
   └─ Validates all MCP server definitions
   └─ Checks for namespace prefix collisions

2. For each MCP server in config (parallel where safe):
   └─ Create MCPServerConnection(serverConfig)
   └─ Call conn.Connect(ctx) [respects startup timeout]

3. After connect, call conn.discoverTools(ctx):
   └─ MCP CallTool("ListTools", {}) OR equivalent
   └─ Apply tool allowlist/blocklist filters
   └─ Cache ToolDefinitionMCP objects

4. For each discovered tool:
   └─ Create MCPToolBridge for the server
   └─ Call bridge.BridgeTools(ctx)
   └─ For each tool, create gateway.ToolDefinition:
      ├─ Name: "{namespace_prefix}:{tool_name}"
      ├─ Description: from MCP tool description
      ├─ Parameters: JSON schema from MCP
      ├─ Function: closure that calls conn.InvokeTool()
      ├─ Timeout: from config.timeouts.tool_default
   └─ Call registry.RegisterTool(namespacedName, toolDef)

5. After all servers registered:
   └─ Start background health check goroutine
   └─ Log status: "Registered N tools from M servers"
   └─ Return to caller

6. If any server fails to start:
   └─ Log error with server ID and reason
   └─ Continue with other servers (best-effort)
   └─ Return accumulated errors
   └─ Gateway continues; orchestrations referencing unavailable tools will fail at runtime
```

**Pseudo-code:**

```go
func (m *MCPHostManager) Start(ctx context.Context) error {
    for _, serverCfg := range m.config.Servers {
        go m.startServer(ctx, serverCfg)  // Parallel startup
    }

    m.waitForServerStartup()  // Wait for all to complete

    // Validate no namespace collisions
    if err := m.validateNamespaceCollisions(); err != nil {
        return err
    }

    // Start health check loop
    go m.runHealthChecks(ctx)

    return nil
}
```

### 4.3 Tool Invocation Flow

**Step-by-Step Invocation Process:**

```
1. Orchestration DAG requests tool:
   ├─ Query: "resolve tool 'mcp_by_dojo:create_artifact' with args {...}"

2. ToolRegistry.GetTool("mcp_by_dojo:create_artifact"):
   └─ Returns gateway.ToolDefinition with Function = MCPToolFunc

3. Orchestration invokes ToolFunc(ctx, args):
   └─ ToolFunc is the closure created by MCPToolBridge.createToolFunc()

4. Inside closure:
   ├─ Create OTEL span for invocation (sampled per config.observability.tool_span_sample_rate)
   ├─ Extract server from namespace prefix → get MCPServerConnection
   ├─ Call conn.InvokeTool(ctx, mcpToolName, args)

5. MCPServerConnection.InvokeTool():
   ├─ Respect per-tool timeout from config.timeouts.tool_default
   ├─ Implement retry loop (respects config.retry_policy):
   │  ├─ Attempt 1: call MCP client.CallTool(ctx, toolName, args)
   │  ├─ If error AND attempts < max:
   │  │  └─ Sleep with exponential backoff
   │  │  └─ Retry
   │  └─ Return result or error

6. Back in closure:
   ├─ Convert MCP result to map[string]interface{}
   ├─ Record OTEL span attributes:
   │  ├─ mcp.server_id, mcp.tool_name, mcp.tool_latency_ms
   │  ├─ mcp.tool_success (bool)
   │  └─ mcp.tool_error (if error)
   ├─ Return result or error

7. Orchestration receives result:
   └─ Continues DAG execution or marks node as failed
```

**OTEL Span Structure:**

```
Span Name: "mcp.tool.invoke"
Attributes:
  mcp.server_id: "mcp_by_dojo"
  mcp.server_display_name: "MCP By Dojo Genesis"
  mcp.tool_name: "create_artifact"
  mcp.tool_namespaced: "mcp_by_dojo:create_artifact"
  mcp.tool_latency_ms: 150
  mcp.tool_success: true
Events:
  (none if success)
  "error" (with exception) if failure
Status:
  OK if success
  ERROR if failure
```

### 4.4 ToolRegistry Interface Integration

**From ADR-001, the Gateway's ToolRegistry interface:**

```go
// pkg/gateway/registry.go
type ToolRegistry interface {
	// RegisterTool adds a tool to the registry
	RegisterTool(name string, def *ToolDefinition) error

	// GetTool retrieves a tool by name
	GetTool(name string) (*ToolDefinition, error)

	// GetAllTools returns all registered tools
	GetAllTools() map[string]*ToolDefinition

	// UnregisterTool removes a tool (optional for MCP)
	UnregisterTool(name string) error

	// ListToolNames returns all tool names
	ListToolNames() []string
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  interface{} // JSON Schema
	Function    ToolFunc
	Timeout     time.Duration
}

type ToolFunc func(context.Context, map[string]interface{}) (map[string]interface{}, error)
```

**How mcp/ integrates:**

1. MCPHostManager receives a reference to the Gateway's ToolRegistry (from ADR-001 interfaces)
2. During startServer(), MCPToolBridge.BridgeTools() creates gateway.ToolDefinition objects
3. Each tool's Function field is the closure from MCPToolBridge.createToolFunc()
4. MCPHostManager calls registry.RegisterTool(namespacedName, toolDef) for each tool
5. The orchestration engine (pkg/gateway/orchestration) queries the registry and invokes tools normally
6. No changes needed to the orchestration engine; it treats MCP tools identically to built-in tools

---

## 5. Protocol Requirements

### 5.1 Transport Support

**Supported Transports:**

| Transport | Type | Use Case | Status |
|-----------|------|----------|--------|
| **stdio** | local process | MCPByDojoGenesis, custom servers | MVP |
| **SSE** | HTTP Server-Sent Events | Remote servers (Composio) | MVP |
| **streamable_http** | Bidirectional HTTP | Future; reserved | Future |

**stdio Transport Details:**

- **Method:** Fork child process running the MCP server binary
- **I/O:** stdin/stdout for MCP messages, stderr for logging
- **Framework:** Uses mcp-go's StdioTransport
- **Startup:** Server must indicate readiness via MCP protocol handshake
- **Graceful shutdown:** Send EOF to stdin or wait for process exit

**SSE Transport Details:**

- **Method:** HTTP GET to server URL; server streams MCP messages as SSE events
- **I/O:** One-way stream; responses sent as separate HTTP requests or implicit in events
- **Framework:** Uses mcp-go's SSETransport
- **Authentication:** HTTP headers (e.g., Authorization bearer token)
- **Heartbeats:** Server sends keep-alive events to prevent connection timeout
- **Graceful shutdown:** Close HTTP connection

**streamable_http Transport (Future):**

- **Method:** POST requests with streaming bodies
- **I/O:** Bidirectional over single HTTP connection
- **Use Case:** Firewalls that don't allow long-lived connections
- **Status:** Placeholder; not required for MVP

### 5.2 Connection Lifecycle

**State Machine:**

```
[Initializing]
    ↓
    ├─→ [Connected] ← (successful connect) ← [Reconnecting]
    │       ↓
    │   (health check success)
    │       ↓ (health check failure)
    │   [Unhealthy]
    │       ↓
    │   (reconnection logic kicks in → [Reconnecting])
    │
    ├─→ [Disconnected] ← (connect failed)
    │       ↓
    │   (retry logic: sleep + reconnect → [Reconnecting])
    │
    └─→ [Terminated] ← (explicit shutdown or fatal error)
```

**Startup Sequence:**

```
1. MCPServerConnection.Connect(ctx) with startup timeout
2. Based on transport type:
   - stdio: fork process, wait for handshake
   - sse: HTTP GET, establish stream
3. On success: state = Connected, discover tools
4. On timeout/error: state = Disconnected, return error
```

**Health Checking:**

```
1. Background goroutine in MCPHostManager (runHealthChecks)
2. Interval: config.global.health_check_interval
3. For each connection:
   - Call discoverTools(ctx) with health_check timeout
   - If error: increment failure counter, mark Unhealthy
   - If success: reset failure counter, update LastHealthCheck
4. If failure count > threshold: consider reconnection
```

**Reconnection:**

```
1. Triggered by: connection failure, health check failure, explicit call
2. Strategy: exponential backoff with jitter
   - InitialBackoff: config.global.reconnect.initial_backoff_ms
   - MaxBackoff: config.global.reconnect.max_backoff_ms
   - Multiplier: config.global.reconnect.backoff_multiplier
3. Max attempts: config.global.reconnect.max_retries
4. On success: reset failure counter, rediscover tools, re-register
5. On exhaustion: log error, mark Disconnected, do not attempt further reconnects (admin intervention required)
```

**Graceful Shutdown:**

```
1. MCPHostManager.Stop(ctx) is called
2. For each connection:
   - Call conn.Close(ctx) with shutdown timeout
   - stdio: close stdin, wait for process exit
   - sse: close HTTP connection
3. State transitions to Terminated
4. After all closed, return any errors
5. No further operations on closed connections are allowed
```

### 5.3 Error Handling

**Error Categories and Handling:**

| Error | Cause | Action | Recoverable |
|-------|-------|--------|-------------|
| **StartupTimeout** | Server didn't start within timeout | Log error, mark Disconnected, continue | Yes (retry) |
| **ConnectionRefused** | Server not listening | Log error, mark Disconnected, retry | Yes (retry) |
| **ToolNotFound** | Requested tool not in server | Return error to orchestration | No (config issue) |
| **ToolInvocationTimeout** | Tool exceeded timeout | Retry (per retry_policy) or error | Yes (retry) |
| **ToolInvocationError** | Tool returned error | Return error to orchestration | Depends on error |
| **HealthCheckFailed** | Server didn't respond to health check | Mark Unhealthy, initiate reconnect | Yes (reconnect) |
| **InvalidSchema** | MCP message invalid | Log error, close connection | No (protocol error) |

**Retry Semantics:**

```
- Tool invocation failures trigger per-tool retry policy
- Each retry respects the tool timeout independently
- Backoff between retries (exponential)
- If all retries exhausted, error returned to orchestration
- Orchestration can then decide to retry the entire DAG node or fail

Example:
  Tool call fails with transient error
  → Retry 1: wait 100ms, call again
  → Retry 2: wait 200ms, call again
  → All retries exhausted: return error
```

**Error Logging:**

```
- All errors logged with:
  ├─ Timestamp
  ├─ Server ID and display name
  ├─ Operation (connect, invoke, health check)
  ├─ Tool name (if applicable)
  ├─ Error message and stack trace (if internal error)
  └─ Recommended action (reconnect, retry, config change)

Example log entry:
  ERROR mcp: server=mcp_by_dojo tool=create_artifact: invocation timeout (30s) after 2 retries; returning error to orchestration
```

---

## 6. Composio Integration (ADR-003)

**Architecture:**

Composio is treated as a "just another MCP server". The Gateway wraps Composio's REST API inside an MCP-compliant server stub, then connects to it as a standard MCP server.

**YAML Configuration Example:**

```yaml
servers:
  - id: "composio"
    display_name: "Composio Integration"
    namespace_prefix: "composio"

    transport:
      type: "sse"
      url: "https://composio.api.example.com/mcp/sse"
      headers:
        Authorization: "Bearer ${COMPOSIO_API_KEY}"
        X-Client-ID: "agentic-gateway-prod"

    tools:
      # Optionally restrict which Composio tools are available
      blocklist:
        - "composio:admin:delete_org"
        - "composio:admin:modify_config"

    timeouts:
      startup: 15
      tool_default: 30
      health_check: 10

    health_check:
      enabled: true
      interval_sec: 120

    retry_policy:
      max_attempts: 3
      backoff_multiplier: 1.5
      max_backoff_ms: 10000
```

**Composio as MCP Server:**

Composio operates as an SSE-based MCP server exposing:

1. **ListTools:** Returns all available Composio actions (e.g., "gmail:send_email", "slack:post_message")
2. **CallTool:** Routes requests to Composio REST API, translates requests/responses
3. **Tool Parameters:** Composio provides JSON schemas for each action's input/output

**Advantages:**

- No special handling needed in Gateway
- Same tool registration, invocation, and observability as other MCP servers
- Composio can be deployed standalone or co-located with Gateway
- Firewall-friendly (HTTPS SSE, no long-lived connections except SSE stream)

**Composio Server Stub (Reference Implementation):**

The Composio team (or a community contributor) provides a minimal MCP server wrapper that:
- Listens on SSE endpoint
- Translates MCP ListTools → Composio API list actions
- Translates MCP CallTool → Composio API execute action
- Returns MCP tool results from Composio API responses

Example (pseudocode):

```
GET /mcp/sse
  → Stream MCP protocol messages
  → On ListTools call: query Composio API /actions → translate to MCP tools
  → On CallTool call: POST to Composio API /execute → return result
```

---

## 7. Testing Requirements

### 7.1 Unit Tests

**Location:** `mcp/internal/mcp_test.go` (example)

**Test Coverage:**

1. **MCPServerConnection Tests**
   - `TestConnect_Stdio_Success`: Verify stdio connection to local server
   - `TestConnect_Timeout`: Verify startup timeout behavior
   - `TestDiscoverTools_Filtering`: Verify allowlist/blocklist logic
   - `TestInvokeTool_RetryPolicy`: Verify exponential backoff and max retries
   - `TestClose_Graceful`: Verify clean shutdown

2. **MCPToolBridge Tests**
   - `TestBridgeTools_Namespacing`: Verify namespace prefixing
   - `TestCreateToolFunc_OTEL`: Verify OTEL span creation and attributes
   - `TestNormalizeResult_Various`: Verify result conversion (map, string, JSON)

3. **MCPHostManager Tests**
   - `TestStart_MultipleServers`: Verify parallel startup
   - `TestStart_NamespaceCollisionDetection`: Verify collision errors
   - `TestGetStatus`: Verify status reporting
   - `TestHealthChecks`: Verify background health check loop

### 7.2 Integration Tests

**Location:** `mcp/integration_test.go`

**Test Scenarios:**

1. **Mock MCP Server**
   - Implement a simple mock MCP server (stdio) that:
     - Accepts ListTools request, returns fixture tool definitions
     - Accepts CallTool request, returns fixture results
     - Can simulate errors and timeouts

2. **End-to-End Flow**
   - Start Gateway with mock MCP server in YAML config
   - Verify tool registration in ToolRegistry
   - Invoke tool via Orchestration → verify correct result
   - Kill mock server → verify reconnection logic
   - Restart mock server → verify tools re-registered

3. **Namespace Collision**
   - Configure two servers with same namespace prefix
   - Verify startup fails with descriptive error

4. **Tool Filtering**
   - Configure allowlist for some tools
   - Verify only allowlisted tools are registered

5. **Timeout and Retry**
   - Mock server that takes 5s to respond
   - Tool timeout set to 3s
   - Verify timeout error returned
   - Tool timeout set to 10s, retry_policy.max_attempts = 2
   - Mock server fails first attempt, succeeds second
   - Verify tool invocation succeeds after retry

### 7.3 Test Fixtures

```go
// mcp/testdata/mock_server.go
package testdata

import (
	"context"
	"encoding/json"
	"io"
)

// MockMCPServer is a simple stdio-based MCP server for testing
type MockMCPServer struct {
	stdin  io.Reader
	stdout io.Writer
	tools  map[string]*ToolDefinitionMCP
}

func NewMockMCPServer() *MockMCPServer {
	return &MockMCPServer{
		tools: map[string]*ToolDefinitionMCP{
			"test_tool": {
				Name:        "test_tool",
				Description: "A test tool",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"input": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}
}

func (m *MockMCPServer) ListTools(ctx context.Context) ([]*ToolDefinitionMCP, error) {
	// Return fixture tools
	tools := make([]*ToolDefinitionMCP, 0)
	for _, t := range m.tools {
		tools = append(tools, t)
	}
	return tools, nil
}

func (m *MockMCPServer) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	// Return fixture result
	return map[string]interface{}{"status": "ok", "message": "tool executed"}, nil
}
```

---

## 8. Implementation Plan

### Phase 1: Core Protocol Handler (Week 1)

**Deliverables:**

1. Create `mcp/` module in workspace
2. Implement MCPServerConnection with stdio transport
3. Implement basic tool discovery (discoverTools)
4. Create mock MCP server for testing
5. Unit tests for connection lifecycle

**Checklist:**

- [ ] `go.mod` and `go.sum` created for mcp/ module
- [ ] MCPServerConnection struct and Connect() method
- [ ] StdioTransport integration with mcp-go
- [ ] discoverTools() implementation
- [ ] 10+ unit tests passing

### Phase 2: YAML Configuration & Host Manager (Week 1-2)

**Deliverables:**

1. YAML schema and loader (uses standard YAML library)
2. MCPHostManager struct and Start()/Stop() methods
3. Health check loop and reconnection logic
4. Integration with Gateway's ToolRegistry

**Checklist:**

- [ ] YAML parsing (MCPHostConfig unmarshaling)
- [ ] Environment variable expansion (${VAR})
- [ ] Namespace collision detection
- [ ] MCPHostManager.Start() registers all tools
- [ ] Health check loop running in background
- [ ] Integration tests with mock server

### Phase 3: Tool Bridge & Namespace Registration (Week 2)

**Deliverables:**

1. MCPToolBridge struct
2. ToolFunc closure that bridges to MCP protocol
3. Result normalization (map, string, JSON)
4. Integration with gateway.ToolRegistry

**Checklist:**

- [ ] MCPToolBridge.BridgeTools() creates namespaced ToolDefinitions
- [ ] Tool invocation routed correctly to MCP server
- [ ] Result conversion handles various types
- [ ] ToolRegistry.RegisterTool() succeeds
- [ ] End-to-end test: invoke namespaced tool via Orchestration

### Phase 4: OTEL Observability (Week 2-3)

**Deliverables:**

1. OTEL span creation for each tool invocation
2. Span attributes (server, tool, latency, error)
3. Integration with pkg/gateway/ OTEL setup

**Checklist:**

- [ ] Spans created with correct name and attributes
- [ ] Latency recorded in milliseconds
- [ ] Errors captured in span events/status
- [ ] Sample rate honored (from config)
- [ ] Test with OTEL collector (or mock exporter)

### Phase 5: SSE Transport & Composio (Week 3)

**Deliverables:**

1. SSE transport implementation (mcp-go's SSETransport)
2. Composio example YAML config
3. End-to-end test with Composio (or mock SSE server)

**Checklist:**

- [ ] SSETransport creates HTTP connection to server URL
- [ ] Authentication headers passed correctly
- [ ] Tool discovery and invocation via SSE
- [ ] Composio YAML config documented
- [ ] Integration test with mock SSE server

### Phase 6: Documentation & Polish (Week 3)

**Deliverables:**

1. Complete this spec (gateway-mcp-contract.md)
2. godoc comments on all exported functions/types
3. Example configurations in docs/examples/
4. Troubleshooting guide

**Checklist:**

- [ ] All exported functions have godoc comments
- [ ] Example YAML files in docs/examples/
- [ ] Troubleshooting guide covers common errors
- [ ] Spec reviewed and approved

**Timeline Summary:**

```
Week 1:
  - Mon-Tue: Protocol handler (Phase 1)
  - Wed-Fri: Config + host manager (Phase 2)

Week 2:
  - Mon-Tue: Tool bridge (Phase 3)
  - Wed-Fri: OTEL integration (Phase 4)

Week 3:
  - Mon: SSE transport (Phase 5)
  - Tue-Wed: Composio integration (Phase 5)
  - Thu-Fri: Polish and documentation (Phase 6)
```

---

## 9. Risk Assessment

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|-----------|
| **mcp-go API instability** | Breaking changes in mcp-go library | Medium | Pin version; monitor upstream; maintain adapter layer |
| **Namespace collision hard to debug** | Users frustrated with cryptic startup errors | Medium | Clear validation messages; startup logs list all registered namespaces |
| **Tool invocation deadlock** | Goroutine hang if MCP server misbehaves | Low | Context timeouts on all operations; test with misbehaving servers |
| **Health check overhead** | Too frequent checks impact performance | Low | Configurable interval; can be disabled per-server |
| **Reconnection loop consumes resources** | Exponential backoff misconfigured | Low | Max backoff cap; max retries limit; log reconnection attempts |
| **OTEL overhead in high-frequency tools** | Span creation adds latency | Medium | Configurable sample rate; start with 1.0, reduce if needed |

### Operational Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|-----------|
| **MCP server process crash** | Tools unavailable until restart | Medium | Health checks detect; log prominently; consider auto-restart (future) |
| **Network latency to remote MCP server** | Increased tool invocation latency | Medium | Configurable timeouts; document recommended settings; monitor in OTEL |
| **Composio API rate limiting** | Tool invocations fail with 429 | Medium | Respect HTTP 429; implement backoff; document rate limit strategy |
| **Configuration file typos** | Gateway fails to start or silently ignores servers | Medium | Strict YAML validation; warn on unknown fields; startup validation logging |

### Mitigation Strategies

1. **Comprehensive Testing:** Integration tests cover all transports, error scenarios, and edge cases
2. **Clear Error Messages:** Validation errors include suggested fixes
3. **Monitoring & Observability:** OTEL spans + structured logging + health check status endpoint
4. **Configuration Validation:** Schema validation at startup; suggest corrections
5. **Documentation:** Troubleshooting guide, examples, best practices
6. **Graceful Degradation:** If one server fails, others continue; DAG nodes referencing unavailable tools fail with clear error

---

## 10. References

### Architecture Decision Records
- **ADR-001:** Gateway Interfaces (pkg/gateway/ ToolRegistry, Server, Orchestration)
- **ADR-002:** MCP Host Architecture (this spec)
- **ADR-003:** Composio Integration as MCP
- **ADR-005:** OTEL Observability for MCP Tool Invocations

### External Standards & Libraries
- **MCP Protocol:** https://spec.modelcontextprotocol.io/
- **mcp-go:** github.com/mark3labs/mcp-go (v0.8.0+)
- **OpenTelemetry Go:** go.opentelemetry.io/otel
- **JSON Schema:** https://json-schema.org/

### Related Code Locations
- **Gateway Module:** `github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway`
- **Orchestration Engine:** `github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway/orchestration`
- **Tool Registry:** `github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway/tool_registry.go`
- **MCPByDojoGenesis:** `github.com/TresPies-source/MCPByDojoGenesis` (reference MCP server)

### Configuration Examples
- YAML config with MCPByDojoGenesis + Composio: Section 3.2
- Mock MCP server for testing: Section 7.3

---

## Appendix: YAML Configuration Complete Example

```yaml
version: "1.0"

mcp:
  global:
    default_tool_timeout: 30
    reconnect:
      max_retries: 3
      initial_backoff_ms: 1000
      max_backoff_ms: 30000
      backoff_multiplier: 2.0
    health_check_interval: 60
    response_buffer_size: 1048576

  servers:
    # MCPByDojoGenesis
    - id: "mcp_by_dojo"
      display_name: "MCP By Dojo Genesis"
      namespace_prefix: "mcp_by_dojo"

      transport:
        type: "stdio"
        command: "/opt/mcp-servers/mcp-by-dojo-genesis"
        args:
          - "--port=0"
          - "--seed-mode=full"
        env:
          LOG_LEVEL: "info"
          MCPByDojo_WORKSPACE: "/var/mcp/dojo-workspace"

      tools:
        allowlist: []
        blocklist: []

      timeouts:
        startup: 10
        tool_default: 30
        health_check: 5

      health_check:
        enabled: true
        path: "/health"
        interval_sec: 60

      retry_policy:
        max_attempts: 2
        backoff_multiplier: 2.0
        max_backoff_ms: 5000

    # Composio
    - id: "composio"
      display_name: "Composio Integration"
      namespace_prefix: "composio"

      transport:
        type: "sse"
        url: "https://composio.api.example.com/mcp/sse"
        headers:
          Authorization: "Bearer ${COMPOSIO_API_KEY}"
          X-Client-ID: "agentic-gateway-prod"

      tools:
        allowlist: []
        blocklist:
          - "composio:admin:delete_org"
          - "composio:admin:modify_config"

      timeouts:
        startup: 15
        tool_default: 30
        health_check: 10

      health_check:
        enabled: true
        interval_sec: 120

      retry_policy:
        max_attempts: 3
        backoff_multiplier: 1.5
        max_backoff_ms: 10000

    # Custom User Server
    - id: "user_agents"
      display_name: "User-Built Agent Tools"
      namespace_prefix: "user_agents"

      transport:
        type: "stdio"
        command: "/home/deployment/agents/custom-server"
        args:
          - "--mode=agent"
        env:
          API_KEY: "${USER_AGENTS_API_KEY}"
          DEBUG: "false"

      tools:
        allowlist: []
        blocklist: []

      timeouts:
        startup: 5
        tool_default: 60
        health_check: 5

      health_check:
        enabled: true
        interval_sec: 30

      retry_policy:
        max_attempts: 2
        backoff_multiplier: 2.0
        max_backoff_ms: 3000

  observability:
    enabled: true
    trace_provider: "otel"
    attributes:
      service.name: "agentic-gateway"
      service.version: "1.0.0"
      mcp.host.enabled: true
    log_level: "info"
    tool_span_sample_rate: 1.0
```

---

**Document Status:** DRAFT
**Next Review:** After Phase 1 implementation
**Owner:** Architecture Team
**Last Updated:** 2026-02-13
