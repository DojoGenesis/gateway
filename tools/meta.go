package tools

import (
	"context"
	"strings"
)

func SearchTools(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	query, _ := params["query"].(string)
	query = strings.ToLower(query)

	category, _ := params["category"].(string)
	maxResults := 50
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	} else if mr, ok := params["max_results"].(int); ok {
		maxResults = mr
	}

	allTools := GetAllTools()
	matchedTools := []map[string]interface{}{}

	for _, tool := range allTools {
		matches := true

		if query != "" {
			nameLower := strings.ToLower(tool.Name)
			descLower := strings.ToLower(tool.Description)

			if !strings.Contains(nameLower, query) && !strings.Contains(descLower, query) {
				matches = false
			}
		}

		if category != "" {
			toolCategory := getCategoryForTool(tool.Name)
			if toolCategory != category {
				matches = false
			}
		}

		if matches {
			matchedTools = append(matchedTools, map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"category":    getCategoryForTool(tool.Name),
				"parameters":  tool.Parameters,
			})

			if len(matchedTools) >= maxResults {
				break
			}
		}
	}

	return map[string]interface{}{
		"success":  true,
		"query":    query,
		"category": category,
		"count":    len(matchedTools),
		"tools":    matchedTools,
	}, nil
}

func GetToolInfo(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	toolName, ok := params["tool_name"].(string)
	if !ok || toolName == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "tool_name parameter is required",
		}, nil
	}

	tool, err := GetTool(toolName)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"success":     true,
		"name":        tool.Name,
		"description": tool.Description,
		"category":    getCategoryForTool(tool.Name),
		"parameters":  tool.Parameters,
	}, nil
}

func getCategoryForTool(toolName string) string {
	categories := map[string]string{
		"read_file":              "file_operations",
		"write_file":             "file_operations",
		"list_directory":         "file_operations",
		"search_files":           "file_operations",
		"web_search":             "web_operations",
		"fetch_url":              "web_operations",
		"api_call":               "web_operations",
		"web_navigate":           "web_advanced",
		"web_scrape_structured":  "web_advanced",
		"web_screenshot":         "web_advanced",
		"web_monitor":            "web_advanced",
		"web_extract_links":      "web_advanced",
		"execute_python":         "computation",
		"calculate":              "computation",
		"run_command":            "system",
		"create_plan":            "planning",
		"update_plan":            "planning",
		"track_progress":         "planning",
		"validate_plan":          "planning",
		"create_milestone":       "planning",
		"gather_sources":         "research",
		"synthesize_info":        "research",
		"fact_check":             "research",
		"analyze_sentiment":      "research",
		"extract_entities":       "research",
		"search_tools":           "meta",
		"get_tool_info":          "meta",
		"create_project":         "project_management",
		"list_projects":          "project_management",
		"get_project":            "project_management",
		"switch_project":         "project_management",
		"list_templates":         "project_management",
		"update_project":         "project_management",
		"delete_project":         "project_management",
		"create_artifact":        "project_management",
		"update_artifact":        "project_management",
		"get_artifact":           "project_management",
		"list_artifacts":         "project_management",
		"list_artifact_versions": "project_management",
		"export_artifact":        "project_management",
		"prepare_diagram":        "visual",
		"export_diagram":         "visual",
		"prepare_chart":          "visual",
		"export_chart":           "visual",
		"generate_image":         "visual",
	}

	if cat, exists := categories[toolName]; exists {
		return cat
	}
	return "unknown"
}

func init() {
	RegisterTool(&ToolDefinition{
		Name:        "search_tools",
		Description: "Search for available tools by name, description, or category",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query to match against tool names and descriptions",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Filter by category (file_operations, web_operations, web_advanced, computation, system, planning, research, meta, project_management)",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 50)",
					"default":     50,
				},
			},
			"required": []string{},
		},
		Function: SearchTools,
	})

	RegisterTool(&ToolDefinition{
		Name:        "get_tool_info",
		Description: "Get detailed information about a specific tool",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tool_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the tool to get information about",
				},
			},
			"required": []string{"tool_name"},
		},
		Function: GetToolInfo,
	})
}
