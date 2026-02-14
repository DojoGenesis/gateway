# Architecture Decisions, Debugging Narrative & Strategy

## 1. Curlable Binary Assessment

### Question
Should we ship a separate curlable binary (like `wget` or `curl`) alongside the gateway binary for healthchecks?

### Assessment: No — The `--health-check` flag is the correct approach.

**Why a curlable binary is wrong for this project:**

| Factor | Curlable Binary | `--health-check` flag |
|--------|----------------|----------------------|
| Image size | +5-10MB (static curl/wget) | +0 bytes (already compiled in) |
| Attack surface | Additional binary = additional CVE surface | No additional binaries |
| Distroless compatibility | Requires switching to Alpine or adding binary | Native — no shell needed |
| Maintenance | Separate binary to version/update | Same binary, same version |
| Port awareness | Hardcoded or env-var parsing | Reads PORT env natively |

**The `--health-check` flag we implemented:**
```go
if len(os.Args) > 1 && os.Args[1] == "--health-check" {
    resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port))
    // exit 0 or 1
}
```

This is the pattern used by production Go services (Consul, Vault, Traefik) — the binary self-probes. It costs zero additional bytes, works in distroless, and has zero dependency on external tools.

**Only scenario where a curlable binary wins:** If you need to healthcheck *other* services from the same container. This gateway only healthchecks itself.

---

## 2. Health Endpoint Strategy Assessment

### Options Evaluated

| Option | Image Size | Security | Complexity | Verdict |
|--------|-----------|----------|-----------|---------|
| A. `--health-check` on distroless (current) | ~8MB | Best | Minimal | **Selected** |
| B. Switch to Alpine + curl | ~15MB | Good | Low | Unnecessary |
| C. `HEALTHCHECK NONE` + K8s probes | 0 overhead | Best | Medium | Rejected by requirement |
| D. Embedded separate HTTP listener | 0 overhead | Good | High | Over-engineered |

### Decision: Keep `--health-check` flag on distroless

**Rationale:**
- Alpine adds ~7MB and a full shell (attack surface)
- The current `--health-check` flag solves Docker Compose, Docker Swarm, and standalone Docker
- For Kubernetes: use `httpGet` liveness/readiness probes directly against `/health:8080` — the flag isn't needed there
- An embedded secondary HTTP listener (option D) adds complexity for zero benefit since the main server already serves `/health`

**Docker Compose (current, working):**
```yaml
healthcheck:
  test: ["CMD", "/agentic-gateway", "--health-check"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

**Kubernetes (future, no flag needed):**
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 40
  periodSeconds: 30
readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

---

## 3. Orchestration Dependency Strategy

### Problem
`orchestration/go.mod` transitively pulls in chromedp (browser automation) and go-chart (charting) through the tools/ module. These are heavy, inappropriate dependencies for an orchestration engine.

### Root Cause Analysis
```
orchestration/
  requires → root module (github.com/.../AgenticGatewayByDojoGenesis)
    contains → pkg/gateway/interfaces.go
      imports → tools.ToolDefinition
        requires → chromedp, go-chart
```

**Key finding:** Orchestration has ZERO direct code dependency on tools/. It never imports `tools.*` anywhere. The entire dependency chain exists because:
1. Orchestration imports `pkg/disposition` and `pkg/skill` from the root module
2. The root module also contains `pkg/gateway/interfaces.go` which imports `tools`
3. Go modules are all-or-nothing — requiring the root module pulls in all its transitive dependencies

### Strategy: Extract `pkg/` into a standalone `interfaces/` module

**Phase 1 — Extract interfaces (recommended now):**
Create a new `interfaces/` module that contains only type definitions and interfaces:

```
interfaces/
  go.mod          # minimal deps: only shared/
  gateway.go      # ToolRegistry, OrchestrationExecutor, MemoryStore, AgentInitializer
  disposition.go  # DispositionConfig, AgentInitializer
  skill.go        # SkillRegistry, SkillExecutor
  tool_types.go   # ToolDefinition (moved from tools/)
```

Then orchestration depends on `interfaces/` instead of the root module.

**Phase 2 — Restructure tools/ (optional, later):**
Split tools/ into:
- `tools/core/` — ToolDefinition, ToolResult types (zero external deps)
- `tools/browser/` — chromedp-based tools
- `tools/chart/` — go-chart-based tools

**Effort estimate:** Phase 1 is ~2-3 hours of careful refactoring. Phase 2 is a larger module restructure.

**Why not just use build tags?** Build tags hide the dependency from the binary but not from `go.sum`. The module graph still downloads chromedp. True decoupling requires module separation.

---

## 4. Key Decision Errors & Debugging Narrative

### Decision Error #1: Bulk `replace_all` for err.Error() fixes (Previous Session)

**What happened:** In the previous session, we used `replace_all` to fix `err.Error()` leaks across handler files. This caused:
- Broken `gin.H{}` map formatting in some files
- Accidental removal of the `fmt` import from `user_preferences.go` (the handlers stopped using `fmt` but the internal DB functions still needed it)
- Required manual cleanup of malformed JSON response literals

**Lesson learned:** `replace_all` is dangerous for patterns that appear in different structural contexts. Each `err.Error()` usage had different surrounding code. Targeted per-site edits are safer, even if slower.

**Debugging steps:**
1. `go build ./...` immediately caught the broken syntax
2. Manual file review identified the `gin.H` formatting issue
3. Adding back `fmt` import fixed the user_preferences compilation

### Decision Error #2: Healthcheck with curl on distroless (Original Design)

**What happened:** The original `docker-compose.yaml` used `curl -f http://localhost:8080/health` as the healthcheck. Distroless images have no shell, no curl, no wget.

