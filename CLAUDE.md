# dojo-AgenticGateway — CLAUDE.md (nexus)

## Orientation

**DojoGenesis** · **AgenticGatewayByDojoGenesis** · production-ready · v3.2.0 / server v1.1.0 · Era 3 complete, Era 4 Phase 0 spec written.

Self-hosted agentic AI runtime: multi-provider LLM routing, DAG orchestration, content-addressable skill storage (CAS), MCP host, real-time SSE, portal auth — one Go binary. Hard dependency for the whole DojoGenesis platform. Primary client is the `dojo` CLI at `C:\Users\cruzr\cli\`.

**Folder / canonical-name note:** This Windows checkout lives at `C:\Users\cruzr\dojo-AgenticGateway\`. The canonical registry name is `AgenticGatewayByDojoGenesis`. GitHub remotes are `origin → TresPies-source/AgenticGatewayByDojoGenesis` and `dojogenesis → DojoGenesis/gateway`. Go module path (confirmed in `go.mod`): `github.com/DojoGenesis/gateway`.

Workspace nexus is NOT auto-loaded here (flat Windows layout). Read `C:\Users\cruzr\zenflow-projects\CLAUDE.md` for cross-repo context.

---

## Hot facts / FAQ

**Port: 7340 — not 8080.**
Hardcoded default in `server/config/config.go` `loadDefaults()`. Override via `PORT` env var or `config/config.yaml`. Health check: `GET http://localhost:7340/health`. Chat: `POST http://localhost:7340/v1/chat/completions`.

**TWO INDEPENDENT HANDLER PATHS — MAP ROUTES BEFORE TOUCHING ANY HANDLER.**
The gateway has two distinct handler registration paths on different routes. Before touching any handler file, grep all route registrations first:
- `server/router.go` `setupRoutes()` — Gin routes: `/v1/*`, `/api/*`, `/admin/*`, `/auth/*`, `/chat/*`, `/workflow/*`, `/mesh/*`
- `workflow/api/` — `http.NewServeMux` wrapped via `gin.WrapH` for `/api/workflows/*` and `/api/skills`

These two paths do NOT share middleware or error-handling. A handler that looks unreachable from one path may be live on the other. **Grep `s.router\.` and `wfHandler\.RegisterRoutes` before any handler work — this gotcha has burned multiple sessions.**

**Start:**
```
make build          # go build -o bin/agentic-gateway main.go
./bin/agentic-gateway
```
Or directly: `go run main.go`

**Build / test gates:**
```
go build ./...      # must pass before any PR
go test ./...       # 156 test files; run with -race: go test -race ./...
make test           # equivalent (adds -race automatically)
go vet ./...
```

**Config loading order:** `.env` (lowest) → `config/config.yaml` → env vars (highest). Port default is in `loadDefaults()` — `"7340"`.

**Memory DB:** defaults to `~/.dojo/memory.db` (absolute). Override via `MEMORY_DB_PATH`. A relative path here silently creates a new empty DB on working-directory changes — do not use relative paths.

**CAS (skills/workflows):** defaults to `~/.dojo/skills.db`. Override via `DOJO_CAS_PATH`.

**Auth DB:** defaults to `.dojo/dojo.db` (relative to CWD). Override via `AUTH_DB_DIR`.

**MCP config:** loaded from `gateway-config.yaml` if present. Override via `MCP_CONFIG_PATH`.

**MCP Apps:** disabled by default. Enable via `MCP_APPS_ENABLED=true`.

**Remotes:** `origin → TresPies-source/AgenticGatewayByDojoGenesis` · `dojogenesis → DojoGenesis/gateway`. Push to DojoGenesis org requires `gh auth switch` to DojoGenesis account.

---

## Router

| Task | Go to |
|------|-------|
| Go conventions, cross-repo dispatch rules | `C:\Users\cruzr\zenflow-projects\CLAUDE.md` |
| Primary client (CLI) | `C:\Users\cruzr\cli\` |
| Gateway ADRs (28 total) | `C:\Users\cruzr\AgenticStackOrchestration\decisions\` |
| Local gateway decisions | `C:\Users\cruzr\dojo-AgenticGateway\decisions\` |
| Platform era roadmap, test health | `C:\Users\cruzr\TresPies-AI-Orchestration\project_agentic_stack.md` |
| Architecture deep-doc | `ARCHITECTURE.md` (this repo) |
| Deployment / Docker | `DEPLOYMENT.md` (this repo) |
| Era 4 Phase 0 spec | `C:\Users\cruzr\AgenticStackOrchestration\` |

---

## Rules that bite here

- **Two-handler-path rule (above):** grep `s.router.` AND `wfHandler.RegisterRoutes` before any handler work. Failing this is the single biggest source of handler bugs in this repo.
- **Port 7340:** any config, test, healthcheck, or curl example that references 8080 is wrong.
- **Memory/CAS absolute paths:** relative DB paths create silent duplicate databases on deploy restarts. Always use absolute paths or the env-var overrides.
- **DojoGenesis push needs account switch:** `gh auth switch --user DojoGenesis` (or equivalent) before pushing to `dojogenesis` remote. The `origin` remote pushes to TresPies-source.
- **SPA must be built before binary:** `make build-spa` (Workflow Builder) and `make build-chat-spa` (Chat UI) must run before `make build` if those SPAs have changed, or the embedded dist will be stale.

---

## Map

```
main.go              entry point — wires all subsystems, starts server
server/
  server.go          Server struct, Start/Stop
  router.go          ALL Gin route registration — read this first
  router_gaps_patch.go  doc artifact — route correction history (Gap 1,3,5)
  config/config.go   Config struct + loadDefaults() — port 7340 lives here
  handle_*.go        handler files (one concern per file)
  handlers/          additional handler structs (chat, memory, health, etc.)
  middleware/        auth, logging, security headers
orchestration/       standalone orchestration engine (DAG, planner)
workflow/api/        http.ServeMux workflow handler — SECOND handler path
mcp/                 MCP host manager (stdio + SSE + streamable_http)
memory/              SQLite-backed memory + garden manager
cas/                 content-addressable skill storage
provider/            LLM provider plugin system (8 providers)
apps/                MCP Apps bridge (feature-flagged)
decisions/           local ADRs (one file currently: 015-addendum)
bin/                 compiled binary output
```

---

## Bookend

This repo participates in the platform convergence cadence. After any multi-session implementation wave: deposit a distillate to `C:\Users\cruzr\AgenticStackOrchestration\` (decisions) and update `C:\Users\cruzr\TresPies-AI-Orchestration\project_agentic_stack.md` with era/test-health status. Transcripts never; decisions and test-health snapshots always.

Active priority as of 2026-06-11: Era 4 Phase 0 — see `AgenticStackOrchestration\` for spec.
