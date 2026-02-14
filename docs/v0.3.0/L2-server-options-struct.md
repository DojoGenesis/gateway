# L2: Server Constructor → Options Struct

**Priority:** Medium
**Effort:** Medium (~3-4 hours)
**Impact:** Better extensibility, easier testing, cleaner API

## Context

The server constructor currently takes 18 parameters, which is a code smell. The signature is difficult to read, error-prone to extend, and hard to test with partial mocks.

**Current signature (server/server.go:84-103):**
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
) *Server
```

**Problems:**
1. **Hard to extend** - Adding a new dependency requires changing all call sites
2. **Hard to read** - 18 parameters is overwhelming
3. **Hard to test** - Must provide all 18 parameters even for simple tests
4. **No optional dependencies** - Can't express "this is optional"
5. **Parameter order matters** - Easy to mix up similarly-typed parameters

## Target Architecture

### Phase 1: Create ServerDeps Struct

```go
// server/server.go

// ServerDeps contains all dependencies required to construct a Server.
// Dependencies are grouped logically for clarity.
type ServerDeps struct {
	// Configuration
	Config *ServerConfig

	// Core dependencies (required)
	PluginManager       *provider.PluginManager
	OrchestrationEngine *orchestrationpkg.Engine
	Planner             orchestrationpkg.PlannerInterface
	MemoryManager       *memory.MemoryManager

	// Agent system
	PrimaryAgent     *agent.PrimaryAgent
	IntentClassifier *agent.IntentClassifier
	UserRouter       *services.UserRouter

	// Observability
	TraceLogger *trace.TraceLogger
	CostTracker *services.CostTracker

	// Optional dependencies (can be nil)
	GardenManager     *memory.GardenManager
	BudgetTracker     *services.BudgetTracker
	MemoryMaintenance *maintenance.MemoryMaintenance

	// Phase 2+ Gateway interfaces (optional)
	ToolRegistry          gateway.ToolRegistry
	AgentInitializer      gateway.AgentInitializer
	MCPHostManager        MCPStatusProvider
	OrchestrationExecutor gateway.OrchestrationExecutor
	MemoryStore           gateway.MemoryStore
}

// New creates a new Server with all dependencies provided via ServerDeps.
func New(deps ServerDeps) *Server {
	// Validate required dependencies
	if deps.Config == nil {
		deps.Config = &ServerConfig{
			Port:            "8080",
			AllowedOrigins:  []string{"http://localhost:3000"},
			AuthMode:        "api_key",
			Environment:     "production",
			ShutdownTimeout: 30 * time.Second,
		}
	}

	if deps.Config.ShutdownTimeout == 0 {
		deps.Config.ShutdownTimeout = 30 * time.Second
	}

	if deps.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	s := &Server{
		router:                gin.New(),
		cfg:                   deps.Config,
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
		orchestrationExecutor: deps.OrchestrationExecutor,
		memoryStore:           deps.MemoryStore,
		orchestrations:        NewOrchestrationStore(),
		agents:                make(map[string]*gateway.AgentConfig),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}
```

### Phase 2: Update NewFromConfig

```go
// NewFromConfig creates a Server from the application Config.
// This is a convenience wrapper around New() for the common case.
func NewFromConfig(
	appCfg *config.Config,
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
	serverCfg := &ServerConfig{
		Port:            appCfg.Port,
		AllowedOrigins:  appCfg.AllowedOrigins,
		AuthMode:        "api_key",
		Environment:     appCfg.Environment,
		ShutdownTimeout: 30 * time.Second,
	}

	return New(ServerDeps{
		Config:                serverCfg,
		PluginManager:         pm,
		OrchestrationEngine:   orch,
		Planner:               plan,
		MemoryManager:         mm,
		GardenManager:         gm,
		PrimaryAgent:          pa,
		IntentClassifier:      ic,
		UserRouter:            ur,
		TraceLogger:           tl,
		CostTracker:           ct,
		BudgetTracker:         bt,
		MemoryMaintenance:     maint,
		ToolRegistry:          toolReg,
		AgentInitializer:      agentInit,
		MCPHostManager:        mcpMgr,
		OrchestrationExecutor: orchExec,
		MemoryStore:           memStore,
	})
}
```

### Phase 3: Builder Pattern (Optional Enhancement)

For even better ergonomics, add a builder:

```go
// ServerBuilder provides a fluent API for constructing a Server.
type ServerBuilder struct {
	deps ServerDeps
}

// NewServerBuilder creates a new builder with defaults.
func NewServerBuilder() *ServerBuilder {
	return &ServerBuilder{
		deps: ServerDeps{
			Config: &ServerConfig{
				Port:            "8080",
				AllowedOrigins:  []string{"http://localhost:3000"},
				AuthMode:        "api_key",
				Environment:     "production",
				ShutdownTimeout: 30 * time.Second,
			},
		},
	}
}

// WithConfig sets the server configuration.
func (b *ServerBuilder) WithConfig(cfg *ServerConfig) *ServerBuilder {
	b.deps.Config = cfg
	return b
}

// WithPluginManager sets the plugin manager.
func (b *ServerBuilder) WithPluginManager(pm *provider.PluginManager) *ServerBuilder {
	b.deps.PluginManager = pm
	return b
}

// ... (add WithX methods for each dependency)

// Build constructs the Server.
func (b *ServerBuilder) Build() *Server {
	return New(b.deps)
}
```

Usage:
```go
server := NewServerBuilder().
	WithPluginManager(pluginMgr).
	WithMemoryManager(memMgr).
	WithPrimaryAgent(agent).
	Build()
```

## Migration Strategy

### Step 1: Add ServerDeps Struct (Non-Breaking)

1. Add `ServerDeps` struct to `server/server.go`
2. Keep existing `New()` signature
3. Add new `NewWithDeps(deps ServerDeps)` function
4. Implement `NewWithDeps()` with the new logic

**No breaking changes yet** - both constructors coexist.

### Step 2: Update main.go to Use NewWithDeps

Update `main.go` to use the new constructor:

```go
// main.go (before)
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
	nil, // memory maintenance
	toolRegistry,
	agentInitializer,
	mcpHostManager,
	orchestrationExecutor,
	memoryStore,
)

