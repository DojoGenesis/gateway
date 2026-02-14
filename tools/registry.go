package tools

import (
	"fmt"
	"sync"
)

var (
	toolRegistry = make(map[string]*ToolDefinition)
	registryMu   sync.RWMutex
)

// RegisterTool adds a tool to the global registry.
// Returns error if tool already registered or definition is invalid.
func RegisterTool(def *ToolDefinition) error {
	if def == nil {
		return fmt.Errorf("tool definition cannot be nil")
	}
	if def.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if def.Function == nil {
		return fmt.Errorf("tool function cannot be nil")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := toolRegistry[def.Name]; exists {
		return fmt.Errorf("tool already registered: %s", def.Name)
	}

	toolRegistry[def.Name] = def
	return nil
}

// GetTool retrieves a tool by name.
// Returns error if tool not found.
func GetTool(name string) (*ToolDefinition, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	tool, exists := toolRegistry[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// GetAllTools returns a snapshot of all registered tools.
func GetAllTools() []*ToolDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	tools := make([]*ToolDefinition, 0, len(toolRegistry))
	for _, tool := range toolRegistry {
		tools = append(tools, tool)
	}
	return tools
}

// UnregisterTool removes a tool from the registry.
// Returns error if tool not found.
func UnregisterTool(name string) error {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := toolRegistry[name]; !exists {
		return fmt.Errorf("tool not found: %s", name)
	}

	delete(toolRegistry, name)
	return nil
}

// ClearRegistry removes all tools (useful for testing).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()

	toolRegistry = make(map[string]*ToolDefinition)
}
