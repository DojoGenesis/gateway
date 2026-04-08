package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/DojoGenesis/gateway/server/agent"
	"github.com/gin-gonic/gin"
)

// AgentHandler handles agent-related HTTP requests.
type AgentHandler struct {
	manager *agent.AgentManager
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(am *agent.AgentManager) *AgentHandler {
	return &AgentHandler{manager: am}
}

func (h *AgentHandler) ListAgents(c *gin.Context) {
	if h.manager == nil {
		respondInternalError(c, "agent manager not initialized")
		return
	}

	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	if pageStr != "" || limitStr != "" {
		page := 1
		limit := 20

		if pageStr != "" {
			parsedPage, err := strconv.Atoi(pageStr)
			if err != nil || parsedPage < 1 {
				respondBadRequest(c, "invalid page parameter")
				return
			}
			page = parsedPage
		}

		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil || parsedLimit < 1 {
				respondBadRequest(c, "invalid limit parameter")
				return
			}
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			limit = parsedLimit
		}

		response, err := h.manager.ListAgentsPaginated(c.Request.Context(), page, limit)
		if err != nil {
			slog.Error("failed to list agents paginated", "error", err)
			respondInternalError(c, "Internal server error")
			return
		}

		c.JSON(http.StatusOK, response)
		return
	}

	agents, err := h.manager.ListAgents(c.Request.Context())
	if err != nil {
		slog.Error("failed to list agents", "error", err)
		respondInternalError(c, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, agents)
}

func (h *AgentHandler) GetAgent(c *gin.Context) {
	if h.manager == nil {
		respondInternalError(c, "agent manager not initialized")
		return
	}

	agentID := c.Param("id")
	if agentID == "" {
		respondBadRequest(c, "agent ID is required")
		return
	}

	agentResult, err := h.manager.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		if errors.Is(err, agent.ErrAgentNotFound) {
			respondNotFound(c, "agent")
			return
		}
		slog.Error("failed to get agent", "error", err, "agent_id", agentID)
		respondInternalError(c, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, agentResult)
}

func (h *AgentHandler) GetAgentCapabilities(c *gin.Context) {
	if h.manager == nil {
		respondInternalError(c, "agent manager not initialized")
		return
	}

	agentID := c.Param("id")
	if agentID == "" {
		respondBadRequest(c, "agent ID is required")
		return
	}

	capabilities, err := h.manager.GetAgentCapabilities(c.Request.Context(), agentID)
	if err != nil {
		if errors.Is(err, agent.ErrAgentNotFound) {
			respondNotFound(c, "agent")
			return
		}
		slog.Error("failed to get agent capabilities", "error", err, "agent_id", agentID)
		respondInternalError(c, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, capabilities)
}

func (h *AgentHandler) SeedAgents(c *gin.Context) {
	if h.manager == nil {
		respondInternalError(c, "agent manager not initialized")
		return
	}

	if err := h.manager.SeedDefaultAgents(c.Request.Context()); err != nil {
		slog.Error("failed to seed default agents", "error", err)
		respondInternalError(c, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "default agents seeded successfully"})
}
