package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/DojoGenesis/gateway/server/maintenance"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MemoryHandler handles memory, garden, and maintenance-related HTTP requests.
type MemoryHandler struct {
	memory      *memory.MemoryManager
	garden      *memory.GardenManager
	maintenance *maintenance.MemoryMaintenance
}

// NewMemoryHandler creates a new MemoryHandler.
func NewMemoryHandler(mm *memory.MemoryManager, gm *memory.GardenManager, maint *maintenance.MemoryMaintenance) *MemoryHandler {
	return &MemoryHandler{
		memory:      mm,
		garden:      gm,
		maintenance: maint,
	}
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

func (h *MemoryHandler) StoreMemory(c *gin.Context) {
	if h.memory == nil {
		respondInternalErrorWithSuccess(c, "memory manager not initialized")
		return
	}

	var req StoreMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
		return
	}

	if err := validateContextType(req.ContextType); err != nil {
		respondBadRequestWithSuccess(c, "Invalid context_type: must be one of 'private', 'group', or 'public'")
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

	if err := h.memory.Store(c.Request.Context(), mem); err != nil {
		slog.Error("failed to store memory", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to store memory")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"memory":  mem,
	})
}

func (h *MemoryHandler) RetrieveMemory(c *gin.Context) {
	if h.memory == nil {
		respondInternalErrorWithSuccess(c, "memory manager not initialized")
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		respondBadRequestWithSuccess(c, "Memory ID is required")
		return
	}

	mem, err := h.memory.Retrieve(c.Request.Context(), memoryID)
	if err != nil {
		slog.Warn("memory not found", "id", memoryID, "error", err)
		respondNotFoundWithSuccess(c, "Memory not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"memory":  mem,
	})
}

func (h *MemoryHandler) SearchMemory(c *gin.Context) {
	if h.memory == nil {
		respondInternalErrorWithSuccess(c, "memory manager not initialized")
		return
	}

	var req SearchMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
		return
	}

	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 100 {
		maxResults = 100
	}

	memories, err := h.memory.Search(c.Request.Context(), req.Query, maxResults)
	if err != nil {
		slog.Error("failed to search memories", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to search memories")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(memories),
		"results": memories,
	})
}

