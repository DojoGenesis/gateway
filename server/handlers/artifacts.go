package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/artifacts"
	"github.com/gin-gonic/gin"
)

// ArtifactHandler handles artifact-related HTTP requests.
type ArtifactHandler struct {
	manager *artifacts.ArtifactManager
}

// NewArtifactHandler creates a new ArtifactHandler.
func NewArtifactHandler(am *artifacts.ArtifactManager) *ArtifactHandler {
	return &ArtifactHandler{manager: am}
}

type CreateArtifactRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	SessionID   string `json:"session_id"`
	Type        string `json:"type" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Content     string `json:"content" binding:"required"`
}

type UpdateArtifactRequest struct {
	Content       string `json:"content" binding:"required"`
	CommitMessage string `json:"commit_message"`
}

type ListArtifactsRequest struct {
	ProjectID string `form:"project_id"`
	Type      string `form:"type"`
	Limit     int    `form:"limit"`
}

func (h *ArtifactHandler) CreateArtifact(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "artifact manager not initialized")
		return
	}

	var req CreateArtifactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid create artifact request body", "error", err)
		respondBadRequestWithSuccess(c, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respondBadRequestWithSuccess(c, "artifact name cannot be empty")
		return
	}

	if len(req.Name) > 200 {
		respondBadRequestWithSuccess(c, "artifact name too long (max 200 characters)")
		return
	}

	if len(req.Description) > 5000 {
		respondBadRequestWithSuccess(c, "description too long (max 5000 characters)")
		return
	}

	const maxContentSize = 10 * 1024 * 1024
	if len(req.Content) > maxContentSize {
		respondBadRequestWithSuccess(c, "content too large (max 10MB)")
		return
	}

	validTypes := map[string]bool{
		"document":   true,
		"code":       true,
		"image":      true,
		"data":       true,
		"config":     true,
		"script":     true,
		"markdown":   true,
		"stylesheet": true,
		"other":      true,
	}
	if !validTypes[req.Type] {
		respondBadRequestWithSuccess(c, "invalid artifact type",
			"valid types: document, code, image, data, config, script, markdown, stylesheet, other")
		return
	}

	artifact, err := h.manager.CreateArtifact(
		c.Request.Context(),
		req.ProjectID,
		req.SessionID,
		artifacts.ArtifactType(req.Type),
		req.Name,
		req.Description,
		req.Content,
	)
	if err != nil {
		slog.Error("failed to create artifact", "error", err, "project_id", req.ProjectID, "name", req.Name)
		respondInternalErrorWithSuccess(c, "failed to create artifact")
		return
	}

	initialVersion, err := h.manager.GetArtifactVersion(c.Request.Context(), artifact.ID, 1)
	if err != nil {
		slog.Error("failed to retrieve initial version", "error", err, "artifact_id", artifact.ID)
		respondInternalErrorWithSuccess(c, "failed to retrieve initial version")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"artifact": artifact,
		"version":  initialVersion,
	})
}

func (h *ArtifactHandler) GetArtifact(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "artifact manager not initialized")
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		respondBadRequestWithSuccess(c, "artifact ID is required")
		return
	}

	versionStr := c.Query("version")
	version := 0
	if versionStr != "" {
		var err error
		version, err = strconv.Atoi(versionStr)
		if err != nil {
			slog.Warn("invalid version parameter", "error", err, "version_str", versionStr)
			respondBadRequestWithSuccess(c, "invalid version parameter")
			return
		}
	}

	artifact, err := h.manager.GetArtifact(c.Request.Context(), artifactID)
	if err != nil {
		slog.Warn("artifact not found", "error", err, "artifact_id", artifactID)
		respondNotFoundWithSuccess(c, "artifact not found")
		return
	}

	if version > 0 {
		artifactVersion, err := h.manager.GetArtifactVersion(c.Request.Context(), artifactID, version)
		if err != nil {
			slog.Warn("artifact version not found", "error", err, "artifact_id", artifactID, "version", version)
			respondNotFoundWithSuccess(c, "artifact version not found")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"artifact": artifact,
			"version":  artifactVersion,
		})
		return
	}

	latestVersion, err := h.manager.GetArtifactVersion(c.Request.Context(), artifactID, artifact.LatestVersion)
	if err != nil {
		slog.Error("failed to retrieve latest artifact version", "error", err, "artifact_id", artifactID, "version", artifact.LatestVersion)
		respondNotFoundWithSuccess(c, "artifact version not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"artifact": artifact,
		"version":  latestVersion,
	})
}

func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "artifact manager not initialized")
		return
	}

	var req ListArtifactsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		slog.Warn("invalid list artifacts query parameters", "error", err)
		respondBadRequestWithSuccess(c, "invalid query parameters")
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	artifactsList, err := h.manager.ListArtifacts(c.Request.Context(), req.ProjectID, artifacts.ArtifactType(req.Type), limit)
	if err != nil {
		slog.Error("failed to list artifacts", "error", err, "project_id", req.ProjectID, "type", req.Type)
		respondInternalErrorWithSuccess(c, "failed to list artifacts")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"count":     len(artifactsList),
		"artifacts": artifactsList,
	})
}

func (h *ArtifactHandler) UpdateArtifact(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "artifact manager not initialized")
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		respondBadRequestWithSuccess(c, "artifact ID is required")
		return
	}

	var req UpdateArtifactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("invalid update artifact request body", "error", err)
		respondBadRequestWithSuccess(c, "invalid request body")
		return
	}

	const maxContentSize = 10 * 1024 * 1024
	if len(req.Content) > maxContentSize {
		respondBadRequestWithSuccess(c, "content too large (max 10MB)")
		return
	}

	commitMessage := req.CommitMessage
	if commitMessage == "" {
		commitMessage = "Updated artifact"
	}

	updatedArtifact, err := h.manager.UpdateArtifact(
		c.Request.Context(),
		artifactID,
		req.Content,
		commitMessage,
	)
	if err != nil {
		slog.Error("failed to update artifact", "error", err, "artifact_id", artifactID)
		respondInternalErrorWithSuccess(c, "failed to update artifact")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"artifact": updatedArtifact,
	})
}

func (h *ArtifactHandler) DeleteArtifact(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "artifact manager not initialized")
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		respondBadRequestWithSuccess(c, "artifact ID is required")
		return
	}

	if err := h.manager.DeleteArtifact(c.Request.Context(), artifactID); err != nil {
		slog.Error("failed to delete artifact", "error", err, "artifact_id", artifactID)
		respondInternalErrorWithSuccess(c, "failed to delete artifact")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "artifact deleted successfully",
	})
}

func (h *ArtifactHandler) ListArtifactVersions(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "artifact manager not initialized")
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		respondBadRequestWithSuccess(c, "artifact ID is required")
		return
	}

	versions, err := h.manager.ListVersions(c.Request.Context(), artifactID)
	if err != nil {
		slog.Error("failed to list artifact versions", "error", err, "artifact_id", artifactID)
		respondInternalErrorWithSuccess(c, "failed to list versions")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"count":    len(versions),
		"versions": versions,
	})
}

func (h *ArtifactHandler) ExportArtifact(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "artifact manager not initialized")
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		respondBadRequestWithSuccess(c, "artifact ID is required")
		return
	}

	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Minute)
	defer cancel()

	artifact, err := h.manager.GetArtifact(ctx, artifactID)
	if err != nil {
		slog.Warn("artifact not found for export", "error", err, "artifact_id", artifactID)
		respondNotFoundWithSuccess(c, "artifact not found")
		return
	}

	versions, err := h.manager.ListVersions(ctx, artifactID)
	if err != nil {
		slog.Error("failed to retrieve versions for export", "error", err, "artifact_id", artifactID)
		respondInternalErrorWithSuccess(c, "failed to retrieve versions")
		return
	}

	switch format {
	case "json":
		exportData := gin.H{
			"artifact":    artifact,
			"versions":    versions,
			"exported_at": time.Now(),
		}

		safeArtifactName := sanitizeFilename(artifact.Name)
		filename := fmt.Sprintf("%s_v%d_%s.json", safeArtifactName, artifact.LatestVersion, time.Now().Format("20060102_150405"))
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.JSON(http.StatusOK, exportData)

	case "markdown", "md":
		latestVersion, err := h.manager.GetArtifactVersion(ctx, artifactID, artifact.LatestVersion)
		if err != nil {
			respondInternalErrorWithSuccess(c, "failed to retrieve latest version")
			return
		}

		safeArtifactName := sanitizeFilename(artifact.Name)
		filename := fmt.Sprintf("%s_v%d.md", safeArtifactName, artifact.LatestVersion)
		c.Header("Content-Type", "text/markdown")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.String(http.StatusOK, latestVersion.Content)

	case "text", "txt":
		latestVersion, err := h.manager.GetArtifactVersion(ctx, artifactID, artifact.LatestVersion)
		if err != nil {
			respondInternalErrorWithSuccess(c, "failed to retrieve latest version")
			return
		}

		safeArtifactName := sanitizeFilename(artifact.Name)
		filename := fmt.Sprintf("%s_v%d.txt", safeArtifactName, artifact.LatestVersion)
		c.Header("Content-Type", "text/plain")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.String(http.StatusOK, latestVersion.Content)

	default:
		respondBadRequestWithSuccess(c, fmt.Sprintf("unsupported export format: %s", format),
			"supported formats: json, markdown, md, text, txt")
	}
}
