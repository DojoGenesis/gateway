package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newWSTestServer returns an httptest.Server whose only route is the WebSocket
// handler backed by the given hub. Callers must defer ts.Close().
func newWSTestServer(hub *WorkflowWSHub) *httptest.Server {
	r := gin.New()
	r.GET("/api/ws/workflow", hub.HandleWS)
	return httptest.NewServer(r)
}

// wsURL converts an httptest.Server URL from http:// to ws://.
func wsURL(ts *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(ts.URL, "http") + path
}

// dialWS opens a WebSocket connection to the test server.
func dialWS(t *testing.T, ts *httptest.Server) *websocket.Conn {
	t.Helper()
	d := websocket.Dialer{}
	conn, resp, err := d.Dial(wsURL(ts, "/api/ws/workflow"), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("unexpected status %d, want %d", resp.StatusCode, http.StatusSwitchingProtocols)
	}
	return conn
}

// TestWebSocketUpgrade verifies that a client can upgrade to WebSocket.
func TestWebSocketUpgrade(t *testing.T) {
	hub := NewWorkflowWSHub()
	go hub.Run()

	ts := newWSTestServer(hub)
	defer ts.Close()

	conn := dialWS(t, ts)
	defer conn.Close()

	if hub.ClientCount() != 1 {
		t.Errorf("client count = %d, want 1", hub.ClientCount())
	}
}

// TestWebSocketPublishReceive verifies that a published event reaches connected
// WebSocket clients as a JSON message.
func TestWebSocketPublishReceive(t *testing.T) {
	hub := NewWorkflowWSHub()
	go hub.Run()

	ts := newWSTestServer(hub)
	defer ts.Close()

	conn := dialWS(t, ts)
	defer conn.Close()

	// Give the client registration a moment to propagate.
	time.Sleep(20 * time.Millisecond)

	hub.Publish("wf-1", "step-a", "running")

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var evt WorkflowEvent
	if err := json.Unmarshal(msg, &evt); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if evt.WorkflowID != "wf-1" {
		t.Errorf("workflow_id = %q, want %q", evt.WorkflowID, "wf-1")
	}
	if evt.StepID != "step-a" {
		t.Errorf("step_id = %q, want %q", evt.StepID, "step-a")
	}
	if evt.Status != "running" {
		t.Errorf("status = %q, want %q", evt.Status, "running")
	}
	if evt.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

// TestWebSocketMultipleClients verifies fan-out to multiple clients.
func TestWebSocketMultipleClients(t *testing.T) {
	hub := NewWorkflowWSHub()
	go hub.Run()

	ts := newWSTestServer(hub)
	defer ts.Close()

	conn1 := dialWS(t, ts)
	defer conn1.Close()
	conn2 := dialWS(t, ts)
	defer conn2.Close()

	time.Sleep(20 * time.Millisecond)

	if hub.ClientCount() != 2 {
		t.Fatalf("client count = %d, want 2", hub.ClientCount())
	}

	hub.Publish("wf-2", "step-b", "completed")

	for i, conn := range []*websocket.Conn{conn1, conn2} {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("client %d: read failed: %v", i, err)
		}
		var evt WorkflowEvent
		json.Unmarshal(msg, &evt)
		if evt.StepID != "step-b" {
			t.Errorf("client %d: step_id = %q, want %q", i, evt.StepID, "step-b")
		}
	}
}

// TestWebSocketDisconnectCleanup verifies that closing a connection removes it
// from the hub's client set.
func TestWebSocketDisconnectCleanup(t *testing.T) {
	hub := NewWorkflowWSHub()
	go hub.Run()

	ts := newWSTestServer(hub)
	defer ts.Close()

	conn := dialWS(t, ts)

	time.Sleep(20 * time.Millisecond)
	if hub.ClientCount() != 1 {
		t.Fatalf("client count = %d before close, want 1", hub.ClientCount())
	}

	// Close the client connection.
	conn.Close()

	// Publish an event -- the hub's write attempt will detect the closed
	// connection and remove it.
	hub.Publish("wf-3", "step-c", "failed")

	// Allow the broadcast loop time to process the write failure.
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("client count = %d after close, want 0", hub.ClientCount())
	}
}

// TestWorkflowWSHubPublishNoClients ensures publishing with no clients does not
// block or panic.
func TestWorkflowWSHubPublishNoClients(t *testing.T) {
	hub := NewWorkflowWSHub()
	go hub.Run()

	// Should not block or panic.
	hub.Publish("wf-0", "step-0", "running")

	// Brief sleep to let the broadcast loop drain.
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("client count = %d, want 0", hub.ClientCount())
	}
}
