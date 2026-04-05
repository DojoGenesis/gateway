package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/workflow"
)

// ---------------------------------------------------------------------------
// ExecutionBus — fans step events to SSE subscribers (one channel per run)
// ---------------------------------------------------------------------------

// stepEvent is a single step lifecycle notification published by WorkflowExecutor.
type stepEvent struct {
	StepID string `json:"stepId"`
	Status string `json:"status"` // running | completed | failed | skipped
}

// executionRun holds the buffered event channel and a done signal for one run.
type executionRun struct {
	ch   chan stepEvent
	done chan struct{}
}

// ExecutionBus maps run IDs to their event channels. WorkflowExecutor publishes
// into the bus via Publish; SSE handlers subscribe via Subscribe.
type ExecutionBus struct {
	mu   sync.RWMutex
	runs map[string]*executionRun
}

func newExecutionBus() *ExecutionBus {
	return &ExecutionBus{runs: make(map[string]*executionRun)}
}

// Register creates a slot for runID. Must be called before Publish or Subscribe.
func (b *ExecutionBus) Register(runID string) {
	b.mu.Lock()
	b.runs[runID] = &executionRun{
		ch:   make(chan stepEvent, 256),
		done: make(chan struct{}),
	}
	b.mu.Unlock()
}

// Publish sends a step event to the run's channel (non-blocking; drops if full).
func (b *ExecutionBus) Publish(runID, stepID, status string) {
	b.mu.RLock()
	run, ok := b.runs[runID]
	b.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case run.ch <- stepEvent{StepID: stepID, Status: status}:
	default:
		slog.Warn("execbus: dropped event (channel full)", "run_id", runID, "step", stepID)
	}
}

// Close signals the run is complete, causing SSE handlers to terminate the stream.
func (b *ExecutionBus) Close(runID string) {
	b.mu.RLock()
	run, ok := b.runs[runID]
	b.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case <-run.done:
		// already closed
	default:
		close(run.done)
	}
}

// Subscribe returns the event channel and done signal for a run.
// Returns (nil, nil) if the run has not been registered.
func (b *ExecutionBus) Subscribe(runID string) (ch <-chan stepEvent, done <-chan struct{}) {
	b.mu.RLock()
	run, ok := b.runs[runID]
	b.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	return run.ch, run.done
}

// ---------------------------------------------------------------------------
// POST /api/workflows/:name/execute
// ---------------------------------------------------------------------------

// handleWorkflowExecute starts async execution of a saved workflow and returns
// a run ID that the client uses to connect to the SSE execution stream.
//
// Response 202:
//
//	{"run_id": "uuid", "workflow": "name"}
func (s *Server) handleWorkflowExecute(c *gin.Context) {
	if s.workflowCAS == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "workflow execution not configured"})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow name required"})
		return
	}

	runID := uuid.New().String()
	s.execBus.Register(runID)

	executor := workflow.NewWorkflowExecutor(s.workflowCAS, func(_, stepID, status string) {
		s.execBus.Publish(runID, stepID, status)
	})

	// Execute in a background goroutine; use a detached context so the
	// execution continues even after the POST response is flushed.
	go func() {
		result, err := executor.Execute(context.Background(), name)
		if err != nil {
			slog.Error("workflow execution failed",
				"run_id", runID, "workflow", name, "error", err)
		} else {
			slog.Info("workflow execution complete",
				"run_id", runID, "workflow", name, "status", result.Status)
		}
		s.execBus.Close(runID)
	}()

	c.JSON(http.StatusAccepted, gin.H{"run_id": runID, "workflow": name})
}

// ---------------------------------------------------------------------------
// GET /api/workflows/:run_id/execution — SSE execution stream
// ---------------------------------------------------------------------------

// handleWorkflowExecutionStream streams step status events as SSE to the browser.
//
// Each event is a JSON-encoded stepEvent:
//
//	data: {"stepId":"step-1","status":"running"}
//	data: {"stepId":"step-1","status":"completed"}
//	...
//	data: {"type":"done"}
//
// The stream terminates when the run completes or the client disconnects.
func (s *Server) handleWorkflowExecutionStream(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id required"})
		return
	}

	ch, done := s.execBus.Subscribe(runID)
	if ch == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution run not found: " + runID})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	sendEvent := func(evt any) {
		data, err := json.Marshal(evt)
		if err != nil {
			slog.Error("execstream: marshal error", "error", err)
			return
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
	}

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			sendEvent(evt)

		case <-done:
			// Drain any events that arrived before done was closed.
			for {
				select {
				case evt := <-ch:
					sendEvent(evt)
				default:
					sendEvent(map[string]string{"type": "done"})
					return
				}
			}

		case <-c.Request.Context().Done():
			return
		}
	}
}
