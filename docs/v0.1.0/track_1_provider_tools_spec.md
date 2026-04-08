# Track 1: Provider + Tools Module Specification
## Agentic Gateway by Dojo Genesis v0.1.0

**Release Date:** Q1 2026
**Stability:** Pre-release
**API Version:** 0.1.0
**License:** Apache 2.0

---

## Table of Contents

1. [Vision](#vision)
2. [Goals & Objectives](#goals--objectives)
3. [Technical Architecture](#technical-architecture)
4. [Module Specifications](#module-specifications)
5. [API Contracts](#api-contracts)
6. [Implementation Details](#implementation-details)
7. [Success Criteria](#success-criteria)
8. [Testing & Quality](#testing--quality)
9. [Deployment & Configuration](#deployment--configuration)
10. [Migration & Compatibility](#migration--compatibility)
11. [Appendix: Code Examples](#appendix-code-examples)

---

## Vision

**Track 1** extracts and cleanly publishes two foundational, independent Go modules that form the bedrock of the Agentic Gateway framework:

1. **Provider Module** (`github.com/DojoGenesis/gateway/provider`) — A pluggable model provider system supporting in-process and out-of-process (gRPC-based) LLM providers, with lifecycle management, discovery, and dynamic configuration.

2. **Tools Module** (`github.com/DojoGenesis/gateway/tools`) — A thread-safe, context-aware tool registry and execution framework with parameter validation, per-tool timeout control, and a clean interface for registering arbitrary tool implementations.

These are **leaf modules** with no external Dojo dependencies. They form the stable, public API foundation for all higher-level orchestration (Track 2) and HTTP serving (Track 3).

**Target Audience:** Developers integrating Dojo's agentic capabilities into their systems; framework authors building on top of Dojo; enterprises deploying multi-model, multi-tool agents.

---

## Goals & Objectives

### Provider Module Goals

- [x] Expose a **clean, immutable `ModelProvider` interface** (6 methods: GetInfo, ListModels, GenerateCompletion, GenerateCompletionStream, CallTool, GenerateEmbedding)
- [x] Implement **pluggable architecture** via HashiCorp go-plugin (gRPC transport)
- [x] Support **in-process providers** (no gRPC overhead) alongside out-of-process plugins
- [x] Deliver **hot-reloading** and **health monitoring** for long-running deployments
- [x] Provide **provider discovery** (auto-scan or explicit config)
- [x] Enable **dynamic configuration** (API keys, model lists) without recompilation
- [x] Thread-safe provider registry with minimal lock contention
- [x] Comprehensive **provider routing logic** (default, guest, authenticated contexts)

### Tools Module Goals

- [x] Expose **immutable `ToolDefinition` type** with name, description, parameters (JSON schema), function, and per-tool timeout
- [x] Implement **thread-safe global registry** (RegisterTool, GetTool, GetAllTools, UnregisterTool)
- [x] Deliver **context-based timeout management** (per-tool override > env var > default 30s)
- [x] Support **tool invocation** (InvokeTool, InvokeToolWithTimeout) with automatic parameter validation
- [x] Provide **JSON schema validation** for tool parameters (strict type checking)
- [x] Enable **tool discovery** (list all registered tools with schemas and descriptions)
- [x] Support **request-scoped context** (project_id injection for tenant isolation)
- [x] Non-invasive integration with agentic workflows (tools = pure functions)

### Events Module Goals (Shared)

- [x] Define **immutable `StreamEvent` type** for the full request-response lifecycle
- [x] Provide **event constructors** for provider, tool, and orchestration events
- [x] Support **JSON serialization/deserialization** for streaming transports
- [x] Maintain **backward compatibility** with existing event types

---

## Technical Architecture

### Module Layout

```
provider/
  ├── go.mod
  ├── go.sum
  ├── interface.go               # ModelProvider interface
  ├── types.go                   # ProviderInfo, ModelInfo, CompletionRequest, etc.
  ├── manager.go                 # PluginManager with discovery, hot-reload, health
  ├── grpc.go                    # gRPC transport layer (server + client)
  ├── rpc.go                     # RPC (legacy) + streaming transport
  ├── config.go                  # ProviderConfig, PluginManagerConfig types
  ├── inprocess.go               # InProcessProvider helper type
  ├── router.go                  # Provider routing logic (default, guest, authenticated)
  └── *_test.go                  # Unit tests

tools/
  ├── go.mod
  ├── go.sum
  ├── types.go                   # ToolDefinition, ToolFunc, ToolResult
  ├── registry.go                # Global registry (RegisterTool, GetTool, etc.)
  ├── executor.go                # InvokeTool, InvokeToolWithTimeout
  ├── validator.go               # Parameter validation (JSON schema)
  ├── context.go                 # Context helpers (WithProjectID, GetProjectIDFromContext)
  ├── examples/
  │   ├── web_search.go          # Example tool: web search
  │   ├── file_operations.go     # Example tool: read/write files
  │   └── system.go              # Example tool: system info
  └── *_test.go                  # Unit tests

events/
  ├── go.mod
  ├── go.sum
  ├── events.go                  # StreamEvent types, event constructors
  └── *_test.go                  # Unit tests
```

### Dependency Graph

```
provider/     (no external Dojo dependencies)
├── github.com/hashicorp/go-plugin
├── github.com/hashicorp/go-hclog
├── google.golang.org/grpc
└── protobuf

tools/        (no external Dojo dependencies)
├── stdlib (context, time, encoding/json, reflect)
└── (optional) events/ for event constructors

events/       (no external Dojo dependencies)
├── stdlib (encoding/json, time)
└── none

Note: Provider and Tools modules are INDEPENDENT. Tools does not import Provider.
Provider does not import Tools (unless supporting provider-hosted tool registries in Track 2/3).
```

### Design Principles

1. **Modularity:** Each module has a single, well-defined responsibility.
2. **Immutability:** Types are immutable where possible (no public setters). Use constructors.
3. **Context Propagation:** All async operations accept `context.Context` for cancellation and deadlines.
4. **Thread Safety:** Registry and manager use RWMutex for concurrent access without deadlock.
5. **No Dojo Specificity:** Types and interfaces are generic enough for any agentic framework.
6. **Clean Interfaces:** Minimal surface area; easy to mock and test.
7. **Production Ready:** Comprehensive error handling, logging, timeout management, health monitoring.

---

## Module Specifications

### 1. Provider Module

#### 1.1 ModelProvider Interface

The core abstraction. Implementations are responsible for communicating with LLM services.

```go
package provider

import "context"

type ModelProvider interface {
	// GetInfo returns metadata about the provider (name, version, capabilities).
	GetInfo(ctx context.Context) (*ProviderInfo, error)

	// ListModels returns all available models from the provider.
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// GenerateCompletion generates a single text completion.
	GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// GenerateCompletionStream generates a completion with streaming chunks.
	GenerateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan *CompletionChunk, error)

	// CallTool invokes a tool external to the provider (e.g., web search via gateway).
	CallTool(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error)

	// GenerateEmbedding produces vector embeddings for text (e.g., for retrieval).
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}
```

**Design Notes:**
- All methods accept `context.Context` for cancellation and timeout propagation.
- Methods return errors for failures; no panics.
- `GenerateCompletionStream` uses a closed channel to signal completion (no explicit Done marker needed).
- `CallTool` allows providers to delegate certain operations back to the gateway (e.g., function calling with gateway-managed tools).

#### 1.2 Type Definitions

```go
package provider

type ProviderInfo struct {
	Name         string   // e.g., "openai", "anthropic", "local-llama"
	Version      string   // e.g., "1.0.0"
	Description  string   // Human-readable description
	Capabilities []string // e.g., ["text-completion", "streaming", "tool-use", "embeddings"]
}

type ModelInfo struct {
	ID          string  // e.g., "gpt-4-turbo-preview"
	Name        string  // Human-readable name
	Provider    string  // Provider name
	ContextSize int     // Context window in tokens
	Cost        float64 // Cost per token (for budgeting)
}

type CompletionRequest struct {
	Model       string                 // Model ID
	Messages    []Message              // Conversation history
	Temperature float64                // 0.0 to 2.0 (controls randomness)
	MaxTokens   int                    // Max output tokens
	Tools       []Tool                 // Optional tools for function calling
	Stream      bool                   // Whether to stream the response
}

type CompletionResponse struct {
	ID        string     // Unique completion ID
	Model     string     // Model used
	Content   string     // Text response
	Usage     Usage      // Token usage
	ToolCalls []ToolCall // Any tool calls in the response
}

type CompletionChunk struct {
	ID    string // Unique chunk ID (same as response)
	Delta string // Incremental text content
	Done  bool   // True on final chunk
}

type Message struct {
	Role       string     // "system", "user", "assistant"
	Content    string     // Message text
	ToolCalls  []ToolCall // Assistant tool calls (when role == "assistant")
	ToolCallID string     // Tool call ID (when role == "tool")
}

type Tool struct {
	Name        string                 // Tool name
	Description string                 // What the tool does
	Parameters  map[string]interface{} // JSON Schema for tool parameters
}

type Usage struct {
	InputTokens  int // Tokens in the request
	OutputTokens int // Tokens in the response
	TotalTokens  int // InputTokens + OutputTokens
}

type ToolCall struct {
	ID        string                 // Unique call ID
	Name      string                 // Tool name
	Arguments map[string]interface{} // Parsed arguments
}

type ToolCallRequest struct {
	ToolCall ToolCall               // The tool call to execute
	Context  map[string]interface{} // Request context (project_id, user_id, etc.)
}

type ToolCallResponse struct {
	Result interface{} // Tool output (can be any JSON-serializable type)
	Error  string      // Error message (empty if successful)
}
```

#### 1.3 PluginManager

Manages the lifecycle of provider plugins (discovery, loading, hot-reloading, health monitoring).

```go
package provider

import "time"

type ProviderConfig struct {
	Name       string                 // Provider name (e.g., "openai")
	Enabled    bool                   // Whether to load this provider
	Priority   int                    // Load priority (lower = earlier)
	PluginPath string                 // Path to plugin binary (relative or absolute)
	Config     map[string]interface{} // Provider-specific config (API keys, etc.)
}

type PluginManagerConfig struct {
	PluginDir          string          // Directory containing plugin binaries
	Providers          []ProviderConfig // Explicit provider configs (optional)
	MonitorInterval    time.Duration   // Health check interval (default 5s)
	RestartDelay       time.Duration   // Delay before restarting crashed plugin (default 1s)
	MaxRestartAttempts int             // Max restarts before giving up (default 3)
}

type PluginManager struct {
	// private fields
}

// NewPluginManager creates a manager with default config.
func NewPluginManager(pluginDir string) *PluginManager

// NewPluginManagerWithConfig creates a manager with explicit config.
func NewPluginManagerWithConfig(config PluginManagerConfig) *PluginManager

// DiscoverPlugins scans the plugin directory and loads all providers.
// If Providers are specified in config, only those are loaded.
func (pm *PluginManager) DiscoverPlugins() error

// LoadPlugin manually loads a single plugin by name.
func (pm *PluginManager) LoadPlugin(name string) error

// GetProvider retrieves a loaded provider by name.
func (pm *PluginManager) GetProvider(name string) (ModelProvider, error)

// GetProviders returns a snapshot of all loaded providers.
func (pm *PluginManager) GetProviders() map[string]ModelProvider

// RegisterProvider registers an in-process provider (no gRPC).
func (pm *PluginManager) RegisterProvider(name string, provider ModelProvider)

// UpdatePluginConfig updates a plugin's config and restarts it.
func (pm *PluginManager) UpdatePluginConfig(name string, configUpdates map[string]interface{}) error

// IsPluginLoaded checks if a provider is currently loaded.
func (pm *PluginManager) IsPluginLoaded(name string) bool

// Shutdown gracefully stops all plugins.
func (pm *PluginManager) Shutdown() error

// Cleanup cancels all monitoring goroutines (alternative to Shutdown).
func (pm *PluginManager) Cleanup()
```

**Features:**
- **Auto-discovery:** Scans plugin directory for executable binaries.
- **Explicit config:** Can specify providers and their configs in code.
- **Hot-reloading:** Detects provider crashes and automatically restarts them.
- **In-process registration:** Support for embedded providers without gRPC overhead.
- **Configuration updates:** Update API keys and model lists without recompilation.
- **Thread-safe:** RWMutex protects concurrent access to provider registry.

#### 1.4 gRPC Transport Layer

For out-of-process plugins, gRPC provides native streaming and context propagation.

```go
package provider

import (
	"google.golang.org/grpc"
	"github.com/hashicorp/go-plugin"
)

// ModelProviderGRPCPlugin bridges the go-plugin interface.
type ModelProviderGRPCPlugin struct {
	plugin.Plugin
	Impl ModelProvider
}

// GRPCServer implements the server side of the plugin.
func (p *ModelProviderGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error

// GRPCClient implements the client side of the plugin.
func (p *ModelProviderGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error)
```

**Protocol:**
- Handshake: `DOJO_GENESIS_PLUGIN` magic cookie, version v0.0.15
- Streaming: gRPC native streaming for completions
- Serialization: JSON for complex types; protobuf for gRPC payloads

#### 1.5 InProcessProvider Helper

For embedding providers directly without gRPC:

```go
package provider

// InProcessProvider wraps a ModelProvider for use without gRPC.
type InProcessProvider struct {
	impl ModelProvider
}

func NewInProcessProvider(impl ModelProvider) *InProcessProvider {
	return &InProcessProvider{impl: impl}
}

func (p *InProcessProvider) GetInfo(ctx context.Context) (*ProviderInfo, error) {
	return p.impl.GetInfo(ctx)
}

// ... other methods delegate to impl
```

#### 1.6 Provider Router

Routes requests to the appropriate provider based on context (default, guest, authenticated).

```go
package provider

type ProviderRouter struct {
	defaultProvider    ModelProvider
	guestProvider      ModelProvider
	authProviders      map[string]ModelProvider // userID -> provider
	manager            *PluginManager
	routingConfig      RoutingConfig
}

type RoutingConfig struct {
	DefaultProvider   string            // Provider name for default requests
	GuestProvider     string            // Provider name for unauthenticated requests
	AuthProviders     map[string]string // userID -> provider name mappings
	FallbackProvider  string            // If routing fails, use this
}

func NewProviderRouter(manager *PluginManager, config RoutingConfig) *ProviderRouter

// GetProvider selects a provider based on context (userID, isGuest).
func (r *ProviderRouter) GetProvider(ctx context.Context) (ModelProvider, error)
```

---

### 2. Tools Module

#### 2.1 Core Types

```go
package tools

import (
	"context"
	"time"
)

// ToolFunc is the function signature for tool implementations.
// ctx: cancellation and deadline context
// params: input parameters (validated against ToolDefinition.Parameters)
// return: tool output (must be JSON-serializable)
type ToolFunc func(context.Context, map[string]interface{}) (map[string]interface{}, error)

type ToolDefinition struct {
	Name        string                 // Unique tool name (e.g., "web_search")
	Description string                 // Human-readable description
	Parameters  map[string]interface{} // JSON Schema for parameters
	Function    ToolFunc               // The actual function to execute
	Timeout     time.Duration          // Per-tool timeout (0 = use global default)
}

type ToolResult struct {
	ToolCallID string // Tool call ID (for tracking in conversation)
	Content    string // Formatted result (JSON string)
}
```

#### 2.2 Global Registry

Thread-safe registry for tool definitions.

```go
package tools

// RegisterTool adds a tool to the registry.
// Returns error if tool already registered or definition is invalid.
func RegisterTool(def *ToolDefinition) error

// GetTool retrieves a tool by name.
// Returns error if tool not found.
func GetTool(name string) (*ToolDefinition, error)

// GetAllTools returns a snapshot of all registered tools.
func GetAllTools() []*ToolDefinition

// UnregisterTool removes a tool from the registry.
// Returns error if tool not found.
func UnregisterTool(name string) error

// ClearRegistry removes all tools (useful for testing).
func ClearRegistry()
```

#### 2.3 Tool Execution

Execute tools with automatic timeout management and parameter validation.

```go
package tools

// InvokeTool executes a tool with its name and parameters.
// Timeout: per-tool override > TOOL_EXECUTION_TIMEOUT env var > default (30s)
func InvokeTool(ctx context.Context, name string, params map[string]interface{}) (map[string]interface{}, error)

// InvokeToolWithTimeout executes a tool with an explicit timeout.
func InvokeToolWithTimeout(ctx context.Context, name string, params map[string]interface{}, timeout time.Duration) (map[string]interface{}, error)

// FormatToolResult converts a map result to a JSON string.
func FormatToolResult(result map[string]interface{}) string
```

**Execution Flow:**
1. Look up tool definition by name
2. Validate parameters against schema
3. Inject project_id from context (if not already present)
4. Create context with timeout
5. Execute function in goroutine
6. Wait for result or timeout
7. Return result or timeout error

#### 2.4 Parameter Validation

JSON Schema-based validation for tool parameters.

```go
package tools

// ValidateParameters checks that params conform to the JSON schema.
// Validates:
// - Required fields are present
// - Field types match expected types
// - Numeric/integer coercion (float64(1.0) == int(1))
func ValidateParameters(params map[string]interface{}, schema map[string]interface{}) error

// Schema expected format:
// {
//   "type": "object",
//   "properties": {
//     "query": { "type": "string" },
//     "limit": { "type": "integer" }
//   },
//   "required": ["query"]
// }
```

#### 2.5 Context Helpers

Request-scoped context for tenant isolation and project tracking.

```go
package tools

// WithProjectID adds a project_id to the context.
// Used for multi-tenant scenarios where tools need to know which project is being operated on.
func WithProjectID(ctx context.Context, projectID string) context.Context

// GetProjectIDFromContext retrieves the project_id from context.
// Returns empty string if not found.
func GetProjectIDFromContext(ctx context.Context) string
```

**Design:**
- `InvokeTool` automatically injects project_id from context into params (if not already present)
- Enables tools to be context-aware without explicit parameter passing
- Thread-safe: no shared mutable state, each request has its own context

#### 2.6 Example Tools

Three reference implementations included in `tools/examples/`:

**web_search.go** — Search the web and return results
```go
name: "web_search"
params:
  - query (string, required)
  - limit (integer, optional, default 10)
returns: { results: [{ title, url, snippet }] }
```

**file_operations.go** — Read and write files locally
```go
name: "read_file"
params:
  - path (string, required)
returns: { content: string }

name: "write_file"
params:
  - path (string, required)
  - content (string, required)
returns: { success: bool }
```

**system.go** — Get system information
```go
name: "get_system_info"
params: none
returns: { os, arch, go_version, uptime_seconds }
```

---

### 3. Events Module

#### 3.1 StreamEvent Type

Central event type for all agentic gateway events.

```go
package events

import "time"

type EventType string

const (
	// Core workflow events
	IntentClassified EventType = "intent_classified"
	ProviderSelected EventType = "provider_selected"
	ToolInvoked      EventType = "tool_invoked"
	ToolCompleted    EventType = "tool_completed"
	Thinking         EventType = "thinking"
	ResponseChunk    EventType = "response_chunk"
	MemoryRetrieved  EventType = "memory_retrieved"
	Complete         EventType = "complete"
	Error            EventType = "error"
	TraceSpanStart   EventType = "trace_span_start"
	TraceSpanEnd     EventType = "trace_span_end"

	// Artifact engine events
	ArtifactCreated  EventType = "artifact_created"
	ArtifactUpdated  EventType = "artifact_updated"
	ProjectSwitched  EventType = "project_switched"
	DiagramRendered  EventType = "diagram_rendered"

	// Orchestration engine events
	OrchestrationPlanCreated EventType = "orchestration_plan_created"
	OrchestrationNodeStart   EventType = "orchestration_node_start"
	OrchestrationNodeEnd     EventType = "orchestration_node_end"
	OrchestrationReplanning  EventType = "orchestration_replanning"
	OrchestrationComplete    EventType = "orchestration_complete"
	OrchestrationFailed      EventType = "orchestration_failed"
)

type StreamEvent struct {
	Type      EventType              `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}
```

#### 3.2 Event Constructors

Immutable constructors for all event types.

```go
package events

// Core events
func NewIntentClassifiedEvent(intent string, confidence float64) StreamEvent
func NewProviderSelectedEvent(provider, model string) StreamEvent
func NewToolInvokedEvent(toolName string, arguments map[string]interface{}) StreamEvent
func NewToolCompletedEvent(toolName string, result interface{}, durationMs int64) StreamEvent
func NewThinkingEvent(message string) StreamEvent
func NewResponseChunkEvent(content string) StreamEvent
func NewMemoryRetrievedEvent(memoriesFound int, memories []interface{}) StreamEvent
func NewCompleteEvent(usage map[string]interface{}) StreamEvent
func NewErrorEvent(errorMessage string, errorCode string) StreamEvent

// Tracing
func NewTraceSpanStartEvent(traceID, spanID, parentID, name string, startTime time.Time, inputs map[string]interface{}) StreamEvent
func NewTraceSpanEndEvent(traceID, spanID, parentID, name string, startTime time.Time, endTime *time.Time, inputs, outputs, metadata map[string]interface{}, status string, durationMs int64) StreamEvent

// Artifact engine
func NewArtifactCreatedEvent(artifactID, artifactName, artifactType, projectID string) StreamEvent
func NewArtifactUpdatedEvent(artifactID, artifactName string, version int, commitMessage string) StreamEvent
func NewProjectSwitchedEvent(projectID, projectName string) StreamEvent
func NewDiagramRenderedEvent(diagramID, diagramType, format string) StreamEvent

// Orchestration
func NewOrchestrationPlanCreatedEvent(planID, taskID string, nodeCount int, estimatedCost float64, plan interface{}) StreamEvent
func NewOrchestrationNodeStartEvent(nodeID, planID, toolName string, parameters map[string]interface{}) StreamEvent
func NewOrchestrationNodeEndEvent(nodeID, planID, toolName, state string, result interface{}, errorMsg string, durationMs int64) StreamEvent
func NewOrchestrationReplanningEvent(planID, taskID, reason string, failedNodes []string) StreamEvent
func NewOrchestrationCompleteEvent(planID, taskID string, totalNodes, successNodes, failedNodes int, durationMs int64) StreamEvent
func NewOrchestrationFailedEvent(planID, taskID, reason string) StreamEvent
```

#### 3.3 Serialization

```go
package events

// ToJSON serializes the event to JSON bytes.
func (e StreamEvent) ToJSON() ([]byte, error)

// String returns a JSON string representation.
func (e StreamEvent) String() string

// FromJSON deserializes JSON bytes to an event.
func FromJSON(data []byte) (*StreamEvent, error)
```

---

## API Contracts

### Provider Module API Contract

**Guarantee:** The `ModelProvider` interface will not change in minor or patch versions. New methods will only be added in major versions.

**Stability:** Stable (v1.0+)

**Compatibility:** Any v1.x version of provider is compatible with any v1.x version of tools or events.

### Tools Module API Contract

**Guarantee:** The `ToolDefinition` type and registry functions will not change in minor or patch versions.

**Stability:** Stable (v1.0+)

**Compatibility:** Tools module is independent and can be used without provider or events modules.

### Events Module API Contract

**Guarantee:** Existing event types and constructors will not be removed. New event types may be added.

**Stability:** Stable (v1.0+)

**Compatibility:** Events can be used independently of provider or tools modules.

---

## Implementation Details

### 1. Provider Module Implementation

#### Plugin Discovery & Loading

1. **Configuration-Driven (Preferred)**
   - User specifies `PluginManagerConfig.Providers`
   - Manager attempts to load each provider in order
   - Skips disabled providers
   - Reports errors but continues

2. **Auto-Discovery (Fallback)**
   - If no config, scans `PluginManagerConfig.PluginDir`
   - Finds all executable files (mode & 0111 != 0)
   - Attempts to load each as a plugin
   - Logs successes/failures

#### Hot-Reloading

1. **Monitoring Loop**
   - Spawns a goroutine per plugin
   - Checks plugin health every `MonitorInterval` (default 5s)
   - If `client.Exited()` is true, plugin crashed

2. **Restart Logic**
   - Wait `RestartDelay` (default 1s) before restarting
   - Increment restart count
   - If restart count >= `MaxRestartAttempts`, give up
   - Otherwise, reload with same binary and config

3. **Cleanup**
   - On shutdown, cancel all monitor goroutines and kill plugins

#### Configuration Injection

- Config map → environment variables
- `api_key` → provider-specific env var (OPENAI_API_KEY, etc.)
- Plugin reads env vars on startup
- No restart needed for non-API-key config changes (depends on provider)

#### Thread Safety

- `sync.RWMutex` protects provider registry
- Readers use RLock (concurrent reads allowed)
- Writers use Lock (exclusive access)
- Lock held for minimal duration (map operations only)

### 2. Tools Module Implementation

#### Registry

- Global `toolRegistry` map (name → *ToolDefinition)
- Protected by `sync.RWMutex`
- Duplicate registration returns error
- Unregister non-existent tool returns error

#### Parameter Validation

1. **Schema Parsing**
   - Expects JSON Schema format (type: "object", properties, required)
   - If no schema, validation passes

2. **Validation Steps**
   - Check required fields present
   - For each param, check type matches schema
   - Allow float64 → int coercion (1.0 == 1)
   - Return error on type mismatch

3. **Supported Types**
   - string, integer, number, boolean, array, object

#### Timeout Management

1. **Precedence**
   - Per-tool timeout (ToolDefinition.Timeout) > highest priority
   - TOOL_EXECUTION_TIMEOUT env var > medium priority
   - Default 30s > lowest priority

2. **Enforcement**
   - InvokeToolWithTimeout creates context with timeout
   - Function runs in goroutine
   - select {} waits for result or <-ctx.Done()
   - On timeout, returns error: "tool execution timeout after 30s"

3. **Graceful Degradation**
   - Timeout fires, goroutine continues (may perform cleanup)
   - Caller is unblocked (receives timeout error)
   - No resource leak (goroutine eventually completes or goes to GC)

#### Context Injection

- `WithProjectID(ctx, projectID)` stores project_id in context
- `InvokeTool` calls `injectProjectIDIntoParams`
- If params already has project_id, don't override
- If context has project_id and params doesn't, add it
- Returns new params map (non-mutating)

### 3. Events Module Implementation

#### Event Constructors

- All constructors set `Type` and `Timestamp: time.Now()`
- Data map is constructed inline
- No validation (allows flexibility for custom data)
- Timestamp is UTC (time.Now() is always UTC)

#### Serialization

- `ToJSON` uses standard `json.Marshal`
- `FromJSON` uses standard `json.Unmarshal`
- Supports round-trip: event → JSON → event

---

## Success Criteria

### Binary Criteria (all must pass)

- [ ] **Provider Module Tests Pass**
  ```bash
  cd provider && go test -v ./...
  # All tests pass, coverage > 80%
  ```

- [ ] **Tools Module Tests Pass**
  ```bash
  cd tools && go test -v ./...
  # All tests pass, coverage > 80%
  ```

- [ ] **Events Module Tests Pass**
  ```bash
  cd events && go test -v ./...
  # All tests pass, coverage > 80%
  ```

- [ ] **ModelProvider Interface Unchanged**
  - Exactly 6 methods: GetInfo, ListModels, GenerateCompletion, GenerateCompletionStream, CallTool, GenerateEmbedding
  - All methods accept `context.Context` as first parameter
  - No new methods added (backward compatibility)

- [ ] **PluginManager Functional**
  ```bash
  # Start manager, discover plugins, retrieve provider
  # Plugin crashes → auto-restart
  # Plugin config updated → provider restarted with new config
  # Shutdown → all plugins killed, no resource leaks
  ```

- [ ] **Tool Registry Functional**
  ```bash
  # RegisterTool → GetTool → UnregisterTool
  # GetAllTools returns all registered tools
  # InvokeTool executes and returns result
  # InvokeTool with timeout fires correctly
  # Parameter validation rejects invalid params
  ```

- [ ] **No Dojo Dependencies**
  - No imports from parent go_backend package
  - No imports from dojo/ package
  - go mod tidy removes all unused dependencies

- [ ] **Example Provider Compiles & Runs**
  ```bash
  # provider/examples/openai-plugin/main.go
  go build -o openai-plugin
  ./openai-plugin  # Runs without error
  ```

- [ ] **Example Tool Registration Works**
  ```go
  tools.RegisterTool(&ToolDefinition{
    Name: "web_search",
    Description: "...",
    Parameters: { "type": "object", ... },
    Function: func(ctx, params) { ... },
  })
  tools.InvokeTool(ctx, "web_search", map[string]interface{}{"query": "..."})
  ```

### Quality Criteria

- [ ] **API Documentation**
  - godoc comments for all exported types and functions
  - Examples for key functions (RegisterTool, InvokeTool, etc.)
  - README.md for each module with setup and usage

- [ ] **Error Handling**
  - All errors include context (wrapped with %w)
  - No silent failures
  - Timeout errors are specific ("tool execution timeout after 30s")

- [ ] **Concurrency**
  - Registry RWMutex correctly used (no deadlocks)
  - No race conditions (go test -race passes)
  - Concurrent tool invocations don't interfere

- [ ] **Testing**
  - Unit tests for all exported functions
  - Integration tests for end-to-end workflows
  - Benchmarks for registry lookup and tool execution
  - Test coverage > 80% (go tool cover)

---

## Testing & Quality

### Unit Tests

#### Provider Module

```
plugin/
  interface_test.go         # Test ModelProvider contract
  types_test.go             # Test type definitions
  manager_test.go           # Test PluginManager (discovery, restart, config)
  grpc_test.go              # Test gRPC transport layer
  router_test.go            # Test provider routing logic
```

**Key Test Cases:**
- Load plugin from disk
- Plugin not found error
- Plugin health check and auto-restart
- Plugin config update and restart
- In-process provider registration
- gRPC serialization round-trip
- Provider router selects correct provider

#### Tools Module

```
tools/
  registry_test.go          # Test RegisterTool, GetTool, UnregisterTool
  executor_test.go          # Test InvokeTool, timeout handling
  validator_test.go         # Test parameter validation
  context_test.go           # Test project ID context injection
```

**Key Test Cases:**
- Register duplicate tool (error)
- Get non-existent tool (error)
- Register/get/unregister cycle
- InvokeTool executes function
- Timeout fires correctly
- Parameter validation rejects invalid types
- Project ID injection works

#### Events Module

```
events/
  events_test.go            # Test event constructors and serialization
```

**Key Test Cases:**
- All event constructors set Type and Timestamp
- ToJSON/FromJSON round-trip
- Event data is accessible

### Integration Tests

**provider/examples/integration_test.go**
- Load example provider plugin
- Call GetInfo, ListModels, GenerateCompletion
- Stream completion chunks
- Restart plugin and verify recovery

**tools/examples/integration_test.go**
- Register multiple tools
- Invoke tools with varying parameter types
- Tool timeout fires and caller unblocks
- Project ID context flows through

### Benchmarks

```
cd provider && go test -bench=. -benchmem
BenchmarkGetProvider-8       1000000     1234 ns/op     256 B/op      4 allocs/op

cd tools && go test -bench=. -benchmem
BenchmarkGetTool-8           5000000      456 ns/op      48 B/op      2 allocs/op
BenchmarkInvokeTool-8        100000      15000 ns/op     2048 B/op     32 allocs/op
```

**Goals:**
- GetProvider < 2µs (fast registry lookup)
- GetTool < 1µs
- InvokeTool < 50ms (excluding actual tool execution)

---

## Deployment & Configuration

### Provider Module Deployment

#### Docker Image for Plugin

```dockerfile
FROM golang:1.24 as builder
WORKDIR /build
COPY . .
RUN go build -o openai-plugin ./examples/openai-plugin/main.go

FROM gcr.io/distroless/base:nonroot
COPY --from=builder /build/openai-plugin /bin/openai-plugin
ENTRYPOINT ["/bin/openai-plugin"]
```

#### Configuration File (provider-config.yaml)

```yaml
pluginDir: /var/lib/dojo-gateway/plugins
monitorInterval: 5s
restartDelay: 1s
maxRestartAttempts: 3

providers:
  - name: openai
    enabled: true
    priority: 1
    pluginPath: openai-plugin
    config:
      api_key: ${OPENAI_API_KEY}  # From env var

  - name: anthropic
    enabled: true
    priority: 2
    pluginPath: anthropic-plugin
    config:
      api_key: ${ANTHROPIC_API_KEY}

  - name: local-llama
    enabled: false  # Disabled for this deployment
    priority: 100
    pluginPath: llama-plugin
    config: {}
```

#### Plugin Binary Checklist

- [ ] Executable (chmod +x)
- [ ] Implements ModelProvider interface
- [ ] Responds to gRPC/RPC calls
- [ ] Exits gracefully on context cancellation
- [ ] Logs to stderr (picked up by manager)
- [ ] No signal handlers (go-plugin manages signals)

### Tools Module Deployment

#### Tool Registration at Startup

```go
package main

import "github.com/DojoGenesis/gateway/tools"

func init() {
	tools.RegisterTool(&tools.ToolDefinition{
		Name: "web_search",
		Description: "Search the web and return results",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
				"limit": map[string]interface{}{"type": "integer"},
			},
			"required": []interface{}{"query"},
		},
		Function: webSearchFunc,
		Timeout: 30 * time.Second,
	})
}

func webSearchFunc(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	// Implementation
}
```

#### Environment Configuration

```bash
# Global tool timeout (default 30s)
export TOOL_EXECUTION_TIMEOUT=60

# Tool-specific config (passed as params)
export SEARCH_API_KEY=sk-...
export SEARCH_API_ENDPOINT=https://api.example.com
```

---

## Migration & Compatibility

### From Monolith to Modular

**Before (v0.0.x):** All code in single `go_backend` package.

**After (v1.0):** Three independent modules with clean imports.

**Migration Path:**

1. **Phase 1:** Extract provider/ and tools/ into new modules
   - Copy source files
   - Create go.mod for each
   - Update imports in each module
   - Publish as separate packages

2. **Phase 2:** Update go_backend to import provider/ and tools/
   - go get github.com/DojoGenesis/gateway/provider v1.0.0
   - go get github.com/DojoGenesis/gateway/tools v1.0.0
   - Remove local plugin/, tools/, events/ packages
   - Point imports to new modules

3. **Phase 3:** Update consumer applications
   - Replace import "github.com/dojo-genesis/go_backend/plugin" with "github.com/DojoGenesis/gateway/provider"
   - Replace import "github.com/dojo-genesis/go_backend/tools" with "github.com/DojoGenesis/gateway/tools"
   - Replace import "github.com/dojo-genesis/go_backend/events" with "github.com/DojoGenesis/gateway/events"

### Backward Compatibility

- **API Contracts:** All exported types and functions remain unchanged (v1.0 → v1.x).
- **Wire Format:** gRPC messages and JSON serialization are stable.
- **Behavior:** Tool timeout, context injection, registry semantics unchanged.

### Breaking Changes (v1.0 → v2.0, if ever)

- Removing an interface method
- Changing type of struct field
- Changing error behavior
- Removing a public type or function

These would only happen in major version bump with long deprecation period.

---

## Appendix: Code Examples

### Example 1: Using the Provider Module

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/DojoGenesis/gateway/provider"
)

func main() {
	// Create manager
	mgr := provider.NewPluginManager("/opt/providers/plugins")

	// Discover and load providers from disk
	if err := mgr.DiscoverPlugins(); err != nil {
		log.Fatalf("discovery failed: %v", err)
	}

	// Get a specific provider
	openai, err := mgr.GetProvider("openai")
	if err != nil {
		log.Fatalf("provider not found: %v", err)
	}

	// Get provider info
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := openai.GetInfo(ctx)
	if err != nil {
		log.Fatalf("GetInfo failed: %v", err)
	}
	log.Printf("Provider: %s v%s", info.Name, info.Version)

	// List models
	models, err := openai.ListModels(ctx)
	if err != nil {
		log.Fatalf("ListModels failed: %v", err)
	}
	for _, m := range models {
		log.Printf("  - %s (ctx: %d tokens)", m.Name, m.ContextSize)
	}

	// Generate completion
	req := &provider.CompletionRequest{
		Model: "gpt-4-turbo-preview",
		Messages: []provider.Message{
			{Role: "user", Content: "Hello!"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := openai.GenerateCompletion(ctx, req)
	if err != nil {
		log.Fatalf("GenerateCompletion failed: %v", err)
	}
	log.Printf("Response: %s", resp.Content)
	log.Printf("Usage: %d input, %d output", resp.Usage.InputTokens, resp.Usage.OutputTokens)

	// Streaming
	req.Stream = true
	chunks, err := openai.GenerateCompletionStream(ctx, req)
	if err != nil {
		log.Fatalf("GenerateCompletionStream failed: %v", err)
	}

	for chunk := range chunks {
		print(chunk.Delta)
	}
	println()

	// Shutdown
	mgr.Shutdown()
}
```

### Example 2: Using the Tools Module

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DojoGenesis/gateway/tools"
)

func main() {
	// Register a tool
	err := tools.RegisterTool(&tools.ToolDefinition{
		Name: "web_search",
		Description: "Search the web for information",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
				"limit": map[string]interface{}{
					"type": "integer",
					"default": 10,
				},
			},
			"required": []interface{}{"query"},
		},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			query, _ := params["query"].(string)
			limit := 10
			if l, ok := params["limit"].(float64); ok {
				limit = int(l)
			}

			log.Printf("Searching for: %s (limit: %d)", query, limit)

			// Simulate search
			results := []map[string]interface{}{
				{"title": "Result 1", "url": "https://example.com/1", "snippet": "..."},
				{"title": "Result 2", "url": "https://example.com/2", "snippet": "..."},
			}

			return map[string]interface{}{
				"query":   query,
				"results": results,
				"count":   len(results),
			}, nil
		},
		Timeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("RegisterTool failed: %v", err)
	}

	// Get tool info
	tool, err := tools.GetTool("web_search")
	if err != nil {
		log.Fatalf("GetTool failed: %v", err)
	}
	log.Printf("Tool: %s - %s", tool.Name, tool.Description)

	// Create context with project ID
	ctx := tools.WithProjectID(context.Background(), "project-123")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Invoke tool
	params := map[string]interface{}{
		"query": "golang context patterns",
		"limit": 5,
	}

	result, err := tools.InvokeTool(ctx, "web_search", params)
	if err != nil {
		log.Fatalf("InvokeTool failed: %v", err)
	}

	log.Printf("Result: %+v", result)

	// List all tools
	allTools := tools.GetAllTools()
	log.Printf("Registered tools: %d", len(allTools))
	for _, t := range allTools {
		log.Printf("  - %s", t.Name)
	}

	// Unregister tool
	err = tools.UnregisterTool("web_search")
	if err != nil {
		log.Fatalf("UnregisterTool failed: %v", err)
	}
}
```

### Example 3: Writing a Provider Plugin

```go
package main

import (
	"context"
	"os"

	"github.com/hashicorp/go-plugin"
	"github.com/DojoGenesis/gateway/provider"
)

// OpenAIProvider implements ModelProvider
type OpenAIProvider struct {
	apiKey string
}

func (p *OpenAIProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:        "openai",
		Version:     "1.0.0",
		Description: "OpenAI API provider for GPT models",
		Capabilities: []string{
			"text-completion",
			"streaming",
			"tool-use",
			"embeddings",
		},
	}, nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{
			ID:          "gpt-4-turbo-preview",
			Name:        "GPT-4 Turbo",
			Provider:    "openai",
			ContextSize: 128000,
			Cost:        0.00003, // per input token
		},
		{
			ID:          "gpt-3.5-turbo",
			Name:        "GPT-3.5 Turbo",
			Provider:    "openai",
			ContextSize: 16385,
			Cost:        0.0000005,
		},
	}, nil
}

func (p *OpenAIProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	// Call OpenAI API
	// Return response
	return &provider.CompletionResponse{
		ID:      "chatcmpl-123",
		Model:   req.Model,
		Content: "Hello, how can I help?",
		Usage: provider.Usage{
			InputTokens:  10,
			OutputTokens: 10,
			TotalTokens:  20,
		},
	}, nil
}

func (p *OpenAIProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	ch := make(chan *provider.CompletionChunk, 10)

	go func() {
		defer close(ch)

		// Stream chunks from OpenAI API
		ch <- &provider.CompletionChunk{
			ID:    "chatcmpl-123",
			Delta: "Hello, ",
			Done:  false,
		}
		ch <- &provider.CompletionChunk{
			ID:    "chatcmpl-123",
			Delta: "how can I help?",
			Done:  true,
		}
	}()

	return ch, nil
}

func (p *OpenAIProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	// Delegate to gateway (not implemented in provider)
	return &provider.ToolCallResponse{
		Result: nil,
		Error:  "tool calling not supported",
	}, nil
}

func (p *OpenAIProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Call OpenAI embedding API
	return make([]float32, 1536), nil // Example: 1536-dim embedding
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: provider.Handshake,
		Plugins: map[string]plugin.Plugin{
			"provider": &provider.ModelProviderGRPCPlugin{
				Impl: &OpenAIProvider{apiKey: apiKey},
			},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
```

### Example 4: Streaming Events

```go
package main

import (
	"encoding/json"
	"log"

	"github.com/DojoGenesis/gateway/events"
)

func main() {
	// Create an event
	event := events.NewToolInvokedEvent("web_search", map[string]interface{}{
		"query": "golang best practices",
	})

	log.Printf("Event Type: %s", event.Type)
	log.Printf("Event Time: %s", event.Timestamp)
	log.Printf("Event Data: %v", event.Data)

	// Serialize to JSON
	jsonBytes, err := event.ToJSON()
	if err != nil {
		log.Fatalf("ToJSON failed: %v", err)
	}
	log.Printf("JSON: %s", string(jsonBytes))

	// Deserialize from JSON
	var parsed events.StreamEvent
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		log.Fatalf("Unmarshal failed: %v", err)
	}

	log.Printf("Parsed Event Type: %s", parsed.Type)
	log.Printf("Parsed Event Data: %v", parsed.Data)

	// Create a sequence of events
	sequence := []events.StreamEvent{
		events.NewIntentClassifiedEvent("search", 0.95),
		events.NewProviderSelectedEvent("openai", "gpt-4-turbo-preview"),
		events.NewToolInvokedEvent("web_search", map[string]interface{}{"query": "golang"}),
		events.NewToolCompletedEvent("web_search", map[string]interface{}{"results": []string{}}, 1250),
		events.NewResponseChunkEvent("Based on my search, here are the best practices..."),
		events.NewCompleteEvent(map[string]interface{}{
			"input_tokens":  50,
			"output_tokens": 200,
			"total_tokens":  250,
		}),
	}

	for _, evt := range sequence {
		log.Printf("[%s] %s", evt.Type, evt.Timestamp)
	}
}
```

---

## File Manifest

### Provider Module Files

```
provider/
├── go.mod                          (module github.com/DojoGenesis/gateway/provider v1.0.0)
├── go.sum
├── README.md                       (setup, usage, examples)
├── LICENSE                         (Apache 2.0)
├── .gitignore
├── interface.go                    (6 lines: ModelProvider interface)
├── types.go                        (75 lines: type definitions)
├── manager.go                      (480 lines: PluginManager)
├── grpc.go                         (380 lines: gRPC transport)
├── rpc.go                          (295 lines: RPC transport + streaming)
├── config.go                       (40 lines: config types)
├── inprocess.go                    (35 lines: InProcessProvider)
├── router.go                       (60 lines: ProviderRouter)
├── pb/
│   ├── provider.proto              (protobuf definitions)
│   ├── provider.pb.go              (generated code)
│   └── provider_grpc.pb.go         (generated gRPC code)
├── examples/
│   ├── openai-plugin/
│   │   ├── main.go                 (150 lines: OpenAI plugin implementation)
│   │   └── go.mod
│   └── anthropic-plugin/
│       ├── main.go                 (150 lines: Anthropic plugin implementation)
│       └── go.mod
├── interface_test.go               (50 lines)
├── types_test.go                   (40 lines)
├── manager_test.go                 (200 lines)
├── grpc_test.go                    (100 lines)
├── router_test.go                  (80 lines)
└── integration_test.go             (150 lines)
```

### Tools Module Files

```
tools/
├── go.mod                          (module github.com/DojoGenesis/gateway/tools v1.0.0)
├── go.sum
├── README.md                       (setup, usage, examples)
├── LICENSE                         (Apache 2.0)
├── .gitignore
├── types.go                        (30 lines: ToolDefinition, ToolFunc, ToolResult)
├── registry.go                     (90 lines: RegisterTool, GetTool, etc.)
├── executor.go                     (80 lines: InvokeTool, InvokeToolWithTimeout)
├── validator.go                    (110 lines: ValidateParameters)
├── context.go                      (40 lines: WithProjectID, GetProjectIDFromContext)
├── examples/
│   ├── web_search.go               (80 lines: web search tool)
│   ├── file_operations.go          (100 lines: read/write files)
│   └── system.go                   (60 lines: system info tool)
├── registry_test.go                (150 lines)
├── executor_test.go                (200 lines)
├── validator_test.go               (120 lines)
├── context_test.go                 (60 lines)
└── integration_test.go             (150 lines)
```

### Events Module Files

```
events/
├── go.mod                          (module github.com/DojoGenesis/gateway/events v1.0.0)
├── go.sum
├── README.md                       (setup, usage, examples)
├── LICENSE                         (Apache 2.0)
├── .gitignore
├── events.go                       (330 lines: StreamEvent, event constructors)
└── events_test.go                  (200 lines)
```

---

## Summary

**Track 1** delivers a solid, production-ready foundation for agentic systems:

1. **Provider Module:** Pluggable, extensible model provider system with hot-reloading
2. **Tools Module:** Thread-safe, context-aware tool registry and execution
3. **Events Module:** Immutable event types for streaming request-response lifecycle

All three modules are **independent**, **well-tested**, **documented**, and **ready for public use**.

**Next Steps (Track 2):**
- Orchestration engine for multi-step agentic workflows
- Memory and context management
- Plan generation and replanning

**Next Steps (Track 3):**
- HTTP server for request handling
- WebSocket streaming
- Authentication and authorization
- Rate limiting and budgeting

---

**Spec Version:** 0.1.0
**Last Updated:** 2026-02-12
**Status:** APPROVED FOR IMPLEMENTATION

---

## 12. Pre-Implementation Checklist

**Instructions:** Before handing this specification to the implementation agent, ensure every item is checked.

### 1. Vision & Goals

- [x] **Clarity of Purpose:** Extracts provider (pluggable LLM system) and tools (registry + execution) as independent Go modules.
- [x] **Measurable Goals:** Provider goals (8 items) and tools goals (8 items) are specific and measurable.
- [x] **Testable Success Criteria:** Binary checklist with coverage targets (>80%), benchmark baselines, and thread-safety verification.
- [x] **Scope is Defined:** Events module included as shared leaf; orchestration (Track 2) and HTTP (Track 3) explicitly out of scope.

### 2. Technical Readiness

- [x] **Architecture is Sound:** Module layout, dependency graph, and design principles are well-defined.
- [x] **Code is Production-Ready:** Complete Go interface definitions, type definitions, and constructor signatures. All code uses `github.com/DojoGenesis/gateway/*` paths.
- [x] **APIs are Specified:** ModelProvider (6 methods), PluginManager, ToolDefinition, Registry, Executor all fully specified.
- [x] N/A **Database Schema is Final:** No database in provider/tools modules.
- [x] **Dependencies are Met:** External deps identified (hashicorp/go-plugin, grpc, protobuf). Events module as standalone leaf.

### 3. Implementation Plan

- [x] **Plan is Actionable:** File manifest with estimated line counts per file. Clear module boundaries.
- [x] **Timeline is Realistic:** Aligned with overall 5-8 week timeline (Phase 1: Core Modules).
- [x] **Testing Strategy is Comprehensive:** Unit tests per file, mock providers, thread-safety tests, integration tests.

### 4. Risk & Quality

- [x] **Risks are Mitigated:** gRPC compatibility, plugin stability, and hot-reload risks addressed with fallback strategies.
- [x] N/A **Rollback Plan is Clear:** New modules; rollback = use monolith directly.
- [x] N/A **Feature Flags are Defined:** Library modules, no feature flags needed.

### 5. Handoff

- [x] **Final Review Complete:** Pre-flight report reviewed; all cross-spec fixes applied (paths, dates, Go version, stability claim).
- [x] **Specification is Final:** Document status marked as APPROVED FOR IMPLEMENTATION.
- [x] **Implementation Ready:** Ready to commission.

### 0. Track 0 — Pre-Commission Alignment

- [x] **Codebase Verified:** plugin/manager.go (PluginManager), tools/ (registry, executor) confirmed in monolith during ingestion.
- [x] **Types Verified:** ModelProvider interface matches codebase's 6-method pattern. ToolDefinition struct matches registry pattern.
- [x] **APIs Verified:** All interface methods match existing codebase signatures (verified during context ingestion).
- [x] **File Structure Verified:** Module paths updated to `github.com/DojoGenesis/gateway/*`. Dockerfile updated to golang:1.24.
- [x] **Remediation Complete:** Date fixed (2026), version claim fixed (0.1.0 pre-release), module paths standardized, Go version corrected.
