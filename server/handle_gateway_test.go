package server

// Regression test for the bug where handleGatewayOrchestrate passed
// c.Request.Context() to the background Execute goroutine. Gin cancels the
// request context the moment the HTTP 202 response is flushed, so any in-flight
// orchestration saw an already-cancelled context and silently set its state to
// "failed". The fix uses context.Background() (with a 10-minute timeout), which
// matches the pattern in handle_orchestrate.go:165.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ctxRecordingInvoker is a ToolInvokerInterface that records the context.Err()
// value at the moment InvokeTool is called. This lets us assert whether the
// background goroutine received a live or already-cancelled context.
type ctxRecordingInvoker struct {
	mu      sync.Mutex
	ctxErrs []error
	called  chan struct{}
}

func newCtxRecordingInvoker() *ctxRecordingInvoker {
	return &ctxRecordingInvoker{called: make(chan struct{}, 1)}
}

func (r *ctxRecordingInvoker) InvokeTool(ctx context.Context, toolName string, parameters map[string]interface{}) (map[string]interface{}, error) {
	r.mu.Lock()
	r.ctxErrs = append(r.ctxErrs, ctx.Err())
	r.mu.Unlock()
	select {
	case r.called <- struct{}{}:
	default:
	}
	return map[string]interface{}{"status": "ok"}, nil
}

// firstCtxErr returns the context.Err() captured during the first InvokeTool
// call, or nil if InvokeTool was never called.
func (r *ctxRecordingInvoker) firstCtxErr() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.ctxErrs) == 0 {
		return nil
	}
	return r.ctxErrs[0]
}

// TestHandleGatewayOrchestrate_BackgroundContextNotCancelled is the regression
// test for the cancelled-context bug. It submits a one-node orchestration plan
// using an HTTP request whose context is pre-cancelled — mimicking the moment
// after Gin has flushed the 202 response and the request context is torn down.
//
// With the old (buggy) code the goroutine calls Execute with the already-
// cancelled request context; InvokeTool would receive a cancelled ctx and the
// orchestration state would transition to "failed". With the fix the goroutine
// always uses context.Background(), so InvokeTool receives a live context and
// the state transitions to "complete".
func TestHandleGatewayOrchestrate_BackgroundContextNotCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Build a real Engine with our context-recording invoker. No planner,
	// traceLogger, or budgetTracker needed — the plan is fully pre-built so
	// the planner is never called.
	invoker := newCtxRecordingInvoker()
	engine := orchestrationpkg.NewEngine(
		orchestrationpkg.DefaultEngineConfig(),
		nil,      // planner — not used; plan is pre-built by convertToOrchestrationPlan
		invoker,  // toolInvoker — records the context it receives
		nil,      // traceLogger
		nil,      // eventEmitter (intentionally unused by Engine per ADR-022 P0)
		nil,      // budgetTracker — nil means no budget check
	)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "7340",
			Environment: "test",
		},
		orchestrations:    NewOrchestrationStore(),
		agents:            make(map[string]*AgentRuntime),
		orchestrationEngine: engine,
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req")
		c.Next()
	})
	router.POST("/v1/gateway/orchestrate", s.handleGatewayOrchestrate)

	// Build the request body: one node in the DAG so the engine actually
	// calls InvokeTool and we can observe the context it received.
	body := map[string]interface{}{
		"plan": map[string]interface{}{
			"id":   "plan-ctx-regression",
			"name": "Context regression plan",
			"dag": []map[string]interface{}{
				{
					"id":        "node-1",
					"tool_name": "noop",
					"input":     map[string]interface{}{},
					"depends_on": []string{},
				},
			},
		},
		"user_id": "test-user",
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	// Use a pre-cancelled context to mimic the state of c.Request.Context()
	// after Gin has already flushed the HTTP response. The buggy code passes
	// this cancelled context to Execute; the fix passes context.Background().
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately — simulate post-flush state

	req, err := http.NewRequestWithContext(cancelledCtx, http.MethodPost, "/v1/gateway/orchestrate", bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Handler must return 202 Accepted.
	assert.Equal(t, http.StatusAccepted, w.Code, "handler should return 202 regardless of request context state")

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	executionID, ok := resp["execution_id"].(string)
	require.True(t, ok, "response should contain execution_id")
	require.NotEmpty(t, executionID)

	// Wait up to 2 seconds for the background goroutine to call InvokeTool.
	select {
	case <-invoker.called:
		// InvokeTool was reached — check the context it saw.
	case <-time.After(2 * time.Second):
		t.Fatal("background goroutine did not call InvokeTool within 2s")
	}

	// The context passed to InvokeTool must NOT be cancelled. With the buggy
	// code this assertion fails because c.Request.Context() (already cancelled)
	// is forwarded; with the fix context.Background() is used and ctx.Err() is nil.
	ctxErr := invoker.firstCtxErr()
	assert.NoError(t, ctxErr,
		"Execute must receive a live (non-cancelled) context even when the HTTP request context is already cancelled; "+
			"got ctx.Err() = %v", ctxErr)

	// Also verify the orchestration state reaches "complete" (not "failed").
	var finalStatus string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state, exists := s.orchestrations.Get(executionID)
		if exists {
			state.mu.Lock()
			finalStatus = state.Status
			state.mu.Unlock()
			if finalStatus == "complete" || finalStatus == "failed" {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, "complete", finalStatus,
		"orchestration state should be 'complete', not 'failed'; a 'failed' status indicates the context was cancelled before Execute finished")
}
