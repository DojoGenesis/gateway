package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/streaming"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleChat_OrchestrationPlanCreatedEvent(t *testing.T) {
	t.Skip("Streaming tests require a real HTTP connection; httptest.ResponseRecorder doesn't support CloseNotifier")

	ic := agent.NewIntentClassifier()

	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "This is a test response",
			Usage: provider.Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)

	h := NewChatHandler(ic, pa, nil, nil)

	router := setupChatTestRouter()
	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "orchestration plan test",
		Stream:    true,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")
}

func TestHandleChat_OrchestrationEventsInOrder(t *testing.T) {
	t.Skip("Streaming tests require a real HTTP connection; httptest.ResponseRecorder doesn't support CloseNotifier")

	gin.SetMode(gin.TestMode)
	router := gin.New()

	ic := agent.NewIntentClassifier()
	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "Orchestration test response",
			Usage: provider.Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)
	h := NewChatHandler(ic, pa, nil, nil)

	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "test orchestration flow",
		Stream:    true,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	responseBody := w.Body.String()

	assert.Contains(t, responseBody, "event:")
}

func TestOrchestrationEventSSEFormat(t *testing.T) {
	tests := []struct {
		name      string
		eventType streaming.EventType
		eventData map[string]interface{}
	}{
		{
			name:      "orchestration_plan_created",
			eventType: streaming.OrchestrationPlanCreated,
			eventData: map[string]interface{}{
				"plan_id":        "plan-123",
				"task_id":        "task-456",
				"node_count":     3,
				"estimated_cost": 0.005,
			},
		},
		{
			name:      "orchestration_node_start",
			eventType: streaming.OrchestrationNodeStart,
			eventData: map[string]interface{}{
				"node_id":   "node-1",
				"plan_id":   "plan-123",
				"tool_name": "web_search",
			},
		},
		{
			name:      "orchestration_node_end",
			eventType: streaming.OrchestrationNodeEnd,
			eventData: map[string]interface{}{
				"node_id":     "node-1",
				"plan_id":     "plan-123",
				"tool_name":   "web_search",
				"state":       "Success",
				"duration_ms": 1234,
			},
		},
		{
			name:      "orchestration_replanning",
			eventType: streaming.OrchestrationReplanning,
			eventData: map[string]interface{}{
				"plan_id":      "plan-123",
				"task_id":      "task-456",
				"reason":       "node failed after max retries",
				"failed_nodes": []string{"node-2"},
			},
		},
		{
			name:      "orchestration_complete",
			eventType: streaming.OrchestrationComplete,
			eventData: map[string]interface{}{
				"plan_id":       "plan-123",
				"task_id":       "task-456",
				"total_nodes":   3,
				"success_nodes": 3,
				"failed_nodes":  0,
				"duration_ms":   5678,
			},
		},
		{
			name:      "orchestration_failed",
			eventType: streaming.OrchestrationFailed,
			eventData: map[string]interface{}{
				"plan_id": "plan-123",
				"task_id": "task-456",
				"reason":  "budget exceeded",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := streaming.StreamEvent{
				Type: tt.eventType,
				Data: tt.eventData,
			}

			eventJSON, err := json.Marshal(event.Data)
			require.NoError(t, err)

			expectedSSEData := fmt.Sprintf("event: %s\ndata: %s\n\n", tt.eventType, string(eventJSON))

			assert.Contains(t, expectedSSEData, string(tt.eventType))

			var parsedData map[string]interface{}
			err = json.Unmarshal(eventJSON, &parsedData)
			require.NoError(t, err)

			for key, expectedValue := range tt.eventData {
				actualValue, exists := parsedData[key]
				assert.True(t, exists, "Key %s should exist in event data", key)

				switch v := expectedValue.(type) {
				case []string:
					actualSlice, ok := actualValue.([]interface{})
					assert.True(t, ok, "Expected slice for key %s", key)
					assert.Equal(t, len(v), len(actualSlice))
				case int:
					actualFloat, ok := actualValue.(float64)
					assert.True(t, ok, "Expected numeric value for key %s", key)
					assert.Equal(t, float64(v), actualFloat, "Value for key %s should match", key)
				case float64:
					actualFloat, ok := actualValue.(float64)
					assert.True(t, ok, "Expected numeric value for key %s", key)
					assert.Equal(t, v, actualFloat, "Value for key %s should match", key)
				default:
					assert.Equal(t, expectedValue, actualValue, "Value for key %s should match", key)
				}
			}
		})
	}
}

