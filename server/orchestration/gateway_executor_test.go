package orchestration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCancel_NotFound(t *testing.T) {
	executor := &GatewayOrchestrationExecutor{
		executions: make(map[string]*executionState),
	}

	err := executor.Cancel(context.Background(), "nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution not found")
	assert.Contains(t, err.Error(), "nonexistent-id")
}

func TestCancel_ExistingExecution(t *testing.T) {
	executor := &GatewayOrchestrationExecutor{
		executions: make(map[string]*executionState),
	}

	// Simulate a tracked execution
	ctx, cancel := context.WithCancel(context.Background())
	executor.executions["exec-1"] = &executionState{
		ctx:    ctx,
		cancel: cancel,
	}

	// Cancel should succeed
	err := executor.Cancel(context.Background(), "exec-1")
	assert.NoError(t, err)

	// Verify context was cancelled
	assert.Error(t, ctx.Err())
}

func TestStatus_NotFound(t *testing.T) {
	executor := &GatewayOrchestrationExecutor{
		executions: make(map[string]*executionState),
	}

	status, err := executor.Status("nonexistent-id")
	assert.Error(t, err)
	assert.Equal(t, "not_found", status)
	assert.Contains(t, err.Error(), "execution not found")
}

func TestStatus_Running(t *testing.T) {
	executor := &GatewayOrchestrationExecutor{
		executions: make(map[string]*executionState),
	}

	// Simulate a running execution
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	executor.executions["exec-1"] = &executionState{
		ctx:    ctx,
		cancel: cancel,
	}

	status, err := executor.Status("exec-1")
	assert.NoError(t, err)
	assert.Equal(t, "running", status)
}

func TestStatus_Completed(t *testing.T) {
	executor := &GatewayOrchestrationExecutor{
		executions: make(map[string]*executionState),
	}

	// Simulate a completed execution
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	executor.executions["exec-1"] = &executionState{
		ctx:       ctx,
		cancel:    cancel,
		completed: true,
	}

	status, err := executor.Status("exec-1")
	assert.NoError(t, err)
	assert.Equal(t, "completed", status)
}

func TestStatus_Cancelled(t *testing.T) {
	executor := &GatewayOrchestrationExecutor{
		executions: make(map[string]*executionState),
	}

	// Simulate a cancelled execution (context cancelled but not marked completed)
	ctx, cancel := context.WithCancel(context.Background())
	executor.executions["exec-1"] = &executionState{
		ctx:    ctx,
		cancel: cancel,
	}

	// Cancel the context
	cancel()

	status, err := executor.Status("exec-1")
	assert.NoError(t, err)
	assert.Equal(t, "cancelled", status)
}

func TestCancel_AlreadyCompleted(t *testing.T) {
	executor := &GatewayOrchestrationExecutor{
		executions: make(map[string]*executionState),
	}

	// Simulate a completed execution
	ctx, cancel := context.WithCancel(context.Background())
	executor.executions["exec-1"] = &executionState{
		ctx:       ctx,
		cancel:    cancel,
		completed: true,
	}

	// Cancel should still succeed (calling cancel on completed context is harmless)
	err := executor.Cancel(context.Background(), "exec-1")
	assert.NoError(t, err)

	// Status should still show completed (completed takes precedence over cancelled)
	status, err := executor.Status("exec-1")
	assert.NoError(t, err)
	assert.Equal(t, "completed", status)
}
