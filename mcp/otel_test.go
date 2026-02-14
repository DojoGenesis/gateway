package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestInitTracer(t *testing.T) {
	// Save original tracer
	originalTracer := mcpTracer
	defer func() { mcpTracer = originalTracer }()

	// Test with nil tracer
	InitTracer(nil)
	if mcpTracer != nil {
		t.Error("InitTracer(nil) should set mcpTracer to nil")
	}

	// Test with noop tracer
	noopTracer := noop.NewTracerProvider().Tracer("test")
	InitTracer(noopTracer)
	if mcpTracer != noopTracer {
		t.Error("InitTracer() should set mcpTracer")
	}
}

func TestGetTracer(t *testing.T) {
	// Save original tracer
	originalTracer := mcpTracer
	defer func() { mcpTracer = originalTracer }()

	// Test when tracer is not initialized
	mcpTracer = nil
	tracer := GetTracer()
	if tracer == nil {
		t.Error("GetTracer() should return default tracer when not initialized")
	}

	// Test when tracer is initialized
	noopTracer := noop.NewTracerProvider().Tracer("test")
	mcpTracer = noopTracer
	tracer = GetTracer()
	if tracer != noopTracer {
		t.Error("GetTracer() should return initialized tracer")
	}
}

func TestStartToolCallSpan(t *testing.T) {
	ctx := context.Background()
	serverID := "test-server"
	serverDisplayName := "Test Server"
	toolName := "test-tool"
	toolNamespaced := "test-server:test-tool"

	ctx, span := StartToolCallSpan(ctx, serverID, serverDisplayName, toolName, toolNamespaced)
	if ctx == nil {
		t.Error("StartToolCallSpan() returned nil context")
	}
	if span == nil {
		t.Error("StartToolCallSpan() returned nil span")
	}

	// Clean up
	FinishSpan(span)
}

func TestRecordToolCallSuccess(t *testing.T) {
	ctx := context.Background()
	_, span := StartToolCallSpan(ctx, "test", "Test Server", "tool", "test:tool")
	defer FinishSpan(span)

	// Test recording success
	latency := 100 * time.Millisecond
	RecordToolCallSuccess(span, latency, 50, 100)

	// Test with nil span (should not panic)
	RecordToolCallSuccess(nil, latency, 50, 100)
}

func TestRecordToolCallError(t *testing.T) {
	ctx := context.Background()
	_, span := StartToolCallSpan(ctx, "test", "Test Server", "tool", "test:tool")
	defer FinishSpan(span)

	// Test recording error
	err := errors.New("test error")
	RecordToolCallError(span, err, 2)

	// Test with nil span (should not panic)
	RecordToolCallError(nil, err, 2)

	// Test with nil error (should not panic)
	RecordToolCallError(span, nil, 0)
}

func TestFinishSpan(t *testing.T) {
	ctx := context.Background()
	_, span := StartToolCallSpan(ctx, "test", "Test Server", "tool", "test:tool")

	// Should not panic
	FinishSpan(span)

	// Test with nil span (should not panic)
	FinishSpan(nil)
}

func TestSpanAttributeConstants(t *testing.T) {
	// Verify constant values match spec requirements (per gateway-mcp-contract.md)
	if attrServerID != "mcp.server_id" {
		t.Errorf("attrServerID = %v, want 'mcp.server_id'", attrServerID)
	}
	if attrServerDisplayName != "mcp.server_display_name" {
		t.Errorf("attrServerDisplayName = %v, want 'mcp.server_display_name'", attrServerDisplayName)
	}
	if attrToolName != "mcp.tool_name" {
		t.Errorf("attrToolName = %v, want 'mcp.tool_name'", attrToolName)
	}
	if attrToolNamespaced != "mcp.tool_namespaced" {
		t.Errorf("attrToolNamespaced = %v, want 'mcp.tool_namespaced'", attrToolNamespaced)
	}
	if attrToolLatencyMs != "mcp.tool_latency_ms" {
		t.Errorf("attrToolLatencyMs = %v, want 'mcp.tool_latency_ms'", attrToolLatencyMs)
	}
	if attrToolSuccess != "mcp.tool_success" {
		t.Errorf("attrToolSuccess = %v, want 'mcp.tool_success'", attrToolSuccess)
	}
	if attrToolError != "mcp.tool_error" {
		t.Errorf("attrToolError = %v, want 'mcp.tool_error'", attrToolError)
	}
}
