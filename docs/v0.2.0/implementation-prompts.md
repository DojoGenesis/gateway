# Implementation Prompts: Agentic Stack v0.2.0 Commission

**Created:** 2026-02-13
**Specs:** gateway-v0.2.0-api-release.md, gateway-mcp-contract.md, gateway-ada-finalization.md
**Target Agents:** Sonnet-level autonomous agents (Claude Sonnet via Zenflow or Claude Code)
**Orchestration:** Hybrid sequential (numbered) + parallel (lettered)
**Commission Order:** 1 → 2A, 2B, 2C (parallel) → 3
**Estimated Total:** 4 weeks
**Critical Path:** Prompt 1 (1 week) → Prompt 2A (2 weeks) → Prompt 3 (0.5 weeks)

---

## Orchestration Overview

```
Phase 1 (Sequential)
       │
       └─→ Prompt 1: pkg/gateway/ Interfaces
              (1 week)
                  │
                  ▼
Phase 2 (Parallel — NO dependencies between 2A, 2B, 2C)
       │
       ├─→ Prompt 2A: MCP Host Module
       │   (2 weeks, starts after Prompt 1)
       │
       ├─→ Prompt 2B: ADA Disposition Package
       │   (1 week, starts after Prompt 1)
       │
       └─→ Prompt 2C: OTEL + Release Automation
           (1 week, starts after Prompt 1)
                  │
                  │ (all three complete)
                  ▼
Phase 3 (Sequential)
       │
       └─→ Prompt 3: Integration & Wiring
              (0.5 weeks, starts after Phase 2)
```

**Parallelization Key:** Once Prompt 1 completes, assign Prompts 2A, 2B, and 2C to different agents simultaneously. They have NO inter-dependencies — each uses only the outputs from Prompt 1 (the gateway interfaces).

---

## Commission Checklist

- [ ] **Prompt 1 Complete:** `/pkg/gateway/` interfaces and types defined, tests passing
- [ ] **Prompt 2A Complete:** MCP host module functional with tool bridging and OTEL
- [ ] **Prompt 2B Complete:** ADA disposition package with agent initialization
- [ ] **Prompt 2C Complete:** OTEL export infrastructure and release automation tools
- [ ] **Prompt 3 Complete:** All modules wired in main.go, integration tests passing, Docker builds

**Estimated Timeline:**
- Week 1: Prompt 1 completes
- Weeks 2-3: Prompts 2A, 2B, 2C execute simultaneously
- Week 4: Prompt 3 completes and integrates all outputs

---

# Phase 1: Foundation Interfaces (Sequential — Week 1)

> Prompt 1 must complete before any Phase 2 prompts begin.

---

## Prompt 1: pkg/gateway/ Extension Point Interfaces

**Objective:** Create the `pkg/gateway/` package with 4 core interfaces that define the Gateway's extension point contract. These interfaces are the foundation that all Phase 2 components will implement or use.

### 1. Context & Grounding

**Primary Specification:** `specs/gateway-v0.2.0-api-release.md`, Section 3.1

**Existing Pattern Files (read these for style guidance):**
- `tools/types.go` — ToolFunc, ToolDefinition signatures and struct tags
- `tools/registry.go` — Registry pattern with Register, Get, List methods
- `server/trace/span.go` — Context-based patterns for tracing
- `server/config/config.go` — YAML struct tag patterns (json, yaml)
- `main.go` — Dependency injection wiring style

**Read These Files First:**
- `tools/types.go` — Understand ToolFunc signature: `func(context.Context, map[string]interface{}) (map[string]interface{}, error)`
- `tools/registry.go` — Understand how tools are registered and looked up
- `server/trace/logger.go` — Understand context patterns
- `main.go` — Understand DI initialization order

### 2. Detailed Requirements

1. **Create directory and module files:**
   - Create `pkg/gateway/` directory at repository root level
   - All files in this package are within the main module (not a separate go.mod)

2. **Create `pkg/gateway/interfaces.go` with exactly 4 interfaces:**

   a. **ToolRegistry interface** — Wraps tool registration and lookup
      - `Register(ctx context.Context, def ToolDefinition) error` — Register a new tool
      - `Get(ctx context.Context, name string) (*ToolDefinition, error)` — Get tool by exact name
      - `List(ctx context.Context) ([]*ToolDefinition, error)` — List all registered tools
      - `ListByNamespace(ctx context.Context, prefix string) ([]*ToolDefinition, error)` — Filter by namespace prefix (e.g., "composio." returns all composio tools)

   b. **AgentInitializer interface** — Loads agent configuration based on workspace
      - `Initialize(ctx context.Context, workspaceRoot string, activeMode string) (*AgentConfig, error)` — Load agent.yaml, parse, apply mode overrides, return config
      - Signature: workspaceRoot is the absolute path to workspace; activeMode is "" for base config or a mode name (e.g., "debug", "prod") for merging mode overrides

   c. **MemoryStore interface** — Abstracts memory backend (will be implemented in Phase 3)
      - `Store(ctx context.Context, entry *MemoryEntry) error` — Save a memory entry
      - `Search(ctx context.Context, query *SearchQuery, limit int) ([]*MemoryEntry, error)` — Search memory by embedding or text
      - `Get(ctx context.Context, id string) (*MemoryEntry, error)` — Retrieve memory by ID
      - `Delete(ctx context.Context, id string) error` — Delete memory entry

   d. **OrchestrationExecutor interface** — Runs orchestration DAGs
      - `Execute(ctx context.Context, plan *ExecutionPlan) (*ExecutionResult, error)` — Execute a DAG plan, return results
      - `Cancel(ctx context.Context, executionID string) error` — Cancel a running execution by ID

