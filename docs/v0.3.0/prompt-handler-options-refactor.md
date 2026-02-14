# Prompt: Server Options Struct + Handler Struct Refactor

## Context for the Implementing Agent

You are refactoring the HTTP server layer of a Go project (AgenticGatewayByDojoGenesis). The server uses Gin framework. There are two architectural code smells to fix:

1. **18-parameter constructor** — `server.New()` and `server.NewFromConfig()` accept 18 positional parameters
2. **13 global variable injection sites** — `server/handlers/` uses package-level `var` + `Initialize*()` functions instead of struct methods

**IMPORTANT:** This is a large refactor touching ~35 files across two phases. Phase 1 (options struct) is a prerequisite for Phase 2 (handler structs). Run `go build ./...` after completing each numbered step as a gate. Phase 2 should be done handler-by-handler, testing after each one.

---

## Phase 1: ServerDeps Options Struct

### Goal

Replace the 18-parameter `New()` and `NewFromConfig()` constructors with a single `ServerDeps` struct.

### Step 1.1: Create `server/deps.go` (NEW FILE)

```go
package server

import (
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/maintenance"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
)

// ServerDeps holds all injectable dependencies for the Server.
// This replaces the 18-parameter New() constructor with a single struct.
type ServerDeps struct {
	Config              *ServerConfig
	PluginManager       *provider.PluginManager
	OrchestrationEngine *orchestrationpkg.Engine
	Planner             orchestrationpkg.PlannerInterface
	MemoryManager       *memory.MemoryManager
	GardenManager       *memory.GardenManager
	PrimaryAgent        *agent.PrimaryAgent
	IntentClassifier    *agent.IntentClassifier
	UserRouter          *services.UserRouter
	TraceLogger         *trace.TraceLogger
	CostTracker         *services.CostTracker
	BudgetTracker       *services.BudgetTracker
	MemoryMaintenance   *maintenance.MemoryMaintenance
	ToolRegistry        gateway.ToolRegistry
	AgentInitializer    gateway.AgentInitializer
	MCPHostManager      MCPStatusProvider
	OrchestrationExec   gateway.OrchestrationExecutor
	MemoryStore         gateway.MemoryStore
}
```

### Step 1.2: Modify `server/server.go`

**BEFORE** (lines 85-153 — the full `New()` function signature):
```go
func New(
	cfg *ServerConfig,
	pm *provider.PluginManager,
	orch *orchestrationpkg.Engine,
	plan orchestrationpkg.PlannerInterface,
	mm *memory.MemoryManager,
	gm *memory.GardenManager,
	pa *agent.PrimaryAgent,
	ic *agent.IntentClassifier,
	ur *services.UserRouter,
	tl *trace.TraceLogger,
	ct *services.CostTracker,
	bt *services.BudgetTracker,
	maint *maintenance.MemoryMaintenance,
	toolReg gateway.ToolRegistry,
	agentInit gateway.AgentInitializer,
	mcpMgr MCPStatusProvider,
	orchExec gateway.OrchestrationExecutor,
	memStore gateway.MemoryStore,
) *Server {
```

**AFTER:**
```go
func New(deps ServerDeps) *Server {
	cfg := deps.Config
```

Then update ALL field assignments in the function body. Replace each positional parameter reference:

```go
	s := &Server{
		router:                gin.New(),
		cfg:                   cfg,
		pluginManager:         deps.PluginManager,
		orchestrationEngine:   deps.OrchestrationEngine,
		planner:               deps.Planner,
		memoryManager:         deps.MemoryManager,
		gardenManager:         deps.GardenManager,
		primaryAgent:          deps.PrimaryAgent,
		intentClassifier:      deps.IntentClassifier,
		userRouter:            deps.UserRouter,
		traceLogger:           deps.TraceLogger,
		costTracker:           deps.CostTracker,
		budgetTracker:         deps.BudgetTracker,
		memoryMaintenance:     deps.MemoryMaintenance,
		toolRegistry:          deps.ToolRegistry,
		agentInitializer:      deps.AgentInitializer,
		mcpHostManager:        deps.MCPHostManager,
		orchestrationExecutor: deps.OrchestrationExec,
		memoryStore:           deps.MemoryStore,
		orchestrations:        NewOrchestrationStore(),
		agents:                make(map[string]*gateway.AgentConfig),
	}
```

