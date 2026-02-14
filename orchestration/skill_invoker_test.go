package orchestration

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/skill"
)

// Mock implementations for testing

type mockSkillExecutor struct {
	executeFn func(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error)
}

func (m *mockSkillExecutor) Execute(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, skillName, args)
	}
	return map[string]interface{}{"status": "success"}, nil
}

func (m *mockSkillExecutor) ExecuteAsSubtask(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error) {
	return m.Execute(ctx, skillName, args)
}

type mockBaseInvoker struct {
	invokeFn func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)
}

func (m *mockBaseInvoker) InvokeTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, toolName, params)
	}
	return map[string]interface{}{"result": "base invoker result"}, nil
}

// Tests

func TestNewSkillInvoker(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	executor := &mockSkillExecutor{}
	baseInvoker := &mockBaseInvoker{}

	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	assert.NotNil(t, invoker)
	assert.Equal(t, registry, invoker.GetSkillRegistry())
	assert.Equal(t, executor, invoker.GetSkillExecutor())
}

func TestInvokeTool_InvokeSkill_HappyPath(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	ctx := context.Background()

	// Create a mock executor that returns success
	executor := &mockSkillExecutor{
		executeFn: func(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error) {
			assert.Equal(t, "test-skill", skillName)
			assert.Equal(t, "test value", args["key1"])

			return map[string]interface{}{
				"status": "success",
				"output": "Skill executed",
			}, nil
		},
	}

	baseInvoker := &mockBaseInvoker{}
	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	// Invoke a skill
	params := map[string]interface{}{
		"skill_name": "test-skill",
		"args": map[string]interface{}{
			"key1": "test value",
		},
	}

	result, err := invoker.InvokeTool(ctx, "invoke_skill", params)

	assert.NoError(t, err)
	assert.Equal(t, "success", result["status"])
	assert.Equal(t, "Skill executed", result["output"])
}

func TestInvokeTool_InvokeSkill_MissingSkillName(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	executor := &mockSkillExecutor{}
	baseInvoker := &mockBaseInvoker{}
	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	ctx := context.Background()

	// Missing skill_name parameter
	params := map[string]interface{}{
		"args": map[string]interface{}{},
	}

	_, err := invoker.InvokeTool(ctx, "invoke_skill", params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill_name")
}

func TestInvokeTool_InvokeSkill_InvalidSkillName(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	executor := &mockSkillExecutor{}
	baseInvoker := &mockBaseInvoker{}
	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	ctx := context.Background()

	// skill_name is not a string
	params := map[string]interface{}{
		"skill_name": 123, // Should be string
		"args":       map[string]interface{}{},
	}

	_, err := invoker.InvokeTool(ctx, "invoke_skill", params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill_name")
}

func TestInvokeTool_InvokeSkill_EmptyArgs(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	ctx := context.Background()

	// Executor should receive empty args map
	executor := &mockSkillExecutor{
		executeFn: func(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error) {
			assert.NotNil(t, args)
			assert.Len(t, args, 0)
			return map[string]interface{}{"status": "ok"}, nil
		},
	}

	baseInvoker := &mockBaseInvoker{}
	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	// No args parameter
	params := map[string]interface{}{
		"skill_name": "test-skill",
	}

	result, err := invoker.InvokeTool(ctx, "invoke_skill", params)

	assert.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestInvokeTool_InvokeSkill_ExecutionError(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	ctx := context.Background()

	// Executor returns an error
	executor := &mockSkillExecutor{
		executeFn: func(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error) {
			return nil, errors.New("skill execution failed")
		},
	}

	baseInvoker := &mockBaseInvoker{}
	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	params := map[string]interface{}{
		"skill_name": "failing-skill",
		"args":       map[string]interface{}{},
	}

	_, err := invoker.InvokeTool(ctx, "invoke_skill", params)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill invocation failed")
	assert.Contains(t, err.Error(), "skill execution failed")
}

func TestInvokeTool_NonSkillTool_DelegatesToBase(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	executor := &mockSkillExecutor{}
	ctx := context.Background()

	// Base invoker should be called for non-skill tools
	baseCalled := false
	baseInvoker := &mockBaseInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			baseCalled = true
			assert.Equal(t, "some_other_tool", toolName)
			assert.Equal(t, "param value", params["param1"])
			return map[string]interface{}{"result": "from base"}, nil
		},
	}

	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	params := map[string]interface{}{
		"param1": "param value",
	}

	result, err := invoker.InvokeTool(ctx, "some_other_tool", params)

	assert.NoError(t, err)
	assert.True(t, baseCalled)
	assert.Equal(t, "from base", result["result"])
}

func TestInvokeTool_NonSkillTool_NoBaseInvoker(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	executor := &mockSkillExecutor{}
	invoker := NewSkillInvoker(registry, executor, nil) // No base invoker

	ctx := context.Background()

	_, err := invoker.InvokeTool(ctx, "some_other_tool", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no base invoker configured")
}

func TestInvokeTool_SkillInvokerAsWrapper(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	ctx := context.Background()

	// Test that SkillInvoker correctly wraps base invoker
	skillCalled := false
	baseCalled := false

	executor := &mockSkillExecutor{
		executeFn: func(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error) {
			skillCalled = true
			return map[string]interface{}{"from": "skill"}, nil
		},
	}

	baseInvoker := &mockBaseInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			baseCalled = true
			return map[string]interface{}{"from": "base"}, nil
		},
	}

	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	// Invoke a skill
	skillParams := map[string]interface{}{
		"skill_name": "test-skill",
		"args":       map[string]interface{}{},
	}
	skillResult, err := invoker.InvokeTool(ctx, "invoke_skill", skillParams)
	require.NoError(t, err)
	assert.Equal(t, "skill", skillResult["from"])
	assert.True(t, skillCalled)
	assert.False(t, baseCalled)

	// Reset flags
	skillCalled = false
	baseCalled = false

	// Invoke a regular tool
	baseParams := map[string]interface{}{}
	baseResult, err := invoker.InvokeTool(ctx, "regular_tool", baseParams)
	require.NoError(t, err)
	assert.Equal(t, "base", baseResult["from"])
	assert.False(t, skillCalled)
	assert.True(t, baseCalled)
}

func TestInvokeTool_ComplexArgs(t *testing.T) {
	registry := skill.NewInMemorySkillRegistry()
	ctx := context.Background()

	// Test with complex nested arguments
	executor := &mockSkillExecutor{
		executeFn: func(ctx context.Context, skillName string, args map[string]interface{}) (map[string]interface{}, error) {
			// Verify complex nested structure
			assert.Equal(t, "test-skill", skillName)
			assert.Equal(t, "value1", args["key1"])

			nested, ok := args["nested"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "nested value", nested["inner"])

			list, ok := args["list"].([]interface{})
			require.True(t, ok)
			assert.Len(t, list, 3)
			assert.Equal(t, "item1", list[0])

			return map[string]interface{}{"processed": true}, nil
		},
	}

	baseInvoker := &mockBaseInvoker{}
	invoker := NewSkillInvoker(registry, executor, baseInvoker)

	params := map[string]interface{}{
		"skill_name": "test-skill",
		"args": map[string]interface{}{
			"key1": "value1",
			"nested": map[string]interface{}{
				"inner": "nested value",
			},
			"list": []interface{}{"item1", "item2", "item3"},
		},
	}

	result, err := invoker.InvokeTool(ctx, "invoke_skill", params)

	assert.NoError(t, err)
	assert.Equal(t, true, result["processed"])
}
