# Orchestration Architecture (v0.3.0)

## Overview

This architecture was implemented following the **Zenflow v0.3.0 commission specification**, which defines the orchestration extraction requirements for **AgenticGatewayByDojoGenesis**.

As of v0.3.0, the orchestration system has been refactored into a clean, standalone package with a thin adapter layer. This architecture enables:

- **Reusability**: The `orchestration/` package can be used independently
- **Testability**: Clear boundaries enable focused unit testing
- **Maintainability**: Single source of truth for orchestration logic
- **Extensibility**: New implementations can be added via adapters

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     AgenticGateway Server                        │
│                                                                   │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │   main.go   │  │ Server Layer │  │   HTTP Handlers        │  │
│  │             │  │              │  │  - handle_orchestrate  │  │
│  │ - Creates   │  │ - Gin routes │  │  - handle_gateway      │  │
│  │   adapters  │  │ - Middleware │  │  - handle_admin        │  │
│  │ - Wires     │  │ - Auth       │  │                        │  │
│  │   engine    │  │              │  │                        │  │
│  └──────┬──────┘  └──────────────┘  └────────────────────────┘  │
│         │                                                         │
│         │ Creates & Injects                                      │
│         ▼                                                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │         server/orchestration (Adapter Layer)            │    │
│  │                                                          │    │
│  │  ┌──────────────────────────────────────────────────┐   │    │
│  │  │              adapters.go                         │   │    │
│  │  │                                                  │   │    │
│  │  │  • ToolInvokerAdapter                           │   │    │
│  │  │    └─> tools.Registry                           │   │    │
│  │  │                                                  │   │    │
│  │  │  • TraceLoggerAdapter                           │   │    │
│  │  │    └─> server/trace.TraceLogger                 │   │    │
│  │  │                                                  │   │    │
│  │  │  • EventEmitterAdapter                          │   │    │
│  │  │    └─> chan StreamEvent                         │   │    │
│  │  │                                                  │   │    │
│  │  │  • BudgetTrackerAdapter                         │   │    │
│  │  │    └─> server/services.BudgetTracker            │   │    │
│  │  │                                                  │   │    │
│  │  └──────────────────────────────────────────────────┘   │    │
│  │                                                          │    │
│  │  ┌──────────────────────────────────────────────────┐   │    │
│  │  │              planner.go                          │   │    │
│  │  │                                                  │   │    │
│  │  │  • LLM-based plan generation                    │   │    │
│  │  │  • Uses provider.ModelProvider                  │   │    │
│  │  │  • Implements orchestration.PlannerInterface    │   │    │
│  │  └──────────────────────────────────────────────────┘   │    │
│  │                                                          │    │
│  │  ┌──────────────────────────────────────────────────┐   │    │
│  │  │         gateway_executor.go                      │   │    │
│  │  │                                                  │   │    │
│  │  │  • Converts gateway.ExecutionPlan to Plan       │   │    │
│  │  │  • Implements gateway.OrchestrationExecutor     │   │    │
│  │  └──────────────────────────────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                   │
└───────────────────────────┬───────────────────────────────────────┘
                            │ Implements Interfaces
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│               orchestration/ (Standalone Package)                │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     planner.go                            │   │
│  │                                                           │   │
│  │  • PlannerInterface                                      │   │
│  │  • ToolInvokerInterface                                  │   │
│  │  • TraceLoggerInterface                                  │   │
│  │  • EventEmitterInterface                                 │   │
│  │  • BudgetTrackerInterface                                │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     engine.go                             │   │
│  │                                                           │   │
│  │  • Engine - orchestrates plan execution                  │   │
│  │  • Parallel node execution                               │   │
│  │  • Auto-replanning on failure                            │   │
│  │  • Circuit breakers                                      │   │
│  │  • Retry logic with exponential backoff                  │   │
│  │  • Budget enforcement                                    │   │
│  │  • Disposition-aware pacing                              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                      task.go                              │   │
│  │                                                           │   │
│  │  • Task - high-level user request                        │   │
│  │  • Plan - DAG of PlanNodes                               │   │
│  │  • PlanNode - individual tool invocation                 │   │
│  │  • NodeState - execution state tracking                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  Dependencies: context, fmt, time, sync, encoding/json,          │
│                pkg/disposition (for pacing config)               │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

## Package Responsibilities

### `orchestration/` (Standalone Package)

**Purpose**: Pure orchestration logic with zero server dependencies

**Key Components**:
- `Engine`: Executes plans with retry, replanning, and circuit breakers
- `Task`, `Plan`, `PlanNode`: Core data structures
- Interfaces: Define contracts for external dependencies

**Dependencies**:
- Standard library only
- `pkg/disposition` for pacing configuration (easily removable if extracting)

**Can be used**:
- As part of AgenticGateway
- As a standalone library in other projects
- Extracted to separate repository

### `server/orchestration/` (Adapter Layer)