// main.go (after)
server := srv.New(srv.ServerDeps{
	Config:                serverCfg,
	PluginManager:         pluginManager,
	OrchestrationEngine:   orchestrationEngine,
	Planner:               planner,
	MemoryManager:         memoryManager,
	GardenManager:         gardenManager,
	PrimaryAgent:          primaryAgent,
	IntentClassifier:      intentClassifier,
	UserRouter:            userRouter,
	TraceLogger:           traceLogger,
	CostTracker:           costTracker,
	BudgetTracker:         budgetTracker,
	MemoryMaintenance:     nil, // optional - explicit nil is clearer
	ToolRegistry:          toolRegistry,
	AgentInitializer:      agentInitializer,
	MCPHostManager:        mcpHostManager,
	OrchestrationExecutor: orchestrationExecutor,
	MemoryStore:           memoryStore,
})
```

### Step 3: Update Tests

Update all test files that construct servers:

```go
// Before (verbose, hard to read)
server := New(
	&ServerConfig{Port: "8080"},
	nil, // plugin manager
	nil, // orchestration
	nil, // planner
	mockMemory,
	nil, // garden
	nil, // agent
	nil, // classifier
	nil, // router
	nil, // trace
	nil, // cost
	nil, // budget
	nil, // maintenance
	nil, // tool registry
	nil, // agent init
	nil, // mcp
	nil, // orch exec
	nil, // mem store
)

// After (clear, extensible)
server := New(ServerDeps{
	Config:        &ServerConfig{Port: "8080"},
	MemoryManager: mockMemory,
	// All other fields nil by default - clearer intent
})
```

### Step 4: Deprecate Old Constructor

Add deprecation comment:

```go
// Deprecated: Use New(ServerDeps{...}) instead.
// This function will be removed in v0.4.0.
func NewFromConfig(...) *Server {
	// Implementation stays for backwards compatibility
}
```

### Step 5: Remove Old Constructor (v0.4.0)

After one release cycle with deprecation warning, remove the old constructor entirely.

## Files to Update

### Core Implementation
- `server/server.go` - Add ServerDeps, update New()

### Call Sites
- `main.go` - Primary server construction
- `server/server_test.go` - Unit tests (if any)

### Test Files (search for `srv.New` or `server.New`)
```bash
grep -r "srv\.New\|server\.New" . --include="*_test.go"
```

Update each test to use the new constructor.

## Testing Strategy

### 1. Backwards Compatibility Tests

Ensure old constructor still works during deprecation period:

```go
func TestOldConstructorStillWorks(t *testing.T) {
	// Verify NewFromConfig still functions
	server := NewFromConfig(/* all 18 params */)
	assert.NotNil(t, server)
}
```

### 2. New Constructor Tests

```go
func TestNewWithMinimalDeps(t *testing.T) {
	server := New(ServerDeps{
		Config: &ServerConfig{Port: "8080"},
	})
	assert.NotNil(t, server)
	assert.Equal(t, "8080", server.cfg.Port)
}

