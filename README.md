# Agentic Gateway

**Your AI infrastructure. Your servers. Your rules.**

Building with AI agents today means choosing between renting from Big Tech (goodbye privacy), building from scratch ($2M and 12 months), or duct-taping research prototypes (good luck in production).

Agentic Gateway is the fourth option: a production-ready Go framework for self-hosted agentic AI. Multi-provider LLM routing, DAG-based task orchestration, 44 built-in skills, MCP tool integration, and semantic memory — running on your infrastructure, not someone else's.

**140,000+ lines of Go. 19 independently-versioned modules. Zero vendor lock-in.**

## Quick Start

```bash
# Download and run (macOS Apple Silicon)
curl -L https://github.com/DojoGenesis/gateway/releases/latest/download/agentic-gateway_darwin_arm64.tar.gz | tar xz
./agentic-gateway

# Or with Docker
docker-compose up -d
```

> **What you get:** Multi-provider LLM routing | DAG orchestration | 44 skills | MCP integration (14+ tools) | Semantic memory with compression | OTEL observability | Agent personality system (ADA) | Plugin architecture

## Why This Exists

Every serious AI deployment hits the same wall: you need orchestration, memory, tool access, observability, and cost control — and no single hosted service gives you all of it without owning your data. Agentic Gateway is the missing infrastructure layer. Like nginx for web servers or postgres for databases — except for agentic AI.

## Architecture

The gateway is structured as a Go workspace with ten independently-versioned modules:

```
AgenticGatewayByDojoGenesis/
├── go.work                 # Workspace root (19 modules)
├── main.go                 # Entry point
├── shared/                 # Cross-cutting types (package in root module)
├── events/                 # SSE event catalog (package in root module)
├── provider/               # Model provider plugin system (gRPC, 8 providers)
├── tools/                  # Tool registry and execution engine (33 tools)
├── memory/                 # Conversation memory with semantic compression
├── mcp/                    # MCP host integration (stdio + SSE + streamable_http)
├── orchestration/          # DAG-based task planning and execution (standalone)
├── disposition/            # Agent personality and behavior config (ADA contract)
├── skill/                  # Tiered skill executor (84 skills, Tiers 0-3)
├── apps/                   # MCP Apps host infrastructure (resource serving, tool proxy)
├── workflow/               # Durable workflow execution engine
├── integration/            # Integration testing harness
├── runtime/
│   ├── actor/              # Actor supervision tree
│   ├── cas/                # Content-addressable storage
│   ├── d1client/           # Cloudflare D1 client (edge sync)
│   ├── event/              # Event bus and routing
│   └── wasm/               # WASM sandbox execution
├── wasm-modules/
│   └── dip-scorer/         # DIP scoring compiled to WASM
├── cmd/dojo/               # CLI integration module
└── server/                 # HTTP API server, agent logic, handlers
```

### Module Dependency Graph

```
root module     (shared + events packages — stdlib only)
  │
provider        (root, go-plugin, gRPC, protobuf — 8 providers)
  │
tools           (root, provider — 33 registered tools)
  │
memory          (root, provider, sqlite3)
  │
mcp             (root, tools — stdio + SSE + streamable_http)
  │
orchestration   (root, tools — standalone DAG engine)
  │
disposition     (root — ADA contract v1.0.0)
  │
skill           (root, tools, orchestration — 84 skills, Tiers 0-3)
  │
workflow        (root, orchestration — durable workflow engine)
  │
runtime/*       (actor, cas, d1client, event, wasm — infrastructure layer)
  │
apps            (root, tools — MCP Apps host)
  │
server          (all modules above + gin, cors, cron, OTEL, Langfuse)
```

## Module Overview

