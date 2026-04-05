package tools

import (
	"context"
	"testing"
)

func TestToolIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("All tools are registered", func(t *testing.T) {
		if err := VerifyAllTools(); err != nil {
			t.Fatalf("Tool verification failed: %v", err)
		}

		count := GetToolCount()
		if count != 33 {
			t.Errorf("Expected 33 tools, got %d", count)
		}
	})

	t.Run("Search tools works correctly", func(t *testing.T) {
		result, err := SearchTools(ctx, map[string]interface{}{
			"query": "file",
		})
		if err != nil {
			t.Fatalf("SearchTools failed: %v", err)
		}

		success, _ := result["success"].(bool)
		if !success {
			t.Errorf("SearchTools returned success: false")
		}

		count, _ := result["count"].(int)
		if count < 2 {
			t.Errorf("Expected at least 2 tools for 'file' query, got %d", count)
		}
	})

	t.Run("Get tool info works correctly", func(t *testing.T) {
		result, err := GetToolInfo(ctx, map[string]interface{}{
			"tool_name": "read_file",
		})
		if err != nil {
			t.Fatalf("GetToolInfo failed: %v", err)
		}

		success, _ := result["success"].(bool)
		if !success {
			t.Errorf("GetToolInfo returned success: false")
		}

		name, _ := result["name"].(string)
		if name != "read_file" {
			t.Errorf("Expected name 'read_file', got %s", name)
		}
	})

	t.Run("All categories have correct tool counts", func(t *testing.T) {
		categories := ListAllTools()

		expectedCounts := map[string]int{
			"file_operations": 5,
			"web_operations":  3,
			"web_advanced":    5,
			"computation":     2,
			"system":          1,
			"planning":        5,
			"research":        5,
			"meta":            2,
			"visual":          5,
		}

		for category, expectedCount := range expectedCounts {
			tools, exists := categories[category]
			if !exists {
				t.Errorf("Category %s not found", category)
				continue
			}
			if len(tools) != expectedCount {
				t.Errorf("Category %s: expected %d tools, got %d", category, expectedCount, len(tools))
			}
		}
	})

	t.Run("All tool categories are correct", func(t *testing.T) {
		expectedCategories := map[string]string{
			"read_file":             "file_operations",
			"write_file":            "file_operations",
			"list_directory":        "file_operations",
			"search_files":          "file_operations",
			"get_document":          "file_operations",
			"web_search":            "web_operations",
			"fetch_url":             "web_operations",
			"api_call":              "web_operations",
			"web_navigate":          "web_advanced",
			"web_scrape_structured": "web_advanced",
			"web_screenshot":        "web_advanced",
			"web_monitor":           "web_advanced",
			"web_extract_links":     "web_advanced",
			"execute_python":        "computation",
			"calculate":             "computation",
			"run_command":           "system",
			"create_plan":           "planning",
			"update_plan":           "planning",
			"track_progress":        "planning",
			"validate_plan":         "planning",
			"create_milestone":      "planning",
			"gather_sources":        "research",
			"synthesize_info":       "research",
			"fact_check":            "research",
			"analyze_sentiment":     "research",
			"extract_entities":      "research",
			"search_tools":          "meta",
			"get_tool_info":         "meta",
			"prepare_diagram":       "visual",
			"export_diagram":        "visual",
			"prepare_chart":         "visual",
			"export_chart":          "visual",
			"generate_image":        "visual",
		}

		for toolName, expectedCategory := range expectedCategories {
			actualCategory := getCategoryForTool(toolName)
			if actualCategory != expectedCategory {
				t.Errorf("Tool %s: expected category %s, got %s",
					toolName, expectedCategory, actualCategory)
			}
		}
	})
}
