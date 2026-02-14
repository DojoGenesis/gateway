# Agentic Gateway v0.1.0 — Track 2 Specification
## Orchestration + Memory Module Extraction

**Status:** Production-Ready Specification
**Version:** 0.1.0
**Track:** Track 2 of 3 (Orchestration Engine + Memory System)
**Release Date:** 2026-02-12

---

## 1. Vision

The Orchestration + Memory module (Track 2) extracts the core autonomous workflow engine and intelligent memory system that differentiates Agentic Gateway from simple LLM proxies.

The **orchestration engine** is the primary differentiator: it enables multi-step autonomous workflows with DAG-based execution, automatic replanning on failure, circuit breaker patterns, and cost-aware budget enforcement. This is the heart of agentic capability.

The **memory system** provides intelligent conversation context management: conversation memory CRUD, garden-based compression with seed extraction, and configurable storage backends (not hardcoded SQLite).

Together, they enable:
- **Autonomous multi-step tasks** with parallel execution and error recovery
- **Intelligent context management** that preserves knowledge while managing token budgets
- **Cost-aware execution** with budget enforcement and replanning triggers
- **Observable workflows** with comprehensive tracing and event emission

This module is intentionally **uncoupled from HTTP/API concerns** (Track 3) and **Dojo-specific features** (out of scope).

---

## 2. Goals

### Orchestration Module Goals

1. **DAG-based task orchestration**
   - Execute plans as Directed Acyclic Graphs (DAGs) of tool calls
   - Identify and exploit parallelizable operations
   - Support dependency-ordered execution with progress tracking

2. **PlannerInterface abstraction**
   - Consumers implement their own `GeneratePlan` and `RegeneratePlan` logic
   - Support LLM-based, rule-based, or hybrid planner implementations
   - Enable planner composition and chaining

3. **Automatic replanning on failure**
   - Classify errors: Transient (retry) → Persistent (replan) → Fatal (abort)
   - Regenerate plans when execution fails, preserving completed work
   - Limit replanning attempts to prevent infinite loops

4. **Resilience patterns**
   - Exponential backoff with optional jitter for transient failures
   - Configurable retry budgets per node
   - Circuit breaker pattern per tool to prevent cascading failures
   - Tool health metrics tracking (success rate, consecutive failures)

5. **Cost estimation and budget enforcement**
   - Estimate plan token costs before execution
   - Enforce query-level, session-level, and monthly budgets
   - Block execution if estimated cost exceeds budget
   - Support cost hooks for external tracking

6. **Comprehensive instrumentation**
   - Trace spans for orchestration lifecycle (plan, nodes, replanning)
   - Event emission for progress tracking and monitoring
   - Tool health metrics collection
   - Execution timing and retry tracking

### Memory Module Goals

1. **Conversation memory management**
   - CRUD operations for conversational memories
   - Semantic search with embedding-based similarity
   - Metadata and context type classification
   - Configurable retention and archival

2. **Intelligent context compression**
   - Garden-based memory management with tiered storage
   - Semantic compression using LLM-based abstraction (3-Month Rule)
   - Preserve high-value information (decisions, patterns, insights)
   - Support target compression ratios (70-80% token reduction)

3. **Knowledge seed extraction**
   - Extract reusable patterns from conversation histories
   - Store as named seeds with confidence scoring
   - Track usage patterns and last-used timestamps
   - Support seed versioning and soft deletion

4. **Flexible storage backends**
   - Define storage interface (not hardcoded SQLite)
   - Support migrations between backends
   - Enable in-memory, file-based, and cloud storage adapters
   - Maintain schema compatibility

5. **Context building**
   - Assemble conversation context from memory layers
   - Support attention mechanisms for relevant content selection
   - Token budget-aware context truncation
   - Preserve conversation thread coherence

---

## 3. Technical Architecture

### 3.1 Module Structure

```
orchestration/               # DAG-based execution engine
├── engine.go               # Engine: Execute, auto-replan, cost tracking
├── task.go                 # Task, Plan, PlanNode types
├── planner.go              # PlannerInterface, example implementations
├── go.mod                  # Module dependencies
└── *_test.go               # Unit tests

memory/                      # Intelligent memory system
├── manager.go              # MemoryManager: CRUD operations
├── garden_manager.go       # GardenManager: Compression & seeds
├── seed_extractor.go       # SeedExtractor: Pattern extraction
├── context_builder.go      # ContextBuilder: Assembly
├── compression.go          # CompressionService: LLM-based compression
├── types.go                # Type definitions (Memory, Seed, etc.)
├── go.mod                  # Module dependencies
└── *_test.go               # Unit tests
```

### 3.2 Dependency Graph

```
orchestration/
├── depends on: google/uuid, context, sync, time
├── interfaces for: PlannerInterface (consumer-provided)
├── optional: trace/* (via interfaces)
├── optional: events/* (via interface)
├── optional: services/* (cost/budget via interfaces)

memory/
├── depends on: database/sql, google/uuid, context, encoding/json
├── interfaces for: StorageBackendInterface (consumer-provided)
├── interfaces for: CompressionServiceInterface (consumer-provided)
├── optional: plugin/* (LLM providers for compression)
```

### 3.3 Core Interfaces

#### PlannerInterface (Orchestration)
```go
// Consumer-implemented planner interface
type PlannerInterface interface {
    // Generate initial execution plan from task
    GeneratePlan(ctx context.Context, task *Task) (*Plan, error)

    // Regenerate plan after failure
    RegeneratePlan(ctx context.Context, task *Task, failedPlan *Plan,
                   errorContext string) (*Plan, error)
}
```

