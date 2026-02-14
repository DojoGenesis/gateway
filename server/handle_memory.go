package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ─── Memory API Types ────────────────────────────────────────────────────────

// MemoryStoreRequest is the request body for POST /v1/memory.
type MemoryStoreRequest struct {
	Type        string                 `json:"type" binding:"required"`
	Content     string                 `json:"content" binding:"required"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ContextType string                 `json:"context_type,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
}

// MemoryResponse is the API response for a single memory.
type MemoryResponse struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ContextType string                 `json:"context_type"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// MemoryListResponse is the response for GET /v1/memory.
type MemoryListResponse struct {
	Memories   []MemoryResponse `json:"memories"`
	TotalCount int              `json:"total_count"`
	Limit      int              `json:"limit"`
	Offset     int              `json:"offset"`
}

// MemorySearchRequest is the request body for POST /v1/memory/search.
type MemorySearchRequest struct {
	Query       string `json:"query" binding:"required"`
	Type        string `json:"type,omitempty"`
	ContextType string `json:"context_type,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}

// MemorySearchResponse is the response for POST /v1/memory/search.
type MemorySearchResponse struct {
	Results    []MemorySearchResult `json:"results"`
	TotalCount int                  `json:"total_count"`
}

// MemorySearchResult is a search result with relevance score.
type MemorySearchResult struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	Content        string                 `json:"content"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	RelevanceScore float64                `json:"relevance_score,omitempty"`
}

// MemoryUpdateRequest is the request body for PUT /v1/memory/:id.
type MemoryUpdateRequest struct {
	Content  string                 `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func memoryToResponse(m memory.Memory) MemoryResponse {
	return MemoryResponse{
		ID:          m.ID,
		Type:        m.Type,
		Content:     m.Content,
		Metadata:    m.Metadata,
		ContextType: m.ContextType,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
	}
}

// handleStoreMemory handles POST /v1/memory.
func (s *Server) handleStoreMemory(c *gin.Context) {
	if s.memoryManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "Memory manager not initialized")
		return
	}

	var req MemoryStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request: "+err.Error())
		return
	}

	contextType := req.ContextType
	if contextType == "" {
		contextType = "private"
	}
	if contextType != "private" && contextType != "group" && contextType != "public" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "context_type must be 'private', 'group', or 'public'")
		return
	}

	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	now := time.Now()
	mem := memory.Memory{
		ID:          uuid.New().String(),
		Type:        req.Type,
		Content:     req.Content,
		Metadata:    req.Metadata,
		ContextType: contextType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.memoryManager.Store(c.Request.Context(), mem); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to store memory: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, memoryToResponse(mem))
}

// handleListMemories handles GET /v1/memory.
func (s *Server) handleListMemories(c *gin.Context) {
	if s.memoryManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "Memory manager not initialized")
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}
	if limit > 100 {
		limit = 100
	}

	// Use query param for filtering if provided
	query := c.Query("query")
	sessionID := c.Query("session_id")

	var memories []memory.Memory
	var err error

	if query != "" {
		results, searchErr := s.memoryManager.Search(c.Request.Context(), query, limit)
		if searchErr != nil {
			s.errorResponse(c, http.StatusInternalServerError, "server_error", "Search failed: "+searchErr.Error())
			return
		}
		for _, r := range results {
			memories = append(memories, r.Memory)
		}
	} else {
		memories, err = s.memoryManager.List(c.Request.Context(), sessionID, limit)
		if err != nil {
			s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to list memories: "+err.Error())
			return
		}
	}

	respMemories := make([]MemoryResponse, 0, len(memories))
	for _, m := range memories {
		respMemories = append(respMemories, memoryToResponse(m))
	}

	c.JSON(http.StatusOK, MemoryListResponse{
		Memories:   respMemories,
		TotalCount: len(respMemories),
		Limit:      limit,
		Offset:     offset,
	})
}

// handleGetMemory handles GET /v1/memory/:id.
func (s *Server) handleGetMemory(c *gin.Context) {
	if s.memoryManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "Memory manager not initialized")
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Memory ID is required")
		return
	}

	mem, err := s.memoryManager.Retrieve(c.Request.Context(), memoryID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Memory not found")
		return
	}

	c.JSON(http.StatusOK, memoryToResponse(*mem))
}

// handleUpdateMemory handles PUT /v1/memory/:id.
func (s *Server) handleUpdateMemory(c *gin.Context) {
	if s.memoryManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "Memory manager not initialized")
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Memory ID is required")
		return
	}

	var req MemoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request: "+err.Error())
		return
	}

	if req.Content == "" && req.Metadata == nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "At least one of content or metadata must be provided")
		return
	}

	updatedMem, err := s.memoryManager.Update(c.Request.Context(), memoryID, req.Content, req.Metadata)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Failed to update memory: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, memoryToResponse(*updatedMem))
}

// handleDeleteMemory handles DELETE /v1/memory/:id.
func (s *Server) handleDeleteMemory(c *gin.Context) {
	if s.memoryManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "Memory manager not initialized")
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Memory ID is required")
		return
	}

	if err := s.memoryManager.Delete(c.Request.Context(), memoryID); err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", "Memory not found")
		return
	}

	c.Status(http.StatusNoContent)
}

// handleSearchMemory handles POST /v1/memory/search.
func (s *Server) handleSearchMemory(c *gin.Context) {
	if s.memoryManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "Memory manager not initialized")
		return
	}

	var req MemorySearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request: "+err.Error())
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	results, err := s.memoryManager.Search(c.Request.Context(), req.Query, limit)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Search failed: "+err.Error())
		return
	}

	searchResults := make([]MemorySearchResult, 0, len(results))
	for _, r := range results {
		searchResults = append(searchResults, MemorySearchResult{
			ID:             r.Memory.ID,
			Type:           r.Memory.Type,
			Content:        r.Memory.Content,
			Metadata:       r.Memory.Metadata,
			RelevanceScore: r.Similarity,
		})
	}

	c.JSON(http.StatusOK, MemorySearchResponse{
		Results:    searchResults,
		TotalCount: len(searchResults),
	})
}
