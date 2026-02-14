# AgenticGateway v0.2.0: The API Release

**Author:** Cruz + Claude (Cowork)
**Status:** Final
**Created:** 2026-02-13
**Grounded In:** ADR-001, ADR-005, ADR-006, v0.1.0 retrospective, v0.2.0 planning notes
**Target Release:** Q1 2026 (3-4 weeks from spec finalization)

---

## 1. Vision

AgenticGateway v0.2.0 transforms the server from a closed monolith into an **extensible platform**. The API surface is the product. By exposing clean Go interfaces (`pkg/gateway/`) behind multiple HTTP layers, we enable:

1. **Organizational independence:** Teams define their own tools, memory stores, and orchestration logic via interfaces, not fork the codebase
2. **Observability by default:** OTEL span export gives operators the visibility to debug agentic workflows in production
3. **Clear architectural boundaries:** Frozen OpenAI-compatible layer (`/v1/`), new agentic layer (`/v1/gateway/`), operator control plane (`/admin/`)
4. **Release readiness:** Automated builds (Goreleaser), comprehensive documentation (OpenAPI, CONTRIBUTING.md), Godoc completeness

This release does NOT extract orchestration (that is v0.3.0). Instead, it provides the **interfaces and observability** that make extraction safe.

---

## 2. Goals & Success Criteria

### Goals

| Goal | Rationale |
|------|-----------|
| **Define 4 core Go interfaces** | Teams implement ToolRegistry, MemoryStore, AgentInitializer, OrchestrationExecutor without modifying server code |
| **Three-layer HTTP surface** | Separate frozen OpenAI-compat from new agentic routes; add operator control plane |
| **OTEL span export** | Operators see model invocations, token counts, latency, estimated cost in tracing backend (Langfuse, Jaeger, etc.) |
| **Automated release pipeline** | Makefile + Goreleaser produce binaries, container images, checksums; no manual steps |
| **Developer onboarding** | OpenAPI spec, CONTRIBUTING.md, 100% Godoc coverage on exported types; new team member can add a tool in <30 min |
| **Production docker-compose** | Example stack with Gateway + OTEL Collector + Langfuse; operators can `docker-compose up` and see full workflow |

### Success Criteria

- [ ] All 4 interfaces in `pkg/gateway/` are exported, documented, and have at least one in-repo implementation
- [ ] `/v1/chat/completions` endpoint unchanged; `/v1/gateway/*` new routes documented and tested
- [ ] `server/trace/` exports OTEL metrics for every LLM invocation (model, input_tokens, output_tokens, latency_ms, estimated_cost_usd)
- [ ] `make release` produces binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 with checksums
- [ ] docker-compose.example.yml is tested and boots with zero manual steps
- [ ] CONTRIBUTING.md + tool walkthrough shows new contributor path from fork → local build → custom tool → PR
- [ ] Go test coverage on new `pkg/gateway/` ≥ 80%; no regressions in v0.1.0 test suite

### Non-Goals

- Extracting server/orchestration/ into standalone module (deferred to v0.3.0)
- Breaking changes to v0.1.0 routes or existing tool registry format
- GraphQL layer (HTTP + gRPC are the surfaces for v0.2.0)
- Persistent audit logging in the gateway itself (operators implement via OTEL sink)
- First public git tag (tag waits for orchestration extraction)

---

## 3. Technical Architecture

### 3.1 pkg/gateway/ — Extension Point Interfaces

The heart of v0.2.0. These 4 interfaces define what it means to extend the gateway **without forking**.

#### 3.1.1 ToolRegistry Interface

```go
// File: pkg/gateway/registry.go
package gateway

import (
    "context"
)

// ToolFunc is the execution signature for all tools.
// Context carries cancellation, timeout, and trace span.
// Input is a map of parameter names to values; output is result map or error.
type ToolFunc func(context.Context, map[string]interface{}) (map[string]interface{}, error)

// ToolDefinition describes a callable tool.
type ToolDefinition struct {
    // Name is the unique identifier for this tool (e.g., "web_search", "send_email").
    Name string

    // Description is human-readable documentation of what the tool does.
    Description string

    // Parameters maps parameter name to JSON Schema for input validation.
    // Example: map[string]interface{}{
    //   "query": map[string]interface{}{
    //     "type": "string",
    //     "description": "search query",
    //   },
    // }
    Parameters map[string]interface{}

    // Function is the tool implementation.
    Function ToolFunc

    // Timeout is the hard deadline for tool execution (e.g., 30 seconds for API calls).
    // Zero means no timeout.
    Timeout time.Duration
}

// ToolRegistry is the primary extension point for tool implementations.
// Implementations must be thread-safe and may cache definitions.
//
// Example: A team implements ToolRegistry to expose custom tools for their domain
// (e.g., CRM lookups, internal APIs) without modifying the gateway codebase.
type ToolRegistry interface {
    // RegisterTool adds a tool definition to the registry.
    // Returns error if a tool with the same Name already exists.
    RegisterTool(def *ToolDefinition) error

    // GetTool retrieves a tool definition by name.
    // Returns nil, false if not found (not an error).
    GetTool(name string) (*ToolDefinition, bool)

    // GetAllTools returns all registered tool definitions.
    // Slice is a copy and safe to mutate.
    GetAllTools() []*ToolDefinition

    // UnregisterTool removes a tool from the registry.
    // Returns false if the tool was not found.
    UnregisterTool(name string) bool

    // ListTools returns a filtered list of tools; used by API for tool browsing.
    // Filters may include category, owner, enabled status.
    ListTools(ctx context.Context, filters map[string]string) ([]*ToolDefinition, error)
}
```

**Rationale:** Replaces the current global `tools/registry.go` map. Enables:
- Multiple registry implementations (in-memory, database, remote gRPC)
- Tool lifecycle management (enable/disable, versioning)
- Audit and access control per implementation

**Usage in v0.2.0:** Core implementation is `tools/standard_registry.go` wrapping the existing v0.1.0 map. DI wiring in main.go binds the interface to this implementation.

---

#### 3.1.2 MemoryStore Interface

