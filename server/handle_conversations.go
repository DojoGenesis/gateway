package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/DojoGenesis/gateway/server/database"
)

// ─── Request / Response types ───────────────────────────────────────────────

type createConversationRequest struct {
	Title     *string `json:"title"`
	ProjectID *string `json:"project_id"`
	Metadata  *string `json:"metadata"`
}

type createMessageRequest struct {
	Role       string  `json:"role"` // "user", "assistant", "system"
	Content    string  `json:"content"`
	Model      *string `json:"model"`
	Provider   *string `json:"provider"`
	TokensUsed *int    `json:"tokens_used"`
	Metadata   *string `json:"metadata"`
}

// ─── Helpers ────────────────────────────────────────────────────────────────

// getUserIDFromContext extracts the authenticated user ID from the Gin context.
// The OptionalAuthMiddleware sets "user_id" on every request.
func getUserIDFromContext(c *gin.Context) (string, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok && id != ""
}

// ─── Conversation Handlers ──────────────────────────────────────────────────

// handleListConversations handles GET /api/conversations.
// Returns all conversations for the authenticated user.
func (s *Server) handleListConversations(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)
	convs, err := adapter.ListConversations(c.Request.Context(), userID)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to list conversations")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": convs,
		"count":         len(convs),
	})
}

// handleGetConversation handles GET /api/conversations/:id.
// Returns the conversation plus its most recent messages.
func (s *Server) handleGetConversation(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	convID := c.Param("id")
	if convID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Conversation ID is required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)

	conv, err := adapter.GetConversation(c.Request.Context(), convID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Conversation not found")
		return
	}

	// Verify ownership
	if conv.UserID != userID {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	// Fetch recent messages (last 50)
	msgs, err := adapter.ListMessages(c.Request.Context(), convID, 50, 0)
	if err != nil {
		// Non-fatal: return conversation without messages
		msgs = []*database.Message{}
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation": conv,
		"messages":     msgs,
	})
}

// handleCreateConversation handles POST /api/conversations.
func (s *Server) handleCreateConversation(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	now := time.Now()
	conv := &database.Conversation{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     req.Title,
		ProjectID: req.ProjectID,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  req.Metadata,
	}

	adapter := database.NewLocalAdapter(s.authDB)
	if err := adapter.CreateConversation(c.Request.Context(), conv); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to create conversation")
		return
	}

	c.JSON(http.StatusCreated, conv)
}

// handleDeleteConversation handles DELETE /api/conversations/:id.
func (s *Server) handleDeleteConversation(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	convID := c.Param("id")
	if convID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Conversation ID is required")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)

	// Verify ownership before deleting
	conv, err := adapter.GetConversation(c.Request.Context(), convID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Conversation not found")
		return
	}
	if conv.UserID != userID {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	if err := adapter.DeleteConversation(c.Request.Context(), convID); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to delete conversation")
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": convID})
}

// ─── Message Handlers ────────────────────────────────────────────────────────

// handleListMessages handles GET /api/conversations/:id/messages.
// Supports ?limit=N&offset=N pagination.
func (s *Server) handleListMessages(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	convID := c.Param("id")
	if convID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Conversation ID is required")
		return
	}

	limit := 50
	offset := 0
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if oStr := c.Query("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)

	// Verify ownership
	conv, err := adapter.GetConversation(c.Request.Context(), convID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Conversation not found")
		return
	}
	if conv.UserID != userID {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	msgs, err := adapter.ListMessages(c.Request.Context(), convID, limit, offset)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to list messages")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages":        msgs,
		"count":           len(msgs),
		"conversation_id": convID,
		"limit":           limit,
		"offset":          offset,
	})
}

// handleCreateMessage handles POST /api/conversations/:id/messages.
func (s *Server) handleCreateMessage(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		s.errorResponse(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	convID := c.Param("id")
	if convID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Conversation ID is required")
		return
	}

	var req createMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.Content == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "content is required")
		return
	}

	validRoles := map[string]bool{"user": true, "assistant": true, "system": true}
	if !validRoles[req.Role] {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "role must be one of: user, assistant, system")
		return
	}

	if s.authDB == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "db_unavailable", "Database not configured")
		return
	}

	adapter := database.NewLocalAdapter(s.authDB)

	// Verify ownership
	conv, err := adapter.GetConversation(c.Request.Context(), convID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Conversation not found")
		return
	}
	if conv.UserID != userID {
		s.errorResponse(c, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	now := time.Now()
	msg := &database.Message{
		ID:             uuid.New().String(),
		ConversationID: convID,
		Role:           req.Role,
		Content:        req.Content,
		Model:          req.Model,
		Provider:       req.Provider,
		TokensUsed:     req.TokensUsed,
		CreatedAt:      now,
		Metadata:       req.Metadata,
	}

	if err := adapter.CreateMessage(c.Request.Context(), msg); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to create message")
		return
	}

	// Update conversation's message count and last_message_at
	conv.MessageCount++
	conv.LastMessageAt = &now
	conv.UpdatedAt = now
	_ = adapter.UpdateConversation(c.Request.Context(), conv) // best-effort

	c.JSON(http.StatusCreated, msg)
}
