package apptools

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/artifacts"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

var artifactManager *artifacts.ArtifactManager

func InitializeArtifactTools(am *artifacts.ArtifactManager) {
	artifactManager = am
}

func CreateArtifact(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if artifactManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact manager not initialized",
		}, nil
	}

	projectID := tools.GetStringParam(params, "project_id", "")
	if projectID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "project_id is required",
		}, nil
	}

	artifactType := tools.GetStringParam(params, "type", "")
	if artifactType == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "type is required",
		}, nil
	}

	name := tools.GetStringParam(params, "name", "")
	if name == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "name is required",
		}, nil
	}

	content := tools.GetStringParam(params, "content", "")
	description := tools.GetStringParam(params, "description", "")
	sessionID := tools.GetStringParam(params, "session_id", "")

	artifact, err := artifactManager.CreateArtifact(ctx, projectID, sessionID, artifacts.ArtifactType(artifactType), name, description, content)
	if err != nil {
		slog.Error("failed to create artifact", "error", err)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to create artifact",
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"artifact": map[string]interface{}{
			"id":             artifact.ID,
			"project_id":     artifact.ProjectID,
			"session_id":     artifact.SessionID,
			"type":           string(artifact.Type),
			"name":           artifact.Name,
			"description":    artifact.Description,
			"latest_version": artifact.LatestVersion,
			"created_at":     artifact.CreatedAt,
			"updated_at":     artifact.UpdatedAt,
		},
		"message": fmt.Sprintf("Artifact '%s' created successfully", artifact.Name),
	}, nil
}

func UpdateArtifact(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if artifactManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact manager not initialized",
		}, nil
	}

	artifactID := tools.GetStringParam(params, "artifact_id", "")
	if artifactID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact_id is required",
		}, nil
	}

	newContent := tools.GetStringParam(params, "content", "")
	if newContent == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "content is required",
		}, nil
	}

	commitMessage := tools.GetStringParam(params, "commit_message", "Updated artifact")

	artifact, err := artifactManager.UpdateArtifact(ctx, artifactID, newContent, commitMessage)
	if err != nil {
		slog.Error("failed to update artifact", "error", err)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to update artifact",
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"artifact": map[string]interface{}{
			"id":             artifact.ID,
			"project_id":     artifact.ProjectID,
			"type":           string(artifact.Type),
			"name":           artifact.Name,
			"latest_version": artifact.LatestVersion,
			"updated_at":     artifact.UpdatedAt,
		},
		"commit_message": commitMessage,
		"message":        fmt.Sprintf("Artifact '%s' updated to version %d", artifact.Name, artifact.LatestVersion),
	}, nil
}

func GetArtifact(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if artifactManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact manager not initialized",
		}, nil
	}

	artifactID := tools.GetStringParam(params, "artifact_id", "")
	if artifactID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact_id is required",
		}, nil
	}

	version := tools.GetIntParam(params, "version", 0)

	artifact, err := artifactManager.GetArtifact(ctx, artifactID)
	if err != nil {
		slog.Error("failed to get artifact", "error", err, "artifact_id", artifactID)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to get artifact",
		}, nil
	}

	artifactVersion, err := artifactManager.GetArtifactVersion(ctx, artifactID, artifact.LatestVersion)
	if version > 0 && version != artifact.LatestVersion {
		artifactVersion, err = artifactManager.GetArtifactVersion(ctx, artifactID, version)
	}

	if err != nil {
		slog.Error("failed to get artifact version", "error", err, "artifact_id", artifactID)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to get artifact version",
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"artifact": map[string]interface{}{
			"id":             artifact.ID,
			"project_id":     artifact.ProjectID,
			"session_id":     artifact.SessionID,
			"type":           artifact.Type,
			"name":           artifact.Name,
			"description":    artifact.Description,
			"latest_version": artifact.LatestVersion,
			"created_at":     artifact.CreatedAt,
			"updated_at":     artifact.UpdatedAt,
			"metadata":       artifact.Metadata,
			"content":        artifactVersion.Content,
			"version":        artifactVersion.Version,
		},
	}, nil
}

func ListArtifacts(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if artifactManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact manager not initialized",
		}, nil
	}

	projectID := tools.GetStringParam(params, "project_id", "")
	artifactTypeStr := tools.GetStringParam(params, "type", "")
	limit := tools.GetIntParam(params, "limit", 50)

	if limit <= 0 || limit > 1000 {
		limit = 50
	}

	var artifactType artifacts.ArtifactType
	if artifactTypeStr != "" {
		artifactType = artifacts.ArtifactType(artifactTypeStr)
	}

	artifactList, err := artifactManager.ListArtifacts(ctx, projectID, artifactType, limit)
	if err != nil {
		slog.Error("failed to list artifacts", "error", err)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to list artifacts",
		}, nil
	}

	return map[string]interface{}{
		"success":   true,
		"count":     len(artifactList),
		"artifacts": artifactList,
	}, nil
}

func ListArtifactVersions(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if artifactManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact manager not initialized",
		}, nil
	}

	artifactID := tools.GetStringParam(params, "artifact_id", "")
	if artifactID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact_id is required",
		}, nil
	}

	versions, err := artifactManager.ListVersions(ctx, artifactID)
	if err != nil {
		slog.Error("failed to list versions", "error", err, "artifact_id", artifactID)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to list versions",
		}, nil
	}

	return map[string]interface{}{
		"success":  true,
		"count":    len(versions),
		"versions": versions,
	}, nil
}

