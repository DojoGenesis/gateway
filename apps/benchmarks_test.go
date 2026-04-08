package apps

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/tools"
)

func BenchmarkResourceRegistry_Get(b *testing.B) {
	r := NewResourceRegistry()
	r.Register(&ResourceMeta{
		URI:      "ui://bench/app.html",
		MimeType: "text/html",
		Content:  []byte("<html>benchmark</html>"),
		CacheKey: "bench-1",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Get("ui://bench/app.html")
	}
}

func BenchmarkAppRegistry_LaunchClose(b *testing.B) {
	r := NewAppRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inst, _ := r.Launch(fmt.Sprintf("ui://bench/%d.html", i), "session-1")
		_ = r.Close(inst.ID)
	}
}

func BenchmarkToolCallProxy_ProxyCall(b *testing.B) {
	reg := newMockToolRegistry()
	reg.Register(context.Background(), &tools.ToolDefinition{
		Name: "bench-tool",
		Function: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"ok": true}, nil
		},
	})

	policy := NewAuthorizationPolicy()
	policy.GrantAllToolAccess("bench-app")

	proxy := NewToolCallProxy(reg, policy)
	proxy.SetDefaultTimeout(5 * time.Second)

	req := &ToolCallRequest{
		AppID:    "bench-app",
		ToolName: "bench-tool",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proxy.ProxyCall(context.Background(), req)
	}
}

func BenchmarkAuthorizationPolicy_CanCallTool(b *testing.B) {
	p := NewAuthorizationPolicy()
	p.GrantToolAccess("app-1", "tool-a")
	p.GrantToolAccess("app-1", "tool-b")
	p.GrantToolAccess("app-1", "tool-c")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.CanCallTool("app-1", "tool-b")
	}
}

func BenchmarkSecurityPolicy_BuildCSPHeader(b *testing.B) {
	p := NewSecurityPolicy()
	p.AddAllowedOrigin("https://api.example.com")

	meta := &ResourceMeta{
		URI: "ui://bench/app.html",
		CSP: []string{"https://api.example.com"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.BuildCSPHeader(meta)
	}
}
