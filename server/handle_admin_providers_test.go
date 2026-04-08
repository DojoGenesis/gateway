package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/config"
	"github.com/DojoGenesis/gateway/server/services"
)

// mockProviderForAdmin implements provider.ModelProvider for testing.
type mockProviderForAdmin struct{}

func (m *mockProviderForAdmin) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:         "test-provider",
		Version:      "1.0.0",
		Description:  "Test provider",
		Capabilities: []string{"completion"},
	}, nil
}

func (m *mockProviderForAdmin) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{ID: "model-1", Name: "Model One", Provider: "test-provider"},
	}, nil
}

func (m *mockProviderForAdmin) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return &provider.CompletionResponse{Content: "test"}, nil
}

func (m *mockProviderForAdmin) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	ch := make(chan *provider.CompletionChunk, 1)
	ch <- &provider.CompletionChunk{Done: true}
	close(ch)
	return ch, nil
}

func (m *mockProviderForAdmin) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{Error: "unsupported"}, nil
}

func (m *mockProviderForAdmin) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, nil
}

func TestHandleAdminProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm := provider.NewPluginManager("test-plugins")
	pm.RegisterProvider("test-provider", &mockProviderForAdmin{})

	s := &Server{
		pluginManager: pm,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/providers", nil)

	s.handleAdminProviders(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Providers []struct {
			Name         string   `json:"name"`
			Version      string   `json:"version"`
			Description  string   `json:"description"`
			Capabilities []string `json:"capabilities"`
			Models       int      `json:"models"`
			Healthy      bool     `json:"healthy"`
		} `json:"providers"`
		Total int `json:"total"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}

	if len(resp.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(resp.Providers))
	}

	p := resp.Providers[0]
	if p.Name != "test-provider" {
		t.Errorf("expected name test-provider, got %s", p.Name)
	}
	if p.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", p.Version)
	}
	if p.Models != 1 {
		t.Errorf("expected 1 model, got %d", p.Models)
	}
	if !p.Healthy {
		t.Error("expected healthy=true")
	}
}

func TestHandleAdminProviders_WithRouting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm := provider.NewPluginManager("test-plugins")
	pm.RegisterProvider("test-provider", &mockProviderForAdmin{})

	cfg := &config.Config{
		Routing: config.RoutingConfig{
			DefaultProvider:       "auto",
			GuestProvider:         "auto",
			AuthenticatedProvider: "auto",
		},
		Budget: config.BudgetConfig{
			QueryLimit:   1000,
			SessionLimit: 5000,
			MonthlyLimit: 10000,
		},
	}
	bt := services.NewBudgetTracker(cfg.Budget.QueryLimit, cfg.Budget.SessionLimit, cfg.Budget.MonthlyLimit)
	ur := services.NewUserRouter(cfg, pm, bt)

	s := &Server{
		pluginManager: pm,
		userRouter:    ur,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/providers", nil)

	s.handleAdminProviders(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Total   int `json:"total"`
		Routing struct {
			Default       string `json:"default"`
			Guest         string `json:"guest"`
			Authenticated string `json:"authenticated"`
		} `json:"routing"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// With "auto" config, routing should show resolved values
	if resp.Routing.Default == "" {
		t.Error("expected routing.default to be non-empty")
	}
	if resp.Routing.Guest == "" {
		t.Error("expected routing.guest to be non-empty")
	}
	if resp.Routing.Authenticated == "" {
		t.Error("expected routing.authenticated to be non-empty")
	}
}

func TestHandleAdminProviders_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm := provider.NewPluginManager("test-plugins")

	s := &Server{
		pluginManager: pm,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/providers", nil)

	s.handleAdminProviders(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Total int `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}
}
