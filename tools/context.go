package tools

import "context"

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	projectIDKey contextKey = "project_id"
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
