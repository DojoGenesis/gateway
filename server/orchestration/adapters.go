package orchestration

import (
	"context"
	"fmt"

	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/DojoGenesis/gateway/server/services"
	"github.com/DojoGenesis/gateway/server/trace"
	"github.com/DojoGenesis/gateway/tools"
)

// ToolInvokerAdapter adapts the tools registry to implement orchestration.ToolInvokerInterface.
type ToolInvokerAdapter struct {
	// We use the tools package directly since it provides global tool access
}

// NewToolInvokerAdapter creates a new tool invoker adapter.
func NewToolInvokerAdapter() *ToolInvokerAdapter {
	return &ToolInvokerAdapter{}
}

// InvokeTool executes a tool by name with the given parameters.
func (t *ToolInvokerAdapter) InvokeTool(ctx context.Context, toolName string, parameters map[string]interface{}) (map[string]interface{}, error) {
	// Get the tool definition from the registry
	toolDef, err := tools.GetTool(toolName)
	if err != nil {
		return nil, fmt.Errorf("ToolInvokerAdapter: tool not found '%s': %w", toolName, err)
	}

	// Execute the tool using its function
	if toolDef.Function == nil {
		return nil, fmt.Errorf("ToolInvokerAdapter: tool '%s' has no function", toolName)
	}

	result, err := toolDef.Function(ctx, parameters)
	if err != nil {
		return nil, fmt.Errorf("ToolInvokerAdapter: execution failed for tool '%s': %w", toolName, err)
	}

	return result, nil
}

// TraceLoggerAdapter adapts server/trace.TraceLogger to orchestration.TraceLoggerInterface.
type TraceLoggerAdapter struct {
	logger *trace.TraceLogger
}

// NewTraceLoggerAdapter creates a new trace logger adapter.
func NewTraceLoggerAdapter(logger *trace.TraceLogger) *TraceLoggerAdapter {
	if logger == nil {
		return nil
	}
	return &TraceLoggerAdapter{logger: logger}
}

// StartSpan begins a new trace span.
func (t *TraceLoggerAdapter) StartSpan(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (orchestrationpkg.SpanHandle, error) {
	if t == nil || t.logger == nil {
		return nil, fmt.Errorf("TraceLoggerAdapter: trace logger not initialized")
	}

	span, err := t.logger.StartSpan(ctx, traceID, spanName, metadata)
	if err != nil {
		return nil, fmt.Errorf("TraceLoggerAdapter: failed to start span '%s': %w", spanName, err)
	}

	// Wrap the span to implement SpanHandle interface
	return &spanHandleAdapter{span: span}, nil
}

// EndSpan completes a span successfully.
func (t *TraceLoggerAdapter) EndSpan(ctx context.Context, span orchestrationpkg.SpanHandle, metadata map[string]interface{}) error {
	if t == nil || t.logger == nil {
		return fmt.Errorf("TraceLoggerAdapter: trace logger not initialized")
	}

	adapter, ok := span.(*spanHandleAdapter)
	if !ok {
		return fmt.Errorf("TraceLoggerAdapter: invalid span handle type (expected *spanHandleAdapter)")
	}

	if err := t.logger.EndSpan(ctx, adapter.span, metadata); err != nil {
		return fmt.Errorf("TraceLoggerAdapter: failed to end span: %w", err)
	}

	return nil
}

// FailSpan marks a span as failed.
func (t *TraceLoggerAdapter) FailSpan(ctx context.Context, span orchestrationpkg.SpanHandle, errorMsg string) error {
	if t == nil || t.logger == nil {
		return fmt.Errorf("TraceLoggerAdapter: trace logger not initialized")
	}

	adapter, ok := span.(*spanHandleAdapter)
	if !ok {
		return fmt.Errorf("TraceLoggerAdapter: invalid span handle type (expected *spanHandleAdapter)")
	}

	if err := t.logger.FailSpan(ctx, adapter.span, errorMsg); err != nil {
		return fmt.Errorf("TraceLoggerAdapter: failed to mark span as failed: %w", err)
	}

	return nil
}

// spanHandleAdapter wraps trace.Span to implement orchestration.SpanHandle.
type spanHandleAdapter struct {
	span *trace.Span
}

// AddMetadata adds metadata to the span.
func (s *spanHandleAdapter) AddMetadata(key string, value interface{}) {
	if s.span != nil {
		s.span.AddMetadata(key, value)
	}
}

// EventEmitterAdapter adapts event emission to orchestration.EventEmitterInterface.
type EventEmitterAdapter struct {
	eventChan chan orchestrationpkg.StreamEvent
}

// NewEventEmitterAdapter creates a new event emitter adapter.
func NewEventEmitterAdapter(eventChan chan orchestrationpkg.StreamEvent) *EventEmitterAdapter {
	if eventChan == nil {
		return nil
	}
	return &EventEmitterAdapter{eventChan: eventChan}
}

// Emit sends a stream event for monitoring/progress tracking.
func (e *EventEmitterAdapter) Emit(event orchestrationpkg.StreamEvent) {
	if e == nil || e.eventChan == nil {
		return
	}

	select {
	case e.eventChan <- event:
		// Event sent successfully
	default:
		// Channel full, drop event (non-blocking)
	}
}

// BudgetTrackerAdapter adapts services.BudgetTracker to orchestration.BudgetTrackerInterface.
type BudgetTrackerAdapter struct {
	budgetTracker *services.BudgetTracker
}

// NewBudgetTrackerAdapter creates a new budget tracker adapter.
func NewBudgetTrackerAdapter(budgetTracker *services.BudgetTracker) *BudgetTrackerAdapter {
	if budgetTracker == nil {
		return nil
	}
	return &BudgetTrackerAdapter{budgetTracker: budgetTracker}
}

// GetRemaining returns the remaining token budget for a user.
// If no budget tracker is configured, returns a default unlimited budget (1000000 tokens).
func (b *BudgetTrackerAdapter) GetRemaining(userID string) (int, error) {
	if b == nil || b.budgetTracker == nil {
		// If no budget tracker, return unlimited budget (configurable default)
		return 1000000, nil
	}

	remaining, err := b.budgetTracker.GetRemaining(userID)
	if err != nil {
		return 0, fmt.Errorf("BudgetTrackerAdapter: failed to get remaining budget for user '%s': %w", userID, err)
	}

	return remaining, nil
}
