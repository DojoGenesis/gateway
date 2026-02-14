package orchestration

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock implementations ---

type mockToolInvoker struct {
	invokeFunc func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)
	callCount  atomic.Int32
}

func (m *mockToolInvoker) InvokeTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	m.callCount.Add(1)
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, toolName, params)
	}
	return map[string]interface{}{"success": true}, nil
}

type mockPlanner struct {
	generateFunc    func(ctx context.Context, task *Task) (*Plan, error)
	regenerateFunc  func(ctx context.Context, task *Task, failedPlan *Plan, errorContext string) (*Plan, error)
	regenerateCount atomic.Int32
}

func (m *mockPlanner) GeneratePlan(ctx context.Context, task *Task) (*Plan, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, task)
	}
	return NewPlan(task.ID), nil
}

func (m *mockPlanner) RegeneratePlan(ctx context.Context, task *Task, failedPlan *Plan, errorContext string) (*Plan, error) {
	m.regenerateCount.Add(1)
	if m.regenerateFunc != nil {
		return m.regenerateFunc(ctx, task, failedPlan, errorContext)
	}
	plan := NewPlan(task.ID)
	plan.Version = failedPlan.Version + 1
	return plan, nil
}

type mockBudgetTracker struct {
	remaining int
	err       error
}

func (m *mockBudgetTracker) GetRemaining(userID string) (int, error) {
	return m.remaining, m.err
}

type mockEventEmitter struct {
	events []StreamEvent
}

func (m *mockEventEmitter) Emit(event StreamEvent) {
	m.events = append(m.events, event)
}

// --- Tests ---

func TestEngine_Execute_ThreeNodeDAGWithParallelPair(t *testing.T) {
	ctx := context.Background()

	var executionOrder []string
	var orderChan = make(chan string, 10)

	invoker := &mockToolInvoker{
		invokeFunc: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			orderChan <- toolName
			time.Sleep(50 * time.Millisecond)
			return map[string]interface{}{"tool": toolName, "result": "ok"}, nil
		},
	}

	config := DefaultEngineConfig()
	planner := &mockPlanner{}

	// Use rapid pacing for this test (no delays) to test parallel execution timing
	rapidDisp := &disposition.DispositionConfig{
		Pacing: "rapid",
	}

	engine := NewEngine(config, planner, invoker, nil, nil, nil, WithDisposition(rapidDisp))

	// Create 3-node DAG: A -> (B, C) parallel
	plan := &Plan{
		ID:      "test-dag-1",
		TaskID:  "task-1",
		Version: 1,
		Nodes:   make([]*PlanNode, 0),
	}

	nodeA := &PlanNode{
		ID:           "node-a",
		ToolName:     "fetch",
		Parameters:   map[string]interface{}{},
		Dependencies: []string{},
		State:        NodeStatePending,
	}

	nodeB := &PlanNode{
		ID:           "node-b",
		ToolName:     "parse",
		Parameters:   map[string]interface{}{},
		Dependencies: []string{"node-a"},
		State:        NodeStatePending,
	}

	nodeC := &PlanNode{
		ID:           "node-c",
		ToolName:     "validate",
		Parameters:   map[string]interface{}{},
		Dependencies: []string{"node-a"},
		State:        NodeStatePending,
	}

	plan.Nodes = []*PlanNode{nodeA, nodeB, nodeC}
	require.NoError(t, plan.ValidateDAG())

	task := NewTask("user1", "test dag execution")

	start := time.Now()
	err := engine.Execute(ctx, plan, task, "user1")
	duration := time.Since(start)

	require.NoError(t, err)

	// Drain order channel
	close(orderChan)
	for tool := range orderChan {
		executionOrder = append(executionOrder, tool)
	}

	// All 3 nodes should have executed
	assert.Len(t, executionOrder, 3)
	// First execution must be "fetch" (node A)
	assert.Equal(t, "fetch", executionOrder[0])

	// All nodes should be successful
	for _, node := range plan.Nodes {
		assert.Equal(t, NodeStateSuccess, node.State, "node %s should be success", node.ID)
		assert.NotNil(t, node.Result)
	}

	// Parallel execution: should be ~100ms (A:50ms + B||C:50ms), not ~150ms
	assert.Less(t, duration, 500*time.Millisecond)

	t.Logf("3-node DAG with parallel pair completed in %v", duration)
}

