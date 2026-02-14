package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestResolveAPIKey_Precedence(t *testing.T) {
	// Setup: env var
	os.Setenv("TEST_PROVIDER_KEY", "env-key")
	defer os.Unsetenv("TEST_PROVIDER_KEY")

	bp := &BaseProvider{
		Name:       "test",
		EnvKeyName: "TEST_PROVIDER_KEY",
	}

	// 1. Env var only
	key := bp.ResolveAPIKey(context.Background())
	if key != "env-key" {
		t.Errorf("expected env-key, got %s", key)
	}

	// 2. Static key overrides env
	bp.APIKey = "static-key"
	key = bp.ResolveAPIKey(context.Background())
	if key != "static-key" {
		t.Errorf("expected static-key, got %s", key)
	}

	// 3. Resolver overrides static
	bp.KeyResolver = func(ctx context.Context) string {
		return "resolver-key"
	}
	key = bp.ResolveAPIKey(context.Background())
	if key != "resolver-key" {
		t.Errorf("expected resolver-key, got %s", key)
	}

	// 4. Empty resolver falls back to static
	bp.KeyResolver = func(ctx context.Context) string {
		return ""
	}
	key = bp.ResolveAPIKey(context.Background())
	if key != "static-key" {
		t.Errorf("expected static-key (fallback), got %s", key)
	}
}

func TestHasAPIKey(t *testing.T) {
	bp := &BaseProvider{
		Name:       "test",
		EnvKeyName: "NONEXISTENT_KEY_XYZ",
	}

	if bp.HasAPIKey(context.Background()) {
		t.Error("expected no API key")
	}

	bp.APIKey = "test"
	if !bp.HasAPIKey(context.Background()) {
		t.Error("expected API key to be present")
	}
}

func TestSetKeyResolver(t *testing.T) {
	bp := &BaseProvider{Name: "test"}

	resolver := func(ctx context.Context) string { return "resolved" }
	bp.SetKeyResolver(resolver)

	if bp.KeyResolver == nil {
		t.Error("expected key resolver to be set")
	}
	if bp.ResolveAPIKey(context.Background()) != "resolved" {
		t.Error("expected resolver to return 'resolved'")
	}
}

func TestDoRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("expected Authorization header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	bp := &BaseProvider{
		Name:    "test",
		BaseURL: server.URL,
		Client:  server.Client(),
	}

	resp, err := bp.DoRequest(context.Background(), "POST", "/test", nil, map[string]string{
		"Authorization": "Bearer test-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDoRequest_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	bp := &BaseProvider{
		Name:    "test",
		BaseURL: server.URL,
		Client:  server.Client(),
	}

	_, err := bp.DoRequest(context.Background(), "POST", "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestStreamSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: chunk1\n\ndata: chunk2\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()

	bp := &BaseProvider{
		Name:    "test",
		BaseURL: server.URL,
		Client:  server.Client(),
	}

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ch := make(chan string, 10)
	done := make(chan struct{})
	go func() {
		bp.StreamSSE(context.Background(), resp, ch)
		close(done)
	}()

	// Wait for StreamSSE to finish, then collect
	<-done

	var chunks []string
	close(ch)
	for data := range ch {
		chunks = append(chunks, data)
	}

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
	if len(chunks) >= 1 && chunks[0] != "chunk1" {
		t.Errorf("expected chunk1, got %s", chunks[0])
	}
	if len(chunks) >= 2 && chunks[1] != "chunk2" {
		t.Errorf("expected chunk2, got %s", chunks[1])
	}
}

func TestEnvOrDefault(t *testing.T) {
	os.Setenv("TEST_ENV_VAR", "custom")
	defer os.Unsetenv("TEST_ENV_VAR")

	if envOrDefault("TEST_ENV_VAR", "default") != "custom" {
		t.Error("expected custom value from env")
	}
	if envOrDefault("NONEXISTENT_VAR", "fallback") != "fallback" {
		t.Error("expected fallback value")
	}
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Timeout != 5*60*1e9 {
		t.Errorf("expected 5 minute timeout, got %v", client.Timeout)
	}
}
