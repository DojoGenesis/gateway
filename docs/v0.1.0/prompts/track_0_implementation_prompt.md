# Implementation Commission: Track 0 — Foundation & Module Scaffolding

**Objective:** Fork the dojo-genesis/go_backend monolith into a new repository and restructure it as a Go workspace with 7 independently-versioned modules, stripping all Dojo-specific code.

---

## 1. Context & Grounding

**Primary Specification:**
- `docs/agentic-gateway-v0.1.0/track_0_foundation_spec.md`

**Strategic Context:**
- `docs/agentic-gateway-v0.1.0/00_strategic_scout.md`

**Source Codebase (to extract from):**
- `dojo-genesis/go_backend/` — 52,000+ lines, 162 source files, 178 test files
- Go 1.24, currently uses module path `github.com/TresPies-source/dojo-genesis/go_backend`

**Pattern Files (Follow these examples):**
- `go_backend/plugin/manager.go`: Provider module pattern (PluginManager, gRPC transport)
- `go_backend/tools/registry.go`: Tools module pattern (ToolDefinition, Registry)
- `go_backend/orchestration/engine.go`: Orchestration module pattern
- `go_backend/memory/manager.go`: Memory module pattern

**Target Module Path Convention:**
```
github.com/TresPies-source/AgenticGatewayByDojoGenesis/{module}
```

---

## 2. Detailed Requirements

1. **Create new repository** `agentic-gateway` under `github.com/dojo-genesis/` org. Initialize with Go 1.24.

2. **Create `go.work` file** at repo root listing all 7 modules:
   ```
   go 1.24
   use (
       ./shared
       ./events
       ./provider
       ./tools
       ./orchestration
       ./memory
       ./server
   )
   ```

3. **Create `shared/` module** (`github.com/TresPies-source/AgenticGatewayByDojoGenesis/shared`):
   - `shared/go.mod` with `go 1.24`
   - `shared/types.go` — Cross-cutting currency types: `Message`, `ToolCall`, `ToolResult`, `Usage`, `TaskStatus`, `NodeState` (see spec section 5.3)
   - `shared/errors.go` — Standard error types
   - No external dependencies (stdlib only)

4. **Create `events/` module** (`github.com/TresPies-source/AgenticGatewayByDojoGenesis/events`):
   - `events/go.mod` with `go 1.24`, depends on `shared`
   - `events/events.go` — `StreamEvent` type and event constructors for provider, tool, and orchestration events
   - Extract from `go_backend/events/` in the monolith
   - Unit tests for serialization/deserialization

5. **Create stub modules** with `go.mod` files for: `provider/`, `tools/`, `orchestration/`, `memory/`, `server/`. Each module gets:
   - `go.mod` with correct module path and `go 1.24`
   - A minimal `.go` file with `package {name}` declaration
   - Dependency on `shared` (and `events` where needed)

6. **Copy source packages** from monolith into new modules:
   - `go_backend/plugin/` → `provider/`
   - `go_backend/tools/` → `tools/`
   - `go_backend/orchestration/` → `orchestration/`
   - `go_backend/memory/` → `memory/`
   - `go_backend/handlers/` + `go_backend/streaming/` + `go_backend/middleware/` → `server/`

7. **Update all import paths** across all copied files:
   - FROM: `github.com/TresPies-source/dojo-genesis/go_backend/...`
   - TO: `github.com/TresPies-source/AgenticGatewayByDojoGenesis/...`

8. **Strip Dojo-specific packages** — Remove ALL code referencing:
   - `calibration/`, `compassion/`, `proactive/`, `telegram/`, `judgment/`, `goals/`, `context/` (the engine), `entities/`, `workspaces/`
   - Trace all imports of these packages and remove dependent code paths

9. **Verify dependency graph** has no circular imports:
   ```
   shared → (stdlib only)
   events → shared
   provider → shared, events
   tools → shared, events
   orchestration → shared, tools, events
   memory → shared
   server → shared, events, provider, tools, orchestration, memory
   ```