| Module | Description | Key Types |
|--------|-------------|-----------|
| `shared` | Cross-cutting currency types, standard errors (root package) | `Message`, `ToolCall`, `Usage`, `TaskStatus` |
| `events` | SSE event catalog, 254-line schema (root package) | `StreamEvent`, event constructors |
| `provider` | Plugin-based model provider system via gRPC (8 providers) | `ModelProvider`, `PluginManager`, `CompletionRequest` |
| `tools` | Tool registry, execution, and helper utilities (33 tools) | `ToolDefinition`, `RegisterTool`, `InvokeTool` |
| `memory` | Conversation memory with semantic compression | `MemoryManager`, `CompressionService`, `EmbeddingService` |
| `mcp` | MCP host integration — stdio, SSE, streamable_http | `MCPHostManager`, `MCPServerConnection`, `MCPToolBridge` |
| `orchestration` | DAG-based task planning and execution (standalone) | `Planner`, `Engine`, `ExecutionContext` |
| `disposition` | Agent personality and behavior config (ADA contract v1.0.0) | `Disposition`, `PersonalityTraits` |
| `skill` | Tiered skill executor (84 skills, Tiers 0-3) | `SkillExecutor`, `SkillRegistry`, `SkillLoader` |
| `apps` | MCP Apps host infrastructure | `AppManager`, `ResourceRegistry`, `ToolCallProxy` |
| `workflow` | Durable workflow execution engine | `WorkflowEngine`, `WorkflowState`, `StepRunner` |
| `integration` | Integration testing harness | `IntegrationSuite`, `GatewayHarness` |
| `runtime/actor` | Actor supervision tree | `ActorSystem`, `Supervisor`, `Mailbox` |
| `runtime/cas` | Content-addressable storage | `CASStore`, `ContentRef`, `BlobWriter` |
| `runtime/d1client` | Cloudflare D1 client for edge sync | `D1Client`, `SyncManager` |
| `runtime/event` | Event bus and routing | `EventBus`, `EventRouter`, `Subscription` |
| `runtime/wasm` | WASM sandbox execution | `WASMRuntime`, `SandboxContext` |
| `wasm-modules/dip-scorer` | DIP scoring compiled to WASM | `DIPScorer`, `ScoringResult` |
| `server` | HTTP API server with agent logic | `PrimaryAgent`, handlers, middleware, config |

## Installation

### Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/dojogenesis/agentic-gateway/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/dojogenesis/agentic-gateway/releases/latest/download/agentic-gateway_darwin_arm64.tar.gz | tar xz
./agentic-gateway

# Linux (amd64)
curl -L https://github.com/dojogenesis/agentic-gateway/releases/latest/download/agentic-gateway_linux_amd64.tar.gz | tar xz
./agentic-gateway
```

### Docker

```bash
docker pull ghcr.io/dojogenesis/agentic-gateway:latest
docker run -p 8080:8080 ghcr.io/dojogenesis/agentic-gateway:latest
```

Or use docker compose for the full observability stack:

```bash
docker compose up -d
```

### Build from Source

#### Prerequisites

- Go 1.24+
- C compiler (for sqlite3 via CGO)

#### Build

```bash
git clone https://github.com/DojoGenesis/gateway.git
cd gateway
make build
```

#### Test

```bash
make test
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/DojoGenesis/gateway/provider"
    "github.com/DojoGenesis/gateway/tools"
)

func main() {
    // Create a plugin manager and discover providers
    pm := provider.NewPluginManager("./plugins")
    if err := pm.DiscoverPlugins(); err != nil {
        panic(err)
    }
    defer pm.Shutdown()

    // Register a custom tool
    tools.RegisterTool(&tools.ToolDefinition{
        Name:        "hello",
        Description: "Says hello",
        Handler: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
            name := tools.GetStringParam(params, "name", "world")
            return map[string]interface{}{"greeting": fmt.Sprintf("Hello, %s!", name)}, nil
        },
    })

    // Invoke it
    result, _ := tools.InvokeTool(context.Background(), "hello", map[string]interface{}{"name": "Gateway"})
    fmt.Println(result)
}
```

## License

See [LICENSE](LICENSE) for details.
