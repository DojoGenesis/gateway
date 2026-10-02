package agent

import (
	"context"
	"sync/atomic"
	"testing"

	providerpkg "github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/tools"
)

// recordingToolProvider remembers which tools each completion request offered.
type recordingToolProvider struct {
	mockToolProvider
	offered [][]string
}

func (r *recordingToolProvider) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	names := make([]string, 0, len(req.Tools))
	for _, tl := range req.Tools {
		names = append(names, tl.Name)
	}
	r.offered = append(r.offered, names)
	return r.mockToolProvider.GenerateCompletion(ctx, req)
}

func offeredContains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// TestPrimaryAgent_ToolCallsRunWithTheCallersAuthority: the agent loop reaches
// tools.InvokeTool on behalf of whoever sent the chat message, so a tool the
// caller may not run must be neither offered to the model nor executed if the
// model asks for it anyway.
func TestPrimaryAgent_ToolCallsRunWithTheCallersAuthority(t *testing.T) {
	tools.ClearRegistry()
	defer tools.ClearRegistry()

	var readFileRan, calcRan atomic.Int32
	probe := func(name string, ran *atomic.Int32, arg string) *tools.ToolDefinition {
		return &tools.ToolDefinition{
			Name: name,
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{arg: map[string]interface{}{"type": "string"}},
				"required":   []interface{}{arg},
			},
			Function: func(context.Context, map[string]interface{}) (map[string]interface{}, error) {
				ran.Add(1)
				return map[string]interface{}{"success": true}, nil
			},
		}
	}
	if err := tools.RegisterTool(probe("read_file", &readFileRan, "file_path")); err != nil {
		t.Fatal(err)
	}
	if err := tools.RegisterTool(probe("calculate", &calcRan, "expression")); err != nil {
		t.Fatal(err)
	}

	run := func(ctx context.Context) *recordingToolProvider {
		pm := newMockPluginManager()
		prov := &recordingToolProvider{mockToolProvider: mockToolProvider{
			toolCallResponses: [][]providerpkg.ToolCall{{
				{ID: "c1", Name: "read_file", Arguments: map[string]interface{}{"file_path": "/etc/hosts"}},
				{ID: "c2", Name: "calculate", Arguments: map[string]interface{}{"expression": "1+1"}},
			}},
		}}
		pm.AddProvider("mock", prov)
		a := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")
		if _, err := a.HandleQueryWithTools(ctx, QueryRequest{Query: "read /etc/hosts and add", UserID: "u", UserTier: "guest"}); err != nil {
			t.Fatalf("HandleQueryWithTools: %v", err)
		}
		return prov
	}

	deny := tools.Authorizer(func(name string) error {
		if name == "read_file" {
			return &tools.ToolAccessError{Tool: name, Code: "denied", Message: "refused"}
		}
		return nil
	})

	prov := run(tools.WithAuthorizer(context.Background(), deny))
	if readFileRan.Load() != 0 {
		t.Fatal("the agent loop ran a tool the caller may not run")
	}
	if calcRan.Load() != 1 {
		t.Fatalf("a permitted tool should still run; ran %d times", calcRan.Load())
	}
	if len(prov.offered) == 0 || offeredContains(prov.offered[0], "read_file") || !offeredContains(prov.offered[0], "calculate") {
		t.Fatalf("the model was offered %v; it should see calculate but not read_file", prov.offered)
	}

	// Control: with no authorizer bound the same conversation runs both, which
	// is what proves the refusal above came from the authorizer.
	readFileRan.Store(0)
	calcRan.Store(0)
	prov = run(context.Background())
	if readFileRan.Load() != 1 || calcRan.Load() != 1 {
		t.Fatalf("unrestricted run: read_file=%d calculate=%d, want 1 and 1", readFileRan.Load(), calcRan.Load())
	}
	if !offeredContains(prov.offered[0], "read_file") {
		t.Fatalf("unrestricted run should offer read_file, got %v", prov.offered[0])
	}
}
