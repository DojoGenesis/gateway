package trace

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/events"
	_ "modernc.org/sqlite"
)

func setupLoggerTestDB(t *testing.T) (*TraceStorage, *sql.DB, string) {
	tmpfile := "/tmp/test_logger_" + time.Now().Format("20060102150405") + ".db"

	db, err := sql.Open("sqlite", tmpfile)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	ts, err := NewTraceStorage(db)
	if err != nil {
		t.Fatalf("failed to create trace storage: %v", err)
	}

	return ts, db, tmpfile
}

func teardownLoggerTestDB(t *testing.T, db *sql.DB, dbPath string) {
	if err := db.Close(); err != nil {
		t.Errorf("failed to close database: %v", err)
	}
	if err := os.Remove(dbPath); err != nil {
		t.Errorf("failed to remove test database: %v", err)
	}
}

func TestTraceLogger_StartTrace(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	if traceID == "" {
		t.Errorf("expected trace ID to be generated")
	}

	trace, err := logger.GetTrace(ctx, traceID)
	if err != nil {
		t.Fatalf("failed to get trace: %v", err)
	}

	if trace.SessionID != "session-123" {
		t.Errorf("expected session_id 'session-123', got '%s'", trace.SessionID)
	}

	if trace.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", trace.Status)
	}
}

func TestTraceLogger_EndTrace(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	if err := logger.EndTrace(ctx, traceID, "completed"); err != nil {
		t.Fatalf("failed to end trace: %v", err)
	}

	trace, err := logger.GetTrace(ctx, traceID)
	if err != nil {
		t.Fatalf("failed to get trace: %v", err)
	}

	if trace.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", trace.Status)
	}

	if trace.EndTime == nil {
		t.Errorf("expected end_time to be set")
	}
}

func TestTraceLogger_StartSpan(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	inputs := map[string]interface{}{
		"query": "test",
	}

	span, err := logger.StartSpan(ctx, traceID, "test_operation", inputs)
	if err != nil {
		t.Fatalf("failed to start span: %v", err)
	}

	if span.SpanID == "" {
		t.Errorf("expected span ID to be generated")
	}

	if span.TraceID != traceID {
		t.Errorf("expected trace_id '%s', got '%s'", traceID, span.TraceID)
	}

	if span.Name != "test_operation" {
		t.Errorf("expected name 'test_operation', got '%s'", span.Name)
	}

	if span.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", span.Status)
	}

	activeSpans := logger.GetActiveSpans()
	if len(activeSpans) != 1 {
		t.Errorf("expected 1 active span, got %d", len(activeSpans))
	}
}

func TestTraceLogger_EndSpan(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	span, err := logger.StartSpan(ctx, traceID, "test_operation", nil)
	if err != nil {
		t.Fatalf("failed to start span: %v", err)
	}

	outputs := map[string]interface{}{
		"result": "success",
	}

	if err := logger.EndSpan(ctx, span, outputs); err != nil {
		t.Fatalf("failed to end span: %v", err)
	}

	if span.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", span.Status)
	}

	if span.EndTime == nil {
		t.Errorf("expected end_time to be set")
	}

	if span.Outputs["result"] != "success" {
		t.Errorf("expected output result 'success', got '%v'", span.Outputs["result"])
	}

	activeSpans := logger.GetActiveSpans()
	if len(activeSpans) != 0 {
		t.Errorf("expected 0 active spans after ending, got %d", len(activeSpans))
	}
}

func TestTraceLogger_FailSpan(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	span, err := logger.StartSpan(ctx, traceID, "test_operation", nil)
	if err != nil {
		t.Fatalf("failed to start span: %v", err)
	}

	if err := logger.FailSpan(ctx, span, "test error"); err != nil {
		t.Fatalf("failed to fail span: %v", err)
	}

	if span.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", span.Status)
	}

	if span.Metadata["error"] != "test error" {
		t.Errorf("expected error metadata 'test error', got '%v'", span.Metadata["error"])
	}

	activeSpans := logger.GetActiveSpans()
	if len(activeSpans) != 0 {
		t.Errorf("expected 0 active spans after failing, got %d", len(activeSpans))
	}
}

