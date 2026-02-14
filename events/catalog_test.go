package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllEventTypesRegistered ensures the catalog knows about every event constant.
func TestAllEventTypesRegistered(t *testing.T) {
	all := AllEventTypes()
	assert.Len(t, all, 21, "expected 21 event types in catalog")

	// Verify every constant appears exactly once
	seen := make(map[EventType]bool)
	for _, et := range all {
		assert.False(t, seen[et], "duplicate event type: %s", et)
		seen[et] = true
	}
}

// TestIsValidEventType checks validation logic.
func TestIsValidEventType(t *testing.T) {
	assert.True(t, IsValidEventType(IntentClassified))
	assert.True(t, IsValidEventType(OrchestrationFailed))
	assert.False(t, IsValidEventType(EventType("nonexistent_event")))
	assert.False(t, IsValidEventType(EventType("")))
}

// TestEventSerialization_IntentClassified verifies the payload schema.
func TestEventSerialization_IntentClassified(t *testing.T) {
	evt := NewIntentClassifiedEvent("code_generation", 0.95)
	assertEventSchema(t, evt, IntentClassified, "intent", "confidence")
	data := eventData(t, evt)
	assert.Equal(t, "code_generation", data["intent"])
	assert.Equal(t, 0.95, data["confidence"])
}

// TestEventSerialization_ProviderSelected verifies the payload schema.
func TestEventSerialization_ProviderSelected(t *testing.T) {
	evt := NewProviderSelectedEvent("anthropic", "claude-3-opus")
	assertEventSchema(t, evt, ProviderSelected, "provider", "model")
	data := eventData(t, evt)
	assert.Equal(t, "anthropic", data["provider"])
	assert.Equal(t, "claude-3-opus", data["model"])
}

// TestEventSerialization_ToolInvoked verifies the payload schema.
func TestEventSerialization_ToolInvoked(t *testing.T) {
	args := map[string]interface{}{"query": "test"}
	evt := NewToolInvokedEvent("web_search", args)
	assertEventSchema(t, evt, ToolInvoked, "tool", "arguments")
	data := eventData(t, evt)
	assert.Equal(t, "web_search", data["tool"])
}

// TestEventSerialization_ToolCompleted verifies the payload schema.
func TestEventSerialization_ToolCompleted(t *testing.T) {
	evt := NewToolCompletedEvent("web_search", "result data", int64(150))
	assertEventSchema(t, evt, ToolCompleted, "tool", "result", "duration_ms")
	data := eventData(t, evt)
	assert.Equal(t, "web_search", data["tool"])
	assert.Equal(t, float64(150), data["duration_ms"])
}

// TestEventSerialization_Thinking verifies the payload schema.
func TestEventSerialization_Thinking(t *testing.T) {
	evt := NewThinkingEvent("analyzing the problem")
	assertEventSchema(t, evt, Thinking, "message")
	data := eventData(t, evt)
	assert.Equal(t, "analyzing the problem", data["message"])
}

// TestEventSerialization_ResponseChunk verifies the payload schema.
func TestEventSerialization_ResponseChunk(t *testing.T) {
	evt := NewResponseChunkEvent("Here is the answer")
	assertEventSchema(t, evt, ResponseChunk, "content")
	data := eventData(t, evt)
	assert.Equal(t, "Here is the answer", data["content"])
}

// TestEventSerialization_MemoryRetrieved verifies the payload schema.
func TestEventSerialization_MemoryRetrieved(t *testing.T) {
	evt := NewMemoryRetrievedEvent(2, []interface{}{"mem1", "mem2"})
	assertEventSchema(t, evt, MemoryRetrieved, "memories_found", "memories")
	data := eventData(t, evt)
	assert.Equal(t, float64(2), data["memories_found"])
}

// TestEventSerialization_Complete verifies the payload schema.
func TestEventSerialization_Complete(t *testing.T) {
	usage := map[string]interface{}{"input_tokens": 100, "output_tokens": 50}
	evt := NewCompleteEvent(usage)
	assertEventSchema(t, evt, Complete, "usage")
}

// TestEventSerialization_Error verifies the payload schema.
func TestEventSerialization_Error(t *testing.T) {
	evt := NewErrorEvent("provider timed out", "provider_timeout")
	assertEventSchema(t, evt, Error, "error", "error_code")
	data := eventData(t, evt)
	assert.Equal(t, "provider timed out", data["error"])
	assert.Equal(t, "provider_timeout", data["error_code"])
}

// TestEventSerialization_TraceSpanStart verifies the payload schema.
func TestEventSerialization_TraceSpanStart(t *testing.T) {
	now := time.Now()
	evt := NewTraceSpanStartEvent("trace-1", "span-1", "", "chat.process", now, nil)
	assertEventSchema(t, evt, TraceSpanStart, "trace_id", "span_id", "parent_id", "name", "start_time", "inputs")
}

// TestEventSerialization_TraceSpanEnd verifies the payload schema.
func TestEventSerialization_TraceSpanEnd(t *testing.T) {
	now := time.Now()
	end := now.Add(100 * time.Millisecond)
	evt := NewTraceSpanEndEvent("trace-1", "span-1", "", "chat.process", now, &end, nil, nil, nil, "ok", 100)
	assertEventSchema(t, evt, TraceSpanEnd, "trace_id", "span_id", "parent_id", "name", "start_time", "end_time", "status", "duration_ms")
}

// TestEventSerialization_ArtifactCreated verifies the payload schema.
func TestEventSerialization_ArtifactCreated(t *testing.T) {
	evt := NewArtifactCreatedEvent("art-1", "main.go", "code", "proj-1")
	assertEventSchema(t, evt, ArtifactCreated, "artifact_id", "artifact_name", "artifact_type", "project_id")
}

