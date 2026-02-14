package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/projects"
	"github.com/gin-gonic/gin"
)

var projectManager *projects.ProjectManager

func InitializeProjectHandlers(pm *projects.ProjectManager) {
	projectManager = pm
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

func HandleCreateProject(c *gin.Context) {
	if projectManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "project manager not initialized",
		})
		return
	}

	var req CreateProjectRequest
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
			"error":   "project name cannot be empty",
		})
		return
	}

	if len(req.Name) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "project name too long (max 200 characters)",
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

	project, err := projectManager.CreateProject(c.Request.Context(), req.Name, req.Description, req.TemplateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to create project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"project": project,
	})
}

func HandleGetProject(c *gin.Context) {
	if projectManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "project manager not initialized",
		})
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "project ID is required",
		})
		return
	}

	project, err := projectManager.GetProject(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "project not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"project": project,
	})
}

func HandleListProjects(c *gin.Context) {
	if projectManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "project manager not initialized",
		})
		return
	}

	var req ListProjectsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	projectsList, err := projectManager.ListProjects(c.Request.Context(), req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to list projects",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"count":    len(projectsList),
		"projects": projectsList,
	})
}

func HandleUpdateProject(c *gin.Context) {
	if projectManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "project manager not initialized",
		})
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "project ID is required",
		})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	updatedProject, err := projectManager.UpdateProject(
		c.Request.Context(),
		projectID,
		req.Name,
		req.Description,
		req.Status,
		req.Settings,
		req.Metadata,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to update project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"project": updatedProject,
	})
}

func HandleDeleteProject(c *gin.Context) {
	if projectManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "project manager not initialized",
		})
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "project ID is required",
		})
		return
	}

	if err := projectManager.DeleteProject(c.Request.Context(), projectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to delete project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "project deleted successfully",
	})
}

func HandleListProjectTemplates(c *gin.Context) {
	if projectManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "project manager not initialized",
		})
		return
	}

	templates, err := projectManager.ListTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to list templates",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"count":     len(templates),
		"templates": templates,
	})
}

func HandleExportProject(c *gin.Context) {
	if projectManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "project manager not initialized",
		})
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "project ID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	zipData, err := projectManager.ExportProject(ctx, projectID)
	if err != nil {
		log.Printf("Failed to export project %s: %v", projectID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to export project",
		})
		return
	}

	project, err := projectManager.GetProject(ctx, projectID)
	if err != nil {
		log.Printf("Failed to get project details for %s: %v", projectID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get project details",
		})
		return
	}

	safeProjectName := sanitizeFilename(project.Name)
	filename := fmt.Sprintf("%s_export_%s.zip", safeProjectName, time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/zip", zipData)
}

func HandleImportProject(c *gin.Context) {
	if projectManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "project manager not initialized",
		})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "failed to read uploaded file",
		})
		return
	}
	defer file.Close()

	const maxUploadSize = 100 * 1024 * 1024
	limitedReader := io.LimitReader(file, maxUploadSize)
	zipData, err := io.ReadAll(limitedReader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to read file content",
		})
		return
	}

	if int64(len(zipData)) == maxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"success": false,
			"error":   "uploaded file exceeds maximum size (100MB)",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	project, err := projectManager.ImportProject(ctx, zipData)
	if err != nil {
		log.Printf("Failed to import project: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to import project",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "project imported successfully",
		"project": project,
	})
}
