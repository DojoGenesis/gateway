package shared

import (
	"encoding/json"
	"errors"
	"testing"
)

// --- Type construction and JSON round-trip tests ---

func TestMessage_JSONRoundTrip(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "Hello, world!",
		ToolCalls: []ToolCall{
			{
				ID:   "call-1",
				Name: "search",
				Arguments: map[string]interface{}{
					"query": "test",
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Role != msg.Role {
		t.Errorf("role: expected %q, got %q", msg.Role, decoded.Role)
	}
	if decoded.Content != msg.Content {
		t.Errorf("content: expected %q, got %q", msg.Content, decoded.Content)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(decoded.ToolCalls))
	}
	if decoded.ToolCalls[0].Name != "search" {
		t.Errorf("tool name: expected search, got %s", decoded.ToolCalls[0].Name)
	}
}

func TestMessage_ToolResponse(t *testing.T) {
	msg := Message{
		Role:       "tool",
		Content:    `{"result": "found"}`,
		ToolCallID: "call-1",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ToolCallID != "call-1" {
		t.Errorf("tool_call_id: expected call-1, got %s", decoded.ToolCallID)
	}
}

func TestToolResult_JSONRoundTrip(t *testing.T) {
	tr := ToolResult{
		ToolName: "file_read",
		Success:  true,
		Content:  "file contents here",
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ToolName != tr.ToolName || decoded.Success != tr.Success {
		t.Errorf("mismatch: got %+v", decoded)
	}
}

func TestToolResult_Error(t *testing.T) {
	tr := ToolResult{
		ToolName: "exec",
		Success:  false,
		Error:    "command failed",
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Success {
		t.Error("expected success=false")
	}
	if decoded.Error != "command failed" {
		t.Errorf("expected 'command failed', got %q", decoded.Error)
	}
}

func TestUsage_JSONRoundTrip(t *testing.T) {
	u := Usage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Usage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.TotalTokens != 150 {
		t.Errorf("expected 150 total tokens, got %d", decoded.TotalTokens)
	}
}

func TestCallRequest_JSONRoundTrip(t *testing.T) {
	req := CallRequest{
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
		Tools: []ToolUse{
			{
				Name:        "search",
				Description: "search the web",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
		MaxTokens:   1024,
		Temperature: 0.7,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded CallRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Messages) != 1 || decoded.Messages[0].Role != "user" {
		t.Errorf("messages mismatch: %+v", decoded.Messages)
	}
	if len(decoded.Tools) != 1 || decoded.Tools[0].Name != "search" {
		t.Errorf("tools mismatch: %+v", decoded.Tools)
	}
	if decoded.MaxTokens != 1024 {
		t.Errorf("expected max_tokens=1024, got %d", decoded.MaxTokens)
	}
}

func TestCallResponse_JSONRoundTrip(t *testing.T) {
	resp := CallResponse{
		Message:    Message{Role: "assistant", Content: "I found it"},
		TokensUsed: 42,
		StopReason: "stop",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded CallResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.StopReason != "stop" {
		t.Errorf("expected stop_reason=stop, got %s", decoded.StopReason)
	}
}

func TestProviderConfig_JSONRoundTrip(t *testing.T) {
	cfg := ProviderConfig{
		Name:        "anthropic",
		MaxTokens:   4096,
		Temperature: 0.5,
		Endpoint:    "https://api.anthropic.com",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ProviderConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Name != "anthropic" || decoded.MaxTokens != 4096 {
		t.Errorf("mismatch: %+v", decoded)
	}
}

// --- TaskStatus and NodeState constants ---

func TestTaskStatus_Values(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		want   string
	}{
		{"pending", TaskPending, "pending"},
		{"running", TaskRunning, "running"},
		{"completed", TaskCompleted, "completed"},
		{"failed", TaskFailed, "failed"},
		{"skipped", TaskSkipped, "skipped"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("TaskStatus %s = %q, want %q", tt.name, string(tt.status), tt.want)
			}
		})
	}
}

func TestNodeState_Values(t *testing.T) {
	tests := []struct {
		name  string
		state NodeState
		want  string
	}{
		{"pending", NodeStatePending, "pending"},
		{"running", NodeStateRunning, "running"},
		{"success", NodeStateSuccess, "success"},
		{"failed", NodeStateFailed, "failed"},
		{"skipped", NodeStateSkipped, "skipped"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.want {
				t.Errorf("NodeState %s = %q, want %q", tt.name, string(tt.state), tt.want)
			}
		})
	}
}

// --- Sentinel errors ---

func TestSentinelErrors_Distinct(t *testing.T) {
	errs := []error{
		ErrToolNotFound,
		ErrToolTimeout,
		ErrToolExecution,
		ErrProviderNotFound,
		ErrProviderUnavailable,
		ErrTaskFailed,
		ErrTaskCancelled,
		ErrCircuitOpen,
		ErrMaxRetriesExceeded,
		ErrMemoryNotFound,
		ErrInvalidInput,
		ErrUnauthorized,
		ErrBudgetExceeded,
	}

	// Verify all errors are non-nil
	for i, err := range errs {
		if err == nil {
			t.Errorf("error at index %d is nil", i)
		}
	}

	// Verify all errors are distinct
	seen := make(map[string]bool)
	for _, err := range errs {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("duplicate error message: %q", msg)
		}
		seen[msg] = true
	}
}

func TestSentinelErrors_Is(t *testing.T) {
	// Wrap and unwrap with errors.Is
	wrapped := errors.New("context: " + ErrToolNotFound.Error())
	_ = wrapped // wrapped doesn't chain, so errors.Is won't match

	// Direct match should work
	if !errors.Is(ErrToolNotFound, ErrToolNotFound) {
		t.Error("ErrToolNotFound should match itself")
	}
	if errors.Is(ErrToolNotFound, ErrToolTimeout) {
		t.Error("ErrToolNotFound should not match ErrToolTimeout")
	}
}

func TestSentinelErrors_Messages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrToolNotFound, "tool not found"},
		{ErrToolTimeout, "tool execution timed out"},
		{ErrToolExecution, "tool execution failed"},
		{ErrProviderNotFound, "provider not found"},
		{ErrProviderUnavailable, "provider unavailable"},
		{ErrTaskFailed, "task failed"},
		{ErrTaskCancelled, "task cancelled"},
		{ErrCircuitOpen, "circuit breaker open"},
		{ErrMaxRetriesExceeded, "max retries exceeded"},
		{ErrMemoryNotFound, "memory not found"},
		{ErrInvalidInput, "invalid input"},
		{ErrUnauthorized, "unauthorized"},
		{ErrBudgetExceeded, "budget exceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

// --- ToolUse ---

func TestToolUse_JSONOmitEmpty(t *testing.T) {
	// Message with no tool calls should omit tool_calls in JSON
	msg := Message{Role: "user", Content: "hi"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)

	if _, exists := raw["tool_calls"]; exists {
		t.Error("expected tool_calls to be omitted when empty")
	}
	if _, exists := raw["tool_call_id"]; exists {
		t.Error("expected tool_call_id to be omitted when empty")
	}
}
