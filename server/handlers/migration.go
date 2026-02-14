package handlers

import (
	"errors"
	"log/slog"
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
		respondUnauthorized(c, "unauthorized")
		return
	}

	var req StartMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid request body")
		return
	}

	summary, err := h.migrationManager.GetDataSummary(c.Request.Context(), userID.(string))
	if err != nil {
		slog.Error("failed to get data summary", "error", err)
		respondInternalError(c, "failed to get data summary")
		return
	}

	migrationID, err := h.migrationManager.StartMigration(c.Request.Context(), userID.(string), req.CloudUserID)
	if err != nil {
		if errors.Is(err, database.ErrMigrationInProgress) {
			respondConflict(c, "migration already in progress")
			return
		}
		if errors.Is(err, database.ErrMigrationAlreadyDone) {
			respondConflict(c, "migration already completed")
			return
		}
		if errors.Is(err, database.ErrNoDataToMigrate) {
			respondBadRequest(c, "no data to migrate")
			return
		}
		if errors.Is(err, database.ErrCloudAdapterRequired) {
			respondError(c, http.StatusServiceUnavailable, "cloud adapter not available")
			return
		}
		slog.Error("failed to start migration", "error", err)
		respondInternalError(c, "failed to start migration")
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
		respondBadRequest(c, "migration_id is required")
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		respondUnauthorized(c, "unauthorized")
		return
	}

	progress, err := h.migrationManager.GetMigrationStatus(c.Request.Context(), migrationID)
	if err != nil {
		if errors.Is(err, database.ErrMigrationNotFound) {
			respondNotFound(c, "migration")
			return
		}
		slog.Error("failed to get migration status", "error", err)
		respondInternalError(c, "failed to get migration status")
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
		respondUnauthorized(c, "unauthorized")
		return
	}

	progress, err := h.migrationManager.GetLatestMigration(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, database.ErrMigrationNotFound) {
			c.JSON(http.StatusOK, GetLatestMigrationResponse{Progress: nil})
			return
		}
		slog.Error("failed to get latest migration", "error", err)
		respondInternalError(c, "failed to get latest migration")
		return
	}

	c.JSON(http.StatusOK, GetLatestMigrationResponse{
		Progress: progress,
	})
}

func (h *MigrationHandlers) HandleCancelMigration(c *gin.Context) {
	migrationID := c.Param("id")
	if migrationID == "" {
		respondBadRequest(c, "migration_id is required")
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		respondUnauthorized(c, "unauthorized")
		return
	}

	err := h.migrationManager.CancelMigration(c.Request.Context(), migrationID)
	if err != nil {
		if errors.Is(err, database.ErrMigrationNotFound) {
			respondError(c, http.StatusNotFound, "migration not found or already completed")
			return
		}
		slog.Error("failed to cancel migration", "error", err)
		respondInternalError(c, "failed to cancel migration")
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
		respondUnauthorized(c, "unauthorized")
		return
	}

	summary, err := h.migrationManager.GetDataSummary(c.Request.Context(), userID.(string))
	if err != nil {
		slog.Error("failed to get data summary", "error", err)
		respondInternalError(c, "failed to get data summary")
		return
	}

	c.JSON(http.StatusOK, GetDataSummaryResponse{
		Summary: summary,
	})
}
