package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/models"
	"github.com/gin-gonic/gin"
)

var (
	clients      = make(map[string]*models.Client)
	clientsMutex sync.RWMutex
)

const channelBufferSize = 100

// HandleSSE handles Server-Sent Events connections from frontend clients.
// Clients connect with a unique client_id query parameter and receive events
// through a long-lived HTTP connection. Each client gets a dedicated goroutine
// and buffered channel for event delivery.
func HandleSSE(c *gin.Context) {
	clientID := c.Query("client_id")
	if clientID == "" {
		respondBadRequest(c, "client_id query parameter is required")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	client := &models.Client{
		ID:      clientID,
		Channel: make(chan string, channelBufferSize),
		Created: time.Now(),
	}

	clientsMutex.Lock()
	if existingClient, exists := clients[clientID]; exists {
		slog.Warn("SSE client already connected, closing old connection", "client_id", clientID)
		close(existingClient.Channel)
	}
	clients[clientID] = client
	clientsMutex.Unlock()

	slog.Info("SSE client connected", "client_id", clientID)

	defer func() {
		clientsMutex.Lock()
		if c, exists := clients[clientID]; exists && c == client {
			delete(clients, clientID)
			close(client.Channel)
		}
		clientsMutex.Unlock()
		slog.Info("SSE client disconnected", "client_id", clientID)
	}()

	c.Writer.WriteHeader(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		slog.Warn("SSE streaming not supported", "client_id", clientID)
		return
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-client.Channel:
			if !ok {
				return
			}
			_, err := fmt.Fprintf(c.Writer, "%s", msg)
			if err != nil {
				slog.Error("SSE write error", "client_id", clientID, "error", err)
				return
			}
			flusher.Flush()
		}
	}
}

// GetConnectedClients returns the count of currently connected clients.
// This is primarily for monitoring and testing purposes.
func GetConnectedClients() int {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	return len(clients)
}

// GetClientIDs returns a list of all currently connected client IDs.
// This is primarily for testing purposes.
func GetClientIDs() []string {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	ids := make([]string, 0, len(clients))
	for id := range clients {
		ids = append(ids, id)
	}
	return ids
}

// SendToClient sends a message to a specific client's channel.
// Returns an error if the client is not found or the channel buffer is full.
// This is used by the broadcast handler to route events to specific clients.
func SendToClient(clientID string, event string, data string) error {
	clientsMutex.RLock()
	client, exists := clients[clientID]
	clientsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	message := formatSSEMessage(event, data)

	select {
	case client.Channel <- message:
		return nil
	default:
		return fmt.Errorf("client %s channel buffer full", clientID)
	}
}

// formatSSEMessage formats an event and data into SSE wire format.
func formatSSEMessage(event string, data string) string {
	msg := ""
	if event != "" {
		msg += fmt.Sprintf("event: %s\n", event)
	}
	if data != "" {
		msg += fmt.Sprintf("data: %s\n", data)
	}
	return msg + "\n"
}

// DisconnectClient forcefully disconnects a client.
// This is primarily for testing and administrative purposes.
func DisconnectClient(clientID string) bool {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	if client, exists := clients[clientID]; exists {
		close(client.Channel)
		delete(clients, clientID)
		slog.Info("SSE client forcefully disconnected", "client_id", clientID)
		return true
	}
	return false
}
