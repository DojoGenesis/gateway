package mcp

import (
	"encoding/json"
	"reflect"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

// OCH-154: the SDK's Tool.InputSchema is a typed struct, not a map. It must be
// surfaced faithfully rather than replaced by the generic "input" wrapper.
func TestConvertInputSchema_TypedSDKSchema(t *testing.T) {
	sdkTool := mcpgo.Tool{
		Name: "search",
		InputSchema: mcpgo.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"query": map[string]any{"type": "string"},
			},
			Required: []string{"query"},
		},
	}

	got := convertInputSchema(sdkTool.InputSchema)

	props, ok := got["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("properties missing or wrong type: %#v", got)
	}
	if _, ok := props["query"]; !ok {
		t.Errorf("properties.query missing: %#v", props)
	}
	if _, wrapped := props["input"]; wrapped {
		t.Errorf("got generic input wrapper: %#v", got)
	}
	if !reflect.DeepEqual(got["required"], []interface{}{"query"}) {
		t.Errorf("required = %#v, want [query]", got["required"])
	}
}

func TestConvertInputSchema_RawMessage(t *testing.T) {
	raw := json.RawMessage(`{"type":"object","properties":{"content":{"type":"string"}},"required":["content"]}`)
	got := convertInputSchema(raw)
	props, _ := got["properties"].(map[string]interface{})
	if _, ok := props["content"]; !ok {
		t.Errorf("properties.content missing: %#v", got)
	}
	if _, wrapped := props["input"]; wrapped {
		t.Errorf("got generic input wrapper: %#v", got)
	}
}

func TestConvertInputSchema_EmptyTypedSchemaFallsBack(t *testing.T) {
	got := convertInputSchema(mcpgo.ToolInputSchema{})
	props, ok := got["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("fallback missing properties: %#v", got)
	}
	if _, ok := props["input"]; !ok {
		t.Errorf("expected generic input fallback, got %#v", got)
	}
	if got["type"] != "object" {
		t.Errorf("fallback type = %v", got["type"])
	}
}
