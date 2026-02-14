# Prompt: Medium & Low Priority Fixes

Use this prompt to address the remaining medium and low priority issues identified in the architecture audit. All critical and high items have been resolved.

---

## Context

You are working on AgenticGatewayByDojoGenesis, a Go workspace with 9 modules. The build is currently clean (`go build ./...` and `go vet ./...` pass). Critical and high priority fixes are complete:
- err.Error() leaks fixed in handlers, user_preferences.go, budget middleware
- Sentinel error pattern for ErrAgentNotFound
- SQLite WAL mode + connection pooling configured in memory/manager.go
- Docker healthcheck using --health-check binary flag
- Pagination upper bound in agents.go

## Medium Priority Fixes

### M1: Fix err.Error() leaks in server/apptools/

Nine instances of `fmt.Sprintf("failed to...: %v", err)` in tool result messages that may be surfaced to clients:

**Files:**
- `server/apptools/artifact_tools.go` — 4 instances
- `server/apptools/project_tools.go` — 4 instances
- `server/apptools/memory_tools.go` — 1 instance

**Pattern to apply:**
```go
// BEFORE:
return tools.NewToolResult(fmt.Sprintf("failed to store artifact: %v", err))

// AFTER:
slog.Error("failed to store artifact", "error", err)
return tools.NewToolResult("failed to store artifact")
```

Add `"log/slog"` import where needed.

### M2: Fix unsafe type assertions in chat.go and trace.go

**server/handlers/chat.go line 336:**
```go
// BEFORE:
content := event.Data["content"].(string)

// AFTER:
content, _ := event.Data["content"].(string)
```

**server/handlers/chat.go line 356:**
```go
// Add nil/type check for event.Data["error"]
```

**server/handlers/trace.go line 207:**
```go
// BEFORE:
spanID := span["span_id"].(string)

// AFTER:
spanID, _ := span["span_id"].(string)
```

### M3: Apply SQLite tuning to server database

The server's `LocalAdapter` receives a pre-opened `*sql.DB` without PRAGMA tuning. Find where the server DB is opened (likely in main.go or a database initialization function) and apply the same tuning as memory/manager.go:

```go
pragmas := []string{
    "PRAGMA journal_mode=WAL",
    "PRAGMA synchronous=NORMAL",
    "PRAGMA cache_size=-64000",
    "PRAGMA foreign_keys=ON",
    "PRAGMA busy_timeout=5000",
}
db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)
```

### M4: Add security headers middleware

Create `server/middleware/security_headers.go`:
```go
func SecurityHeadersMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Next()
    }
}
```

Add it to the middleware chain in server.go after Recovery, before RateLimiting.

### M5: Extract orchestration from root module dependency

See `docs/architecture-decisions.md` section 3 for the full strategy. The key change:
1. Create `interfaces/go.mod` with `ToolDefinition` type and gateway interfaces
2. Update orchestration/go.mod to depend on `interfaces/` instead of root
3. Update go.work to include `interfaces/`
4. This removes the transitive chromedp + go-chart dependency from orchestration

## Low Priority Fixes

### L1: Adopt `respondError` helper consistently

`server/handlers/types.go` defines `respondError()` and `respondValidationError()` but they're only used in 2 files. Migrate all handlers to use these helpers instead of inline `c.JSON(statusCode, gin.H{"error": "..."})` for consistent error response format.

### L2: Server constructor → options struct

Replace the 18-parameter `New()` and `NewFromConfig()` with:
```go
type ServerDeps struct {
    Config              *ServerConfig
    PluginManager       *provider.PluginManager
    OrchestrationEngine *orchestrationpkg.Engine
    Planner             orchestrationpkg.PlannerInterface
    MemoryManager       *memory.MemoryManager
    // ... remaining fields
}

func New(deps ServerDeps) *Server { ... }
```

### L3: Handler struct pattern

Replace global variable injection:
```go
// BEFORE:
var agentManager *agent.AgentManager
func InitializeAgentHandlers(am *agent.AgentManager) { ... }
func HandleGetAgent(c *gin.Context) { ... }

// AFTER:
type AgentHandler struct {
    manager *agent.AgentManager
}
func NewAgentHandler(am *agent.AgentManager) *AgentHandler { ... }
func (h *AgentHandler) GetAgent(c *gin.Context) { ... }
```

This is a significant refactor that touches route registration in router.go.

### L4: Add Strict-Transport-Security in production

In the security headers middleware, add HSTS only when environment is production:
```go
if c.GetString("environment") == "production" {
    c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
}
```

### L5: Rename memory/compat.go local Message type

`memory/context_builder.go` defines a local `Message` struct that shadows `shared.Message`. Rename to `ContextMessage` for clarity. This is purely cosmetic.

## Verification

After all changes:
```bash
go build ./...
go vet ./...
go test ./... -count=1 -race
```
