package skill

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockToolInvoker struct {
	invokeFn func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)
}

func (m *mockToolInvoker) InvokeTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, toolName, params)
	}
	return map[string]interface{}{"status": "success"}, nil
}

type mockTraceLogger struct {
	startSpanFn func(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error)
	endSpanFn   func(ctx context.Context, span SpanHandle, metadata map[string]interface{}) error
	failSpanFn  func(ctx context.Context, span SpanHandle, errorMsg string) error
}

func (m *mockTraceLogger) StartSpan(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error) {
	if m.startSpanFn != nil {
		return m.startSpanFn(ctx, traceID, spanName, metadata)
	}
	return &mockSpanHandle{}, nil
}

func (m *mockTraceLogger) EndSpan(ctx context.Context, span SpanHandle, metadata map[string]interface{}) error {
	if m.endSpanFn != nil {
		return m.endSpanFn(ctx, span, metadata)
	}
	return nil
}

func (m *mockTraceLogger) FailSpan(ctx context.Context, span SpanHandle, errorMsg string) error {
	if m.failSpanFn != nil {
		return m.failSpanFn(ctx, span, errorMsg)
	}
	return nil
}

type mockSpanHandle struct {
	metadata map[string]interface{}
}

func (m *mockSpanHandle) AddMetadata(key string, value interface{}) {
	if m.metadata == nil {
		m.metadata = make(map[string]interface{})
	}
	m.metadata[key] = value
}

// Test functions

func TestNewSkillExecutor(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	invoker := &mockToolInvoker{}
	tracer := &mockTraceLogger{}

	executor := NewSkillExecutor(registry, invoker, tracer)

	assert.NotNil(t, executor)
	assert.Equal(t, 0, executor.GetCallDepth())
}

func TestExecute_HappyPath(t *testing.T) {
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
		Content:          "# Test Skill\n\nThis is test content.",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Mock tool invoker that returns success
	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			assert.Equal(t, "invoke_skill", toolName)
			assert.Equal(t, "test-skill", params["skill_name"])
			assert.Contains(t, params["skill_content"], "Test Skill")

			return map[string]interface{}{
				"status": "success",
				"output": "Skill executed successfully",
			}, nil
		},
	}

	executor := NewSkillExecutor(registry, invoker, nil)

	// Execute the skill
	args := map[string]interface{}{
		"input": "test input",
	}
	result, err := executor.Execute(ctx, "test-skill", args)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result["status"])
	assert.Equal(t, "Skill executed successfully", result["output"])
}

