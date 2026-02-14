package tools

import (
	"fmt"
	"sort"
)

func VerifyAllTools() error {
	expectedTools := []string{
		"read_file",
		"write_file",
		"list_directory",
		"search_files",
		"web_search",
		"fetch_url",
		"api_call",
		"web_navigate",
		"web_scrape_structured",
		"web_screenshot",
		"web_monitor",
		"web_extract_links",
		"execute_python",
		"calculate",
		"run_command",
		"create_plan",
		"update_plan",
		"track_progress",
		"validate_plan",
		"create_milestone",
		"gather_sources",
		"synthesize_info",
		"fact_check",
		"analyze_sentiment",
		"extract_entities",
		"search_tools",
		"get_tool_info",
		"prepare_diagram",
		"export_diagram",
		"prepare_chart",
		"export_chart",
		"generate_image",
	}

	allTools := GetAllTools()
	registeredTools := make(map[string]bool)
	for _, tool := range allTools {
		registeredTools[tool.Name] = true
	}

	missing := []string{}
	for _, expectedTool := range expectedTools {
		if !registeredTools[expectedTool] {
			missing = append(missing, expectedTool)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing tools: %v", missing)
	}

	if len(allTools) != len(expectedTools) {
		return fmt.Errorf("expected %d tools, got %d", len(expectedTools), len(allTools))
	}

	return nil
}

func ListAllTools() map[string][]string {
	categories := map[string][]string{
		"file_operations":    {},
		"web_operations":     {},
		"web_advanced":       {},
		"computation":        {},
		"system":             {},
		"planning":           {},
		"research":           {},
		"meta":   {},
		"visual": {},
	}

	allTools := GetAllTools()
	for _, tool := range allTools {
		category := getCategoryForTool(tool.Name)
		categories[category] = append(categories[category], tool.Name)
	}

	for category := range categories {
		sort.Strings(categories[category])
	}

	return categories
}

func GetToolCount() int {
	return len(GetAllTools())
}
