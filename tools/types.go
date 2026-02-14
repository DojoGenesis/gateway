package tools

import (
	"context"
	"encoding/json"
	"time"
)

type ToolFunc func(context.Context, map[string]interface{}) (map[string]interface{}, error)

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Function    ToolFunc               `json:"-"`
	Timeout     time.Duration          `json:"timeout,omitempty"` // Per-tool timeout override; 0 means use global default
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

func FormatToolResult(result map[string]interface{}) string {
	data, err := json.Marshal(result)
	if err != nil {
		return err.Error()
	}
	return string(data)
}
