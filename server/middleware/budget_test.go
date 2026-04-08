package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/DojoGenesis/gateway/server/services"
	"github.com/gin-gonic/gin"
)

func TestBudgetMiddleware_GuestUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	r := gin.New()
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Guest users should not have budget headers
	if w.Header().Get("X-Budget-Exceeded") != "" {
		t.Error("Guest user should not have budget exceeded header")
	}
}

func TestBudgetMiddleware_AuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.Set("token_usage", 1000)
		c.Set("model", "deepseek-chat")
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check budget headers
	if w.Header().Get("X-Budget-Remaining") == "" {
		t.Error("Expected X-Budget-Remaining header")
	}
	if w.Header().Get("X-Budget-Query-Limit") == "" {
		t.Error("Expected X-Budget-Query-Limit header")
	}
	if w.Header().Get("X-Budget-Session-Limit") == "" {
		t.Error("Expected X-Budget-Session-Limit header")
	}
	if w.Header().Get("X-Budget-Monthly-Limit") == "" {
		t.Error("Expected X-Budget-Monthly-Limit header")
	}

	// Check usage tracking
	// Note: X-Budget-Remaining shows budget BEFORE this request
	remaining := w.Header().Get("X-Budget-Remaining")
	if remaining == "" {
		t.Fatal("X-Budget-Remaining header not set")
	}

	remainingInt, err := strconv.Atoi(remaining)
	if err != nil {
		t.Fatalf("Failed to parse remaining: %v", err)
	}

	// First request shows full budget (usage tracked after response)
	expected := 2000000
	if remainingInt != expected {
		t.Errorf("Expected remaining %d, got %d", expected, remainingInt)
	}

	// Check that usage was tracked
	sessionUsage := tracker.GetSessionUsage("user1")
	if sessionUsage != 1000 {
		t.Errorf("Expected session usage 1000, got %d", sessionUsage)
	}
}

func TestBudgetMiddleware_MonthlyLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	// Exhaust monthly budget
	tracker.TrackUsage("user1", 2500000)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should still succeed (fallback to free provider)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should have exceeded header
	if w.Header().Get("X-Budget-Exceeded") != "true" {
		t.Error("Expected X-Budget-Exceeded header to be true")
	}
	if w.Header().Get("X-Budget-Type") != "monthly" {
		t.Error("Expected X-Budget-Type to be monthly")
	}
}

func TestBudgetMiddleware_SessionLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	// Approach session limit
	tracker.TrackUsage("user1", 250000)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should have warning header
	if w.Header().Get("X-Budget-Warning") != "true" {
		t.Error("Expected X-Budget-Warning header to be true")
	}
	if w.Header().Get("X-Budget-Type") != "session" {
		t.Error("Expected X-Budget-Type to be session")
	}
}

func TestBudgetMiddleware_WarningThreshold(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	// Use 91% of monthly budget
	tracker.TrackUsage("user1", 1820000)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should have warning header
	if w.Header().Get("X-Budget-Warning") != "true" {
		t.Error("Expected X-Budget-Warning header to be true at 91%")
	}

	// Should have warning message
	message := w.Header().Get("X-Budget-Warning-Message")
	if message == "" {
		t.Error("Expected warning message")
	}

	// Check percent used
	percentStr := w.Header().Get("X-Budget-Percent-Used")
	if percentStr == "" {
		t.Fatal("Expected X-Budget-Percent-Used header")
	}

	percent, err := strconv.ParseFloat(percentStr, 64)
	if err != nil {
		t.Fatalf("Failed to parse percent: %v", err)
	}

	if percent < 90.0 {
		t.Errorf("Expected percent >= 90, got %f", percent)
	}
}

func TestBudgetMiddleware_QueryLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		// Single query exceeds limit
		c.Set("token_usage", 60000)
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should have query exceeded header
	if w.Header().Get("X-Budget-Query-Exceeded") != "true" {
		t.Error("Expected X-Budget-Query-Exceeded header to be true")
	}
}

