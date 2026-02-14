package trace

import (
	"context"
	"testing"
	"time"
)

func TestSpanBuilder_NewSpan(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")

	inputs := map[string]interface{}{
		"query": "test query",
	}

	span := sb.NewSpan("test_operation", inputs)

	if span.SpanID == "" {
		t.Errorf("expected span ID to be generated")
	}

	if span.TraceID != "trace-123" {
		t.Errorf("expected trace_id 'trace-123', got '%s'", span.TraceID)
	}

	if span.Name != "test_operation" {
		t.Errorf("expected name 'test_operation', got '%s'", span.Name)
	}

	if span.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", span.Status)
	}

	if span.Inputs["query"] != "test query" {
		t.Errorf("expected input query 'test query', got '%v'", span.Inputs["query"])
	}

	if span.StartTime.IsZero() {
		t.Errorf("expected start_time to be set")
	}
}

func TestSpanBuilder_NewChildSpan(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")

	parent := sb.NewSpan("parent_operation", nil)
	child := sb.NewChildSpan(parent, "child_operation", nil)

	if child.ParentID != parent.SpanID {
		t.Errorf("expected child parent_id '%s', got '%s'", parent.SpanID, child.ParentID)
	}

	if child.TraceID != parent.TraceID {
		t.Errorf("expected child and parent to share trace_id")
	}

	grandchild := sb.NewChildSpan(child, "grandchild_operation", nil)

	if grandchild.ParentID != child.SpanID {
		t.Errorf("expected grandchild parent_id '%s', got '%s'", child.SpanID, grandchild.ParentID)
	}
}

func TestSpanBuilder_NewChildSpan_NilParent(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")

	child := sb.NewChildSpan(nil, "orphan_operation", nil)

	if child.ParentID != "" {
		t.Errorf("expected empty parent_id for nil parent, got '%s'", child.ParentID)
	}
}

func TestSpan_Complete(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")
	span := sb.NewSpan("test_operation", nil)

	time.Sleep(10 * time.Millisecond)

	outputs := map[string]interface{}{
		"result": "success",
	}

	span.Complete(outputs)

	if span.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", span.Status)
	}

	if span.EndTime == nil {
		t.Errorf("expected end_time to be set")
	}

	if span.Outputs["result"] != "success" {
		t.Errorf("expected output result 'success', got '%v'", span.Outputs["result"])
	}

	if !span.IsCompleted() {
		t.Errorf("expected IsCompleted to return true")
	}
}

func TestSpan_Fail(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")
	span := sb.NewSpan("test_operation", nil)

	span.Fail("test error message")

	if span.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", span.Status)
	}

	if span.EndTime == nil {
		t.Errorf("expected end_time to be set")
	}

	if span.Metadata["error"] != "test error message" {
		t.Errorf("expected metadata error 'test error message', got '%v'", span.Metadata["error"])
	}

	if !span.IsCompleted() {
		t.Errorf("expected IsCompleted to return true")
	}
}

func TestSpan_AddMetadata(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")
	span := sb.NewSpan("test_operation", nil)

	span.AddMetadata("key1", "value1")
	span.AddMetadata("key2", 123)

	if span.Metadata["key1"] != "value1" {
		t.Errorf("expected metadata key1 'value1', got '%v'", span.Metadata["key1"])
	}

	if span.Metadata["key2"] != 123 {
		t.Errorf("expected metadata key2 123, got '%v'", span.Metadata["key2"])
	}
}

func TestSpan_Duration(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")
	span := sb.NewSpan("test_operation", nil)

	time.Sleep(50 * time.Millisecond)

	duration := span.Duration()
	if duration < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", duration)
	}

	span.Complete(nil)

	finalDuration := span.Duration()
	if finalDuration < 50*time.Millisecond {
		t.Errorf("expected final duration >= 50ms, got %v", finalDuration)
	}
}

func TestSpan_Clone(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")

	original := sb.NewSpan("test_operation", map[string]interface{}{
		"input": "test",
	})
	original.AddMetadata("meta", "data")
	original.Complete(map[string]interface{}{
		"output": "result",
	})

	clone := original.Clone()

	if clone.SpanID != original.SpanID {
		t.Errorf("expected cloned span_id to match")
	}

	if clone.Name != original.Name {
		t.Errorf("expected cloned name to match")
	}

	if clone.Status != original.Status {
		t.Errorf("expected cloned status to match")
	}

	clone.Metadata["meta"] = "modified"
	if original.Metadata["meta"] != "data" {
		t.Errorf("modifying clone should not affect original metadata")
	}

	clone.Inputs["input"] = "modified"
	if original.Inputs["input"] != "test" {
		t.Errorf("modifying clone should not affect original inputs")
	}
}

func TestWithSpan_CurrentSpan(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")
	span := sb.NewSpan("test_operation", nil)

	ctx := context.Background()

	if CurrentSpan(ctx) != nil {
		t.Errorf("expected nil span from empty context")
	}

	ctx = WithSpan(ctx, span)

	retrieved := CurrentSpan(ctx)
	if retrieved == nil {
		t.Errorf("expected span to be retrieved from context")
	}

	if retrieved.SpanID != span.SpanID {
		t.Errorf("expected retrieved span to match original")
	}
}

func TestSpan_NestedHierarchy(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")

	root := sb.NewSpan("root", nil)
	child1 := sb.NewChildSpan(root, "child1", nil)
	child2 := sb.NewChildSpan(root, "child2", nil)
	grandchild := sb.NewChildSpan(child1, "grandchild", nil)

	if child1.ParentID != root.SpanID {
		t.Errorf("child1 should have root as parent")
	}

	if child2.ParentID != root.SpanID {
		t.Errorf("child2 should have root as parent")
	}

	if grandchild.ParentID != child1.SpanID {
		t.Errorf("grandchild should have child1 as parent")
	}

	if child1.TraceID != root.TraceID || child2.TraceID != root.TraceID || grandchild.TraceID != root.TraceID {
		t.Errorf("all spans should share the same trace_id")
	}
}

func TestSpan_MultipleComplete(t *testing.T) {
	sb := NewSpanBuilder("trace-123", "session-456")
	span := sb.NewSpan("test_operation", nil)

	firstOutputs := map[string]interface{}{"result": "first"}
	span.Complete(firstOutputs)

	firstEndTime := *span.EndTime

	time.Sleep(10 * time.Millisecond)

	secondOutputs := map[string]interface{}{"result": "second"}
	span.Complete(secondOutputs)

	if span.EndTime.Equal(firstEndTime) {
		t.Errorf("expected end_time to be updated on second Complete call")
	}

	if span.Outputs["result"] != "second" {
		t.Errorf("expected outputs to be overwritten with second call")
	}
}