func ExportArtifact(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if artifactManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact manager not initialized",
		}, nil
	}

	artifactID := tools.GetStringParam(params, "artifact_id", "")
	if artifactID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "artifact_id is required",
		}, nil
	}

	format := tools.GetStringParam(params, "format", "raw")
	version := tools.GetIntParam(params, "version", 0)

	artifact, err := artifactManager.GetArtifact(ctx, artifactID)
	if err != nil {
		slog.Error("failed to get artifact", "error", err, "artifact_id", artifactID)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to get artifact",
		}, nil
	}

	versionNum := artifact.LatestVersion
	if version > 0 {
		versionNum = version
	}

	artifactVersion, err := artifactManager.GetArtifactVersion(ctx, artifactID, versionNum)
	if err != nil {
		slog.Error("failed to get artifact version", "error", err, "artifact_id", artifactID, "version", versionNum)
		return map[string]interface{}{
			"success": false,
			"error":   "failed to get artifact version",
		}, nil
	}

	var exportedContent string
	var filename string
	var mimeType string

	switch format {
	case "raw":
		exportedContent = artifactVersion.Content
		filename = getFilename(artifact, versionNum)
		mimeType = getMimeType(artifact.Type)

	case "base64":
		exportedContent = base64.StdEncoding.EncodeToString([]byte(artifactVersion.Content))
		filename = getFilename(artifact, versionNum)
		mimeType = getMimeType(artifact.Type)

	case "json":
		exportedContent = artifactVersion.Content
		filename = fmt.Sprintf("%s_v%d.json", artifact.Name, versionNum)
		mimeType = "application/json"

	default:
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("unsupported export format: %s", format),
		}, nil
	}

	return map[string]interface{}{
		"success":   true,
		"artifact":  artifact,
		"version":   versionNum,
		"format":    format,
		"content":   exportedContent,
		"filename":  filename,
		"mime_type": mimeType,
		"message":   fmt.Sprintf("Artifact '%s' exported successfully", artifact.Name),
	}, nil
}

func getFilename(artifact *artifacts.Artifact, version int) string {
	var ext string
	switch artifact.Type {
	case artifacts.TypeDocument:
		ext = "md"
	case artifacts.TypeDiagram:
		ext = "mmd"
	case artifacts.TypeCodeProject:
		ext = "zip"
	case artifacts.TypeDataViz:
		ext = "json"
	case artifacts.TypeImage:
		ext = "png"
	default:
		ext = "txt"
	}
	return fmt.Sprintf("%s_v%d.%s", artifact.Name, version, ext)
}

func getMimeType(artifactType artifacts.ArtifactType) string {
	switch artifactType {
	case artifacts.TypeDocument:
		return "text/markdown"
	case artifacts.TypeDiagram:
		return "text/plain"
	case artifacts.TypeCodeProject:
		return "application/zip"
	case artifacts.TypeDataViz:
		return "application/json"
	case artifacts.TypeImage:
		return "image/png"
	default:
		return "text/plain"
	}
}

func init() {
	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "create_artifact",
		Description: "Create a new artifact in a project. Artifacts are persistent, versionable outputs like documents, diagrams, code projects, data visualizations, or images.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the project this artifact belongs to",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Type of artifact",
					"enum":        []string{"document", "diagram", "code_project", "data_viz", "image"},
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the artifact",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content of the artifact (e.g., Markdown text, Mermaid diagram code, JSON data)",
				},
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Optional session ID that created this artifact",
				},
			},
			"required": []string{"project_id", "type", "name", "content"},
		},
		Function: CreateArtifact,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "update_artifact",
		Description: "Update an existing artifact with new content. This creates a new version and calculates a diff from the previous version.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"artifact_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the artifact to update",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "New content for the artifact",
				},
				"commit_message": map[string]interface{}{
					"type":        "string",
					"description": "Optional message describing the changes made",
				},
			},
			"required": []string{"artifact_id", "content"},
		},
		Function: UpdateArtifact,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "get_artifact",
		Description: "Get details and content of a specific artifact, optionally at a specific version",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"artifact_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the artifact to retrieve",
				},
				"version": map[string]interface{}{
					"type":        "integer",
					"description": "Optional specific version to retrieve (defaults to latest)",
				},
			},
			"required": []string{"artifact_id"},
		},
		Function: GetArtifact,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "list_artifacts",
		Description: "List artifacts, optionally filtered by project and/or type",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Filter by project ID",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Filter by artifact type",
					"enum":        []string{"document", "diagram", "code_project", "data_viz", "image"},
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of artifacts to return (default: 50, max: 1000)",
				},
			},
		},
		Function: ListArtifacts,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "list_artifact_versions",
		Description: "Get version history for a specific artifact",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"artifact_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the artifact to get version history for",
				},
			},
			"required": []string{"artifact_id"},
		},
		Function: ListArtifactVersions,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "export_artifact",
		Description: "Export an artifact to a specific format (raw, base64, json) for download or external use",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"artifact_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the artifact to export",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Export format",
					"enum":        []string{"raw", "base64", "json"},
				},
				"version": map[string]interface{}{
					"type":        "integer",
					"description": "Optional specific version to export (defaults to latest)",
				},
			},
			"required": []string{"artifact_id"},
		},
		Function: ExportArtifact,
	})
}