func (h *MemoryHandler) DeleteMemory(c *gin.Context) {
	if h.memory == nil {
		respondInternalErrorWithSuccess(c, "memory manager not initialized")
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		respondBadRequestWithSuccess(c, "Memory ID is required")
		return
	}

	if err := h.memory.Delete(c.Request.Context(), memoryID); err != nil {
		slog.Warn("failed to delete memory", "id", memoryID, "error", err)
		respondNotFoundWithSuccess(c, "Failed to delete memory")
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

func (h *MemoryHandler) UpdateMemory(c *gin.Context) {
	if h.memory == nil {
		respondInternalErrorWithSuccess(c, "memory manager not initialized")
		return
	}

	memoryID := c.Param("id")
	if memoryID == "" {
		respondBadRequestWithSuccess(c, "Memory ID is required")
		return
	}

	var req UpdateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
		return
	}

	if req.Content == "" && req.Metadata == nil {
		respondBadRequestWithSuccess(c, "At least one of content or metadata must be provided")
		return
	}

	updatedMemory, err := h.memory.Update(c.Request.Context(), memoryID, req.Content, req.Metadata)
	if err != nil {
		slog.Error("failed to update memory", "id", memoryID, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to update memory")
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

func (h *MemoryHandler) ListMemories(c *gin.Context) {
	if h.memory == nil {
		respondInternalErrorWithSuccess(c, "memory manager not initialized")
		return
	}

	var req ListMemoriesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid query parameters")
		return
	}

	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 100 {
		maxResults = 100
	}

	memories, err := h.memory.List(c.Request.Context(), req.SessionID, maxResults)
	if err != nil {
		slog.Error("failed to list memories", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to list memories")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"count":    len(memories),
		"memories": memories,
	})
}

// MaintenanceResponse represents the response for maintenance operations.
// It includes a report field for partial results even on failure.
type MaintenanceResponse struct {
	Success bool                        `json:"success"`
	Message string                      `json:"message,omitempty"`
	Error   string                      `json:"error,omitempty"`
	Report  *maintenance.MaintenanceReport `json:"report,omitempty"`
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

func (h *MemoryHandler) ListSeeds(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
		return
	}

	var req ListSeedsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid query parameters")
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}

	seeds, err := h.garden.ListSeeds(c.Request.Context(), limit)
	if err != nil {
		slog.Error("failed to list seeds", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to list seeds")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(seeds),
		"seeds":   seeds,
	})
}

func (h *MemoryHandler) CreateSeed(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
		return
	}

	var req CreateSeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
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

	if err := h.garden.StoreSeed(c.Request.Context(), seed); err != nil {
		slog.Error("failed to create seed", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to create seed")
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

func (h *MemoryHandler) ListSnapshots(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
		return
	}

	sessionID := c.Param("session")
	if sessionID == "" {
		sessionID = c.Query("session")
	}
	if sessionID == "" {
		respondBadRequestWithSuccess(c, "Session ID is required")
		return
	}

	snapshots, err := h.garden.ListSnapshots(c.Request.Context(), sessionID)
	if err != nil {
		slog.Error("failed to list snapshots", "session_id", sessionID, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to list snapshots")
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

func (h *MemoryHandler) CreateSnapshot(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
		return
	}

	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "Invalid request body")
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

	if err := h.garden.StoreSnapshot(c.Request.Context(), snapshot); err != nil {
		slog.Error("failed to create snapshot", "error", err)
		respondInternalErrorWithSuccess(c, "Failed to create snapshot")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"snapshot": snapshot,
	})
}

func (h *MemoryHandler) RestoreSnapshot(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
		return
	}

	snapshotID := c.Param("snapshot")
	if snapshotID == "" {
		respondBadRequestWithSuccess(c, "Snapshot ID is required")
		return
	}

	snapshot, err := h.garden.RetrieveSnapshot(c.Request.Context(), snapshotID)
	if err != nil {
		slog.Warn("snapshot not found", "id", snapshotID, "error", err)
		respondNotFoundWithSuccess(c, "Snapshot not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"snapshot": snapshot,
		"message":  "Snapshot retrieved. Apply snapshot_data manually.",
	})
}

func (h *MemoryHandler) GetGardenContext(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
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

func (h *MemoryHandler) GetGardenStats(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
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

func (h *MemoryHandler) DeleteSeed(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
		return
	}

	seedID := c.Param("id")
	if seedID == "" {
		respondBadRequestWithSuccess(c, "Seed ID is required")
		return
	}

	if err := h.garden.DeleteSeed(c.Request.Context(), seedID); err != nil {
		slog.Error("failed to delete seed", "id", seedID, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to delete seed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Seed deleted successfully",
	})
}

func (h *MemoryHandler) DeleteSnapshot(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		respondBadRequestWithSuccess(c, "Snapshot ID is required")
		return
	}

	if err := h.garden.DeleteSnapshot(c.Request.Context(), snapshotID); err != nil {
		slog.Error("failed to delete snapshot", "id", snapshotID, "error", err)
		respondInternalErrorWithSuccess(c, "Failed to delete snapshot")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Snapshot deleted successfully",
	})
}

func (h *MemoryHandler) ExportSnapshot(c *gin.Context) {
	if h.garden == nil {
		respondInternalErrorWithSuccess(c, "garden manager not initialized")
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		respondBadRequestWithSuccess(c, "Snapshot ID is required")
		return
	}

	snapshot, err := h.garden.RetrieveSnapshot(c.Request.Context(), snapshotID)
	if err != nil {
		slog.Warn("snapshot not found", "snapshot_id", snapshotID, "error", err)
		respondNotFoundWithSuccess(c, "Snapshot not found")
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

func (h *MemoryHandler) RunMaintenance(c *gin.Context) {
	if h.maintenance == nil {
		respondInternalErrorWithSuccess(c, "memory maintenance not initialized")
		return
	}

	report, err := h.maintenance.RunMaintenance(c.Request.Context())
	if err != nil {
		slog.Error("failed to run maintenance", "error", err)
		c.JSON(http.StatusInternalServerError, MaintenanceResponse{
			Success: false,
			Error:   "Failed to run maintenance",
			Report:  report,
		})
		return
	}

	c.JSON(http.StatusOK, MaintenanceResponse{
		Success: true,
		Message: "Maintenance completed",
		Report:  report,
	})
}