```go
// File: pkg/gateway/memory.go
package gateway

// MemoryStore is the abstraction for conversation and semantic memory.
// Implementations may range from in-memory caches to vector databases.
type MemoryStore interface {
    // StoreConversationTurn records a single exchange (user input + agent output).
    // sessionID groups related turns; turnID is a unique identifier within that session.
    // Returns error if storage fails (e.g., database unavailable).
    StoreConversationTurn(ctx context.Context, sessionID, turnID string, turn *ConversationTurn) error

    // GetConversationHistory retrieves all turns for a session, in temporal order.
    // limit ≤ 0 means no limit. Implementations should respect context cancellation.
    GetConversationHistory(ctx context.Context, sessionID string, limit int) ([]*ConversationTurn, error)

    // StoreSemanticMemory persists named embeddings or facts for semantic retrieval.
    // category groups related memories (e.g., "user_preferences", "facts").
    StoreSemanticMemory(ctx context.Context, sessionID, category, key string, value interface{}, metadata map[string]string) error

    // QuerySemanticMemory retrieves memories matching a query, optionally ranked by similarity.
    // Returns at most limit results.
    QuerySemanticMemory(ctx context.Context, sessionID, category string, query string, limit int) ([]*SemanticMemory, error)

    // DeleteSession removes all memory for a session (cleanup on expiry or user request).
    DeleteSession(ctx context.Context, sessionID string) error

    // GetSessionMetadata returns creation time, last access, and custom tags for a session.
    GetSessionMetadata(ctx context.Context, sessionID string) (*SessionMetadata, error)
}

// ConversationTurn represents a single user-input → agent-output cycle.
type ConversationTurn struct {
    // ID is a unique identifier for this turn (e.g., UUID or monotonic counter).
    ID string

    // Timestamp is when the turn was recorded (UTC).
    Timestamp time.Time

    // UserInput is the raw user message.
    UserInput string

    // AgentOutput is the final response to the user.
    AgentOutput string

    // ToolCalls is a list of tool invocations during this turn.
    ToolCalls []*ToolInvocation

    // Metadata is arbitrary key-value context (e.g., model used, temperature).
    Metadata map[string]interface{}
}

// ToolInvocation represents a single tool call within a turn.
type ToolInvocation struct {
    ToolName  string                 `json:"tool_name"`
    Input     map[string]interface{} `json:"input"`
    Output    map[string]interface{} `json:"output"`
    Duration  time.Duration          `json:"duration"`
    Error     string                 `json:"error,omitempty"` // non-empty if tool failed
}

// SemanticMemory is a named fact or embedding in the store.
type SemanticMemory struct {
    Key       string                 `json:"key"`
    Value     interface{}            `json:"value"`
    Metadata  map[string]string      `json:"metadata"`
    Timestamp time.Time              `json:"timestamp"`
    Similarity float64               `json:"similarity,omitempty"` // 0-1, populated by QuerySemanticMemory
}

// SessionMetadata tracks session lifecycle.
type SessionMetadata struct {
    SessionID   string            `json:"session_id"`
    CreatedAt   time.Time         `json:"created_at"`
    LastAccessAt time.Time         `json:"last_access_at"`
    Tags        map[string]string `json:"tags"`
    TTLSeconds  int64             `json:"ttl_seconds"`
}
```

**Rationale:** Separates conversation history from semantic memory (embeddings, facts). Enables:
- Pluggable backends: PostgreSQL + pgvector, Pinecone, Weaviate, in-memory
- Lifecycle management: TTL, archival, deletion
- Session isolation and audit

**Usage in v0.2.0:** Core implementation is `memory/standard_store.go` wrapping v0.1.0 in-memory logic. DI binds interface to implementation.

---

#### 3.1.3 AgentInitializer Interface

```go
// File: pkg/gateway/agent.go
package gateway

// AgentInitializer constructs and configures an Agent instance.
// Implementations encapsulate model selection, prompt engineering, and parameter tuning.
type AgentInitializer interface {
    // InitializeAgent creates a ready-to-run Agent with model, tools, and system prompt.
    // config contains user-facing options: model name, temperature, max_tokens, system_prompt_override.
    InitializeAgent(ctx context.Context, config *AgentConfig) (*Agent, error)

    // GetSupportedModels returns the list of available LLM models.
    GetSupportedModels(ctx context.Context) ([]*ModelInfo, error)

    // ValidateAgentConfig checks if a config is valid before initialization.
    ValidateAgentConfig(ctx context.Context, config *AgentConfig) error
}

// AgentConfig is the request payload for agent initialization.
type AgentConfig struct {
    // Model is the LLM model identifier (e.g., "gpt-4", "claude-3-sonnet").
    Model string `json:"model"`

    // Temperature controls randomness (0 = deterministic, 1 = very random).
    Temperature float32 `json:"temperature"`

    // MaxTokens is the upper limit on response length.
    MaxTokens int `json:"max_tokens"`

    // SystemPrompt overrides the default system instruction.
    SystemPrompt string `json:"system_prompt,omitempty"`

    // IncludedTools is a whitelist of tool names to enable for this agent.
    // If empty, all tools are available.
    IncludedTools []string `json:"included_tools,omitempty"`

    // Metadata is arbitrary context (e.g., user ID, session ID, request source).
    Metadata map[string]string `json:"metadata,omitempty"`
}

// Agent is a stateful orchestrator for a single conversation.
type Agent struct {
    // ID is a unique identifier for this agent instance.
    ID string

    // Model is the LLM provider and model name.
    Model string

    // SystemPrompt is the base instruction for the LLM.
    SystemPrompt string

    // Config is the full initialization config.
    Config *AgentConfig

    // ToolRegistry is the set of available tools.
    ToolRegistry ToolRegistry

    // MemoryStore is the conversation and semantic memory backend.
    MemoryStore MemoryStore

    // ExecutionContext is the orchestration state (conversation history, decision tree, etc.).
    ExecutionContext *ExecutionContext
}

// ExecutionContext tracks an ongoing agentic conversation.
type ExecutionContext struct {
    // SessionID groups related turns.
    SessionID string

    // ConversationHistory is the transcript so far.
    ConversationHistory []*ConversationTurn

    // CurrentTurn is the in-flight user input being processed.
    CurrentTurn *ConversationTurn

    // DecisionTree tracks the reasoning steps (optional, for explainability).
    DecisionTree map[string]interface{}

    // Metadata is arbitrary state (e.g., context windows, cost tracking).
    Metadata map[string]interface{}
}

// ModelInfo describes an available LLM.
type ModelInfo struct {
    ID             string  `json:"id"`
    Name           string  `json:"name"`
    Provider       string  `json:"provider"` // "openai", "anthropic", etc.
    ContextWindow  int     `json:"context_window"`
    InputTokenCost  float64 `json:"input_token_cost"` // per 1M tokens
    OutputTokenCost float64 `json:"output_token_cost"`
}
```

**Rationale:** Separates agent construction from orchestration. Enables:
- Model selection logic
- Prompt engineering templates
- Cost tracking and quota enforcement
- Multi-tenant configuration

**Usage in v0.2.0:** Core implementation is `server/agent/initializer.go` wrapping v0.1.0 agent setup. DI wires it.

---

#### 3.1.4 OrchestrationExecutor Interface

