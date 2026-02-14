package tools

import (
	"context"
	"fmt"
	"strings"
)

// ContextAwareRegistry wraps the global tool registry to implement gateway.ToolRegistry interface.
// It provides context-aware methods for tool registration and lookup.
type ContextAwareRegistry struct{}

// NewContextAwareRegistry creates a new context-aware tool registry wrapper.
func NewContextAwareRegistry() *ContextAwareRegistry {
	return &ContextAwareRegistry{}
}

// Register adds a new tool to the registry.
// Context is passed for future cancellation support but not currently used.
func (r *ContextAwareRegistry) Register(ctx context.Context, def *ToolDefinition) error {
	return RegisterTool(def)
}

// Get retrieves a tool by its exact name.
func (r *ContextAwareRegistry) Get(ctx context.Context, name string) (*ToolDefinition, error) {
	return GetTool(name)
}

// List returns all registered tools in the registry.
func (r *ContextAwareRegistry) List(ctx context.Context) ([]*ToolDefinition, error) {
	tools := GetAllTools()
	return tools, nil
}

// ListByNamespace returns all tools matching a namespace prefix.
// For example, "composio." returns all tools registered by the Composio MCP server.
// The prefix match is case-sensitive.
func (r *ContextAwareRegistry) ListByNamespace(ctx context.Context, prefix string) ([]*ToolDefinition, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	matching := make([]*ToolDefinition, 0)
	for name, tool := range toolRegistry {
		if strings.HasPrefix(name, prefix) {
			matching = append(matching, tool)
		}
	}

	return matching, nil
}

// Delete removes a tool from the registry by name.
// This is not part of the gateway.ToolRegistry interface but provided for completeness.
func (r *ContextAwareRegistry) Delete(ctx context.Context, name string) error {
	return UnregisterTool(name)
}

// Clear removes all tools from the registry.
// This is not part of the gateway.ToolRegistry interface but provided for testing.
func (r *ContextAwareRegistry) Clear(ctx context.Context) error {
	ClearRegistry()
	return nil
}

// Count returns the total number of registered tools.
// This is not part of the gateway.ToolRegistry interface but provided for observability.
func (r *ContextAwareRegistry) Count(ctx context.Context) (int, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return len(toolRegistry), nil
}

// Exists checks if a tool with the given name exists in the registry.
// This is not part of the gateway.ToolRegistry interface but provided for convenience.
func (r *ContextAwareRegistry) Exists(ctx context.Context, name string) (bool, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, exists := toolRegistry[name]
	return exists, nil
}

// GetNamespaces returns all unique namespace prefixes currently in use.
// A namespace is the portion of a tool name before the first dot (e.g., "composio" from "composio.create_task").
func (r *ContextAwareRegistry) GetNamespaces(ctx context.Context) ([]string, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	namespaces := make(map[string]bool)
	for name := range toolRegistry {
		if idx := strings.Index(name, "."); idx > 0 {
			namespace := name[:idx]
			namespaces[namespace] = true
		}
	}

	result := make([]string, 0, len(namespaces))
	for ns := range namespaces {
		result = append(result, ns)
	}

	return result, nil
}

// ValidateTool checks if a tool definition is valid before registration.
func ValidateTool(def *ToolDefinition) error {
	if def == nil {
		return fmt.Errorf("tool definition cannot be nil")
	}
	if def.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if def.Function == nil {
		return fmt.Errorf("tool function cannot be nil")
	}
	if def.Description == "" {
		return fmt.Errorf("tool description cannot be empty")
	}
	return nil
}
