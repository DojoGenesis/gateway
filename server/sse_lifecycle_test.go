package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/events"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSSEStreamingLifecycle verifies the full SSE event streaming lifecycle
// for an orchestration: event replay, live streaming, and terminal event handling.
func TestSSEStreamingLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "7340",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	// Create an orchestration with pre-existing events (simulating a plan that already ran)
	now := time.Now()
	eventChan := make(chan events.StreamEvent, 100)

	state := &OrchestrationState{
		ID:        "orch-sse-test",
		TaskID:    "task-1",
		Status:    "executing",
		CreatedAt: now,
		Events: []events.StreamEvent{
			{
				Type: events.OrchestrationPlanCreated,
				Data: map[string]interface{}{
					"orchestration_id": "orch-sse-test",
					"plan":             map[string]interface{}{"id": "plan-1", "node_count": 2},
				},
				Timestamp: now,
			},
			{
				Type: events.OrchestrationNodeStart,
				Data: map[string]interface{}{
					"orchestration_id": "orch-sse-test",
					"node_id":          "node-1",
					"tool_name":        "web_search",
				},
				Timestamp: now.Add(10 * time.Millisecond),
			},
		},
		EventChan: eventChan,
	}
	s.orchestrations.Store(state)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req")
		c.Next()
	})
	router.GET("/v1/orchestrate/:id/events", s.handleOrchestrationEvents)

	// Push a completion event into the channel so the stream terminates
	go func() {
		time.Sleep(50 * time.Millisecond)
		completeEvt := events.StreamEvent{
			Type: events.OrchestrationComplete,
			Data: map[string]interface{}{
				"orchestration_id":  "orch-sse-test",
				"total_duration_ms": 250,
			},
			Timestamp: now.Add(100 * time.Millisecond),
		}
		state.mu.Lock()
		state.Events = append(state.Events, completeEvt)
		state.mu.Unlock()
		eventChan <- completeEvt
	}()

	req, _ := http.NewRequest(http.MethodGet, "/v1/orchestrate/orch-sse-test/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify SSE headers
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))

	// Verify body contains SSE data lines
	body := w.Body.String()
	assert.Contains(t, body, "data: {")

	// Parse individual events from the SSE stream
	sseEvents := parseSSEEvents(t, body)
	require.GreaterOrEqual(t, len(sseEvents), 3, "should have at least 3 events (2 replayed + 1 live)")

	// Verify first event is plan_created (replayed)
	assert.Equal(t, string(events.OrchestrationPlanCreated), sseEvents[0]["type"])

	// Verify second event is node_start (replayed)
	assert.Equal(t, string(events.OrchestrationNodeStart), sseEvents[1]["type"])

	// Verify final event is complete (live)
	lastEvent := sseEvents[len(sseEvents)-1]
	assert.Equal(t, string(events.OrchestrationComplete), lastEvent["type"])
}

// TestSSEStreamingLifecycle_CompletedOrchestration verifies that connecting
// to a completed orchestration replays all events without blocking.
func TestSSEStreamingLifecycle_CompletedOrchestration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "7340",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	now := time.Now()

	state := &OrchestrationState{
		ID:        "orch-done",
		TaskID:    "task-2",
		Status:    "complete", // Already terminal
		CreatedAt: now,
		Events: []events.StreamEvent{
			{
				Type:      events.OrchestrationPlanCreated,
				Data:      map[string]interface{}{"plan_id": "p1"},
				Timestamp: now,
			},
			{
				Type:      events.OrchestrationComplete,
				Data:      map[string]interface{}{"total_nodes": 1},
				Timestamp: now.Add(50 * time.Millisecond),
			},
		},
		EventChan: make(chan events.StreamEvent, 10),
	}
	s.orchestrations.Store(state)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req")
		c.Next()
	})
	router.GET("/v1/orchestrate/:id/events", s.handleOrchestrationEvents)

	req, _ := http.NewRequest(http.MethodGet, "/v1/orchestrate/orch-done/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should complete immediately (not block) because status is already terminal
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

	body := w.Body.String()
	sseEvents := parseSSEEvents(t, body)
	assert.Len(t, sseEvents, 2, "should replay exactly 2 events")
}

// TestSSEStreamingLifecycle_NotFound verifies error for unknown orchestration ID.
func TestSSEStreamingLifecycle_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "7340",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req")
		c.Next()
	})
	router.GET("/v1/orchestrate/:id/events", s.handleOrchestrationEvents)

	req, _ := http.NewRequest(http.MethodGet, "/v1/orchestrate/nonexistent/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertErrorShape(t, w, http.StatusNotFound, "not_found")
}

// parseSSEEvents parses raw SSE body text into individual event data objects.
func parseSSEEvents(t *testing.T, body string) []map[string]interface{} {
	t.Helper()
	var result []map[string]interface{}

	// SSE format: "data: {json}\n\n"
	lines := splitSSELines(body)
	for _, line := range lines {
		if len(line) < 6 || line[:6] != "data: " {
			continue
		}
		jsonStr := line[6:]
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &evt); err != nil {
			continue // skip non-JSON data lines (like [DONE])
		}
		result = append(result, evt)
	}
	return result
}

// splitSSELines splits SSE body into individual data lines.
func splitSSELines(body string) []string {
	var lines []string
	current := ""
	for _, ch := range body {
		if ch == '\n' {
			if current != "" {
				lines = append(lines, current)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