Also update the nil-check for `cfg` at the top:
```go
func New(deps ServerDeps) *Server {
	cfg := deps.Config
	if cfg == nil {
		cfg = &ServerConfig{
			Port:            "8080",
			AllowedOrigins:  []string{"http://localhost:3000"},
			AuthMode:        "api_key",
			Environment:     "production",
			ShutdownTimeout: 30 * time.Second,
		}
	}
	// ... rest unchanged
```

And update the budget middleware check to use `deps`:
```go
	if deps.BudgetTracker != nil && deps.CostTracker != nil {
		s.router.Use(middleware.BudgetMiddleware(deps.BudgetTracker, deps.CostTracker))
	}
```

Wait — actually the budget check uses `s.budgetTracker` and `s.costTracker` which are already set on the struct. Keep the existing check as `s.budgetTracker` and `s.costTracker` — it's in `setupMiddleware()` which runs after struct initialization.

### Step 1.3: Remove `NewFromConfig()`

**Delete the entire `NewFromConfig()` function** (lines 156-185 in server.go). Its logic moves to the caller in `main.go`.

### Step 1.4: Update `main.go`

**BEFORE** (lines 230-249):
```go
server := srv.NewFromConfig(
	cfg,
	pluginManager,
	orchestrationEngine,
	planner,
	memoryManager,
	gardenManager,
	primaryAgent,
	intentClassifier,
	userRouter,
	traceLogger,
	costTracker,
	budgetTracker,
	nil,                   // memory maintenance (optional)
	toolRegistry,          // Phase 2: Gateway tool registry
	agentInitializer,      // Phase 2: Agent disposition initializer
	mcpHostManager,        // Phase 2: MCP host manager (optional)
	orchestrationExecutor, // Phase 3: Orchestration executor (gateway interface)
	memoryStore,           // Phase 3: Memory store (gateway interface)
)
```

**AFTER:**
```go
server := srv.New(srv.ServerDeps{
	Config: &srv.ServerConfig{
		Port:            cfg.Port,
		AllowedOrigins:  cfg.AllowedOrigins,
		AuthMode:        "api_key",
		Environment:     cfg.Environment,
		ShutdownTimeout: 30 * time.Second,
	},
	PluginManager:       pluginManager,
	OrchestrationEngine: orchestrationEngine,
	Planner:             planner,
	MemoryManager:       memoryManager,
	GardenManager:       gardenManager,
	PrimaryAgent:        primaryAgent,
	IntentClassifier:    intentClassifier,
	UserRouter:          userRouter,
	TraceLogger:         traceLogger,
	CostTracker:         costTracker,
	BudgetTracker:       budgetTracker,
	MemoryMaintenance:   nil,
	ToolRegistry:        toolRegistry,
	AgentInitializer:    agentInitializer,
	MCPHostManager:      mcpHostManager,
	OrchestrationExec:   orchestrationExecutor,
	MemoryStore:         memoryStore,
})
```

Also add the `time` import to main.go if not already present (it should already be there).

### Step 1.5: Update test files

Search for any test files that call `server.New(` or `server.NewFromConfig(`:

```bash
grep -rn 'srv\.New\|server\.New\|srv\.NewFromConfig\|server\.NewFromConfig' --include='*.go' .
```

For each call site, convert to the `ServerDeps{}` pattern. Most test files create a server with nil dependencies — the struct makes this trivially clear:

```go
// BEFORE (test):
s := server.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

// AFTER (test):
s := server.New(server.ServerDeps{})
```

### Step 1.6: Remove unused imports from server.go

After removing `NewFromConfig()`, the `config` import may become unused:
```go
"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/config"
```

Check if `config.Config` is still used anywhere in `server.go`. If `NewFromConfig()` was the only user, remove the import.

### Step 1.7: BUILD GATE

```bash
go build ./...
go vet ./...
go test ./server/... -count=1
```

---

## Phase 2: Handler Structs

### Goal

Convert all 13 `Initialize*()` + global-var patterns into proper struct methods. Each handler file becomes a struct with constructor + methods.

### Migration Order (simplest first, each one is independently testable):

1. `agents.go` — 1 global var, 4 handler methods
2. `user_preferences.go` — 1 global var, 2 handler methods
3. `models.go` — 1 global var (+ 2 timeout vars that stay), 2 handler methods
4. `memory_seeds.go` — 1 global var, 7 handler methods
5. `artifacts.go` — 1 global var, 7 handler methods
6. `projects.go` — 1 global var, 8 handler methods
7. `trace.go` — 1 global var, 5 handler methods
8. `api_keys.go` — 3 global vars, 6 handler methods
9. `search_conversations.go` — 1 global var, 1 handler method
10. `memory.go` — 3 global vars, 17 handler methods (largest)
11. `chat.go` — 6 global vars, streaming complexity (most complex)
12. `sse.go` — global client map + mutex (may stay as singleton)