func TestEngine_Execute_AutoReplanning(t *testing.T) {
	ctx := context.Background()

	callCount := atomic.Int32{}

	invoker := &mockToolInvoker{
		invokeFunc: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			count := callCount.Add(1)
			if toolName == "failing_tool" && count <= 4 { // Fail initial retries
				return nil, fmt.Errorf("not found: resource missing")
			}
			return map[string]interface{}{"success": true}, nil
		},
	}

	planner := &mockPlanner{
		regenerateFunc: func(ctx context.Context, task *Task, failedPlan *Plan, errorContext string) (*Plan, error) {
			newPlan := NewPlan(task.ID)
			newPlan.Version = failedPlan.Version + 1
			// Replace failing tool with alternative
			newPlan.Nodes = append(newPlan.Nodes, &PlanNode{
				ID:           "node-alt",
				ToolName:     "alternative_tool",
				Parameters:   map[string]interface{}{},
				Dependencies: []string{},
				State:        NodeStatePending,
			})
			return newPlan, nil
		},
	}

	config := DefaultEngineConfig()
	config.MaxRetries = 0 // Don't retry, go straight to replan
	config.EnableAutoReplanning = true
	config.MaxReplanningAttempts = 2
	config.RetryBackoff = 1 * time.Millisecond

	engine := NewEngine(config, planner, invoker, nil, nil, nil)

	plan := NewPlan("task-1")
	plan.Nodes = append(plan.Nodes, &PlanNode{
		ID:           "node-1",
		ToolName:     "failing_tool",
		Parameters:   map[string]interface{}{},
		Dependencies: []string{},
		State:        NodeStatePending,
	})

	task := NewTask("user1", "test replanning")

	err := engine.Execute(ctx, plan, task, "user1")
	require.NoError(t, err)

	// Planner should have been called for regeneration
	assert.GreaterOrEqual(t, int(planner.regenerateCount.Load()), 1)
}

func TestEngine_ErrorClassification(t *testing.T) {
	engine := NewEngine(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		errMsg   string
		expected ErrorType
	}{
		{"timeout exceeded", ErrorTypeTransient},
		{"deadline exceeded", ErrorTypeTransient},
		{"service unavailable", ErrorTypeTransient},
		{"rate limit exceeded", ErrorTypeTransient},
		{"connection refused", ErrorTypeTransient},
		{"invalid parameter: missing field", ErrorTypePersistent},
		{"not found: resource", ErrorTypePersistent},
		{"validation failed", ErrorTypePersistent},
		{"budget exceeded", ErrorTypeFatal},
		{"forbidden access", ErrorTypeFatal},
		{"unauthorized request", ErrorTypeFatal},
		{"circuit_breaker_open", ErrorTypeFatal},
		{"some unknown error", ErrorTypePersistent}, // default
	}

	for _, tt := range tests {
		t.Run(tt.errMsg, func(t *testing.T) {
			result := engine.ClassifyError(fmt.Errorf("%s", tt.errMsg))
			assert.Equal(t, tt.expected, result)
		})
	}

	// nil error should classify as transient
	assert.Equal(t, ErrorTypeTransient, engine.ClassifyError(nil))
}

func TestEngine_CircuitBreaker(t *testing.T) {
	ctx := context.Background()

	invoker := &mockToolInvoker{
		invokeFunc: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("persistent error: invalid parameter")
		},
	}

	config := DefaultEngineConfig()
	config.CircuitBreakerThreshold = 3
	config.CircuitBreakerTimeout = 1 * time.Second
	config.MaxRetries = 0 // No retries, each attempt counts
	config.EnableAutoReplanning = false
	config.RetryBackoff = 1 * time.Millisecond

	planner := &mockPlanner{}
	engine := NewEngine(config, planner, invoker, nil, nil, nil)

	// Execute multiple plans to trigger circuit breaker
	for i := 0; i < 4; i++ {
		plan := NewPlan(fmt.Sprintf("task-%d", i))
		plan.Nodes = append(plan.Nodes, &PlanNode{
			ID:           fmt.Sprintf("node-%d", i),
			ToolName:     "broken_tool",
			Parameters:   map[string]interface{}{},
			Dependencies: []string{},
			State:        NodeStatePending,
		})
		task := NewTask("user1", "test")
		_ = engine.Execute(ctx, plan, task, "user1")
	}

	// Circuit breaker should now be open
	isOpen, err := engine.checkCircuitBreaker("broken_tool")
	assert.True(t, isOpen)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit_breaker_open")

	// After timeout, circuit breaker should close (half-open)
	time.Sleep(1100 * time.Millisecond)
	isOpen, _ = engine.checkCircuitBreaker("broken_tool")
	assert.False(t, isOpen)
}

