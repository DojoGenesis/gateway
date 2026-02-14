package orchestration

import (
	"context"
	"fmt"
	"time"

	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/google/uuid"
)

// GatewayOrchestrationExecutor adapts the existing orchestration.Engine
// to implement the gateway.OrchestrationExecutor interface.
// This allows the engine to be used through the gateway's standard interface.
type GatewayOrchestrationExecutor struct {
	engine  *orchestrationpkg.Engine
	planner orchestrationpkg.PlannerInterface
}

// NewGatewayOrchestrationExecutor creates a new gateway-compatible orchestration executor.
func NewGatewayOrchestrationExecutor(engine *orchestrationpkg.Engine, planner orchestrationpkg.PlannerInterface) *GatewayOrchestrationExecutor {
	return &GatewayOrchestrationExecutor{
		engine:  engine,
		planner: planner,
	}
}

// Execute runs an orchestration plan represented as a DAG of tool invocations.
// It converts the gateway.ExecutionPlan to the internal Plan format and executes it.
func (e *GatewayOrchestrationExecutor) Execute(ctx context.Context, plan *gateway.ExecutionPlan) (*gateway.ExecutionResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("execution plan cannot be nil")
	}

	startTime := time.Now()

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
	err := e.engine.Execute(ctx, orchPlan, task, "system")

	duration := time.Since(startTime).Milliseconds()

	// Build result
	result := &gateway.ExecutionResult{
		ExecutionID: plan.ID,
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
// Note: The current orchestration engine doesn't support cancellation by ID,
// so this is a placeholder implementation.
func (e *GatewayOrchestrationExecutor) Cancel(ctx context.Context, executionID string) error {
	// TODO: Implement execution tracking and cancellation
	// For now, return not implemented
	return fmt.Errorf("execution cancellation not yet implemented (execution ID: %s)", executionID)
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
