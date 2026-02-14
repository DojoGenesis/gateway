package tools

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	savedTools := GetAllTools()
	t.Cleanup(func() {
		ClearRegistry()
		for _, tool := range savedTools {
			RegisterTool(tool)
		}
	})
}

func TestRegisterTool(t *testing.T) {
	setupTestRegistry(t)

	t.Run("successful registration", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name:        "test_tool",
			Description: "A test tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"success": true}, nil
			},
		}

		err := RegisterTool(def)
		assert.NoError(t, err)

		retrieved, err := GetTool("test_tool")
		assert.NoError(t, err)
		assert.Equal(t, "test_tool", retrieved.Name)
	})

	t.Run("nil tool definition", func(t *testing.T) {
		ClearRegistry()
		err := RegisterTool(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("empty tool name", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		}

		err := RegisterTool(def)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("nil tool function", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name:     "test_tool",
			Function: nil,
		}

		err := RegisterTool(def)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function cannot be nil")
	})

	t.Run("duplicate registration", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "duplicate_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		}

		err := RegisterTool(def)
		assert.NoError(t, err)

		err = RegisterTool(def)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

func TestGetTool(t *testing.T) {
	setupTestRegistry(t)

	t.Run("retrieve existing tool", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name:        "existing_tool",
			Description: "An existing tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"result": "success"}, nil
			},
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		retrieved, err := GetTool("existing_tool")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "existing_tool", retrieved.Name)
		assert.Equal(t, "An existing tool", retrieved.Description)
	})

	t.Run("tool not found", func(t *testing.T) {
		ClearRegistry()
		tool, err := GetTool("nonexistent_tool")
		assert.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestGetAllTools(t *testing.T) {
	setupTestRegistry(t)

	t.Run("get all registered tools", func(t *testing.T) {
		ClearRegistry()

		tools := []*ToolDefinition{
			{
				Name: "tool1",
				Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				},
			},
			{
				Name: "tool2",
				Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				},
			},
			{
				Name: "tool3",
				Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				},
			},
		}

		for _, tool := range tools {
			err := RegisterTool(tool)
			require.NoError(t, err)
		}

		allTools := GetAllTools()
		assert.Len(t, allTools, 3)

		toolNames := make(map[string]bool)
		for _, tool := range allTools {
			toolNames[tool.Name] = true
		}

		assert.True(t, toolNames["tool1"])
		assert.True(t, toolNames["tool2"])
		assert.True(t, toolNames["tool3"])
	})

	t.Run("empty registry", func(t *testing.T) {
		ClearRegistry()
		tools := GetAllTools()
		assert.Empty(t, tools)
	})
}

func TestUnregisterTool(t *testing.T) {
	setupTestRegistry(t)

	t.Run("unregister existing tool", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "removable_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		err = UnregisterTool("removable_tool")
		assert.NoError(t, err)

		_, err = GetTool("removable_tool")
		assert.Error(t, err)
	})

	t.Run("unregister nonexistent tool", func(t *testing.T) {
		ClearRegistry()
		err := UnregisterTool("nonexistent_tool")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestInvokeTool(t *testing.T) {
	setupTestRegistry(t)

	t.Run("successful invocation", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "echo_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{
					"input":  params["message"],
					"result": "echoed",
				}, nil
			},
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type": "string",
					},
				},
			},
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		result, err := InvokeTool(context.Background(), "echo_tool", map[string]interface{}{
			"message": "hello",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "hello", result["input"])
		assert.Equal(t, "echoed", result["result"])
	})

	t.Run("tool not found", func(t *testing.T) {
		ClearRegistry()
		result, err := InvokeTool(context.Background(), "nonexistent_tool", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("function returns error", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "error_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return nil, errors.New("tool execution failed")
			},
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		result, err := InvokeTool(context.Background(), "error_tool", nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "tool execution failed")
	})

	t.Run("parameter validation failure", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "validated_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"success": true}, nil
			},
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"name"},
			},
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		result, err := InvokeTool(context.Background(), "validated_tool", map[string]interface{}{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "parameter validation failed")
	})
}

func TestInvokeToolWithTimeout(t *testing.T) {
	setupTestRegistry(t)

	t.Run("successful execution within timeout", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "fast_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"success": true}, nil
			},
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		result, err := InvokeToolWithTimeout(context.Background(), "fast_tool", nil, 1*time.Second)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result["success"].(bool))
	})

	t.Run("timeout exceeded", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "slow_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				time.Sleep(2 * time.Second)
				return map[string]interface{}{"success": true}, nil
			},
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		result, err := InvokeToolWithTimeout(context.Background(), "slow_tool", nil, 100*time.Millisecond)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("context cancellation", func(t *testing.T) {
		ClearRegistry()
		def := &ToolDefinition{
			Name: "cancelable_tool",
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				select {
				case <-time.After(2 * time.Second):
					return map[string]interface{}{"success": true}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		}

		err := RegisterTool(def)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := InvokeToolWithTimeout(ctx, "cancelable_tool", nil, 5*time.Second)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestConcurrentAccess(t *testing.T) {
	setupTestRegistry(t)

	ClearRegistry()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			toolName := "concurrent_tool"
			def := &ToolDefinition{
				Name: toolName,
				Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{"id": id}, nil
				},
			}

			RegisterTool(def)
			GetTool(toolName)
			GetAllTools()
		}(i)
	}

	wg.Wait()

	tools := GetAllTools()
	assert.NotEmpty(t, tools)
}

func TestClearRegistry(t *testing.T) {
	savedTools := GetAllTools()
	defer func() {
		ClearRegistry()
		for _, tool := range savedTools {
			RegisterTool(tool)
		}
	}()

	ClearRegistry()

	for i := 0; i < 5; i++ {
		def := &ToolDefinition{
			Name: "tool_" + string(rune(i)),
			Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		}
		RegisterTool(def)
	}

	tools := GetAllTools()
	assert.NotEmpty(t, tools)

	ClearRegistry()

	tools = GetAllTools()
	assert.Empty(t, tools)
}
