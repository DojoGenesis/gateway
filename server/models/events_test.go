package models

import (
	"testing"
	"time"
)

func TestBroadcastRequest_Structure(t *testing.T) {
	tests := []struct {
		name     string
		request  BroadcastRequest
		validate func(t *testing.T, req BroadcastRequest)
	}{
		{
			name: "Valid broadcast request",
			request: BroadcastRequest{
				ClientID: "test-client-123",
				Event:    "message",
				Data:     `{"content":"hello"}`,
			},
			validate: func(t *testing.T, req BroadcastRequest) {
				if req.ClientID != "test-client-123" {
					t.Errorf("Expected ClientID 'test-client-123', got '%s'", req.ClientID)
				}
				if req.Event != "message" {
					t.Errorf("Expected Event 'message', got '%s'", req.Event)
				}
				if req.Data != `{"content":"hello"}` {
					t.Errorf("Expected Data with JSON, got '%s'", req.Data)
				}
			},
		},
		{
			name: "Empty fields allowed in struct",
			request: BroadcastRequest{
				ClientID: "",
				Event:    "",
				Data:     "",
			},
			validate: func(t *testing.T, req BroadcastRequest) {
				if req.ClientID != "" {
					t.Error("Expected empty ClientID")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.request)
		})
	}
}

func TestClient_Structure(t *testing.T) {
	now := time.Now()
	ch := make(chan string, 10)

	client := Client{
		ID:      "client-456",
		Channel: ch,
		Created: now,
	}

	if client.ID != "client-456" {
		t.Errorf("Expected ID 'client-456', got '%s'", client.ID)
	}

	if client.Channel == nil {
		t.Error("Expected non-nil channel")
	}

	if client.Created != now {
		t.Errorf("Expected Created timestamp %v, got %v", now, client.Created)
	}

	// Verify channel is writable
	select {
	case client.Channel <- "test":
		// Success
	default:
		t.Error("Channel should be writable")
	}

	close(ch)
}

func TestSSEEvent_FormatSSE(t *testing.T) {
	tests := []struct {
		name     string
		event    SSEEvent
		expected string
	}{
		{
			name: "Full SSE event",
			event: SSEEvent{
				Event: "message",
				Data:  `{"text":"hello"}`,
				ID:    "msg-1",
				Retry: 3000,
			},
			expected: "event: message\ndata: {\"text\":\"hello\"}\nid: msg-1\nretry: 3000\n\n",
		},
		{
			name: "Data only",
			event: SSEEvent{
				Data: "simple message",
			},
			expected: "data: simple message\n\n",
		},
		{
			name: "Event and data",
			event: SSEEvent{
				Event: "update",
				Data:  "status changed",
			},
			expected: "event: update\ndata: status changed\n\n",
		},
		{
			name: "With ID no event type",
			event: SSEEvent{
				Data: "content",
				ID:   "123",
			},
			expected: "data: content\nid: 123\n\n",
		},
		{
			name: "With retry",
			event: SSEEvent{
				Data:  "reconnect test",
				Retry: 5000,
			},
			expected: "data: reconnect test\nretry: 5000\n\n",
		},
		{
			name: "Empty data (edge case)",
			event: SSEEvent{
				Event: "ping",
				Data:  "",
			},
			expected: "event: ping\n\n",
		},
		{
			name: "Zero retry not included",
			event: SSEEvent{
				Data:  "test",
				Retry: 0,
			},
			expected: "data: test\n\n",
		},
		{
			name: "Multiline JSON data",
			event: SSEEvent{
				Event: "data",
				Data:  `{"key":"value","nested":{"a":1}}`,
			},
			expected: "event: data\ndata: {\"key\":\"value\",\"nested\":{\"a\":1}}\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.FormatSSE()
			if result != tt.expected {
				t.Errorf("FormatSSE() mismatch:\nExpected:\n%q\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestSSEEvent_FormatSSE_DoubleNewline(t *testing.T) {
	// Verify all formatted messages end with double newline (SSE spec requirement)
	events := []SSEEvent{
		{Data: "test"},
		{Event: "msg", Data: "test"},
		{Event: "msg", Data: "test", ID: "1", Retry: 1000},
	}

	for i, event := range events {
		result := event.FormatSSE()
		if len(result) < 2 {
			t.Errorf("Event %d: result too short", i)
			continue
		}

		if result[len(result)-2:] != "\n\n" {
			t.Errorf("Event %d: expected double newline at end, got %q", i, result[len(result)-2:])
		}
	}
}

func TestSSEEvent_FormatSSE_FieldOrdering(t *testing.T) {
	// Verify field ordering matches SSE spec (event, data, id, retry)
	event := SSEEvent{
		Event: "test",
		Data:  "content",
		ID:    "123",
		Retry: 1000,
	}

	result := event.FormatSSE()
	expected := "event: test\ndata: content\nid: 123\nretry: 1000\n\n"

	if result != expected {
		t.Errorf("Field ordering incorrect:\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func BenchmarkSSEEvent_FormatSSE(b *testing.B) {
	event := SSEEvent{
		Event: "message",
		Data:  `{"type":"agent_response","content":"Processing your request..."}`,
		ID:    "evt-123",
		Retry: 3000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.FormatSSE()
	}
}

func BenchmarkSSEEvent_FormatSSE_DataOnly(b *testing.B) {
	event := SSEEvent{
		Data: "simple message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event.FormatSSE()
	}
}