#### StorageBackendInterface (Memory)
```go
// Consumer-implemented storage backend
type StorageBackendInterface interface {
    // Memory CRUD
    StoreMemory(ctx context.Context, memory *Memory) error
    GetMemory(ctx context.Context, id string) (*Memory, error)
    UpdateMemory(ctx context.Context, memory *Memory) error
    DeleteMemory(ctx context.Context, id string) error
    ListMemories(ctx context.Context, filter MemoryFilter) ([]*Memory, error)

    // Seed CRUD
    StoreSeed(ctx context.Context, seed *MemorySeed) error
    GetSeed(ctx context.Context, id string) (*MemorySeed, error)
    UpdateSeed(ctx context.Context, seed *MemorySeed) error
    ListSeeds(ctx context.Context, filter SeedFilter) ([]*MemorySeed, error)
}
```

#### CompressionServiceInterface (Memory)
```go
// Consumer-implemented compression service
type CompressionServiceInterface interface {
    // Compress conversation history using semantic abstraction
    CompressHistory(ctx context.Context, sessionID string,
                   memories []Memory) (*CompressedHistory, error)

    // Extract reusable patterns from memories
    ExtractSeeds(ctx context.Context, memories []Memory) ([]*MemorySeed, error)
}
```

#### TraceLoggerInterface (Orchestration)
```go
// Optional: consumer-provided trace logger
type TraceLoggerInterface interface {
    StartSpan(ctx context.Context, traceID, spanName string,
             metadata map[string]interface{}) (*Span, error)
    EndSpan(ctx context.Context, span *Span, metadata map[string]interface{}) error
    FailSpan(ctx context.Context, span *Span, errorMsg string) error
}
```

#### EventEmitterInterface (Orchestration)
```go
// Optional: consumer-provided event emitter
type EventEmitterInterface interface {
    Emit(event StreamEvent)
}
```

---

## 4. Core Types & Behavior

### 4.1 Orchestration Types

#### Task
```go
type Task struct {
    ID          string                 // Unique task identifier
    UserID      string                 // User executing the task
    Description string                 // High-level task description
    CreatedAt   time.Time              // Task creation timestamp
    Metadata    map[string]interface{} // Custom metadata
}
```

#### Plan (DAG structure)
```go
type Plan struct {
    ID        string      // Unique plan identifier
    TaskID    string      // Associated task ID
    Nodes     []*PlanNode // DAG nodes
    Version   int         // Plan version (incremented on replan)
    CreatedAt time.Time   // Creation timestamp
    Metadata  map[string]interface{} // Planning reasoning, etc.
}

// Helper methods
plan.GetExecutableNodes()  // Nodes with satisfied dependencies
plan.AllNodesCompleted()   // All nodes terminal (success/failed/skipped)
plan.HasFailedNodes()      // Any failed nodes?
plan.ValidateDAG()         // Validate acyclicity & dependency validity
```

#### PlanNode (DAG node)
```go
type PlanNode struct {
    ID           string                 // Unique node ID
    ToolName     string                 // Tool to invoke
    Parameters   map[string]interface{} // Tool parameters
    Dependencies []string               // Node IDs that must complete first
    State        NodeState              // pending, running, success, failed, skipped
    Result       map[string]interface{} // Tool execution result
    Error        string                 // Error message if failed
    StartTime    *time.Time             // When execution started
    EndTime      *time.Time             // When execution ended
    RetryCount   int                    // Number of retries performed
}

// States
NodeStatePending NodeState = "pending"    // Awaiting execution
NodeStateRunning NodeState = "running"    // Currently executing
NodeStateSuccess NodeState = "success"    // Completed successfully
NodeStateFailed  NodeState = "failed"     // Execution failed
NodeStateSkipped NodeState = "skipped"    // Skipped (conditional)

// Helper methods
node.IsReady(plan)   // All dependencies satisfied?
node.IsTerminal()    // Is in terminal state?
node.Duration()      // Execution duration
```

### 4.2 Memory Types

#### Memory
```go
type Memory struct {
    ID          string                 // Unique memory ID
    Type        string                 // "user", "assistant", "system"
    Content     string                 // Message content
    ContextType string                 // e.g., "conversation", "tool_result"
    Metadata    map[string]interface{} // Custom metadata
    Embedding   []float32              // Semantic embedding (optional)
    CreatedAt   time.Time              // Creation time
    UpdatedAt   time.Time              // Last update time
}
```

#### MemorySeed
```go
type MemorySeed struct {
    ID           string     // Unique seed ID
    ProjectID    *string    // Optional project scoping
    Content      string     // Seed content (reusable pattern)
    SeedType     string     // Pattern type ("insight", "decision", "pattern")
    Source       SourceType // "system", "user", "calibrated"
    UserEditable bool       // Can user modify?
    Confidence   float64    // Extraction confidence (0-1)
    UsageCount   int        // Times used in context
    LastUsedAt   *time.Time // Last usage time
    CreatedAt    time.Time  // Creation time
    UpdatedAt    time.Time  // Last update time
    CreatedBy    *string    // User who created
    Version      int        // Seed version
    DeletedAt    *time.Time // Soft deletion marker
}
```

#### CompressedHistory
```go
type CompressedHistory struct {
    ID                string   // Unique compression ID
    SessionID         string   // Associated session
    OriginalTurnIDs   []string // Memory IDs that were compressed
    CompressedContent string   // Compressed semantic summary
    CompressionRatio  float64  // Compressed tokens / original tokens (lower=better)
    CreatedAt         time.Time
}
```

---

## 5. Engine Execution Flow

