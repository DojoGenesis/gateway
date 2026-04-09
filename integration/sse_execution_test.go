package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test 2: SSE execution state delivery
//
// Starts a real HTTP test server with an ExecutionBus-backed SSE handler.
// Triggers a workflow run, connects an HTTP client to the SSE endpoint,
// and verifies step events arrive with stepId + status fields.
// Verifies each step transitions through queued -> running -> completed.
// ---------------------------------------------------------------------------

// sseExecutionBus mirrors server.ExecutionBus without importing the server
// package (which has heavy dependencies). This is a focused copy for
// integration testing that proves the SSE pattern works end-to-end.
type sseExecutionBus struct {
	mu   sync.RWMutex
	runs map[string]*sseRun
}

type sseRun struct {
	ch   chan sseStepEvent
	done chan struct{}
}

type sseStepEvent struct {
	StepID string `json:"stepId"`
	Status string `json:"status"`
}

func newSSEExecutionBus() *sseExecutionBus {
	return &sseExecutionBus{runs: make(map[string]*sseRun)}
}

func (b *sseExecutionBus) Register(runID string) {
	b.mu.Lock()
	b.runs[runID] = &sseRun{
		ch:   make(chan sseStepEvent, 256),
		done: make(chan struct{}),
	}
	b.mu.Unlock()
}

func (b *sseExecutionBus) Publish(runID, stepID, status string) {
	b.mu.RLock()
	run, ok := b.runs[runID]
	b.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case run.ch <- sseStepEvent{StepID: stepID, Status: status}:
	default:
	}
}

func (b *sseExecutionBus) Close(runID string) {
	b.mu.RLock()
	run, ok := b.runs[runID]
	b.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case <-run.done:
	default:
		close(run.done)
	}
}

func (b *sseExecutionBus) Subscribe(runID string) (ch <-chan sseStepEvent, done <-chan struct{}) {
	b.mu.RLock()
	run, ok := b.runs[runID]
	b.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	return run.ch, run.done
}

// sseHandler implements the SSE execution stream endpoint.
func sseHandler(bus *sseExecutionBus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract run_id from path: /api/workflows/{run_id}/execution
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/workflows/"), "/")
		if len(parts) < 1 || parts[0] == "" {
			http.Error(w, "run_id required", http.StatusBadRequest)
			return
		}
		runID := parts[0]

		ch, done := bus.Subscribe(runID)
		if ch == nil {
			http.Error(w, "execution run not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		sendEvent := func(evt any) {
			data, err := json.Marshal(evt)
			if err != nil {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
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
				// Drain remaining events.
				for {
					select {
					case evt := <-ch:
						sendEvent(evt)
					default:
						sendEvent(map[string]string{"type": "done"})
						return
					}
				}

			case <-r.Context().Done():
				return
			}
		}
	}
}

func TestSSE_ExecutionStateDelivery(t *testing.T) {
	bus := newSSEExecutionBus()
	runID := "test-run-001"
	bus.Register(runID)

	// Create HTTP test server with SSE handler.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/workflows/", sseHandler(bus))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Collect SSE events in a goroutine.
	var events []sseStepEvent
	var eventsMu sync.Mutex
	var sseErr error
	done := make(chan struct{})

	go func() {
		defer close(done)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			srv.URL+"/api/workflows/"+runID+"/execution", nil)
		if err != nil {
			sseErr = fmt.Errorf("create request: %w", err)
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			sseErr = fmt.Errorf("SSE connect: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			sseErr = fmt.Errorf("SSE status = %d, want 200", resp.StatusCode)
			return
		}

		if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
			sseErr = fmt.Errorf("Content-Type = %q, want text/event-stream", ct)
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			// Check if this is the "done" sentinel.
			var raw map[string]string
			if err := json.Unmarshal([]byte(data), &raw); err == nil {
				if raw["type"] == "done" {
					return
				}
			}

			var evt sseStepEvent
			if err := json.Unmarshal([]byte(data), &evt); err != nil {
				continue
			}
			eventsMu.Lock()
			events = append(events, evt)
			eventsMu.Unlock()
		}
	}()

	// Give the SSE client a moment to connect.
	time.Sleep(50 * time.Millisecond)

	// Simulate workflow execution: emit step lifecycle events.
	bus.Publish(runID, "step-1", "queued")
	bus.Publish(runID, "step-1", "running")
	bus.Publish(runID, "step-1", "completed")
	bus.Publish(runID, "step-2", "queued")
	bus.Publish(runID, "step-2", "running")
	bus.Publish(runID, "step-2", "completed")

	// Brief pause to let events drain to the SSE client.
	time.Sleep(50 * time.Millisecond)

	// Signal run completion.
	bus.Close(runID)

	// Wait for SSE reader goroutine to finish.
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SSE reader timed out")
	}

	if sseErr != nil {
		t.Fatal(sseErr)
	}

	// --- Assertions ---

	eventsMu.Lock()
	defer eventsMu.Unlock()

	if len(events) < 6 {
		t.Fatalf("received %d SSE events, want >= 6", len(events))
	}

	// Verify each event has stepId and status fields.
	for i, evt := range events {
		if evt.StepID == "" {
			t.Errorf("event[%d]: missing stepId", i)
		}
		if evt.Status == "" {
			t.Errorf("event[%d]: missing status", i)
		}
	}

	// Verify step-1 transitions: queued -> running -> completed.
	step1Statuses := filterStatuses(events, "step-1")
	assertTransitions(t, "step-1", step1Statuses, []string{"queued", "running", "completed"})

	// Verify step-2 transitions: queued -> running -> completed.
	step2Statuses := filterStatuses(events, "step-2")
	assertTransitions(t, "step-2", step2Statuses, []string{"queued", "running", "completed"})
}

func TestSSE_NotFoundForUnregisteredRun(t *testing.T) {
	bus := newSSEExecutionBus()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/workflows/", sseHandler(bus))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/workflows/nonexistent-run/execution")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body = %s", resp.StatusCode, string(body))
	}
}

// filterStatuses extracts the ordered list of statuses for a given stepID.
func filterStatuses(events []sseStepEvent, stepID string) []string {
	var statuses []string
	for _, e := range events {
		if e.StepID == stepID {
			statuses = append(statuses, e.Status)
		}
	}
	return statuses
}

// assertTransitions verifies that the status sequence matches the expected transitions.
func assertTransitions(t *testing.T, stepID string, got, want []string) {
	t.Helper()
	if len(got) < len(want) {
		t.Errorf("%s: got %d transitions %v, want at least %d %v", stepID, len(got), got, len(want), want)
		return
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("%s: transition[%d] = %q, want %q (full: %v)", stepID, i, got[i], w, got)
			return
		}
	}
}
