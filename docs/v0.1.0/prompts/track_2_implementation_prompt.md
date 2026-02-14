# Implementation Commission: Track 2 — Orchestration + Memory Modules

**Objective:** Implement the orchestration engine (DAG-based task execution with auto-replanning, circuit breakers, and cost estimation) and memory system (conversation CRUD, garden compression, seed extraction) as independent Go modules.

**Depends On:** Track 0 (Foundation) must be complete. Track 1 (tools/ module) must be complete — orchestration depends on `tools.ToolDefinition`.

---

## 1. Context & Grounding

**Primary Specification:**
- `docs/agentic-gateway-v0.1.0/track_2_orchestration_memory_spec.md`

**Foundation Specification:**
- `docs/agentic-gateway-v0.1.0/track_0_foundation_spec.md` (for shared types, dependency graph)

**Pattern Files (Follow these examples from monolith):**
- `go_backend/orchestration/engine.go`: DAG execution engine lifecycle
- `go_backend/orchestration/planner.go`: PlannerInterface and plan generation
- `go_backend/memory/manager.go`: MemoryManager CRUD operations
- `go_backend/memory/garden_manager.go`: Garden compression and seed extraction

**Module Paths:**
- `github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration`
- `github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory`

---

## 2. Detailed Requirements

### Orchestration Module

1. **Define `PlannerInterface`** in `orchestration/planner.go`:
   - `GeneratePlan(ctx, *Task) (*Plan, error)`
   - `RegeneratePlan(ctx, *Task, *Plan, errorContext) (*Plan, error)`
   - This is a consumer-provided interface — ship with a `NoOpPlanner` stub for testing

2. **Define core types** in `orchestration/task.go`:
   - `Task` struct: ID, UserID, Description, CreatedAt, Metadata
   - `Plan` struct: ID, TaskID, Nodes, Version, CreatedAt, Metadata
   - `PlanNode` struct: ID, ToolName, Parameters, Dependencies, State, Result, Error, StartTime, EndTime, RetryCount
   - `NodeState`: pending, running, success, failed, skipped
   - Helper methods: `plan.GetExecutableNodes()`, `plan.AllNodesCompleted()`, `plan.HasFailedNodes()`, `plan.ValidateDAG()`

3. **Implement `Engine`** in `orchestration/engine.go`:
   - `NewEngine(config, planner, toolRegistry, opts...)` — constructor with functional options
   - `Execute(ctx, *Task) (*ExecutionResult, error)` — main entry point
   - DAG execution: identify parallelizable nodes, execute with goroutines, respect dependencies
   - Auto-replanning: on persistent failure, call `planner.RegeneratePlan()`, limit attempts (default 3)
   - Cost estimation: estimate token costs before execution, enforce budget limits
   - Event emission via `EventEmitterInterface` (optional)
   - Trace logging via `TraceLoggerInterface` (optional)

4. **Implement error classification** in `orchestration/errors.go`:
   - `ClassifyError(err) ErrorClass` — Transient, Persistent, Fatal
   - Transient: timeout, rate limit, temporary unavailable → retry with backoff
   - Persistent: invalid parameters, not found, validation error → replan
   - Fatal: budget exceeded, forbidden, authentication failure → abort

5. **Implement circuit breaker** in `orchestration/circuit_breaker.go`:
   - States: Closed (normal) → Open (after threshold failures) → Half-Open (after timeout)
   - Configurable: threshold (default 5), timeout (default 60s), half-open max attempts (default 1)
   - Per-tool circuit breakers (not global)
   - Thread-safe with mutex

6. **Implement retry logic** in `orchestration/retry.go`:
   - Exponential backoff with optional jitter
   - Configurable max retries per node (default 3)
   - Context-aware cancellation

7. **Implement cost estimation** in `orchestration/cost.go`:
   - `EstimatePlanCost(plan) *CostEstimate`
   - Tool category multipliers: Web 1.4x, File 1.3x, Memory 1.2x, Compute 1.5x, Generic 1.35x
   - Budget enforcement: query-level, session-level, monthly
   - Cost hooks for external tracking

### Memory Module

8. **Define `MemoryManager`** in `memory/manager.go`:
   - `NewMemoryManager(storage StorageBackendInterface, opts...)` — constructor
   - `StoreMemory(ctx, *Memory) error`
   - `GetMemory(ctx, id) (*Memory, error)`
   - `UpdateMemory(ctx, *Memory) error`
   - `DeleteMemory(ctx, id) error`
   - `ListMemories(ctx, filter) ([]*Memory, error)`
   - `SearchMemories(ctx, query, limit) ([]*Memory, error)` — semantic search

9. **Define `StorageBackendInterface`** in `memory/types.go`:
   - 9 methods for Memory and Seed CRUD (see spec section 3.3)
   - Ship with `InMemoryStorage` implementation for testing

