package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/gin-gonic/gin"
)

var seedManager *memory.SeedManager

func InitializeMemorySeedHandlers(sm *memory.SeedManager) {
	seedManager = sm
}

type MemorySeedResponse struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Content      string    `json:"content"`
	UserEditable bool      `json:"user_editable"`
	Source       string    `json:"source"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func convertToSeedResponse(seed *memory.MemorySeed) MemorySeedResponse {
	projectID := ""
	if seed.ProjectID != nil {
		projectID = *seed.ProjectID
	}
	return MemorySeedResponse{
		ID:           seed.ID,
		ProjectID:    projectID,
		Content:      seed.Content,
		UserEditable: seed.UserEditable,
		Source:       string(seed.Source),
		CreatedAt:    seed.CreatedAt,
		UpdatedAt:    seed.UpdatedAt,
	}
}

type GetMemorySeedsResponse struct {
	Seeds []MemorySeedResponse `json:"seeds"`
	Count int                  `json:"count"`
}

func HandleGetMemorySeeds(c *gin.Context) {
	if seedManager == nil {
		respondError(c, http.StatusInternalServerError, "seed manager not initialized")
		return
	}

	projectID := c.Param("project_id")
	if projectID == "" {
		respondError(c, http.StatusBadRequest, "project_id is required")
		return
	}

	seeds, err := seedManager.GetSeeds(c.Request.Context(), &projectID, map[string]interface{}{})
	if err != nil {
		if errors.Is(err, memory.ErrProjectNotFound) {
			respondError(c, http.StatusNotFound, "Project not found")
			return
		}
		log.Printf("ERROR [MemorySeedHandler]: Failed to get seeds for project %s: %v", projectID, err)
		respondError(c, http.StatusInternalServerError, "Failed to retrieve memory seeds")
		return
	}

	response := make([]MemorySeedResponse, len(seeds))
	for i, seed := range seeds {
		response[i] = convertToSeedResponse(&seed)
	}

	c.JSON(http.StatusOK, GetMemorySeedsResponse{
		Seeds: response,
		Count: len(response),
	})
}

func HandleGetMemorySeed(c *gin.Context) {
	if seedManager == nil {
		respondError(c, http.StatusInternalServerError, "seed manager not initialized")
		return
	}

	seedID := c.Param("id")
	if seedID == "" {
		respondError(c, http.StatusBadRequest, "seed_id is required")
		return
	}

	seed, err := seedManager.GetSeedByID(c.Request.Context(), seedID)
	if err != nil {
		if errors.Is(err, memory.ErrSeedNotFound) {
			respondError(c, http.StatusNotFound, "Seed not found")
			return
		}
		log.Printf("ERROR [MemorySeedHandler]: Failed to get seed %s: %v", seedID, err)
		respondError(c, http.StatusInternalServerError, "Failed to retrieve seed")
		return
	}

	c.JSON(http.StatusOK, convertToSeedResponse(seed))
}

type UpdateMemorySeedRequest struct {
	Content string `json:"content" binding:"required,min=1,max=10240"`
}

type SeedOperationResponse struct {
	Status  string              `json:"status"`
	Seed    *MemorySeedResponse `json:"seed,omitempty"`
	Message string              `json:"message,omitempty"`
}

func HandleUpdateMemorySeed(c *gin.Context) {
	if seedManager == nil {
		respondError(c, http.StatusInternalServerError, "seed manager not initialized")
		return
	}

	seedID := c.Param("id")
	if seedID == "" {
		respondError(c, http.StatusBadRequest, "seed_id is required")
		return
	}

	var req UpdateMemorySeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	seed, err := seedManager.UpdateSeed(c.Request.Context(), seedID, req.Content, &userID)
	if err != nil {
		if errors.Is(err, memory.ErrSeedNotEditable) {
			respondError(c, http.StatusForbidden, "This seed is not editable")
			return
		}
		if errors.Is(err, memory.ErrConcurrentModification) {
			respondError(c, http.StatusConflict, "Seed was modified by another process, please retry")
			return
		}
		if errors.Is(err, memory.ErrInvalidContent) {
			respondError(c, http.StatusBadRequest, "Invalid seed content", err.Error())
			return
		}
		if errors.Is(err, memory.ErrSeedNotFound) {
			respondError(c, http.StatusNotFound, "Seed not found")
			return
		}

		log.Printf("ERROR [MemorySeedHandler]: Failed to update seed %s: %v", seedID, err)
		respondError(c, http.StatusInternalServerError, "Failed to update seed")
		return
	}

	log.Printf("INFO [MemorySeedHandler]: Updated seed %s: user=%s", seedID, userID)

	seedResponse := convertToSeedResponse(seed)
	c.JSON(http.StatusOK, SeedOperationResponse{
		Status: "success",
		Seed:   &seedResponse,
	})
}

type CreateMemorySeedRequest struct {
	Content  string `json:"content" binding:"required,min=1,max=10240"`
	SeedType string `json:"seed_type" binding:"required"`
}

func HandleCreateMemorySeed(c *gin.Context) {
	if seedManager == nil {
		respondError(c, http.StatusInternalServerError, "seed manager not initialized")
		return
	}

	projectID := c.Param("project_id")
	if projectID == "" {
		respondError(c, http.StatusBadRequest, "project_id is required")
		return
	}

	var req CreateMemorySeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	seed, err := seedManager.CreateUserSeed(
		c.Request.Context(),
		&projectID,
		req.Content,
		req.SeedType,
		userID,
	)

	if err != nil {
		if errors.Is(err, memory.ErrProjectNotFound) {
			respondError(c, http.StatusNotFound, "Project not found")
			return
		}
		if errors.Is(err, memory.ErrInvalidContent) {
			respondError(c, http.StatusBadRequest, "Invalid seed content", err.Error())
			return
		}

		log.Printf("ERROR [MemorySeedHandler]: Failed to create seed: project=%s, user=%s, error=%v",
			projectID, userID, err)
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("INFO [MemorySeedHandler]: Created seed %s: project=%s, user=%s", seed.ID, projectID, userID)

	seedResponse := convertToSeedResponse(seed)
	c.Header("Location", fmt.Sprintf("/api/v1/memory/seeds/%s", seed.ID))
	c.JSON(http.StatusCreated, SeedOperationResponse{
		Status: "success",
		Seed:   &seedResponse,
	})
}

func HandleDeleteMemorySeed(c *gin.Context) {
	if seedManager == nil {
		respondError(c, http.StatusInternalServerError, "seed manager not initialized")
		return
	}

	seedID := c.Param("id")
	if seedID == "" {
		respondError(c, http.StatusBadRequest, "seed_id is required")
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	err := seedManager.DeleteSeed(c.Request.Context(), seedID, &userID)
	if err != nil {
		if errors.Is(err, memory.ErrCannotDeleteSystemSeed) {
			respondError(c, http.StatusForbidden, "Cannot delete system seeds")
			return
		}
		if errors.Is(err, memory.ErrSeedNotFound) {
			respondError(c, http.StatusNotFound, "Seed not found")
			return
		}
		if errors.Is(err, memory.ErrPermissionDenied) {
			respondError(c, http.StatusForbidden, "Permission denied: user can only delete their own seeds")
			return
		}

		log.Printf("ERROR [MemorySeedHandler]: Failed to delete seed %s: user=%s, error=%v", seedID, userID, err)
		respondError(c, http.StatusInternalServerError, "Failed to delete seed")
		return
	}

	log.Printf("INFO [MemorySeedHandler]: Deleted seed %s: user=%s", seedID, userID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Seed deleted successfully",
	})
}

// BulkDeleteSeedsRequest defines the request structure for bulk seed deletion
type BulkDeleteSeedsRequest struct {
	SeedIDs []string `json:"seed_ids" binding:"required,min=1,max=100"`
}

// BulkDeleteSeedsResponse defines the response structure for bulk deletion
type BulkDeleteSeedsResponse struct {
	Deleted []string          `json:"deleted"`
	Failed  map[string]string `json:"failed,omitempty"`
}

// HandleBulkDeleteMemorySeeds handles bulk deletion of memory seeds
func HandleBulkDeleteMemorySeeds(c *gin.Context) {
	if seedManager == nil {
		respondError(c, http.StatusInternalServerError, "seed manager not initialized")
		return
	}

	var req BulkDeleteSeedsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default"
	}

	deleted := []string{}
	failed := make(map[string]string)

	for _, seedID := range req.SeedIDs {
		if err := seedManager.DeleteSeed(c.Request.Context(), seedID, &userID); err != nil {
			failed[seedID] = err.Error()
			log.Printf("WARNING [MemorySeedHandler]: Failed to delete seed %s in bulk operation: %v", seedID, err)
		} else {
			deleted = append(deleted, seedID)
		}
	}

	statusCode := http.StatusOK
	if len(deleted) == 0 {
		statusCode = http.StatusBadRequest
	} else if len(failed) > 0 {
		statusCode = http.StatusMultiStatus // 207
	}

	log.Printf("INFO [MemorySeedHandler]: Bulk delete completed: deleted=%d, failed=%d", len(deleted), len(failed))

	c.JSON(statusCode, BulkDeleteSeedsResponse{
		Deleted: deleted,
		Failed:  failed,
	})
}

// SearchSeedsRequest defines the request structure for seed search
type SearchSeedsRequest struct {
	Query string `form:"q" binding:"required,min=2"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// HandleSearchMemorySeeds handles full-text search on memory seeds
func HandleSearchMemorySeeds(c *gin.Context) {
	if seedManager == nil {
		respondError(c, http.StatusInternalServerError, "seed manager not initialized")
		return
	}

	projectID := c.Param("project_id")
	if projectID == "" {
		respondError(c, http.StatusBadRequest, "project_id is required")
		return
	}

	var req SearchSeedsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	seeds, err := seedManager.SearchSeeds(c.Request.Context(), &projectID, req.Query, req.Limit)
	if err != nil {
		log.Printf("ERROR [MemorySeedHandler]: Failed to search seeds: %v", err)
		respondError(c, http.StatusInternalServerError, "Failed to search seeds")
		return
	}

	response := make([]MemorySeedResponse, len(seeds))
	for i, seed := range seeds {
		response[i] = convertToSeedResponse(&seed)
	}

	c.JSON(http.StatusOK, gin.H{
		"seeds": response,
		"count": len(response),
		"query": req.Query,
	})
}
