# AgenticGateway Architecture

**Version:** v0.2.0 (Phase 3: MCP Server Wiring)
**Last Updated:** 2026-02-13

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Core Components](#core-components)
3. [MCP Integration](#mcp-integration)
4. [Data Flow](#data-flow)
5. [Observability](#observability)
6. [Deployment](#deployment)

---

## System Overview

AgenticGateway is a unified API gateway for agentic systems, providing:

- **OpenAI-compatible chat API** (`/v1/chat/completions`)
- **Tool registry and orchestration** (DAG-based execution)
- **Multi-provider routing** (Anthropic, OpenAI, local models)
- **MCP integration** (dynamic tool discovery from external servers)
- **Memory management** (SQLite-based persistence + garden metaphor)
- **OTEL observability** (distributed tracing to Langfuse/OTEL collector)

**High-Level Architecture:**

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Applications                      │
│  (CLI tools, web apps, IDEs, chat interfaces)                   │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ HTTP/SSE
                 │
┌────────────────▼────────────────────────────────────────────────┐
│                      AgenticGateway                              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Server Layer (Gin)                                      │  │
│  │  - /v1/chat/completions (OpenAI-compatible)              │  │
│  │  - /v1/gateway/* (Gateway-specific APIs)                 │  │
│  │  - /admin/* (Admin + diagnostics)                        │  │
│  └────┬─────────────────────────────────────────────────────┘  │
│       │                                                          │
│  ┌────▼──────────────────────────────────────────────────────┐ │
│  │  Orchestration Engine                                    │  │
│  │  - DAG planner (dependency resolution)                   │  │
│  │  - Tool resolver (namespace support)                     │  │
│  │  - Execution engine (parallel + sequential)              │  │
│  └────┬─────────────────────────────────────────────────────┘  │
│       │                                                          │
│  ┌────▼──────────────────────────────────────────────────────┐ │
│  │  Tool Registry                                           │  │
│  │  - Built-in tools (code execution, file ops)             │  │
│  │  - MCP tools (namespace-prefixed)                        │  │
│  │  - Provider tools (model-specific)                       │  │
│  └────┬──────────────────────────┬──────────────────────────┘  │
│       │                          │                              │
│  ┌────▼─────────┐       ┌────────▼───────────────────────────┐ │
│  │   Memory     │       │  MCP Host Manager                  │ │
│  │   Manager    │       │  - Server lifecycle                │ │
│  │   (SQLite)   │       │  - Health monitoring               │ │
│  └──────────────┘       │  - Tool bridge                     │ │
│                         └────┬───────────────┬───────────────┘ │
│                              │               │                  │
└──────────────────────────────┼───────────────┼──────────────────┘
                               │               │
               ┌───────────────▼─┐         ┌──▼──────────────┐
               │ MCPByDojoGenesis│         │    Composio     │
               │   (stdio)       │         │     (SSE)       │
               │  14 tools       │         │   25+ tools     │
               └─────────────────┘         └─────────────────┘
```

---

## Core Components

### 1. Server Layer (`server/`)

**Responsibility:** HTTP API surface, request routing, middleware

**Key files:**
- `server/server.go` — Main server struct, initialization
- `server/router.go` — Route definitions
- `server/handle_*.go` — Request handlers
- `server/middleware/` — Auth, logging, budgeting

**API Groups:**
- `/health`, `/metrics` — Infrastructure
- `/v1/chat/completions` — OpenAI-compatible chat API
- `/v1/gateway/*` — Gateway-specific APIs (tools, agents, orchestration)
- `/admin/*` — Admin endpoints (health, config, MCP status)

### 2. Orchestration Engine (`server/orchestration/`)

**Responsibility:** DAG planning and execution

**Key components:**
- `Planner` — Converts tool calls into dependency DAGs
- `Engine` — Executes DAG with parallelism + error handling
- `ExecutionContext` — Tracks state during execution

**DAG Example:**

```
┌─────────────┐
│ search_docs │──┐
└─────────────┘  │
                 │
┌─────────────┐  │    ┌──────────────┐
│ list_files  │──┼───►│ synthesize   │
└─────────────┘  │    └──────────────┘
                 │
┌─────────────┐  │
│ get_context │──┘
└─────────────┘
```

### 3. Tool Registry (`tools/`)

**Responsibility:** Tool registration, lookup, invocation

**Tool types:**
- **Built-in:** File operations, code execution, memory access
- **MCP tools:** External servers (namespace-prefixed)
- **Provider tools:** Model-specific capabilities

**Interface:**

```go
type ToolRegistry interface {
    Register(ctx context.Context, def *ToolDefinition) error
    Get(ctx context.Context, name string) (*ToolDefinition, error)
    List(ctx context.Context) ([]*ToolDefinition, error)
    ListByNamespace(ctx context.Context, prefix string) ([]*ToolDefinition, error)
}
```

### 4. Provider Layer (`provider/`)

**Responsibility:** Multi-provider LLM routing

**Supported providers:**
- **Anthropic** (Claude models)
- **OpenAI** (GPT models)
- **Local models** (Ollama, LM Studio)

**Plugin architecture:**
- Providers register via plugin system
- Gateway routes requests based on model ID
- Provider-specific context injection

### 5. Memory Manager (`memory/`)

**Responsibility:** Persistent memory storage with garden metaphor

**Features:**
- SQLite-based storage
- Vector search (optional)
- "Garden" organization (seeds, snapshots, growth tracking)
- Session-based memory isolation

**Schema:**

```
memories:
  - id (UUID)
  - content (text)
  - embedding (blob)
  - context (JSON)
  - created_at, updated_at

seeds:
  - id (UUID)
  - name, description
  - knowledge (text)
  - growth_stage
```

---

## MCP Integration

**Added in:** v0.2.0 (Phase 3)

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Gateway Runtime                                                 │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  MCPHostManager (mcp/host.go)                              │ │
│  │  - Lifecycle management (Start/Stop)                       │ │
│  │  - Health monitoring (periodic checks)                     │ │
│  │  - Auto-reconnection (with exponential backoff)            │ │
│  └──┬─────────────────────────────────────────────────────────┘ │
│     │                                                             │
│     │ Manages                                                     │
│     │                                                             │
│  ┌──▼────────────────────────────────────────────────────────┐  │
│  │  MCPServerConnection (mcp/connection.go)                   │  │
│  │  - Per-server connection state                             │  │
│  │  - Transport abstraction (stdio, SSE, HTTP)                │  │
│  │  - Tool discovery (ListTools)                              │  │
│  │  - Tool invocation (CallTool with retry)                   │  │
│  └──┬─────────────────────────────────────────────────────────┘  │
│     │                                                             │
│     │ Bridges to                                                  │
│     │                                                             │
│  ┌──▼────────────────────────────────────────────────────────┐  │
│  │  MCPToolBridge (mcp/bridge.go)                             │  │
│  │  - Namespace prefixing (server_id:tool_name)               │  │
│  │  - ToolFunc adapter (MCP → Gateway ToolFunc)               │  │
│  │  - Result normalization (MCP result → map)                 │  │
│  │  - OTEL span creation                                      │  │
│  └────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### MCP Server Types

**1. MCPByDojoGenesis (stdio)**

- **Binary:** `/opt/mcp-servers/mcp-by-dojo-genesis`
- **Transport:** stdio (subprocess)
- **Tools:** 14 Dojo-specific tools (wisdom search, seed application, reflection)
- **Namespace:** `mcp_by_dojo:`

**2. Composio (SSE)**

- **Endpoint:** `https://mcp.composio.dev/sse`
- **Transport:** SSE (HTTP Server-Sent Events)
- **Tools:** 100+ external integrations (GitHub, Slack, Linear, Jira, etc.)
- **Namespace:** `composio:`

**3. Custom MCP Servers**

Users can add custom MCP servers via YAML config:

```yaml
servers:
  - id: "custom"
    namespace_prefix: "custom"
    transport:
      type: "stdio"
      command: "/path/to/custom-server"
```

### Tool Discovery Flow

```
1. Gateway starts
   ↓
2. MCPHostManager reads gateway-config.yaml
   ↓
3. For each server:
   a. MCPServerConnection.Connect()
   b. MCPServerConnection.ListTools()
   c. MCPToolBridge.CreateToolDefinition() (with namespace prefix)
   d. ToolRegistry.Register()
   ↓
4. Health check loop starts (periodic reconnection)
   ↓
5. Tools available via /v1/gateway/tools
```

### Tool Invocation Flow

```
1. Orchestration engine resolves tool "mcp_by_dojo:search_wisdom"
   ↓
2. ToolRegistry.Get("mcp_by_dojo:search_wisdom")
   ↓
3. Returns ToolDefinition with ToolFunc (from MCPToolBridge)
   ↓
4. Execute ToolFunc(ctx, args)
   ↓
5. ToolFunc calls MCPServerConnection.CallTool("search_wisdom", args)
   ↓
6. MCPServerConnection sends JSON-RPC to MCP server
   ↓
7. MCP server executes tool and returns result
   ↓
8. Result normalized to map[string]interface{}
   ↓
9. OTEL span recorded with latency, input/output size
   ↓
10. Result returned to orchestration engine
```

### Configuration

**File:** `gateway-config.yaml`

**Schema:**

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

  servers:
    - id: "mcp_by_dojo"
      display_name: "MCP By Dojo Genesis"
      namespace_prefix: "mcp_by_dojo"
      transport:
        type: "stdio"
        command: "${MCP_BY_DOJO_BINARY}"
      tools:
        allowlist: []
        blocklist: []
      timeouts:
        startup: 10
        tool_default: 30
        health_check: 5
      health_check:
        enabled: true
        interval_sec: 60
      retry_policy:
        max_attempts: 2
        backoff_multiplier: 2.0
        max_backoff_ms: 5000

  observability:
    enabled: true
    trace_provider: "otel"
    tool_span_sample_rate: 1.0
```

### Admin Endpoint

**Endpoint:** `GET /admin/mcp/status`

**Response:**

```json
{
  "servers": {
    "mcp_by_dojo": {
      "server_id": "mcp_by_dojo",
      "display_name": "MCP By Dojo Genesis",
      "state": "connected",
      "tool_count": 14,
      "last_health_check": "2026-02-13T12:00:00Z",
      "last_error": ""
    }
  },
  "total_servers": 1,
  "total_tools": 14,
  "healthy": true
}
```

---

## Data Flow

### Chat Completion Request

```
Client
  │
  │ POST /v1/chat/completions
  │ { "model": "claude-3-5-sonnet-20241022", "messages": [...] }
  ▼
Server (handleChatCompletions)
  │
  │ Extract model ID, route to provider
  ▼
Provider Plugin (Anthropic)
  │
  │ If tool calls in response:
  ▼
Orchestration Engine
  │
  │ Plan DAG, resolve tools
  ▼
Tool Registry
  │
  │ Lookup tools (may be MCP tools)
  ▼
MCPHostManager (if MCP tool)
  │
  │ Invoke via MCPServerConnection
  ▼
MCP Server (e.g., MCPByDojoGenesis)
  │
  │ Execute tool, return result
  ▼
Orchestration Engine
  │
  │ Collect results, return to provider
  ▼
Provider Plugin
  │
  │ Format response
  ▼
Server
  │
  │ SSE stream or JSON response
  ▼
Client
```

### Observability (OTEL)

```
Every MCP tool call emits OTEL span:

Span attributes:
  - mcp.server_id: "mcp_by_dojo"
  - mcp.server_display_name: "MCP By Dojo Genesis"
  - mcp.tool_name: "search_wisdom"
  - mcp.tool_namespaced: "mcp_by_dojo:search_wisdom"
  - mcp.tool_latency_ms: 1234
  - mcp.tool_input_size_bytes: 512
  - mcp.tool_output_size_bytes: 2048

Export path:
  Gateway → OTEL Collector → Langfuse
```

---

## Observability

### OTEL Integration

**Architecture:**

```
┌─────────────┐
│   Gateway   │
│             │
│ (emits      │
│  OTEL       │
│  spans)     │
└──────┬──────┘
       │
       │ OTLP/HTTP
       │ (port 4318)
       ▼
┌─────────────┐
│    OTEL     │
│  Collector  │
│             │
│ (processes, │
│  batches)   │
└──────┬──────┘
       │
       │ OTLP/HTTP
       ▼
┌─────────────┐
│  Langfuse   │
│   (UI +     │
│  storage)   │
└─────────────┘
```

**Configuration:**

- Gateway: Set `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`
- OTEL Collector: See `otel-config.yaml`
- Langfuse: Access UI at http://localhost:3000

**Span Types:**
- `mcp.tool_call` — MCP tool invocations
- `orchestration.dag` — DAG executions
- `http.request` — HTTP API calls (optional)

### Metrics (Prometheus)

**Endpoint:** `GET /admin/metrics/prometheus`

**Metrics:**
- `gateway_tools_total` — Total registered tools
- `gateway_agents_active` — Active agent count
- `gateway_memory_alloc_bytes` — Memory usage
- `gateway_uptime_seconds` — Gateway uptime

---

## Deployment

### Docker Compose (Development + Production)

**File:** `docker-compose.yaml`

**Services:**
- `gateway` — Main gateway service
- `mcp-by-dojo` — Binary provider (copies binary to shared volume)
- `otel-collector` — OTEL trace collection
- `langfuse` — Observability UI
- `postgres` — Langfuse database

**Startup:**

```bash
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f gateway

# Check MCP status
curl http://localhost:8080/admin/mcp/status
```

### Environment Variables

**Required:**
- `OTEL_EXPORTER_OTLP_ENDPOINT` — OTEL collector endpoint
- `MCP_CONFIG_PATH` — Path to gateway-config.yaml

**Optional:**
- `MCP_BY_DOJO_BINARY` — Path to MCPByDojoGenesis binary
- `COMPOSIO_API_KEY` — Composio integration (if enabled)
- `LANGFUSE_NEXTAUTH_SECRET` — Langfuse authentication secret
- `LANGFUSE_SALT` — Langfuse encryption salt

### Health Checks

**Gateway health:** `GET /health`
**MCP health:** `GET /admin/mcp/status`
**OTEL collector health:** `http://otel-collector:13133/`
**Langfuse health:** `http://langfuse:3000/api/health`

---

## References

- [MCP Configuration Guide](./docs/mcp-configuration.md)
- [Composio Setup Guide](./docs/composio-setup.md)
- [Gateway-MCP Contract](./docs/v0.2.0/gateway-mcp-contract.md) (full specification)
- [ADR-002: MCP Host Architecture](../AgenticStackOrchestration/decisions/002-mcp-host-architecture.md)
- [ADR-003: Composio MCP Bridge](../AgenticStackOrchestration/decisions/003-composio-mcp-bridge.md)
- [ADR-005: OTEL + Langfuse](../AgenticStackOrchestration/decisions/005-otel-with-langfuse-example.md)

---

**Version History:**
- v0.1.0 — Initial release (provider abstraction, memory, orchestration)
- v0.2.0 — MCP integration (Phase 3: MCP Server Wiring)
