package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	_ = RegisterTool(&ToolDefinition{
		Name:        "get_document",
		Description: "Fetch the current content of a ZenithScience document by its ID. Returns the full document JSON including sections and metadata.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"document_id": map[string]interface{}{
					"type":        "string",
					"description": "The unique document ID to fetch.",
				},
			},
			"required": []string{"document_id"},
		},
		Function: getDocument,
	})
}

func getDocument(_ context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	docID, ok := params["document_id"].(string)
	if !ok || docID == "" {
		return nil, fmt.Errorf("document_id is required")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}

	docPath := filepath.Join(home, ".zen-sci", "documents", docID+".json")
	data, err := os.ReadFile(docPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("document not found: %s", docID)
		}
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("document file is corrupt: %w", err)
	}

	return doc, nil
}