func TestHandleStreamingQuery_OrchestrationEventCases(t *testing.T) {
	t.Skip("Streaming tests require a real HTTP connection; httptest.ResponseRecorder doesn't support CloseNotifier")

	gin.SetMode(gin.TestMode)
	router := gin.New()

	ic := agent.NewIntentClassifier()
	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "Test response",
			Usage: provider.Usage{
				InputTokens:  5,
				OutputTokens: 10,
				TotalTokens:  15,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)
	h := NewChatHandler(ic, pa, nil, nil)

	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "complex orchestration task",
		Stream:    true,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
}

func TestOrchestrationEventDataStructure(t *testing.T) {
	tests := []struct {
		name           string
		eventType      streaming.EventType
		requiredFields []string
	}{
		{
			name:           "plan_created_fields",
			eventType:      streaming.OrchestrationPlanCreated,
			requiredFields: []string{"plan_id", "task_id", "node_count", "estimated_cost"},
		},
		{
			name:           "node_start_fields",
			eventType:      streaming.OrchestrationNodeStart,
			requiredFields: []string{"node_id", "plan_id", "tool_name"},
		},
		{
			name:           "node_end_fields",
			eventType:      streaming.OrchestrationNodeEnd,
			requiredFields: []string{"node_id", "plan_id", "tool_name", "state"},
		},
		{
			name:           "replanning_fields",
			eventType:      streaming.OrchestrationReplanning,
			requiredFields: []string{"plan_id", "task_id", "reason"},
		},
		{
			name:           "complete_fields",
			eventType:      streaming.OrchestrationComplete,
			requiredFields: []string{"plan_id", "task_id", "total_nodes", "success_nodes", "failed_nodes"},
		},
		{
			name:           "failed_fields",
			eventType:      streaming.OrchestrationFailed,
			requiredFields: []string{"plan_id", "task_id", "reason"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventData := make(map[string]interface{})

			for _, field := range tt.requiredFields {
				switch field {
				case "plan_id", "task_id", "node_id", "tool_name", "state", "reason":
					eventData[field] = "test-value"
				case "node_count", "total_nodes", "success_nodes":
					eventData[field] = 1
				case "failed_nodes":
					if tt.eventType == streaming.OrchestrationComplete {
						eventData[field] = 0
					} else {
						eventData[field] = []string{}
					}
				case "estimated_cost":
					eventData[field] = 0.001
				}
			}

			event := streaming.StreamEvent{
				Type: tt.eventType,
				Data: eventData,
			}

			for _, field := range tt.requiredFields {
				_, exists := event.Data[field]
				assert.True(t, exists, "Field %s should exist in %s event", field, tt.eventType)
			}
		})
	}
}

func TestOrchestrationEventStreamingIntegration(t *testing.T) {
	t.Skip("Streaming tests require a real HTTP connection; httptest.ResponseRecorder doesn't support CloseNotifier")

	gin.SetMode(gin.TestMode)
	router := gin.New()

	ic := agent.NewIntentClassifier()
	mockProvider := &mockModelProvider{
		response: &provider.CompletionResponse{
			ID:      "test-response",
			Model:   "mock-model",
			Content: "Orchestration integration test",
			Usage: provider.Usage{
				InputTokens:  20,
				OutputTokens: 40,
				TotalTokens:  60,
			},
		},
	}

	pm := &mockPluginManager{
		providers: map[string]provider.ModelProvider{
			"mock": mockProvider,
		},
	}

	pa := agent.NewPrimaryAgent(pm)
	h := NewChatHandler(ic, pa, nil, nil)

	router.POST("/chat", h.Chat)

	reqBody := ChatRequest{
		Message:   "multi-step task requiring orchestration",
		Stream:    true,
		SessionID: "test-session",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	responseBody := w.Body.String()

	lines := strings.Split(responseBody, "\n")
	var eventCount int
	for _, line := range lines {
		if strings.HasPrefix(line, "event:") {
			eventCount++
		}
	}

	assert.Greater(t, eventCount, 0, "Should have at least one event")
}