**For each handler conversion, follow this exact pattern:**

---

### Conversion Template (use this for EVERY handler file)

#### A. In the handler file (`server/handlers/<name>.go`):

**BEFORE pattern:**
```go
var someManager *SomeType

func InitializeSomeHandlers(sm *SomeType) {
	someManager = sm
}

func HandleDoThing(c *gin.Context) {
	if someManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not initialized"})
		return
	}
	// ... uses someManager
}
```

**AFTER pattern:**
```go
// SomeHandler handles HTTP requests for [feature].
type SomeHandler struct {
	manager *SomeType
}

// NewSomeHandler creates a new SomeHandler.
func NewSomeHandler(sm *SomeType) *SomeHandler {
	return &SomeHandler{manager: sm}
}

func (h *SomeHandler) DoThing(c *gin.Context) {
	if h.manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not initialized"})
		return
	}
	// ... uses h.manager
}
```

Changes:
1. Delete the `var` block
2. Delete the `Initialize*()` function
3. Create a struct with the same fields
4. Create a `New*Handler()` constructor
5. Convert every `Handle*()` function to a method on the struct
6. Replace all references to the global var with `h.<field>`
7. Rename methods: `HandleListAgents` → `ListAgents` (drop the `Handle` prefix)

#### B. In `server/router.go`:

**BEFORE pattern:**
```go
func (s *Server) setupRoutes() {
	handlers.InitializeSomeHandlers(s.someManager)
	// ...
	v1.GET("/things", handlers.HandleDoThing)
}
```

**AFTER pattern:**
```go
func (s *Server) setupRoutes() {
	someHandler := handlers.NewSomeHandler(s.someManager)
	// ...
	v1.GET("/things", someHandler.DoThing)
}
```

#### C. In test files (`server/handlers/<name>_test.go`):

**BEFORE pattern:**
```go
func TestHandleDoThing(t *testing.T) {
	InitializeSomeHandlers(mockManager)
	// ... test using HandleDoThing
}
```

**AFTER pattern:**
```go
func TestSomeHandler_DoThing(t *testing.T) {
	h := NewSomeHandler(mockManager)
	// ... test using h.DoThing
}
```

---

### Detailed conversion: agents.go (example of the pattern applied)

#### agents.go

**BEFORE:**
```go
var agentManager *agent.AgentManager

func InitializeAgentHandlers(am *agent.AgentManager) {
	agentManager = am
}

func HandleListAgents(c *gin.Context) { /* uses agentManager */ }
func HandleGetAgent(c *gin.Context) { /* uses agentManager */ }
func HandleGetAgentCapabilities(c *gin.Context) { /* uses agentManager */ }
func HandleSeedAgents(c *gin.Context) { /* uses agentManager */ }
```

**AFTER:**
```go
// AgentHandler handles agent-related HTTP requests.
type AgentHandler struct {
	manager *agent.AgentManager
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(am *agent.AgentManager) *AgentHandler {
	return &AgentHandler{manager: am}
}

func (h *AgentHandler) ListAgents(c *gin.Context) { /* uses h.manager */ }
func (h *AgentHandler) GetAgent(c *gin.Context) { /* uses h.manager */ }
func (h *AgentHandler) GetAgentCapabilities(c *gin.Context) { /* uses h.manager */ }
func (h *AgentHandler) SeedAgents(c *gin.Context) { /* uses h.manager */ }
```

Inside each method, replace `agentManager` → `h.manager`.

#### router.go changes for agents:

**BEFORE:**
```go
if s.agentManager != nil {
	handlers.InitializeAgentHandlers(s.agentManager)
}
// ... later:
v1.GET("/agents", handlers.HandleListAgents)
```

**AFTER:**
```go
agentHandler := handlers.NewAgentHandler(s.agentManager)
// ... later:
v1.GET("/agents", agentHandler.ListAgents)
```

Note: the nil-check moves into the handler (it already has one: `if h.manager == nil`).

---

### Detailed conversion: memory.go (the complex one)

`memory.go` has 3 global vars and 3 Initialize functions because it serves memory, garden, AND maintenance endpoints. Split into 3 handlers or keep as one:

**Recommended: Keep as one `MemoryHandler` with 3 fields:**

