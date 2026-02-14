package server

import (
	"net/http"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
	"github.com/gin-gonic/gin"
)

// ─── Tool API Types ──────────────────────────────────────────────────────────

// ToolResponse represents a tool in the API response.
type ToolResponse struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolListResponse is the response for GET /v1/tools.
type ToolListResponse struct {
	Tools []ToolResponse `json:"tools"`
	Count int            `json:"count"`
}

// ToolInvokeRequest is the request body for POST /v1/tools/:name/invoke.
type ToolInvokeRequest struct {
	Inputs    map[string]interface{} `json:"inputs"`
	SessionID string                 `json:"session_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
}

// ToolInvokeResponse is the response for POST /v1/tools/:name/invoke.
type ToolInvokeResponse struct {
	ToolName   string                 `json:"tool_name"`
	Inputs     map[string]interface{} `json:"inputs"`
	Output     interface{}            `json:"output"`
	DurationMs int64                  `json:"duration_ms"`
	Status     string                 `json:"status"`
}

// handleListTools handles GET /v1/tools.
func (s *Server) handleListTools(c *gin.Context) {
	allTools := tools.GetAllTools()

	toolList := make([]ToolResponse, 0, len(allTools))
	for _, tool := range allTools {
		toolList = append(toolList, ToolResponse{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}

	c.JSON(http.StatusOK, ToolListResponse{
		Tools: toolList,
		Count: len(toolList),
	})
}

// handleGetTool handles GET /v1/tools/:name.
func (s *Server) handleGetTool(c *gin.Context) {
	toolName := c.Param("name")
	if toolName == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Tool name is required")
		return
	}

	result, err := tools.GetToolInfo(c.Request.Context(), map[string]interface{}{
		"tool_name": toolName,
	})
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", err.Error())
		return
	}

	success, _ := result["success"].(bool)
	if !success {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Tool not found: "+toolName)
		return
	}

	c.JSON(http.StatusOK, result)
}

// handleInvokeTool handles POST /v1/tools/:name/invoke.
func (s *Server) handleInvokeTool(c *gin.Context) {
	toolName := c.Param("name")
	if toolName == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Tool name is required")
		return
	}

	var req ToolInvokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body: "+err.Error())
		return
	}

	if req.Inputs == nil {
		req.Inputs = make(map[string]interface{})
	}

	start := time.Now()
	result, err := tools.InvokeTool(c.Request.Context(), toolName, req.Inputs)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		s.errorResponseWithDetails(c, http.StatusInternalServerError, "tool_execution_failed",
			"Tool execution failed: "+err.Error(),
			gin.H{
				"tool_name":   toolName,
				"inputs":      req.Inputs,
				"duration_ms": durationMs,
			})
		return
	}

	c.JSON(http.StatusOK, ToolInvokeResponse{
		ToolName:   toolName,
		Inputs:     req.Inputs,
		Output:     result,
		DurationMs: durationMs,
		Status:     "success",
	})
}
