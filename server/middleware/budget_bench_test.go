package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/DojoGenesis/gateway/server/services"
	"github.com/gin-gonic/gin"
)

func BenchmarkBudgetMiddleware(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
	}
}

func BenchmarkBudgetCheck(b *testing.B) {
	tracker := services.NewBudgetTracker(50000, 200000, 2000000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetRemaining("user1")
		tracker.IsMonthlyLimitExceeded("user1")
		tracker.IsSessionLimitExceeded("user1")
	}
}

func BenchmarkCostCalculation(b *testing.B) {
	costTracker := services.NewCostTracker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		costTracker.GetCost("deepseek-chat", 1000)
		costTracker.GetBudgetInfo("deepseek-chat", 3.0, 1000000)
	}
}
