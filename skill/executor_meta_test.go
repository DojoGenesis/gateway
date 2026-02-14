package skill

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteAsSubtask_BasicFunctionality tests that ExecuteAsSubtask works end-to-end
func TestExecuteAsSubtask_BasicFunctionality(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a parent skill
	parentSkill := &SkillDefinition{
		Name:             "parent-skill",
		Description:      "A parent skill that invokes children",
		PluginName:       "test-plugin",
		Triggers:         []string{"parent trigger"},
		ToolDependencies: []string{"meta_skill"},
		Tier:             3,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Parent Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, parentSkill))

	// Register a child skill
	childSkill := &SkillDefinition{
		Name:             "child-skill",
		Description:      "A child skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"child trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Child Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, childSkill))

	// Mock tool invoker that returns success
	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"status": "success", "result": "completed"}, nil
		},
	}

	tracer := &mockTraceLogger{}

	executor := NewSkillExecutor(registry, invoker, tracer)

	// Execute child as subtask
	result, err := executor.ExecuteAsSubtask(ctx, "child-skill", map[string]interface{}{
		"arg1": "value1",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result["status"])
}

// TestExecuteAsSubtask_DepthTracking verifies that call depth is incremented
func TestExecuteAsSubtask_DepthTracking(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"test trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Test Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	invoker := &mockToolInvoker{
		invokeFn: func(invokeCtx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			// Verify depth was incremented in the invoked context
			depth := GetCallDepth(invokeCtx)
			return map[string]interface{}{"depth": depth}, nil
		},
	}

	tracer := &mockTraceLogger{}
	executor := NewSkillExecutor(registry, invoker, tracer)

	// Start at depth 0
	assert.Equal(t, 0, GetCallDepth(ctx))

	// Execute as subtask
	result, err := executor.ExecuteAsSubtask(ctx, "test-skill", nil)
	require.NoError(t, err)

	// Result should show depth was 1 during execution
	assert.Equal(t, 1, result["depth"])

	// Context depth should still be 0 (not mutated)
	assert.Equal(t, 0, GetCallDepth(ctx))
}

// TestExecuteAsSubtask_MaxDepthExceeded tests depth limit enforcement
func TestExecuteAsSubtask_MaxDepthExceeded(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"test trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Test Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	invoker := &mockToolInvoker{}
	tracer := &mockTraceLogger{}
	executor := NewSkillExecutor(registry, invoker, tracer)

	// Set context to depth 3 (at the limit)
	ctx = context.WithValue(ctx, callDepthKey, 3)

	// Attempt to execute as subtask
	result, err := executor.ExecuteAsSubtask(ctx, "test-skill", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrMaxDepthExceeded)
	assert.Contains(t, err.Error(), "current depth 3 exceeds maximum 3")
}

// TestExecuteAsSubtask_SkillNotFound tests error handling for missing skill
func TestExecuteAsSubtask_SkillNotFound(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	invoker := &mockToolInvoker{}
	tracer := &mockTraceLogger{}
	executor := NewSkillExecutor(registry, invoker, tracer)

	// Attempt to execute non-existent skill
	result, err := executor.ExecuteAsSubtask(ctx, "non-existent-skill", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrSkillNotFound)
	assert.Contains(t, err.Error(), "meta-skill invocation failed for 'non-existent-skill'")
}

// TestExecuteAsSubtask_BudgetTracking tests token budget reservation and consumption
func TestExecuteAsSubtask_BudgetTracking(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"test trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Test Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Create budget tracker with 10,000 tokens
	tracker := NewBudgetTracker(10000)
	ctx = WithBudgetTracker(ctx, tracker)

	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"status": "success"}, nil
		},
	}

	tracer := &mockTraceLogger{}
	executor := NewSkillExecutor(registry, invoker, tracer)

	// Execute as subtask
	_, err := executor.ExecuteAsSubtask(ctx, "test-skill", nil)
	require.NoError(t, err)

	// Verify budget was consumed
	assert.Greater(t, tracker.ConsumedTokens, 0)
	assert.Less(t, tracker.ConsumedTokens, 10000)
	assert.Equal(t, 0, tracker.ReservedTokens) // All reservations should be consumed or released
}

