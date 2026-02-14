package handlers

import (
	"net/http"
	"strconv"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/gin-gonic/gin"
)

var agentManager *agent.AgentManager

func InitializeAgentHandlers(am *agent.AgentManager) {
	agentManager = am
}

func HandleListAgents(c *gin.Context) {
	if agentManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent manager not initialized"})
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
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page parameter"})
				return
			}
			page = parsedPage
		}

		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil || parsedLimit < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
				return
			}
			limit = parsedLimit
		}

		response, err := agentManager.ListAgentsPaginated(c.Request.Context(), page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
		return
	}

	agents, err := agentManager.ListAgents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agents)
}

func HandleGetAgent(c *gin.Context) {
	if agentManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent manager not initialized"})
		return
	}

	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent ID is required"})
		return
	}

	agent, err := agentManager.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

func HandleGetAgentCapabilities(c *gin.Context) {
	if agentManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent manager not initialized"})
		return
	}

	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent ID is required"})
		return
	}

	capabilities, err := agentManager.GetAgentCapabilities(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, capabilities)
}

func HandleSeedAgents(c *gin.Context) {
	if agentManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent manager not initialized"})
		return
	}

	if err := agentManager.SeedDefaultAgents(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "default agents seeded successfully"})
}
