// Package examples provides reference tool implementations demonstrating
// how to register and implement tools using the tools package.
//
// These examples are not imported by the tools package itself.
// They serve as documentation and starting points for tool developers.
package examples

import (
	"context"
	"fmt"
	"time"

	"github.com/DojoGenesis/gateway/tools"
)

// RegisterWebSearchTool registers a web search tool.
// This is a reference implementation showing the tool registration pattern.
func RegisterWebSearchTool() error {
	return tools.RegisterTool(&tools.ToolDefinition{
		Name:        "web_search",
		Description: "Search the web for information and return results",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 10)",
				},
			},
			"required": []interface{}{"query"},
		},
		Function: webSearchFunc,
		Timeout:  30 * time.Second,
	})
}

func webSearchFunc(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	// Placeholder: In a real implementation, this would call a search API.
	_ = limit
	results := []map[string]interface{}{
		{"title": "Example Result 1", "url": "https://example.com/1", "snippet": "First result for: " + query},
		{"title": "Example Result 2", "url": "https://example.com/2", "snippet": "Second result for: " + query},
	}

	return map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	}, nil
}
