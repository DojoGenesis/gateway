# Implementation Commission: Track 3 — Server + API Surface

**Objective:** Implement the HTTP server that wires all framework modules together, exposing an OpenAI-compatible API surface and a richer agentic API for orchestration, tool management, and memory management, with SSE streaming support.

**Depends On:** Track 0, Track 1, and Track 2 must ALL be complete — server/ imports all other modules.

---

## 1. Context & Grounding

**Primary Specification:**
- `docs/agentic-gateway-v0.1.0/track_3_server_api_spec.md`

**Foundation Specification:**
- `docs/agentic-gateway-v0.1.0/track_0_foundation_spec.md` (for dependency graph)

**Pattern Files (Follow these examples from monolith):**
- `go_backend/handlers/chat.go`: Chat completion handler pattern
- `go_backend/handlers/streaming.go`: SSE streaming handler pattern
- `go_backend/streaming/streaming_agent.go`: Broadcaster pattern for SSE
- `go_backend/middleware/`: Auth, logging, CORS middleware patterns

**Module Path:**
- `github.com/TresPies-source/AgenticGatewayByDojoGenesis/server`

**HTTP Framework:** Gin (`github.com/gin-gonic/gin`)

---

## 2. Detailed Requirements

### Server Core

1. **Implement `Server` struct** in `server/server.go`:
   - Constructor: `New(cfg, providerRouter, engine, planner, toolRegistry, memoryManager, broadcaster) *Server`
   - Dependency injection — server receives all modules, doesn't create them
   - `Start(addr) error` — start HTTP server
   - `Shutdown(ctx) error` — graceful shutdown with 30s timeout
   - `setupMiddleware()` and `setupRoutes()` internal methods

2. **Implement middleware stack** in `server/middleware.go`:
   - `middlewareRequestID()` — generate and attach X-Request-ID header
   - `middlewareLogging()` — log method, path, duration, status
   - `middlewareCORS(origins)` — configurable CORS using `gin-contrib/cors`
   - `middlewareAuth(mode)` — Bearer token auth (configurable: none, api_key, custom)
   - `middlewareRecovery()` — panic recovery with error logging
   - Middleware order: Recovery → RequestID → Logging → CORS → Auth

3. **Implement `ServerConfig`** in `server/config.go`:
   - Port, AllowedOrigins, AuthMode, Environment, ShutdownTimeout
   - Support YAML config file + environment variable overrides
   - Sensible defaults: port 8080, auth api_key, shutdown 30s

### OpenAI-Compatible Endpoints

4. **POST /v1/chat/completions** in `server/openai_api.go`:
   - Accept OpenAI-format request body (model, messages, temperature, max_tokens, stream, tools)
   - Non-streaming: call `provider.GenerateCompletion()`, return OpenAI-format response
   - Streaming: call `provider.GenerateCompletionStream()`, return SSE chunks in OpenAI format
   - Response shapes must pass OpenAI SDK validation (Python `openai>=1.0.0`, Node `openai>=4.0.0`)

5. **GET /v1/models** in `server/openai_api.go`:
   - Call `provider.ListModels()` across all enabled providers
   - Return OpenAI-format model list with id, object, created, owned_by fields

### Agentic Endpoints

6. **POST /v1/orchestrate** in `server/agentic_api.go`:
   - Accept: task_description, user_id, session_id, timeout_seconds, max_replanning_attempts
   - Create Task, submit to Engine for background execution
   - Return immediately with orchestration_id and status "planning"
   - Execute in goroutine; emit events via broadcaster

7. **GET /v1/orchestrate/:id/events** in `server/agentic_api.go`:
   - SSE stream of orchestration lifecycle events
   - Events: plan_created, node_start, tool_invoked, tool_completed, node_end, replanning, complete, failed
   - Support `follow` query param (default true — keep connection until complete)
   - Register SSE client with broadcaster, stream events

8. **Tool Management API** in `server/tools_api.go`:
   - `GET /v1/tools` — list all registered tools (name, description, parameters)
   - `GET /v1/tools/:name` — get tool details with examples
   - `POST /v1/tools/:name/invoke` — direct tool invocation (bypass orchestration)
   - `POST /v1/tools` — register custom tool (advanced)
   - `PUT /v1/tools/:name` — update tool definition
   - `DELETE /v1/tools/:name` — remove tool

