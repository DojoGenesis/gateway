package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/artifacts"
	"github.com/gin-gonic/gin"
)

var artifactManager *artifacts.ArtifactManager

func InitializeArtifactHandlers(am *artifacts.ArtifactManager) {
	artifactManager = am
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

func HandleCreateArtifact(c *gin.Context) {
	if artifactManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "artifact manager not initialized",
		})
		return
	}

	var req CreateArtifactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "artifact name cannot be empty",
		})
		return
	}

	if len(req.Name) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "artifact name too long (max 200 characters)",
		})
		return
	}

	if len(req.Description) > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "description too long (max 5000 characters)",
		})
		return
	}

	const maxContentSize = 10 * 1024 * 1024
	if len(req.Content) > maxContentSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "content too large (max 10MB)",
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success":     false,
			"error":       "invalid artifact type",
			"valid_types": []string{"document", "code", "image", "data", "config", "script", "markdown", "stylesheet", "other"},
		})
		return
	}

	artifact, err := artifactManager.CreateArtifact(
		c.Request.Context(),
		req.ProjectID,
		req.SessionID,
		artifacts.ArtifactType(req.Type),
		req.Name,
		req.Description,
		req.Content,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to create artifact",
			"details": err.Error(),
		})
		return
	}

	initialVersion, err := artifactManager.GetArtifactVersion(c.Request.Context(), artifact.ID, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to retrieve initial version",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"artifact": artifact,
		"version":  initialVersion,
	})
}

func HandleGetArtifact(c *gin.Context) {
	if artifactManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "artifact manager not initialized",
		})
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "artifact ID is required",
		})
		return
	}

	versionStr := c.Query("version")
	version := 0
	if versionStr != "" {
		var err error
		version, err = strconv.Atoi(versionStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid version parameter",
				"details": err.Error(),
			})
			return
		}
	}

	artifact, err := artifactManager.GetArtifact(c.Request.Context(), artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "artifact not found",
			"details": err.Error(),
		})
		return
	}

	if version > 0 {
		artifactVersion, err := artifactManager.GetArtifactVersion(c.Request.Context(), artifactID, version)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "artifact version not found",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"artifact": artifact,
			"version":  artifactVersion,
		})
		return
	}

	latestVersion, err := artifactManager.GetArtifactVersion(c.Request.Context(), artifactID, artifact.LatestVersion)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "artifact version not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"artifact": artifact,
		"version":  latestVersion,
	})
}

func HandleListArtifacts(c *gin.Context) {
	if artifactManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "artifact manager not initialized",
		})
		return
	}

	var req ListArtifactsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	artifactsList, err := artifactManager.ListArtifacts(c.Request.Context(), req.ProjectID, artifacts.ArtifactType(req.Type), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to list artifacts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"count":     len(artifactsList),
		"artifacts": artifactsList,
	})
}

func HandleUpdateArtifact(c *gin.Context) {
	if artifactManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "artifact manager not initialized",
		})
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "artifact ID is required",
		})
		return
	}

	var req UpdateArtifactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	const maxContentSize = 10 * 1024 * 1024
	if len(req.Content) > maxContentSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "content too large (max 10MB)",
		})
		return
	}

	commitMessage := req.CommitMessage
	if commitMessage == "" {
		commitMessage = "Updated artifact"
	}

	updatedArtifact, err := artifactManager.UpdateArtifact(
		c.Request.Context(),
		artifactID,
		req.Content,
		commitMessage,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to update artifact",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"artifact": updatedArtifact,
	})
}

func HandleDeleteArtifact(c *gin.Context) {
	if artifactManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "artifact manager not initialized",
		})
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "artifact ID is required",
		})
		return
	}

	if err := artifactManager.DeleteArtifact(c.Request.Context(), artifactID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to delete artifact",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "artifact deleted successfully",
	})
}

func HandleListArtifactVersions(c *gin.Context) {
	if artifactManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "artifact manager not initialized",
		})
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "artifact ID is required",
		})
		return
	}

	versions, err := artifactManager.ListVersions(c.Request.Context(), artifactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to list versions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"count":    len(versions),
		"versions": versions,
	})
}

func HandleExportArtifact(c *gin.Context) {
	if artifactManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "artifact manager not initialized",
		})
		return
	}

	artifactID := c.Param("id")
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "artifact ID is required",
		})
		return
	}

	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Minute)
	defer cancel()

	artifact, err := artifactManager.GetArtifact(ctx, artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "artifact not found",
			"details": err.Error(),
		})
		return
	}

	versions, err := artifactManager.ListVersions(ctx, artifactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to retrieve versions",
			"details": err.Error(),
		})
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
		latestVersion, err := artifactManager.GetArtifactVersion(ctx, artifactID, artifact.LatestVersion)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "failed to retrieve latest version",
			})
			return
		}

		safeArtifactName := sanitizeFilename(artifact.Name)
		filename := fmt.Sprintf("%s_v%d.md", safeArtifactName, artifact.LatestVersion)
		c.Header("Content-Type", "text/markdown")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.String(http.StatusOK, latestVersion.Content)

	case "text", "txt":
		latestVersion, err := artifactManager.GetArtifactVersion(ctx, artifactID, artifact.LatestVersion)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "failed to retrieve latest version",
			})
			return
		}

		safeArtifactName := sanitizeFilename(artifact.Name)
		filename := fmt.Sprintf("%s_v%d.txt", safeArtifactName, artifact.LatestVersion)
		c.Header("Content-Type", "text/plain")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.String(http.StatusOK, latestVersion.Content)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success":           false,
			"error":             fmt.Sprintf("unsupported export format: %s", format),
			"supported_formats": []string{"json", "markdown", "md", "text", "txt"},
		})
	}
}