```go
type MemoryHandler struct {
	memory      *memory.MemoryManager
	garden      *memory.GardenManager
	maintenance *maintenance.MemoryMaintenance
}

func NewMemoryHandler(mm *memory.MemoryManager, gm *memory.GardenManager, maint *maintenance.MemoryMaintenance) *MemoryHandler {
	return &MemoryHandler{
		memory:      mm,
		garden:      gm,
		maintenance: maint,
	}
}
```

Each method checks only the field it needs:
- `StoreMemory`, `RetrieveMemory`, etc. → check `h.memory`
- `ListSeeds`, `CreateSeed`, etc. → check `h.garden`
- `RunMaintenance` → check `h.maintenance`

---

### Detailed conversion: chat.go (the most complex)

chat.go has 6 global vars:
```go
var (
	intentClassifier *agent.IntentClassifier
	primaryAgent     *agent.PrimaryAgent
	userRouter       *services.UserRouter
	responseCache    *agent.ResponseCache
	streamingAgent   *streaming.StreamingAgentWithEvents
	chatPluginMgr    *provider.PluginManager
)
```

Note that `responseCache` and `streamingAgent` are **derived** from injected deps (created in `InitializeChatHandlers`).

```go
type ChatHandler struct {
	classifier   *agent.IntentClassifier
	agent        *agent.PrimaryAgent
	router       *services.UserRouter
	cache        *agent.ResponseCache
	streaming    *streaming.StreamingAgentWithEvents
	pluginMgr    *provider.PluginManager
}

func NewChatHandler(ic *agent.IntentClassifier, pa *agent.PrimaryAgent, ur *services.UserRouter, pm *provider.PluginManager) *ChatHandler {
	return &ChatHandler{
		classifier: ic,
		agent:      pa,
		router:     ur,
		cache:      agent.NewResponseCache(1*time.Hour, 1000),
		streaming:  streaming.NewStreamingAgentWithEvents(pa),
		pluginMgr:  pm,
	}
}
```

All private helper functions (`handleTemplateQuery`, `handleStreamingQuery`, `handleNonStreamingQuery`, `selectProviderWithRouting`, `resolveProviderAlias`, `firstLoadedProvider`, `getUserTier`) become methods on `ChatHandler`:

```go
func (h *ChatHandler) Chat(c *gin.Context) { ... }
func (h *ChatHandler) handleTemplateQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) { ... }
func (h *ChatHandler) handleNonStreamingQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) { ... }
func (h *ChatHandler) handleStreamingQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) { ... }
func (h *ChatHandler) selectProviderWithRouting(userID, model string, decision agent.RoutingDecision) (string, string, error) { ... }
func (h *ChatHandler) resolveProviderAlias(provider string) string { ... }
func (h *ChatHandler) firstLoadedProvider(candidates []string) string { ... }
```

Inside these, replace:
- `intentClassifier` → `h.classifier`
- `primaryAgent` → `h.agent`
- `userRouter` → `h.router`
- `responseCache` → `h.cache`
- `streamingAgent` → `h.streaming`
- `chatPluginMgr` → `h.pluginMgr`

