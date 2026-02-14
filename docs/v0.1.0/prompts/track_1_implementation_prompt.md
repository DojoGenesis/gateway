# Implementation Commission: Track 1 — Provider + Tools Modules

**Objective:** Implement the production-ready provider module (pluggable LLM provider system with gRPC transport, hot-reload, routing) and tools module (thread-safe registry, executor, validator) as clean, independent Go modules.

**Depends On:** Track 0 (Foundation) must be complete — go.work, shared/, events/ modules exist.

---

## 1. Context & Grounding

**Primary Specification:**
- `docs/agentic-gateway-v0.1.0/track_1_provider_tools_spec.md`

**Foundation Specification:**
- `docs/agentic-gateway-v0.1.0/track_0_foundation_spec.md` (for shared types, dependency graph)

**Pattern Files (Follow these examples from monolith):**
- `go_backend/plugin/manager.go`: PluginManager lifecycle (discovery, loading, health monitoring, hot-reload)
- `go_backend/plugin/grpc.go`: gRPC transport layer for providers
- `go_backend/tools/registry.go`: Thread-safe tool registry with RWMutex
- `go_backend/tools/executor.go`: Tool invocation with context-based timeouts
- `go_backend/events/events.go`: StreamEvent types and constructors

**Module Paths:**
- `github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider`
- `github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools`
- `github.com/TresPies-source/AgenticGatewayByDojoGenesis/events`

---

## 2. Detailed Requirements

### Provider Module

1. **Define `ModelProvider` interface** (6 methods) in `provider/interface.go`:
   - `GetInfo(ctx) (*ProviderInfo, error)`
   - `ListModels(ctx) ([]ModelInfo, error)`
   - `GenerateCompletion(ctx, *CompletionRequest) (*CompletionResponse, error)`
   - `GenerateCompletionStream(ctx, *CompletionRequest) (<-chan *CompletionChunk, error)`
   - `CallTool(ctx, *ToolCallRequest) (*ToolCallResponse, error)`
   - `GenerateEmbedding(ctx, text) ([]float32, error)`

2. **Define all supporting types** in `provider/types.go`: ProviderInfo, ModelInfo, CompletionRequest, CompletionResponse, CompletionChunk, Message, Tool, Usage, ToolCall, ToolCallRequest, ToolCallResponse, ProviderConfig, PluginManagerConfig.

3. **Implement `PluginManager`** in `provider/manager.go`:
   - `NewPluginManager(pluginDir)` and `NewPluginManagerWithConfig(config)`
   - `Start(ctx)` — discover and load plugins
   - `Stop()` — graceful shutdown
   - `GetProvider(name)` — retrieve by name
   - `GetAllProviders()` — list all active
   - `RestartProvider(name)` — force restart
   - Hot-reload: detect binary changes via file watcher
   - Health monitoring: periodic heartbeat, auto-restart on failure (max 3 attempts)
   - Use `sync.RWMutex` for thread-safe provider map

4. **Implement gRPC transport** in `provider/grpc.go`:
   - Use HashiCorp go-plugin for process isolation
   - GRPCServer and GRPCClient implementing ModelProvider over gRPC
   - Proto definitions in `provider/pb/`

5. **Implement `InProcessProvider`** in `provider/inprocess.go`:
   - Adapter for running providers in-process (no gRPC overhead)
   - Implements ModelProvider interface directly

6. **Implement `ProviderRouter`** in `provider/router.go`:
   - Route requests to appropriate provider based on context (default, guest, authenticated)
   - Thread-safe with RWMutex
   - Support setting default provider

### Tools Module

7. **Define `ToolDefinition` struct** in `tools/types.go`:
   - Fields: Name, Description, Parameters (JSON schema), Function (ToolFunc), Timeout
   - `ToolFunc` type: `func(ctx context.Context, input map[string]interface{}) (interface{}, error)`
   - `ToolResult` struct: ToolName, Success, Content, Error

8. **Implement thread-safe registry** in `tools/registry.go`:
   - `RegisterTool(def ToolDefinition)` — register with validation
   - `GetTool(name) (*ToolDefinition, bool)` — retrieve by name
   - `GetAllTools() []ToolDefinition` — list all
   - `UnregisterTool(name) bool` — remove
   - Use `sync.RWMutex` for concurrent access