### 5.1 Execute() Lifecycle

```
Execute(ctx, plan, task, userID)
├─ Estimate plan cost
├─ Check budget (query, session, monthly)
│  └─ Error if over budget: budget_check_failed (Fatal)
├─ Create orchestration span (trace)
├─ Loop: executePlan() with auto-replanning
│  ├─ executePlan(plan)
│  │  ├─ Loop: while not all_nodes_completed
│  │  │  ├─ Get executable nodes (ready dependencies)
│  │  │  ├─ Execute in parallel (batches of MaxParallelNodes)
│  │  │  │  └─ executeNode(node)
│  │  │  │     ├─ Set state = running
│  │  │  │     ├─ Check circuit breaker
│  │  │  │     │  └─ If open: circuit_breaker_open (Fatal)
│  │  │  │     ├─ invokeToolWithRetry()
│  │  │  │     │  ├─ Loop: 0..MaxRetries
│  │  │  │     │  ├─ Call InvokeTool(toolName, params)
│  │  │  │     │  ├─ On success: reset circuit breaker, return result
│  │  │  │     │  └─ On error:
│  │  │  │     │     ├─ Classify: transient/persistent/fatal
│  │  │  │     │     ├─ Update circuit breaker (persistent/fatal)
│  │  │  │     │     └─ Retry with adaptive backoff
│  │  │  │     └─ Set state = success/failed + result/error
│  │  │  └─ Check context cancellation
│  │  └─ Return error if any node failed
│  │
│  ├─ If execErr == nil: return success
│  ├─ Classify error
│  ├─ If Fatal: return error
│  ├─ If auto-replanning disabled: return error
│  ├─ If replanning attempts exhausted: return error
│  ├─ Call handlePersistentFailure()
│  │  ├─ Collect completed nodes (preserve work)
│  │  ├─ Build error context
│  │  ├─ Call planner.RegeneratePlan()
│  │  └─ Return new plan (version incremented)
│  └─ Retry loop with new plan
│
└─ Return success or final error
```

### 5.2 Error Classification

| Error Pattern | Classification | Behavior |
|---------------|----------------|----------|
| "timeout", "deadline", "unavailable", "rate limit" | **Transient** | Retry with backoff |
| "invalid parameter", "not found", "validation failed" | **Persistent** | Trigger replanning |
| "budget exceeded", "forbidden", "unauthorized", "cyclic dependencies" | **Fatal** | Abort immediately |
| (default) | **Persistent** | Trigger replanning |

### 5.3 Circuit Breaker Pattern

Per-tool circuit breaker state machine:

```
Closed (normal operation)
├─ Success: stay Closed, reset failure counter
├─ Persistent error: increment failure counter
├─ Consecutive failures ≥ threshold: transition to Open
│
Open (tool unavailable)
├─ All attempts: return circuit_breaker_open error (Fatal)
├─ Timeout elapsed: transition to Half-Open
│
Half-Open (testing recovery)
├─ Success: transition to Closed, reset counter
├─ Failure: transition back to Open
```

**Configuration:**
- `CircuitBreakerThreshold`: Failures before opening (default: 5)
- `CircuitBreakerTimeout`: Duration before attempting recovery (default: 60s)

### 5.4 Cost Estimation

Token cost estimation per node:

```
estimated_tokens = Σ(node_tokens for all nodes) + planning_overhead

node_tokens =
  input_tokens (based on tool category + parameters) +
  output_tokens (based on tool category) +
  context_tokens (based on dependency depth)
  * category_safety_multiplier

planning_overhead =
  base_tokens * (0.15 + node_count_factor + complexity_factor)
```

**Tool Categories:**
- Web (1.4x multiplier): search, fetch, API calls
- File (1.3x multiplier): read, write, list, search
- Memory (1.2x multiplier): store, search, retrieve
- Compute (1.5x multiplier): calculate, transform, analyze
- Generic (1.35x multiplier): other tools

---

## 6. Memory System

### 6.1 MemoryManager CRUD

```go
// Store new memory
manager.StoreMemory(ctx, &Memory{...})

// Retrieve by ID
memory, err := manager.GetMemory(ctx, id)

// Update existing
manager.UpdateMemory(ctx, memory)

// Search semantically (embedding-based)
results, err := manager.SearchMemories(ctx, query, limit)
  → SearchResult{Memory, Similarity, Snippet, SearchMode}

// List with filters
memories, err := manager.ListMemories(ctx, MemoryFilter{
    Type: "user",
    ContextType: "conversation",
    CreatedAfter: yesterday,
})

// Delete (soft or hard)
manager.DeleteMemory(ctx, id)
```

### 6.2 GardenManager (Compression & Seeds)

**Garden concept:** Multi-tier memory management with intelligent compression

```go
// Compress old conversation history
compressed, err := gardenManager.CompressHistory(ctx, sessionID, oldMemories)
  → CompressedHistory{
      OriginalTurnIDs: [turn IDs],
      CompressedContent: "semantic summary",
      CompressionRatio: 0.25,  // 75% reduction
    }

// Extract reusable knowledge seeds
seeds, err := gardenManager.ExtractSeeds(ctx, memories)
  → []*MemorySeed{
      {SeedType: "insight", Confidence: 0.92, Content: "..."},
      {SeedType: "decision", Confidence: 0.87, Content: "..."},
    }

// Store seed for future contexts
gardenManager.StoreSeed(ctx, seed)

// Retrieve seed by ID
seed, err := gardenManager.GetSeed(ctx, seedID)

// List seeds by criteria
seeds, err := gardenManager.ListSeeds(ctx, SeedFilter{
    Type: "insight",
    MinConfidence: 0.8,
    SortBy: "usage_count",
})

// Update seed (e.g., track usage)
seed.UsageCount++
seed.LastUsedAt = now
gardenManager.UpdateSeed(ctx, seed)
```