9. **Memory Management API** in `server/memory_api.go`:
   - `POST /v1/memory` — store memory (type, content, metadata, context_type)
   - `GET /v1/memory` — list/search memories (with query params)
   - `GET /v1/memory/:id` — get specific memory
   - `PUT /v1/memory/:id` — update memory
   - `DELETE /v1/memory/:id` — delete memory
   - `POST /v1/memory/search` — semantic search (query, limit)

### Infrastructure Endpoints

10. **GET /health** — health check (server + module health)
11. **GET /metrics** (optional) — Prometheus-style metrics

### SSE Streaming

12. **Implement `Broadcaster`** in `server/streaming.go`:
    - `NewBroadcaster()` constructor
    - `Register(clientID) <-chan events.StreamEvent` — register SSE client
    - `Unregister(clientID)` — remove client, close channel
    - `Broadcast(event)` — fan-out to all registered clients
    - Support 10,000+ concurrent connections
    - Automatic cleanup on client disconnect
    - Thread-safe with mutex

---

## 3. File Manifest

**Create:**
- `server/server.go` — Server struct, constructor, Start, Shutdown
- `server/config.go` — ServerConfig with YAML + env var support
- `server/routes.go` — Route group setup (/v1, /health, /metrics)
- `server/middleware.go` — All middleware functions
- `server/openai_api.go` — POST /v1/chat/completions, GET /v1/models
- `server/agentic_api.go` — POST /v1/orchestrate, GET /v1/orchestrate/:id/events
- `server/tools_api.go` — GET/POST/PUT/DELETE /v1/tools/*
- `server/memory_api.go` — GET/POST/PUT/DELETE /v1/memory/*
- `server/streaming.go` — Broadcaster for SSE
- `server/*_test.go` — Unit tests for each handler
- `server/testutil/` — Test helpers (mock providers, tools, memory)

---

## 4. Success Criteria

- [ ] `cd server && go build` succeeds
- [ ] `go test ./server/...` passes
- [ ] `go test -race ./server/...` detects no data races
- [ ] POST /v1/chat/completions returns valid OpenAI-format response (non-streaming)
- [ ] GET /v1/models returns OpenAI-format model list
- [ ] Streaming mode sends SSE chunks in OpenAI format (data: prefix, [DONE] terminator)
- [ ] Python `openai` SDK works with `base_url="http://localhost:8080/v1"`
- [ ] Node `openai` SDK works with `baseURL: "http://localhost:8080/v1"`
- [ ] POST /v1/orchestrate returns orchestration_id immediately
- [ ] GET /v1/orchestrate/:id/events streams lifecycle events via SSE
- [ ] Tool CRUD endpoints work (register, list, get, invoke, update, delete)
- [ ] Memory CRUD endpoints work (store, list, get, update, delete, search)
- [ ] GET /health returns 200 with module status
- [ ] Graceful shutdown completes within 30 seconds
- [ ] Middleware: Request ID header present, auth rejects invalid tokens, CORS headers set

---

## 5. Constraints & Non-Goals

- **DO NOT** implement real LLM provider backends — use mock providers for testing
- **DO NOT** implement database migrations — use in-memory backends
- **DO NOT** implement WebSocket transport — SSE only for v0.1.0
- **DO NOT** implement rate limiting — future work
- **DO NOT** implement multi-tenant isolation — future work
- **DO NOT** modify provider/, tools/, orchestration/, or memory/ modules — use their public APIs only
- **DO NOT** implement a documentation site or Swagger/OpenAPI spec generation

---

## 6. Source Codebase Reference

**Source Files to Extract From:**
- `go_backend/handlers/chat.go` → `server/openai_api.go`
- `go_backend/handlers/streaming.go` → `server/streaming.go`
- `go_backend/streaming/streaming_agent.go` → `server/streaming.go`
- `go_backend/middleware/` → `server/middleware.go`
- `go_backend/handlers/tools.go` → `server/tools_api.go` (if exists)

**Dependencies:**
- `server` → `shared`, `events`, `provider`, `tools`, `orchestration`, `memory`
- External: `github.com/gin-gonic/gin`, `github.com/gin-contrib/cors`

**Import Paths (all must use):**
```go
import (
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/shared"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/events"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
)
```
