package skill

import "context"

// ToolInvoker is a local interface to avoid circular dependency with orchestration package.
// It matches orchestration.ToolInvokerInterface but is defined here to break the import cycle.
type ToolInvoker interface {
	// InvokeTool executes a named tool with the given parameters.
	InvokeTool(ctx context.Context, toolName string, parameters map[string]interface{}) (map[string]interface{}, error)
}

// TraceLogger is a local interface to avoid circular dependency with orchestration package.
// It matches orchestration.TraceLoggerInterface but is defined here to break the import cycle.
type TraceLogger interface {
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
