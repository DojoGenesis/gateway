# Orchestration Package

A standalone, interface-driven orchestration engine for executing DAG-based task plans with built-in auto-replanning, circuit breakers, and disposition-aware pacing.

## Overview

The `orchestration` package provides a robust execution engine for coordinating multi-step workflows represented as Directed Acyclic Graphs (DAGs). It's designed to be:

- **Standalone**: Zero dependencies on application-specific code (only imports `pkg/disposition`)
- **Interface-driven**: All external dependencies are injected via interfaces
- **Testable**: Comprehensive test coverage with mock implementations
- **Production-ready**: Built-in retry logic, circuit breakers, and auto-replanning

## Architecture

### Core Types

- **`Task`**: Represents a high-level user request to be executed
- **`Plan`**: A DAG of `PlanNode` instances that decompose a task
- **`PlanNode`**: A single step in the plan (tool invocation) with dependencies
- **`Engine`**: Orchestrates plan execution with parallel node execution, retries, and replanning

### Interfaces

The engine depends on five clean interfaces that must be implemented by consumers:

1. **`PlannerInterface`**: Generates and regenerates execution plans
   ```go
   type PlannerInterface interface {
       GeneratePlan(ctx context.Context, task *Task) (*Plan, error)
       RegeneratePlan(ctx context.Context, task *Task, failedPlan *Plan, errorContext string) (*Plan, error)
   }
   ```

2. **`ToolInvokerInterface`**: Executes individual tools
   ```go
   type ToolInvokerInterface interface {
       InvokeTool(ctx context.Context, toolName string, parameters map[string]interface{}) (map[string]interface{}, error)
   }
   ```

3. **`TraceLoggerInterface`**: Provides observability and tracing
   ```go
   type TraceLoggerInterface interface {
       StartSpan(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error)
       EndSpan(ctx context.Context, span SpanHandle, metadata map[string]interface{}) error
       FailSpan(ctx context.Context, span SpanHandle, errorMsg string) error
   }
   ```

4. **`EventEmitterInterface`**: Emits execution events for monitoring
   ```go
   type EventEmitterInterface interface {
       Emit(event StreamEvent)
   }
   ```

5. **`BudgetTrackerInterface`**: Enforces token/cost budgets
   ```go
   type BudgetTrackerInterface interface {
       GetRemaining(userID string) (int, error)
   }
   ```

All interfaces are optional (can be nil). The engine gracefully degrades when optional components are not provided.

## Usage

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
)

func main() {
    // 1. Create engine configuration
    config := orchestration.DefaultEngineConfig()
    config.MaxParallelNodes = 10
    config.EnableAutoReplanning = true

    // 2. Implement required interfaces (or use adapters)
    var planner orchestration.PlannerInterface = &MyPlanner{}
    var toolInvoker orchestration.ToolInvokerInterface = &MyToolInvoker{}
    var tracer orchestration.TraceLoggerInterface = nil  // Optional
    var emitter orchestration.EventEmitterInterface = nil  // Optional
    var budgetTracker orchestration.BudgetTrackerInterface = nil  // Optional

    // 3. Create engine instance
    engine := orchestration.NewEngine(
        config,
        planner,
        toolInvoker,
        tracer,
        emitter,
        budgetTracker,
    )

    // 4. Create a task
    task := orchestration.NewTask("user-123", "Analyze sales data and generate report")

    // 5. Generate a plan (planner creates the DAG)
    ctx := context.Background()
    plan, err := planner.GeneratePlan(ctx, task)
    if err != nil {
        panic(err)
    }

    // 6. Execute the plan
    err = engine.Execute(ctx, plan, task, "user-123")
    if err != nil {
        fmt.Printf("Execution failed: %v\n", err)
    } else {
        fmt.Println("Execution completed successfully!")
    }
}
```

### With Disposition-Aware Pacing

```go
import "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"

// Create disposition config
dispConfig := &disposition.DispositionConfig{
    Pacing: "deliberate",  // "rapid", "responsive", "measured", "deliberate"
    Depth:  "thorough",
}

// Create engine with disposition
engine := orchestration.NewEngine(
    orchestration.DefaultEngineConfig(),
    planner,
    toolInvoker,
    tracer,
    emitter,
    budgetTracker,
    orchestration.WithDisposition(dispConfig),  // Apply pacing
)
```

Pacing affects delays between node executions:
- **rapid**: 0ms (no delay)
- **responsive**: 100ms
- **measured**: 500ms
- **deliberate**: 2000ms

### Creating a Plan

Plans are DAGs where nodes can execute in parallel if they have no dependencies:

```go
plan := orchestration.NewPlan("task-123")