```go
// File: pkg/gateway/orchestration.go
package gateway

// OrchestrationExecutor is the planning and execution loop.
// It reads the conversation, decides tool calls, and drives the agent to completion.
type OrchestrationExecutor interface {
    // Execute runs a single agentic step: assess state, call tools, update context.
    // Returns the next action (tool call, user prompt, or completion).
    Execute(ctx context.Context, agent *Agent, sessionID string) (*ExecutionStep, error)

    // CanContinue checks if the agent should take another step (e.g., max steps reached).
    CanContinue(agent *Agent, steps int) bool

    // GetExecutionStats returns cost, latency, and tool call counts for a session.
    GetExecutionStats(ctx context.Context, sessionID string) (*ExecutionStats, error)
}

// ExecutionStep is the output of one Executor.Execute call.
type ExecutionStep struct {
    // Type is one of: TOOL_CALL, USER_PROMPT, COMPLETION, ERROR.
    Type ExecutionStepType `json:"type"`

    // ToolCall is populated if Type == TOOL_CALL.
    ToolCall *PendingToolCall `json:"tool_call,omitempty"`

    // UserPrompt is populated if Type == USER_PROMPT (agent needs clarification).
    UserPrompt string `json:"user_prompt,omitempty"`

    // Completion is populated if Type == COMPLETION (agent is done).
    Completion string `json:"completion,omitempty"`

    // Error is populated if Type == ERROR.
    Error string `json:"error,omitempty"`

    // Reasoning is the agent's internal thought process (for explainability).
    Reasoning string `json:"reasoning,omitempty"`

    // Metadata is arbitrary step-level context.
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionStepType enumerates the types of orchestration steps.
type ExecutionStepType string

const (
    StepTypeToolCall    ExecutionStepType = "TOOL_CALL"
    StepTypeUserPrompt  ExecutionStepType = "USER_PROMPT"
    StepTypeCompletion  ExecutionStepType = "COMPLETION"
    StepTypeError       ExecutionStepType = "ERROR"
)

// PendingToolCall represents a single tool invocation to execute.
type PendingToolCall struct {
    ID       string                 `json:"id"`
    ToolName string                 `json:"tool_name"`
    Input    map[string]interface{} `json:"input"`
}

// ExecutionStats aggregates metrics for a session.
type ExecutionStats struct {
    SessionID           string        `json:"session_id"`
    TotalSteps          int           `json:"total_steps"`
    ToolCallsCompleted  int           `json:"tool_calls_completed"`
    ToolCallsFailed     int           `json:"tool_calls_failed"`
    TotalLatencyMs      int64         `json:"total_latency_ms"`
    EstimatedCostUSD    float64       `json:"estimated_cost_usd"`
    TokensInput         int           `json:"tokens_input"`
    TokensOutput        int           `json:"tokens_output"`
}
```

**Rationale:** Separates planning and execution from HTTP routing. Enables:
- Pluggable orchestration strategies (ReAct, Langchain-style, custom)
- Extraction to standalone module in v0.3.0
- Testing agentic logic in isolation

**Usage in v0.2.0:** Core implementation is `server/orchestration/executor.go` wrapping v0.1.0 agent loop. DI wires it.

---

### 3.2 Three-Layer HTTP Surface

The gateway exposes three distinct HTTP layers, each with different stability and audience guarantees.

#### 3.2.1 Layer 1: `/v1/*` — Frozen OpenAI-Compatible Surface

These routes are **locked for backward compatibility**. v0.1.0 clients must work unchanged.

| Method | Path | Request | Response | Purpose | Status |
|--------|------|---------|----------|---------|--------|
| POST | `/v1/chat/completions` | `ChatCompletionRequest` (OpenAI format) | `ChatCompletionResponse` | LLM invocation, tool-use fallback | Unchanged from v0.1.0 |
| GET | `/v1/models` | — | `[{id, object, owned_by, created}]` | List available models | Unchanged |
| GET | `/v1/health` | — | `{status, uptime_seconds, version}` | Liveness check | Unchanged |

**Implementation:** Routes are served by `server/routes.go` middleware group under `gin.Group("/v1")`. No changes to request/response schemas.

**Observability:** All requests are traced via `server/trace/` span propagation (see section 3.3). Tracing is **transparent** to clients; no new headers required.

---

#### 3.2.2 Layer 2: `/v1/gateway/*` — Agentic API Surface

New routes that expose `pkg/gateway/` interfaces. These routes will evolve through v0.2.x and may have breaking changes in v0.3.0 during orchestration extraction.

| Method | Path | Request | Response | Purpose |
|--------|------|---------|----------|---------|
| POST | `/v1/gateway/agents/init` | `AgentConfig` | `{agent_id, model, system_prompt, tools}` | Create stateful agent instance |
| POST | `/v1/gateway/agents/{agent_id}/execute` | `{user_input, context}` | `ExecutionStep` | Single agentic step (plan, tool call, or completion) |
| GET | `/v1/gateway/agents/{agent_id}/history` | `{limit, offset}` | `[ConversationTurn]` | Fetch conversation history |
| POST | `/v1/gateway/agents/{agent_id}/memory/semantic` | `{category, key, value, metadata}` | `{key, timestamp}` | Store semantic memory |
| GET | `/v1/gateway/agents/{agent_id}/memory/semantic` | `{category, query, limit}` | `[SemanticMemory]` | Query semantic memory |
| GET | `/v1/gateway/agents/{agent_id}/stats` | — | `ExecutionStats` | Aggregate metrics for session |
| GET | `/v1/gateway/tools` | `{filters}` | `[ToolDefinition]` | Browse registered tools |
| POST | `/v1/gateway/tools/{tool_name}/invoke` | `{input}` | `{output}` or error | Direct tool invocation (bypasses agent) |
| GET | `/v1/gateway/models` | — | `[ModelInfo]` | List available models with cost info |

**Request/Response Examples:**

```json
// POST /v1/gateway/agents/init
{
  "model": "gpt-4",
  "temperature": 0.7,
  "max_tokens": 2048,
  "system_prompt": "You are a helpful research assistant.",
  "included_tools": ["web_search", "fetch_url"],
  "metadata": {"user_id": "user_123", "session_id": "sess_456"}
}

// Response
{
  "agent_id": "agent_uuid_xxxx",
  "model": "gpt-4",
  "system_prompt": "You are a helpful research assistant.",
  "tools": [
    {
      "name": "web_search",
      "description": "Search the web for information",
      "parameters": {...}
    },
    {
      "name": "fetch_url",
      "description": "Fetch content from a URL",
      "parameters": {...}
    }
  ]
}

// POST /v1/gateway/agents/{agent_id}/execute
{
  "user_input": "What is the capital of France?",
  "context": {"turn_id": "turn_001"}
}

// Response (tool call)
{
  "type": "TOOL_CALL",
  "tool_call": {
    "id": "tool_call_001",
    "tool_name": "web_search",
    "input": {"query": "capital of France"}
  },
  "reasoning": "The user is asking a factual question; I'll search the web."
}

// Response (completion)
{
  "type": "COMPLETION",
  "completion": "The capital of France is Paris, located in the north-central part of the country.",
  "reasoning": "I have sufficient information from previous searches to answer confidently."
}
```

**Implementation:** Routes are served by `server/routes.go` under `gin.Group("/v1/gateway")`. Handlers call `OrchestrationExecutor.Execute()` in a loop until completion.

**Stability:** Routes are stable within v0.2.x but may break in v0.3.0 during orchestration extraction. Clients should pin to v0.2.x if breaking changes are unacceptable.

---

#### 3.2.3 Layer 3: `/admin/*` — Operator Control Plane

Administrative endpoints for operators (not end users). Require API key or mTLS authentication (per implementation).

| Method | Path | Request | Response | Purpose |
|--------|------|---------|----------|---------|
| GET | `/admin/config` | — | `{env, version, build_time, modules}` | System configuration |
| POST | `/admin/tools/register` | `{name, description, parameters, timeout}` | `{name, status}` | Register a tool at runtime |
| DELETE | `/admin/tools/{tool_name}` | — | `{status}` | Unregister a tool |
| GET | `/admin/traces` | `{limit, filter}` | `[Span]` | Export recent spans (debugging) |
| GET | `/admin/metrics` | — | Prometheus text format | Prometheus metrics endpoint |
| POST | `/admin/memory/sessions/{session_id}/delete` | — | `{status}` | Hard-delete session memory |
| GET | `/admin/health/detailed` | — | `{status, services, latencies}` | Deep health check |