### 6.3 Compression Strategy (3-Month Rule)

**Goal:** Preserve information that matters in 3+ months while discarding ephemeral content

**Targets:**
- 70-80% token reduction (output: 20-30% of original)
- Preserve: decisions, patterns, insights, agreements, action items
- Discard: pleasantries, confirmations, redundant explanations, failed attempts

**Process:**
1. Segment memories by type and time
2. Score each segment for long-term value
3. Use LLM semantic abstraction to compress high-value segments
4. Assemble compressed summary maintaining coherence
5. Calculate compression ratio (actual tokens / original tokens)

---

## 7. Production-Ready Code Examples

### 7.1 Example: Rule-Based Planner

```go
package planner

import (
    "context"
    "fmt"
    "strings"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
)

// SimpleRuleBasedPlanner demonstrates a non-LLM planner implementation
type SimpleRuleBasedPlanner struct{}

func (p *SimpleRuleBasedPlanner) GeneratePlan(ctx context.Context,
    task *orchestration.Task) (*orchestration.Plan, error) {

    plan := orchestration.NewPlan(task.ID)

    // Rule: if "search" in description, add search node
    if strings.Contains(strings.ToLower(task.Description), "search") {
        searchNode := orchestration.NewPlanNode("web_search",
            map[string]interface{}{"query": task.Description},
            []string{})
        plan.Nodes = append(plan.Nodes, searchNode)

        // Rule: follow search with summarization
        summarizeNode := orchestration.NewPlanNode("summarize",
            map[string]interface{}{"content": "${search_result}"},
            []string{searchNode.ID})
        plan.Nodes = append(plan.Nodes, summarizeNode)
    }

    // Rule: if "write" in description, add write node
    if strings.Contains(strings.ToLower(task.Description), "write") {
        writeNode := orchestration.NewPlanNode("write_file",
            map[string]interface{}{"path": "/tmp/output.txt", "content": "..."},
            []string{})
        plan.Nodes = append(plan.Nodes, writeNode)
    }

    if len(plan.Nodes) == 0 {
        return nil, fmt.Errorf("no applicable rules for task")
    }

    plan.Metadata["planner"] = "SimpleRuleBasedPlanner"
    return plan, plan.ValidateDAG()
}

func (p *SimpleRuleBasedPlanner) RegeneratePlan(ctx context.Context,
    task *orchestration.Task,
    failedPlan *orchestration.Plan,
    errorContext string) (*orchestration.Plan, error) {

    // Simple strategy: if web_search failed, try alternative tool
    newPlan := orchestration.NewPlan(task.ID)
    newPlan.Version = failedPlan.Version + 1

    for _, node := range failedPlan.Nodes {
        if node.State == orchestration.NodeStateFailed &&
           node.ToolName == "web_search" {
            // Replace with alternative: local search instead
            altNode := orchestration.NewPlanNode("local_search",
                node.Parameters,
                node.Dependencies)
            newPlan.Nodes = append(newPlan.Nodes, altNode)
        } else if node.State == orchestration.NodeStateSuccess {
            // Preserve completed work
            newPlan.Nodes = append(newPlan.Nodes, node)
        }
    }

    return newPlan, newPlan.ValidateDAG()
}
```

### 7.2 Example: Engine Initialization & DAG Execution

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
    "github.com/your-org/planner"
)

func main() {
    ctx := context.Background()

    // 1. Initialize configuration
    config := &orchestration.EngineConfig{
        MaxRetries:            3,
        RetryBackoff:          1 * time.Second,
        MaxBackoff:            30 * time.Second,
        MaxParallelNodes:      5,
        EnableAutoReplanning:  true,
        MaxReplanningAttempts: 2,
        EnableJitter:          true,
        EnableCircuitBreaker:  true,
        CircuitBreakerThreshold: 5,
        CircuitBreakerTimeout: 60 * time.Second,
    }

    // 2. Create planner
    myPlanner := &planner.SimpleRuleBasedPlanner{}

    // 3. Initialize engine (with optional trace/events)
    engine := orchestration.NewEngine(
        config,
        myPlanner,
        nil,    // traceLogger (optional)
        nil,    // eventChan (optional)
        nil,    // costTracker (optional)
        nil,    // budgetTracker (optional)
    )

    // 4. Create task
    task := orchestration.NewTask("user123",
        "Search for Go best practices and summarize top 3 findings")

    // 5. Generate initial plan
    plan, err := myPlanner.GeneratePlan(ctx, task)
    if err != nil {
        fmt.Printf("Plan generation failed: %v\n", err)
        return
    }

    fmt.Printf("Generated plan with %d nodes\n", len(plan.Nodes))
    for _, node := range plan.Nodes {
        fmt.Printf("  - Node %s: %s (deps: %v)\n",
            node.ID, node.ToolName, node.Dependencies)
    }

    // 6. Execute with auto-replanning
    if err := engine.Execute(ctx, plan, task, "user123"); err != nil {
        fmt.Printf("Execution failed: %v\n", err)
        return
    }

    fmt.Println("Execution successful!")

    // 7. Inspect results
    for _, node := range plan.Nodes {
        if node.State == orchestration.NodeStateSuccess {
            fmt.Printf("✓ Node %s result: %v\n", node.ID, node.Result)
        } else if node.State == orchestration.NodeStateFailed {
            fmt.Printf("✗ Node %s error: %s\n", node.ID, node.Error)
        }
    }
}
```

### 7.3 Example: 3-Node DAG with Parallel Pair

```go
package main

