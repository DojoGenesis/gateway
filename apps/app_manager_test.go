package apps

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

func newTestAppManager() *AppManager {
	reg := newMockToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name:        "test-tool",
		Description: "A test tool",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"result": "ok"}, nil
		},
	})

	return NewAppManager(AppManagerConfig{
		AllowedOrigins:     []string{"http://localhost:3000"},
		DefaultToolTimeout: 5 * time.Second,
	}, reg)
}

func TestAppManager_RegisterAndGetResource(t *testing.T) {
	m := newTestAppManager()

	meta := &ResourceMeta{
		URI:      "ui://test/app.html",
		MimeType: "text/html",
		Content:  []byte("<html>hello</html>"),
		CacheKey: "test-1",
	}

	if err := m.RegisterResource(meta); err != nil {
		t.Fatalf("RegisterResource failed: %v", err)
	}

	got, err := m.GetResource("ui://test/app.html")
	if err != nil {
		t.Fatalf("GetResource failed: %v", err)
	}

	if string(got.Content) != "<html>hello</html>" {
		t.Errorf("content = %q, want %q", got.Content, "<html>hello</html>")
	}
}

func TestAppManager_RegisterResource_DangerousPermissions(t *testing.T) {
	m := newTestAppManager()

	meta := &ResourceMeta{
		URI:         "ui://test/evil.html",
		Content:     []byte("x"),
		Permissions: []string{"camera"},
	}

	if err := m.RegisterResource(meta); err == nil {
		t.Fatal("expected error for dangerous permission")
	}
}

func TestAppManager_LaunchAndCloseApp(t *testing.T) {
	m := newTestAppManager()

	// Register resource first
	m.RegisterResource(&ResourceMeta{
		URI:     "ui://test/app.html",
		Content: []byte("app"),
	})

	// Launch
	inst, err := m.LaunchApp("ui://test/app.html", "session-1")
	if err != nil {
		t.Fatalf("LaunchApp failed: %v", err)
	}
	if inst.ID == "" {
		t.Fatal("instance ID is empty")
	}

	status := m.Status()
	if status.ActiveAppCount != 1 {
		t.Errorf("ActiveAppCount = %d, want 1", status.ActiveAppCount)
	}

	// Close
	if err := m.CloseApp(inst.ID); err != nil {
		t.Fatalf("CloseApp failed: %v", err)
	}

	status = m.Status()
	if status.ActiveAppCount != 0 {
		t.Errorf("ActiveAppCount = %d after close, want 0", status.ActiveAppCount)
	}
}

func TestAppManager_LaunchApp_ResourceNotFound(t *testing.T) {
	m := newTestAppManager()

	_, err := m.LaunchApp("ui://nonexistent/app.html", "session-1")
	if err == nil {
		t.Fatal("expected error launching with non-existent resource")
	}
}

func TestAppManager_ProxyToolCall_Authorized(t *testing.T) {
	m := newTestAppManager()

	m.RegisterResource(&ResourceMeta{
		URI:     "ui://test/app.html",
		Content: []byte("app"),
	})

	inst, _ := m.LaunchApp("ui://test/app.html", "session-1")

	resp, err := m.ProxyToolCall(context.Background(), &ToolCallRequest{
		AppID:    inst.ID,
		ToolName: "test-tool",
	})

	if err != nil {
		t.Fatalf("ProxyToolCall failed: %v", err)
	}
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
	if resp.Result["result"] != "ok" {
		t.Errorf("result = %v, want ok", resp.Result)
	}
}

func TestAppManager_ProxyToolCall_Unauthorized(t *testing.T) {
	m := newTestAppManager()

	// Don't launch an app - just try to call tool directly
	resp, err := m.ProxyToolCall(context.Background(), &ToolCallRequest{
		AppID:    "fake-app-id",
		ToolName: "test-tool",
	})

	if err != nil {
		t.Fatalf("ProxyToolCall returned Go error: %v", err)
	}
	if !resp.IsError {
		t.Fatal("expected error response for unauthorized call")
	}
}

func TestAppManager_Status(t *testing.T) {
	m := newTestAppManager()

	// Initially empty
	status := m.Status()
	if status.ResourceCount != 0 {
		t.Errorf("ResourceCount = %d, want 0", status.ResourceCount)
	}
	if status.ActiveAppCount != 0 {
		t.Errorf("ActiveAppCount = %d, want 0", status.ActiveAppCount)
	}
	if !status.Healthy {
		t.Error("expected healthy")
	}

	// Register resources
	for i := 0; i < 3; i++ {
		m.RegisterResource(&ResourceMeta{
			URI:     fmt.Sprintf("ui://test/app%d.html", i),
			Content: []byte("x"),
		})
	}

	status = m.Status()
	if status.ResourceCount != 3 {
		t.Errorf("ResourceCount = %d, want 3", status.ResourceCount)
	}
}

func TestAppManager_ListApps(t *testing.T) {
	m := newTestAppManager()

	m.RegisterResource(&ResourceMeta{URI: "ui://a/1.html", Content: []byte("a")})
	m.RegisterResource(&ResourceMeta{URI: "ui://a/2.html", Content: []byte("b")})

	m.LaunchApp("ui://a/1.html", "session-1")
	m.LaunchApp("ui://a/2.html", "session-1")

	list := m.ListApps("session-1")
	if len(list) != 2 {
		t.Errorf("ListApps = %d, want 2", len(list))
	}

	list = m.ListApps("session-unknown")
	if len(list) != 0 {
		t.Errorf("ListApps(unknown) = %d, want 0", len(list))
	}
}
