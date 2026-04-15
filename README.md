# Agentic Gateway

Self-hosted agentic AI runtime. Multi-provider LLM routing, DAG orchestration, content-addressable skill storage, and real-time SSE observability in a single Go binary.

## Quick Start

```bash
# Clone and build from source
git clone https://github.com/DojoGenesis/gateway.git
cd gateway
cp .env.example .env          # add your API keys
make build
./bin/agentic-gateway         # listening on :7340

# Or with Docker
docker compose -f docker-compose.example.yml up -d

# Or download a pre-built binary (macOS Apple Silicon)
curl -L https://github.com/DojoGenesis/gateway/releases/latest/download/agentic-gateway_darwin_arm64.tar.gz | tar xz
./agentic-gateway
```

The gateway is now running at `http://localhost:7340`. Send a chat completion:

```bash
curl -X POST http://localhost:7340/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Hello"}]}'
```

## Architecture

148K lines of Go across 19 independently-versioned modules in a Go workspace (`go.work`). Each module has its own `go.mod` and can be versioned, tested, and imported independently.

```
                        ┌─────────────────────────────┐
                        │     Chat UI / Workflow UI    │
                        │   (embedded Svelte 5 SPAs)   │
                        └──────────────┬──────────────┘
                                       │
                        ┌──────────────▼──────────────┐
                        │       server (Gin HTTP)      │
                        │  Auth, Conversations, Admin  │
                        │  Templates, Documents/RAG    │
                        │  CAS API, Mesh, WebSocket    │
                        └──────────────┬──────────────┘
                                       │
          ┌───────────┬───────────┬────┴────┬───────────┬───────────┐
          │           │           │         │           │           │
     ┌────▼───┐  ┌────▼───┐ ┌────▼───┐ ┌───▼────┐ ┌───▼────┐ ┌───▼────┐
     │provider│  │  tools │ │ memory │ │  mcp   │ │  skill │ │  apps  │
     │8 provs │  │33 tools│ │semantic│ │3 trans-│ │89 skills│ │MCP Apps│
     │gRPC    │  │registry│ │compress│ │ports   │ │Tiers 0-3│ │host    │
     └────────┘  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘
          │           │           │         │           │
     ┌────▼───────────▼───────────▼─────────▼───────────▼───────────┐
     │                    orchestration                              │
     │              DAG-based task planning + execution              │
     └──────────────────────────┬────────────────────────────────────┘
                                │
     ┌──────────┬───────────┬───┴───┬───────────┬───────────┐
     │          │           │       │           │           │
  ┌──▼───┐  ┌──▼───┐  ┌────▼──┐ ┌──▼───┐  ┌───▼────┐  ┌──▼───────┐
  │actor │  │ cas  │  │d1client│ │event │  │ wasm   │  │disposition│
  │super-│  │content│  │D1 edge│ │event │  │sandbox │  │ADA agent │
  │vision│  │addr. │  │ sync  │ │ bus  │  │execute │  │personality│
  └──────┘  └──────┘  └───────┘ └──────┘  └────────┘  └──────────┘
            runtime/*                                   (standalone)
```

## Providers

The gateway routes LLM requests across providers. Set the corresponding API key in `.env` to enable a provider -- providers without keys are silently skipped at startup.

| Provider | Env Var | Notes |
|----------|---------|-------|
| Anthropic (Claude) | `ANTHROPIC_API_KEY` | |
| OpenAI | `OPENAI_API_KEY` | Custom base URL via `OPENAI_BASE_URL` |
| Google (Gemini) | `GOOGLE_API_KEY` | |
| Groq | `GROQ_API_KEY` | |
| Mistral | `MISTRAL_API_KEY` | |
| DeepSeek | `DEEPSEEK_API_KEY` | Custom base URL via `DEEPSEEK_BASE_URL` |
| Kimi (Moonshot) | `KIMI_API_KEY` | Custom base URL via `KIMI_BASE_URL` |
| Ollama | `OLLAMA_HOST` | Auto-detected on `localhost:11434`; text-mode tool fallback for models without native tool support |

Explicit `provider/model` in the request always overrides the intent classifier. The classifier is fallback only.

## Module Overview

