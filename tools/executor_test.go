package tools

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeToolWithPerToolTimeout(t *testing.T) {
	setupTestRegistry(t)
	ClearRegistry()

	def := &ToolDefinition{
		Name: "custom_timeout_tool",
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"done": true}, nil
		},
		Timeout: 5 * time.Second,
	}

	err := RegisterTool(def)
	require.NoError(t, err)

	result, err := InvokeTool(context.Background(), "custom_timeout_tool", map[string]interface{}{})
	assert.NoError(t, err)
	assert.True(t, result["done"].(bool))
}

func TestInvokeToolTimeoutPrecedence(t *testing.T) {
	setupTestRegistry(t)

	t.Run("per-tool timeout takes precedence over env var", func(t *testing.T) {
		ClearRegistry()
		os.Setenv("TOOL_EXECUTION_TIMEOUT", "1")
		defer os.Unsetenv("TOOL_EXECUTION_TIMEOUT")

		def := &ToolDefinition{
			Name: "precedence_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				time.Sleep(1500 * time.Millisecond)
				return map[string]interface{}{"done": true}, nil
			},
			Timeout: 5 * time.Second,
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		result, err := InvokeTool(context.Background(), "precedence_tool", map[string]interface{}{})
		assert.NoError(t, err)
		assert.True(t, result["done"].(bool))
	})

	t.Run("env var timeout used when no per-tool timeout", func(t *testing.T) {
		ClearRegistry()
		os.Setenv("TOOL_EXECUTION_TIMEOUT", "60")
		defer os.Unsetenv("TOOL_EXECUTION_TIMEOUT")

		timeout := getGlobalToolTimeout()
		assert.Equal(t, 60*time.Second, timeout)
	})

	t.Run("default timeout used when nothing set", func(t *testing.T) {
		os.Unsetenv("TOOL_EXECUTION_TIMEOUT")
		timeout := getGlobalToolTimeout()
		assert.Equal(t, 30*time.Second, timeout)
	})

	t.Run("invalid env var falls back to default", func(t *testing.T) {
		os.Setenv("TOOL_EXECUTION_TIMEOUT", "invalid")
		defer os.Unsetenv("TOOL_EXECUTION_TIMEOUT")

		timeout := getGlobalToolTimeout()
		assert.Equal(t, 30*time.Second, timeout)
	})

	t.Run("negative env var falls back to default", func(t *testing.T) {
		os.Setenv("TOOL_EXECUTION_TIMEOUT", "-5")
		defer os.Unsetenv("TOOL_EXECUTION_TIMEOUT")

		timeout := getGlobalToolTimeout()
		assert.Equal(t, 30*time.Second, timeout)
	})
}

func TestInvokeToolWithProjectContext(t *testing.T) {
	setupTestRegistry(t)
	ClearRegistry()

	def := &ToolDefinition{
		Name: "context_aware_tool",
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"project_id": params["project_id"],
			}, nil
		},
	}

	err := RegisterTool(def)
	require.NoError(t, err)

	ctx := WithProjectID(context.Background(), "proj-456")
	result, err := InvokeTool(ctx, "context_aware_tool", map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, "proj-456", result["project_id"])
}

func TestInvokeToolValidationBeforeExecution(t *testing.T) {
	setupTestRegistry(t)
	ClearRegistry()

	executed := false
	def := &ToolDefinition{
		Name: "guarded_tool",
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			executed = true
			return map[string]interface{}{}, nil
		},
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
			},
			"required": []interface{}{"query"},
		},
	}

	err := RegisterTool(def)
	require.NoError(t, err)

	_, err = InvokeTool(context.Background(), "guarded_tool", map[string]interface{}{})
	assert.Error(t, err)
	assert.False(t, executed, "Tool function should not execute if validation fails")
}

func TestInvokeToolWithTimeoutExplicit(t *testing.T) {
	setupTestRegistry(t)
	ClearRegistry()

	def := &ToolDefinition{
		Name: "explicit_timeout_tool",
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			select {
			case <-time.After(5 * time.Second):
				return map[string]interface{}{"done": true}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	err := RegisterTool(def)
	require.NoError(t, err)

	result, err := InvokeToolWithTimeout(context.Background(), "explicit_timeout_tool", map[string]interface{}{}, 100*time.Millisecond)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "timeout")
}