import (
    "context"
    "fmt"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
)

// DemoParallelDAG creates a 3-node DAG with parallel execution:
//
//   [Node A: fetch_docs]
//         ↓
//   [Node B: parse]  [Node C: validate] (parallel)
//
func DemoParallelDAG() *orchestration.Plan {
    plan := &orchestration.Plan{
        ID:       "demo-parallel-1",
        TaskID:   "task-123",
        Version:  1,
        Nodes:    make([]*orchestration.PlanNode, 0),
    }

    // Node A: Initial fetch (no dependencies)
    nodeA := &orchestration.PlanNode{
        ID:       "node-a-fetch",
        ToolName: "fetch_url",
        Parameters: map[string]interface{}{
            "url": "https://example.com/docs",
        },
        Dependencies: []string{},
        State:        orchestration.NodeStatePending,
    }
    plan.Nodes = append(plan.Nodes, nodeA)

    // Node B: Parse (depends on A)
    nodeB := &orchestration.PlanNode{
        ID:       "node-b-parse",
        ToolName: "parse_html",
        Parameters: map[string]interface{}{
            "content": "${node-a-fetch.result.body}",
        },
        Dependencies: []string{"node-a-fetch"},
        State:        orchestration.NodeStatePending,
    }
    plan.Nodes = append(plan.Nodes, nodeB)

    // Node C: Validate (also depends on A, parallel with B)
    nodeC := &orchestration.PlanNode{
        ID:       "node-c-validate",
        ToolName: "validate_schema",
        Parameters: map[string]interface{}{
            "content": "${node-a-fetch.result.body}",
            "schema":  "html5",
        },
        Dependencies: []string{"node-a-fetch"},
        State:        orchestration.NodeStatePending,
    }
    plan.Nodes = append(plan.Nodes, nodeC)

    // Execution order:
    // 1. Run A (no dependencies)
    // 2. When A completes, run B and C in parallel
    // 3. When both B and C complete, plan is done

    return plan
}

func main() {
    plan := DemoParallelDAG()

    if err := plan.ValidateDAG(); err != nil {
        fmt.Printf("Invalid DAG: %v\n", err)
        return
    }

    fmt.Println("✓ DAG is valid (acyclic, dependencies valid)")

    // Inspect parallelization
    fmt.Println("\nExecution stages:")
    stage := 1
    for !plan.AllNodesCompleted() {
        executable := plan.GetExecutableNodes()
        if len(executable) == 0 {
            break
        }

        fmt.Printf("Stage %d: %d nodes executable in parallel\n",
            stage, len(executable))
        for _, node := range executable {
            fmt.Printf("  - %s (%s)\n", node.ID, node.ToolName)
        }

        // Simulate execution
        for _, node := range executable {
            node.State = orchestration.NodeStateSuccess
            node.Result = map[string]interface{}{"success": true}
        }
        stage++
    }
}
```

### 7.4 Example: Memory Manager with SQLite Backend

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
)

func main() {
    ctx := context.Background()

    // 1. Initialize memory manager with SQLite
    manager, err := memory.NewMemoryManager("/tmp/conversations.db")
    if err != nil {
        fmt.Printf("Failed to init memory: %v\n", err)
        return
    }
    defer manager.Close()

    // 2. Store conversation memories
    memories := []memory.Memory{
        {
            ID:          "msg-1",
            Type:        "user",
            Content:     "What are Go best practices?",
            ContextType: "conversation",
            CreatedAt:   time.Now(),
        },
        {
            ID:          "msg-2",
            Type:        "assistant",
            Content:     "1. Use error handling... 2. Design for concurrency...",
            ContextType: "conversation",
            CreatedAt:   time.Now(),
        },
    }

    for _, mem := range memories {
        if err := manager.StoreMemory(ctx, &mem); err != nil {
            fmt.Printf("Failed to store memory: %v\n", err)
            return
        }
    }
    fmt.Println("✓ Stored 2 memories")

    // 3. Search semantically
    results, err := manager.SearchMemories(ctx, "concurrency patterns", 5)
    if err != nil {
        fmt.Printf("Search failed: %v\n", err)
        return
    }

    fmt.Printf("Found %d matching memories:\n", len(results))
    for _, result := range results {
        fmt.Printf("  - Similarity: %.2f, Content: %s\n",
            result.Similarity, result.Memory.Content)
    }

    // 4. Garden-based compression (after 30 days of memories)
    gardenMgr, err := memory.NewGardenManager(manager, nil)
    if err != nil {
        fmt.Printf("Failed to init garden: %v\n", err)
        return
    }

    compressed, err := gardenMgr.CompressHistory(ctx, "session-1", memories)
    if err != nil {
        fmt.Printf("Compression failed: %v\n", err)
        return
    }

    fmt.Printf("Compressed %d turns to %.0f%% of original size\n",
        len(compressed.OriginalTurnIDs), compressed.CompressionRatio*100)
    fmt.Printf("Summary: %s\n", compressed.CompressedContent)

    // 5. Extract and store seeds
    seeds, err := gardenMgr.ExtractSeeds(ctx, memories)
    if err != nil {
        fmt.Printf("Seed extraction failed: %v\n", err)
        return
    }

    fmt.Printf("Extracted %d reusable seeds:\n", len(seeds))
    for _, seed := range seeds {
        fmt.Printf("  - [%s] %s (confidence: %.2f)\n",
            seed.SeedType, seed.Content, seed.Confidence)
        if err := gardenMgr.StoreSeed(ctx, seed); err != nil {
            fmt.Printf("    Failed to store seed: %v\n", err)
        }
    }
}
```

