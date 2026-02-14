package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/projects"
	"github.com/gin-gonic/gin"
)

// ProjectHandler handles project-related HTTP requests.
type ProjectHandler struct {
	manager *projects.ProjectManager
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(pm *projects.ProjectManager) *ProjectHandler {
	return &ProjectHandler{manager: pm}
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	TemplateID  string `json:"template_id"`
}

type UpdateProjectRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ListProjectsRequest struct {
	Status string `form:"status"`
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "project manager not initialized")
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respondBadRequestWithSuccess(c, "project name cannot be empty")
		return
	}

	if len(req.Name) > 200 {
		respondBadRequestWithSuccess(c, "project name too long (max 200 characters)")
		return
	}

	if len(req.Description) > 5000 {
		respondBadRequestWithSuccess(c, "description too long (max 5000 characters)")
		return
	}

	project, err := h.manager.CreateProject(c.Request.Context(), req.Name, req.Description, req.TemplateID)
	if err != nil {
		slog.Error("failed to create project", "error", err)
		respondInternalErrorWithSuccess(c, "failed to create project")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"project": project,
	})
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "project manager not initialized")
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		respondBadRequestWithSuccess(c, "project ID is required")
		return
	}

	project, err := h.manager.GetProject(c.Request.Context(), projectID)
	if err != nil {
		slog.Warn("project not found", "project_id", projectID, "error", err)
		respondNotFoundWithSuccess(c, "project not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"project": project,
	})
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "project manager not initialized")
		return
	}

	var req ListProjectsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondBadRequestWithSuccess(c, "invalid query parameters")
		return
	}

	projectsList, err := h.manager.ListProjects(c.Request.Context(), req.Status)
	if err != nil {
		slog.Error("failed to list projects", "error", err)
		respondInternalErrorWithSuccess(c, "failed to list projects")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"count":    len(projectsList),
		"projects": projectsList,
	})
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "project manager not initialized")
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		respondBadRequestWithSuccess(c, "project ID is required")
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequestWithSuccess(c, "invalid request body")
		return
	}

	updatedProject, err := h.manager.UpdateProject(
		c.Request.Context(),
		projectID,
		req.Name,
		req.Description,
		req.Status,
		req.Settings,
		req.Metadata,
	)
	if err != nil {
		slog.Error("failed to update project", "project_id", projectID, "error", err)
		respondInternalErrorWithSuccess(c, "failed to update project")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"project": updatedProject,
	})
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "project manager not initialized")
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		respondBadRequestWithSuccess(c, "project ID is required")
		return
	}

	if err := h.manager.DeleteProject(c.Request.Context(), projectID); err != nil {
		slog.Error("failed to delete project", "project_id", projectID, "error", err)
		respondInternalErrorWithSuccess(c, "failed to delete project")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "project deleted successfully",
	})
}

func (h *ProjectHandler) ListProjectTemplates(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "project manager not initialized")
		return
	}

	templates, err := h.manager.ListTemplates(c.Request.Context())
	if err != nil {
		slog.Error("failed to list templates", "error", err)
		respondInternalErrorWithSuccess(c, "failed to list templates")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"count":     len(templates),
		"templates": templates,
	})
}

func (h *ProjectHandler) ExportProject(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "project manager not initialized")
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		respondBadRequestWithSuccess(c, "project ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	zipData, err := h.manager.ExportProject(ctx, projectID)
	if err != nil {
		slog.Error("failed to export project", "project_id", projectID, "error", err)
		respondInternalErrorWithSuccess(c, "failed to export project")
		return
	}

	project, err := h.manager.GetProject(ctx, projectID)
	if err != nil {
		slog.Error("failed to get project details", "project_id", projectID, "error", err)
		respondInternalErrorWithSuccess(c, "failed to get project details")
		return
	}

	safeProjectName := sanitizeFilename(project.Name)
	filename := fmt.Sprintf("%s_export_%s.zip", safeProjectName, time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/zip", zipData)
}

func (h *ProjectHandler) ImportProject(c *gin.Context) {
	if h.manager == nil {
		respondInternalErrorWithSuccess(c, "project manager not initialized")
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		respondBadRequestWithSuccess(c, "failed to read uploaded file")
		return
	}
	defer file.Close()

	const maxUploadSize = 100 * 1024 * 1024
	limitedReader := io.LimitReader(file, maxUploadSize)
	zipData, err := io.ReadAll(limitedReader)
	if err != nil {
		respondInternalErrorWithSuccess(c, "failed to read file content")
		return
	}

	if int64(len(zipData)) == maxUploadSize {
		respondErrorWithSuccess(c, http.StatusRequestEntityTooLarge, "uploaded file exceeds maximum size (100MB)")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	project, err := h.manager.ImportProject(ctx, zipData)
	if err != nil {
		slog.Error("failed to import project", "error", err)
		respondInternalErrorWithSuccess(c, "failed to import project")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "project imported successfully",
		"project": project,
	})
}