**Purpose**: Bridge between AgenticGateway components and standalone orchestration

**Key Components**:
- `adapters.go`: Concrete implementations of orchestration interfaces
- `planner.go`: LLM-based planner using AgenticGateway's provider system
- `gateway_executor.go`: Converts gateway types to orchestration types

**Dependencies**:
- `orchestration/` (standalone package)
- `server/` components (trace, services, config)
- `tools/` registry
- `provider/` for LLM access

**Cannot be extracted**: Tightly coupled to AgenticGateway server

## Data Flow

### 1. Request Initiation

```
User Request → HTTP Handler → Server.handleOrchestrate()
```

### 2. Task Creation

```go
// In server/handle_orchestrate.go
task := orchestrationpkg.NewTask(userID, taskDescription)
```

### 3. Plan Generation

```go
// Planner (server/orchestration/planner.go) generates DAG
plan, err := planner.GeneratePlan(ctx, task)

// Returns orchestration.Plan with PlanNodes:
// Plan {
//   Nodes: [
//     {ID: "node1", ToolName: "search", Dependencies: []},
//     {ID: "node2", ToolName: "analyze", Dependencies: ["node1"]},
//   ]
// }
```

### 4. Engine Execution

```go
// Engine (orchestration/engine.go) executes plan
err := engine.Execute(ctx, plan, task, userID)

// Engine orchestrates:
// 1. Find nodes with satisfied dependencies
// 2. Execute them in parallel (up to MaxParallelNodes)
// 3. Invoke tools via ToolInvokerAdapter
// 4. Handle errors (retry/replan/abort)
// 5. Update node states
// 6. Emit events via EventEmitterAdapter
// 7. Track traces via TraceLoggerAdapter
// 8. Enforce budgets via BudgetTrackerAdapter
```

### 5. Tool Invocation

```
Engine → ToolInvokerAdapter → tools.Registry → Tool.Function()
```

### 6. Result Collection

```go
// Completed nodes have results
for _, node := range plan.Nodes {
    if node.State == NodeStateSuccess {
        result := node.Result // map[string]interface{}
    }
}
```

## Interface Contracts

### PlannerInterface

```go
type PlannerInterface interface {
    // GeneratePlan creates an initial execution plan from a task
    GeneratePlan(ctx context.Context, task *Task) (*Plan, error)

    // RegeneratePlan creates a new plan after a failure, preserving completed work
    RegeneratePlan(ctx context.Context, task *Task, failedPlan *Plan, errorContext string) (*Plan, error)
}
```

**Implementation**: `server/orchestration/planner.go`
- Uses LLM to generate DAGs from natural language tasks
- Validates tool names against registry
- Preserves completed nodes during replanning

### ToolInvokerInterface

```go
type ToolInvokerInterface interface {
    // InvokeTool executes a tool by name with given parameters
    InvokeTool(ctx context.Context, toolName string, parameters map[string]interface{}) (map[string]interface{}, error)
}
```

**Implementation**: `server/orchestration/adapters.go#ToolInvokerAdapter`
- Looks up tools in `tools.Registry`
- Executes `tool.Function(ctx, parameters)`
- Wraps errors with context

### TraceLoggerInterface

```go
type TraceLoggerInterface interface {
    StartSpan(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error)
    EndSpan(ctx context.Context, span SpanHandle, metadata map[string]interface{}) error
    FailSpan(ctx context.Context, span SpanHandle, errorMsg string) error
}
```

**Implementation**: `server/orchestration/adapters.go#TraceLoggerAdapter`
- Wraps `server/trace.TraceLogger`
- Provides observability for orchestration operations
- Can be nil (optional)

### EventEmitterInterface

```go
type EventEmitterInterface interface {
    // Emit sends a stream event for monitoring
    Emit(event StreamEvent)
}
```

**Implementation**: `server/orchestration/adapters.go#EventEmitterAdapter`
- Sends events to channel for SSE streaming
- Non-blocking (drops events if channel full)
- Can be nil (optional)

### BudgetTrackerInterface

```go
type BudgetTrackerInterface interface {
    // GetRemaining returns the remaining token budget for a user
    GetRemaining(userID string) (int, error)
}
```

**Implementation**: `server/orchestration/adapters.go#BudgetTrackerAdapter`
- Wraps `server/services.BudgetTracker`
- Returns default 1,000,000 if nil
- Engine checks before executing nodes

## Error Handling Strategy

### Error Classification

The engine classifies errors into three categories:

```go
type ErrorType string

const (
    ErrorTypeTransient  ErrorType = "transient"   // Retry with backoff
    ErrorTypePersistent ErrorType = "persistent"  // Trigger replanning
    ErrorTypeFatal      ErrorType = "fatal"       // Abort immediately
)
```

### Classification Logic

