# L3: Handler Struct Pattern

**Priority:** Medium-High
**Effort:** High (~8-12 hours)
**Impact:** Better testability, explicit dependencies, cleaner architecture

## Context

The codebase currently uses **package-level global variables** for handler dependencies, which creates several problems:

**Current pattern (problematic):**
```go
// handlers/agents.go
var agentManager *agent.AgentManager

func InitializeAgentHandlers(am *agent.AgentManager) {
	agentManager = am
}

func HandleGetAgent(c *gin.Context) {
	if agentManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "agent manager not initialized",
		})
		return
	}
	// Use agentManager...
}
```

**Problems:**
1. **Hard to test** - Global state makes parallel tests unsafe
2. **Hidden dependencies** - Not clear what HandleGetAgent needs
3. **Race conditions** - Global variables + concurrent tests = data races
4. **No type safety** - Easy to forget initialization
5. **Tight coupling** - Handlers directly access global state

**Current initialization (main.go → server.go → router.go):**
```go
// server/router.go:9-24
if s.pluginManager != nil {
	handlers.InitializeModelHandlers(s.pluginManager)
}
if s.intentClassifier != nil && s.primaryAgent != nil {
	handlers.InitializeChatHandlers(s.intentClassifier, s.primaryAgent, s.userRouter, s.pluginManager)
}
if s.memoryManager != nil {
	handlers.InitializeMemoryHandlers(s.memoryManager)
}
// ... more initializers
```

## Target Architecture

### Handler Struct Pattern

**Better pattern (goal):**
```go
// handlers/agents.go

// AgentHandler handles agent-related HTTP requests.
type AgentHandler struct {
	manager *agent.AgentManager
	tracer  *trace.TraceLogger
}

// NewAgentHandler creates a new agent handler.
func NewAgentHandler(manager *agent.AgentManager, tracer *trace.TraceLogger) *AgentHandler {
	return &AgentHandler{
		manager: manager,
		tracer:  tracer,
	}
}

// GetAgent handles GET /v1/agents/:id
func (h *AgentHandler) GetAgent(c *gin.Context) {
	// Dependencies are fields - no nil check needed if constructor validates
	agentID := c.Param("id")
	if agentID == "" {
		respondError(c, http.StatusBadRequest, "agent_id is required")
		return
	}

	agent, err := h.manager.GetAgent(c.Request.Context(), agentID)
	if errors.Is(err, agent.ErrAgentNotFound) {
		respondNotFound(c, "agent")
		return
	}
	if err != nil {
		slog.Error("failed to get agent", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, agent)
}

// ListAgents handles GET /v1/agents
func (h *AgentHandler) ListAgents(c *gin.Context) {
	// ...
}
```

**Route registration (router.go):**
```go
func (s *Server) setupRoutes() {
	// Create handler instances with dependencies
	agentHandler := handlers.NewAgentHandler(s.agentManager, s.traceLogger)
	memoryHandler := handlers.NewMemoryHandler(s.memoryManager)
	chatHandler := handlers.NewChatHandler(s.primaryAgent, s.intentClassifier, s.userRouter, s.pluginManager)

	// Register routes using handler methods
	v1 := s.router.Group("/v1")
	{
		v1.GET("/agents", agentHandler.ListAgents)
		v1.GET("/agents/:id", agentHandler.GetAgent)
		v1.POST("/agents", agentHandler.CreateAgent)
		v1.DELETE("/agents/:id", agentHandler.DeleteAgent)

		v1.POST("/chat", chatHandler.HandleChat)

		v1.GET("/memory", memoryHandler.ListMemories)
		v1.POST("/memory", memoryHandler.StoreMemory)
	}
}
```

## Benefits

### 1. Explicit Dependencies
```go
// Clear what each handler needs
chatHandler := handlers.NewChatHandler(
	primaryAgent,      // needs primary agent
	intentClassifier,  // needs classifier
	userRouter,        // needs router
	pluginManager,     // needs plugins
)
```

### 2. Easy Testing
```go
func TestAgentHandler_GetAgent(t *testing.T) {
	// Create mock dependencies
	mockManager := &MockAgentManager{}
	mockTracer := &MockTracer{}

	// Create handler with mocks - no global state!
	handler := NewAgentHandler(mockManager, mockTracer)

	// Test the handler method
	req := httptest.NewRequest("GET", "/agents/123", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "123"}}

	handler.GetAgent(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockManager.AssertCalled(t, "GetAgent", mock.Anything, "123")
}
```

### 3. No Race Conditions
```go
// Each test creates its own handler - parallel tests are safe
func TestAgentHandler_Parallel(t *testing.T) {
	t.Parallel() // Safe now!

	handler := NewAgentHandler(mockManager, mockTracer)
	// Test handler...
}
```

