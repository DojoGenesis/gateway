# Agentic Gateway by Dojo Genesis

A modular Go framework for building agentic AI systems with pluggable providers, tool execution, DAG-based orchestration, and persistent memory.

## Architecture

The gateway is structured as a Go workspace with ten independently-versioned modules:

```
AgenticGatewayByDojoGenesis/
├── go.work                 # Workspace root
├── shared/                 # Cross-cutting types and error definitions
├── events/                 # Structured streaming events (SSE)
├── provider/               # Model provider plugin system (gRPC)
├── tools/                  # Tool registry and execution engine
├── memory/                 # Conversation memory with semantic compression
├── mcp/                    # MCP host integration (server lifecycle + tool bridge)
├── orchestration/          # DAG-based task planning and execution (standalone)
├── disposition/            # Agent personality and behavior config (ADA contract)
├── skill/                  # Tiered skill executor (44 skills, Tiers 0-2)
└── server/                 # HTTP server, agent logic, handlers
```

### Module Dependency Graph

```
shared          (stdlib only)
  │
events          (stdlib only)
  │
provider        (shared, go-plugin, gRPC, protobuf)
  │
tools           (shared, provider)
  │
memory          (shared, provider, sqlite3)
  │
mcp             (shared, tools)
  │
orchestration   (shared, tools — standalone DAG engine)
  │
disposition     (shared — agent personality config)
  │
skill           (shared, tools, orchestration — tiered skill executor)
  │
server          (all modules above + gin, cors, cron, etc.)
```

## Module Overview

| Module | Description | Key Types |
|--------|-------------|-----------|
| `shared` | Cross-cutting currency types, standard errors | `Message`, `ToolCall`, `Usage`, `TaskStatus` |
| `events` | Structured streaming events for SSE | `StreamEvent`, event constructors |
| `provider` | Plugin-based model provider system via gRPC | `ModelProvider`, `PluginManager`, `CompletionRequest` |
| `tools` | Tool registry, execution, and helper utilities | `ToolDefinition`, `RegisterTool`, `InvokeTool` |
| `memory` | Conversation memory with semantic compression | `MemoryManager`, `CompressionService`, `EmbeddingService` |
| `mcp` | MCP host integration, server lifecycle, tool bridge | `MCPHostManager`, `MCPServerConnection`, `MCPToolBridge` |
| `orchestration` | DAG-based task planning and execution (standalone) | `Planner`, `Engine`, `ExecutionContext` |
| `disposition` | Agent personality and behavior config (ADA contract) | `Disposition`, `PersonalityTraits` |
| `skill` | Tiered skill executor (44 skills, Tiers 0-2) | `SkillExecutor`, `SkillRegistry`, `SkillLoader` |
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
git clone https://github.com/dojogenesis/agentic-gateway.git
cd agentic-gateway
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

    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
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
