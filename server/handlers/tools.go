package handlers

import (
	"log/slog"
	"net/http"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
	"github.com/gin-gonic/gin"
)

type SearchToolsRequest struct {
	Query      string `json:"query,omitempty"`
	Category   string `json:"category,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type InvokeToolRequest struct {
	ToolName  string                 `json:"tool_name"`
	Params    map[string]interface{} `json:"params"`
	SessionID string                 `json:"session_id,omitempty"`
}

func HandleListTools(c *gin.Context) {
	allTools := tools.GetAllTools()

	toolList := make([]map[string]interface{}, 0, len(allTools))
	for _, tool := range allTools {
		toolList = append(toolList, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(toolList),
		"tools":   toolList,
	})
}

func HandleSearchTools(c *gin.Context) {
	var req SearchToolsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
		return
	}

	params := make(map[string]interface{})
	if req.Query != "" {
		params["query"] = req.Query
	}
	if req.Category != "" {
		params["category"] = req.Category
	}
	if req.MaxResults > 0 {
		params["max_results"] = req.MaxResults
	}

	result, err := tools.SearchTools(c.Request.Context(), params)
	if err != nil {
		slog.Error("failed to search tools", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to search tools")
		return
	}

	c.JSON(http.StatusOK, result)
}

func HandleGetToolInfo(c *gin.Context) {
	toolName := c.Param("name")
	if toolName == "" {
		respondBadRequestWithSuccess(c, "Tool name is required")
		return
	}

	params := map[string]interface{}{
		"tool_name": toolName,
	}

	result, err := tools.GetToolInfo(c.Request.Context(), params)
	if err != nil {
		slog.Error("failed to get tool info", "error", err, "tool", toolName)
		respondInternalErrorWithSuccess(c, "Failed to get tool info")
		return
	}

	success, _ := result["success"].(bool)
	if !success {
		c.JSON(http.StatusNotFound, result)
		return
	}

	c.JSON(http.StatusOK, result)
}

func HandleInvokeTool(c *gin.Context) {
	var req InvokeToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
		return
	}

	if req.ToolName == "" {
		respondBadRequestWithSuccess(c, "tool_name is required")
		return
	}

	if req.Params == nil {
		req.Params = make(map[string]interface{})
	}

	result, err := tools.InvokeTool(c.Request.Context(), req.ToolName, req.Params)
	if err != nil {
		slog.Error("failed to invoke tool", "error", err, "tool", req.ToolName)
		respondInternalErrorWithSuccess(c, "Failed to invoke tool")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}
