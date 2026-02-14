package streaming

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  string
	}{
		{"IntentClassified", IntentClassified, "intent_classified"},
		{"ProviderSelected", ProviderSelected, "provider_selected"},
		{"ToolInvoked", ToolInvoked, "tool_invoked"},
		{"ToolCompleted", ToolCompleted, "tool_completed"},
		{"Thinking", Thinking, "thinking"},
		{"ResponseChunk", ResponseChunk, "response_chunk"},
		{"MemoryRetrieved", MemoryRetrieved, "memory_retrieved"},
		{"Complete", Complete, "complete"},
		{"Error", Error, "error"},
		{"TraceSpanStart", TraceSpanStart, "trace_span_start"},
		{"TraceSpanEnd", TraceSpanEnd, "trace_span_end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.eventType))
			}
		})
	}
}

func TestNewIntentClassifiedEvent(t *testing.T) {
	event := NewIntentClassifiedEvent("THINK", 0.85)

	if event.Type != IntentClassified {
		t.Errorf("Expected type %s, got %s", IntentClassified, event.Type)
	}

	if event.Data["intent"] != "THINK" {
		t.Errorf("Expected intent 'THINK', got %v", event.Data["intent"])
	}

	if event.Data["confidence"] != 0.85 {
		t.Errorf("Expected confidence 0.85, got %v", event.Data["confidence"])
	}

	if event.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestNewProviderSelectedEvent(t *testing.T) {
	event := NewProviderSelectedEvent("deepseek-api", "deepseek-chat")

	if event.Type != ProviderSelected {
		t.Errorf("Expected type %s, got %s", ProviderSelected, event.Type)
	}

	if event.Data["provider"] != "deepseek-api" {
		t.Errorf("Expected provider 'deepseek-api', got %v", event.Data["provider"])
	}

	if event.Data["model"] != "deepseek-chat" {
		t.Errorf("Expected model 'deepseek-chat', got %v", event.Data["model"])
	}
}

func TestNewToolInvokedEvent(t *testing.T) {
	args := map[string]interface{}{
		"file_path": "/tmp/test.txt",
		"encoding":  "utf-8",
	}
	event := NewToolInvokedEvent("read_file", args)

	if event.Type != ToolInvoked {
		t.Errorf("Expected type %s, got %s", ToolInvoked, event.Type)
	}

	if event.Data["tool"] != "read_file" {
		t.Errorf("Expected tool 'read_file', got %v", event.Data["tool"])
	}

	argsData, ok := event.Data["arguments"].(map[string]interface{})
	if !ok {
		t.Error("Arguments should be a map")
	}

	if argsData["file_path"] != "/tmp/test.txt" {
		t.Errorf("Expected file_path '/tmp/test.txt', got %v", argsData["file_path"])
	}
}

func TestNewToolCompletedEvent(t *testing.T) {
	result := map[string]interface{}{
		"success": true,
		"content": "file contents",
	}
	event := NewToolCompletedEvent("read_file", result, 150)

	if event.Type != ToolCompleted {
		t.Errorf("Expected type %s, got %s", ToolCompleted, event.Type)
	}

	if event.Data["tool"] != "read_file" {
		t.Errorf("Expected tool 'read_file', got %v", event.Data["tool"])
	}

	if event.Data["duration_ms"] != int64(150) {
		t.Errorf("Expected duration_ms 150, got %v", event.Data["duration_ms"])
	}
}

func TestNewThinkingEvent(t *testing.T) {
	event := NewThinkingEvent("Processing your request...")

	if event.Type != Thinking {
		t.Errorf("Expected type %s, got %s", Thinking, event.Type)
	}

	if event.Data["message"] != "Processing your request..." {
		t.Errorf("Expected message 'Processing your request...', got %v", event.Data["message"])
	}
}

func TestNewResponseChunkEvent(t *testing.T) {
	event := NewResponseChunkEvent("Hello, world!")

	if event.Type != ResponseChunk {
		t.Errorf("Expected type %s, got %s", ResponseChunk, event.Type)
	}

	if event.Data["content"] != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %v", event.Data["content"])
	}
}

