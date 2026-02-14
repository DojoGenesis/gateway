# Plan Registry Design Proposal

**Purpose:** Enable RegisterMetaSkillInvocation by making Plans accessible to SkillExecutor

---

## Current Architecture (Why It Doesn't Work)

```
┌─────────┐
│ Engine  │ Creates Plan, passes to executePlan()
└────┬────┘
     │ (Plan passed as parameter)
     ▼
┌─────────────┐
│ executeNode │ Gets Plan from parameters
└──────┬──────┘
       │
       ▼
┌──────────────┐
│ ToolInvoker  │ No Plan reference
└──────┬───────┘
       │
       ▼
┌─────────────────┐
│ SkillExecutor   │ ❌ Can't access Plan
└─────────────────┘
```

**Problem:** Plan lives on Engine's call stack, never stored globally

---

## Proposed Architecture

### 1. Create Plan Registry

```go
// orchestration/plan_registry.go

package orchestration

import (
    "context"
    "fmt"
    "sync"
)

// PlanRegistry stores and retrieves Plans by ID.
// Thread-safe for concurrent access.
type PlanRegistry interface {
    // StorePlan adds a plan to the registry
    StorePlan(plan *Plan) error

    // GetPlan retrieves a plan by ID
    GetPlan(planID string) (*Plan, error)

    // UpdatePlan atomically updates a plan
    UpdatePlan(planID string, updateFn func(*Plan) error) error

    // RemovePlan removes a plan from registry
    RemovePlan(planID string)

    // AppendNode atomically appends a node to a plan
    AppendNode(planID string, node *PlanNode) error
}

type InMemoryPlanRegistry struct {
    plans map[string]*Plan
    mu    sync.RWMutex
}

func NewInMemoryPlanRegistry() *InMemoryPlanRegistry {
    return &InMemoryPlanRegistry{
        plans: make(map[string]*Plan),
    }
}

func (r *InMemoryPlanRegistry) StorePlan(plan *Plan) error {
    if plan == nil || plan.ID == "" {
        return fmt.Errorf("invalid plan")
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    r.plans[plan.ID] = plan
    return nil
}

func (r *InMemoryPlanRegistry) GetPlan(planID string) (*Plan, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    plan, exists := r.plans[planID]
    if !exists {
        return nil, fmt.Errorf("plan not found: %s", planID)
    }
    return plan, nil
}

func (r *InMemoryPlanRegistry) AppendNode(planID string, node *PlanNode) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    plan, exists := r.plans[planID]
    if !exists {
        return fmt.Errorf("plan not found: %s", planID)
    }

    plan.Nodes = append(plan.Nodes, node)
    return nil
}

func (r *InMemoryPlanRegistry) UpdatePlan(planID string, updateFn func(*Plan) error) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    plan, exists := r.plans[planID]
    if !exists {
        return fmt.Errorf("plan not found: %s", planID)
    }

    return updateFn(plan)
}

func (r *InMemoryPlanRegistry) RemovePlan(planID string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    delete(r.plans, planID)
}
```

### 2. Propagate Plan ID via Context

```go
// orchestration/context.go

package orchestration

import "context"

type contextKey string

const (
    planIDKey    contextKey = "orchestration_plan_id"
    planNodeIDKey contextKey = "orchestration_node_id"
)

// WithPlanID adds the plan ID to context
func WithPlanID(ctx context.Context, planID string) context.Context {
    return context.WithValue(ctx, planIDKey, planID)
}

// GetPlanID retrieves the plan ID from context
func GetPlanID(ctx context.Context) (string, bool) {
    planID, ok := ctx.Value(planIDKey).(string)
    return planID, ok
}

// WithNodeID adds the current node ID to context
func WithNodeID(ctx context.Context, nodeID string) context.Context {
    return context.WithValue(ctx, planNodeIDKey, nodeID)
}

// GetNodeID retrieves the current node ID from context
func GetNodeID(ctx context.Context) (string, bool) {
    nodeID, ok := ctx.Value(planNodeIDKey).(string)
    return nodeID, ok
}
```

### 3. Modify Engine to Use Registry

```go
// orchestration/engine.go

type Engine struct {
    // ... existing fields ...
    planRegistry PlanRegistry  // ← ADD THIS
}

func NewEngine(
    config *EngineConfig,
    planner PlannerInterface,
    toolInvoker ToolInvokerInterface,
    traceLogger TraceLoggerInterface,
    eventEmitter EventEmitterInterface,
    budgetTracker BudgetTrackerInterface,
    planRegistry PlanRegistry,  // ← ADD THIS
    opts ...EngineOption,
) *Engine {
    // ...
    e := &Engine{
        // ...
        planRegistry: planRegistry,  // ← ADD THIS
    }
    return e
}

func (e *Engine) executePlan(ctx context.Context, plan *Plan, task *Task) error {
    // Store plan in registry
    if e.planRegistry != nil {
        if err := e.planRegistry.StorePlan(plan); err != nil {
            return fmt.Errorf("failed to store plan: %w", err)
        }
        defer e.planRegistry.RemovePlan(plan.ID)
    }

    // Add plan ID to context
    ctx = WithPlanID(ctx, plan.ID)

    // ... rest of execution ...
}

func (e *Engine) executeNode(ctx context.Context, node *PlanNode, plan *Plan, traceID string) error {
    // Add node ID to context
    ctx = WithNodeID(ctx, node.ID)

    // ... rest of node execution ...
}
```

### 4. Inject Registry into SkillExecutor

