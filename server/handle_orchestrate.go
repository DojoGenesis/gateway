package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/events"
	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
)

// ─── Orchestration API Types ─────────────────────────────────────────────────

// OrchestrateRequest is the request body for POST /v1/orchestrate.
type OrchestrateRequest struct {
	TaskDescription       string `json:"task_description" binding:"required"`
	UserID                string `json:"user_id,omitempty"`
	SessionID             string `json:"session_id,omitempty"`
	TimeoutSeconds        int    `json:"timeout_seconds,omitempty"`
	MaxReplanningAttempts int    `json:"max_replanning_attempts,omitempty"`
}

// OrchestrateResponse is the response for POST /v1/orchestrate.
type OrchestrateResponse struct {
	OrchestrationID     string `json:"orchestration_id"`
	Status              string `json:"status"`
	CreatedAt           string `json:"created_at"`
	EstimatedCompletion string `json:"estimated_completion,omitempty"`
}

// OrchestrationState tracks the lifecycle of an orchestration.
type OrchestrationState struct {
	ID          string
	TaskID      string
	Status      string // "planning", "executing", "complete", "failed"
	CreatedAt   time.Time
	Events      []events.StreamEvent
	EventChan   chan events.StreamEvent
	FinalOutput interface{}
	Error       string
	Plan        *orchestrationpkg.Plan // Store the plan for DAG retrieval
	mu          sync.Mutex
}

// OrchestrationStore is a thread-safe in-memory store for orchestration state.
type OrchestrationStore struct {
	orchestrations map[string]*OrchestrationState
	mu             sync.RWMutex
}

// NewOrchestrationStore creates a new orchestration store.
func NewOrchestrationStore() *OrchestrationStore {
	return &OrchestrationStore{
		orchestrations: make(map[string]*OrchestrationState),
	}
}

func (os *OrchestrationStore) Store(state *OrchestrationState) {
	os.mu.Lock()
	defer os.mu.Unlock()
	os.orchestrations[state.ID] = state
}

func (os *OrchestrationStore) Get(id string) (*OrchestrationState, bool) {
	os.mu.RLock()
	defer os.mu.RUnlock()
	state, exists := os.orchestrations[id]
	return state, exists
}

// handleOrchestrate handles POST /v1/orchestrate.
func (s *Server) handleOrchestrate(c *gin.Context) {
	if s.orchestrationEngine == nil || s.planner == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "Orchestration engine not initialized")
		return
	}

	var req OrchestrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request: "+err.Error())
		return
	}

	userID := req.UserID
	if userID == "" {
		userID = c.GetString("user_id")
	}

	timeout := 300 * time.Second
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	// Create task
	task := orchestrationpkg.NewTask(userID, req.TaskDescription)

	// Create orchestration state
	orchID := "orch-" + uuid.New().String()[:12]
	now := time.Now()

	eventChan := make(chan events.StreamEvent, 100)

	state := &OrchestrationState{
		ID:        orchID,
		TaskID:    task.ID,
		Status:    "planning",
		CreatedAt: now,
		Events:    make([]events.StreamEvent, 0),
		EventChan: eventChan,
	}
	s.orchestrations.Store(state)

	// Start orchestration in background
	go s.executeOrchestration(state, task, eventChan, timeout)

	c.JSON(http.StatusOK, OrchestrateResponse{
		OrchestrationID:     orchID,
		Status:              "planning",
		CreatedAt:           now.Format(time.RFC3339),
		EstimatedCompletion: now.Add(timeout).Format(time.RFC3339),
	})
}

