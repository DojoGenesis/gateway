package tools

import (
	"context"
	"os"
	"path/filepath"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	projectIDKey      contextKey = "project_id"
	workspaceRootKey  contextKey = "workspace_root"
)

// WithProjectID adds project_id to the context for request-scoped project isolation.
// This ensures concurrent requests don't interfere with each other's project context.
func WithProjectID(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, projectIDKey, projectID)
}

// GetProjectIDFromContext retrieves the project_id from context if present.
// Returns empty string if not found in context.
func GetProjectIDFromContext(ctx context.Context) string {
	if projectID, ok := ctx.Value(projectIDKey).(string); ok {
		return projectID
	}
	return ""
}

// injectProjectIDIntoParams adds project_id from context to params if not already present.
func injectProjectIDIntoParams(ctx context.Context, params map[string]interface{}) map[string]interface{} {
	if _, exists := params["project_id"]; exists {
		return params
	}

	if projectID := GetProjectIDFromContext(ctx); projectID != "" {
		newParams := make(map[string]interface{}, len(params)+1)
		for k, v := range params {
			newParams[k] = v
		}
		newParams["project_id"] = projectID
		return newParams
	}

	return params
}

// WithWorkspaceRoot adds the user's workspace directory to the context.
// File tools use this as their base directory when resolving relative paths,
// allowing the agent to reference project files by relative path even though
// the gateway process runs from a different working directory.
func WithWorkspaceRoot(ctx context.Context, root string) context.Context {
	return context.WithValue(ctx, workspaceRootKey, root)
}

// GetWorkspaceRootFromContext retrieves the workspace root from context.
// Falls back to DOJO_WORKSPACE_ROOT env var, then to the process CWD.
func GetWorkspaceRootFromContext(ctx context.Context) string {
	if root, ok := ctx.Value(workspaceRootKey).(string); ok && root != "" {
		return root
	}
	if root := os.Getenv("DOJO_WORKSPACE_ROOT"); root != "" {
		return root
	}
	cwd, _ := os.Getwd()
	return cwd
}

// resolveFilePath resolves a file path against the workspace root from context.
// Absolute paths are returned as-is (cleaned). Relative paths are joined with
// the workspace root so agents can reference project files by relative path.
func resolveFilePath(ctx context.Context, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(GetWorkspaceRootFromContext(ctx), path)
}
