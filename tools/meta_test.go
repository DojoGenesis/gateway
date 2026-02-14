package tools

import (
	"context"
	"testing"
)

func TestSearchTools(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		params       map[string]interface{}
		wantSuccess  bool
		minResults   int
		wantContains []string
	}{
		{
			name: "search by query - file",
			params: map[string]interface{}{
				"query": "file",
			},
			wantSuccess:  true,
			minResults:   2,
			wantContains: []string{"read_file", "write_file"},
		},
		{
			name: "search by query - web",
			params: map[string]interface{}{
				"query": "web",
			},
			wantSuccess:  true,
			minResults:   5,
			wantContains: []string{"web_search", "web_navigate"},
		},
		{
			name: "search by category - file_operations",
			params: map[string]interface{}{
				"category": "file_operations",
			},
			wantSuccess:  true,
			minResults:   4,
			wantContains: []string{"read_file", "write_file", "list_directory", "search_files"},
		},
		{
			name: "search by category - planning",
			params: map[string]interface{}{
				"category": "planning",
			},
			wantSuccess:  true,
			minResults:   5,
			wantContains: []string{"create_plan", "update_plan", "track_progress", "validate_plan", "create_milestone"},
		},
		{
			name: "search by category - research",
			params: map[string]interface{}{
				"category": "research",
			},
			wantSuccess:  true,
			minResults:   5,
			wantContains: []string{"gather_sources", "synthesize_info", "fact_check", "analyze_sentiment", "extract_entities"},
		},
		{
			name:        "search all tools",
			params:      map[string]interface{}{},
			wantSuccess: true,
			minResults:  25,
		},
		{
			name: "search with max_results",
			params: map[string]interface{}{
				"max_results": 5,
			},
			wantSuccess: true,
			minResults:  5,
		},
		{
			name: "search by query and category",
			params: map[string]interface{}{
				"query":    "plan",
				"category": "planning",
			},
			wantSuccess:  true,
			minResults:   2,
			wantContains: []string{"create_plan", "update_plan"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SearchTools(ctx, tt.params)
			if err != nil {
				t.Fatalf("SearchTools() error = %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Errorf("SearchTools() success = %v, want %v", success, tt.wantSuccess)
			}

			if !success {
				return
			}

			count, _ := result["count"].(int)
			if count < tt.minResults {
				t.Errorf("SearchTools() count = %d, want at least %d", count, tt.minResults)
			}

			tools, _ := result["tools"].([]map[string]interface{})
			toolNames := make([]string, len(tools))
			for i, tool := range tools {
				toolNames[i] = tool["name"].(string)
			}

			for _, expectedTool := range tt.wantContains {
				found := false
				for _, name := range toolNames {
					if name == expectedTool {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SearchTools() missing expected tool %s in results: %v", expectedTool, toolNames)
				}
			}
		})
	}
}

func TestGetToolInfo(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		params       map[string]interface{}
		wantSuccess  bool
		wantError    bool
		wantName     string
		wantCategory string
	}{
		{
			name: "get info for read_file",
			params: map[string]interface{}{
				"tool_name": "read_file",
			},
			wantSuccess:  true,
			wantName:     "read_file",
			wantCategory: "file_operations",
		},
		{
			name: "get info for web_search",
			params: map[string]interface{}{
				"tool_name": "web_search",
			},
			wantSuccess:  true,
			wantName:     "web_search",
			wantCategory: "web_operations",
		},
		{
			name: "get info for create_plan",
			params: map[string]interface{}{
				"tool_name": "create_plan",
			},
			wantSuccess:  true,
			wantName:     "create_plan",
			wantCategory: "planning",
		},
		{
			name: "get info for gather_sources",
			params: map[string]interface{}{
				"tool_name": "gather_sources",
			},
			wantSuccess:  true,
			wantName:     "gather_sources",
			wantCategory: "research",
		},
		{
			name: "get info for non-existent tool",
			params: map[string]interface{}{
				"tool_name": "non_existent_tool",
			},
			wantSuccess: false,
			wantError:   true,
		},
		{
			name:        "get info without tool_name",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetToolInfo(ctx, tt.params)
			if err != nil {
				t.Fatalf("GetToolInfo() error = %v", err)
			}

			success, _ := result["success"].(bool)
			if success != tt.wantSuccess {
				t.Errorf("GetToolInfo() success = %v, want %v", success, tt.wantSuccess)
			}

			if tt.wantError {
				if _, hasError := result["error"]; !hasError {
					t.Errorf("GetToolInfo() expected error field in result")
				}
				return
			}

			if !success {
				return
			}

			name, _ := result["name"].(string)
			if name != tt.wantName {
				t.Errorf("GetToolInfo() name = %v, want %v", name, tt.wantName)
			}

			category, _ := result["category"].(string)
			if category != tt.wantCategory {
				t.Errorf("GetToolInfo() category = %v, want %v", category, tt.wantCategory)
			}

			if _, hasParams := result["parameters"]; !hasParams {
				t.Errorf("GetToolInfo() missing parameters field")
			}

			if _, hasDesc := result["description"]; !hasDesc {
				t.Errorf("GetToolInfo() missing description field")
			}
		})
	}
}

func TestGetCategoryForTool(t *testing.T) {
	tests := []struct {
		toolName string
		want     string
	}{
		{"read_file", "file_operations"},
		{"write_file", "file_operations"},
		{"web_search", "web_operations"},
		{"web_navigate", "web_advanced"},
		{"execute_python", "computation"},
		{"run_command", "system"},
		{"create_plan", "planning"},
		{"gather_sources", "research"},
		{"search_tools", "meta"},
		{"get_tool_info", "meta"},
		{"unknown_tool", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			got := getCategoryForTool(tt.toolName)
			if got != tt.want {
				t.Errorf("getCategoryForTool(%s) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}
