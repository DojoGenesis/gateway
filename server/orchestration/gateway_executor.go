package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"

	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/google/uuid"
)

// executionState tracks the lifecycle of a single execution.
type executionState struct {
	ctx       context.Context
	cancel    context.CancelFunc
	completed bool
}

// GatewayOrchestrationExecutor adapts the existing orchestration.Engine
// to implement the gateway.OrchestrationExecutor interface.
// This allows the engine to be used through the gateway's standard interface.
type GatewayOrchestrationExecutor struct {
	engine  *orchestrationpkg.Engine
	planner orchestrationpkg.PlannerInterface

	mu         sync.Mutex
	executions map[string]*executionState
}

// NewGatewayOrchestrationExecutor creates a new gateway-compatible orchestration executor.
func NewGatewayOrchestrationExecutor(engine *orchestrationpkg.Engine, planner orchestrationpkg.PlannerInterface) *GatewayOrchestrationExecutor {
	return &GatewayOrchestrationExecutor{
		engine:     engine,
		planner:    planner,
		executions: make(map[string]*executionState),
	}
}

// Execute runs an orchestration plan represented as a DAG of tool invocations.
// It converts the gateway.ExecutionPlan to the internal Plan format and executes it.
func (e *GatewayOrchestrationExecutor) Execute(ctx context.Context, plan *gateway.ExecutionPlan) (*gateway.ExecutionResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("execution plan cannot be nil")
	}

	startTime := time.Now()

	// Wrap context with cancellation for execution tracking
	execCtx, cancel := context.WithCancel(ctx)

	executionID := plan.ID
	if executionID == "" {
		executionID = uuid.New().String()
	}

	// Store the execution state for tracking and cancellation
	e.mu.Lock()
	e.executions[executionID] = &executionState{
		ctx:    execCtx,
		cancel: cancel,
	}
	e.mu.Unlock()

	// Convert gateway.ExecutionPlan to orchestration.Plan
	orchPlan := convertGatewayPlanToOrchestrationPlan(plan)

	// Create task for execution
	task := &orchestrationpkg.Task{
		ID:          plan.ID,
		UserID:      "system",
		Description: plan.Name,
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Execute the plan
	// Note: Using "system" as userID since gateway interface doesn't provide it
	err := e.engine.Execute(execCtx, orchPlan, task, "system")

	duration := time.Since(startTime).Milliseconds()

	// Mark execution as completed
	e.mu.Lock()
	if state, ok := e.executions[executionID]; ok {
		state.completed = true
	}
	e.mu.Unlock()

	// Build result
	result := &gateway.ExecutionResult{
		ExecutionID: executionID,
		Duration:    duration,
		Output:      make(map[string]interface{}),
	}

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, nil // Return result with error info, not error itself
	}

	// Collect outputs from completed nodes
	for _, node := range orchPlan.Nodes {
		if node.State == orchestrationpkg.NodeStateSuccess && node.Result != nil {
			result.Output[node.ID] = node.Result
		}
	}

	result.Status = "success"
	return result, nil
}

// Cancel terminates a running execution by its unique execution ID.
func (e *GatewayOrchestrationExecutor) Cancel(ctx context.Context, executionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, exists := e.executions[executionID]
	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	state.cancel()
	return nil
}

// Status returns the current state of an execution: "running", "completed", "cancelled", or "not_found".
func (e *GatewayOrchestrationExecutor) Status(executionID string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, exists := e.executions[executionID]
	if !exists {
		return "not_found", fmt.Errorf("execution not found: %s", executionID)
	}

	if state.completed {
		return "completed", nil
	}

	// Context cancelled but execution not marked complete means it was cancelled externally
	if state.ctx.Err() != nil {
		return "cancelled", nil
	}

	return "running", nil
}

// convertGatewayPlanToOrchestrationPlan converts a gateway.ExecutionPlan
// to the internal orchestration.Plan format.
func convertGatewayPlanToOrchestrationPlan(gatewayPlan *gateway.ExecutionPlan) *orchestrationpkg.Plan {
	nodes := make([]*orchestrationpkg.PlanNode, 0, len(gatewayPlan.DAG))

	for _, invocation := range gatewayPlan.DAG {
		node := &orchestrationpkg.PlanNode{
			ID:           invocation.ID,
			ToolName:     invocation.ToolName,
			Parameters:   invocation.Input,
			Dependencies: invocation.DependsOn,
			State:        orchestrationpkg.NodeStatePending,
		}
		nodes = append(nodes, node)
	}

	planID := gatewayPlan.ID
	if planID == "" {
		planID = uuid.New().String()
	}

	return &orchestrationpkg.Plan{
		ID:        planID,
		TaskID:    gatewayPlan.Name, // Store gateway plan name in TaskID field for now
		Nodes:     nodes,
		CreatedAt: time.Now(),
		Version:   1,
		Metadata:  map[string]interface{}{"gateway_plan_name": gatewayPlan.Name},
	}
}
