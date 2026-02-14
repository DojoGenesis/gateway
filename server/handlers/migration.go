package handlers

import (
	"errors"
	"net/http"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/database"
	"github.com/gin-gonic/gin"
)

type MigrationHandlers struct {
	migrationManager *database.MigrationManager
}

func NewMigrationHandlers(migrationManager *database.MigrationManager) *MigrationHandlers {
	return &MigrationHandlers{
		migrationManager: migrationManager,
	}
}

type StartMigrationRequest struct {
	CloudUserID string `json:"cloud_user_id" binding:"required"`
}

type StartMigrationResponse struct {
	MigrationID string                `json:"migration_id"`
	Status      string                `json:"status"`
	Summary     *database.DataSummary `json:"summary"`
}

func (h *MigrationHandlers) HandleStartMigration(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req StartMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	summary, err := h.migrationManager.GetDataSummary(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get data summary", "details": err.Error()})
		return
	}

	migrationID, err := h.migrationManager.StartMigration(c.Request.Context(), userID.(string), req.CloudUserID)
	if err != nil {
		if errors.Is(err, database.ErrMigrationInProgress) {
			c.JSON(http.StatusConflict, gin.H{"error": "migration already in progress"})
			return
		}
		if errors.Is(err, database.ErrMigrationAlreadyDone) {
			c.JSON(http.StatusConflict, gin.H{"error": "migration already completed"})
			return
		}
		if errors.Is(err, database.ErrNoDataToMigrate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no data to migrate"})
			return
		}
		if errors.Is(err, database.ErrCloudAdapterRequired) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cloud adapter not available"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start migration", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, StartMigrationResponse{
		MigrationID: migrationID,
		Status:      "started",
		Summary:     summary,
	})
}

type GetMigrationStatusResponse struct {
	Progress *database.MigrationProgress `json:"progress"`
}

func (h *MigrationHandlers) HandleGetMigrationStatus(c *gin.Context) {
	migrationID := c.Param("id")
	if migrationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "migration_id is required"})
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	progress, err := h.migrationManager.GetMigrationStatus(c.Request.Context(), migrationID)
	if err != nil {
		if errors.Is(err, database.ErrMigrationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "migration not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get migration status", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetMigrationStatusResponse{
		Progress: progress,
	})
}

type GetLatestMigrationResponse struct {
	Progress *database.MigrationProgress `json:"progress,omitempty"`
}

func (h *MigrationHandlers) HandleGetLatestMigration(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	progress, err := h.migrationManager.GetLatestMigration(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, database.ErrMigrationNotFound) {
			c.JSON(http.StatusOK, GetLatestMigrationResponse{Progress: nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get latest migration", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetLatestMigrationResponse{
		Progress: progress,
	})
}

func (h *MigrationHandlers) HandleCancelMigration(c *gin.Context) {
	migrationID := c.Param("id")
	if migrationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "migration_id is required"})
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.migrationManager.CancelMigration(c.Request.Context(), migrationID)
	if err != nil {
		if errors.Is(err, database.ErrMigrationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "migration not found or already completed"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel migration", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "migration cancelled successfully",
	})
}

type GetDataSummaryResponse struct {
	Summary *database.DataSummary `json:"summary"`
}

func (h *MigrationHandlers) HandleGetDataSummary(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	summary, err := h.migrationManager.GetDataSummary(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get data summary", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, GetDataSummaryResponse{
		Summary: summary,
	})
}