### 4. Type Safety
```go
// Compiler enforces dependencies at construction
handler := NewAgentHandler(manager, tracer) // Compile-time check

// vs old way:
InitializeAgentHandlers(manager) // Runtime nil check needed
```

## Migration Strategy

### Phase 1: Identify Handler Families (by dependency)

Group handlers by shared dependencies:

**Group 1: Agent Handlers**
- File: `handlers/agents.go`
- Dependencies: `*agent.AgentManager`
- Routes: `/v1/agents/*`

**Group 2: Memory Handlers**
- File: `handlers/memory.go`
- Dependencies: `*memory.MemoryManager`
- Routes: `/v1/memory/*`

**Group 3: Memory Seeds Handlers**
- File: `handlers/memory_seeds.go`
- Dependencies: `*memory.GardenManager`
- Routes: `/v1/seeds/*`, `/v1/snapshots/*`

**Group 4: Chat Handler**
- File: `handlers/chat.go`
- Dependencies: `*agent.IntentClassifier`, `*agent.PrimaryAgent`, `*services.UserRouter`, `*provider.PluginManager`
- Routes: `/v1/chat`, `/v1/chat/completions`

**Group 5: Model/Provider Handlers**
- File: `handlers/models.go`
- Dependencies: `*provider.PluginManager`
- Routes: `/v1/models`, `/v1/providers`
- **Status:** Already migrated! ✓ (see router.go:10,81)

**Group 6: Trace Handlers**
- File: `handlers/trace.go`
- Dependencies: `*trace.TraceStorage`
- Routes: `/v1/traces/*`

**Group 7: Tools Handlers**
- File: `handlers/tools.go`
- Dependencies: `gateway.ToolRegistry`
- Routes: `/v1/tools/*`

**Group 8: Project Handlers**
- File: `handlers/projects.go`
- Dependencies: `*projects.ProjectManager`
- Routes: `/v1/projects/*`

**Group 9: Artifact Handlers**
- File: `handlers/artifacts.go`
- Dependencies: `*artifacts.ArtifactManager`
- Routes: `/v1/artifacts/*`

**Group 10: Broadcast/SSE Handlers**
- File: `handlers/broadcast.go`, `handlers/sse.go`
- Dependencies: None (pure SSE broadcaster)
- Routes: `/events`

### Phase 2: Migration Order (Least Risk → Most Risk)

#### **Tier 1: Already Migrated ✓**
- `models.go` - Already uses `NewModelHandler()` struct pattern

#### **Tier 2: Simple Handlers (Low Risk)**
Start with handlers that:
- Have few dependencies
- Are less critical to core functionality
- Have simpler logic

1. **Broadcast/SSE** (`broadcast.go`, `sse.go`) - No dependencies
2. **Tools** (`tools.go`) - Single dependency (ToolRegistry)
3. **Trace** (`trace.go`) - Single dependency (TraceStorage)

#### **Tier 3: Medium Complexity**
4. **Projects** (`projects.go`) - Single dependency, moderate logic
5. **Artifacts** (`artifacts.go`) - Single dependency, moderate logic
6. **Memory Seeds** (`memory_seeds.go`) - Single dependency (GardenManager)
7. **Agents** (`agents.go`) - Single dependency (AgentManager)

#### **Tier 4: Complex Handlers (High Risk)**
Migrate last, test thoroughly:

8. **Memory** (`memory.go`) - Core functionality
9. **Chat** (`chat.go`) - Complex, streaming, multiple dependencies

### Phase 3: Step-by-Step Migration Template

For each handler file:

#### Step 1: Define Handler Struct

```go
// handlers/agents.go

// AgentHandler handles agent-related HTTP endpoints.
type AgentHandler struct {
	manager *agent.AgentManager
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(manager *agent.AgentManager) *AgentHandler {
	if manager == nil {
		panic("AgentHandler requires non-nil agent.AgentManager")
	}
	return &AgentHandler{
		manager: manager,
	}
}
```

#### Step 2: Convert Functions to Methods

```go
// Before
func HandleGetAgent(c *gin.Context) {
	if agentManager == nil {
		// nil check
	}
	// use agentManager
}

// After
func (h *AgentHandler) GetAgent(c *gin.Context) {
	// h.manager is guaranteed non-nil by constructor
	// no nil check needed
}
```

#### Step 3: Update router.go

```go
// Before
if s.agentManager != nil {
	handlers.InitializeAgentHandlers(s.agentManager)
}
// ...
v1.GET("/agents/:id", handlers.HandleGetAgent)

// After
agentHandler := handlers.NewAgentHandler(s.agentManager)
v1.GET("/agents/:id", agentHandler.GetAgent)
```

