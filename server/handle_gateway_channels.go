package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ─── Agent-Channel Binding (Gap 5) ──────────────────────────────────────────
//
// These handlers manage the binding between agents and communication channels.
// Channels are stored in-memory on AgentRuntime.Channels []string.
//
// Routes:
//   POST   /v1/gateway/agents/:id/channels          - Bind channels to agent
//   GET    /v1/gateway/agents/:id/channels          - List agent's channels
//   DELETE /v1/gateway/agents/:id/channels/:channel  - Remove a channel binding

// handleGatewayListAgentChannels returns the channels bound to an agent.
// GET /v1/gateway/agents/:id/channels
func (s *Server) handleGatewayListAgentChannels(c *gin.Context) {
	agentID := c.Param("id")

	// Copy the Channels slice while the read lock is held.  Releasing the lock
	// before reading runtime.Channels creates a data race against concurrent
	// writers in handleGatewayBindAgentChannels (write lock + slice append).
	var channels []string
	var exists bool
	s.agentMu.RLock()
	runtime, exists := s.agents[agentID]
	if exists {
		if len(runtime.Channels) > 0 {
			channels = make([]string, len(runtime.Channels))
			copy(channels, runtime.Channels)
		} else {
			channels = []string{}
		}
	}
	s.agentMu.RUnlock()

	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found",
			fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"channels": channels,
		"count":    len(channels),
	})
}

// handleGatewayUnbindAgentChannel removes a single channel from an agent's bindings.
// DELETE /v1/gateway/agents/:id/channels/:channel
func (s *Server) handleGatewayUnbindAgentChannel(c *gin.Context) {
	agentID := c.Param("id")
	channel := c.Param("channel")

	s.agentMu.Lock()
	runtime, exists := s.agents[agentID]
	if !exists {
		s.agentMu.Unlock()
		s.errorResponse(c, http.StatusNotFound, "not_found",
			fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

	// Find and remove the channel
	found := false
	newChannels := make([]string, 0, len(runtime.Channels))
	for _, ch := range runtime.Channels {
		if ch == channel {
			found = true
			continue
		}
		newChannels = append(newChannels, ch)
	}

	if !found {
		s.agentMu.Unlock()
		s.errorResponse(c, http.StatusNotFound, "not_found",
			fmt.Sprintf("Channel %q not bound to agent %s", channel, agentID))
		return
	}

	runtime.Channels = newChannels
	s.agentMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"agent_id":        agentID,
		"removed_channel": channel,
		"channels":        newChannels,
		"count":           len(newChannels),
	})
}