func TestTraceLogger_NestedSpans(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	rootSpan, err := logger.StartSpan(ctx, traceID, "root", nil)
	if err != nil {
		t.Fatalf("failed to start root span: %v", err)
	}

	ctxWithRoot := WithSpan(ctx, rootSpan)

	childSpan, err := logger.StartSpan(ctxWithRoot, traceID, "child", nil)
	if err != nil {
		t.Fatalf("failed to start child span: %v", err)
	}

	if childSpan.ParentID != rootSpan.SpanID {
		t.Errorf("expected child parent_id '%s', got '%s'", rootSpan.SpanID, childSpan.ParentID)
	}

	ctxWithChild := WithSpan(ctxWithRoot, childSpan)

	grandchildSpan, err := logger.StartSpan(ctxWithChild, traceID, "grandchild", nil)
	if err != nil {
		t.Fatalf("failed to start grandchild span: %v", err)
	}

	if grandchildSpan.ParentID != childSpan.SpanID {
		t.Errorf("expected grandchild parent_id '%s', got '%s'", childSpan.SpanID, grandchildSpan.ParentID)
	}

	activeSpans := logger.GetActiveSpans()
	if len(activeSpans) != 3 {
		t.Errorf("expected 3 active spans, got %d", len(activeSpans))
	}

	if err := logger.EndSpan(ctx, grandchildSpan, nil); err != nil {
		t.Fatalf("failed to end grandchild span: %v", err)
	}

	if err := logger.EndSpan(ctx, childSpan, nil); err != nil {
		t.Fatalf("failed to end child span: %v", err)
	}

	if err := logger.EndSpan(ctx, rootSpan, nil); err != nil {
		t.Fatalf("failed to end root span: %v", err)
	}

	activeSpans = logger.GetActiveSpans()
	if len(activeSpans) != 0 {
		t.Errorf("expected 0 active spans after ending all, got %d", len(activeSpans))
	}
}

func TestTraceLogger_GetTraceSpans(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	span1, err := logger.StartSpan(ctx, traceID, "operation1", nil)
	if err != nil {
		t.Fatalf("failed to start span1: %v", err)
	}

	span2, err := logger.StartSpan(ctx, traceID, "operation2", nil)
	if err != nil {
		t.Fatalf("failed to start span2: %v", err)
	}

	logger.EndSpan(ctx, span1, nil)
	logger.EndSpan(ctx, span2, nil)

	spans, err := logger.GetTraceSpans(ctx, traceID)
	if err != nil {
		t.Fatalf("failed to get trace spans: %v", err)
	}

	if len(spans) != 2 {
		t.Errorf("expected 2 spans, got %d", len(spans))
	}
}

func TestTraceLogger_WithEvents(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	eventChan := make(chan events.StreamEvent, 10)
	logger := NewTraceLogger(ts, eventChan)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	span, err := logger.StartSpan(ctx, traceID, "test_operation", map[string]interface{}{
		"input": "test",
	})
	if err != nil {
		t.Fatalf("failed to start span: %v", err)
	}

	select {
	case event := <-eventChan:
		if event.Type != "trace_span_start" {
			t.Errorf("expected event type 'trace_span_start', got '%s'", event.Type)
		}
		if event.Data["span_id"] != span.SpanID {
			t.Errorf("expected span_id in event to match")
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("expected to receive trace_span_start event")
	}

	if err := logger.EndSpan(ctx, span, map[string]interface{}{
		"output": "result",
	}); err != nil {
		t.Fatalf("failed to end span: %v", err)
	}

	select {
	case event := <-eventChan:
		if event.Type != "trace_span_end" {
			t.Errorf("expected event type 'trace_span_end', got '%s'", event.Type)
		}
		if event.Data["span_id"] != span.SpanID {
			t.Errorf("expected span_id in event to match")
		}
		if event.Data["status"] != "completed" {
			t.Errorf("expected status 'completed' in event, got '%v'", event.Data["status"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("expected to receive trace_span_end event")
	}
}

func TestTraceLogger_EndNilSpan(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	err := logger.EndSpan(ctx, nil, nil)
	if err == nil {
		t.Errorf("expected error when ending nil span")
	}
}

func TestTraceLogger_FailNilSpan(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	err := logger.FailSpan(ctx, nil, "error")
	if err == nil {
		t.Errorf("expected error when failing nil span")
	}
}

func TestTraceLogger_MultipleActiveSpans(t *testing.T) {
	ts, db, dbPath := setupLoggerTestDB(t)
	defer teardownLoggerTestDB(t, db, dbPath)

	logger := NewTraceLoggerWithoutEvents(ts)
	ctx := context.Background()

	traceID, err := logger.StartTrace(ctx, "session-123")
	if err != nil {
		t.Fatalf("failed to start trace: %v", err)
	}

	spans := make([]*Span, 5)
	for i := 0; i < 5; i++ {
		span, err := logger.StartSpan(ctx, traceID, "operation", nil)
		if err != nil {
			t.Fatalf("failed to start span %d: %v", i, err)
		}
		spans[i] = span
	}

	activeSpans := logger.GetActiveSpans()
	if len(activeSpans) != 5 {
		t.Errorf("expected 5 active spans, got %d", len(activeSpans))
	}

	for i := 0; i < 3; i++ {
		logger.EndSpan(ctx, spans[i], nil)
	}

	activeSpans = logger.GetActiveSpans()
	if len(activeSpans) != 2 {
		t.Errorf("expected 2 active spans after ending 3, got %d", len(activeSpans))
	}
}
