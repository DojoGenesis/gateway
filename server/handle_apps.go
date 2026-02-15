package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/apps"
)

// handleGetResource serves a UI resource by URI with security headers.
// GET /v1/gateway/resources?uri=ui://...
func (s *Server) handleGetResource(c *gin.Context) {
	if s.appManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "mcp_apps_disabled", "MCP Apps not enabled")
		return
	}

	uri := c.Query("uri")
	if uri == "" {
		s.errorResponse(c, http.StatusBadRequest, "missing_uri", "uri query parameter is required")
		return
	}
	if !strings.HasPrefix(uri, "ui://") {
		s.errorResponse(c, http.StatusBadRequest, "invalid_uri", "URI must have ui:// scheme")
		return
	}

	meta, err := s.appManager.GetResource(uri)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "resource_not_found", err.Error())
		return
	}

	// Inject security headers
	s.appManager.SecurityPolicy().InjectSecurityHeaders(c.Writer, meta)

	// Set caching headers
	if meta.CacheKey != "" {
		c.Header("ETag", `"`+meta.CacheKey+`"`)
		c.Header("Cache-Control", "public, max-age=3600")

		// Handle If-None-Match for 304
		if match := c.GetHeader("If-None-Match"); match == `"`+meta.CacheKey+`"` {
			c.Status(http.StatusNotModified)
			c.Writer.WriteHeaderNow()
			return
		}
	}

	mimeType := meta.MimeType
	if mimeType == "" {
		mimeType = "text/html"
	}

	c.Data(http.StatusOK, mimeType, meta.Content)
}

// handleLaunchApp launches a new app instance.
// POST /v1/gateway/apps/launch
func (s *Server) handleLaunchApp(c *gin.Context) {
	if s.appManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "mcp_apps_disabled", "MCP Apps not enabled")
		return
	}

	var req struct {
		ResourceURI string `json:"resource_uri"`
		SessionID   string `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.ResourceURI == "" || req.SessionID == "" {
		s.errorResponse(c, http.StatusBadRequest, "missing_fields", "resource_uri and session_id are required")
		return
	}

	inst, err := s.appManager.LaunchApp(req.ResourceURI, req.SessionID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "launch_failed", err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"instance_id":  inst.ID,
		"resource_uri": inst.ResourceURI,
		"launched_at":  inst.LaunchedAt,
	})
}

// handleCloseApp closes an app instance.
// POST /v1/gateway/apps/close
func (s *Server) handleCloseApp(c *gin.Context) {
	if s.appManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "mcp_apps_disabled", "MCP Apps not enabled")
		return
	}

	var req struct {
		InstanceID string `json:"instance_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.InstanceID == "" {
		s.errorResponse(c, http.StatusBadRequest, "missing_fields", "instance_id is required")
		return
	}

	if err := s.appManager.CloseApp(req.InstanceID); err != nil {
		s.errorResponse(c, http.StatusNotFound, "close_failed", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// handleListApps lists app instances for a session.
// GET /v1/gateway/apps?session_id=...
func (s *Server) handleListApps(c *gin.Context) {
	if s.appManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "mcp_apps_disabled", "MCP Apps not enabled")
		return
	}

	sessionID := c.Query("session_id")
	instances := s.appManager.ListApps(sessionID)

	if instances == nil {
		instances = []*apps.AppInstance{}
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": instances,
		"count":     len(instances),
	})
}

// handleProxyToolCall proxies a tool call from an app.
// POST /v1/gateway/apps/tool-call
func (s *Server) handleProxyToolCall(c *gin.Context) {
	if s.appManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "mcp_apps_disabled", "MCP Apps not enabled")
		return
	}

	var req apps.ToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.AppID == "" || req.ToolName == "" {
		s.errorResponse(c, http.StatusBadRequest, "missing_fields", "app_id and tool_name are required")
		return
	}

	resp, err := s.appManager.ProxyToolCall(c.Request.Context(), &req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "proxy_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleAppStatus returns the app manager's status.
// GET /v1/gateway/apps/status
func (s *Server) handleAppStatus(c *gin.Context) {
	if s.appManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "mcp_apps_disabled", "MCP Apps not enabled")
		return
	}

	c.JSON(http.StatusOK, s.appManager.Status())
}