**Implementation:** Routes are served by `server/routes.go` under `gin.Group("/admin")`. Middleware enforces authentication.

**Observability:** Admin endpoints are **not** traced to avoid operator activity pollution in observability backends.

---

### 3.3 OTEL Span Export

v0.2.0 adds OpenTelemetry span export to `server/trace/` (~400 LOC). This enables operators to see every LLM invocation, tool call, and orchestration step in external backends (Langfuse, Jaeger, Grafana Tempo, etc.).

#### 3.3.1 Span Exporter Implementation

```go
// File: server/trace/exporter.go
package trace

import (
    "context"
    "fmt"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// ExporterConfig holds OTEL exporter configuration.
type ExporterConfig struct {
    // Endpoint is the OTEL Collector gRPC endpoint (e.g., "localhost:4317").
    Endpoint string

    // ServiceName is the resource name for this gateway instance (e.g., "agentic-gateway").
    ServiceName string

    // ServiceVersion is the gateway version (e.g., "v0.2.0").
    ServiceVersion string

    // Environment is the deployment environment (e.g., "development", "production").
    Environment string

    // SampleRate is the fraction of traces to export (0.0-1.0). 1.0 = all traces.
    SampleRate float64

    // Timeout is the max time to wait for span export.
    Timeout time.Duration
}

// InitializeExporter creates and starts an OTEL trace exporter.
// Returns a TraceProvider that must be shut down on graceful shutdown.
func InitializeExporter(ctx context.Context, cfg *ExporterConfig) (*sdktrace.TracerProvider, error) {
    // Create gRPC exporter to OTEL Collector.
    exporter, err := otlptracegrpc.New(
        ctx,
        otlptracegrpc.WithEndpoint(cfg.Endpoint),
        otlptracegrpc.WithTimeout(cfg.Timeout),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create OTEL exporter: %w", err)
    }

    // Create resource with service metadata.
    res, err := resource.New(
        ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String(cfg.ServiceName),
            semconv.ServiceVersionKey.String(cfg.ServiceVersion),
            attribute.String("deployment.environment", cfg.Environment),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    // Create trace provider with sampler.
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithResource(res),
        sdktrace.WithBatcher(exporter),
        sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
    )

    // Set global tracer provider.
    otel.SetTracerProvider(tp)

    return tp, nil
}

// Shutdown gracefully shuts down the exporter and flushes pending spans.
func (tp *sdktrace.TracerProvider) Shutdown(ctx context.Context) error {
    return tp.ForceFlush(ctx)
}
```

#### 3.3.2 Span Attributes for LLM Observability

All LLM invocations are traced with standardized attributes:

```go
// File: server/trace/llm_attributes.go
package trace

import (
    "go.opentelemetry.io/otel/attribute"
)

// LLMSpanAttributes returns a set of attributes for an LLM invocation.
type LLMSpanAttributes struct {
    // Model is the LLM model identifier (e.g., "gpt-4").
    Model string

    // InputTokens is the number of input tokens consumed.
    InputTokens int

    // OutputTokens is the number of output tokens generated.
    OutputTokens int

    // LatencyMs is the round-trip latency in milliseconds.
    LatencyMs int64

    // EstimatedCostUSD is the inferred cost based on token counts and model pricing.
    EstimatedCostUSD float64

    // TemperatureParam is the temperature setting.
    TemperatureParam float32

    // MaxTokensParam is the max_tokens setting.
    MaxTokensParam int

    // ToolUsed is the name of a tool called by the model (if any).
    ToolUsed string

    // Error is the error message if the invocation failed.
    Error string

    // UserID is the user making the request (from metadata).
    UserID string

    // SessionID is the conversation session ID.
    SessionID string
}

// ToAttributes converts LLMSpanAttributes to OTEL attributes.
// This is called when creating a span for an LLM invocation.
func (l *LLMSpanAttributes) ToAttributes() []attribute.KeyValue {
    attrs := []attribute.KeyValue{
        attribute.String("llm.model", l.Model),
        attribute.Int("llm.usage.input_tokens", l.InputTokens),
        attribute.Int("llm.usage.output_tokens", l.OutputTokens),
        attribute.Int64("llm.latency_ms", l.LatencyMs),
        attribute.Float64("llm.estimated_cost_usd", l.EstimatedCostUSD),
        attribute.Float32("llm.temperature", l.TemperatureParam),
        attribute.Int("llm.max_tokens", l.MaxTokensParam),
        attribute.String("user.id", l.UserID),
        attribute.String("session.id", l.SessionID),
    }
    if l.ToolUsed != "" {
        attrs = append(attrs, attribute.String("llm.tool_used", l.ToolUsed))
    }
    if l.Error != "" {
        attrs = append(attrs, attribute.String("llm.error", l.Error))
    }
    return attrs
}
```

#### 3.3.3 Span Integration in Orchestration

When `OrchestrationExecutor.Execute()` calls an LLM, it creates a span:

```go
// File: server/orchestration/executor.go (excerpt)
package orchestration

import (
    "context"
    "go.opentelemetry.io/otel"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
)

func (e *Executor) Execute(ctx context.Context, agent *Agent, sessionID string) (*ExecutionStep, error) {
    tracer := otel.Tracer("agentic-gateway/orchestration")
    ctx, span := tracer.Start(ctx, "orchestration.execute")
    defer span.End()

    // ... assess state, prepare prompt ...

    // Call LLM with instrumentation.
    ctx, llmSpan := tracer.Start(ctx, "llm.invocation")
    startTime := time.Now()

    response, inputTokens, outputTokens, err := e.callLLM(ctx, systemPrompt, conversationHistory)
    latencyMs := time.Since(startTime).Milliseconds()

    estimatedCost := e.estimateCost(agent.Model, inputTokens, outputTokens)

    llmAttrs := &trace.LLMSpanAttributes{
        Model:            agent.Model,
        InputTokens:      inputTokens,
        OutputTokens:     outputTokens,
        LatencyMs:        latencyMs,
        EstimatedCostUSD: estimatedCost,
        UserID:           agent.Config.Metadata["user_id"],
        SessionID:        sessionID,
    }
    if err != nil {
        llmAttrs.Error = err.Error()
    }

    for _, attr := range llmAttrs.ToAttributes() {
        llmSpan.SetAttribute(attr.Key, attr.Value)
    }
    llmSpan.End()

    // ... parse response, possibly call tools ...

    return &ExecutionStep{...}, nil
}
```

#### 3.3.4 Observability Dashboard Configuration

The docker-compose.example.yml (section 3.4) includes an OTEL Collector that forwards to Langfuse. Operators can build dashboards in Langfuse to visualize:

- **Token costs over time** (aggregated by model)
- **Tool invocation heatmap** (which tools are called most often)
- **Latency distribution** (p50, p95, p99 for LLM calls)
- **Session traces** (full execution graph with tool calls and timings)
- **Error rates** (failed tools, LLM rejections, etc.)