9. **Implement executor** in `tools/executor.go`:
   - `InvokeTool(ctx, name, input) (*ToolResult, error)` — execute with default timeout
   - `InvokeToolWithTimeout(ctx, name, input, timeout) (*ToolResult, error)` — custom timeout
   - Timeout precedence: explicit > per-tool > env var > default (30s)
   - Context cancellation support

10. **Implement parameter validator** in `tools/validator.go`:
    - Validate input against tool's JSON schema Parameters field
    - Type checking: string, number, integer, boolean, array, object
    - Required field validation
    - Return descriptive error messages

11. **Implement context helpers** in `tools/context.go`:
    - `WithProjectID(ctx, projectID) context.Context`
    - `GetProjectIDFromContext(ctx) (string, bool)`
    - Request-scoped context propagation for tenant isolation

### Events Module

12. **Define `StreamEvent` type** in `events/events.go`:
    - Fields: Type, Data, Timestamp, OrchestrationID (optional)
    - 25+ event constructors for provider, tool, and orchestration events
    - JSON serialization/deserialization
    - Event types include: intent_classified, provider_selected, response_chunk, thinking, tool_invoked, tool_completed, plan_created, node_start, node_end, replanning, complete, failed, error

---

## 3. File Manifest

**Create:**
- `provider/interface.go` — ModelProvider interface
- `provider/types.go` — All type definitions
- `provider/manager.go` — PluginManager
- `provider/grpc.go` — gRPC transport (server + client)
- `provider/rpc.go` — RPC streaming transport
- `provider/config.go` — Configuration types
- `provider/inprocess.go` — InProcessProvider
- `provider/router.go` — ProviderRouter
- `provider/pb/` — Protobuf definitions and generated code
- `provider/*_test.go` — Unit tests for each file
- `tools/types.go` — ToolDefinition, ToolFunc, ToolResult
- `tools/registry.go` — Thread-safe registry
- `tools/executor.go` — Tool invocation with timeouts
- `tools/validator.go` — JSON schema parameter validation
- `tools/context.go` — Context helpers
- `tools/examples/` — Example tool implementations (web_search, file_operations)
- `tools/*_test.go` — Unit tests for each file
- `events/events.go` — StreamEvent types and constructors
- `events/events_test.go` — Serialization tests

---

## 4. Success Criteria

- [ ] `cd provider && go build` succeeds
- [ ] `cd tools && go build` succeeds
- [ ] `cd events && go build` succeeds
- [ ] `go test ./provider/...` passes with >80% coverage
- [ ] `go test ./tools/...` passes with >80% coverage
- [ ] `go test ./events/...` passes
- [ ] `go test -race ./provider/...` detects no data races
- [ ] `go test -race ./tools/...` detects no data races
- [ ] No imports from `go_backend` or `TresPies-source` remain
- [ ] Provider module has zero dependencies on tools, orchestration, memory, or server modules
- [ ] Tools module has zero dependencies on provider, orchestration, memory, or server modules
- [ ] Events module depends only on shared/ and stdlib
- [ ] PluginManager can discover, load, and health-check a mock provider
- [ ] ToolRegistry supports concurrent Register/Get/Unregister without data races
- [ ] Tool invocation respects timeout precedence chain

---

## 5. Constraints & Non-Goals

- **DO NOT** implement HTTP handlers — that's Track 3
- **DO NOT** implement orchestration logic (DAG, replanning) — that's Track 2
- **DO NOT** implement memory management — that's Track 2
- **DO NOT** add database dependencies — provider and tools are in-memory only
- **DO NOT** implement real LLM providers (OpenAI, Anthropic) — use mock implementations for tests
- **DO NOT** change the shared/ types — use them as defined in Track 0

---

## 6. Source Codebase Reference

**Source Files to Extract From:**
- `go_backend/plugin/manager.go` → `provider/manager.go`
- `go_backend/plugin/grpc.go` → `provider/grpc.go`
- `go_backend/plugin/rpc.go` → `provider/rpc.go`
- `go_backend/tools/tools.go` → `tools/types.go` + `tools/registry.go`
- `go_backend/tools/executor.go` → `tools/executor.go`
- `go_backend/events/` → `events/events.go`

**External Dependencies:**
- `github.com/hashicorp/go-plugin` (provider gRPC)
- `github.com/hashicorp/go-hclog` (provider logging)
- `google.golang.org/grpc` (gRPC transport)
- `google.golang.org/protobuf` (protobuf serialization)
