package skill

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetaSkill_EndToEnd_MultiLevel tests a complete meta-skill chain
// with multiple levels of invocation, budget tracking, and tracing.
func TestMetaSkill_EndToEnd_MultiLevel(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a hierarchy of skills:
	// parent-skill → child-skill-1 → child-skill-2

	parentSkill := &SkillDefinition{
		Name:             "parent-skill",
		Description:      "Top-level parent skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"parent trigger"},
		ToolDependencies: []string{"meta_skill"},
		Tier:             3,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Parent Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, parentSkill))

	childSkill1 := &SkillDefinition{
		Name:             "child-skill-1",
		Description:      "First-level child skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"child-1 trigger"},
		ToolDependencies: []string{"meta_skill"},
		Tier:             3,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Child Skill 1",
	}
	require.NoError(t, registry.RegisterSkill(ctx, childSkill1))

	childSkill2 := &SkillDefinition{
		Name:             "child-skill-2",
		Description:      "Second-level child skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"child-2 trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Child Skill 2",
	}
	require.NoError(t, registry.RegisterSkill(ctx, childSkill2))

	// Create budget tracker with 10,000 tokens
	tracker := NewBudgetTracker(10000)
	ctx = WithBudgetTracker(ctx, tracker)

	// Track OTEL spans
	var spansMu sync.Mutex
	spans := make([]string, 0)

	// Mock tool invoker that simulates the skill execution
	// For this test, we'll invoke the next skill in the chain
	var executor *DefaultSkillExecutor
	invoker := &mockToolInvoker{
		invokeFn: func(invokeCtx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			if toolName != "invoke_skill" {
				return nil, errors.New("unexpected tool")
			}

			skillName := params["skill_name"].(string)
			depth := GetCallDepth(invokeCtx)

			// Simulate meta-skill invocation within the skill content
			switch skillName {
			case "parent-skill":
				// Parent invokes child-skill-1
				if depth == 1 {
					result, err := executor.ExecuteAsSubtask(invokeCtx, "child-skill-1", map[string]interface{}{
						"from": "parent",
					})
					if err != nil {
						return nil, err
					}
					return map[string]interface{}{
						"status":       "parent-completed",
						"child_result": result,
					}, nil
				}

			case "child-skill-1":
				// Child-1 invokes child-skill-2
				if depth == 2 {
					result, err := executor.ExecuteAsSubtask(invokeCtx, "child-skill-2", map[string]interface{}{
						"from": "child-1",
					})
					if err != nil {
						return nil, err
					}
					return map[string]interface{}{
						"status":       "child-1-completed",
						"child_result": result,
					}, nil
				}

			case "child-skill-2":
				// Leaf skill - no further invocation
				return map[string]interface{}{
					"status": "child-2-completed",
					"depth":  depth,
				}, nil
			}

			return map[string]interface{}{"status": "completed"}, nil
		},
	}

	tracer := &mockTraceLogger{
		startSpanFn: func(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error) {
			spansMu.Lock()
			defer spansMu.Unlock()
			spans = append(spans, spanName)
			return &mockSpanHandle{}, nil
		},
	}

	executor = NewSkillExecutor(registry, invoker, tracer)

	// Execute parent skill (which will invoke child-skill-1, which will invoke child-skill-2)
	result, err := executor.ExecuteAsSubtask(ctx, "parent-skill", map[string]interface{}{
		"initial": "request",
	})

	// Verify execution succeeded
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the full chain executed
	assert.Equal(t, "parent-completed", result["status"])
	assert.Contains(t, result, "child_result")

	// Verify budget was consumed
	assert.Greater(t, tracker.ConsumedTokens, 0)
	assert.Less(t, tracker.ConsumedTokens, 10000)
	assert.Equal(t, 0, tracker.ReservedTokens) // All reservations settled

	// Verify OTEL spans were created for the chain
	// We expect spans for each level of invocation
	assert.GreaterOrEqual(t, len(spans), 3, "Expected spans for multi-level invocation")
}

