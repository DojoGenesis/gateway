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

// intentToolSets maps agent intent strings to the minimal tool names required.
// This is the primary lever for reducing input token count — sending 6-10 tools
// instead of all 33 cuts tool-schema tokens by ~70%, helping stay under rate limits.
var intentToolSets = map[string][]string{
	"BUILD": {
		"read_file", "write_file", "list_directory", "search_files",
		"run_command", "calculate",
	},
	"DEBUG": {
		"read_file", "write_file", "search_files", "list_directory",
		"run_command", "calculate",
	},
	"THINK": {
		"read_file", "list_directory", "search_files",
		"gather_sources", "synthesize_info", "calculate",
	},
	"SEARCH": {
		"web_search", "fetch_url", "gather_sources", "synthesize_info",
		"read_file", "search_files",
	},
	"GENERAL": {
		"read_file", "write_file", "list_directory", "search_files",
		"run_command", "web_search", "fetch_url", "calculate",
	},
}

// GetToolsForIntent returns the tool subset appropriate for the given intent string.
// Sending only relevant tools reduces input token count, which is critical for
// providers with tight TPM rate limits.
// Falls back to GetAllTools() for unknown intent strings.
func GetToolsForIntent(intent string) []*ToolDefinition {
	names, ok := intentToolSets[intent]
	if !ok {
		return GetAllTools()
	}

	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make([]*ToolDefinition, 0, len(names))
	for _, name := range names {
		if tool, exists := toolRegistry[name]; exists {
			result = append(result, tool)
		}
	}
	return result
}

// ClearRegistry removes all tools (useful for testing).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()

	toolRegistry = make(map[string]*ToolDefinition)
}