func TestEngine_CostEstimation(t *testing.T) {
	engine := NewEngine(nil, nil, nil, nil, nil, nil)

	plan := &Plan{
		ID:     "plan-1",
		TaskID: "task-1",
		Nodes: []*PlanNode{
			{ID: "n1", ToolName: "web_search", Parameters: map[string]interface{}{"query": "test"}},
			{ID: "n2", ToolName: "read_file", Parameters: map[string]interface{}{"path": "/tmp/f.txt"}, Dependencies: []string{"n1"}},
			{ID: "n3", ToolName: "calculate", Parameters: map[string]interface{}{"expression": "1+1"}, Dependencies: []string{"n1"}},
		},
	}

	cost, err := engine.EstimatePlanCost(plan)
	require.NoError(t, err)
	assert.Greater(t, cost, 0)

	// Cost should scale with number of nodes
	smallPlan := &Plan{
		ID:     "plan-2",
		TaskID: "task-2",
		Nodes: []*PlanNode{
			{ID: "n1", ToolName: "web_search", Parameters: map[string]interface{}{"query": "test"}},
		},
	}

	smallCost, err := engine.EstimatePlanCost(smallPlan)
	require.NoError(t, err)
	assert.Greater(t, cost, smallCost)

	// Empty plan should return 0
	emptyCost, err := engine.EstimatePlanCost(&Plan{})
	require.NoError(t, err)
	assert.Equal(t, 0, emptyCost)

	// Nil plan should return 0
	nilCost, err := engine.EstimatePlanCost(nil)
	require.NoError(t, err)
	assert.Equal(t, 0, nilCost)
}

func TestEngine_BudgetEnforcement(t *testing.T) {
	ctx := context.Background()

	invoker := &mockToolInvoker{}
	planner := &mockPlanner{}

	budget := &mockBudgetTracker{remaining: 10} // Very low budget
	config := DefaultEngineConfig()

	engine := NewEngine(config, planner, invoker, nil, nil, budget)

	plan := &Plan{
		ID:     "plan-1",
		TaskID: "task-1",
		Nodes: []*PlanNode{
			{ID: "n1", ToolName: "web_search", Parameters: map[string]interface{}{"query": "test"}},
			{ID: "n2", ToolName: "web_search", Parameters: map[string]interface{}{"query": "test2"}},
			{ID: "n3", ToolName: "web_search", Parameters: map[string]interface{}{"query": "test3"}},
		},
	}

	task := NewTask("user1", "test budget")
	err := engine.Execute(ctx, plan, task, "user1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "budget")
}

func TestEngine_BudgetEnforcement_NilTracker(t *testing.T) {
	ctx := context.Background()

	invoker := &mockToolInvoker{}
	planner := &mockPlanner{}
	config := DefaultEngineConfig()

	engine := NewEngine(config, planner, invoker, nil, nil, nil) // No budget tracker

	plan := &Plan{
		ID:     "plan-1",
		TaskID: "task-1",
		Nodes: []*PlanNode{
			{ID: "n1", ToolName: "read_file", Parameters: map[string]interface{}{}, Dependencies: []string{}, State: NodeStatePending},
		},
	}

	task := NewTask("user1", "test")
	err := engine.Execute(ctx, plan, task, "user1")
	assert.NoError(t, err) // Should succeed without budget tracker
}

func TestEngine_EventEmission(t *testing.T) {
	ctx := context.Background()

	invoker := &mockToolInvoker{}
	planner := &mockPlanner{}
	emitter := &mockEventEmitter{}
	config := DefaultEngineConfig()

	engine := NewEngine(config, planner, invoker, nil, emitter, nil)

	plan := &Plan{
		ID:     "plan-1",
		TaskID: "task-1",
		Nodes: []*PlanNode{
			{ID: "n1", ToolName: "test_tool", Parameters: map[string]interface{}{}, Dependencies: []string{}, State: NodeStatePending},
		},
	}

	task := NewTask("user1", "test events")
	err := engine.Execute(ctx, plan, task, "user1")
	require.NoError(t, err)

	// Should have emitted start and end events
	assert.GreaterOrEqual(t, len(emitter.events), 2)

	hasStart := false
	hasEnd := false
	for _, e := range emitter.events {
		if e.Type == "orchestration.node.start" {
			hasStart = true
		}
		if e.Type == "orchestration.node.end" {
			hasEnd = true
		}
	}
	assert.True(t, hasStart, "should have node start event")
	assert.True(t, hasEnd, "should have node end event")
}

func TestEngine_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	invoker := &mockToolInvoker{
		invokeFunc: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			cancel()
			time.Sleep(100 * time.Millisecond) // Give time for cancellation
			return map[string]interface{}{}, nil
		},
	}

	config := DefaultEngineConfig()
	planner := &mockPlanner{}
	engine := NewEngine(config, planner, invoker, nil, nil, nil)

	plan := &Plan{
		ID:     "plan-1",
		TaskID: "task-1",
		Nodes: []*PlanNode{
			{ID: "n1", ToolName: "tool1", Parameters: map[string]interface{}{}, Dependencies: []string{}, State: NodeStatePending},
			{ID: "n2", ToolName: "tool2", Parameters: map[string]interface{}{}, Dependencies: []string{"n1"}, State: NodeStatePending},
		},
	}

	task := NewTask("user1", "test cancel")
	err := engine.Execute(ctx, plan, task, "user1")
	// Should either succeed (if n1 completed before cancel was noticed) or fail with context error
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	}
}

