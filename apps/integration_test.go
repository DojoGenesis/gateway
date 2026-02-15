package apps

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

func newIntegrationAppManager() *AppManager {
	reg := newMockToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name:        "test-tool",
		Description: "A test tool for integration testing",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"status": "ok", "input": args}, nil
		},
	})

	return NewAppManager(AppManagerConfig{
		AllowedOrigins:     []string{"http://localhost:3000"},
		DefaultToolTimeout: 5 * time.Second,
	}, reg)
}

func TestE2E_FullAppLifecycle(t *testing.T) {
	m := newIntegrationAppManager()

	// 1. Register resource
	meta := &ResourceMeta{
		URI:      "ui://e2e/app.html",
		MimeType: "text/html",
		Content:  []byte("<html>e2e test</html>"),
		CacheKey: "e2e-1",
	}
	if err := m.RegisterResource(meta); err != nil {
		t.Fatalf("RegisterResource failed: %v", err)
	}

	// 2. Verify resource is retrievable
	got, err := m.GetResource("ui://e2e/app.html")
	if err != nil {
		t.Fatalf("GetResource failed: %v", err)
	}
	if string(got.Content) != "<html>e2e test</html>" {
		t.Fatalf("content mismatch")
	}

	// 3. Launch app
	inst, err := m.LaunchApp("ui://e2e/app.html", "e2e-session")
	if err != nil {
		t.Fatalf("LaunchApp failed: %v", err)
	}

	// 4. Verify app appears in list
	list := m.ListApps("e2e-session")
	if len(list) != 1 {
		t.Fatalf("expected 1 app, got %d", len(list))
	}

	// 5. Call tool through app
	resp, err := m.ProxyToolCall(context.Background(), &ToolCallRequest{
		AppID:     inst.ID,
		ToolName:  "test-tool",
		Arguments: map[string]interface{}{"key": "value"},
	})
	if err != nil {
		t.Fatalf("ProxyToolCall failed: %v", err)
	}
	if resp.IsError {
		t.Fatalf("tool call returned error: %s", resp.Error)
	}
	if resp.Result["status"] != "ok" {
		t.Fatalf("unexpected result: %v", resp.Result)
	}

	// 6. Close app
	if err := m.CloseApp(inst.ID); err != nil {
		t.Fatalf("CloseApp failed: %v", err)
	}

	// 7. Verify cleanup — tool call should fail now (authorization revoked)
	resp, err = m.ProxyToolCall(context.Background(), &ToolCallRequest{
		AppID:    inst.ID,
		ToolName: "test-tool",
	})
	if err != nil {
		t.Fatalf("ProxyToolCall returned Go error: %v", err)
	}
	if !resp.IsError {
		t.Fatal("expected error after app closure")
	}

	// 8. Verify app no longer in list
	list = m.ListApps("e2e-session")
	if len(list) != 0 {
		t.Fatalf("expected 0 apps after close, got %d", len(list))
	}

	// 9. Verify status
	status := m.Status()
	if status.ActiveAppCount != 0 {
		t.Errorf("ActiveAppCount = %d, want 0", status.ActiveAppCount)
	}
	if status.ResourceCount != 1 {
		t.Errorf("ResourceCount = %d, want 1", status.ResourceCount)
	}
}

func TestE2E_MultipleAppsPerSession(t *testing.T) {
	m := newIntegrationAppManager()

	// Register 3 resources
	for i := 0; i < 3; i++ {
		m.RegisterResource(&ResourceMeta{
			URI:     fmt.Sprintf("ui://multi/app%d.html", i),
			Content: []byte(fmt.Sprintf("app %d", i)),
		})
	}

	// Launch 3 apps
	var instances []*AppInstance
	for i := 0; i < 3; i++ {
		inst, err := m.LaunchApp(fmt.Sprintf("ui://multi/app%d.html", i), "multi-session")
		if err != nil {
			t.Fatalf("Launch %d failed: %v", i, err)
		}
		instances = append(instances, inst)
	}

	// Verify all 3 in list
	list := m.ListApps("multi-session")
	if len(list) != 3 {
		t.Fatalf("expected 3 apps, got %d", len(list))
	}

	// Close all
	for _, inst := range instances {
		if err := m.CloseApp(inst.ID); err != nil {
			t.Fatalf("Close %s failed: %v", inst.ID, err)
		}
	}

	list = m.ListApps("multi-session")
	if len(list) != 0 {
		t.Fatalf("expected 0 apps after close all, got %d", len(list))
	}
}

func TestE2E_ConcurrentToolCalls(t *testing.T) {
	m := newIntegrationAppManager()

	m.RegisterResource(&ResourceMeta{
		URI:     "ui://concurrent/app.html",
		Content: []byte("concurrent"),
	})

	inst, _ := m.LaunchApp("ui://concurrent/app.html", "concurrent-session")

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := m.ProxyToolCall(context.Background(), &ToolCallRequest{
				AppID:     inst.ID,
				ToolName:  "test-tool",
				Arguments: map[string]interface{}{"iteration": i},
			})
			if err != nil {
				errors <- fmt.Errorf("call %d Go error: %w", i, err)
				return
			}
			if resp.IsError {
				errors <- fmt.Errorf("call %d tool error: %s", i, resp.Error)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestE2E_SecurityHeaders(t *testing.T) {
	m := newIntegrationAppManager()

	m.RegisterResource(&ResourceMeta{
		URI:      "ui://security/app.html",
		MimeType: "text/html",
		Content:  []byte("<html>secure</html>"),
	})

	// Get resource and verify security policy works
	meta, err := m.GetResource("ui://security/app.html")
	if err != nil {
		t.Fatalf("GetResource failed: %v", err)
	}

	sp := m.SecurityPolicy()
	csp := sp.BuildCSPHeader(meta)

	if csp == "" {
		t.Fatal("CSP header should not be empty")
	}

	// Validate permissions
	if err := sp.ValidatePermissions([]string{"clipboard-read"}); err != nil {
		t.Errorf("clipboard-read should be safe: %v", err)
	}
	if err := sp.ValidatePermissions([]string{"camera"}); err == nil {
		t.Error("camera should be dangerous")
	}
}
