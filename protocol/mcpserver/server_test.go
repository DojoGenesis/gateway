package mcpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	mcpserver "github.com/DojoGenesis/gateway/protocol/mcpserver"
)

func newTestServer(t *testing.T) mcpserver.Server {
	t.Helper()
	srv, err := mcpserver.NewServer(mcpserver.DefaultConfig())
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv
}

func TestNewServer(t *testing.T) {
	srv, err := mcpserver.NewServer(mcpserver.DefaultConfig())
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestStartStop(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Double start should fail.
	if err := srv.Start(ctx); err == nil {
		t.Error("Double Start: expected error")
	}

	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestRegisterTool(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop(context.Background())

	err := srv.RegisterTool(mcpserver.ToolRegistration{
		Name:        "dojo.skill.list",
		Description: "List available skills",
		Handler: func(_ context.Context, _ string, _ map[string]interface{}) (interface{}, error) {
			return []string{"analyze", "summarize"}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterTool: %v", err)
	}

	// Duplicate should fail.
	err = srv.RegisterTool(mcpserver.ToolRegistration{Name: "dojo.skill.list"})
	if err == nil {
		t.Error("Duplicate RegisterTool: expected error")
	}
}

func TestRegisterToolEmptyName(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop(context.Background())

	err := srv.RegisterTool(mcpserver.ToolRegistration{Name: ""})
	if err == nil {
		t.Error("RegisterTool with empty name: expected error")
	}
}

func TestRegisterResource(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop(context.Background())

	err := srv.RegisterResource(mcpserver.ResourceRegistration{
		URI:         "dojo://skill/analyze",
		Name:        "analyze",
		Description: "The analyze skill",
		MimeType:    "application/json",
		Handler: func(_ context.Context, _ string) (interface{}, error) {
			return map[string]string{"name": "analyze"}, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterResource: %v", err)
	}

	// Duplicate should fail.
	err = srv.RegisterResource(mcpserver.ResourceRegistration{URI: "dojo://skill/analyze"})
	if err == nil {
		t.Error("Duplicate RegisterResource: expected error")
	}
}

func TestRegisterResourceEmptyURI(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop(context.Background())

	err := srv.RegisterResource(mcpserver.ResourceRegistration{URI: ""})
	if err == nil {
		t.Error("RegisterResource with empty URI: expected error")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := mcpserver.DefaultConfig()
	if cfg.Transport != "streamable_http" {
		t.Errorf("Transport: got %q, want %q", cfg.Transport, "streamable_http")
	}
	if cfg.Listen != ":9090" {
		t.Errorf("Listen: got %q, want %q", cfg.Listen, ":9090")
	}
	if len(cfg.Tools) != 5 {
		t.Errorf("Tools count: got %d, want 5", len(cfg.Tools))
	}
}

func TestHandleMessageInitialize(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	result, err := srv.HandleMessage(ctx, "initialize", nil)
	if err != nil {
		t.Fatalf("HandleMessage initialize: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("initialize result type: got %T", result)
	}
	if m["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion: got %v", m["protocolVersion"])
	}
}

func TestHandleMessageToolsList(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	srv.RegisterTool(mcpserver.ToolRegistration{
		Name:        "test-tool",
		Description: "A test tool",
	})

	result, err := srv.HandleMessage(ctx, "tools/list", nil)
	if err != nil {
		t.Fatalf("HandleMessage tools/list: %v", err)
	}
	m := result.(map[string]interface{})
	tools := m["tools"].([]map[string]interface{})
	if len(tools) != 1 {
		t.Fatalf("tools count: got %d, want 1", len(tools))
	}
	if tools[0]["name"] != "test-tool" {
		t.Errorf("tool name: got %v", tools[0]["name"])
	}
}

func TestHandleMessageToolsCall(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	srv.RegisterTool(mcpserver.ToolRegistration{
		Name: "echo",
		Handler: func(_ context.Context, _ string, args map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"echoed": args["msg"]}, nil
		},
	})

	params, _ := json.Marshal(map[string]interface{}{
		"name":      "echo",
		"arguments": map[string]interface{}{"msg": "hello"},
	})
	result, err := srv.HandleMessage(ctx, "tools/call", params)
	if err != nil {
		t.Fatalf("HandleMessage tools/call: %v", err)
	}

	// Result text should be JSON (not fmt.Sprintf %v). (#28)
	m := result.(map[string]interface{})
	content := m["content"].([]map[string]interface{})
	text := content[0]["text"].(string)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("tool result is not valid JSON: %v (text was: %q)", err, text)
	}
	if parsed["echoed"] != "hello" {
		t.Errorf("echoed: got %v, want hello", parsed["echoed"])
	}
}

func TestHandleMessageToolsCallError(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	srv.RegisterTool(mcpserver.ToolRegistration{
		Name: "fail-tool",
		Handler: func(_ context.Context, _ string, _ map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("tool failed")
		},
	})

	params, _ := json.Marshal(map[string]interface{}{"name": "fail-tool"})
	result, err := srv.HandleMessage(ctx, "tools/call", params)
	if err != nil {
		t.Fatalf("HandleMessage tools/call (error case): should not return Go error, got: %v", err)
	}

	// Domain errors are returned as isError in the MCP response. (#13)
	m := result.(map[string]interface{})
	if m["isError"] != true {
		t.Error("expected isError=true for tool handler failure")
	}
}

func TestHandleMessageResourcesRead(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	srv.RegisterResource(mcpserver.ResourceRegistration{
		URI:      "dojo://test",
		Name:     "test",
		MimeType: "application/json",
		Handler: func(_ context.Context, _ string) (interface{}, error) {
			return map[string]string{"key": "value"}, nil
		},
	})

	params, _ := json.Marshal(map[string]interface{}{"uri": "dojo://test"})
	result, err := srv.HandleMessage(ctx, "resources/read", params)
	if err != nil {
		t.Fatalf("HandleMessage resources/read: %v", err)
	}

	m := result.(map[string]interface{})
	contents := m["contents"].([]map[string]interface{})
	text := contents[0]["text"].(string)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("resource result is not valid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("key: got %q, want %q", parsed["key"], "value")
	}
}

func TestHandleMessageUnknownMethod(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	_, err := srv.HandleMessage(ctx, "bogus/method", nil)
	if err == nil {
		t.Error("HandleMessage unknown method: expected error")
	}
}