3. **Create `pkg/gateway/types.go` with supporting types:**

   a. **AgentConfig** struct (used by AgentInitializer output)
      ```
      type AgentConfig struct {
          AgentID        string                 `json:"agent_id" yaml:"agent_id"`
          Name           string                 `json:"name" yaml:"name"`
          Mode           string                 `json:"mode" yaml:"mode"`
          ValidationConfig                      `json:"validation,inline"`
          ErrorHandlingConfig                   `json:"error_handling,inline"`
          CollaborationConfig                   `json:"collaboration,inline"`
          ReflectionConfig                      `json:"reflection,inline"`
      }
      ```

   b. **MemoryEntry** struct
      ```
      type MemoryEntry struct {
          ID        string                 `json:"id" yaml:"id"`
          EntryType string                 `json:"type" yaml:"type"` // "conversation", "tool_call", "reflection"
          Content   string                 `json:"content" yaml:"content"`
          Metadata  map[string]interface{} `json:"metadata" yaml:"metadata"`
          CreatedAt time.Time              `json:"created_at" yaml:"created_at"`
          UpdatedAt time.Time              `json:"updated_at" yaml:"updated_at"`
          Embedding []float64              `json:"embedding,omitempty" yaml:"embedding,omitempty"`
      }
      ```

   c. **SearchQuery** struct
      ```
      type SearchQuery struct {
          Text      string `json:"text" yaml:"text"`
          EntryType string `json:"type,omitempty" yaml:"type,omitempty"`
      }
      ```

   d. **ExecutionPlan** struct
      ```
      type ExecutionPlan struct {
          ID        string         `json:"id" yaml:"id"`
          Name      string         `json:"name" yaml:"name"`
          DAG       []*ToolInvocation `json:"dag" yaml:"dag"`
      }
      ```

   e. **ExecutionResult** struct
      ```
      type ExecutionResult struct {
          ExecutionID string                          `json:"execution_id" yaml:"execution_id"`
          Status      string                          `json:"status" yaml:"status"` // "success", "failed", "cancelled"
          Output      map[string]interface{}          `json:"output" yaml:"output"`
          Error       string                          `json:"error,omitempty" yaml:"error,omitempty"`
          Duration    int64                           `json:"duration_ms" yaml:"duration_ms"`
      }
      ```

   f. **ToolInvocation** struct
      ```
      type ToolInvocation struct {
          ID        string                 `json:"id" yaml:"id"`
          ToolName  string                 `json:"tool_name" yaml:"tool_name"`
          Input     map[string]interface{} `json:"input" yaml:"input"`
          DependsOn []string               `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
      }
      ```

   g. **ValidationConfig**, **ErrorHandlingConfig**, **CollaborationConfig**, **ReflectionConfig** — empty structs with json/yaml tags (will be populated in Phase 2B per gateway-ada.md)

4. **Create `pkg/gateway/errors.go` with typed errors:**
   - `ErrToolNotFound` — tool does not exist
   - `ErrAgentInitFailed` — agent initialization failed (wrap underlying error)
   - `ErrMemoryUnavailable` — memory store error
   - `ErrExecutionCancelled` — execution was cancelled
   - `ErrInvalidPlan` — DAG plan is invalid
   - All errors should wrap underlying errors using `fmt.Errorf("%w", err)`

5. **Create `pkg/gateway/gateway_test.go`:**
   - Test that each interface can be implemented (no compilation errors)
   - Test that all types have proper json/yaml struct tags
   - Test error values are correctly typed

6. **Create `pkg/gateway/doc.go` with package documentation:**
   ```go
   // Package gateway defines the extension point interfaces for the Agentic Gateway.
   // These interfaces allow external implementations to be plugged into the Gateway
   // for tool registration, agent initialization, memory storage, and orchestration.
   package gateway
   ```

7. **All exported symbols must have Godoc comments:**
   - Every interface, type, and method must have a comment starting with the symbol name
   - Example: `// ToolRegistry is the interface for managing tool registration...`

8. **Struct tags requirements:**
   - All fields in types must have both `json:""` and `yaml:""` tags
   - Use inline tags where appropriate (e.g., `ValidationConfig` is embedded with `json:"validation,inline"`)
   - Use omitempty for optional fields (e.g., `json:"error,omitempty"`)

### 3. File Manifest

**Create:**
- `pkg/gateway/interfaces.go` — The 4 interfaces
- `pkg/gateway/types.go` — AgentConfig, MemoryEntry, SearchQuery, ExecutionPlan, ExecutionResult, ToolInvocation, plus 4 config structs
- `pkg/gateway/errors.go` — Typed errors
- `pkg/gateway/gateway_test.go` — Interface compliance and type tag tests
- `pkg/gateway/doc.go` — Package documentation

**Do NOT modify any existing files in this prompt.**

### 4. Success Criteria

- [ ] `go build ./pkg/gateway/` compiles without errors
- [ ] `go vet ./pkg/gateway/` passes (no warnings)
- [ ] All 4 interfaces are defined with complete method signatures
- [ ] All supporting types (6 types) are defined with json and yaml struct tags
- [ ] All exported symbols have Godoc comments (interfaces, types, methods, fields)
- [ ] `gateway_test.go` compiles and passes: `go test ./pkg/gateway/ -v`
- [ ] No import of external packages (only stdlib + existing internal imports like tools/)
- [ ] Error handling is done with `fmt.Errorf("%w", err)` pattern for wrapping

### 5. Constraints & Non-Goals

