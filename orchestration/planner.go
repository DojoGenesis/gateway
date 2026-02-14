package orchestration

import "context"

// PlannerInterface defines the contract for task decomposition and plan generation.
// Implementations must be able to decompose high-level user tasks into executable DAGs
// (Directed Acyclic Graphs) of tool calls and regenerate plans when execution failures occur.
//
// The planner is responsible for:
//   - Analyzing user intent and breaking it down into discrete, executable steps
//   - Identifying dependencies between steps to enable parallel execution
//   - Selecting appropriate tools from the registry for each step
//   - Generating valid DAGs (no cycles, valid tool names, proper dependency ordering)
//   - Adapting to failures by creating corrective plans that preserve completed work
type PlannerInterface interface {
	// GeneratePlan decomposes a high-level Task into an executable Plan (DAG of tool calls).
	GeneratePlan(ctx context.Context, task *Task) (*Plan, error)

	// RegeneratePlan creates a corrective plan after execution failures.
	// The new plan should preserve results from successfully completed nodes
	// and avoid repeating the same failing approach.
	RegeneratePlan(ctx context.Context, task *Task, failedPlan *Plan, errorContext string) (*Plan, error)
}

// ToolInvokerInterface defines the contract for executing tool calls.
// Consumers provide their own tool execution logic.
type ToolInvokerInterface interface {
	// InvokeTool executes a named tool with the given parameters.
	InvokeTool(ctx context.Context, toolName string, parameters map[string]interface{}) (map[string]interface{}, error)
}

// TraceLoggerInterface defines optional tracing for orchestration spans.
type TraceLoggerInterface interface {
	// StartSpan begins a new trace span.
	StartSpan(ctx context.Context, traceID, spanName string, metadata map[string]interface{}) (SpanHandle, error)
	// EndSpan completes a span successfully.
	EndSpan(ctx context.Context, span SpanHandle, metadata map[string]interface{}) error
	// FailSpan marks a span as failed.
	FailSpan(ctx context.Context, span SpanHandle, errorMsg string) error
}

// SpanHandle is an opaque handle to a trace span.
type SpanHandle interface {
	// AddMetadata adds metadata to the span.
	AddMetadata(key string, value interface{})
}

// EventEmitterInterface defines optional event emission for orchestration progress.
type EventEmitterInterface interface {
	// Emit sends a stream event for monitoring/progress tracking.
	Emit(event StreamEvent)
}

// StreamEvent represents a structured event emitted during orchestration.
type StreamEvent struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp interface{}            `json:"timestamp"`
}

// BudgetTrackerInterface defines optional budget enforcement.
type BudgetTrackerInterface interface {
	// GetRemaining returns the remaining token budget for a user.
	GetRemaining(userID string) (int, error)
}
