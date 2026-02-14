package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/maintenance"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var memoryManager *memory.MemoryManager
var gardenManager *memory.GardenManager
var memoryMaintenance *maintenance.MemoryMaintenance

func InitializeMemoryHandlers(mm *memory.MemoryManager) {
	memoryManager = mm
}

func InitializeGardenHandlers(gm *memory.GardenManager) {
	gardenManager = gm
}

func InitializeMaintenanceHandlers(mm *maintenance.MemoryMaintenance) {
	memoryMaintenance = mm
}

type StoreMemoryRequest struct {
	Type        string                 `json:"type" binding:"required"`
	Content     string                 `json:"content" binding:"required"`
	Metadata    map[string]interface{} `json:"metadata"`
	ContextType string                 `json:"context_type,omitempty"`
}

type SearchMemoryRequest struct {
	Query      string `json:"query" binding:"required"`
	MaxResults int    `json:"max_results,omitempty"`
}

func validateContextType(contextType string) error {
	validTypes := map[string]bool{
		"private": true,
		"group":   true,
		"public":  true,
	}

	if contextType != "" && !validTypes[contextType] {
		return fmt.Errorf("invalid context_type: must be one of 'private', 'group', or 'public'")
	}
	return nil
}

func HandleStoreMemory(c *gin.Context) {
	if memoryManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "memory manager not initialized",
		})
		return
	}

	var req StoreMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := validateContextType(req.ContextType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if req.ContextType == "" {
		req.ContextType = "private"
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
		ContextType: req.ContextType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := memoryManager.Store(c.Request.Context(), mem); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to store memory",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"memory":  mem,
	})
}

func HandleRetrieveMemory(c *gin.Context) {
	if memoryManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "memory manager not initialized",
		})
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Memory ID is required",
		})
		return
	}

	mem, err := memoryManager.Retrieve(c.Request.Context(), memoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Memory not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"memory":  mem,
	})
}

func HandleSearchMemory(c *gin.Context) {
	if memoryManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "memory manager not initialized",
		})
		return
	}

	var req SearchMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 100 {
		maxResults = 100
	}

	memories, err := memoryManager.Search(c.Request.Context(), req.Query, maxResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to search memories",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(memories),
		"results": memories,
	})
}

func HandleDeleteMemory(c *gin.Context) {
	if memoryManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "memory manager not initialized",
		})
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Memory ID is required",
		})
		return
	}

	if err := memoryManager.Delete(c.Request.Context(), memoryID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Failed to delete memory",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Memory deleted successfully",
	})
}