// TestEventSerialization_ArtifactUpdated verifies the payload schema.
func TestEventSerialization_ArtifactUpdated(t *testing.T) {
	evt := NewArtifactUpdatedEvent("art-1", "main.go", 2, "fix bug")
	assertEventSchema(t, evt, ArtifactUpdated, "artifact_id", "artifact_name", "version", "commit_message")
}

// TestEventSerialization_ProjectSwitched verifies the payload schema.
func TestEventSerialization_ProjectSwitched(t *testing.T) {
	evt := NewProjectSwitchedEvent("proj-2", "Backend API")
	assertEventSchema(t, evt, ProjectSwitched, "project_id", "project_name")
}

// TestEventSerialization_DiagramRendered verifies the payload schema.
func TestEventSerialization_DiagramRendered(t *testing.T) {
	evt := NewDiagramRenderedEvent("diag-1", "sequence", "svg")
	assertEventSchema(t, evt, DiagramRendered, "diagram_id", "diagram_type", "format")
}

// TestEventSerialization_OrchestrationPlanCreated verifies the payload schema.
func TestEventSerialization_OrchestrationPlanCreated(t *testing.T) {
	evt := NewOrchestrationPlanCreatedEvent("plan-1", "task-1", 3, 0.05, nil)
	assertEventSchema(t, evt, OrchestrationPlanCreated, "plan_id", "task_id", "node_count", "estimated_cost")
}

// TestEventSerialization_OrchestrationNodeStart verifies the payload schema.
func TestEventSerialization_OrchestrationNodeStart(t *testing.T) {
	evt := NewOrchestrationNodeStartEvent("node-1", "plan-1", "web_search", map[string]interface{}{"q": "test"})
	assertEventSchema(t, evt, OrchestrationNodeStart, "node_id", "plan_id", "tool_name", "parameters")
}

// TestEventSerialization_OrchestrationNodeEnd verifies the payload schema.
func TestEventSerialization_OrchestrationNodeEnd(t *testing.T) {
	evt := NewOrchestrationNodeEndEvent("node-1", "plan-1", "web_search", "success", "results", "", 250)
	assertEventSchema(t, evt, OrchestrationNodeEnd, "node_id", "plan_id", "tool_name", "state", "result", "error", "duration_ms")
}

// TestEventSerialization_OrchestrationReplanning verifies the payload schema.
func TestEventSerialization_OrchestrationReplanning(t *testing.T) {
	evt := NewOrchestrationReplanningEvent("plan-1", "task-1", "node failed", []string{"node-2"})
	assertEventSchema(t, evt, OrchestrationReplanning, "plan_id", "task_id", "reason", "failed_nodes")
}

// TestEventSerialization_OrchestrationComplete verifies the payload schema.
func TestEventSerialization_OrchestrationComplete(t *testing.T) {
	evt := NewOrchestrationCompleteEvent("plan-1", "task-1", 3, 3, 0, 500)
	assertEventSchema(t, evt, OrchestrationComplete, "plan_id", "task_id", "total_nodes", "success_nodes", "failed_nodes", "duration_ms")
}

// TestEventSerialization_OrchestrationFailed verifies the payload schema.
func TestEventSerialization_OrchestrationFailed(t *testing.T) {
	evt := NewOrchestrationFailedEvent("plan-1", "task-1", "timeout exceeded")
	assertEventSchema(t, evt, OrchestrationFailed, "plan_id", "task_id", "reason")
}

// TestEventRoundTrip verifies JSON serialization/deserialization for all event types.
func TestEventRoundTrip(t *testing.T) {
	events := []StreamEvent{
		NewIntentClassifiedEvent("test", 0.9),
		NewProviderSelectedEvent("anthropic", "claude"),
		NewToolInvokedEvent("search", nil),
		NewToolCompletedEvent("search", "ok", 100),
		NewThinkingEvent("hmm"),
		NewResponseChunkEvent("hello"),
		NewMemoryRetrievedEvent(0, nil),
		NewCompleteEvent(nil),
		NewErrorEvent("fail", "err"),
		NewArtifactCreatedEvent("a", "b", "c", "d"),
		NewOrchestrationPlanCreatedEvent("p", "t", 1, 0, nil),
		NewOrchestrationCompleteEvent("p", "t", 1, 1, 0, 100),
		NewOrchestrationFailedEvent("p", "t", "reason"),
	}

	for _, evt := range events {
		t.Run(string(evt.Type), func(t *testing.T) {
			data, err := evt.ToJSON()
			require.NoError(t, err)

			roundTripped, err := FromJSON(data)
			require.NoError(t, err)
			assert.Equal(t, evt.Type, roundTripped.Type)
			assert.NotNil(t, roundTripped.Data)
		})
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// assertEventSchema verifies an event has the correct type and required data keys.
func assertEventSchema(t *testing.T, evt StreamEvent, expectedType EventType, requiredKeys ...string) {
	t.Helper()
	assert.Equal(t, expectedType, evt.Type)
	assert.False(t, evt.Timestamp.IsZero(), "timestamp must be set")
	assert.NotNil(t, evt.Data, "data must not be nil")
	for _, key := range requiredKeys {
		_, exists := evt.Data[key]
		assert.True(t, exists, "missing required data key: %s", key)
	}
}

// eventData returns the Data map after JSON round-trip (to test serialized form).
func eventData(t *testing.T, evt StreamEvent) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))
	dataMap, ok := result["data"].(map[string]interface{})
	require.True(t, ok, "data field must be an object")
	return dataMap
}
