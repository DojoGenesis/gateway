package handlers

import (
	"log/slog"
	"net/http"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/models"
	"github.com/gin-gonic/gin"
)

// HandleBroadcast receives events and routes them to the appropriate SSE
// client connection. This endpoint can be called by internal services or
// external integrations when streaming events need to be sent to frontend clients.
//
// The handler is event-agnostic by design - it accepts and broadcasts all
// StreamEvent types uniformly without requiring event-type-specific handlers.
// This includes standard events (intent_classified, tool_invoked, etc.) as well
// as trace events (trace_span_start, trace_span_end) introduced in v0.0.17.
//
// The handler implements non-blocking sends - if a client's buffer is full
// or the client has disconnected, it logs a warning but returns 200 OK to
// prevent callers from blocking on failed broadcasts.
func HandleBroadcast(c *gin.Context) {
	var req models.BroadcastRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("broadcast: invalid request payload", "error", err)
		respondBadRequest(c, "Invalid request", err.Error())
		return
	}

	err := SendToClient(req.ClientID, req.Event, req.Data)
	if err != nil {
		slog.Warn("broadcast delivery failed", "error", err, "event", req.Event)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"client_id": req.ClientID,
		"delivered": err == nil,
	})
}
