package examples

import (
	"context"
	"fmt"
	"os"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

// RegisterFileTools registers read_file and write_file reference tools.
func RegisterFileTools() error {
	if err := tools.RegisterTool(&tools.ToolDefinition{
		Name:        "read_file",
		Description: "Read the contents of a file from the local filesystem",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to read",
				},
			},
			"required": []interface{}{"path"},
		},
		Function: readFileFunc,
	}); err != nil {
		return err
	}

	return tools.RegisterTool(&tools.ToolDefinition{
		Name:        "write_file",
		Description: "Write content to a file on the local filesystem",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to write",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content to write to the file",
				},
			},
			"required": []interface{}{"path", "content"},
		},
		Function: writeFileFunc,
	})
}

func readFileFunc(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return map[string]interface{}{
		"content": string(data),
		"path":    path,
		"size":    len(data),
	}, nil
}

func writeFileFunc(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required")
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
		"size":    len(content),
	}, nil
}
