package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/DojoGenesis/gateway/server/database"
)

// ─── Request / Response types ────────────────────────────────────────────────

type createTemplateRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
	IsPublic     bool   `json:"is_public"`
}

type updateTemplateRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
	IsPublic     bool   `json:"is_public"`
}

// ─── Template Handlers ───────────────────────────────────────────────────────

// handleListTemplates handles GET /api/templates.
// Returns all templates owned by the user plus all public system templates.
// Pass ?public=false to suppress public templates.
func (s *Server) handleListTemplates(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	includePublic := true
	if pub := c.Query("public"); pub == "false" {
		includePublic = false
	}

	adapter := database.NewLocalAdapter(s.authDB)
	tmpls, err := adapter.ListPromptTemplates(c.Request.Context(), userID, includePublic)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to list templates")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": tmpls,
		"count":     len(tmpls),
	})
}

// handleGetTemplate handles GET /api/templates/:id.
// Returns the template if the caller owns it or the template is public.
func (s *Server) handleGetTemplate(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	tmplID := c.Param("id")
	if tmplID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Template ID is required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)
	tmpl, err := adapter.GetPromptTemplate(c.Request.Context(), tmplID)
	if err != nil {
		if errors.Is(err, database.ErrTemplateNotFound) {
			s.errorResponse(c, http.StatusNotFound, "not_found", "Template not found")
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to get template")
		return
	}

	// Allow access if the caller owns it or it is public.
	if tmpl.UserID != userID && !tmpl.IsPublic {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	c.JSON(http.StatusOK, tmpl)
}

// handleCreateTemplate handles POST /api/templates.
func (s *Server) handleCreateTemplate(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req createTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.Title == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "title is required")
		return
	}
	if req.SystemPrompt == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "system_prompt is required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	now := time.Now()
	tmpl := &database.PromptTemplate{
		ID:           uuid.New().String(),
		UserID:       userID,
		Title:        req.Title,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		IsPublic:     req.IsPublic,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	adapter := database.NewLocalAdapter(s.authDB)
	if err := adapter.CreatePromptTemplate(c.Request.Context(), tmpl); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to create template")
		return
	}

	c.JSON(http.StatusCreated, tmpl)
}

// handleUpdateTemplate handles PUT /api/templates/:id.
// Only the template owner can update it. Public system templates (user_id = "system") are immutable.
func (s *Server) handleUpdateTemplate(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	tmplID := c.Param("id")
	if tmplID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Template ID is required")
		return
	}

	var req updateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.Title == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "title is required")
		return
	}
	if req.SystemPrompt == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "system_prompt is required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)

	// Verify ownership — system templates and other users' templates are read-only.
	existing, err := adapter.GetPromptTemplate(c.Request.Context(), tmplID)
	if err != nil {
		if errors.Is(err, database.ErrTemplateNotFound) {
			s.errorResponse(c, http.StatusNotFound, "not_found", "Template not found")
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to get template")
		return
	}

	if existing.UserID != userID {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	existing.Title = req.Title
	existing.Description = req.Description
	existing.SystemPrompt = req.SystemPrompt
	existing.IsPublic = req.IsPublic
	existing.UpdatedAt = time.Now()

	if err := adapter.UpdatePromptTemplate(c.Request.Context(), existing); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to update template")
		return
	}

	c.JSON(http.StatusOK, existing)
}

// handleDeleteTemplate handles DELETE /api/templates/:id.
// Only the template owner can delete it.
func (s *Server) handleDeleteTemplate(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	tmplID := c.Param("id")
	if tmplID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Template ID is required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)

	// Verify ownership before deleting.
	tmpl, err := adapter.GetPromptTemplate(c.Request.Context(), tmplID)
	if err != nil {
		if errors.Is(err, database.ErrTemplateNotFound) {
			s.errorResponse(c, http.StatusNotFound, "not_found", "Template not found")
			return
		}
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to get template")
		return
	}

	if tmpl.UserID != userID {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	if err := adapter.DeletePromptTemplate(c.Request.Context(), tmplID); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to delete template")
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": tmplID})
}