func TestNewMemoryRetrievedEvent(t *testing.T) {
	memories := []interface{}{
		map[string]interface{}{"id": "1", "content": "memory 1"},
		map[string]interface{}{"id": "2", "content": "memory 2"},
	}
	event := NewMemoryRetrievedEvent(2, memories)

	if event.Type != MemoryRetrieved {
		t.Errorf("Expected type %s, got %s", MemoryRetrieved, event.Type)
	}

	if event.Data["memories_found"] != 2 {
		t.Errorf("Expected memories_found 2, got %v", event.Data["memories_found"])
	}

	memoriesData, ok := event.Data["memories"].([]interface{})
	if !ok {
		t.Error("Memories should be a slice")
	}

	if len(memoriesData) != 2 {
		t.Errorf("Expected 2 memories, got %d", len(memoriesData))
	}
}

func TestNewCompleteEvent(t *testing.T) {
	usage := map[string]interface{}{
		"prompt_tokens":     100,
		"completion_tokens": 200,
		"total_tokens":      300,
	}
	event := NewCompleteEvent(usage)

	if event.Type != Complete {
		t.Errorf("Expected type %s, got %s", Complete, event.Type)
	}

	usageData, ok := event.Data["usage"].(map[string]interface{})
	if !ok {
		t.Error("Usage should be a map")
	}

	if usageData["total_tokens"] != 300 {
		t.Errorf("Expected total_tokens 300, got %v", usageData["total_tokens"])
	}
}

func TestNewErrorEvent(t *testing.T) {
	event := NewErrorEvent("Something went wrong", "ERR_500")

	if event.Type != Error {
		t.Errorf("Expected type %s, got %s", Error, event.Type)
	}

	if event.Data["error"] != "Something went wrong" {
		t.Errorf("Expected error 'Something went wrong', got %v", event.Data["error"])
	}

	if event.Data["error_code"] != "ERR_500" {
		t.Errorf("Expected error_code 'ERR_500', got %v", event.Data["error_code"])
	}
}

func TestStreamEventJSONSerialization(t *testing.T) {
	event := NewThinkingEvent("Test message")

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded["type"] != string(Thinking) {
		t.Errorf("Expected type 'thinking', got %v", decoded["type"])
	}

	data, ok := decoded["data"].(map[string]interface{})
	if !ok {
		t.Error("Data should be a map")
	}

	if data["message"] != "Test message" {
		t.Errorf("Expected message 'Test message', got %v", data["message"])
	}
}

func TestStreamEventString(t *testing.T) {
	event := NewThinkingEvent("Test message")

	str := event.String()
	if str == "" {
		t.Error("String() should not return empty string")
	}

	var decoded map[string]interface{}
	err := json.Unmarshal([]byte(str), &decoded)
	if err != nil {
		t.Fatalf("String result should be valid JSON: %v", err)
	}
}

func TestFromJSON(t *testing.T) {
	original := NewProviderSelectedEvent("ollama", "llama2")

	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	decoded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Expected type %s, got %s", original.Type, decoded.Type)
	}

	if decoded.Data["provider"] != "ollama" {
		t.Errorf("Expected provider 'ollama', got %v", decoded.Data["provider"])
	}

	if decoded.Data["model"] != "llama2" {
		t.Errorf("Expected model 'llama2', got %v", decoded.Data["model"])
	}
}

func TestFromJSONInvalidData(t *testing.T) {
	invalidJSON := []byte(`{"invalid json`)

	_, err := FromJSON(invalidJSON)
	if err == nil {
		t.Error("FromJSON should fail with invalid JSON")
	}
}

func TestEventTimestampConsistency(t *testing.T) {
	before := time.Now()
	event := NewThinkingEvent("Test")
	after := time.Now()

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Error("Event timestamp should be between before and after times")
	}
}

func TestEventDataModification(t *testing.T) {
	event := NewThinkingEvent("Original message")
	event.Data["message"] = "Modified message"

	if event.Data["message"] != "Modified message" {
		t.Error("Event data should be modifiable")
	}
}

func TestMultipleEventsIndependence(t *testing.T) {
	event1 := NewThinkingEvent("Message 1")
	event2 := NewThinkingEvent("Message 2")

	if event1.Data["message"] == event2.Data["message"] {
		t.Error("Events should have independent data")
	}

	event1.Data["message"] = "Modified"
	if event2.Data["message"] == "Modified" {
		t.Error("Modifying one event should not affect another")
	}
}

