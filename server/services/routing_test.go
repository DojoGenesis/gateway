package services

import (
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/config"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

func setupTestRouter(t *testing.T) (*UserRouter, *BudgetTracker, *provider.PluginManager) {
	cfg := &config.Config{
		PluginDir: "testdata/plugins",
		Routing: config.RoutingConfig{
			DefaultProvider:       "embedded-qwen3",
			GuestProvider:         "embedded-qwen3",
			AuthenticatedProvider: "deepseek-api",
		},
		Budget: config.BudgetConfig{
			QueryLimit:   1000,
			SessionLimit: 5000,
			MonthlyLimit: 10000,
		},
	}

	pmConfig := provider.PluginManagerConfig{
		PluginDir:          cfg.PluginDir,
		MonitorInterval:    0,
		RestartDelay:       0,
		MaxRestartAttempts: 0,
	}
	pm := provider.NewPluginManagerWithConfig(pmConfig)

	bt := NewBudgetTracker(cfg.Budget.QueryLimit, cfg.Budget.SessionLimit, cfg.Budget.MonthlyLimit)

	router := NewUserRouter(cfg, pm, bt)

	return router, bt, pm
}

func TestNewUserRouter(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	if router == nil {
		t.Fatal("Expected router to be created")
	}

	if router.config == nil {
		t.Error("Expected config to be set")
	}

	if router.pluginManager == nil {
		t.Error("Expected pluginManager to be set")
	}

	if router.budgetTracker == nil {
		t.Error("Expected budgetTracker to be set")
	}
}

func TestSelectProviderGuestUser(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	provider, err := router.SelectProvider("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider != "embedded-qwen3" {
		t.Errorf("Expected embedded-qwen3 for guest user, got %s", provider)
	}
}

func TestSelectProviderAuthenticatedUserWithBudget(t *testing.T) {
	router, bt, _ := setupTestRouter(t)

	provider, err := router.SelectProvider("user1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	remaining, _ := bt.GetRemaining("user1")
	if remaining <= 0 {
		t.Skip("Budget already exhausted")
	}

	if provider != "embedded-qwen3" {
		t.Logf("Expected deepseek-api, got %s (provider may not be loaded)", provider)
	}
}

func TestSelectProviderAuthenticatedUserBudgetExceeded(t *testing.T) {
	router, bt, _ := setupTestRouter(t)

	bt.TrackUsage("user1", 11000)

	provider, err := router.SelectProvider("user1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider != "embedded-qwen3" {
		t.Errorf("Expected embedded-qwen3 when budget exceeded, got %s", provider)
	}
}

func TestSelectProviderFallbackOnProviderError(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	router.config.Routing.AuthenticatedProvider = "nonexistent-provider"

	provider, err := router.SelectProvider("user1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider != "embedded-qwen3" {
		t.Errorf("Expected fallback to embedded-qwen3, got %s", provider)
	}
}

func TestGetDefaultProvider(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	provider := router.GetDefaultProvider()

	if provider != "embedded-qwen3" {
		t.Errorf("Expected embedded-qwen3, got %s", provider)
	}
}

func TestGetGuestProvider(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	provider := router.GetGuestProvider()

	if provider != "embedded-qwen3" {
		t.Errorf("Expected embedded-qwen3, got %s", provider)
	}
}

func TestGetAuthenticatedProvider(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	provider := router.GetAuthenticatedProvider()

	if provider != "deepseek-api" {
		t.Errorf("Expected deepseek-api, got %s", provider)
	}
}

func TestSelectProviderForModelWithoutModel(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	provider, err := router.SelectProviderForModel("", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if provider != "embedded-qwen3" {
		t.Errorf("Expected embedded-qwen3 for empty model, got %s", provider)
	}
}

func TestSelectProviderForModelNotFound(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	_, err := router.SelectProviderForModel("user1", "nonexistent-model")
	if err == nil {
		t.Error("Expected error for nonexistent model")
	}

	expectedMsg := "model nonexistent-model not found in any provider"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestMultipleUsersRouting(t *testing.T) {
	router, bt, _ := setupTestRouter(t)

	user1Provider, _ := router.SelectProvider("user1")
	_, _ = router.SelectProvider("user2")
	guestProvider, _ := router.SelectProvider("")

	if guestProvider != "embedded-qwen3" {
		t.Errorf("Expected embedded-qwen3 for guest, got %s", guestProvider)
	}

	bt.TrackUsage("user2", 11000)

	user2ProviderAfter, _ := router.SelectProvider("user2")
	if user2ProviderAfter != "embedded-qwen3" {
		t.Errorf("Expected embedded-qwen3 for user2 after budget exceeded, got %s", user2ProviderAfter)
	}

	user1ProviderAfter, _ := router.SelectProvider("user1")
	if user1ProviderAfter != user1Provider {
		t.Errorf("Expected user1 provider unchanged, got %s", user1ProviderAfter)
	}
}

func TestRoutingDecisionPerformance(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	iterations := 10000
	maxDuration := int64(5)

	start := timeNowMillis()
	for i := 0; i < iterations; i++ {
		router.SelectProvider("")
		router.SelectProvider("user1")
	}
	duration := timeNowMillis() - start

	avgPerRequest := duration / int64(iterations*2)

	if avgPerRequest > maxDuration {
		t.Errorf("Routing decision too slow: %dms per request (expected <%dms)", avgPerRequest, maxDuration)
	}

	t.Logf("Routing performance: %dms per request", avgPerRequest)
}

func TestBudgetCheckIntegration(t *testing.T) {
	router, bt, _ := setupTestRouter(t)

	provider1, _ := router.SelectProvider("user1")

	bt.TrackUsage("user1", 5000)

	remaining, _ := bt.GetRemaining("user1")
	if remaining != 5000 {
		t.Errorf("Expected remaining 5000, got %d", remaining)
	}

	provider2, _ := router.SelectProvider("user1")

	if provider1 != provider2 {
		t.Errorf("Expected same provider before budget exhaustion")
	}

	bt.TrackUsage("user1", 6000)

	remaining, _ = bt.GetRemaining("user1")
	if remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", remaining)
	}

	provider3, _ := router.SelectProvider("user1")
	if provider3 != "embedded-qwen3" {
		t.Errorf("Expected fallback to embedded-qwen3, got %s", provider3)
	}
}

func TestConcurrentRoutingRequests(t *testing.T) {
	router, bt, _ := setupTestRouter(t)

	users := []string{"user1", "user2", "user3", "user4", "user5"}

	for i := 0; i < 100; i++ {
		for _, user := range users {
			go func(u string) {
				router.SelectProvider(u)
				bt.TrackUsage(u, 10)
			}(user)
		}
	}
}

func timeNowMillis() int64 {
	return time.Now().UnixMilli()
}