// Add nodes to the plan
node1 := &orchestration.PlanNode{
    ID:           "fetch-data",
    ToolName:     "database.query",
    Parameters:   map[string]interface{}{"query": "SELECT * FROM sales"},
    Dependencies: []string{},  // No dependencies, can run immediately
    State:        orchestration.NodeStatePending,
}

node2 := &orchestration.PlanNode{
    ID:           "transform-data",
    ToolName:     "data.transform",
    Parameters:   map[string]interface{}{"format": "csv"},
    Dependencies: []string{"fetch-data"},  // Depends on node1
    State:        orchestration.NodeStatePending,
}

node3 := &orchestration.PlanNode{
    ID:           "generate-report",
    ToolName:     "report.create",
    Parameters:   map[string]interface{}{"template": "monthly"},
    Dependencies: []string{"transform-data"},  // Depends on node2
    State:        orchestration.NodeStatePending,
}

plan.Nodes = []*orchestration.PlanNode{node1, node2, node3}

// Validate the DAG structure
if err := plan.ValidateDAG(); err != nil {
    panic(err)
}
```

## Features

### Auto-Replanning

When a node fails with a retriable error, the engine can automatically generate a new plan that preserves completed work:

```go
config := orchestration.DefaultEngineConfig()
config.EnableAutoReplanning = true
config.MaxReplanningAttempts = 2
```

### Circuit Breakers

Prevent cascading failures by opening circuits for repeatedly failing tools:

```go
config.EnableCircuitBreaker = true
config.CircuitBreakerThreshold = 5     // Open after 5 failures
config.CircuitBreakerTimeout = 60 * time.Second  // Reset after 60s
```

### Retry Logic

Transient errors trigger exponential backoff retries:

```go
config.MaxRetries = 3
config.RetryBackoff = 1 * time.Second
config.MaxBackoff = 30 * time.Second
config.EnableJitter = true  // Add randomization to prevent thundering herd
```

### Error Classification

The engine classifies errors to determine retry/replan/abort strategies:

- **Transient**: Retry with backoff (timeouts, rate limits, connection issues)
- **Persistent**: Trigger replanning (invalid parameters, tool failures)
- **Fatal**: Abort immediately (budget exceeded, authorization failures)

### Parallel Execution

Nodes with satisfied dependencies execute in parallel:

```go
config.MaxParallelNodes = 10  // Up to 10 nodes can run concurrently
```

### Budget Enforcement

Prevent runaway costs by enforcing token budgets:

```go
// Engine checks budget before executing nodes
// Aborts execution if user budget is exceeded
```

## Node States

Nodes transition through these states during execution:

- `NodeStatePending`: Initial state, waiting for dependencies
- `NodeStateRunning`: Currently executing
- `NodeStateSuccess`: Completed successfully
- `NodeStateFailed`: Execution failed
- `NodeStateSkipped`: Skipped due to upstream failure

## Testing

The package includes comprehensive tests:

```bash
go test ./orchestration/... -v
```

Test coverage includes:
- Parallel node execution
- Auto-replanning on failure
- Circuit breaker behavior
- Budget enforcement
- Error classification
- Disposition-aware pacing
- Context cancellation
- DAG validation

## Integration

### As Part of Agentic Gateway by Dojo Genesis

The `server/orchestration/` package provides adapter implementations that bridge this standalone package with the Agentic Gateway server components.

### As Standalone Package

This package can be extracted to a separate repository and used independently. The only non-standard dependency is `pkg/disposition` for pacing configuration.

## Configuration Reference

### EngineConfig

```go
type EngineConfig struct {
    MaxRetries              int           // Number of retries for transient errors (default: 3)
    RetryBackoff            time.Duration // Initial backoff duration (default: 1s)
    MaxBackoff              time.Duration // Maximum backoff duration (default: 30s)
    MaxParallelNodes        int           // Max concurrent node execution (default: 5)
    EnableAutoReplanning    bool          // Enable automatic replanning (default: true)
    MaxReplanningAttempts   int           // Max replanning attempts (default: 2)
    EnableJitter            bool          // Add jitter to backoff (default: true)
    EnableCircuitBreaker    bool          // Enable circuit breakers (default: true)
    CircuitBreakerThreshold int           // Failures before opening (default: 5)
    CircuitBreakerTimeout   time.Duration // Reset timeout (default: 60s)
}
```

Use `orchestration.DefaultEngineConfig()` to get sensible defaults.

## License

Part of the **Agentic Gateway by Dojo Genesis** project. See main repository for license information.