func (s *Server) executeOrchestration(state *OrchestrationState, task *orchestrationpkg.Task, eventChan chan events.StreamEvent, timeout time.Duration) {
	defer close(eventChan)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Generate plan
	plan, err := s.planner.GeneratePlan(ctx, task)
	if err != nil {
		state.mu.Lock()
		state.Status = "failed"
		state.Error = err.Error()
		state.mu.Unlock()

		eventChan <- events.StreamEvent{
			Type: events.OrchestrationFailed,
			Data: map[string]interface{}{
				"orchestration_id": state.ID,
				"error":            err.Error(),
			},
			Timestamp: time.Now(),
		}
		return
	}

	// Emit plan_created event
	planEvent := events.StreamEvent{
		Type: events.OrchestrationPlanCreated,
		Data: map[string]interface{}{
			"orchestration_id": state.ID,
			"plan": map[string]interface{}{
				"id":         plan.ID,
				"node_count": len(plan.Nodes),
				"nodes":      plan.Nodes,
				"created_at": plan.CreatedAt.Format(time.RFC3339),
			},
		},
		Timestamp: time.Now(),
	}
	state.mu.Lock()
	state.Status = "executing"
	state.Plan = plan // Store plan for DAG retrieval
	state.Events = append(state.Events, planEvent)
	state.mu.Unlock()
	eventChan <- planEvent

	// Create engine event channel that forwards to our event channel
	engineEventChan := make(chan events.StreamEvent, 100)
	go func() {
		for evt := range engineEventChan {
			// Tag with orchestration ID
			if evt.Data == nil {
				evt.Data = make(map[string]interface{})
			}
			evt.Data["orchestration_id"] = state.ID

			state.mu.Lock()
			state.Events = append(state.Events, evt)
			state.mu.Unlock()

			eventChan <- evt
		}
	}()

	// Execute plan
	execErr := s.orchestrationEngine.Execute(ctx, plan, task, task.UserID)

	// Close engine event channel
	close(engineEventChan)

	if execErr != nil {
		state.mu.Lock()
		state.Status = "failed"
		state.Error = execErr.Error()
		state.mu.Unlock()

		failedEvent := events.StreamEvent{
			Type: events.OrchestrationFailed,
			Data: map[string]interface{}{
				"orchestration_id": state.ID,
				"error":            execErr.Error(),
			},
			Timestamp: time.Now(),
		}
		state.mu.Lock()
		state.Events = append(state.Events, failedEvent)
		state.mu.Unlock()
		eventChan <- failedEvent
		return
	}

	// Collect final output from plan nodes
	finalOutput := make(map[string]interface{})
	var totalDurationMs int64
	for _, node := range plan.Nodes {
		if node.Result != nil {
			finalOutput[node.ID] = node.Result
		}
		if node.StartTime != nil && node.EndTime != nil {
			totalDurationMs += node.EndTime.Sub(*node.StartTime).Milliseconds()
		}
	}

	state.mu.Lock()
	state.Status = "complete"
	state.FinalOutput = finalOutput
	state.mu.Unlock()

	completeEvent := events.StreamEvent{
		Type: events.OrchestrationComplete,
		Data: map[string]interface{}{
			"orchestration_id":  state.ID,
			"final_output":      finalOutput,
			"total_duration_ms": totalDurationMs,
		},
		Timestamp: time.Now(),
	}
	state.mu.Lock()
	state.Events = append(state.Events, completeEvent)
	state.mu.Unlock()
	eventChan <- completeEvent
}

// handleOrchestrationEvents handles GET /v1/orchestrate/:id/events.
func (s *Server) handleOrchestrationEvents(c *gin.Context) {
	orchID := c.Param("id")
	if orchID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Orchestration ID is required")
		return
	}

	state, exists := s.orchestrations.Get(orchID)
	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Orchestration not found: "+orchID)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Streaming not supported")
		return
	}

	// Send any already-collected events first (replay)
	state.mu.Lock()
	pastEvents := make([]events.StreamEvent, len(state.Events))
	copy(pastEvents, state.Events)
	state.mu.Unlock()

	for _, evt := range pastEvents {
		data, err := json.Marshal(evt)
		if err != nil {
			slog.Error("failed to marshal event", "error", err)
			continue
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
	}

	// If already terminal, we're done
	state.mu.Lock()
	status := state.Status
	state.mu.Unlock()

	if status == "complete" || status == "failed" {
		return
	}

	// Stream live events
	for {
		select {
		case evt, ok := <-state.EventChan:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				slog.Error("failed to marshal event", "error", err)
				continue
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			flusher.Flush()

			// Check if terminal event
			if evt.Type == events.OrchestrationComplete || evt.Type == events.OrchestrationFailed {
				return
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}