```go
// skill/executor.go

type DefaultSkillExecutor struct {
    registry     SkillRegistry
    toolInvoker  ToolInvoker
    traceLogger  TraceLogger
    callDepth    int
    planRegistry PlanRegistryInterface  // ← ADD THIS
}

// PlanRegistryInterface is a local interface to avoid circular dependency
type PlanRegistryInterface interface {
    AppendNode(planID string, node interface{}) error
}

func NewSkillExecutor(
    registry SkillRegistry,
    toolInvoker ToolInvoker,
    traceLogger TraceLogger,
    planRegistry PlanRegistryInterface,  // ← ADD THIS
) *DefaultSkillExecutor {
    return &DefaultSkillExecutor{
        registry:     registry,
        toolInvoker:  toolInvoker,
        traceLogger:  traceLogger,
        callDepth:    0,
        planRegistry: planRegistry,  // ← ADD THIS
    }
}
```

### 5. Implement RegisterMetaSkillInvocation

```go
// skill/executor.go

func (e *DefaultSkillExecutor) RegisterMetaSkillInvocation(
    ctx context.Context,
    skillName string,
    args map[string]interface{},
) (nodeID string, error) {
    // Get plan ID from context
    planID, ok := GetPlanIDFromContext(ctx)
    if !ok {
        // No plan in context - skill not invoked from orchestration
        // This is okay - just means we're in a different execution mode
        return "", nil
    }

    // Get parent node ID from context (if any)
    parentNodeID, _ := GetNodeIDFromContext(ctx)

    // Create a new PlanNode for this meta-skill invocation
    node := &PlanNode{
        ID:           uuid.New().String(),
        ToolName:     "invoke_skill",
        Parameters: map[string]interface{}{
            "skill_name": skillName,
            "args":       args,
            "meta_skill": true,
        },
        Dependencies: []string{parentNodeID},  // Depends on parent
        State:        NodeStatePending,
    }

    // Atomically append to plan
    if e.planRegistry != nil {
        if err := e.planRegistry.AppendNode(planID, node); err != nil {
            return "", fmt.Errorf("failed to register meta-skill invocation: %w", err)
        }
    }

    return node.ID, nil
}

// Helper to get plan ID from context (avoids import cycle)
func GetPlanIDFromContext(ctx context.Context) (string, bool) {
    // This would need to be imported from orchestration package
    // or duplicated here to avoid circular dependency
    planID, ok := ctx.Value("orchestration_plan_id").(string)
    return planID, ok
}
```

### 6. Update ExecuteAsSubtask to Use It

```go
// skill/executor.go

func (e *DefaultSkillExecutor) ExecuteAsSubtask(
    ctx context.Context,
    skillName string,
    args map[string]interface{},
) (map[string]interface{}, error) {
    // 1. Check call depth limit
    if err := CheckDepthLimit(ctx, MaxMetaSkillDepth); err != nil {
        return nil, err
    }

    // 2. Look up skill in registry
    skill, err := e.registry.GetSkill(ctx, skillName)
    if err != nil {
        return nil, fmt.Errorf("meta-skill invocation failed for '%s': %w", skillName, err)
    }

    // 3. Register DAG node (if in orchestration context)
    nodeID, err := e.RegisterMetaSkillInvocation(ctx, skillName, args)
    if err != nil {
        return nil, err
    }

    // Add node ID to context for child invocations
    if nodeID != "" {
        ctx = WithNodeIDInContext(ctx, nodeID)
    }

    // 4. Get budget tracker from context (if available)
    // ... rest of existing implementation ...
}
```

---

## Migration Path

### Phase 1: Add Registry Infrastructure
1. Implement `PlanRegistry` interface
2. Add context helpers for plan/node IDs
3. **No breaking changes** - registry is optional

### Phase 2: Integrate with Engine
1. Add `planRegistry` parameter to `NewEngine`
2. Store/retrieve plans in `executePlan`
3. Propagate IDs via context
4. **Breaking change:** Engine constructor signature

### Phase 3: Enable in SkillExecutor
1. Add `planRegistry` parameter to `NewSkillExecutor`
2. Implement `RegisterMetaSkillInvocation`
3. Call it from `ExecuteAsSubtask`
4. **Breaking change:** SkillExecutor constructor signature

### Phase 4: Update Wiring
1. Update main.go to create PlanRegistry
2. Pass to Engine and SkillExecutor
3. Update all tests

---

## Benefits

✅ **True DAG Tracking** - Meta-skill invocations appear as nodes
✅ **Unified Observability** - Same view for all tool/skill invocations
✅ **Debugging** - Can inspect full DAG structure
✅ **Replanning** - Can analyze failed meta-skill chains

---

## Drawbacks

⚠️ **Complexity** - Adds global state (registry)
⚠️ **Concurrency** - Need locks for Plan mutation
⚠️ **Memory** - Plans stay in memory during execution
⚠️ **Breaking Changes** - Constructor signatures change

---

## Recommendation

**Implement in v0.4.0** after Phase 4b is stable.

For v0.3.0 (Phase 4b), OTEL tracing provides sufficient observability without architectural changes.

---

## Alternative: Hybrid Approach

Keep OTEL tracing but also register nodes when registry is available:

```go
func (e *DefaultSkillExecutor) ExecuteAsSubtask(...) {
    // Always create OTEL span (existing code)
    span, _ := e.traceLogger.StartSpan(...)

    // Optionally register DAG node (new code)
    if e.planRegistry != nil {
        nodeID, _ := e.RegisterMetaSkillInvocation(ctx, skillName, args)
        // If it fails, continue anyway - OTEL provides fallback
    }

    // Continue with execution...
}
```

This gives best of both worlds:
- ✅ Works without registry (backward compatible)
- ✅ Enhanced when registry available
- ✅ OTEL always provides observability

**Co-Authored-By:** Claude Sonnet 4.5 <noreply@anthropic.com>