#### Step 4: Remove Global Variables

```go
// Delete these from handlers/agents.go:
var agentManager *agent.AgentManager
func InitializeAgentHandlers(am *agent.AgentManager) {
	agentManager = am
}
```

#### Step 5: Add Tests

```go
// handlers/agents_test.go

func TestAgentHandler_GetAgent_Success(t *testing.T) {
	mockManager := &MockAgentManager{}
	mockManager.On("GetAgent", mock.Anything, "test-id").Return(&agent.Agent{ID: "test-id"}, nil)

	handler := NewAgentHandler(mockManager)

	req := httptest.NewRequest("GET", "/agents/test-id", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	handler.GetAgent(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockManager.AssertExpectations(t)
}

func TestAgentHandler_GetAgent_NotFound(t *testing.T) {
	mockManager := &MockAgentManager{}
	mockManager.On("GetAgent", mock.Anything, "missing").Return(nil, agent.ErrAgentNotFound)

	handler := NewAgentHandler(mockManager)

	req := httptest.NewRequest("GET", "/agents/missing", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "missing"}}

	handler.GetAgent(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

### Phase 4: Handle Special Cases

#### Case 1: Handlers with Multiple Dependencies (chat.go)

```go
type ChatHandler struct {
	intentClassifier *agent.IntentClassifier
	primaryAgent     *agent.PrimaryAgent
	userRouter       *services.UserRouter
	pluginManager    *provider.PluginManager
	responseCache    *agent.ResponseCache
	streamingAgent   *streaming.StreamingAgentWithEvents
}

func NewChatHandler(
	ic *agent.IntentClassifier,
	pa *agent.PrimaryAgent,
	ur *services.UserRouter,
	pm *provider.PluginManager,
) *ChatHandler {
	if ic == nil || pa == nil {
		panic("ChatHandler requires non-nil IntentClassifier and PrimaryAgent")
	}

	return &ChatHandler{
		intentClassifier: ic,
		primaryAgent:     pa,
		userRouter:       ur,
		pluginManager:    pm,
		responseCache:    agent.NewResponseCache(1*time.Hour, 1000),
		streamingAgent:   streaming.NewStreamingAgentWithEvents(pa),
	}
}
```

#### Case 2: Shared State Between Handlers (SSE broadcaster)

If handlers share state (e.g., SSE broadcaster), inject the shared component:

```go
// handlers/broadcast.go
type BroadcastHandler struct {
	broadcaster *SSEBroadcaster // Shared across all SSE clients
}

func NewBroadcastHandler(broadcaster *SSEBroadcaster) *BroadcastHandler {
	return &BroadcastHandler{
		broadcaster: broadcaster,
	}
}

// router.go
sseBroadcaster := handlers.NewSSEBroadcaster()
broadcastHandler := handlers.NewBroadcastHandler(sseBroadcaster)
v1.GET("/events", broadcastHandler.HandleSSE)
```

#### Case 3: Optional Dependencies

Use pointer types and nil checks:

```go
type MemoryHandler struct {
	manager     *memory.MemoryManager
	maintenance *maintenance.MemoryMaintenance // Optional
}

func NewMemoryHandler(manager *memory.MemoryManager, maint *maintenance.MemoryMaintenance) *MemoryHandler {
	if manager == nil {
		panic("MemoryHandler requires non-nil MemoryManager")
	}
	return &MemoryHandler{
		manager:     manager,
		maintenance: maint, // Can be nil
	}
}

func (h *MemoryHandler) RunMaintenance(c *gin.Context) {
	if h.maintenance == nil {
		respondError(c, http.StatusNotImplemented, "maintenance not configured")
		return
	}
	// Use h.maintenance
}
```

## Testing Strategy

### 1. Unit Tests for Each Handler Method

```go
// Test success path
func TestAgentHandler_GetAgent_Success(t *testing.T)

// Test not found
func TestAgentHandler_GetAgent_NotFound(t *testing.T)

// Test validation errors
func TestAgentHandler_GetAgent_MissingID(t *testing.T)