type UpdateMemoryRequest struct {
	Content  string                 `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func HandleUpdateMemory(c *gin.Context) {
	if memoryManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "memory manager not initialized",
		})
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Memory ID is required",
		})
		return
	}

	var req UpdateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.Content == "" && req.Metadata == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "At least one of content or metadata must be provided",
		})
		return
	}

	updatedMemory, err := memoryManager.Update(c.Request.Context(), memoryID, req.Content, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update memory",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"memory":  updatedMemory,
	})
}

type ListMemoriesRequest struct {
	SessionID  string `form:"session_id"`
	MaxResults int    `form:"max_results"`
}

func HandleListMemories(c *gin.Context) {
	if memoryManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "memory manager not initialized",
		})
		return
	}

	var req ListMemoriesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 100 {
		maxResults = 100
	}

	memories, err := memoryManager.List(c.Request.Context(), req.SessionID, maxResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list memories",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"count":    len(memories),
		"memories": memories,
	})
}

type ListSeedsRequest struct {
	Limit int `form:"limit"`
}

type CreateSeedRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Trigger     string `json:"trigger"`
	Content     string `json:"content" binding:"required"`
}

func HandleListSeeds(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	var req ListSeedsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}

	seeds, err := gardenManager.ListSeeds(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list seeds",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(seeds),
		"seeds":   seeds,
	})
}

func HandleCreateSeed(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	var req CreateSeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	now := time.Now()
	seed := &memory.Seed{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Trigger:     req.Trigger,
		Content:     req.Content,
		UsageCount:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := gardenManager.StoreSeed(c.Request.Context(), seed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create seed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"seed":    seed,
	})
}

type ListSnapshotsRequest struct {
	SessionID string `form:"session_id" binding:"required"`
}

func HandleListSnapshots(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	sessionID := c.Param("session")
	if sessionID == "" {
		sessionID = c.Query("session")
	}
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID is required",
		})
		return
	}

	snapshots, err := gardenManager.ListSnapshots(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list snapshots",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"count":     len(snapshots),
		"snapshots": snapshots,
	})
}

type CreateSnapshotRequest struct {
	SessionID    string                 `json:"session_id" binding:"required"`
	SnapshotName string                 `json:"snapshot_name"`
	SnapshotData map[string]interface{} `json:"snapshot_data"`
}

func HandleCreateSnapshot(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.SnapshotData == nil {
		req.SnapshotData = make(map[string]interface{})
	}

	snapshot := &memory.MemorySnapshot{
		ID:           uuid.New().String(),
		SessionID:    req.SessionID,
		SnapshotName: req.SnapshotName,
		SnapshotData: req.SnapshotData,
		CreatedAt:    time.Now(),
	}

	if err := gardenManager.StoreSnapshot(c.Request.Context(), snapshot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create snapshot",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"snapshot": snapshot,
	})
}

func HandleRestoreSnapshot(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	snapshotID := c.Param("snapshot")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Snapshot ID is required",
		})
		return
	}

	snapshot, err := gardenManager.RetrieveSnapshot(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Snapshot not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"snapshot": snapshot,
		"message":  "Snapshot retrieved. Apply snapshot_data manually.",
	})
}

func HandleGetGardenContext(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	sessionID := c.Query("session")
	if sessionID == "" {
		sessionID = "default"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tiers": []map[string]interface{}{
			{"tier": 1, "name": "Core Context", "items": 0, "tokens": 0, "capacity": 2000},
			{"tier": 2, "name": "Active Knowledge", "items": 0, "tokens": 0, "capacity": 2000},
			{"tier": 3, "name": "Referenced Content", "items": 0, "tokens": 0, "capacity": 2000},
			{"tier": 4, "name": "Compressed History", "items": 0, "tokens": 0, "capacity": 2000},
		},
		"total_tokens":     0,
		"capacity_limit":   8000,
		"capacity_percent": 0.0,
	})
}

func HandleGetGardenStats(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	sessionID := c.Query("session")
	if sessionID == "" {
		sessionID = "default"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                   true,
		"total_compressions":        0,
		"total_tokens_saved":        0,
		"average_compression_ratio": 0.0,
		"recent_compressions":       []interface{}{},
		"session_stats": map[string]interface{}{
			"total_turns":      0,
			"compressed_turns": 0,
			"active_turns":     0,
		},
	})
}

func HandleDeleteSeed(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	seedID := c.Param("id")
	if seedID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Seed ID is required",
		})
		return
	}

	if err := gardenManager.DeleteSeed(c.Request.Context(), seedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete seed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Seed deleted successfully",
	})
}

func HandleDeleteSnapshot(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Snapshot ID is required",
		})
		return
	}

	if err := gardenManager.DeleteSnapshot(c.Request.Context(), snapshotID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete snapshot",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Snapshot deleted successfully",
	})
}

func HandleExportSnapshot(c *gin.Context) {
	if gardenManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "garden manager not initialized",
		})
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Snapshot ID is required",
		})
		return
	}

	snapshot, err := gardenManager.RetrieveSnapshot(c.Request.Context(), snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Snapshot not found",
			"details": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=snapshot_%s.json", snapshot.SnapshotName))
	c.JSON(http.StatusOK, snapshot)
}

func parseLimit(s string) (int, error) {
	var limit int
	_, err := fmt.Sscanf(s, "%d", &limit)
	return limit, err
}

func HandleRunMaintenance(c *gin.Context) {
	if memoryMaintenance == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "memory maintenance not initialized",
		})
		return
	}

	report, err := memoryMaintenance.RunMaintenance(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to run maintenance",
			"details": err.Error(),
			"report":  report,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Maintenance completed",
		"report":  report,
	})
}