func TestEmptyDataFields(t *testing.T) {
	event := StreamEvent{
		Type:      Thinking,
		Data:      map[string]interface{}{},
		Timestamp: time.Now(),
	}

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON should work with empty data: %v", err)
	}

	decoded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON should work with empty data: %v", err)
	}

	if len(decoded.Data) != 0 {
		t.Error("Empty data should remain empty")
	}
}

func TestNilDataInConstructor(t *testing.T) {
	event := NewToolInvokedEvent("test_tool", nil)

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON should work with nil arguments: %v", err)
	}

	decoded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON should work with nil arguments: %v", err)
	}

	if decoded.Type != ToolInvoked {
		t.Errorf("Expected type %s, got %s", ToolInvoked, decoded.Type)
	}
}

func TestComplexDataStructures(t *testing.T) {
	complexArgs := map[string]interface{}{
		"nested": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep value",
			},
		},
		"array":  []interface{}{"a", "b", "c"},
		"number": 42,
		"float":  3.14,
		"bool":   true,
	}

	event := NewToolInvokedEvent("complex_tool", complexArgs)

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed with complex data: %v", err)
	}

	decoded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed with complex data: %v", err)
	}

	args, ok := decoded.Data["arguments"].(map[string]interface{})
	if !ok {
		t.Fatal("Arguments should be a map")
	}

	nested, ok := args["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("Nested should be a map")
	}

	level2, ok := nested["level2"].(map[string]interface{})
	if !ok {
		t.Fatal("Level2 should be a map")
	}

	if level2["level3"] != "deep value" {
		t.Errorf("Expected 'deep value', got %v", level2["level3"])
	}
}

func TestNewTraceSpanStartEvent(t *testing.T) {
	startTime := time.Now()
	inputs := map[string]interface{}{
		"query":      "test query",
		"session_id": "session-123",
	}

	event := NewTraceSpanStartEvent(
		"trace-123",
		"span-456",
		"parent-789",
		"HandleQuery",
		startTime,
		inputs,
	)

	if event.Type != TraceSpanStart {
		t.Errorf("Expected type %s, got %s", TraceSpanStart, event.Type)
	}

	if event.Data["trace_id"] != "trace-123" {
		t.Errorf("Expected trace_id 'trace-123', got %v", event.Data["trace_id"])
	}

	if event.Data["span_id"] != "span-456" {
		t.Errorf("Expected span_id 'span-456', got %v", event.Data["span_id"])
	}

	if event.Data["parent_id"] != "parent-789" {
		t.Errorf("Expected parent_id 'parent-789', got %v", event.Data["parent_id"])
	}

	if event.Data["name"] != "HandleQuery" {
		t.Errorf("Expected name 'HandleQuery', got %v", event.Data["name"])
	}

	if event.Data["start_time"] != startTime {
		t.Errorf("Expected start_time %v, got %v", startTime, event.Data["start_time"])
	}

	inputsData, ok := event.Data["inputs"].(map[string]interface{})
	if !ok {
		t.Error("Inputs should be a map")
	}

	if inputsData["query"] != "test query" {
		t.Errorf("Expected query 'test query', got %v", inputsData["query"])
	}
}

func TestNewTraceSpanEndEvent(t *testing.T) {
	startTime := time.Now()
	endTime := time.Now().Add(100 * time.Millisecond)

	inputs := map[string]interface{}{
		"query": "test query",
	}
	outputs := map[string]interface{}{
		"response": "test response",
	}
	metadata := map[string]interface{}{
		"model":  "deepseek-chat",
		"tokens": 150,
	}

	event := NewTraceSpanEndEvent(
		"trace-123",
		"span-456",
		"parent-789",
		"HandleQuery",
		startTime,
		&endTime,
		inputs,
		outputs,
		metadata,
		"completed",
		100,
	)

	if event.Type != TraceSpanEnd {
		t.Errorf("Expected type %s, got %s", TraceSpanEnd, event.Type)
	}

	if event.Data["trace_id"] != "trace-123" {
		t.Errorf("Expected trace_id 'trace-123', got %v", event.Data["trace_id"])
	}

	if event.Data["span_id"] != "span-456" {
		t.Errorf("Expected span_id 'span-456', got %v", event.Data["span_id"])
	}

	if event.Data["status"] != "completed" {
		t.Errorf("Expected status 'completed', got %v", event.Data["status"])
	}

	if event.Data["duration_ms"] != int64(100) {
		t.Errorf("Expected duration_ms 100, got %v", event.Data["duration_ms"])
	}

	outputsData, ok := event.Data["outputs"].(map[string]interface{})
	if !ok {
		t.Error("Outputs should be a map")
	}

	if outputsData["response"] != "test response" {
		t.Errorf("Expected response 'test response', got %v", outputsData["response"])
	}

	metadataData, ok := event.Data["metadata"].(map[string]interface{})
	if !ok {
		t.Error("Metadata should be a map")
	}

	if metadataData["model"] != "deepseek-chat" {
		t.Errorf("Expected model 'deepseek-chat', got %v", metadataData["model"])
	}
}

