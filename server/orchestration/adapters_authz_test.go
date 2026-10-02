package orchestration

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/DojoGenesis/gateway/tools"
)

// ToolInvokerAdapter calls the tool function directly, so it does not pass
// through tools.InvokeTool and has to ask the caller's authorizer itself.
func TestToolInvokerAdapter_AppliesCallerPolicy(t *testing.T) {
	const name = "adapter_authz_probe"
	var ran atomic.Int32
	if err := tools.RegisterTool(&tools.ToolDefinition{
		Name:       name,
		Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		Function: func(context.Context, map[string]interface{}) (map[string]interface{}, error) {
			ran.Add(1)
			return map[string]interface{}{"ok": true}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tools.UnregisterTool(name) })

	adapter := NewToolInvokerAdapter()
	deny := tools.Authorizer(func(n string) error {
		return &tools.ToolAccessError{Tool: n, Code: "denied", Message: "refused"}
	})

	_, err := adapter.InvokeTool(tools.WithAuthorizer(context.Background(), deny), name, nil)
	if err == nil {
		t.Fatal("a refused call returned no error")
	}
	if denied, ok := tools.AsToolAccessError(err); !ok || denied.Code != "denied" {
		t.Fatalf("the refusal must stay recognisable through the adapter's wrapping, got %v", err)
	}
	if ran.Load() != 0 {
		t.Fatal("the adapter ran a tool the caller may not run")
	}

	if _, err := adapter.InvokeTool(context.Background(), name, nil); err != nil {
		t.Fatalf("an unrestricted context must still run the tool: %v", err)
	}
	if ran.Load() != 1 {
		t.Fatalf("expected one run, got %d", ran.Load())
	}
}