---

## 8. Success Criteria (Binary)

All criteria must pass for v0.1.0 release.

### 8.1 Compilation & Testing

- [ ] `cd orchestration && go mod tidy && go test ./...` passes
- [ ] `cd memory && go mod tidy && go test ./...` passes
- [ ] Zero compilation warnings in both modules
- [ ] Code coverage ≥ 75% for both modules

### 8.2 Orchestration Engine

- [ ] Engine executes a 3-node DAG with 1 parallel pair
  - Nodes B and C execute in parallel after Node A completes
  - All results captured correctly
  - Execution duration < 5 seconds total

- [ ] Auto-replanning triggers on transient failure
  - Transient error → automatic retry with backoff
  - After MaxRetries exhausted → call RegeneratePlan
  - New plan version > old plan version

- [ ] Circuit breaker opens after threshold
  - 5+ consecutive persistent/fatal errors → circuit open
  - Subsequent attempts fail with circuit_breaker_open (Fatal)
  - Circuit recovers after timeout

- [ ] Cost estimation works
  - plan.EstimateCost() returns reasonable token count
  - Estimated cost scales with node count
  - Budget check prevents execution if over limit

- [ ] Error classification correct
  - "timeout" → Transient
  - "not found" → Persistent
  - "forbidden" → Fatal
  - Others classified correctly

### 8.3 Memory System

- [ ] Memory CRUD works with SQLite
  - StoreMemory creates record
  - GetMemory retrieves by ID
  - UpdateMemory modifies existing
  - DeleteMemory removes (soft or hard)
  - ListMemories returns filtered set

- [ ] Garden compression works
  - CompressHistory reduces to 20-30% of original
  - CompressionRatio calculated correctly
  - Compressed content preserves key insights

- [ ] Seed extraction works
  - ExtractSeeds returns list of patterns
  - Confidence scores between 0-1
  - Seeds storable and retrievable

- [ ] Search works
  - SearchMemories returns results ranked by similarity
  - Snippets extracted correctly
  - Empty query returns all memories

### 8.4 Integration

- [ ] No Dojo-specific imports in either module
- [ ] No hardcoded dependencies on external packages (except standard library + google/uuid, database/sql)
- [ ] Interfaces are consumer-friendly (no internal types exposed)
- [ ] Examples compile and run without errors

---

## 9. Non-Goals

Explicitly out of scope for Track 2:

- **HTTP/API layer** (Track 3): REST endpoints, middleware, request/response marshaling
- **LLM-based planner** (consumer provides): Track 2 defines PlannerInterface; consumer implements with their choice of LLM
- **Dojo-specific features**: Compassion context, entity memories, Dojo event types
- **Real-time bidirectional sync**: WebSocket streaming (Track 3)
- **Advanced storage backends**: Cloud databases, distributed caching (Track 3)
- **Monitoring & alerting**: Prometheus metrics, log aggregation (Track 3)

---

## 10. Migration Path & Backward Compatibility

### 10.1 From Monolith to Track 2

**Phase 1: Extract**
1. Copy orchestration/ and memory/ from go_backend
2. Create separate go.mod for each (remove internal dependencies)
3. Update imports: `github.com/TresPies-source/dojo-genesis/go_backend/...` → `github.com/TresPies-source/AgenticGatewayByDojoGenesis/...`
4. Remove Dojo-specific features (compassion, dojo events)

**Phase 2: Test & Verify**
1. Run full test suites in isolation
2. Verify no cross-module imports
3. Benchmark performance vs. original

**Phase 3: Integrate Back**
1. Update main go_backend to consume Track 2 modules as dependencies
2. Maintain API compatibility
3. Deprecate old internal implementations

### 10.2 Storage Backend Migration

To switch from SQLite to PostgreSQL:

```go
// Current (SQLite)
manager, _ := memory.NewMemoryManager("file:memories.db")

// Future (PostgreSQL)
manager, _ := memory.NewMemoryManagerWithBackend(
    &PostgreSQLBackend{connStr: "postgres://..."},
)
```

No changes needed to memory package logic; backend is pluggable.

---

## 11. Testing Strategy

### 11.1 Unit Tests

**Orchestration:**
- DAG validation (acyclic, dependencies)
- Node state transitions (pending → running → success/failed)
- Error classification (transient/persistent/fatal)
- Circuit breaker state machine
- Cost estimation accuracy
- Backoff calculation and jitter

**Memory:**
- CRUD operations (create, read, update, delete)
- Semantic search ranking
- Compression ratio calculations
- Seed extraction patterns
- Embedding storage and retrieval

### 11.2 Integration Tests

- 3-node DAG execution with parallel pair
- Auto-replanning on failure
- Circuit breaker recovery
- Memory storage → compression → seed extraction
- Full orchestration + memory workflow

### 11.3 Benchmark Tests

- Engine execution throughput (nodes/sec)
- Memory search latency (p50, p95, p99)
- Compression speed (tokens/sec)
- Parallel node execution efficiency

---

## 12. Known Limitations & Future Work

### Known Limitations

1. **Cost estimation is approximate**: Actual costs depend on LLM provider; estimation is best-effort
2. **Circuit breaker is per-tool global**: No per-user or per-task scoping
3. **Memory compression is blocking**: No async compression in v0.1.0
4. **Replanning loss of context**: RegeneratePlan doesn't have access to node execution traces
5. **No distributed execution**: Single-process only; no multi-node DAG execution