// Test internal errors
func TestAgentHandler_GetAgent_InternalError(t *testing.T)
```

### 2. Integration Tests (Existing Tests Still Run)

Existing integration tests should continue to work:
```bash
go test ./server/handlers/... -v
```

### 3. Parallel Test Safety

Verify tests can run in parallel:
```go
func TestAgentHandler_Parallel(t *testing.T) {
	t.Parallel() // Should not cause races
	// Test handler...
}
```

Run with race detector:
```bash
go test ./server/handlers/... -race -count=10
```

## Migration Checklist (Per Handler)

- [ ] Define handler struct with dependencies
- [ ] Create `New{Handler}` constructor with validation
- [ ] Convert all `Handle*` functions to `(h *{Handler}) MethodName` methods
- [ ] Update `router.go` to create handler instance
- [ ] Update `router.go` to register routes using `handler.Method`
- [ ] Remove global variables (`var xyz *Type`)
- [ ] Remove `Initialize*Handlers()` function
- [ ] Add unit tests for each handler method
- [ ] Run integration tests
- [ ] Run with race detector
- [ ] Update documentation

## Files to Update

### Handler Files (9 files to migrate)
1. `handlers/agents.go` ⏭️ Migrate
2. `handlers/memory.go` ⏭️ Migrate
3. `handlers/memory_seeds.go` ⏭️ Migrate
4. `handlers/chat.go` ⏭️ Migrate
5. `handlers/models.go` ✓ Already migrated
6. `handlers/trace.go` ⏭️ Migrate
7. `handlers/tools.go` ⏭️ Migrate
8. `handlers/projects.go` ⏭️ Migrate (may not exist yet)
9. `handlers/artifacts.go` ⏭️ Migrate (may not exist yet)
10. `handlers/broadcast.go` / `handlers/sse.go` ⏭️ Migrate

### Router File
- `server/router.go` - Update route registration for all handlers

### Test Files
- Create/update `handlers/*_test.go` for each migrated handler

## Example: Complete Migration of agents.go

### Before
```go
// handlers/agents.go
package handlers

import (
	"net/http"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/gin-gonic/gin"
)

var agentManager *agent.AgentManager

func InitializeAgentHandlers(am *agent.AgentManager) {
	agentManager = am
}

func HandleGetAgent(c *gin.Context) {
	if agentManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not initialized"})
		return
	}
	// ... logic
}

func HandleListAgents(c *gin.Context) {
	if agentManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not initialized"})
		return
	}
	// ... logic
}
```

### After
```go
// handlers/agents.go
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/gin-gonic/gin"
)

// AgentHandler handles agent-related HTTP requests.
type AgentHandler struct {
	manager *agent.AgentManager
}

// NewAgentHandler creates a new AgentHandler.
// Panics if manager is nil (fail-fast at startup, not at request time).
func NewAgentHandler(manager *agent.AgentManager) *AgentHandler {
	if manager == nil {
		panic("AgentHandler requires non-nil agent.AgentManager")
	}
	return &AgentHandler{
		manager: manager,
	}
}

// GetAgent handles GET /v1/agents/:id
func (h *AgentHandler) GetAgent(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		respondError(c, http.StatusBadRequest, "agent_id is required")
		return
	}

	agent, err := h.manager.GetAgent(c.Request.Context(), agentID)
	if errors.Is(err, agent.ErrAgentNotFound) {
		respondNotFound(c, "agent")
		return
	}
	if err != nil {
		slog.Error("failed to get agent", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, agent)
}

// ListAgents handles GET /v1/agents
func (h *AgentHandler) ListAgents(c *gin.Context) {
	agents, err := h.manager.ListAgents(c.Request.Context())
	if err != nil {
		slog.Error("failed to list agents", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"count":  len(agents),
	})
}
```

### router.go Changes
```go
// Before
if s.agentManager != nil {
	handlers.InitializeAgentHandlers(s.agentManager)
}
v1.GET("/agents/:id", handlers.HandleGetAgent)
v1.GET("/agents", handlers.HandleListAgents)

// After
agentHandler := handlers.NewAgentHandler(s.agentManager)
v1.GET("/agents/:id", agentHandler.GetAgent)
v1.GET("/agents", agentHandler.ListAgents)
```

## Rollback Plan

If migration causes issues:

1. **Revert incrementally** - Roll back one handler at a time
2. **Keep both patterns** - Old global + new struct can coexist temporarily
3. **Feature flag** - Use env var to switch between old/new handlers

## Validation

After each handler migration:

```bash
# Build succeeds
go build ./...

# Tests pass
go test ./server/handlers/... -v

# No race conditions
go test ./server/handlers/... -race -count=10

# Integration tests pass
go test ./... -tags=integration
```

## Estimated Timeline

- **Tier 1 (Already done):** 0 hours ✓
- **Tier 2 (3 simple handlers):** 3-4 hours
- **Tier 3 (5 medium handlers):** 5-6 hours
- **Tier 4 (2 complex handlers):** 3-4 hours
- **Testing & cleanup:** 2 hours
- **Total:** 13-16 hours (spread over 2-3 days)

## Success Metrics

- Zero global variables in handlers package
- Zero `Initialize*Handlers()` functions
- All handlers use struct pattern with constructors
- 100% of handler methods are testable with mocks
- All tests pass with `-race` flag
- Handler test coverage ≥80%