10. **Set up CI/CD** — Create `.github/workflows/ci.yml`:
    - Trigger on push to `main` and PRs
    - Run `go build ./...` from workspace root
    - Run `go test ./...` from workspace root
    - Run `go vet ./...`
    - Run `go test -race ./...`

11. **Create README.md** at repo root with:
    - Project name and one-line description
    - Architecture diagram (7 modules with dependency arrows)
    - Quickstart stub (how to import a single module)
    - Build instructions

---

## 3. File Manifest

**Create:**
- `go.work`
- `README.md`
- `.github/workflows/ci.yml`
- `shared/go.mod`, `shared/types.go`, `shared/errors.go`
- `events/go.mod`, `events/events.go`, `events/events_test.go`
- `provider/go.mod` + copied source from `plugin/`
- `tools/go.mod` + copied source from `tools/`
- `orchestration/go.mod` + copied source from `orchestration/`
- `memory/go.mod` + copied source from `memory/`
- `server/go.mod` + copied source from `handlers/`, `streaming/`, `middleware/`

**Delete (from copied source):**
- All files importing `calibration`, `compassion`, `proactive`, `telegram`, `judgment`, `goals`, `context` (engine), `entities`, `workspaces`

---

## 4. Success Criteria

- [ ] `go.work` file exists at workspace root with all 7 modules listed
- [ ] Each module's `go.mod` uses `github.com/TresPies-source/AgenticGatewayByDojoGenesis/{module}` path with `go 1.24`
- [ ] `go build ./...` from workspace root succeeds without errors
- [ ] `go test ./...` from workspace root runs all tests successfully
- [ ] `go vet ./...` passes without warnings
- [ ] `go test -race ./...` detects no data races
- [ ] Zero compilation errors mentioning stripped Dojo packages
- [ ] `shared/types.go` exports: Message, ToolCall, ToolResult, Usage, TaskStatus, NodeState
- [ ] `events/events.go` exports: StreamEvent type and constructors
- [ ] No circular import dependencies (verified by successful build)
- [ ] GitHub Actions CI workflow triggers and passes on push
- [ ] README.md exists with project overview and build instructions

---

## 5. Constraints & Non-Goals

- **DO NOT** implement new functionality — this is pure extraction and modularization
- **DO NOT** rewrite internal module code — internal cleanup happens in Tracks 1-2
- **DO NOT** implement the OpenAI-compatible API — that's Track 3
- **DO NOT** build a documentation site — basic README only
- **DO NOT** set up independent module versioning — all modules ship as v0.1.0 together
- **DO NOT** add external dependencies not already in the monolith (except for shared/events which are new)

---

## 6. Source Codebase Reference

**Source Repository:** `github.com/TresPies-source/dojo-genesis` (private)
**Source Directory:** `go_backend/`
**Go Version:** 1.24
**Key Source Files:**
- `go_backend/plugin/manager.go` — PluginManager (→ provider/)
- `go_backend/tools/registry.go` — ToolRegistry (→ tools/)
- `go_backend/orchestration/engine.go` — Orchestration Engine (→ orchestration/)
- `go_backend/memory/manager.go` — MemoryManager (→ memory/)
- `go_backend/streaming/streaming_agent.go` — SSE Streaming (→ server/)
- `go_backend/handlers/` — HTTP Handlers (→ server/)
- `go_backend/events/` — Event Types (→ events/)

**Package-to-Module Mapping:**

| Source Package | Target Module | Notes |
|---------------|--------------|-------|
| `plugin/` | `provider/` | Rename package |
| `tools/` | `tools/` | Direct copy |
| `orchestration/` | `orchestration/` | Direct copy |
| `memory/` | `memory/` | Direct copy |
| `events/` | `events/` | Standalone module (was in streaming/) |
| `handlers/` | `server/` | Merge into server/ |
| `streaming/` | `server/` | Merge into server/ |
| `middleware/` | `server/` | Merge into server/ |
