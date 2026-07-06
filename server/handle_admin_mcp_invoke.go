package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/mcp"
	"github.com/DojoGenesis/gateway/tools"
)

// MCPToolInvokeRequest is the request body for POST /admin/mcp/tools/invoke.
type MCPToolInvokeRequest struct {
	Server    string                 `json:"server" binding:"required"`
	Tool      string                 `json:"tool" binding:"required"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPToolInvokeResponse is the response for POST /admin/mcp/tools/invoke.
type MCPToolInvokeResponse struct {
	Server     string                 `json:"server"`
	Tool       string                 `json:"tool"`
	Result     map[string]interface{} `json:"result"`
	DurationMs int64                  `json:"duration_ms"`
}

// handleAdminInvokeMCPTool handles POST /admin/mcp/tools/invoke.
//
// This is an explicit, admin-gated, MCP-aware direct-invoke endpoint. Unlike
// the generic POST /v1/tools/:name/invoke endpoint, callers here pass the MCP
// server name and tool name as separate fields rather than needing to know
// the internal "prefix:toolname" registry-key convention, and the route is
// unreachable without admin auth (it lives under the existing /admin group
// wrapped by middleware.AdminAuthMiddleware()).
func (s *Server) handleAdminInvokeMCPTool(c *gin.Context) {
	var req MCPToolInvokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// SECURITY GUARD (load-bearing): the registry key is built by joining
	// server and tool with a colon below. Without this check, a caller could
	// pass a tool value like "othernamespace:sensitive_tool" and address an
	// unintended namespaced registry entry. Must run before any lookup.
	if strings.Contains(req.Server, ":") || strings.Contains(req.Tool, ":") {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "server and tool must not contain a colon character")
		return
	}

	namespacedName := req.Server + ":" + req.Tool

	if s.toolRegistry == nil {
		s.errorResponseWithDetails(c, http.StatusNotFound, "mcp_tool_not_found",
			"MCP tool not found: "+namespacedName,
			gin.H{
				"server": req.Server,
				"tool":   req.Tool,
			})
		return
	}

	toolDef, err := s.toolRegistry.Get(c.Request.Context(), namespacedName)
	if err != nil || toolDef == nil {
		s.errorResponseWithDetails(c, http.StatusNotFound, "mcp_tool_not_found",
			"MCP tool not found: "+namespacedName,
			gin.H{
				"server": req.Server,
				"tool":   req.Tool,
			})
		return
	}

	if err := tools.ValidateParameters(req.Arguments, toolDef.Parameters); err != nil {
		s.errorResponseWithDetails(c, http.StatusBadRequest, "invalid_arguments", err.Error(),
			gin.H{
				"server": req.Server,
				"tool":   req.Tool,
			})
		return
	}

	if s.mcpHostManager == nil {
		s.errorResponseWithDetails(c, http.StatusNotFound, "mcp_server_not_found",
			"MCP host manager not configured",
			gin.H{
				"server": req.Server,
				"tool":   req.Tool,
			})
		return
	}

	start := time.Now()
	result, err := s.mcpHostManager.CallTool(c.Request.Context(), req.Server, req.Tool, req.Arguments)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		switch {
		case errors.Is(err, mcp.ErrServerNotFound):
			s.errorResponseWithDetails(c, http.StatusNotFound, "mcp_server_not_found", err.Error(),
				gin.H{
					"server": req.Server,
					"tool":   req.Tool,
				})
		case errors.Is(err, mcp.ErrServerUnhealthy):
			s.errorResponseWithDetails(c, http.StatusServiceUnavailable, "mcp_server_unavailable", err.Error(),
				gin.H{
					"server": req.Server,
					"tool":   req.Tool,
				})
		default:
			s.errorResponseWithDetails(c, http.StatusBadGateway, "mcp_tool_execution_failed", err.Error(),
				gin.H{
					"server": req.Server,
					"tool":   req.Tool,
				})
		}
		return
	}

	c.JSON(http.StatusOK, MCPToolInvokeResponse{
		Server:     req.Server,
		Tool:       req.Tool,
		Result:     result,
		DurationMs: durationMs,
	})
}