| Module | Description |
|--------|-------------|
| `shared` | Cross-cutting types: `Message`, `ToolCall`, `Usage`, `TaskStatus` (root package) |
| `events` | SSE event catalog, 254-line schema (root package) |
| `provider` | gRPC-based model provider plugin system (8 providers) |
| `tools` | Tool registry and execution engine (33 tools) |
| `memory` | Conversation memory with semantic compression |
| `mcp` | MCP host integration -- stdio, SSE, and streamable_http transports |
| `orchestration` | Standalone DAG-based task planning and execution |
| `disposition` | ADA agent personality and behavior config (v1.0.0 contract) |
| `skill` | Tiered skill executor (89 skills across Tiers 0-3) |
| `apps` | MCP Apps host infrastructure -- resource serving, tool proxy |
| `workflow` | Durable workflow execution engine |
| `integration` | Integration testing harness |
| `runtime/actor` | Actor supervision tree |
| `runtime/cas` | Content-addressable storage |
| `runtime/d1client` | Cloudflare D1 client for edge sync |
| `runtime/event` | Event bus and routing |
| `runtime/wasm` | WASM sandbox execution |
| `wasm-modules/dip-scorer` | DIP scoring compiled to WASM |
| `server` | HTTP API server: Gin router, agent logic, auth, handlers, middleware, streaming |

## API Routes

### Core

```
POST /v1/chat/completions          OpenAI-compatible chat completions
GET  /v1/models                    List available models across providers
GET  /events                       SSE stream (real-time agent observability)
GET  /health                       Health check
GET  /metrics                      Prometheus metrics
```

### Agent Management

```
POST /v1/gateway/agents            Create agent with disposition
GET  /v1/gateway/agents            List agents
GET  /v1/gateway/agents/:id        Get agent
POST /v1/gateway/agents/:id/chat   Chat with agent (streaming, tool-calling loop)
```

### Conversations (authenticated)

```
GET    /v1/conversations           List conversations
POST   /v1/conversations           Create conversation
GET    /v1/conversations/:id       Get conversation with messages
DELETE /v1/conversations/:id       Delete conversation
POST   /v1/conversations/:id/messages  Send message
```

### Content-Addressable Storage

```
GET  /api/cas/refs                 List all CAS refs
POST /api/cas/refs                 Store content by ref
GET  /api/cas/content/:ref         Retrieve content by ref
POST /api/cas/tags                 Create a tag
GET  /api/cas/delta                D1 sync delta
PUT  /api/cas/batch                Batch sync
```

### Tools and Skills

```
GET  /v1/tools                     List registered tools
POST /v1/tools/:name/invoke        Invoke a tool by name
GET  /v1/gateway/tools             Gateway tool discovery (MCP namespace support)
```

### Memory

```
POST /v1/memory                    Store memory
GET  /v1/memory                    List memories
POST /v1/memory/search             Semantic search
```

### Admin (requires admin auth)

```
GET  /admin/health                 Admin health diagnostics
GET  /admin/providers              Provider status
GET  /admin/costs                  Cost aggregation
GET  /admin/mcp/servers            MCP server registry
POST /admin/config/reload          Hot-reload configuration
GET  /admin/users                  User management
```

### Auth

```
POST /auth/register                Register user
POST /auth/login                   Login (returns JWT)
POST /auth/refresh                 Refresh token
GET  /auth/github                  GitHub OAuth2 flow start
GET  /auth/github/callback         GitHub OAuth2 callback
```

### Additional

```
POST /v1/orchestrate               DAG orchestration
POST /v1/templates                 Prompt templates (CRUD)
POST /v1/documents                 Document upload (RAG pipeline)
POST /v1/documents/search          RAG search
POST /api/workflows/:name/execute  Execute workflow
GET  /api/ws/workflow              WebSocket workflow events
GET  /chat                         Embedded Chat UI SPA
GET  /workflow                     Embedded Workflow Builder SPA
POST /mesh/announce                Federated mesh peer announce
```

## Project Structure