```go
func (e *Engine) classifyError(err error) ErrorType {
    errMsg := strings.ToLower(err.Error())

    // Fatal errors
    if strings.Contains(errMsg, "budget exceeded") ||
       strings.Contains(errMsg, "forbidden") ||
       strings.Contains(errMsg, "unauthorized") {
        return ErrorTypeFatal
    }

    // Transient errors
    if strings.Contains(errMsg, "timeout") ||
       strings.Contains(errMsg, "rate limit") ||
       strings.Contains(errMsg, "connection refused") ||
       strings.Contains(errMsg, "service unavailable") {
        return ErrorTypeTransient
    }

    // Default to persistent (trigger replanning)
    return ErrorTypePersistent
}
```

### Error Flow

```
Node Execution Fails
    ↓
Classify Error
    ↓
┌───────────────┬──────────────────┬──────────────┐
│   Transient   │   Persistent     │    Fatal     │
│               │                  │              │
│ Retry with    │ Auto-Replanning  │ Abort        │
│ exponential   │ (if enabled)     │ Execution    │
│ backoff       │                  │              │
│               │ ┌──────────────┐ │              │
│ Up to         │ │ 1. Preserve  │ │              │
│ MaxRetries    │ │    completed │ │              │
│               │ │    nodes      │ │              │
│ If all fail → │ │ 2. Generate  │ │              │
│ Persistent    │ │    new plan  │ │              │
│               │ │ 3. Execute   │ │              │
│               │ └──────────────┘ │              │
└───────────────┴──────────────────┴──────────────┘
```

## Circuit Breaker Pattern

Prevents cascading failures by tracking tool health:

```go
// Per-tool failure tracking
type circuitBreakerState struct {
    failureCount int
    lastFailure  time.Time
    isOpen       bool
}

// Before executing a tool:
if state.isOpen {
    if time.Since(state.lastFailure) < CircuitBreakerTimeout {
        return errors.New("circuit breaker open")
    }
    // Reset after timeout
    state.isOpen = false
    state.failureCount = 0
}

// After tool failure:
state.failureCount++
if state.failureCount >= CircuitBreakerThreshold {
    state.isOpen = true
}
```

## Disposition-Aware Pacing

The engine supports different execution paces based on agent disposition:

```go
// Create engine with disposition
engine := orchestration.NewEngine(
    config,
    planner,
    toolInvoker,
    tracer,
    emitter,
    budgetTracker,
    orchestration.WithDisposition(dispositionConfig),
)
```

**Pacing Delays**:
- `rapid`: 0ms (no delay between nodes)
- `responsive`: 100ms
- `measured`: 500ms
- `deliberate`: 2000ms

Applied between node executions for controlled orchestration speed.

## Testing Strategy

### Unit Tests

**orchestration/** (Standalone):
- Test engine logic in isolation
- Use mock implementations of interfaces
- 28 tests covering all scenarios

**server/orchestration/** (Adapters):
- Test adapter implementations
- Test planner with mock LLM responses
- 17 tests for adapter behavior

### Integration Tests

**integration_test.go**:
- Test end-to-end workflows
- Verify disposition integration
- Test memory compression integration
- 6 comprehensive integration tests

## Migration Guide (for developers)

### Before v0.3.0

```go
import "github.com/.../server/orchestration"

engine := orchestration.NewEngine(...)
task := orchestration.NewTask(...)
```

### After v0.3.0

```go
import (
    orchestrationpkg "github.com/.../orchestration"
    serverorch "github.com/.../server/orchestration"
)

// Use orchestrationpkg for core types
task := orchestrationpkg.NewTask(...)

// Use server/orchestration for adapters and planner
planner := serverorch.NewPlanner(...)
adapter := serverorch.NewToolInvokerAdapter()

// Create engine with standalone package
engine := orchestrationpkg.NewEngine(config, planner, adapter, ...)
```

## Future Enhancements

### Potential Improvements

1. **Execution Cancellation**
   - Track running executions by ID
   - Implement `Cancel(executionID)` method
   - Graceful shutdown of in-progress nodes

2. **Execution Persistence**
   - Save execution state to database
   - Resume interrupted executions
   - Execution history and replay

3. **Advanced Scheduling**
   - Priority-based node execution
   - Resource-aware scheduling
   - Time-based execution windows

4. **Metrics & Observability**
   - Prometheus metrics export
   - Detailed execution timelines
   - Performance analytics

5. **Plan Optimization**
   - Cost-based optimization
   - Parallel execution hints
   - Plan caching

## Conclusion

The v0.3.0 orchestration architecture provides:

✅ **Clean Separation**: Standalone logic + thin adapters
✅ **Testability**: Comprehensive test coverage with clear boundaries
✅ **Reusability**: Can be extracted and used in other projects
✅ **Maintainability**: Single source of truth for orchestration logic
✅ **Extensibility**: Easy to add new implementations via adapters

This architecture positions AgenticGateway for future growth while maintaining code quality and developer productivity.