func TestExecute_SkillNotFound(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	invoker := &mockToolInvoker{}
	executor := NewSkillExecutor(registry, invoker, nil)

	ctx := context.Background()
	_, err := executor.Execute(ctx, "nonexistent-skill", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill execution failed")
	assert.Contains(t, err.Error(), "nonexistent-skill")
}

func TestExecute_InvocationError(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "failing-skill",
		Description:      "A skill that fails",
		Triggers:         []string{"fail trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Failing Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Mock tool invoker that returns an error
	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return nil, errors.New("simulated execution error")
		},
	}

	executor := NewSkillExecutor(registry, invoker, nil)

	// Execute the skill
	_, err := executor.Execute(ctx, "failing-skill", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to invoke skill")
	assert.Contains(t, err.Error(), "failing-skill")
	assert.Contains(t, err.Error(), "simulated execution error")
}

func TestExecute_WithTracing(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "traced-skill",
		Description:      "A skill with tracing",
		PluginName:       "test-plugin",
		Triggers:         []string{"trace trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             2,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Traced Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Track tracing calls
	spanStarted := false
	spanEnded := false

	tracer := &mockTraceLogger{
		startSpanFn: func(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error) {
			spanStarted = true
			assert.Contains(t, spanName, "execute-skill:traced-skill")
			assert.Equal(t, "traced-skill", metadata["skill_name"])
			assert.Equal(t, 2, metadata["tier"])
			assert.Equal(t, "test-plugin", metadata["plugin"])
			return &mockSpanHandle{}, nil
		},
		endSpanFn: func(ctx context.Context, span SpanHandle, metadata map[string]interface{}) error {
			spanEnded = true
			return nil
		},
	}

	invoker := &mockToolInvoker{}
	executor := NewSkillExecutor(registry, invoker, tracer)

	// Execute the skill
	_, err := executor.Execute(ctx, "traced-skill", nil)

	assert.NoError(t, err)
	assert.True(t, spanStarted, "Span should be started")
	assert.True(t, spanEnded, "Span should be ended")
}

func TestExecute_TracingFailure(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "failing-skill",
		Description:      "A skill that fails with tracing",
		Triggers:         []string{"fail trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Failing Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Track tracing calls
	spanFailed := false

	tracer := &mockTraceLogger{
		startSpanFn: func(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error) {
			return &mockSpanHandle{}, nil
		},
		failSpanFn: func(ctx context.Context, span SpanHandle, errorMsg string) error {
			spanFailed = true
			assert.Contains(t, errorMsg, "simulated execution error")
			return nil
		},
	}

	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			return nil, errors.New("simulated execution error")
		},
	}

	executor := NewSkillExecutor(registry, invoker, tracer)

	// Execute the skill
	_, err := executor.Execute(ctx, "failing-skill", nil)

	assert.Error(t, err)
	assert.True(t, spanFailed, "Span should be failed")
}

func TestExecute_ArgumentsPassed(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill
	skill := &SkillDefinition{
		Name:             "args-skill",
		Description:      "A skill with arguments",
		Triggers:         []string{"args trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Args Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Verify arguments are passed correctly
	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			args, ok := params["args"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "test value", args["key1"])
			assert.Equal(t, 123, args["key2"])

			return map[string]interface{}{"status": "ok"}, nil
		},
	}

	executor := NewSkillExecutor(registry, invoker, nil)

	// Execute with arguments
	args := map[string]interface{}{
		"key1": "test value",
		"key2": 123,
	}
	result, err := executor.Execute(ctx, "args-skill", args)

	assert.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestExecute_MetadataIncluded(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a test skill with rich metadata
	skill := &SkillDefinition{
		Name:             "metadata-skill",
		Description:      "A skill with rich metadata",
		PluginName:       "metadata-plugin",
		Triggers:         []string{"metadata trigger", "another trigger"},
		ToolDependencies: []string{"file_system", "bash"},
		Tier:             2,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Metadata Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Verify metadata is passed correctly
	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			metadata, ok := params["metadata"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, 2, metadata["tier"])
			assert.Equal(t, "metadata-plugin", metadata["plugin"])
			assert.Equal(t, true, metadata["portable"])
			assert.Equal(t, "A skill with rich metadata", metadata["description"])

			triggers, ok := metadata["triggers"].([]string)
			assert.True(t, ok)
			assert.Contains(t, triggers, "metadata trigger")
			assert.Contains(t, triggers, "another trigger")

			return map[string]interface{}{"status": "ok"}, nil
		},
	}

	executor := NewSkillExecutor(registry, invoker, nil)

	// Execute the skill
	_, err := executor.Execute(ctx, "metadata-skill", nil)

	assert.NoError(t, err)
}

func TestSetGetCallDepth(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	invoker := &mockToolInvoker{}
	executor := NewSkillExecutor(registry, invoker, nil)

	assert.Equal(t, 0, executor.GetCallDepth())

	executor.SetCallDepth(2)
	assert.Equal(t, 2, executor.GetCallDepth())

	executor.SetCallDepth(0)
	assert.Equal(t, 0, executor.GetCallDepth())
}

func TestValidateDependencies_WebToolsMissing(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a Tier 2 skill requiring web_tools
	skill := &SkillDefinition{
		Name:             "web-skill",
		Description:      "A skill requiring web tools",
		Triggers:         []string{"web trigger"},
		ToolDependencies: []string{"web_tools"},
		Tier:             2,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Web Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Mock invoker returns "not found" for web_search
	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			if toolName == "web_search" {
				return nil, errors.New("tool not found: web_search")
			}
			return map[string]interface{}{"status": "ok"}, nil
		},
	}

	executor := NewSkillExecutor(registry, invoker, nil)

	_, err := executor.Execute(ctx, "web-skill", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmet dependencies")
	assert.Contains(t, err.Error(), "web_tools adapter required but not loaded")
}

func TestValidateDependencies_ScriptExecutorMissing(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a skill requiring script_execution
	skill := &SkillDefinition{
		Name:             "script-skill",
		Description:      "A skill requiring script execution",
		Triggers:         []string{"script trigger"},
		ToolDependencies: []string{"script_execution"},
		Tier:             2,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Script Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	// Mock invoker returns "not registered" for execute_script
	invoker := &mockToolInvoker{
		invokeFn: func(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
			if toolName == "execute_script" {
				return nil, errors.New("tool not registered: execute_script")
			}
			return map[string]interface{}{"status": "ok"}, nil
		},
	}

	executor := NewSkillExecutor(registry, invoker, nil)

	_, err := executor.Execute(ctx, "script-skill", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmet dependencies")
	assert.Contains(t, err.Error(), "script executor required but not available")
}

func TestValidateDependencies_Tier1Portable(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a Tier 1 portable skill with no web_tools or script_execution deps
	skill := &SkillDefinition{
		Name:             "portable-skill",
		Description:      "A portable Tier 1 skill",
		Triggers:         []string{"portable trigger"},
		ToolDependencies: []string{"file_system"},
		Tier:             1,
		Portable:         true,
		Agents:           []string{"agent1"},
		Content:          "# Portable Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	invoker := &mockToolInvoker{}
	executor := NewSkillExecutor(registry, invoker, nil)

	result, err := executor.Execute(ctx, "portable-skill", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateDependencies_NoDependencies(t *testing.T) {
	registry := NewInMemorySkillRegistry()
	ctx := context.Background()

	// Register a skill with no tool dependencies
	skill := &SkillDefinition{
		Name:        "no-deps-skill",
		Description: "A skill with no dependencies",
		Triggers:    []string{"no deps trigger"},
		Tier:        1,
		Portable:    true,
		Agents:      []string{"agent1"},
		Content:     "# No Deps Skill",
	}
	require.NoError(t, registry.RegisterSkill(ctx, skill))

	invoker := &mockToolInvoker{}
	executor := NewSkillExecutor(registry, invoker, nil)

	result, err := executor.Execute(ctx, "no-deps-skill", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
