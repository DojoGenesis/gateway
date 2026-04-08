package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ---------------------------------------------------------------------------
// WorkflowWSHub -- broadcasts workflow execution events to WebSocket clients
// ---------------------------------------------------------------------------

// WorkflowEvent is the JSON payload sent to WebSocket clients for each step
// lifecycle transition (running, completed, failed, skipped).
type WorkflowEvent struct {
	WorkflowID string `json:"workflow_id"`
	StepID     string `json:"step_id"`
	Status     string `json:"status"`    // "running", "completed", "failed", "skipped"
	Timestamp  string `json:"timestamp"` // RFC 3339
}

// WorkflowWSHub manages connected WebSocket clients and fans out workflow
// execution events to all of them.
type WorkflowWSHub struct {
	mu        sync.RWMutex
	clients   map[*websocket.Conn]bool
	broadcast chan WorkflowEvent
}

// NewWorkflowWSHub creates a hub with a buffered broadcast channel.
func NewWorkflowWSHub() *WorkflowWSHub {
	return &WorkflowWSHub{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan WorkflowEvent, 256),
	}
}

// Run is the background goroutine that reads from the broadcast channel and
// writes each event to every connected client. Clients that fail to receive
// are removed and closed.
func (h *WorkflowWSHub) Run() {
	for evt := range h.broadcast {
		data, err := json.Marshal(evt)
		if err != nil {
			slog.Error("wshub: marshal error", "error", err)
			continue
		}

		h.mu.RLock()
		clients := make([]*websocket.Conn, 0, len(h.clients))
		for c := range h.clients {
			clients = append(clients, c)
		}
		h.mu.RUnlock()

		for _, c := range clients {
			if err := c.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				h.removeClient(c)
				continue
			}
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				slog.Debug("wshub: write failed, removing client", "error", err)
				h.removeClient(c)
			}
		}
	}
}

// upgrader is the WebSocket upgrader shared by all connections. It checks the
// Origin header via a permissive policy; production deployments should tighten
// CheckOrigin to match CORS AllowedOrigins.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // permissive for dev; tighten for production
	},
}

// HandleWS upgrades the HTTP connection to a WebSocket, registers the client
// with the hub, and blocks reading pings/pongs until the client disconnects.
func (h *WorkflowWSHub) HandleWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("wshub: upgrade failed", "error", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	slog.Debug("wshub: client connected", "remote", conn.RemoteAddr())

	// Keep reading until the client disconnects. We don't expect meaningful
	// inbound messages, but the read loop is required to process control
	// frames (ping/pong/close).
	defer func() {
		h.removeClient(conn)
		slog.Debug("wshub: client disconnected", "remote", conn.RemoteAddr())
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			// Normal close or network error -- either way, stop.
			break
		}
	}
}

// Publish creates a WorkflowEvent and sends it to the broadcast channel.
// This method has the same signature as WorkflowExecutor's eventFn callback.
func (h *WorkflowWSHub) Publish(workflowID, stepID, status string) {
	evt := WorkflowEvent{
		WorkflowID: workflowID,
		StepID:     stepID,
		Status:     status,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	select {
	case h.broadcast <- evt:
	default:
		slog.Warn("wshub: broadcast channel full, dropping event",
			"workflow_id", workflowID, "step_id", stepID, "status", status)
	}
}

// ClientCount returns the number of currently connected clients.
func (h *WorkflowWSHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// removeClient removes a connection from the client set and closes it.
func (h *WorkflowWSHub) removeClient(conn *websocket.Conn) {
	h.mu.Lock()
	if h.clients[conn] {
		delete(h.clients, conn)
		conn.Close()
	}
	h.mu.Unlock()
}