**How we caught it:** Architecture audit flagged that `gcr.io/distroless/static-debian12` contains only the binary, CA certs, and timezone data. Any shell command will fail silently (Docker marks the container as unhealthy without useful error output).

**Resolution:** Implemented `--health-check` flag in the binary itself. The binary makes an HTTP GET to its own `/health` endpoint and exits 0/1. This is self-contained and works in any runtime.

### Decision Error #3: String-based error comparison for agent not found

**What happened:** `agents.go` handlers compared `err.Error() == "agent not found"` — a fragile string comparison that would break if the error message changed or if the error was wrapped with `fmt.Errorf("...: %w", err)`.

**Root cause:** The `agent/registry.go` originally returned `fmt.Errorf("agent not found")` — an ad-hoc error with no stable identity. The handler had no choice but to compare strings.

**Resolution:**
1. Added `var ErrAgentNotFound = errors.New("agent not found")` to registry.go
2. Changed handlers to use `errors.Is(err, agent.ErrAgentNotFound)` which handles wrapping correctly
3. Renamed local `agent` variable to `agentResult` to avoid shadowing the package import

---

## 5. Server Architecture Assessment

### Strengths

**Clean dependency injection:** The 18-parameter `New()` constructor is explicit. Every dependency is visible. No global service locator, no init() magic. While 18 parameters is a code smell, it's preferable to hidden coupling.

**Middleware chain ordering is correct:**
1. Recovery (catch panics) — must be first
2. RateLimiting (reject before work) — correct early position
3. CORS (browser preflight) — before auth
4. RequestID (tracing) — before logging
5. Logging (observe) — after ID assigned
6. Auth (identify) — after logging so failed auth is logged
7. Budget (enforce) — after auth so user is identified

**SSE streaming support:** `WriteTimeout: 0` is intentional and correct for Server-Sent Events. The comment documents why. `ReadTimeout: 15s` and `IdleTimeout: 120s` still protect against slow clients.

### Concerns

**18-parameter constructor:** Should migrate to an options struct:
```go
type ServerDeps struct {
    PluginManager       *provider.PluginManager
    OrchestrationEngine *orchestrationpkg.Engine
    // ...
}
```
This is a medium-priority refactor — the current code works, but it's hard to extend.

**Global handler initialization pattern:**
```go
var agentManager *agent.AgentManager
func InitializeAgentHandlers(am *agent.AgentManager) { ... }
```
This package-level state makes testing harder and creates implicit coupling. A handler struct with methods would be cleaner, but this is a larger refactor with test implications.

---

## 6. Database Layer Assessment

### Current State
- **Two SQLite databases:** Memory DB (`dojo_memory.db`) and Server DB (`.dojo/dojo.db`)
- **Driver:** `modernc.org/sqlite` (pure Go, CGO_ENABLED=0) — correct for distroless
- **Connection pooling:** Now configured (SetMaxOpenConns=1 for SQLite's single-writer model)
- **WAL mode:** Now enabled with full PRAGMA tuning
- **Foreign keys:** Enabled in migrations but were NOT enabled at connection level — now fixed via PRAGMA

### SQLite Tuning Applied (memory/manager.go)
```go
PRAGMA journal_mode=WAL          // Concurrent readers, single writer
PRAGMA synchronous=NORMAL        // Safe with WAL, 2x faster than FULL
PRAGMA cache_size=-64000         // 64MB page cache
PRAGMA foreign_keys=ON           // Enforce referential integrity
PRAGMA busy_timeout=5000         // Wait 5s on lock contention

db.SetMaxOpenConns(1)            // SQLite is single-writer
db.SetMaxIdleConns(1)            // Keep connection warm
db.SetConnMaxLifetime(0)         // Never expire (SQLite file is local)
```

**Why SetMaxOpenConns=1:** SQLite serializes all writes through a single file lock. Multiple connections create contention without benefit. WAL mode allows concurrent reads from the same connection, so one connection is optimal for write-heavy workloads.

### Remaining Gap: Server DB
The server's `LocalAdapter` receives an already-opened `*sql.DB` without PRAGMA tuning or pool configuration. The same tuning should be applied wherever that DB is opened. This is a medium-priority item — see the prompt in section 7.
