package apptools

import (
	"context"
	"fmt"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/projects"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

var projectManager *projects.ProjectManager

func InitializeProjectTools(pm *projects.ProjectManager) {
	projectManager = pm
}

func CreateProject(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if projectManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "project manager not initialized",
		}, nil
	}

	name := tools.GetStringParam(params, "name", "")
	if name == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "project name is required",
		}, nil
	}

	description := tools.GetStringParam(params, "description", "")
	templateID := tools.GetStringParam(params, "template_id", "")

	project, err := projectManager.CreateProject(ctx, name, description, templateID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to create project: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"project": project,
		"message": fmt.Sprintf("Project '%s' created successfully", project.Name),
	}, nil
}

func ListProjects(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if projectManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "project manager not initialized",
		}, nil
	}

	status := tools.GetStringParam(params, "status", "")

	projectList, err := projectManager.ListProjects(ctx, status)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to list projects: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success":  true,
		"count":    len(projectList),
		"projects": projectList,
	}, nil
}

func GetProject(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if projectManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "project manager not initialized",
		}, nil
	}

	projectID := tools.GetStringParam(params, "project_id", "")
	if projectID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "project_id is required",
		}, nil
	}

	project, err := projectManager.GetProject(ctx, projectID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to get project: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"project": project,
	}, nil
}

func SwitchProject(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if projectManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "project manager not initialized",
		}, nil
	}

	projectID := tools.GetStringParam(params, "project_id", "")
	if projectID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "project_id is required",
		}, nil
	}

	project, err := projectManager.GetProject(ctx, projectID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to switch to project: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"project": map[string]interface{}{
			"id":   project.ID,
			"name": project.Name,
		},
		"message":    fmt.Sprintf("Switched to project '%s'", project.Name),
		"project_id": project.ID,
	}, nil
}

func ListTemplates(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if projectManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "project manager not initialized",
		}, nil
	}

	templates, err := projectManager.ListTemplates(ctx)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to list templates: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success":   true,
		"count":     len(templates),
		"templates": templates,
	}, nil
}

func UpdateProject(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if projectManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "project manager not initialized",
		}, nil
	}

	projectID := tools.GetStringParam(params, "project_id", "")
	if projectID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "project_id is required",
		}, nil
	}

	name := tools.GetStringParam(params, "name", "")
	description := tools.GetStringParam(params, "description", "")
	status := tools.GetStringParam(params, "status", "")
	settings := tools.GetMapParam(params, "settings", nil)
	metadata := tools.GetMapParam(params, "metadata", nil)

	if name == "" && description == "" && status == "" && settings == nil && metadata == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "at least one field must be provided for update (name, description, status, settings, or metadata)",
		}, nil
	}

	project, err := projectManager.UpdateProject(ctx, projectID, name, description, status, settings, metadata)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to update project: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"project": project,
		"message": fmt.Sprintf("Project '%s' updated successfully", project.Name),
	}, nil
}

func DeleteProject(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if projectManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "project manager not initialized",
		}, nil
	}

	projectID := tools.GetStringParam(params, "project_id", "")
	if projectID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "project_id is required",
		}, nil
	}

	err := projectManager.DeleteProject(ctx, projectID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to delete project: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"message": "Project deleted successfully",
	}, nil
}

func init() {
	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "create_project",
		Description: "Create a new project workspace with optional template",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the project",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Description of the project",
				},
				"template_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the template to use (e.g., 'research-report', 'software-design', 'data-analysis', 'creative-studio')",
				},
			},
			"required": []string{"name"},
		},
		Function: CreateProject,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "list_projects",
		Description: "List all projects, optionally filtered by status",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status: 'active', 'archived', or 'deleted'",
					"enum":        []string{"active", "archived", "deleted"},
				},
			},
		},
		Function: ListProjects,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "get_project",
		Description: "Get details of a specific project",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the project to retrieve",
				},
			},
			"required": []string{"project_id"},
		},
		Function: GetProject,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "switch_project",
		Description: "Switch to a different project context",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the project to switch to",
				},
			},
			"required": []string{"project_id"},
		},
		Function: SwitchProject,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "list_templates",
		Description: "List available project templates",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Function: ListTemplates,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "update_project",
		Description: "Update project details including name, description, status, settings, or metadata",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the project to update",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "New name for the project",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "New description for the project",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "New status: 'active', 'archived', or 'deleted'",
					"enum":        []string{"active", "archived", "deleted"},
				},
				"settings": map[string]interface{}{
					"type":        "object",
					"description": "Project settings as key-value pairs",
				},
				"metadata": map[string]interface{}{
					"type":        "object",
					"description": "Project metadata as key-value pairs",
				},
			},
			"required": []string{"project_id"},
		},
		Function: UpdateProject,
	})

	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "delete_project",
		Description: "Delete a project (marks as deleted but doesn't remove from database)",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the project to delete",
				},
			},
			"required": []string{"project_id"},
		},
		Function: DeleteProject,
	})
}
