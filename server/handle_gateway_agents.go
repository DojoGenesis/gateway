package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ─── Agent Listing with Filters (Gap 4) ─────────────────────────────────────
//
// GET /v1/gateway/agents?status=active|inactive|all&model=X&limit=N&offset=N
//
// This handler replaces the basic handleGatewayListAgents in handle_gateway.go
// with a version that supports filtering and pagination.

// handleGatewayListAgentsFiltered returns registered agents with optional filters.
// GET /v1/gateway/agents?status=active|inactive|all&model=X&limit=N&offset=N
func (s *Server) handleGatewayListAgentsFiltered(c *gin.Context) {
	statusFilter := c.DefaultQuery("status", "all")
	modelFilter := c.Query("model")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		limit = 50
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Validate status filter
	if statusFilter != "active" && statusFilter != "inactive" && statusFilter != "all" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request",
			"Status must be one of: active, inactive, all")
		return
	}

	s.agentMu.RLock()
	defer s.agentMu.RUnlock()

	// Build filtered list
	type agentEntry struct {
		AgentID     string   `json:"agent_id"`
		Status      string   `json:"status"`
		Model       string   `json:"model,omitempty"`
		Channels    []string `json:"channels,omitempty"`
		Disposition gin.H    `json:"disposition,omitempty"`
	}

	allAgents := make([]agentEntry, 0, len(s.agents))
	for id, runtime := range s.agents {
		// Determine agent status
		agentStatus := "active"
		if runtime.Config == nil {
			agentStatus = "inactive"
		}

		// Apply status filter
		if statusFilter != "all" && agentStatus != statusFilter {
			continue
		}

		// Determine model from config
		agentModel := ""
		if runtime.Config != nil {
			agentModel = runtime.Config.Pacing // disposition pacing as proxy; model routing is in provider layer
		}

		// Apply model filter
		if modelFilter != "" && agentModel != modelFilter {
			continue
		}

		entry := agentEntry{
			AgentID:  id,
			Status:   agentStatus,
			Model:    agentModel,
			Channels: runtime.Channels,
		}

		// Include disposition summary if available
		if runtime.Disposition != nil {
			entry.Disposition = gin.H{
				"pacing":     runtime.Disposition.Pacing,
				"depth":      runtime.Disposition.Depth,
				"tone":       runtime.Disposition.Tone,
				"initiative": runtime.Disposition.Initiative,
			}
		}

		allAgents = append(allAgents, entry)
	}

	total := len(allAgents)

	// Apply pagination
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	paginated := allAgents[offset:end]

	c.JSON(http.StatusOK, gin.H{
		"agents": paginated,
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"filters": gin.H{
			"status": statusFilter,
			"model":  modelFilter,
		},
	})
}

// handleGatewayGetAgentDetail retrieves a single agent by ID with full details.
// GET /v1/gateway/agents/:id
func (s *Server) handleGatewayGetAgentDetail(c *gin.Context) {
	agentID := c.Param("id")

	s.agentMu.RLock()
	runtime, exists := s.agents[agentID]
	s.agentMu.RUnlock()

	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found",
			fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

	response := gin.H{
		"agent_id": agentID,
		"status":   "active",
	}

	if runtime.Config != nil {
		response["config"] = runtime.Config
		response["model"] = runtime.Config.Pacing // disposition field; model routing handled by provider layer
	} else {
		response["status"] = "inactive"
	}

	if runtime.Disposition != nil {
		response["disposition"] = gin.H{
			"pacing":     runtime.Disposition.Pacing,
			"depth":      runtime.Disposition.Depth,
			"tone":       runtime.Disposition.Tone,
			"initiative": runtime.Disposition.Initiative,
		}
	}

	if len(runtime.Channels) > 0 {
		response["channels"] = runtime.Channels
	}

	c.JSON(http.StatusOK, response)
}