// TestMetaSkill_MaxDepthEnforced tests that the depth limit is strictly enforced
// by creating a recursive skill that always calls itself.
func TestMetaSkill_MaxDepthEnforced(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a recursive test skill
	recursiveSkill := &SkillDefinition{
		Name:             "recursive-skill",
		Description:      "A skill that recursively calls itself",
		PluginName:       "test-plugin",
		Triggers:         []string{"recursive trigger"},
		ToolDependencies: []string{"meta_skill"},
		Tier:             3,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Recursive Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, recursiveSkill))

	// Track invocation depth and recursion attempts
	var maxDepthReached int
	var depthMu sync.Mutex
	var recursionAttempts int

	// Mock tool invoker that always tries to recurse
	var executor *DefaultSkillExecutor
	invoker := &mockToolInvoker{
		invokeFn: func(invokeCtx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			if toolName != "invoke_skill" {
				return nil, errors.New("unexpected tool")
			}

			currentDepth := GetCallDepth(invokeCtx)

			depthMu.Lock()
			if currentDepth > maxDepthReached {
				maxDepthReached = currentDepth
			}
			recursionAttempts++
			depthMu.Unlock()

			// Try to recurse (this should eventually fail at depth 3)
			childResult, err := executor.ExecuteAsSubtask(invokeCtx, "recursive-skill", map[string]interface{}{
				"depth": currentDepth,
			})

			// If recursion failed due to depth, return success
			// (the skill handles the depth limit gracefully)
			if err != nil && errors.Is(err, ErrMaxDepthExceeded) {
				return map[string]interface{}{
					"status":      "depth-limit-reached",
					"final_depth": currentDepth,
				}, nil
			}

			// If recursion had an error but it wasn't depth limit, propagate it
			if err != nil {
				return nil, err
			}

			// If the child hit depth limit, propagate that status up
			if childResult != nil {
				if status, ok := childResult["status"].(string); ok && status == "depth-limit-reached" {
					return childResult, nil
				}
			}

			// If recursion succeeded, return the child result
			return map[string]interface{}{
				"status": "recursed",
				"depth":  currentDepth,
			}, nil
		},
	}

	tracer := &mockTraceLogger{}
	executor = NewSkillExecutor(registry, invoker, tracer)

	// Execute the recursive skill starting at depth 0
	result, err := executor.ExecuteAsSubtask(ctx, "recursive-skill", nil)

	// The execution should succeed (the skill handles depth limit gracefully)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify that we hit the depth limit
	// Starting at depth 0, ExecuteAsSubtask increments to 1, then 2, then 3
	// At depth 3, the next ExecuteAsSubtask will fail with ErrMaxDepthExceeded
	assert.Equal(t, "depth-limit-reached", result["status"], "Expected to hit depth limit")
	assert.Equal(t, 3, maxDepthReached, "Expected to reach depth 3 before being blocked")
	assert.GreaterOrEqual(t, recursionAttempts, 3, "Expected at least 3 recursion attempts")
}

// TestMetaSkill_BudgetExhaustedMidChain tests budget exhaustion during a chain
func TestMetaSkill_BudgetExhaustedMidChain(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register parent and child skills
	parentSkill := &SkillDefinition{
		Name:             "parent-skill",
		Description:      "Parent skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"parent trigger"},
		ToolDependencies: []string{"meta_skill"},
		Tier:             3,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Parent Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, parentSkill))

	childSkill := &SkillDefinition{
		Name:             "child-skill",
		Description:      "Child skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"child trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Child Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, childSkill))

	// Create budget tracker with only 1500 tokens
	// This is enough for the parent (1000) but not enough for parent + child (2000)
	tracker := NewBudgetTracker(1500)
	ctx = WithBudgetTracker(ctx, tracker)

	var executor *DefaultSkillExecutor
	invoker := &mockToolInvoker{
		invokeFn: func(invokeCtx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			skillName := params["skill_name"].(string)

			if skillName == "parent-skill" {
				// Parent tries to invoke child (this should fail due to budget)
				_, err := executor.ExecuteAsSubtask(invokeCtx, "child-skill", nil)
				if err != nil {
					return nil, err // Propagate budget exhaustion error
				}
				return map[string]interface{}{"status": "parent-completed"}, nil
			}

			return map[string]interface{}{"status": "completed"}, nil
		},
	}

	tracer := &mockTraceLogger{}
	executor = NewSkillExecutor(registry, invoker, tracer)

	// Execute parent skill
	result, err := executor.ExecuteAsSubtask(ctx, "parent-skill", nil)

	// Execution should fail due to budget exhaustion
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrBudgetExhausted)
	assert.Contains(t, err.Error(), "requested 1000 tokens")

	// Verify budget state
	// Parent reserved 1000, consumed 0 (failed), so we should have 1500 remaining
	assert.Equal(t, 0, tracker.ConsumedTokens, "No tokens should be consumed on error")
	assert.Equal(t, 0, tracker.ReservedTokens, "All reservations should be released")
	assert.Equal(t, 1500, tracker.Remaining(), "Budget should be fully restored")
}

// TestMetaSkill_ErrorPropagation tests that errors are properly propagated
// through the meta-skill chain with context preserved.
func TestMetaSkill_ErrorPropagation(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register parent and child skills
	parentSkill := &SkillDefinition{
		Name:             "parent-skill",
		Description:      "Parent skill",
		PluginName:       "test-plugin",
		Triggers:         []string{"parent trigger"},
		ToolDependencies: []string{"meta_skill"},
		Tier:             3,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Parent Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, parentSkill))

	childSkill := &SkillDefinition{
		Name:             "child-skill",
		Description:      "Child skill that fails",
		PluginName:       "test-plugin",
		Triggers:         []string{"child trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Child Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, childSkill))

	expectedErr := errors.New("child execution failed")

	var executor *DefaultSkillExecutor
	invoker := &mockToolInvoker{
		invokeFn: func(invokeCtx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			skillName := params["skill_name"].(string)

			if skillName == "parent-skill" {
				// Parent invokes child
				_, err := executor.ExecuteAsSubtask(invokeCtx, "child-skill", nil)
				return nil, err // Propagate child error
			}

			if skillName == "child-skill" {
				// Child fails
				return nil, expectedErr
			}

			return map[string]interface{}{"status": "completed"}, nil
		},
	}

	tracer := &mockTraceLogger{}
	executor = NewSkillExecutor(registry, invoker, tracer)

	// Execute parent skill
	result, err := executor.ExecuteAsSubtask(ctx, "parent-skill", nil)

	// Execution should fail
	require.Error(t, err)
	assert.Nil(t, result)

	// Verify error chain preserves context
	assert.Contains(t, err.Error(), "meta-skill 'child-skill' execution failed")
	assert.Contains(t, err.Error(), "depth 2") // Child was at depth 2
	assert.ErrorIs(t, err, expectedErr)
}