func TestBudgetMiddleware_CostTracking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.Set("token_usage", 10000)
		c.Set("model", "deepseek-chat")
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should have cost header
	costStr := w.Header().Get("X-Budget-Cost")
	if costStr == "" {
		t.Fatal("Expected X-Budget-Cost header")
	}

	cost, err := strconv.ParseFloat(costStr, 64)
	if err != nil {
		t.Fatalf("Failed to parse cost: %v", err)
	}

	// 10000 tokens * 0.0000003 = 0.003
	expectedCost := 0.003
	if cost < expectedCost-0.0001 || cost > expectedCost+0.0001 {
		t.Errorf("Expected cost around %f, got %f", expectedCost, cost)
	}

	// Should have remaining dollars header
	if w.Header().Get("X-Budget-Remaining-Dollars") == "" {
		t.Error("Expected X-Budget-Remaining-Dollars header")
	}
}

func TestBudgetMiddleware_NoTokenUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		// No token usage set
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should still have budget info headers
	if w.Header().Get("X-Budget-Remaining") == "" {
		t.Error("Expected X-Budget-Remaining header")
	}

	// But no cost header
	if w.Header().Get("X-Budget-Cost") != "" {
		t.Error("Should not have X-Budget-Cost header when no token usage")
	}
}

func TestExtractBudgetInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	// Use 95% of budget (warning threshold)
	tracker.TrackUsage("user1", 1900000)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.Set("token_usage", 1000)
		c.Set("model", "deepseek-chat")
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Note: ExtractBudgetInfo reads from headers after middleware sets them
	// So we need to extract from response headers instead
	c := &gin.Context{Request: &http.Request{Header: http.Header{}}}
	for k, v := range w.Header() {
		for _, val := range v {
			c.Request.Header.Add(k, val)
		}
	}

	info := ExtractBudgetInfo(c)

	if !info.Warning {
		t.Error("Expected warning to be true at 95%")
	}

	if info.RemainingTokens <= 0 {
		t.Error("Expected remaining tokens > 0")
	}

	if info.PercentUsed < 90.0 {
		t.Errorf("Expected percent used >= 90, got %f", info.PercentUsed)
	}

	if info.QueryLimit != 50000 {
		t.Errorf("Expected query limit 50000, got %d", info.QueryLimit)
	}
	if info.SessionLimit != 200000 {
		t.Errorf("Expected session limit 200000, got %d", info.SessionLimit)
	}
	if info.MonthlyLimit != 2000000 {
		t.Errorf("Expected monthly limit 2000000, got %d", info.MonthlyLimit)
	}
}

func TestBudgetMiddleware_RequestDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Should have request duration header
	duration := w.Header().Get("X-Request-Duration")
	if duration == "" {
		t.Error("Expected X-Request-Duration header")
	}
}

func TestBudgetMiddleware_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewBudgetTracker(50000, 200000, 2000000)
	costTracker := services.NewCostTracker()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user1")
		c.Next()
	})
	r.Use(BudgetMiddleware(tracker, costTracker))

	r.GET("/test", func(c *gin.Context) {
		c.Set("token_usage", 1000)
		c.Set("model", "deepseek-chat")
		c.JSON(200, gin.H{"message": "ok"})
	})

	// First request
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w1, req1)

	remaining1Str := w1.Header().Get("X-Budget-Remaining")
	remaining1, _ := strconv.Atoi(remaining1Str)

	// Second request
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w2, req2)

	remaining2Str := w2.Header().Get("X-Budget-Remaining")
	remaining2, _ := strconv.Atoi(remaining2Str)

	// Remaining should decrease by 1000 tokens
	if remaining1-remaining2 != 1000 {
		t.Errorf("Expected remaining to decrease by 1000, got %d -> %d", remaining1, remaining2)
	}

	// Check session usage increased
	// Note: Headers show state BEFORE current request, so second response shows first request's usage
	sessionUsage := w2.Header().Get("X-Budget-Session-Usage")
	sessionUsageInt, _ := strconv.Atoi(sessionUsage)
	if sessionUsageInt != 1000 {
		t.Errorf("Expected session usage 1000 (from first request), got %d", sessionUsageInt)
	}

	// Verify actual tracker state shows cumulative usage
	actualSessionUsage := tracker.GetSessionUsage("user1")
	if actualSessionUsage != 2000 {
		t.Errorf("Expected actual session usage 2000, got %d", actualSessionUsage)
	}
}