Example Langfuse query (pseudo-SQL):
```sql
SELECT
  model,
  SUM(input_tokens) as total_input_tokens,
  SUM(output_tokens) as total_output_tokens,
  SUM(estimated_cost_usd) as total_cost,
  COUNT(*) as invocation_count,
  AVG(latency_ms) as avg_latency_ms
FROM llm_invocations
WHERE timestamp > NOW() - INTERVAL '7 days'
GROUP BY model
ORDER BY total_cost DESC;
```

---

### 3.4 docker-compose.example.yml

A complete, production-ready stack demonstrating the full observability setup.

```yaml
# File: docker-compose.example.yml
version: "3.9"

services:
  # AgenticGateway service.
  gateway:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: agentic-gateway
    ports:
      - "8080:8080"
    environment:
      # LLM configuration.
      OPENAI_API_KEY: "${OPENAI_API_KEY:-sk-your-key-here}"
      LLM_MODEL: "gpt-4"

      # OTEL exporter configuration.
      OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
      OTEL_EXPORTER_OTLP_INSECURE: "true"
      OTEL_SERVICE_NAME: "agentic-gateway"
      OTEL_SERVICE_VERSION: "0.2.0"
      OTEL_ENVIRONMENT: "development"

      # Gateway settings.
      GATEWAY_HOST: "0.0.0.0"
      GATEWAY_PORT: "8080"
      LOG_LEVEL: "info"
      ENABLE_METRICS: "true"

      # Memory configuration (in-memory for demo; use PostgreSQL in production).
      MEMORY_STORE_TYPE: "in_memory"
      MEMORY_TTL_SECONDS: "86400"
    depends_on:
      otel-collector:
        condition: service_healthy
    networks:
      - agentic-stack
    restart: unless-stopped

  # OpenTelemetry Collector: receives spans from gateway, exports to Langfuse.
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.88.0
    container_name: otel-collector
    ports:
      - "4317:4317"   # gRPC receiver for spans
      - "4318:4318"   # HTTP receiver
      - "8888:8888"   # Prometheus metrics
    environment:
      # Langfuse endpoint and API key.
      LANGFUSE_API_ENDPOINT: "${LANGFUSE_API_ENDPOINT:-http://langfuse:3000}"
      LANGFUSE_PUBLIC_KEY: "${LANGFUSE_PUBLIC_KEY:-pk-demo}"
      LANGFUSE_SECRET_KEY: "${LANGFUSE_SECRET_KEY:-sk-demo}"
    volumes:
      - ./otel-collector-config.yaml:/etc/otel/config.yaml
    command: [ "--config=/etc/otel/config.yaml" ]
    healthcheck:
      test: [ "CMD", "curl", "-f", "http://localhost:8888/metrics" ]
      interval: 10s
      timeout: 5s
      retries: 3
    networks:
      - agentic-stack
    restart: unless-stopped

  # Langfuse: observability backend for trace ingestion and dashboarding.
  langfuse:
    image: ghcr.io/langfuse/langfuse:latest
    container_name: langfuse
    ports:
      - "3000:3000"
    environment:
      DATABASE_URL: "postgresql://langfuse:langfuse_password@postgres:5432/langfuse"
      NEXTAUTH_SECRET: "${NEXTAUTH_SECRET:-your-secret-key-here}"
      NEXTAUTH_URL: "http://localhost:3000"
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - agentic-stack
    restart: unless-stopped

  # PostgreSQL: backend for Langfuse and (optionally) memory store.
  postgres:
    image: postgres:15-alpine
    container_name: postgres
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: "langfuse"
      POSTGRES_USER: "langfuse"
      POSTGRES_PASSWORD: "langfuse_password"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: [ "CMD-SHELL", "pg_isready -U langfuse" ]
      interval: 10s
      timeout: 5s
      retries: 3
    networks:
      - agentic-stack
    restart: unless-stopped

  # Prometheus: optional metrics scraping.
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus-config.yaml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - "--config.file=/etc/prometheus/prometheus.yml"
    networks:
      - agentic-stack
    restart: unless-stopped

networks:
  agentic-stack:
    driver: bridge

volumes:
  postgres_data:
    driver: local
  prometheus_data:
    driver: local
```

**File: otel-collector-config.yaml**

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
      http:
        endpoint: "0.0.0.0:4318"

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024

exporters:
  # Forward spans to Langfuse.
  otlp:
    endpoint: "${LANGFUSE_API_ENDPOINT}:3000"  # Langfuse OTLP receiver
    headers:
      Authorization: "Bearer ${LANGFUSE_PUBLIC_KEY}:${LANGFUSE_SECRET_KEY}"

  # Also export metrics to Prometheus.
  prometheus:
    endpoint: "0.0.0.0:8888"

service:
  pipelines:
    traces:
      receivers: [ otlp ]
      processors: [ batch ]
      exporters: [ otlp ]
    metrics:
      receivers: [ otlp ]
      processors: [ batch ]
      exporters: [ prometheus ]
```

**File: prometheus-config.yaml**

```yaml
global:
  scrape_interval: 30s
  scrape_timeout: 10s
  evaluation_interval: 30s

scrape_configs:
  - job_name: "gateway"
    static_configs:
      - targets: [ "gateway:8080" ]
    metrics_path: "/v1/metrics"

  - job_name: "otel-collector"
    static_configs:
      - targets: [ "otel-collector:8888" ]
```

**Quick Start:**

```bash
# Set environment variables.
export OPENAI_API_KEY="sk-your-actual-key"
export LANGFUSE_PUBLIC_KEY="pk-your-public-key"
export LANGFUSE_SECRET_KEY="sk-your-secret-key"

# Start the stack.
docker-compose -f docker-compose.example.yml up -d

# Verify gateway is running.
curl http://localhost:8080/v1/health
# {"status":"ok","uptime_seconds":5,"version":"v0.2.0"}

# Verify Langfuse dashboard.
open http://localhost:3000
# Log in with credentials (default: user@langfuse.com / password)

# Initialize an agent and make a request.
curl -X POST http://localhost:8080/v1/gateway/agents/init \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "temperature": 0.7,
    "max_tokens": 2048,
    "system_prompt": "You are a helpful assistant."
  }'

# Observe the trace in Langfuse dashboard after a moment.
```

---

### 3.5 Release Automation

#### 3.5.1 Makefile Targets

```makefile
# File: Makefile
.PHONY: help build test lint vet docker docker-build release clean

BINARY_NAME := agentic-gateway
VERSION := v0.2.0
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
MODULE_PATH := github.com/TresPies-source/AgenticGatewayByDojoGenesis

help:
	@echo "AgenticGateway v0.2.0 Build Targets"
	@echo "===================================="
	@echo "  make build       - Compile binary for current platform"
	@echo "  make test        - Run all tests with coverage"
	@echo "  make lint        - Run golangci-lint"
	@echo "  make vet         - Run go vet"
	@echo "  make docker      - Build Docker image"
	@echo "  make release     - Build release binaries (via goreleaser)"
	@echo "  make clean       - Remove build artifacts"

build:
	@echo "Building $(BINARY_NAME) $(VERSION) for $(GOOS)/$(GOARCH)..."
	CGO_ENABLED=0 go build \
		-ldflags="-X $(MODULE_PATH)/server.Version=$(VERSION) -X $(MODULE_PATH)/server.BuildTime=$(BUILD_TIME) -X $(MODULE_PATH)/server.Commit=$(COMMIT)" \
		-o bin/$(BINARY_NAME) \
		./cmd/server

test:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...

vet:
	@echo "Running go vet..."
	go vet ./...

docker:
	@echo "Building Docker image $(BINARY_NAME):$(VERSION)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.

release:
	@echo "Building release binaries via goreleaser..."
	@which goreleaser > /dev/null || (echo "goreleaser not installed; install with: go install github.com/goreleaser/goreleaser@latest" && exit 1)
	goreleaser release --clean

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/ dist/ coverage.out
	go clean

fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	go mod tidy
```

#### 3.5.2 Goreleaser Configuration

```yaml
# File: .goreleaser.yaml
version: 2

project_name: agentic-gateway

before:
  hooks:
    - go mod download
    - go mod tidy

builds:
  - id: default
    main: ./cmd/server
    binary: agentic-gateway
    env:
      - CGO_ENABLED=0
    ldflags:
      - -X github.com/TresPies-source/AgenticGatewayByDojoGenesis/server.Version={{.Version}}
      - -X github.com/TresPies-source/AgenticGatewayByDojoGenesis/server.BuildTime={{.Date}}
      - -X github.com/TresPies-source/AgenticGatewayByDojoGenesis/server.Commit={{.Commit}}
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}-{{ .Version }}-{{ .Os }}-{{ .Arch }}"
    files:
      - README.md
      - LICENSE
      - docker-compose.example.yml
      - otel-collector-config.yaml
      - prometheus-config.yaml

checksum:
  name_template: "{{ .ProjectName }}-{{ .Version }}-checksums.txt"
  algorithm: sha256

release:
  github:
    owner: TresPies-source
    name: AgenticGatewayByDojoGenesis
  prerelease: false
  name_template: "Release {{.Version}}"
  disable: true  # Disable GitHub release creation; do this manually per ADR-006

dockers:
  - image_templates:
      - "ghcr.io/TresPies-source/agentic-gateway:{{ .Version }}"
      - "ghcr.io/TresPies-source/agentic-gateway:latest"
    dockerfile: Dockerfile
    build_flag_templates:
      - "--label=org.opencontainers.image.version={{ .Version }}"
      - "--label=org.opencontainers.image.created={{ .Date }}"
      - "--label=org.opencontainers.image.revision={{ .FullCommit }}"
```

#### 3.5.3 Release Process

```bash
# 1. Merge v0.2.0 branch to main.

# 2. Ensure all tests pass.
make clean test lint vet

# 3. Build release binaries (no git tag yet per ADR-006).
make release

# 4. Publish binaries and checksums to GitHub Releases (manual).
# URL: github.com/TresPies-source/AgenticGatewayByDojoGenesis/releases
# Assets: bin/agentic-gateway-* + checksums.txt

# 5. Publish Docker image to GHCR (manual).
docker push ghcr.io/TresPies-source/agentic-gateway:v0.2.0
docker push ghcr.io/TresPies-source/agentic-gateway:latest

# 6. Announce release in GitHub Discussions and #releases Slack channel.
```

---

### 3.6 Developer Documentation

#### 3.6.1 OpenAPI Specification Generation

Gin routes are automatically converted to OpenAPI 3.0 via `swaggo/swag`:

```bash
# Install swag.
go install github.com/swaggo/swag/cmd/swag@latest

# Generate OpenAPI spec from Gin route comments.
swag init -g cmd/server/main.go

# Output: docs/swagger.yaml, docs/swagger.json

# Serve Swagger UI on /docs/swagger.html.
# (Enable via middleware in server/routes.go)
```

**Swagger annotation example:**

```go
// File: server/routes/gateway.go

// @Summary Initialize an agent
// @Description Create a stateful agent instance with model and tools.
// @ID initAgent
// @Accept json
// @Produce json
// @Param config body gateway.AgentConfig true "Agent configuration"
// @Success 200 {object} gateway.Agent
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/gateway/agents/init [post]
func InitAgentHandler(c *gin.Context) {
    var cfg gateway.AgentConfig
    if err := c.ShouldBindJSON(&cfg); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    // ... handler logic ...
}
```

**Generated docs are served at `/docs/swagger.html` and `/docs/swagger.json`.**

#### 3.6.2 CONTRIBUTING.md Outline

```markdown
# Contributing to AgenticGateway

## Code of Conduct

We are committed to providing a welcoming and inclusive environment.

## Development Setup

### Prerequisites
- Go 1.24+
- Docker & Docker Compose
- golangci-lint

### Local Build

```bash
git clone https://github.com/TresPies-source/AgenticGatewayByDojoGenesis
cd AgenticGatewayByDojoGenesis
make build
./bin/agentic-gateway
```

### Running Tests

```bash
make test
make lint
make vet
```

## Adding a Custom Tool

### Step 1: Implement ToolFunc

```go
// File: tools/custom/my_tool.go
package custom

import (
    "context"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

func MyToolFunc(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Implement tool logic
    return map[string]interface{}{"result": "..."}, nil
}
```

### Step 2: Register in DI Container

```go
// File: cmd/server/main.go (in tools initialization section)

registry.RegisterTool(&gateway.ToolDefinition{
    Name:        "my_tool",
    Description: "Does something useful",
    Parameters: map[string]interface{}{
        "query": map[string]interface{}{
            "type":        "string",
            "description": "what to do",
        },
    },
    Function: custom.MyToolFunc,
    Timeout:  30 * time.Second,
})
```

### Step 3: Test

```go
// File: tools/custom/my_tool_test.go
func TestMyToolFunc(t *testing.T) {
    input := map[string]interface{}{"query": "test"}
    output, err := MyToolFunc(context.Background(), input)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    // Assert output...
}
```

### Step 4: Submit PR

- Ensure tests pass: `make test`
- Run linter: `make lint`
- Update this README with tool usage example
- Request review from maintainers

## Extending Interfaces

To implement a custom MemoryStore, ToolRegistry, or AgentInitializer:

1. Implement the interface in `pkg/gateway/`
2. Inject via main.go DI container
3. Add integration tests
4. Document in docs/ARCHITECTURE.md

## Running the Full Stack Locally

```bash
docker-compose -f docker-compose.example.yml up
curl http://localhost:8080/v1/health
open http://localhost:3000  # Langfuse dashboard
```

## Code Style

- Follow Go conventions (effective go.dev)
- Use golangci-lint rules (see .golangci.yml)
- Add Godoc comments to all exported types and functions
- Keep functions under 50 lines; break up long orchestration logic
- Use context.Context for cancellation and timeouts

## Commit Messages

```
<type>: <short summary>

<detailed explanation>

Fixes #<issue-number>
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`

Example:
```
feat: add semantic memory query endpoint

Implement GET /v1/gateway/agents/{id}/memory/semantic for retrieving
stored facts and embeddings from memory store.

Fixes #42
```

## Release Process

See RELEASE.md (TBD for v0.3.0).

## Questions?

- Open an issue on GitHub
- Discuss in GitHub Discussions
- Reach out to @cruz-trespies on Slack
```

#### 3.6.3 Godoc Coverage Requirements

All exported types, functions, and methods must have Godoc comments:

```go
// AgentConfig is the request payload for agent initialization.
// It specifies the model, parameters, and tools to use.
type AgentConfig struct {
    // Model is the LLM model identifier (e.g., "gpt-4").
    Model string `json:"model"`
    // ... other fields documented ...
}