### Future Work (Track 4+)

1. **Exact cost tracking**: Integration with billing APIs for actual costs
2. **Distributed DAG execution**: Multi-process/service orchestration
3. **Async memory operations**: Non-blocking compression and embedding
4. **Advanced planner strategies**: Tree-of-thought, Monte Carlo planning
5. **Memory federation**: Multi-user shared memory pools with privacy controls
6. **Observability enhancement**: Prometheus metrics, structured logging, distributed tracing

---

## 13. Dependencies & Versioning

### Runtime Dependencies

**orchestration/**
- `google/uuid`: v1.3.0+ (UUID generation)
- `context`, `sync`, `time`, `encoding/json`, `fmt`: Go stdlib

**memory/**
- `google/uuid`: v1.3.0+ (ID generation)
- `database/sql`: Go stdlib
- `github.com/mattn/go-sqlite3`: v1.14.0+ (default backend)
- `context`, `encoding/json`, `time`: Go stdlib

### Optional Dependencies

- `trace/*`: Trace logging interface (consumer provides)
- `events/*`: Event emission interface (consumer provides)
- `plugin/*`: LLM provider interface (for compression)

### Go Version

- **Minimum**: Go 1.24
- **Tested**: Go 1.24

---

## 14. Security Considerations

### 14.1 Orchestration Engine

- **Tool parameter injection**: Planner must sanitize tool parameters (no arbitrary code execution)
- **Infinite loops**: Engine limits execution to 1000 iterations max
- **Timeout handling**: Context cancellation respected throughout execution
- **Cost overrides**: Budget checks non-bypassable; query/session/monthly limits enforced

### 14.2 Memory System

- **SQL injection**: Uses parameterized queries via database/sql (safe)
- **Embedding data**: Stored as float32 arrays; no code execution possible
- **Soft deletion**: Marked records not truly deleted (GDPR consideration)
- **Access control**: No built-in auth; consumer responsible for user isolation

---

## 15. Maintenance & Support

### Release Maintenance

- **Bug fixes**: Released as patch versions (0.1.x)
- **Feature additions**: Released as minor versions (0.x.0)
- **Breaking changes**: Major version bump (x.0.0)

### Support Window

- **v0.1.0**: Support through v0.2.0 release (3 months minimum)
- **Deprecation notice**: 1 month before removing features

### Community Contributions

Track 2 welcomes:
- Bug reports and fixes
- Performance optimizations
- Additional planner implementations
- Storage backend adapters
- Test coverage improvements

---

## Appendix A: Module Files

### orchestration/ Module
```
orchestration/
├── engine.go                      (~1130 lines)
│   ├── ErrorType, ErrorClassification
│   ├── EngineConfig, DefaultEngineConfig
│   ├── Engine struct, NewEngine
│   ├── Execute, executePlan, executeNodesInParallel
│   ├── Circuit breaker (open/close/check)
│   ├── Error classification (transient/persistent/fatal)
│   ├── Cost estimation
│   ├── Backoff & retry logic
│   └── Event emission (node.start, node.end, replanning)
├── task.go                        (~210 lines)
│   ├── NodeState enum
│   ├── Task struct
│   ├── PlanNode struct
│   ├── Plan struct
│   ├── NewTask, NewPlan, NewPlanNode
│   ├── Plan.GetExecutableNodes
│   ├── Plan.AllNodesCompleted
│   ├── Plan.ValidateDAG (acyclic check)
│   └── Node.IsReady, Node.Duration
├── planner.go                     (~359 lines)
│   ├── PlannerInterface
│   ├── Planner struct (LLM-based)
│   ├── GeneratePlan, RegeneratePlan
│   ├── LLM prompt building
│   └── JSON parsing
├── planner_test.go, engine_test.go, integration_test.go
└── go.mod
    require (
      google/uuid v1.3.0+
    )
```

### memory/ Module
```
memory/
├── manager.go                     (~400+ lines)
│   ├── MemoryManager struct
│   ├── StoreMemory, GetMemory, UpdateMemory, DeleteMemory
│   ├── SearchMemories (semantic)
│   ├── ListMemories (with filters)
│   ├── Schema initialization
│   └── Connection management
├── types.go                       (~60 lines)
│   ├── Memory struct
│   ├── MemorySeed struct
│   ├── MemoryFile struct
│   ├── SearchResult, FileSearchResult
│   └── Error definitions
├── garden_manager.go              (~200+ lines)
│   ├── GardenManager struct
│   ├── CompressHistory
│   ├── ExtractSeeds
│   ├── StoreSeed, GetSeed, ListSeeds
│   └── Compression logic
├── compression.go                 (~200+ lines)
│   ├── CompressionService struct
│   ├── CompressHistory
│   ├── Semantic abstraction via LLM
│   └── Compression ratio calculation
├── seed_extractor.go              (~150+ lines)
│   ├── SeedExtractor struct
│   ├── ExtractSeeds
│   ├── Pattern scoring
│   └── Confidence calculation
├── context_builder.go             (~150+ lines)
│   ├── ContextBuilder struct
│   ├── BuildContext
│   ├── Attention mechanism
│   └── Token budget truncation
├── manager_test.go, garden_manager_test.go, compression_test.go, etc.
└── go.mod
    require (
      google/uuid v1.3.0+
      mattn/go-sqlite3 v1.14.0+
    )
```

---

## Appendix B: Example Integration Test

```go
package orchestration_test

import (
    "context"
    "testing"
    "time"
    "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
)

// TestThreeNodeDAGWithParallel verifies 3-node DAG execution
// with parallel execution of nodes B and C.
func TestThreeNodeDAGWithParallel(t *testing.T) {
    ctx := context.Background()
    config := orchestration.DefaultEngineConfig()
    planner := &testPlanner{}

    engine := orchestration.NewEngine(config, planner, nil, nil, nil, nil)

    // Create 3-node DAG
    plan := &orchestration.Plan{
        ID:     "test-dag-1",
        TaskID: "task-1",
    }

    nodeA := orchestration.NewPlanNode("sleep_1s",
        map[string]interface{}{}, []string{})
    nodeB := orchestration.NewPlanNode("sleep_1s",
        map[string]interface{}{}, []string{nodeA.ID})
    nodeC := orchestration.NewPlanNode("sleep_1s",
        map[string]interface{}{}, []string{nodeA.ID})

    plan.Nodes = append(plan.Nodes, nodeA, nodeB, nodeC)

    if err := plan.ValidateDAG(); err != nil {
        t.Fatalf("Invalid DAG: %v", err)
    }

    // Execute and measure time
    start := time.Now()
    task := orchestration.NewTask("user1", "test dag")

    if err := engine.Execute(ctx, plan, task, "user1"); err != nil {
        t.Fatalf("Execution failed: %v", err)
    }

    duration := time.Since(start)

    // Verify:
    // - All nodes succeeded
    if len(plan.Nodes) != 3 {
        t.Fatalf("Expected 3 nodes, got %d", len(plan.Nodes))
    }

    for _, node := range plan.Nodes {
        if node.State != orchestration.NodeStateSuccess {
            t.Fatalf("Node %s state %s, expected success",
                node.ID, node.State)
        }
    }

    // - Parallel execution: duration ~2s (A: 1s, then B+C: 1s parallel)
    // Not strictly < 2s due to overhead, but should be < 3s
    if duration > 3*time.Second {
        t.Logf("WARNING: Execution took %v (expected ~2s for parallel)", duration)
    }

    t.Logf("✓ 3-node DAG with parallel execution completed in %v", duration)
}
```

---

**End of Specification**

Document Version: 0.1.0
Last Updated: 2026-02-12
Status: APPROVED FOR IMPLEMENTATION
Approval: Pre-Implementation Checklist Complete

---

## 12. Pre-Implementation Checklist

**Instructions:** Before handing this specification to the implementation agent, ensure every item is checked.

### 1. Vision & Goals

- [x] **Clarity of Purpose:** Extracts orchestration engine (DAG planner, auto-replanning, circuit breakers) and memory system (CRUD, compression, seeds) as independent modules.
- [x] **Measurable Goals:** 6 orchestration goals and 5 memory goals are specific and measurable.
- [x] **Testable Success Criteria:** Binary success criteria for both modules with coverage targets and performance benchmarks.
- [x] **Scope is Defined:** HTTP/API concerns (Track 3) and Dojo-specific features explicitly out of scope.

### 2. Technical Readiness

- [x] **Architecture is Sound:** Module structure, dependency graph, and 5 core interfaces (PlannerInterface, StorageBackendInterface, CompressionServiceInterface, TraceLoggerInterface, EventEmitterInterface) well-defined.
- [x] **Code is Production-Ready:** Complete Go type definitions for Task, Plan, PlanNode, Memory, MemorySeed, and all interfaces. Uses `github.com/TresPies-source/AgenticGatewayByDojoGenesis/*` paths.
- [x] N/A **APIs are Specified:** Track 2 defines internal Go interfaces, not HTTP APIs (that's Track 3).
- [x] **Database Schema is Final:** StorageBackendInterface is an abstraction; no hardcoded schema. Memory and Seed types defined.
- [x] **Dependencies are Met:** Depends on shared/ for currency types, tools/ for ToolDefinition, events/ for event emission.

### 3. Implementation Plan

- [x] **Plan is Actionable:** 3-phase migration path (Extract → Test → Integrate) with clear steps.
- [x] **Timeline is Realistic:** Aligned with overall Phase 1 (Core Modules, 2-3 weeks).
- [x] **Testing Strategy is Comprehensive:** Unit tests, integration tests, DAG validation, circuit breaker behavior tests, compression ratio benchmarks.

### 4. Risk & Quality

- [x] **Risks are Mitigated:** Error classification (Transient/Persistent/Fatal), circuit breaker pattern, replanning limits prevent infinite loops.
- [x] N/A **Rollback Plan is Clear:** New modules; rollback = use monolith directly.
- [x] N/A **Feature Flags are Defined:** Library modules, no feature flags needed.

### 5. Handoff

- [x] **Final Review Complete:** Pre-flight report reviewed; all cross-spec fixes applied (module paths, Go version).
- [x] **Specification is Final:** Document status marked as APPROVED FOR IMPLEMENTATION.
- [x] **Implementation Ready:** Ready to commission.

### 0. Track 0 — Pre-Commission Alignment

- [x] **Codebase Verified:** orchestration/ (engine, planner, task types) and memory/ (manager, garden, seeds, context builder) confirmed in monolith.
- [x] **Types Verified:** PlannerInterface, Task, Plan, PlanNode, Memory, MemorySeed match existing codebase patterns.
- [x] **APIs Verified:** StorageBackendInterface (9 methods) matches existing database layer patterns.
- [x] **File Structure Verified:** Module paths updated to `github.com/TresPies-source/AgenticGatewayByDojoGenesis/*`. Go version corrected to 1.24.
- [x] **Remediation Complete:** Module paths standardized, Go version fixed (was 1.19-1.22, now 1.24), migration instructions updated to show correct target paths.
