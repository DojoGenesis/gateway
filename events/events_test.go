package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStreamEventSerialization(t *testing.T) {
	event := NewIntentClassifiedEvent("greeting", 0.95)

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	parsed, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if parsed.Type != IntentClassified {
		t.Errorf("expected type %s, got %s", IntentClassified, parsed.Type)
	}

	if parsed.Data["intent"] != "greeting" {
		t.Errorf("expected intent 'greeting', got %v", parsed.Data["intent"])
	}

	confidence, ok := parsed.Data["confidence"].(float64)
	if !ok || confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %v", parsed.Data["confidence"])
	}
}

func TestStreamEventDeserialization(t *testing.T) {
	jsonStr := `{"type":"tool_invoked","data":{"tool":"web_search","arguments":{"query":"test"}},"timestamp":"2026-01-01T00:00:00Z"}`

	event, err := FromJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if event.Type != ToolInvoked {
		t.Errorf("expected type %s, got %s", ToolInvoked, event.Type)
	}

	if event.Data["tool"] != "web_search" {
		t.Errorf("expected tool 'web_search', got %v", event.Data["tool"])
	}
}

func TestStreamEventString(t *testing.T) {
	event := NewErrorEvent("something failed", "INTERNAL_ERROR")
	s := event.String()
	if s == "" {
		t.Error("String() returned empty string")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(s), &parsed); err != nil {
		t.Errorf("String() output is not valid JSON: %v", err)
	}
}

func TestAllEventConstructors(t *testing.T) {
	tests := []struct {
		name     string
		event    StreamEvent
		expected EventType
	}{
		{"IntentClassified", NewIntentClassifiedEvent("test", 0.5), IntentClassified},
		{"ProviderSelected", NewProviderSelectedEvent("openai", "gpt-4"), ProviderSelected},
		{"ToolInvoked", NewToolInvokedEvent("search", map[string]interface{}{"q": "test"}), ToolInvoked},
		{"ToolCompleted", NewToolCompletedEvent("search", "result", 100), ToolCompleted},
		{"Thinking", NewThinkingEvent("processing..."), Thinking},
		{"ResponseChunk", NewResponseChunkEvent("Hello"), ResponseChunk},
		{"MemoryRetrieved", NewMemoryRetrievedEvent(2, []interface{}{"a", "b"}), MemoryRetrieved},
		{"Complete", NewCompleteEvent(map[string]interface{}{"tokens": 100}), Complete},
		{"Error", NewErrorEvent("fail", "ERR"), Error},
		{"TraceSpanStart", NewTraceSpanStartEvent("t1", "s1", "p1", "test", time.Now(), nil), TraceSpanStart},
		{"TraceSpanEnd", NewTraceSpanEndEvent("t1", "s1", "p1", "test", time.Now(), nil, nil, nil, nil, "ok", 50), TraceSpanEnd},
		{"ArtifactCreated", NewArtifactCreatedEvent("a1", "name", "code", "p1"), ArtifactCreated},
		{"ArtifactUpdated", NewArtifactUpdatedEvent("a1", "name", 2, "update"), ArtifactUpdated},
		{"ProjectSwitched", NewProjectSwitchedEvent("p1", "MyProject"), ProjectSwitched},
		{"DiagramRendered", NewDiagramRenderedEvent("d1", "flowchart", "svg"), DiagramRendered},
		{"OrchPlanCreated", NewOrchestrationPlanCreatedEvent("pl1", "t1", 5, 0.1, nil), OrchestrationPlanCreated},
		{"OrchNodeStart", NewOrchestrationNodeStartEvent("n1", "pl1", "search", nil), OrchestrationNodeStart},
		{"OrchNodeEnd", NewOrchestrationNodeEndEvent("n1", "pl1", "search", "success", nil, "", 200), OrchestrationNodeEnd},
		{"OrchReplanning", NewOrchestrationReplanningEvent("pl1", "t1", "failure", []string{"n1"}), OrchestrationReplanning},
		{"OrchComplete", NewOrchestrationCompleteEvent("pl1", "t1", 5, 4, 1, 1000), OrchestrationComplete},
		{"OrchFailed", NewOrchestrationFailedEvent("pl1", "t1", "timeout"), OrchestrationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event.Type != tt.expected {
				t.Errorf("expected type %s, got %s", tt.expected, tt.event.Type)
			}
			if tt.event.Timestamp.IsZero() {
				t.Error("timestamp should not be zero")
			}

			// Verify serialization roundtrip
			data, err := tt.event.ToJSON()
			if err != nil {
				t.Fatalf("ToJSON failed: %v", err)
			}
			parsed, err := FromJSON(data)
			if err != nil {
				t.Fatalf("FromJSON failed: %v", err)
			}
			if parsed.Type != tt.expected {
				t.Errorf("roundtrip: expected type %s, got %s", tt.expected, parsed.Type)
			}
		})
	}
}

func TestFromJSONInvalid(t *testing.T) {
	_, err := FromJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