// InitializeAgent creates a ready-to-run Agent with model, tools, and system prompt.
// The config parameter contains user-facing options for model selection and tuning.
// Returns an error if the config is invalid or the model is unavailable.
func (init *AgentInitializer) InitializeAgent(ctx context.Context, config *AgentConfig) (*Agent, error) {
    // ... implementation ...
}
```

**Validation in CI:** `make vet` and `golangci-lint run` will warn on missing Godoc. Target: zero warnings on exported symbols.

---

## 4. Implementation Plan

### Phase 1: Foundation (Week 1)

**Goal:** Define and test all 4 interfaces; set up release automation.

| Day | Task | Owner | Deliverable |
|-----|------|-------|-------------|
| Mon | Write pkg/gateway/ interfaces (all 4) | Cruz | `pkg/gateway/*.go` with comprehensive Godoc |
| Tue | Implement standard registry + memory store wrappers | Claude | `tools/standard_registry.go`, `memory/standard_store.go` |
| Wed | Add OTEL exporter to server/trace/ | Claude | `server/trace/exporter.go`, span attributes |
| Thu | Create Makefile + Goreleaser config | Cruz | `Makefile`, `.goreleaser.yaml` |
| Fri | Test local build + docker build | Both | Confirm `make build` and `make docker` work |

**Testing:** Unit tests for each interface; ensure v0.1.0 tests still pass.

---

### Phase 2: HTTP Routing & Integration (Week 2)

**Goal:** Implement all /v1/gateway/ and /admin/ endpoints; wire DI container.

| Day | Task | Owner | Deliverable |
|-----|------|-------|-------------|
| Mon | Add /v1/gateway/agents/init and /execute handlers | Claude | `server/routes/gateway_agents.go` |
| Tue | Add /v1/gateway/tools/* and /v1/gateway/memory/* handlers | Claude | `server/routes/gateway_tools.go`, `server/routes/gateway_memory.go` |
| Wed | Add /admin/* handlers and auth middleware | Cruz | `server/routes/admin.go`, middleware |
| Thu | Update DI container in main.go to wire interfaces | Both | `cmd/server/main.go` |
| Fri | Integration tests for all new routes | Both | 50+ test cases covering happy path + errors |

**Testing:** End-to-end tests with mock LLM; ensure backward compat on /v1/ routes.

---

### Phase 3: Observability & Stack (Week 3)

**Goal:** Complete OTEL integration; create docker-compose example; OpenAPI docs.

| Day | Task | Owner | Deliverable |
|-----|------|-------|-------------|
| Mon | Integrate OTEL spans into orchestration loop | Claude | LLM span instrumentation in server/orchestration/ |
| Tue | Create docker-compose.example.yml + OTEL config files | Cruz | Complete, tested docker-compose |
| Wed | Generate OpenAPI spec with swaggo | Claude | `docs/swagger.{yaml,json}` + /docs/swagger.html endpoint |
| Thu | Write CONTRIBUTING.md with tool walkthrough | Cruz | Example tool implementation + PR workflow |
| Fri | Full-stack testing: docker-compose up → init agent → observe trace | Both | Langfuse dashboard shows LLM invocations + costs |

**Testing:** docker-compose up + API calls + verify traces in Langfuse.

---

### Phase 4: Documentation & Release Prep (Week 4)

**Goal:** Complete Godoc, finalize spec, prepare release artifacts.

| Day | Task | Owner | Deliverable |
|-----|------|-------|-------------|
| Mon | Audit and complete Godoc on all exported types | Both | 100% Godoc coverage; `go doc ./...` clean |
| Tue | Final testing: test coverage ≥ 80% on new code | Both | coverage.out report; no regressions |
| Wed | Create release notes + update README | Cruz | CHANGELOG.md, updated README.md |
| Thu | Security audit: dependency check, code review | Both | No blocking issues |
| Fri | Release binaries + Docker image; announce | Cruz | GitHub Releases, GHCR, announcements |

**Testing:** Full regression suite on all platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64).

---

### Critical Path Dependencies

```
Foundation (Week 1)
    ↓
Release Automation (Week 1, Fri) ← must precede Phase 2 to test builds
    ↓
HTTP Routing (Week 2)
    ↓
Observability Integration (Week 3)
    ↓
Documentation (Week 4)
```

### Testing Strategy

1. **Unit Tests:** Each interface has ≥3 test cases (happy path, error case, edge case)
2. **Integration Tests:** End-to-end with mock LLM; test orchestration loop with tools
3. **Docker Tests:** docker-compose up → curl → verify health
4. **Backward Compat:** All v0.1.0 routes tested against old clients (if available)
5. **Load Test:** Simple load test with Apache Bench (target: 1000 req/sec on /v1/chat/completions)

---

## 5. Risk Assessment & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|-----------|
| **OTEL exporter crashes on startup** | Medium | High | Pre-integration tests; graceful fallback if exporter unavailable; no tracing is better than no gateway |
| **Interface design changes required after HTTP routing** | Medium | Medium | Strict code review on Phase 1; interfaces locked by Wed of Week 1 |
| **Docker image fails to boot in CI/CD** | Low | High | Test docker-compose locally in Phase 3; use Docker's healthcheck to catch issues early |
| **Token cost calculation is inaccurate** | Low | Medium | Document cost model clearly; allow operators to override pricing in config |
| **Backward compat broken: /v1/chat/completions behaves differently** | Low | Critical | Freeze /v1/* routes in code review; test with v0.1.0 client libraries |
| **Release binaries are missing for a platform (e.g., arm64)** | Low | Low | Test `make release` locally on both x86 and arm64 (qemu if needed) |
| **New contributors can't follow CONTRIBUTING.md walkthrough** | Medium | Low | Walkthrough reviewed by non-author; test walkthrough on clean checkout |
| **OTEL span overhead causes latency regression** | Low | Medium | Measure baseline latency; confirm < 10ms overhead; use sampler to reduce volume |

---

## 6. Rollback & Contingency

### Partial Failure Scenarios

#### Scenario A: OTEL exporter unavailable at startup
**Behavior:** Gateway logs warning, continues with tracing disabled.
**Recovery:** Check OTEL_EXPORTER_OTLP_ENDPOINT config; restart with correct endpoint.
**Time to recovery:** < 5 min.

#### Scenario B: New /v1/gateway/* routes have bugs post-release
**Behavior:** Roll back to v0.1.0; new clients use /v1/chat/completions fallback.
**Recovery:** Hotfix v0.2.1; patch release in 24h.
**Impact:** New agentic features unavailable; classic OpenAI-compat mode works.

#### Scenario C: Docker image fails to boot
**Behavior:** Container exits immediately.
**Recovery:** Check logs: `docker logs agentic-gateway`. Common issue: OPENAI_API_KEY not set.
**Time to recovery:** < 2 min.

#### Scenario D: Orchestration extraction (v0.3.0) breaks existing server code
**Scenario:** Won't happen in v0.2.0. v0.3.0 is deferred; v0.2.0 keeps orchestration in-tree.
**Mitigation:** Code review process for v0.3.0 PR will test extraction safety.

### Rollback Procedure

```bash
# If v0.2.0 has critical bug:

# 1. Tag v0.2.0-rollback as the last good v0.1.0 build.
git tag -a v0.2.0-rollback-to-v0.1.0 <v0.1.0-commit>
git push origin v0.2.0-rollback-to-v0.1.0

# 2. Revert main to v0.1.0.
git revert HEAD --no-edit

# 3. Publish hotfix guidance in GitHub Discussions.
# "Use v0.1.0 until v0.2.1 is released (ETA 24h)."

# 4. Fix bug in v0.2.1 branch, fast-track review, release.
```

---

## 7. Appendices

### A. Future Considerations for v0.3.0

**Orchestration Extraction:** Move `server/orchestration/` to standalone `github.com/TresPies-source/AgenticOrchestrationByDojoGenesis` module. Benefits:
- Teams can use orchestration without running full gateway
- Faster iteration on orchestration logic
- Clearer separation of concerns

**Strategy:** v0.2.0 lands interfaces and observability; v0.3.0 extracts orchestration. Both happen before any public git tag.

**Multi-LLM Support:** v0.2.0 assumes single LLM provider. v0.3.0 should add support for routing to multiple providers (OpenAI, Anthropic, local) based on cost/latency/capability.

**Agent Persistence:** v0.2.0 agents are ephemeral (in-memory). v0.3.0 should add optional persistence: save agent state to PostgreSQL, resume from checkpoint.

**Tool Versioning:** v0.2.0 tools have no version. v0.3.0 should allow multiple versions of the same tool; agents specify which version to use.

### B. Open Questions

1. **Should /admin/ require mTLS or API key?** Current spec is agnostic; implementation should enforce one or both.
2. **Can OTEL spans be filtered/sampled per user?** Yes, via context propagation and sampler rules. Document in Langfuse setup guide.
3. **What's the SLA for OTEL export latency?** Target: < 1s from span end to Langfuse ingestion. Acceptable: < 5s.
4. **Should memory/store support transactions?** Optional in v0.2.0. v0.3.0 may add ACID guarantees for critical operations.
5. **Can operators implement custom ToolRegistry without forking?** Yes, via DI. Need clear example in CONTRIBUTING.md.

### C. References

- **ADR-001:** Hybrid API Surface (pkg/gateway/ interfaces, three-layer HTTP)
- **ADR-005:** OTEL Span Export (server/trace/ integration)
- **ADR-006:** No Public Git Tag Until Orchestration Extraction
- **Module Path:** github.com/TresPies-source/AgenticGatewayByDojoGenesis
- **Go Version:** 1.24+
- **OTEL Specs:** opentelemetry.io/docs
- **Langfuse Docs:** langfuse.com/docs

### D. Tool Registry Example: Web Search

```go
// File: tools/web_search/web_search.go
package websearch

import (
    "context"
    "fmt"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

// WebSearchFunc implements a simple web search tool.
func WebSearchFunc(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    query, ok := input["query"].(string)
    if !ok {
        return nil, fmt.Errorf("query parameter is required and must be a string")
    }

    // Call a real search API (e.g., Google Custom Search, Serper, SerpAPI).
    // For this example, return mock results.
    results := []map[string]string{
        {
            "title": "First Result",
            "url":   "https://example.com/1",
            "snippet": "A snippet of the first result...",
        },
        {
            "title": "Second Result",
            "url":   "https://example.com/2",
            "snippet": "A snippet of the second result...",
        },
    }

    return map[string]interface{}{
        "query":   query,
        "results": results,
    }, nil
}

// Definition returns the ToolDefinition for web search.
func Definition() *gateway.ToolDefinition {
    return &gateway.ToolDefinition{
        Name:        "web_search",
        Description: "Search the web for information on a given query",
        Parameters: map[string]interface{}{
            "query": map[string]interface{}{
                "type":        "string",
                "description": "What to search for",
            },
        },
        Function: WebSearchFunc,
        Timeout:  10 * time.Second,
    }
}
```

**Usage in main.go:**

```go
import (
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools/web_search"
)

// In DI container setup:
registry.RegisterTool(websearch.Definition())
```

### E. MemoryStore Example: PostgreSQL with pgvector

```go
// File: memory/postgres_store.go
package memory

import (
    "context"
    "database/sql"
    "github.com/pgvector/pgvector-go"
    _ "github.com/lib/pq"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

// PostgresStore implements gateway.MemoryStore backed by PostgreSQL + pgvector.
type PostgresStore struct {
    db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed memory store.
func NewPostgresStore(dbURL string) (*PostgresStore, error) {
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        return nil, err
    }
    if err := db.Ping(); err != nil {
        return nil, err
    }
    // Create tables if they don't exist (pgvector extension required).
    // ... schema setup ...
    return &PostgresStore{db: db}, nil
}

// StoreConversationTurn persists a turn to PostgreSQL.
func (ps *PostgresStore) StoreConversationTurn(ctx context.Context, sessionID, turnID string, turn *gateway.ConversationTurn) error {
    query := `
        INSERT INTO conversation_turns (session_id, turn_id, user_input, agent_output, tool_calls, metadata, timestamp)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
    _, err := ps.db.ExecContext(ctx, query, sessionID, turnID, turn.UserInput, turn.AgentOutput, turn.ToolCalls, turn.Metadata, turn.Timestamp)
    return err
}

// QuerySemanticMemory searches semantic memories by embedding similarity.
func (ps *PostgresStore) QuerySemanticMemory(ctx context.Context, sessionID, category string, query string, limit int) ([]*gateway.SemanticMemory, error) {
    // Generate embedding for query (via external embedding model).
    queryEmbedding, err := generateEmbedding(ctx, query)
    if err != nil {
        return nil, err
    }

    // Search pgvector table for nearest neighbors.
    sqlQuery := `
        SELECT key, value, metadata, timestamp, 1 - (embedding <=> $1) as similarity
        FROM semantic_memories
        WHERE session_id = $2 AND category = $3
        ORDER BY embedding <=> $1
        LIMIT $4
    `
    rows, err := ps.db.QueryContext(ctx, sqlQuery, pgvector.NewVector(queryEmbedding), sessionID, category, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var memories []*gateway.SemanticMemory
    for rows.Next() {
        var mem gateway.SemanticMemory
        if err := rows.Scan(&mem.Key, &mem.Value, &mem.Metadata, &mem.Timestamp, &mem.Similarity); err != nil {
            return nil, err
        }
        memories = append(memories, &mem)
    }
    return memories, nil
}

// ... other interface methods ...
```

---

## Conclusion

AgenticGateway v0.2.0 transforms the server from a monolithic tool into an **extensible, observable platform**. By exposing clean Go interfaces and a three-layer HTTP surface, we enable organizational independence. By adding OTEL integration, we give operators production visibility. By automating releases, we lower the friction for adoption.

This release does not extract orchestration or break existing clients. Instead, it provides the **foundation and observability** that make safe extraction possible in v0.3.0.

The 3-4 week timeline is aggressive but achievable with clear phasing, parallel work, and comprehensive testing. Success criteria are measurable: test coverage, interface completeness, docker-compose functionality, and Godoc coverage.

---

**Specification Finalized:** 2026-02-13
**Target Release:** Q1 2026 (3-4 weeks)
**Status:** Ready for implementation phase
