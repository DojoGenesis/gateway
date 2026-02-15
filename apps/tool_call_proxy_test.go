package apps

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

// mockToolRegistry implements gateway.ToolRegistry for testing.
type mockToolRegistry struct {
	tools map[string]*tools.ToolDefinition
}

func newMockToolRegistry() *mockToolRegistry {
	return &mockToolRegistry{tools: make(map[string]*tools.ToolDefinition)}
}

func (m *mockToolRegistry) Register(_ context.Context, def *tools.ToolDefinition) error {
	m.tools[def.Name] = def
	return nil
}

func (m *mockToolRegistry) Get(_ context.Context, name string) (*tools.ToolDefinition, error) {
	t, ok := m.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return t, nil
}

func (m *mockToolRegistry) List(_ context.Context) ([]*tools.ToolDefinition, error) {
	var result []*tools.ToolDefinition
	for _, t := range m.tools {
		result = append(result, t)
	}
	return result, nil
}

func (m *mockToolRegistry) ListByNamespace(_ context.Context, prefix string) ([]*tools.ToolDefinition, error) {
	return nil, nil
}

func TestToolCallProxy_Authorized(t *testing.T) {
	reg := newMockToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name:        "echo",
		Description: "Echoes input",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"echo": args["message"]}, nil
		},
	})

	policy := NewAuthorizationPolicy()
	policy.GrantToolAccess("app-1", "echo")

	proxy := NewToolCallProxy(reg, policy)

	resp, err := proxy.ProxyCall(context.Background(), &ToolCallRequest{
		AppID:     "app-1",
		ToolName:  "echo",
		Arguments: map[string]interface{}{"message": "hello"},
		SessionID: "session-1",
	})

	if err != nil {
		t.Fatalf("ProxyCall failed: %v", err)
	}
	if resp.IsError {
		t.Fatalf("unexpected error response: %s", resp.Error)
	}
	if resp.Result["echo"] != "hello" {
		t.Errorf("result = %v, want echo=hello", resp.Result)
	}
}

func TestToolCallProxy_Unauthorized(t *testing.T) {
	reg := newMockToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name: "echo",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{}, nil
		},
	})

	policy := NewAuthorizationPolicy()
	// No grants for app-1

	proxy := NewToolCallProxy(reg, policy)

	resp, err := proxy.ProxyCall(context.Background(), &ToolCallRequest{
		AppID:    "app-1",
		ToolName: "echo",
	})

	if err != nil {
		t.Fatalf("ProxyCall returned Go error: %v", err)
	}
	if !resp.IsError {
		t.Fatal("expected error response for unauthorized call")
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestToolCallProxy_ToolNotFound(t *testing.T) {
	reg := newMockToolRegistry()
	policy := NewAuthorizationPolicy()
	policy.GrantAllToolAccess("app-1")

	proxy := NewToolCallProxy(reg, policy)

	resp, err := proxy.ProxyCall(context.Background(), &ToolCallRequest{
		AppID:    "app-1",
		ToolName: "nonexistent",
	})

	if err != nil {
		t.Fatalf("ProxyCall returned Go error: %v", err)
	}
	if !resp.IsError {
		t.Fatal("expected error response for tool not found")
	}
}

func TestToolCallProxy_ToolExecutionError(t *testing.T) {
	reg := newMockToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name: "fail",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("intentional failure")
		},
	})

	policy := NewAuthorizationPolicy()
	policy.GrantToolAccess("app-1", "fail")

	proxy := NewToolCallProxy(reg, policy)

	resp, err := proxy.ProxyCall(context.Background(), &ToolCallRequest{
		AppID:    "app-1",
		ToolName: "fail",
	})

	if err != nil {
		t.Fatalf("ProxyCall returned Go error: %v", err)
	}
	if !resp.IsError {
		t.Fatal("expected error response for tool execution error")
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty error message for execution failure")
	}
}

func TestToolCallProxy_Timeout(t *testing.T) {
	reg := newMockToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name: "slow",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return map[string]interface{}{"ok": true}, nil
			}
		},
	})

	policy := NewAuthorizationPolicy()
	policy.GrantToolAccess("app-1", "slow")

	proxy := NewToolCallProxy(reg, policy)
	proxy.SetDefaultTimeout(50 * time.Millisecond)

	resp, err := proxy.ProxyCall(context.Background(), &ToolCallRequest{
		AppID:    "app-1",
		ToolName: "slow",
	})

	if err != nil {
		t.Fatalf("ProxyCall returned Go error: %v", err)
	}
	if !resp.IsError {
		t.Fatal("expected error response for timeout")
	}
}

func TestToolCallProxy_NilRequest(t *testing.T) {
	reg := newMockToolRegistry()
	policy := NewAuthorizationPolicy()
	proxy := NewToolCallProxy(reg, policy)

	resp, err := proxy.ProxyCall(context.Background(), nil)
	if err != nil {
		t.Fatalf("ProxyCall returned Go error: %v", err)
	}
	if !resp.IsError {
		t.Fatal("expected error response for nil request")
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty error message for nil request")
	}
}

func TestToolCallProxy_NilArguments(t *testing.T) {
	reg := newMockToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name: "noargs",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if args == nil {
				return nil, fmt.Errorf("args should not be nil")
			}
			return map[string]interface{}{"ok": true}, nil
		},
	})

	policy := NewAuthorizationPolicy()
	policy.GrantToolAccess("app-1", "noargs")

	proxy := NewToolCallProxy(reg, policy)

	resp, err := proxy.ProxyCall(context.Background(), &ToolCallRequest{
		AppID:     "app-1",
		ToolName:  "noargs",
		Arguments: nil,
	})

	if err != nil {
		t.Fatalf("ProxyCall failed: %v", err)
	}
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
}
