package orchestration

import (
	"context"
	"fmt"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/skill"
)

// SkillInvoker wraps the base ToolInvoker to add skill execution support.
// When toolName == "invoke_skill", it delegates to the SkillExecutor.
// For all other tools, it delegates to the base invoker.
type SkillInvoker struct {
	skillRegistry skill.SkillRegistry
	skillExecutor skill.SkillExecutor
	baseInvoker   ToolInvokerInterface // Delegate to default invoker for other tools
}

// NewSkillInvoker creates a new skill-aware tool invoker
func NewSkillInvoker(
	registry skill.SkillRegistry,
	executor skill.SkillExecutor,
	baseInvoker ToolInvokerInterface,
) *SkillInvoker {
	return &SkillInvoker{
		skillRegistry: registry,
		skillExecutor: executor,
		baseInvoker:   baseInvoker,
	}
}

// InvokeTool handles both skill and non-skill tools
func (s *SkillInvoker) InvokeTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	// If it's a skill invocation, handle specially
	if toolName == "invoke_skill" {
		return s.invokeSkill(ctx, params)
	}

	// For non-skill tools, delegate to base invoker
	if s.baseInvoker == nil {
		return nil, fmt.Errorf("no base invoker configured for tool: %s", toolName)
	}

	return s.baseInvoker.InvokeTool(ctx, toolName, params)
}

// invokeSkill handles skill invocation
func (s *SkillInvoker) invokeSkill(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	// Extract skill_name from parameters
	skillName, ok := params["skill_name"].(string)
	if !ok {
		return nil, fmt.Errorf("invoke_skill requires 'skill_name' string parameter")
	}

	// Extract args (optional, defaults to empty map)
	args, ok := params["args"].(map[string]interface{})
	if !ok {
		// If args not provided or wrong type, use empty map
		args = make(map[string]interface{})
	}

	// Execute the skill through the SkillExecutor
	result, err := s.skillExecutor.Execute(ctx, skillName, args)
	if err != nil {
		return nil, fmt.Errorf("skill invocation failed: %w", err)
	}

	return result, nil
}

// GetSkillRegistry returns the skill registry (useful for tests and introspection)
func (s *SkillInvoker) GetSkillRegistry() skill.SkillRegistry {
	return s.skillRegistry
}

// GetSkillExecutor returns the skill executor (useful for tests and introspection)
func (s *SkillInvoker) GetSkillExecutor() skill.SkillExecutor {
	return s.skillExecutor
}
