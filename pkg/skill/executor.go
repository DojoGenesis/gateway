package skill

import (
	"context"
	"fmt"
)

// SkillExecutor handles skill invocation with DAG subtask binding.
type SkillExecutor interface {
	// Execute runs a skill with the given arguments.
	// If the skill invokes other skills, they become DAG subtask nodes.
	Execute(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error)

	// NOTE: ExecuteAsSubtask and RegisterMetaSkillInvocation are Phase 4b (Tier 3).
	// Phase 4a skips them (will be implemented later).
}

// DefaultSkillExecutor is the default implementation of SkillExecutor
type DefaultSkillExecutor struct {
	registry    SkillRegistry
	toolInvoker ToolInvoker  // Local interface to avoid circular dependency
	traceLogger TraceLogger  // Local interface to avoid circular dependency
	callDepth   int          // Track call depth for meta-skill safety (max = 3, enforced in Phase 4b)
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
	// In Phase 4a, just log warnings for missing dependencies
	// Phase 4b will add stricter validation
	if err := e.validateDependencies(ctx, skill); err != nil {
		// Log warning but don't block execution
		// TODO(Phase 4b): Make this a hard error
		_ = err // Suppress unused variable warning
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
	// Note: In Phase 4a, the actual skill execution logic is delegated to
	// the orchestration layer via the "invoke_skill" tool.
	// The ToolInvoker will handle the actual execution.
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

// validateDependencies checks if required tool dependencies are available
// Phase 4a: warnings only
// Phase 4b: strict validation with errors
func (e *DefaultSkillExecutor) validateDependencies(ctx context.Context, skill *SkillDefinition) error {
	// Check for web_tools dependency
	if skill.RequiresWebTools() {
		// TODO(Phase 4a): Check if web_tools adapter is loaded
		// For now, this is a no-op warning
	}

	// Check for script_execution dependency
	if skill.RequiresScriptExecution() {
		// TODO(Phase 4a): Check if script executor is available
		// For now, this is a no-op warning
	}

	// Phase 4a: always return nil (warnings only)
	return nil
}

// SetCallDepth sets the current call depth (used for nested skill invocations)
// Phase 4b will use this to enforce max depth = 3
func (e *DefaultSkillExecutor) SetCallDepth(depth int) {
	e.callDepth = depth
}

// GetCallDepth returns the current call depth
func (e *DefaultSkillExecutor) GetCallDepth() int {
	return e.callDepth
}