```
main.go                 Entry point (port 7340, .env loading, graceful shutdown)
go.work                 Workspace root (19 modules)
Makefile                build, test, lint, docker, SPA embedding
gateway-config.yaml     Runtime config (feature flags, MCP, routing)
.env.example            All environment variables documented
├── shared/             Cross-cutting types (stdlib only)
├── events/             SSE event catalog
├── provider/           Model provider plugin system (gRPC + protobuf)
│   └── pb/             Generated protobuf code
├── tools/              Tool registry and execution (33 tools)
├── memory/             Semantic memory with compression
├── mcp/                MCP host (stdio + SSE + streamable_http)
├── orchestration/      DAG planning and execution engine
├── disposition/        ADA personality system
├── skill/              Tiered skill executor (89 skills)
├── apps/               MCP Apps host
├── workflow/           Durable workflow engine
├── workflow-builder/   Svelte 5 SPA (workflow canvas UI)
├── chat-ui/            Svelte 5 SPA (chat interface)
├── runtime/
│   ├── actor/          Actor supervision tree
│   ├── cas/            Content-addressable storage
│   ├── d1client/       Cloudflare D1 edge sync
│   ├── event/          Event bus and routing
│   └── wasm/           WASM sandbox
├── wasm-modules/
│   └── dip-scorer/     DIP scoring (compiled to WASM)
├── server/             HTTP server, handlers, middleware, auth, streaming
│   ├── agent/          Primary agent logic
│   ├── config/         Configuration loading (YAML + env)
│   ├── database/       SQLite adapter (local + cloud)
│   ├── handlers/       Handler structs (chat, memory, models, SSE, etc.)
│   ├── middleware/     Auth, CORS, rate limiting
│   ├── migrations/     SQL migration files
│   ├── services/       Provider registry, routing, cost tracking, budget
│   │   └── providers/  Per-provider adapters (8 providers)
│   ├── streaming/      SSE streaming infrastructure
│   └── trace/          OTEL trace integration
├── specialist/         Specialist agent modules
├── integration/        Integration test harness
├── cmd/dojo/           CLI integration module
├── edge/               Cloudflare edge components
├── workers/            Cloudflare Workers
├── deploy/             VPS deployment (Caddy, systemd, provisioning)
├── deployments/        Docker/OTEL configs
├── commissions/        Pipeline commission definitions
├── contracts/          Interface contracts
├── scripts/            Utilities (drift check, skill validation, module rename)
├── docs/               Architecture docs, specs, guides
└── plugins/            Provider plugin binaries
```

## Key Commands

```bash
make build               # Build binary to bin/agentic-gateway
make test                # Run all tests with race detector
make test-cover          # Tests + coverage report (HTML)
make lint                # golangci-lint
make docker              # Build Docker image
make build-spa           # Build Workflow Builder SPA (embed into binary)
make build-chat-spa      # Build Chat UI SPA (embed into binary)
make docker-compose-up   # Full stack: gateway + OTEL collector + Langfuse + Postgres
make clean               # Remove build artifacts
```

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and set your provider API keys. The gateway loads `.env` on startup; existing environment variables take precedence.

### YAML Config

`gateway-config.yaml` controls runtime behavior:

```yaml
features:
  tool_calling: true           # Agentic tool-calling loop
  get_document_tool: true      # Document fetch endpoint
  patch_intent: true           # Extract patch intents from responses
  provider_key_management: true # Accept API keys via settings endpoint
  ollama_tool_fallback: true   # Text-mode fallback for Ollama
```

### Port

Default port is **7340**. Override with `PORT` environment variable.

The Dojo CLI connects to the gateway at `localhost:7340` by default. Set `DOJO_GATEWAY_URL` to point elsewhere.

### CORS

Set `ALLOWED_ORIGINS` to a comma-separated list of allowed origins (each must include scheme):

```
ALLOWED_ORIGINS=http://localhost:3000,https://app.example.com
```

### Observability

The gateway supports OpenTelemetry trace export and Langfuse integration:

```
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
OTEL_SERVICE_NAME=agentic-gateway
```

Use `docker-compose.example.yml` for a full observability stack (OTEL Collector + Langfuse + PostgreSQL).

## Deployment

### Docker (recommended for production)

```bash
docker build -t agentic-gateway .
docker run -p 7340:7340 --env-file .env agentic-gateway
```

The Dockerfile uses a multi-stage build: Go 1.25 Alpine builder with CGO disabled (pure-Go SQLite via `modernc.org/sqlite`), distroless runtime image, runs as non-root (UID 65534).

### VPS

See `deploy/` for production deployment files:

- `Caddyfile` -- reverse proxy with automatic TLS
- `gateway.service` -- systemd unit file
- `provision.sh` -- server provisioning script

Live deployment: `gateway.trespies.dev` (Hetzner CPX21, Caddy TLS, 4 providers active).

### Docker Compose (full stack)

```bash
docker compose -f docker-compose.example.yml up -d
```

Starts: gateway + OTEL Collector + Langfuse + PostgreSQL.

## Requirements

- Go 1.25+ (uses Go workspace with `go.work`)
- Node.js 18+ (for building embedded SPAs, optional if using pre-built binary)
- C compiler not required (pure-Go SQLite via `modernc.org/sqlite`, CGO_ENABLED=0)

## Using with the Dojo CLI

The gateway is the backend for the [Dojo CLI](https://github.com/DojoGenesis/cli). Install via Homebrew and point it at a running gateway:

```bash
brew install DojoGenesis/tap/dojo
export DOJO_GATEWAY_URL=http://localhost:7340
dojo chat
```

## License

Apache-2.0 -- see [LICENSE](LICENSE) for details.
