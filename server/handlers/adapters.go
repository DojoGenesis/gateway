package handlers

import (
	"context"
	"fmt"

	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/DojoGenesis/gateway/server/agent"
	"github.com/DojoGenesis/gateway/specialist"
)

// SpecialistRouterAdapter wraps *specialist.Router to satisfy the ChatHandler's
// SpecialistRouter interface. This avoids importing the specialist package
// directly in the chat handler.
type SpecialistRouterAdapter struct {
	Router *specialist.Router
}

// Route delegates to the specialist router and maps the result to the
// chat handler's SpecialistRoutingResult type.
func (a *SpecialistRouterAdapter) Route(decision agent.RoutingDecision) SpecialistRoutingResult {
	result := a.Router.Route(decision)
	return SpecialistRoutingResult{
		Routed:       result.Routed,
		SpecialistID: result.SpecialistID,
		Reason:       result.Reason,
	}
}

// OrchestratorAdapter wraps the orchestration planner and engine to satisfy
// the ChatHandler's Orchestrator interface.
type OrchestratorAdapter struct {
	Planner orchestrationpkg.PlannerInterface
}

// GeneratePlanForChat creates a Task from the user's query, generates a plan,
// and returns a human-readable summary of the plan steps.
func (a *OrchestratorAdapter) GeneratePlanForChat(ctx context.Context, userID, query string) (string, error) {
	if a.Planner == nil {
		return "", fmt.Errorf("planner not configured")
	}

	task := orchestrationpkg.NewTask(userID, query)

	plan, err := a.Planner.GeneratePlan(ctx, task)
	if err != nil {
		return "", fmt.Errorf("plan generation failed: %w", err)
	}

	// Build a concise plan summary for the LLM context
	summary := fmt.Sprintf("Plan ID: %s\nSteps (%d total):\n", plan.ID, len(plan.Nodes))
	for i, node := range plan.Nodes {
		deps := ""
		if len(node.Dependencies) > 0 {
			deps = fmt.Sprintf(" (depends on: %v)", node.Dependencies)
		}
		summary += fmt.Sprintf("  %d. [%s] %s%s\n", i+1, node.ID, node.ToolName, deps)
	}

	return summary, nil
}