- **DO NOT** implement any interfaces — only define signatures
- **DO NOT** create any HTTP handlers or endpoints
- **DO NOT** modify any existing files (tools/, server/, main.go)
- **DO NOT** add this package to go.work (it's within the main module)
- **DO NOT** import any external dependencies — stdlib only
- **DO NOT** initialize any interfaces in main.go (that's Phase 3)
- **DO NOT** create validation logic for config structs (that's Phase 2B)

### 6. Grounding

- **Module path:** `github.com/DojoGenesis/gateway`
- **Existing ToolFunc signature:** `func(context.Context, map[string]interface{}) (map[string]interface{}, error)` in tools/types.go
- **Existing ToolDefinition shape:** See tools/types.go (has Name, Description, Parameters, Function, Timeout fields)
- **Time package:** `import "time"` for time.Time in MemoryEntry
- **Formatting package:** `import "fmt"` for error wrapping

---

# Phase 2: Parallel Tracks (Weeks 2-3)

> **Parallelization Rule:** All three prompts below can run simultaneously on separate agents. They have ZERO inter-dependencies. They all depend on Prompt 1 being complete.
>
> **Start Conditions:** Begin these prompts only after Prompt 1 is fully complete and merged.

---

## Prompt 2A: MCP Host Module

**Objective:** Create the `mcp/` workspace module that makes the Gateway an MCP host. This module connects to external MCP servers, discovers their tools, and bridges those tools into the Gateway's tool registry.

### 1. Context & Grounding

**Primary Specification:** `specs/gateway-mcp-contract.md`

**Existing Pattern Files (read these for style):**
- `tools/registry.go` — Tool registration patterns
- `tools/types.go` — ToolFunc and ToolDefinition to understand
- `server/config/config.go` — YAML config loading pattern
- `mcp/go.mod` — Will be created as a new workspace module

**Read These Files First:**
- `tools/registry.go` — How tools are registered: `Register(def ToolDefinition) error`, `Get(name string) ToolDefinition`
- `tools/types.go` — ToolFunc signature and ToolDefinition struct
- `pkg/gateway/interfaces.go` — ToolRegistry interface from Prompt 1
- `go.work` — Understand how workspace modules are declared
- `server/config/config.go` — YAML loading pattern using gopkg.in/yaml.v3

**Reference:** `specs/gateway-mcp-contract.md` Section 2.2 for YAML schema and tool namespace prefixing rules.

### 2. Detailed Requirements

1. **Create `mcp/` as a new workspace module:**
   - Create `mcp/` directory at repository root
   - Create `mcp/go.mod` with module path `github.com/DojoGenesis/gateway/mcp`
   - Create `mcp/go.sum` (will populate as you add dependencies)
   - This is a separate module, NOT a package within the main module

2. **Implement YAML configuration parsing (`mcp/config.go`):**
   - Define `MCPConfig` struct that parses from YAML:
     ```
     servers:
       - name: composio
         transport: stdio
         command: python
         args: ["-m", "composio.client"]
         env:
           COMPOSIO_API_KEY: ${COMPOSIO_API_KEY}
         namespace_prefix: "composio."
         tool_allowlist: []          # if empty, allow all
         tool_blocklist: []
         health_check_interval_sec: 30
         timeout_sec: 60
         max_retries: 3
     ```
   - Support environment variable expansion: `${VAR_NAME}` should be replaced with `os.Getenv("VAR_NAME")`
   - All fields must have json and yaml struct tags
   - Validation: namespace_prefix must not be empty; transport must be "stdio" or "sse"

3. **Implement MCPHostManager (`mcp/host.go`):**
   - Constructor: `NewMCPHostManager(cfg *MCPConfig, toolRegistry gateway.ToolRegistry) (*MCPHostManager, error)`
   - Methods:
     - `Start(ctx context.Context) error` — Connect to all servers in config, discover tools, register them
     - `Stop(ctx context.Context) error` — Gracefully close all connections
     - `Status() map[string]ServerStatus` — Return status for each server (connected, tool_count, last_error)
   - Must handle connection errors gracefully (log, continue with other servers)
   - Must implement health check loop (interval from config)
   - Must support reconnection if a server disconnects

4. **Implement MCPToolBridge (`mcp/bridge.go`):**
   - Adapter that converts MCP tool calls to Gateway's ToolFunc signature
   - Function: `AdaptMCPTool(mcpTool *mcp.Tool, serverName string, bridge *MCPServerConnection) gateway.ToolFunc`
   - Returns a `func(context.Context, map[string]interface{}) (map[string]interface{}, error)` that:
     - Calls the MCP tool on the remote server via the connection
     - Maps input/output between formats
     - Emits OTEL spans (if available) with attributes: server_name, tool_name, latency_ms, error (if any)

5. **Implement MCPServerConnection (`mcp/connection.go`):**
   - Manages a single MCP server connection
   - Constructor: `NewMCPServerConnection(name string, cfg ServerConfig) (*MCPServerConnection, error)`
   - Methods:
     - `Connect(ctx context.Context) error` — Establish connection (stdio or sse)
     - `Disconnect(ctx context.Context) error` — Close connection
     - `ListTools(ctx context.Context) ([]*mcp.Tool, error)` — Discover tools from server
     - `CallTool(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error)` — Invoke a tool
     - `IsHealthy() bool` — Check if connection is active
   - Use `github.com/mark3labs/mcp-go` for MCP protocol

6. **OTEL integration (`mcp/bridge.go` + `mcp/otel.go`):**
   - When calling an MCP tool, create an OTEL span named `mcp.tool.call`
   - Span attributes:
     - `mcp.server_name` (string) — name of MCP server
     - `mcp.tool_name` (string) — name of tool invoked
     - `mcp.latency_ms` (int64) — how long the tool call took
     - `mcp.error` (string) — error message if the call failed
   - Only emit OTEL spans if tracer is available (graceful if OTEL not configured)

7. **Environment variable expansion:**
   - In `mcp/config.go`: When parsing YAML, if a value contains `${VAR_NAME}`, replace it with `os.Getenv("VAR_NAME")`
   - If env var not found, use empty string or error (document the behavior)

8. **Graceful degradation:**
   - If an MCP server fails to connect, log the error but continue starting other servers
   - If MCP is not configured (no mcp_servers.yaml), the gateway should start normally without MCP support
   - Tool invocations that fail should not crash the orchestration engine

### 3. File Manifest

**Create:**
- `mcp/go.mod` — Module declaration
- `mcp/go.sum` — Will be auto-generated
- `mcp/config.go` — MCPConfig struct and parsing logic
- `mcp/host.go` — MCPHostManager
- `mcp/bridge.go` — MCPToolBridge and tool adaptation logic
- `mcp/connection.go` — MCPServerConnection for single server management
- `mcp/otel.go` — OTEL span creation helpers
- `mcp/doc.go` — Package documentation
- `mcp/config_test.go` — Config parsing tests (including env var expansion)
- `mcp/host_test.go` — MCPHostManager tests with mock server
- `mcp/bridge_test.go` — Tool bridge adapter tests
- `mcp/testdata/mcp_servers.yaml` — Test fixture with composio example

**Modify:**
- `go.work` — Add `./mcp` to the use directive
- `main.go` — Add MCP host initialization (see Prompt 3 for exact placement)

### 4. Success Criteria

- [ ] `go build ./mcp/` compiles without errors
- [ ] `go test ./mcp/...` passes with >80% coverage
- [ ] YAML config file parses correctly with all field types
- [ ] Environment variables in YAML are expanded before use
- [ ] Mock MCP server tools appear in gateway tool registry with namespace prefix applied
- [ ] Tool invocation through MCPToolBridge returns correct results
- [ ] OTEL spans are emitted for MCP tool calls (attributes present)
- [ ] Graceful handling when MCP server is unreachable (logs, continues)
- [ ] Health check loop runs at configured interval
- [ ] Tool allowlist/blocklist filtering works correctly

### 5. Constraints & Non-Goals

- **DO NOT** modify the existing `tools/` module internals — only call public API (Register, Get, List)
- **DO NOT** implement SSE transport in v0.2.0 — stdio only, mark SSE as TODO comment
- **DO NOT** implement Composio-specific authentication logic beyond env var expansion
- **DO NOT** create any HTTP endpoints — mcp/ module is internal only (no REST API for MCP management in v0.2.0)
- **DO NOT** modify server/, main.go handlers — Phase 3 will do that
- **DO NOT** make MCP a required dependency — must work without mcp_servers.yaml

### 6. Grounding

- **MCP Go library:** `github.com/mark3labs/mcp-go` (version 0.8.0+)
- **YAML:** `gopkg.in/yaml.v3`
- **OTEL:** `go.opentelemetry.io/otel` and `go.opentelemetry.io/otel/trace`
- **Module path:** `github.com/DojoGenesis/gateway/mcp`
- **Workspace:** `go.work` already exists; add `use ./mcp`

---

## Prompt 2B: ADA Disposition Package

**Objective:** Create the `pkg/disposition/` package that parses ADA agent.yaml files and returns DispositionConfig, implementing the gateway-ada contract. This package provides agent initialization based on workspace directory and active mode.

### 1. Context & Grounding

**Primary Specification:** `specs/gateway-ada-finalization.md` and `contracts/gateway-ada.md`

**Existing Pattern Files (read for style):**
- `server/config/config.go` — YAML config loading pattern
- `pkg/gateway/interfaces.go` — AgentInitializer interface to implement

**Read These Files First:**
- `contracts/gateway-ada.md` — Full contract with all type definitions (ValidationConfig, ErrorHandlingConfig, CollaborationConfig, ReflectionConfig, DispositionConfig)
- `pkg/gateway/interfaces.go` — AgentInitializer interface signature
- `server/config/config.go` — YAML unmarshaling and struct tag patterns
- `specs/gateway-ada-finalization.md` — File resolution order and mode merging logic

### 2. Detailed Requirements

1. **Create `pkg/disposition/disposition.go` with all types from contracts/gateway-ada.md:**
   - `DispositionConfig` struct — root struct with all fields from contract
   - `ValidationConfig` struct — validation settings (min_response_length, max_response_length, confidence_threshold, allowed_models, etc.)
   - `ErrorHandlingConfig` struct — error handling (fallback_behavior, retry_limit, timeout_sec, etc.)
   - `CollaborationConfig` struct — multi-agent settings (enabled, max_agents, handoff_protocol, etc.)
   - `ReflectionConfig` struct — agent self-reflection (enabled, frequency, model, etc.)
   - All fields must have json and yaml struct tags with omitempty for optional fields
   - Document every struct and field with Godoc comments

2. **Create `pkg/disposition/resolver.go` with file discovery and loading:**
   - Function: `ResolveDisposition(ctx context.Context, workspaceRoot string, activeMode string) (*DispositionConfig, error)`
   - File resolution order (per contract):
     1. Check environment variable `AGENT_DISPOSITION_PATH` (if set, use that file)
     2. Look for `agent.yaml` in workspaceRoot with pattern: `{agent_id}.agent.yaml` (from bridge file)
     3. Fall back to `agent.yaml` in workspaceRoot
     4. If not found, return DefaultDisposition() with a warning
   - Load and parse YAML into DispositionConfig
   - If activeMode is not empty:
     - Look for mode overrides in the YAML under a `modes:` section
     - Example YAML structure:
       ```yaml
       agent_id: agent_123
       name: MyAgent
       validation:
         min_response_length: 10
       modes:
         debug:
           validation:
             max_response_length: 100
       ```
     - Merge mode overrides: non-nil fields in mode override the base config
     - Implement merge logic that preserves base values for fields not in mode override

3. **Create `pkg/disposition/validator.go` for validation:**
   - Function: `Validate(cfg *DispositionConfig) error`
   - Validate all enum fields (e.g., allowed_models, fallback_behavior) — return error with valid options
   - Validate required fields per contract
   - Validate integer ranges (e.g., max_agents > 0, timeout_sec > 0)
   - Error messages must include: field path (e.g., `validation.confidence_threshold`), invalid value, and valid options
   - Example error: `validation.confidence_threshold: invalid value "1.5" (must be between 0.0 and 1.0)`
   - Call Validate in ResolveDisposition after loading

4. **Create `pkg/disposition/defaults.go`:**
   - Function: `DefaultDisposition() *DispositionConfig`
   - Return a sensible default config (e.g., validation enabled, error handling with 3 retries, no collaboration, reflection off)
   - This is used when agent.yaml is not found or when mode merge leaves fields empty

5. **Create `pkg/disposition/cache.go` for caching:**
   - Simple TTL cache (e.g., 5 minute TTL)
   - Function: `NewDispositionCache(ttl time.Duration) *DispositionCache`
   - Methods:
     - `Get(key string) (*DispositionConfig, bool)` — Return cached value and bool for hit/miss
     - `Set(key string, cfg *DispositionConfig)` — Store in cache with TTL
     - `Clear()` — Clear all cache entries
   - Key format: `"${workspaceRoot}:${activeMode}"` (to distinguish same workspace with different modes)
   - Use in ResolveDisposition: check cache first, then load from disk

6. **Create `pkg/disposition/agent_initializer.go` implementing pkg/gateway.AgentInitializer:**
   - Type: `AgentInitializerImpl struct { cache *DispositionCache }`
   - Constructor: `NewAgentInitializer(cacheTTL time.Duration) *AgentInitializerImpl`
   - Implement: `Initialize(ctx context.Context, workspaceRoot string, activeMode string) (*gateway.AgentConfig, error)`
   - Implementation:
     - Call ResolveDisposition to get DispositionConfig
     - Convert DispositionConfig to gateway.AgentConfig
     - Return AgentConfig
   - Use cache to avoid repeated disk I/O

7. **Performance requirement:**
   - Typical file (<100KB) must parse in <100ms
   - Create a benchmark test to verify: `BenchmarkResolveDisposition`

### 3. File Manifest

**Create:**
- `pkg/disposition/disposition.go` — All config types
- `pkg/disposition/resolver.go` — ResolveDisposition with file discovery
- `pkg/disposition/validator.go` — Validate function
- `pkg/disposition/defaults.go` — DefaultDisposition function
- `pkg/disposition/cache.go` — DispositionCache
- `pkg/disposition/agent_initializer.go` — Implements gateway.AgentInitializer
- `pkg/disposition/doc.go` — Package documentation
- `pkg/disposition/resolver_test.go` — Tests for file resolution, YAML parsing, mode merging
- `pkg/disposition/validator_test.go` — Tests for validation and error messages
- `pkg/disposition/cache_test.go` — Tests for caching behavior
- `pkg/disposition/agent_initializer_test.go` — Tests for AgentInitializer implementation
- `pkg/disposition/testdata/agent-basic.yaml` — Valid agent fixture
- `pkg/disposition/testdata/agent-with-modes.yaml` — Agent with mode overrides
- `pkg/disposition/testdata/agent-invalid.yaml` — Invalid YAML for error testing
- `pkg/disposition/disposition_bench_test.go` — Benchmark for ResolveDisposition

**Do NOT modify any existing files in this prompt** (modification of main.go comes in Phase 3).

### 4. Success Criteria

- [ ] `go build ./pkg/disposition/` compiles without errors
- [ ] `go test ./pkg/disposition/...` passes with >90% coverage
- [ ] All valid YAML fixtures parse correctly
- [ ] Invalid YAML is rejected with descriptive errors (include field path, line number if possible)
- [ ] Mode merging works: base config + mode overrides = correct result
- [ ] DefaultDisposition() returns valid config that passes Validate()
- [ ] Cache returns same result on repeated calls within TTL, reloads after TTL expires
- [ ] Performance <100ms verified by benchmark test
- [ ] AgentInitializer implements gateway.AgentInitializer interface (compiles)
- [ ] All exported types and functions have Godoc comments

### 5. Constraints & Non-Goals

- **DO NOT** modify any existing module code — this is a standalone package
- **DO NOT** create or implement the 7 integration modules (orchestration, memory, etc.) — those come in Phase 3
- **DO NOT** add this to go.work — it's a package within the main module
- **DO NOT** import any LLM-related packages (no openai, anthropic, etc.)
- **DO NOT** initialize agent_initializer in main.go — Phase 3 will do that
- **DO NOT** access the file system in the Initialize method without going through ResolveDisposition

### 6. Grounding

- **YAML parsing:** `gopkg.in/yaml.v3`
- **Time:** `import "time"` for cache TTL
- **Context:** `import "context"`
- **Contract:** `contracts/gateway-ada.md` is canonical source for all type definitions
- **Module path:** `github.com/DojoGenesis/gateway/pkg/disposition`
- **Gateway interfaces:** `github.com/DojoGenesis/gateway/pkg/gateway`

---

## Prompt 2C: OTEL Span Export + Release Automation

**Objective:** Add OpenTelemetry span export to the existing trace system and create release automation tools (Makefile, Goreleaser, docker-compose.example.yml, and contributing guide).

### 1. Context & Grounding

**Primary Specification:** `specs/gateway-v0.2.0-api-release.md`, Sections 3.3-3.5

**Existing Pattern Files (read for style):**
- `server/trace/span.go` — Existing internal span system
- `server/trace/logger.go` — TraceLogger to extend
- `server/trace/storage.go` — Span storage
- `Dockerfile` — Existing Docker build
- `main.go` — Initialization patterns

**Read These Files First:**
- `server/trace/span.go` — Understand Span struct and how spans are created
- `server/trace/logger.go` — TraceLogger.Log() and how spans are stored
- `server/config/config.go` — Configuration patterns
- `Dockerfile` — Build and runtime setup
- `main.go` — Where trace initialization happens

### 2. Detailed Requirements

1. **Add OTEL exporter to `server/trace/otel.go`:**
   - Function: `NewOTELExporter(endpoint string) (trace.SpanExporter, error)`
   - Uses OTLP protocol via gRPC
   - Endpoint format: `localhost:4317` or environment variable `OTEL_EXPORTER_OTLP_ENDPOINT`
   - Must be optional: if endpoint not set, return nil with no error (graceful degradation)
   - Import: `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`

2. **Add LLM-specific span attributes to `server/trace/otel_attributes.go`:**
   - Define constants for attribute keys:
     - `llm.model` (string) — model name (e.g., "claude-opus-4.6")
     - `llm.input_tokens` (int64) — number of input tokens
     - `llm.output_tokens` (int64) — number of output tokens
     - `llm.latency_ms` (int64) — latency in milliseconds
     - `llm.estimated_cost` (float64) — estimated cost in dollars
     - `llm.tool_name` (string) — name of tool if span is from tool invocation
     - `llm.tool_duration_ms` (int64) — tool duration
   - Function: `AddLLMAttributes(span trace.Span, attrs map[string]interface{})` — Helper to add these attributes

3. **Modify `server/trace/logger.go`:**
   - Add OTEL span creation alongside existing internal spans
   - When TraceLogger.Log() is called, also create an OTEL span if exporter is configured
   - OTEL span should include all internal span attributes as OTEL attributes
   - Use context.Context to propagate span context

4. **Add configuration to `server/config/config.go`:**
   - Add `OTELConfig` struct:
     ```go
     type OTELConfig struct {
         Enabled  bool   `json:"enabled" yaml:"enabled"`
         Endpoint string `json:"endpoint" yaml:"endpoint"`
         SamplingRate float64 `json:"sampling_rate" yaml:"sampling_rate"`
         ServiceName string `json:"service_name" yaml:"service_name"`
     }
     ```
   - Add OTELConfig field to main Config struct
   - Values configurable via environment variables:
     - `OTEL_ENABLED` (bool, default false)
     - `OTEL_EXPORTER_OTLP_ENDPOINT` (string)
     - `OTEL_SAMPLING_RATE` (float, 0.0-1.0, default 1.0)
     - `OTEL_SERVICE_NAME` (string, default "agentic-gateway")

5. **Create `server/trace/otel_test.go`:**
   - Test that exporter is created when endpoint is set
   - Test that exporter returns nil when endpoint is empty
   - Test that span export succeeds with mock exporter

6. **Create `docker-compose.example.yml` at repository root:**
   - Services:
     - `gateway` — Builds from Dockerfile, exposes port 8080, sets OTEL env vars
     - `otel-collector` — Image: `otel/opentelemetry-collector-contrib:latest`, port 4317 (gRPC), config from deployments/otel-collector-config.yaml
     - `langfuse` — Image: `langfuse/langfuse:latest`, port 3000, with postgres backend
     - `postgres` — Image: `postgres:15`, port 5432, for Langfuse
   - All services on a single network called `agentic-stack`
   - OTEL Collector receives traces and exports to Langfuse
   - Example:
     ```yaml
     version: '3.8'
     services:
       gateway:
         build: .
         ports:
           - "8080:8080"
         environment:
           OTEL_ENABLED: "true"
           OTEL_EXPORTER_OTLP_ENDPOINT: otel-collector:4317
           OTEL_SERVICE_NAME: agentic-gateway
         depends_on:
           - otel-collector
         networks:
           - agentic-stack
       otel-collector:
         image: otel/opentelemetry-collector-contrib:latest
         ports:
           - "4317:4317"
         volumes:
           - ./deployments/otel-collector-config.yaml:/etc/otel-collector-config.yaml
         command:
           - "--config=/etc/otel-collector-config.yaml"
         networks:
           - agentic-stack
       # ... more services
     networks:
       agentic-stack:
         driver: bridge
     ```

7. **Create `deployments/otel-collector-config.yaml`:**
   - Define OTLP receiver (gRPC on 0.0.0.0:4317)
   - Define Langfuse exporter (HTTP endpoint to langfuse:3000)
   - Create pipeline: otlp receiver → Langfuse exporter
   - Include service name mapping

8. **Create `Makefile` at repository root with targets:**
   - `make build` — Compile binary to `./bin/agentic-gateway`
   - `make test` — Run all tests with `-race` flag
   - `make test-cover` — Run tests with coverage report (`go tool cover -html`)
   - `make vet` — Run `go vet ./...`
   - `make lint` — Run `golangci-lint run` (assumes installed)
   - `make docker` — Build Docker image: `docker build -t agentic-gateway:latest .`
   - `make docker-compose-up` — Start example stack: `docker-compose -f docker-compose.example.yml up -d`
   - `make docker-compose-down` — Stop example stack: `docker-compose -f docker-compose.example.yml down`
   - `make clean` — Remove `./bin` and test artifacts
   - `make generate-openapi` — Run OpenAPI spec generation (command or script)
   - `.PHONY: build test test-cover vet lint docker docker-compose-up docker-compose-down clean generate-openapi`

9. **Create `.goreleaser.yml` at repository root:**
   - Configure for Go binary release (example):
     ```yaml
     project_name: agentic-gateway
     builds:
       - main: ./main.go
         binary: agentic-gateway
         goos:
           - linux
           - darwin
           - windows
         goarch:
           - amd64
           - arm64
     archives:
       - format: tar.gz
         format_overrides:
           - goos: windows
             format: zip
     release:
       github:
         owner: TresPies-source
         name: AgenticGatewayByDojoGenesis
     ```
   - Note: ADR-006 states no automatic tag push — operator must manually tag and push

10. **Create `CONTRIBUTING.md` at repository root:**
    - **Development Setup:**
      - Go version requirement (1.22+)
      - Clone and `go mod download`
      - Run `make build` to compile
    - **Testing:**
      - Run `make test` before submitting PR
      - Aim for >80% coverage
      - Use table-driven tests for complex logic
    - **Code Style:**
      - Follow Go conventions (gofmt, go vet)
      - Godoc comments on exported symbols
      - Error wrapping: `fmt.Errorf("%w", err)`
    - **Docker:**
      - Build with `make docker`
      - Test with `docker-compose.example.yml`
    - **Git Workflow:**
      - Create feature branch from main
      - Commit messages: start with verb (Add, Fix, Refactor, etc.)
      - Push and open PR with description
      - PR must pass CI (tests, vet, lint)
    - **Release Process:**
      - Manual tag: `git tag v0.2.0`
      - Manual push: `git push origin v0.2.0`
      - Goreleaser runs on tag push

### 3. File Manifest

**Create:**
- `server/trace/otel.go` — OTEL exporter setup
- `server/trace/otel_attributes.go` — LLM-specific span attributes and helper
- `server/trace/otel_test.go` — OTEL exporter tests
- `docker-compose.example.yml` — Complete example stack
- `deployments/otel-collector-config.yaml` — OTEL Collector configuration
- `Makefile` — Build and development targets
- `.goreleaser.yml` — Goreleaser configuration
- `CONTRIBUTING.md` — Contributing guide

**Modify:**
- `server/trace/logger.go` — Add OTEL span creation
- `server/config/config.go` — Add OTELConfig section
- `main.go` — Add OTEL initialization (see Prompt 3 for exact placement)

### 4. Success Criteria

- [ ] `make build` compiles binary successfully
- [ ] `make test` passes all tests with no failures
- [ ] `make vet` and `make lint` pass
- [ ] `make docker` builds Docker image without errors
- [ ] OTEL spans are emitted when `OTEL_ENABLED=true` and `OTEL_EXPORTER_OTLP_ENDPOINT` is set
- [ ] OTEL spans include LLM-specific attributes (model, tokens, latency)
- [ ] `docker-compose -f docker-compose.example.yml config` validates (yaml is correct)
- [ ] `docker-compose -f docker-compose.example.yml up` starts all services successfully
- [ ] `goreleaser check` validates the .goreleaser.yml config
- [ ] CONTRIBUTING.md covers setup, testing, code style, and release process

### 5. Constraints & Non-Goals

- **DO NOT** push any git tags automatically (ADR-006 requires manual tag creation)
- **DO NOT** modify existing span behavior — OTEL export is purely additive
- **DO NOT** make OTEL a required dependency — must be opt-in via OTEL_ENABLED env var
- **DO NOT** include secrets or real API keys in docker-compose example
- **DO NOT** modify any existing handlers or logic — only trace infrastructure
- **DO NOT** create actual releases — Makefile and Goreleaser are templates ready for use

### 6. Grounding

- **OTEL SDK:** `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/sdk/trace`
- **OTLP Exporter:** `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
- **OTEL Collector image:** `otel/opentelemetry-collector-contrib:latest`
- **Langfuse image:** `langfuse/langfuse:latest`
- **Goreleaser docs:** Standard Go project configuration
- **Docker Compose:** Version 3.8+ syntax

---

# Phase 3: Integration & Documentation (Sequential — Week 4)

> Prompt 3 begins only after ALL Phase 2 prompts (2A, 2B, 2C) are complete.

---

## Prompt 3: Integration Testing + Module Wiring + Documentation

**Objective:** Wire all Phase 2 outputs together in main.go and throughout the codebase, create comprehensive cross-module integration tests, implement real handlers for /v1/gateway/ routes, and finalize documentation with OpenAPI specs.

### 1. Context & Grounding

**Primary Specifications:** All three specs (v0.2.0-api-release.md, gateway-mcp-contract.md, gateway-ada-finalization.md)

**Existing Pattern Files (read for style):**
- `main.go` — DI wiring structure to extend
- `server/router.go` — Route registration pattern
- `server/orchestration/engine.go` — Orchestration logic to integrate with disposition

**Read These Files First:**
- `main.go` — Current DI chain and initialization order
- `pkg/gateway/interfaces.go` — Interfaces from Prompt 1
- `mcp/host.go` — MCPHostManager from Prompt 2A
- `pkg/disposition/agent_initializer.go` — AgentInitializer from Prompt 2B
- `server/trace/otel.go` — OTEL setup from Prompt 2C
- `server/orchestration/engine.go` — Orchestration engine to extend
- `memory/compression.go` — Memory compression logic to extend
- `server/router.go` — Existing HTTP routes

### 2. Detailed Requirements

1. **Update `main.go` DI chain with new components:**
   - Order matters! Initialize in this sequence:
     1. Load configuration (existing)
     2. Initialize OTEL exporter (new, from Phase 2C) — conditional on OTEL_ENABLED
     3. Initialize trace logger with OTEL span export (new, from Phase 2C)
     4. Initialize tool registry (existing)
     5. Initialize MCP host manager (new, from Phase 2A) — after tool registry, before server start
     6. Initialize disposition cache and agent initializer (new, from Phase 2B)
     7. Initialize orchestration executor (new, implementing gateway.OrchestrationExecutor)
     8. Initialize memory store (new, implementing gateway.MemoryStore)
     9. Initialize HTTP server with all handlers (existing)
   - Wiring example snippet:
     ```go
     // OTEL
     otelExp, err := otel.NewOTELExporter(cfg.OTEL.Endpoint)
     if err != nil && cfg.OTEL.Enabled {
         return fmt.Errorf("OTEL exporter: %w", err)
     }

     // Tool registry
     toolReg := tools.NewRegistry()

     // MCP host
     mcpMgr, err := mcp.NewMCPHostManager(cfg.MCP, toolReg)
     if err != nil {
         return fmt.Errorf("MCP host: %w", err)
     }
     if err := mcpMgr.Start(ctx); err != nil {
         return fmt.Errorf("MCP start: %w", err)
     }

     // Disposition
     dispInit := disposition.NewAgentInitializer(5 * time.Minute)

     // (more initialization...)
     ```

2. **Modify `server/orchestration/engine.go` to be disposition-aware:**
   - Add field: `disposition *gateway.AgentConfig` to the orchestration engine
   - Before executing tool calls, check `disposition.Pacing` settings
   - If `disposition.Pacing.Enabled` is true and `disposition.Pacing.InterToolDelayMs` is set:
     - Insert a sleep between tool invocations: `time.Sleep(time.Duration(disposition.Pacing.InterToolDelayMs) * time.Millisecond)`
   - This allows agents to control execution pacing via their disposition file

3. **Modify `memory/compression.go` to be disposition-aware:**
   - Add function: `ShouldCompress(disposition *gateway.AgentConfig) bool`
   - Check `disposition.Depth.MemoryCompression`:
     - If enabled, compress memory entries according to `compression_strategy` (e.g., "semantic", "temporal")
     - Respect `retention_days` setting (don't compress entries newer than this)
   - This allows agents to control memory retention via disposition

4. **Implement real handlers in `server/handle_gateway.go`:**
   - `POST /v1/gateway/agents` — Create agent (receive disposition config, store, return agent ID)
   - `GET /v1/gateway/agents/:id` — Get agent status and current disposition
   - `POST /v1/gateway/agents/:id/chat` — Chat with agent (use orchestration executor)
   - `GET /v1/gateway/tools` — List all registered tools with namespace info and MCP server origin
   - `POST /v1/gateway/orchestrate` — Submit orchestration plan, return execution ID
   - `GET /v1/gateway/orchestrate/:id/dag` — Get DAG structure and execution status
   - `GET /v1/gateway/traces/:id` — Get trace details (if OTEL enabled)
   - All handlers must:
     - Accept context.Context from request
     - Return JSON responses with proper HTTP status codes
     - Emit OTEL spans with appropriate attributes
     - Use the wired interfaces (ToolRegistry, AgentInitializer, MemoryStore, OrchestrationExecutor)

5. **Implement real handlers in `server/handle_admin.go`:**
   - `GET /admin/health` — Return detailed health: memory available, tool count, MCP connections, OTEL status
   - `GET /admin/config` — Return current configuration (sanitized: no secrets, no keys)
   - `POST /admin/config/reload` — Reload YAML config from disk
   - `GET /admin/metrics/prometheus` — Return Prometheus-format metrics (tool call count, execution duration, memory usage)
   - `GET /admin/mcp/servers` — List MCP server connections, their status, tool counts
   - All handlers must require authentication via middleware

6. **Update `server/handle_gateway_test.go` with integration tests:**
   - Test: Create agent with disposition → list agents → agent has correct disposition
   - Test: POST to /v1/gateway/tools returns all tools including MCP tools with namespace prefix
   - Test: POST to /v1/gateway/orchestrate with valid plan → returns execution ID
   - Test: GET /v1/gateway/orchestrate/:id/dag returns DAG structure
   - Test: OTEL spans are emitted for handler execution

7. **Update `server/handle_admin_test.go` with admin tests:**
   - Test: GET /admin/health returns correct JSON shape
   - Test: GET /admin/mcp/servers lists all connected MCP servers
   - Test: POST /admin/config/reload reloads config

8. **Create `integration_test.go` at repository root with cross-module tests:**
   - Test: Load agent YAML → disposition parse → agent init → orchestration with pacing → memory compression respects disposition
   - Test: MCP server config → tool registration → tool invocation via orchestration → OTEL span emitted
   - Test: All 3 HTTP layers (v1 frozen, v1/gateway new, admin new) respond correctly
   - Test: Graceful degradation when MCP server unreachable
   - Test: Graceful degradation when OTEL not configured

9. **Generate OpenAPI spec:**
   - Use swag or similar tool to generate OpenAPI 3.0 spec from Gin routes
   - Output: `docs/openapi.yaml`
   - Include all /v1/gateway/ and /admin/ routes
   - Freeze /v1/ routes as-is

10. **Add comprehensive Godoc comments:**
    - Review all new packages (mcp, disposition, gateway, trace/otel)
    - Ensure every exported function, type, and method has a comment
    - Include examples where helpful

### 3. File Manifest

**Modify (Major):**
- `main.go` — Complete DI wiring for all Phase 2 outputs
- `server/orchestration/engine.go` — Add disposition-aware pacing
- `memory/compression.go` — Add disposition-aware retention logic
- `server/handle_gateway.go` — Implement real handlers (replace 501 stubs)
- `server/handle_admin.go` — Implement real handlers (replace 501 stubs)
- `server/router.go` — Minor adjustments if needed (route guards, etc.)
- `server/config/config.go` — Add MCP and OTEL config sections if not done in Phase 2

**Create:**
- `integration_test.go` — Cross-module integration tests
- `docs/openapi.yaml` — Generated OpenAPI spec

**Create (Test fixtures):**
- Test fixtures as needed for integration tests (agent YAML files, MCP configs, etc.)

### 4. Success Criteria

- [ ] `go build .` compiles all modules without errors
- [ ] `go test ./...` passes all tests including new integration tests
- [ ] `make build`, `make test`, `make docker` all succeed
- [ ] main.go initializes MCP host, disposition, and OTEL without runtime errors
- [ ] Orchestration respects disposition.Pacing.InterToolDelayMs (sleep happens)
- [ ] Memory respects disposition.Depth.MemoryCompression (compression applied)
- [ ] POST /v1/gateway/agents returns 200 with agent ID
- [ ] GET /v1/gateway/tools returns 200 with all tools (including MCP tools with namespace prefix)
- [ ] POST /v1/gateway/orchestrate returns 200 with execution ID
- [ ] GET /admin/health returns 200 with health details
- [ ] OTEL spans are emitted for handler execution (timestamps, attributes present)
- [ ] docker-compose up starts all services and gateway serves requests
- [ ] OpenAPI spec is generated and validates
- [ ] All exported symbols have Godoc comments

### 5. Constraints & Non-Goals

- **DO NOT** implement all stub handlers if there's no backing logic — only real implementations
- **DO NOT** push any git tags — that's manual operator work per ADR-006
- **DO NOT** break any existing /v1/ endpoint contracts — they are frozen
- **DO NOT** make MCP or OTEL required — graceful degradation when not configured
- **DO NOT** modify Dockerfile or docker-compose.example.yml after Phase 2C (they're done)
- **DO NOT** add new dependencies not already in go.mod from Phase 2 modules

### 6. Grounding

- **All three specs:** in `specs/` directory
- **All ADRs:** in `decisions/` directory
- **contracts/gateway-ada.md:** Canonical source for disposition integration
- **Existing modules:** tools/, server/, memory/, tracing/, etc.
- **New modules from Phase 2:** mcp/, pkg/disposition/, pkg/gateway/, server/trace/otel

---

## Summary & Next Steps

After all three prompts complete:

1. **Code Review:** Review all commits for correctness, test coverage, and adherence to Go style
2. **Documentation Review:** Verify CONTRIBUTING.md, OpenAPI spec, and Godoc comments are complete
3. **Smoke Test:** Run `docker-compose -f docker-compose.example.yml up` and make a test request to the gateway
4. **Release Preparation (Manual):**
   - Create git tag: `git tag v0.2.0`
   - Push tag: `git push origin v0.2.0`
   - Goreleaser will automatically build and publish binaries
5. **Deployment:** Use docker-compose.example.yml as template for production deployments

---

**Commission Status:** Ready to dispatch to agents. All prompts are self-contained and executable independently (with appropriate sequencing).
