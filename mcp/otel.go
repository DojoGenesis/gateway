package mcp

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// SpanNameToolCall is the name used for OTEL spans created during MCP tool calls.
	// Per spec: "mcp.tool.invoke"
	SpanNameToolCall = "mcp.tool.invoke"

	// Attribute keys for MCP tool call spans (per gateway-mcp-contract.md spec)
	attrServerID          = "mcp.server_id"
	attrServerDisplayName = "mcp.server_display_name"
	attrToolName          = "mcp.tool_name"
	attrToolNamespaced    = "mcp.tool_namespaced"
	attrToolLatencyMs     = "mcp.tool_latency_ms"
	attrToolSuccess       = "mcp.tool_success"
	attrToolError         = "mcp.tool_error"
	attrInputSize         = "mcp.input_size"
	attrOutputSize        = "mcp.output_size"
	attrRetryCount        = "mcp.retry_count"
)

var (
	// Global tracer for the MCP package. Can be nil if OTEL is not configured.
	mcpTracer trace.Tracer
)

// InitTracer initializes the global MCP tracer for OpenTelemetry span emission.
// If tracer is nil, OTEL spans will not be emitted (graceful degradation).
func InitTracer(tracer trace.Tracer) {
	mcpTracer = tracer
}

// GetTracer returns the global MCP tracer. Returns the default noop tracer if not initialized.
func GetTracer() trace.Tracer {
	if mcpTracer != nil {
		return mcpTracer
	}
	return otel.Tracer("mcp")
}

// StartToolCallSpan creates a new OTEL span for an MCP tool call.
// Returns the span and a modified context. If OTEL is not configured, returns a noop span.
// Per spec: includes server_id, server_display_name, tool_name, and tool_namespaced attributes.
func StartToolCallSpan(ctx context.Context, serverID, serverDisplayName, toolName, toolNamespaced string) (context.Context, trace.Span) {
	tracer := GetTracer()
	ctx, span := tracer.Start(ctx, SpanNameToolCall,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String(attrServerID, serverID),
			attribute.String(attrServerDisplayName, serverDisplayName),
			attribute.String(attrToolName, toolName),
			attribute.String(attrToolNamespaced, toolNamespaced),
		),
	)
	return ctx, span
}

// RecordToolCallSuccess records successful completion of a tool call on the span.
// Adds latency, success=true, input size, and output size attributes.
// Per spec: includes mcp.tool_success (bool) and mcp.tool_latency_ms.
func RecordToolCallSuccess(span trace.Span, latency time.Duration, inputSize, outputSize int) {
	if span == nil {
		return
	}

	span.SetAttributes(
		attribute.Bool(attrToolSuccess, true),
		attribute.Int64(attrToolLatencyMs, latency.Milliseconds()),
		attribute.Int(attrInputSize, inputSize),
		attribute.Int(attrOutputSize, outputSize),
	)
}

// RecordToolCallError records an error during a tool call on the span.
// Adds error message, success=false, and marks the span as failed.
// Per spec: includes mcp.tool_success (bool) and mcp.tool_error.
func RecordToolCallError(span trace.Span, err error, retryCount int) {
	if span == nil || err == nil {
		return
	}

	span.SetAttributes(
		attribute.Bool(attrToolSuccess, false),
		attribute.String(attrToolError, err.Error()),
		attribute.Int(attrRetryCount, retryCount),
	)
	span.RecordError(err)
}

// FinishSpan ends the span. Safe to call with nil span.
func FinishSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}
