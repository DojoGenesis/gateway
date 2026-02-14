package skill

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SkillExecutor handles skill invocation with DAG subtask binding.
type SkillExecutor interface {
	// Execute runs a skill with the given arguments.
	// If the skill invokes other skills, they become DAG subtask nodes.
	Execute(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error)

	// ExecuteAsSubtask invokes a skill as a child of the current skill.
	// Creates a DAG node for the invocation and tracks it in the parent plan.
	// Includes depth limit enforcement (max=3) and budget tracking.
	ExecuteAsSubtask(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error)
}

// DefaultSkillExecutor is the default implementation of SkillExecutor
type DefaultSkillExecutor struct {
	registry    SkillRegistry
	toolInvoker ToolInvoker // Local interface to avoid circular dependency
	traceLogger TraceLogger // Local interface to avoid circular dependency
	callDepth   int         // Track call depth for meta-skill safety (max = 3)
}

// NewSkillExecutor creates a new skill executor
func NewSkillExecutor(
	registry SkillRegistry,
	toolInvoker ToolInvoker,
	traceLogger TraceLogger,
) *DefaultSkillExecutor {
	return &DefaultSkillExecutor{
		registry:    registry,
		toolInvoker: toolInvoker,
		traceLogger: traceLogger,
		callDepth:   0,
	}
}

// Execute runs a skill with the given arguments
func (e *DefaultSkillExecutor) Execute(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error) {
	// 1. Look up skill in registry
	skill, err := e.registry.GetSkill(ctx, skillName)
	if err != nil {
		return nil, fmt.Errorf("skill execution failed for '%s': %w", skillName, err)
	}

	// 2. Validate tool dependencies are satisfied
	if err := e.validateDependencies(ctx, skill); err != nil {
		return nil, fmt.Errorf("skill '%s' has unmet dependencies: %w", skillName, err)
	}

	// 3. Start tracing span if tracer available
	var span SpanHandle
	if e.traceLogger != nil {
		span, err = e.traceLogger.StartSpan(ctx, fmt.Sprintf("skill-%s", skillName), fmt.Sprintf("execute-skill:%s", skillName), map[string]interface{}{
			"skill_name": skillName,
			"tier":       skill.Tier,
			"plugin":     skill.PluginName,
		})
		if err != nil {
			// Don't fail execution if tracing fails
			span = nil
		}
	}

	// 4. Prepare skill invocation parameters
	// The skill content is passed as context, and args are passed through
	params := map[string]interface{}{
		"skill_name":    skillName,
		"skill_content": skill.Content,
		"args":          args,
		"metadata": map[string]interface{}{
			"tier":        skill.Tier,
			"plugin":      skill.PluginName,
			"triggers":    skill.Triggers,
			"portable":    skill.Portable,
			"description": skill.Description,
		},
	}

	// 5. Invoke the skill through the tool invoker
	// The actual skill execution logic is delegated to the orchestration layer
	// via the "invoke_skill" tool. The ToolInvoker handles the execution.
	result, execErr := e.toolInvoker.InvokeTool(ctx, "invoke_skill", params)

	// 6. Complete tracing span
	if span != nil {
		if execErr != nil {
			_ = e.traceLogger.FailSpan(ctx, span, execErr.Error())
		} else {
			_ = e.traceLogger.EndSpan(ctx, span, map[string]interface{}{
				"result": result,
			})
		}
	}

	// 7. Handle execution error
	if execErr != nil {
		return nil, fmt.Errorf("failed to invoke skill '%s': %w", skillName, execErr)
	}

	return result, nil
}

// validateDependencies checks if required tool dependencies are available.
// Returns an error if any required dependency is missing.
func (e *DefaultSkillExecutor) validateDependencies(ctx context.Context, skill *SkillDefinition) error {
	// Check for web_tools dependency
	if skill.RequiresWebTools() {
		if _, err := e.toolInvoker.InvokeTool(ctx, "web_search", nil); err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not registered") {
				return fmt.Errorf("web_tools adapter required but not loaded")
			}
		}
	}

	// Check for script_execution dependency
	if skill.RequiresScriptExecution() {
		if _, err := e.toolInvoker.InvokeTool(ctx, "execute_script", nil); err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not registered") {
				return fmt.Errorf("script executor required but not available")
			}
		}
	}

	return nil
}

