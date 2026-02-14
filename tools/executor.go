package tools

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"
)

const defaultToolTimeout = 30 * time.Second

// getGlobalToolTimeout reads the TOOL_EXECUTION_TIMEOUT environment variable (in seconds).
// Returns the default timeout (30s) if the variable is not set or invalid.
func getGlobalToolTimeout() time.Duration {
	if envVal := os.Getenv("TOOL_EXECUTION_TIMEOUT"); envVal != "" {
		if seconds, err := strconv.Atoi(envVal); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultToolTimeout
}

// InvokeTool executes a tool with its name and parameters.
// Timeout precedence: per-tool override > TOOL_EXECUTION_TIMEOUT env var > default (30s).
func InvokeTool(ctx context.Context, name string, params map[string]interface{}) (map[string]interface{}, error) {
	tool, err := GetTool(name)
	if err != nil {
		return nil, err
	}

	timeout := getGlobalToolTimeout()
	if tool.Timeout > 0 {
		timeout = tool.Timeout
	}

	return InvokeToolWithTimeout(ctx, name, params, timeout)
}

// InvokeToolWithTimeout executes a tool with an explicit timeout.
// Validates parameters, injects project_id from context, and runs the tool
// function in a goroutine with timeout enforcement.
func InvokeToolWithTimeout(ctx context.Context, name string, params map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	tool, err := GetTool(name)
	if err != nil {
		return nil, err
	}

	// Inject project_id from context into params if not already present
	params = injectProjectIDIntoParams(ctx, params)

	if err := ValidateParameters(params, tool.Parameters); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan map[string]interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := tool.Function(ctx, params)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("tool execution timeout after %v", timeout)
	}
}