func TestTraceSpanStartEventJSONSerialization(t *testing.T) {
	startTime := time.Now()
	inputs := map[string]interface{}{
		"query": "test query",
	}

	event := NewTraceSpanStartEvent(
		"trace-123",
		"span-456",
		"",
		"HandleQuery",
		startTime,
		inputs,
	)

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded["type"] != string(TraceSpanStart) {
		t.Errorf("Expected type 'trace_span_start', got %v", decoded["type"])
	}

	data, ok := decoded["data"].(map[string]interface{})
	if !ok {
		t.Error("Data should be a map")
	}

	if data["trace_id"] != "trace-123" {
		t.Errorf("Expected trace_id 'trace-123', got %v", data["trace_id"])
	}

	if data["span_id"] != "span-456" {
		t.Errorf("Expected span_id 'span-456', got %v", data["span_id"])
	}
}

func TestTraceSpanEndEventJSONSerialization(t *testing.T) {
	startTime := time.Now()
	endTime := time.Now().Add(100 * time.Millisecond)

	event := NewTraceSpanEndEvent(
		"trace-123",
		"span-456",
		"parent-789",
		"HandleQuery",
		startTime,
		&endTime,
		nil,
		nil,
		nil,
		"completed",
		100,
	)

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded["type"] != string(TraceSpanEnd) {
		t.Errorf("Expected type 'trace_span_end', got %v", decoded["type"])
	}

	data, ok := decoded["data"].(map[string]interface{})
	if !ok {
		t.Error("Data should be a map")
	}

	if data["status"] != "completed" {
		t.Errorf("Expected status 'completed', got %v", data["status"])
	}
}

func TestTraceSpanStartEventWithNilInputs(t *testing.T) {
	startTime := time.Now()

	event := NewTraceSpanStartEvent(
		"trace-123",
		"span-456",
		"",
		"HandleQuery",
		startTime,
		nil,
	)

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON should work with nil inputs: %v", err)
	}

	decoded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON should work with nil inputs: %v", err)
	}

	if decoded.Type != TraceSpanStart {
		t.Errorf("Expected type %s, got %s", TraceSpanStart, decoded.Type)
	}
}

func TestTraceSpanEndEventWithNilEndTime(t *testing.T) {
	startTime := time.Now()

	event := NewTraceSpanEndEvent(
		"trace-123",
		"span-456",
		"",
		"HandleQuery",
		startTime,
		nil,
		nil,
		nil,
		nil,
		"running",
		0,
	)

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON should work with nil end time: %v", err)
	}

	decoded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON should work with nil end time: %v", err)
	}

	if decoded.Type != TraceSpanEnd {
		t.Errorf("Expected type %s, got %s", TraceSpanEnd, decoded.Type)
	}

	if decoded.Data["status"] != "running" {
		t.Errorf("Expected status 'running', got %v", decoded.Data["status"])
	}
}

func BenchmarkNewIntentClassifiedEvent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewIntentClassifiedEvent("THINK", 0.85)
	}
}

func BenchmarkNewToolInvokedEvent(b *testing.B) {
	args := map[string]interface{}{
		"file_path": "/tmp/test.txt",
		"encoding":  "utf-8",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewToolInvokedEvent("read_file", args)
	}
}

func BenchmarkToJSON(b *testing.B) {
	event := NewThinkingEvent("Test message")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.ToJSON()
	}
}

func BenchmarkFromJSON(b *testing.B) {
	event := NewThinkingEvent("Test message")
	jsonData, _ := event.ToJSON()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromJSON(jsonData)
	}
}