func TestEngine_ToolHealthMetrics(t *testing.T) {
	engine := NewEngine(nil, nil, nil, nil, nil, nil)

	engine.recordToolAttempt("test_tool")
	engine.recordToolAttempt("test_tool")
	engine.recordToolSuccess("test_tool")
	engine.recordToolFailure("test_tool", ErrorTypeTransient)

	health := engine.GetToolHealthMetrics("test_tool")
	require.NotNil(t, health)
	assert.Equal(t, 2, health.TotalAttempts)
	assert.Equal(t, 1, health.SuccessfulCalls)
	assert.Equal(t, 1, health.FailedCalls)
	assert.Equal(t, ErrorTypeTransient, health.LastErrorType)

	// Non-existent tool
	assert.Nil(t, engine.GetToolHealthMetrics("nonexistent"))
}

func TestEngine_DefaultConfig(t *testing.T) {
	config := DefaultEngineConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryBackoff)
	assert.Equal(t, 30*time.Second, config.MaxBackoff)
	assert.Equal(t, 5, config.MaxParallelNodes)
	assert.True(t, config.EnableAutoReplanning)
	assert.Equal(t, 2, config.MaxReplanningAttempts)
	assert.True(t, config.EnableJitter)
	assert.True(t, config.EnableCircuitBreaker)
	assert.Equal(t, 5, config.CircuitBreakerThreshold)
	assert.Equal(t, 60*time.Second, config.CircuitBreakerTimeout)
}

func TestEngine_Execute_FatalError_NoReplan(t *testing.T) {
	ctx := context.Background()

	invoker := &mockToolInvoker{
		invokeFunc: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("forbidden: access denied")
		},
	}

	planner := &mockPlanner{}
	config := DefaultEngineConfig()
	config.MaxRetries = 0
	config.RetryBackoff = 1 * time.Millisecond

	engine := NewEngine(config, planner, invoker, nil, nil, nil)

	plan := &Plan{
		ID:     "plan-1",
		TaskID: "task-1",
		Nodes: []*PlanNode{
			{ID: "n1", ToolName: "tool", Parameters: map[string]interface{}{}, Dependencies: []string{}, State: NodeStatePending},
		},
	}

	task := NewTask("user1", "test fatal")
	err := engine.Execute(ctx, plan, task, "user1")
	assert.Error(t, err)
	// Should NOT have attempted replanning for fatal errors
	assert.Equal(t, int32(0), planner.regenerateCount.Load())
}