// TestExecuteAsSubtask_BudgetExhausted tests budget exhaustion handling
func TestExecuteAsSubtask_BudgetExhausted(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"test trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Test Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Create budget tracker with insufficient tokens
	tracker := NewBudgetTracker(500) // Less than the 1000 token estimate
	ctx = WithBudgetTracker(ctx, tracker)

	invoker := &mockToolInvoker{}
	tracer := &mockTraceLogger{}
	executor := NewSkillExecutor(registry, invoker, tracer)

	// Attempt to execute as subtask
	result, err := executor.ExecuteAsSubtask(ctx, "test-skill", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrBudgetExhausted)
	assert.Contains(t, err.Error(), "requested 1000 tokens")
}

// TestExecuteAsSubtask_ExecutionError tests error propagation from child execution
func TestExecuteAsSubtask_ExecutionError(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"test trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Test Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Create budget tracker
	tracker := NewBudgetTracker(10000)
	ctx = WithBudgetTracker(ctx, tracker)

	// Mock tool invoker that returns error
	expectedErr := errors.New("execution failed")
	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return nil, expectedErr
		},
	}

	tracer := &mockTraceLogger{}
	executor := NewSkillExecutor(registry, invoker, tracer)

	// Execute as subtask
	result, err := executor.ExecuteAsSubtask(ctx, "test-skill", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "meta-skill 'test-skill' execution failed")

	// Verify budget reservation was released on error
	assert.Equal(t, 0, tracker.ReservedTokens)
	assert.Equal(t, 0, tracker.ConsumedTokens) // Error means no consumption
}

// TestExecuteAsSubtask_WithoutBudgetTracker tests that execution works without budget tracking
func TestExecuteAsSubtask_WithoutBudgetTracker(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"test trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Test Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// No budget tracker in context

	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"status": "success"}, nil
		},
	}

	tracer := &mockTraceLogger{}
	executor := NewSkillExecutor(registry, invoker, tracer)

	// Execute as subtask - should work without budget tracker
	result, err := executor.ExecuteAsSubtask(ctx, "test-skill", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result["status"])
}

// TestExecuteAsSubtask_TracingIntegration tests OTEL span creation
func TestExecuteAsSubtask_TracingIntegration(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "test-skill",
		Description:      "A test skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"test trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Test Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"status": "success"}, nil
		},
	}

	var metaSkillSpanStarted bool
	var regularSpanStarted bool
	var metaSkillSpanEnded bool
	tracer := &mockTraceLogger{
		startSpanFn: func(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error) {
			// ExecuteAsSubtask creates a meta-skill span
			if _, ok := metadata["skill.parent_type"]; ok {
				metaSkillSpanStarted = true
				assert.Contains(t, spanName, "meta-skill.invoke.test-skill")
				assert.Equal(t, "test-skill", metadata["skill.name"])
				assert.Equal(t, 1, metadata["skill.tier"])
				assert.Equal(t, 1, metadata["skill.call_depth"])
				assert.Equal(t, "meta_skill_invocation", metadata["skill.parent_type"])
			} else {
				// Execute also creates its own span
				regularSpanStarted = true
			}
			return &mockSpanHandle{}, nil
		},
		endSpanFn: func(ctx context.Context, span SpanHandle, metadata map[string]interface{}) error {
			// Check for meta-skill span end (has tokens_used and status)
			if status, ok := metadata["skill.result.status"]; ok && status == "success" {
				if _, hasTokens := metadata["skill.result.tokens_used"]; hasTokens {
					metaSkillSpanEnded = true
				}
			}
			return nil
		},
	}

	executor := NewSkillExecutor(registry, invoker, tracer)

	// Execute as subtask
	_, err := executor.ExecuteAsSubtask(ctx, "test-skill", nil)
	require.NoError(t, err)

	assert.True(t, metaSkillSpanStarted, "Expected meta-skill StartSpan to be called")
	assert.True(t, regularSpanStarted, "Expected regular Execute StartSpan to be called")
	assert.True(t, metaSkillSpanEnded, "Expected meta-skill EndSpan to be called")
}