10. **Define core types** in `memory/types.go`:
    - `Memory` struct: ID, Type, Content, ContextType, Metadata, Embedding, CreatedAt, UpdatedAt
    - `MemorySeed` struct: ID, ProjectID, Content, SeedType, Source, UserEditable, Confidence, UsageCount, LastUsedAt, CreatedAt, UpdatedAt, CreatedBy
    - `MemoryFilter`, `SeedFilter` for queries
    - `CompressedHistory` for compression results

11. **Implement `GardenManager`** in `memory/garden_manager.go`:
    - `NewGardenManager(storage, compressionService)` — constructor
    - `CompressSession(ctx, sessionID) (*CompressedHistory, error)` — 3-Month Rule compression
    - Uses `CompressionServiceInterface` (consumer-provided, LLM-based)
    - Target: 70-80% token reduction

12. **Implement `SeedExtractor`** in `memory/seed_extractor.go`:
    - `ExtractSeeds(ctx, memories) ([]*MemorySeed, error)`
    - Uses `CompressionServiceInterface`
    - Confidence scoring for extracted patterns
    - Track usage patterns and timestamps

13. **Implement `ContextBuilder`** in `memory/context_builder.go`:
    - `BuildContext(ctx, sessionID, tokenBudget) ([]Message, error)`
    - Assemble context from memory layers
    - Token budget-aware truncation
    - Attention mechanisms for relevant content selection

14. **Define `CompressionServiceInterface`** in `memory/types.go`:
    - `CompressHistory(ctx, sessionID, memories) (*CompressedHistory, error)`
    - `ExtractSeeds(ctx, memories) ([]*MemorySeed, error)`
    - Ship with `NoOpCompressionService` stub for testing

---

## 3. File Manifest

**Create:**
- `orchestration/engine.go` — Engine with DAG execution
- `orchestration/task.go` — Task, Plan, PlanNode types
- `orchestration/planner.go` — PlannerInterface + NoOpPlanner
- `orchestration/errors.go` — Error classification
- `orchestration/circuit_breaker.go` — Circuit breaker pattern
- `orchestration/retry.go` — Retry with backoff
- `orchestration/cost.go` — Cost estimation and budget enforcement
- `orchestration/config.go` — Configuration types
- `orchestration/*_test.go` — Unit tests for each file
- `memory/manager.go` — MemoryManager
- `memory/types.go` — All type definitions and interfaces
- `memory/garden_manager.go` — GardenManager
- `memory/seed_extractor.go` — SeedExtractor
- `memory/context_builder.go` — ContextBuilder
- `memory/compression.go` — CompressionServiceInterface + NoOp stub
- `memory/storage_inmemory.go` — InMemoryStorage implementation
- `memory/*_test.go` — Unit tests for each file

---

## 4. Success Criteria

- [ ] `cd orchestration && go build` succeeds
- [ ] `cd memory && go build` succeeds
- [ ] `go test ./orchestration/...` passes with >80% coverage
- [ ] `go test ./memory/...` passes with >80% coverage
- [ ] `go test -race ./orchestration/...` detects no data races
- [ ] `go test -race ./memory/...` detects no data races
- [ ] Engine executes a 3-node DAG with 1 parallel pair correctly
- [ ] Engine auto-replans after persistent failure (verified by test)
- [ ] Circuit breaker transitions: Closed → Open → Half-Open (verified by test)
- [ ] Error classification correctly categorizes timeout (Transient), invalid param (Persistent), budget exceeded (Fatal)
- [ ] MemoryManager CRUD operations work with InMemoryStorage backend
- [ ] GardenManager calls CompressionService and stores results
- [ ] ContextBuilder respects token budget limits
- [ ] No imports from provider/ or server/ in orchestration module
- [ ] No imports from provider/, tools/, orchestration/, or server/ in memory module

---

## 5. Constraints & Non-Goals

- **DO NOT** implement HTTP handlers — that's Track 3
- **DO NOT** implement real LLM-based planner or compression — use stub interfaces
- **DO NOT** implement SQLite or PostgreSQL storage — use InMemoryStorage only
- **DO NOT** add provider routing or model selection logic — that's Track 1
- **DO NOT** implement SSE streaming — that's Track 3
- **DO NOT** change shared/ or events/ types — use them as defined

---

## 6. Source Codebase Reference

**Source Files to Extract From:**
- `go_backend/orchestration/engine.go` → `orchestration/engine.go`
- `go_backend/orchestration/planner.go` → `orchestration/planner.go`
- `go_backend/memory/manager.go` → `memory/manager.go`
- `go_backend/memory/garden_manager.go` → `memory/garden_manager.go`
- `go_backend/memory/seed_extractor.go` → `memory/seed_extractor.go`
- `go_backend/memory/context_builder.go` → `memory/context_builder.go`

**Dependencies:**
- `orchestration` → `shared`, `tools` (for ToolDefinition), `events` (for event emission)
- `memory` → `shared` only
- External: `github.com/google/uuid`