The static vars `cloudProviders` and `localProviders` can stay as package-level vars (they're configuration, not state).

---

### Detailed conversion: sse.go (special case)

sse.go manages a global client map with a mutex. Options:

**Option A (recommended): Convert to SSEHandler struct:**
```go
type SSEHandler struct {
	clients   map[string]*models.Client
	mu        sync.RWMutex
}

func NewSSEHandler() *SSEHandler {
	return &SSEHandler{
		clients: make(map[string]*models.Client),
	}
}
```

Then `SendToClient`, `GetConnectedClients`, `DisconnectClient` become methods.

**Option B: Keep as singleton.** If broadcast.go and other code needs to access the SSE client map, it may be simpler to keep the global for now. This is acceptable as a deliberate decision.

Choose Option A if you want full consistency. Choose Option B if you want to minimize risk.

---

### Step 2.FINAL: Create `server/handlers/handlers.go` (optional orchestrator)

After converting all handlers, optionally create an orchestrator struct:

```go
package handlers

// Handlers aggregates all handler structs for easy dependency injection.
type Handlers struct {
	Chat        *ChatHandler
	Memory      *MemoryHandler
	Agents      *AgentHandler
	Models      *ModelHandler
	SSE         *SSEHandler
	Artifacts   *ArtifactHandler
	Projects    *ProjectHandler
	Trace       *TraceHandler
	Seeds       *SeedHandler
	APIKeys     *APIKeyHandler
	Preferences *PreferencesHandler
	Search      *SearchHandler
}
```

Then in `router.go`:
```go
func (s *Server) setupRoutes() {
	h := &handlers.Handlers{
		Chat:   handlers.NewChatHandler(s.intentClassifier, s.primaryAgent, s.userRouter, s.pluginManager),
		Memory: handlers.NewMemoryHandler(s.memoryManager, s.gardenManager, s.memoryMaintenance),
		Agents: handlers.NewAgentHandler(s.agentManager),
		// ... etc
	}

	v1.POST("/chat", h.Chat.Chat)
	v1.GET("/agents", h.Agents.ListAgents)
	// ... etc
}
```

This is optional — you can also just create each handler locally in `setupRoutes()`.

---

## Phase 2 Build Gates

After EACH handler conversion:
```bash
go build ./...
go vet ./...
go test ./server/handlers/... -count=1 -race
```

After ALL handler conversions:
```bash
go build ./...
go vet ./...
go test ./... -count=1 -race
```

---

## Complete File Manifest

### New files:
- `server/deps.go` — ServerDeps struct
- `server/handlers/handlers.go` — (optional) Handlers aggregator

### Modified files (Phase 1):
- `server/server.go` — New() accepts ServerDeps, remove NewFromConfig()
- `main.go` — construct ServerDeps, inline config conversion

### Modified files (Phase 2, per handler):
Each handler has 3 files to modify:

| Handler | Source File | Test File | router.go changes |
|---------|-----------|-----------|-------------------|
| agents | `server/handlers/agents.go` | `server/handlers/agents_test.go` | Remove InitializeAgentHandlers, use AgentHandler |
| user_preferences | `server/handlers/user_preferences.go` | — | Remove InitializePreferencesHandlers |
| models | `server/handlers/models.go` | `server/handlers/models_test.go` | Remove InitializeModelHandlers |
| memory_seeds | `server/handlers/memory_seeds.go` | `server/handlers/memory_seeds_test.go` | Remove InitializeMemorySeedHandlers |
| artifacts | `server/handlers/artifacts.go` | `server/handlers/artifacts_test.go` | Remove InitializeArtifactHandlers |
| projects | `server/handlers/projects.go` | `server/handlers/projects_test.go` | Remove InitializeProjectHandlers |
| trace | `server/handlers/trace.go` | `server/handlers/trace_test.go` | Remove InitializeTraceHandlers |
| api_keys | `server/handlers/api_keys.go` | `server/handlers/api_keys_test.go` | Remove InitializeAPIKeyHandlers |
| search | `server/handlers/search_conversations.go` | — | Remove InitializeSearchHandlers |
| memory | `server/handlers/memory.go` | `server/handlers/memory_test.go` | Remove 3 Initialize* functions |
| chat | `server/handlers/chat.go` | `server/handlers/chat_test.go` + `chat_streaming_integration_test.go` + `chat_orchestration_test.go` | Remove InitializeChatHandlers |
| sse | `server/handlers/sse.go` | `server/handlers/sse_test.go` + `broadcast_test.go` | Global client map → struct |
| broadcast | `server/handlers/broadcast.go` | `server/handlers/broadcast_test.go` | May need SSEHandler reference |

Plus:
- `server/router.go` — all route registrations updated

**Total estimated: ~35 files**

---

## Troubleshooting

### "undefined: HandleListAgents"
You renamed it to `ListAgents` but forgot to update `router.go`. Search for the old name and replace.

### "undefined: InitializeAgentHandlers"
You deleted the function but `router.go` still calls it. Remove the call and replace with `NewAgentHandler()`.

### Tests fail with "nil pointer"
The test was relying on the global var being set by a previous test. With structs, each test must create its own handler instance.

### broadcast.go can't access SSE clients
If you converted sse.go to a struct, broadcast.go needs a reference to the SSEHandler. Either:
- Pass it via constructor: `NewBroadcastHandler(sseHandler *SSEHandler)`
- Or keep the SSE client map as a shared singleton (Option B above)

### chat.go helper functions
All private helper functions in chat.go (`handleTemplateQuery`, `selectProviderWithRouting`, etc.) must become methods on `ChatHandler`. If they call each other, that works naturally since they're all on the same struct.

---

## Why This Matters

1. **Testability:** Handler structs can be instantiated with mock dependencies per test — no global state leakage between tests
2. **Multiple instances:** The server could theoretically run multiple instances (useful for integration testing)
3. **Explicit dependencies:** Each handler declares exactly what it needs
4. **IDE support:** Method sets on structs are easier to navigate than package-level functions
5. **Extensibility:** Adding a new handler group is just a new struct + constructor
6. **No more nil-guard surprise:** Each handler always has its deps from construction time