func TestNewWithAllDeps(t *testing.T) {
	deps := ServerDeps{
		Config:           &ServerConfig{},
		PluginManager:    mockPluginManager,
		MemoryManager:    mockMemoryManager,
		// ... all dependencies
	}
	server := New(deps)
	assert.NotNil(t, server)
	assert.Equal(t, mockPluginManager, server.pluginManager)
}

func TestNewWithNilOptionalDeps(t *testing.T) {
	server := New(ServerDeps{
		Config:            &ServerConfig{},
		PluginManager:     mockPluginManager,
		MemoryManager:     mockMemoryManager,
		// GardenManager, BudgetTracker, etc. are nil
	})
	assert.NotNil(t, server)
	assert.Nil(t, server.gardenManager) // Should gracefully handle nil optional deps
}
```

### 3. Integration Tests

Run full integration tests to ensure server starts correctly:

```bash
go test ./server/... -v -run TestServerStartup
```

## Benefits After Migration

### 1. Easier Testing

```go
// Only provide what you need for the test
func TestMemoryEndpoint(t *testing.T) {
	server := New(ServerDeps{
		Config:        defaultConfig(),
		MemoryManager: mockMemory,
		TraceLogger:   mockTracer,
		// Everything else nil - test focuses on memory functionality
	})
	// Test memory endpoints
}
```

### 2. Clearer Optional Dependencies

```go
// Explicitly show what's optional
deps := ServerDeps{
	// Required
	Config:        cfg,
	PluginManager: pm,
	MemoryManager: mm,

	// Optional (nil is valid)
	GardenManager:     gm,
	MemoryMaintenance: nil, // Explicitly disabled
	MCPHostManager:    nil, // Not configured
}
```

### 3. Easier Extension

Adding a new dependency only requires:
1. Add field to ServerDeps
2. Add assignment in New()
3. No changes to call sites (uses struct field names)

### 4. Better Documentation

```go
// ServerDeps is self-documenting - field names show purpose
deps := ServerDeps{
	PluginManager:    pm,  // Clear: plugin system
	MemoryManager:    mm,  // Clear: memory backend
	TraceLogger:      tl,  // Clear: observability
	// vs New(pm, mm, tl, ...) - what's what?
}
```

## Rollback Plan

If issues arise during migration:

1. **Revert main.go changes** - go back to old constructor
2. **Keep ServerDeps struct** - it's non-breaking, useful for future
3. **Remove deprecation warning** - if we're not ready to migrate

## Validation Checklist

- [ ] ServerDeps struct added
- [ ] New() updated to accept ServerDeps
- [ ] main.go migrated to new constructor
- [ ] All tests updated
- [ ] Old constructor deprecated (comment added)
- [ ] Documentation updated
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] Integration tests pass

## Estimated Timeline

- **Step 1 (Add ServerDeps):** 30 minutes
- **Step 2 (Update main.go):** 15 minutes
- **Step 3 (Update tests):** 1-2 hours
- **Step 4 (Deprecation):** 15 minutes
- **Total:** ~3-4 hours

## Future Enhancements (v0.4.0+)

1. **Builder pattern** (optional, for ergonomics)
2. **Validation** - Add `Validate()` method to ServerDeps
3. **Functional options** - Alternative pattern:
   ```go
   server := New(
       WithPluginManager(pm),
       WithMemoryManager(mm),
       WithTraceLogger(tl),
   )
   ```

## Success Metrics

- Zero parameter constructors remain (except test helpers)
- All call sites use ServerDeps
- Test setup requires <5 lines to construct server
- New dependencies can be added without touching call sites