// SetCallDepth sets the current call depth (used for nested skill invocations).
// Max depth = 3 is enforced in ExecuteAsSubtask via CheckDepthLimit.
func (e *DefaultSkillExecutor) SetCallDepth(depth int) {
	e.callDepth = depth
}

// GetCallDepth returns the current call depth
func (e *DefaultSkillExecutor) GetCallDepth() int {
	return e.callDepth
}

// ExecuteAsSubtask invokes a skill as a child of the current skill.
// Meta-skill invocation with:
// - Call depth tracking (max depth = 3)
// - Budget propagation and tracking
// - OTEL span linking
// - DAG node creation (logged via trace, not directly managed here)
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

	// 3. Get budget tracker from context (if available)
	budgetTracker := GetBudgetTracker(ctx)

	// 4. Estimate token budget for this skill (simplified: 1000 tokens per skill)
	// In production, this would be based on skill metadata or historical usage
	estimatedTokens := 1000

	// 5. Reserve budget if tracker is available
	if budgetTracker != nil {
		if err := budgetTracker.Reserve(estimatedTokens); err != nil {
			return nil, fmt.Errorf("cannot invoke meta-skill '%s': %w", skillName, err)
		}
	}

	// 6. Increment call depth in context
	childCtx := WithIncrementedDepth(ctx)
	currentDepth := GetCallDepth(ctx)

	// 7. Create OTEL child span (if trace logger available)
	var span SpanHandle
	if e.traceLogger != nil {
		traceID := fmt.Sprintf("meta-skill-%s-%d", skillName, time.Now().UnixNano())
		spanStartErr := error(nil)
		span, spanStartErr = e.traceLogger.StartSpan(childCtx, traceID, fmt.Sprintf("meta-skill.invoke.%s", skillName), map[string]interface{}{
			"skill.name":        skillName,
			"skill.tier":        skill.Tier,
			"skill.call_depth":  currentDepth + 1,
			"skill.parent_type": "meta_skill_invocation",
		})
		if spanStartErr != nil {
			// Don't fail execution if tracing fails
			span = nil
		}
	}

	// 8. Execute the skill through the standard execution path
	result, execErr := e.Execute(childCtx, skillName, args)

	// 9. Update budget tracker with actual consumption
	// For now, use a simple heuristic: result size in chars / 4 (approx tokens)
	actualTokens := 0
	if budgetTracker != nil {
		if execErr == nil && result != nil {
			// Success: consume based on actual result size
			resultSize := len(fmt.Sprintf("%v", result))
			actualTokens = resultSize / 4
			if actualTokens == 0 {
				actualTokens = 100 // Minimum token count
			}
			if actualTokens > estimatedTokens {
				actualTokens = estimatedTokens // Cap at reservation
			}
			budgetTracker.Consume(estimatedTokens, actualTokens)
		} else {
			// Error: release the reservation without consuming
			budgetTracker.Release(estimatedTokens)
		}
	} else if execErr == nil && result != nil {
		// No budget tracker, but still calculate tokens for tracing
		resultSize := len(fmt.Sprintf("%v", result))
		actualTokens = resultSize / 4
		if actualTokens == 0 {
			actualTokens = 100
		}
	}

	// 10. Complete OTEL span
	if span != nil {
		if execErr != nil {
			_ = e.traceLogger.FailSpan(childCtx, span, execErr.Error())
		} else {
			_ = e.traceLogger.EndSpan(childCtx, span, map[string]interface{}{
				"skill.result.tokens_used": actualTokens,
				"skill.result.status":      "success",
			})
		}
	}

	// 11. Return result or error
	if execErr != nil {
		return nil, fmt.Errorf("meta-skill '%s' execution failed at depth %d: %w", skillName, currentDepth+1, execErr)
	}

	return result, nil
}
