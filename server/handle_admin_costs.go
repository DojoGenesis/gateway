package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handleAdminCosts returns cost and budget aggregation data.
// GET /admin/costs?period=day|week|month&provider=X&model=X
//
// When CostTracker or BudgetTracker is nil the endpoint returns 503.
// The period query parameter selects the aggregation window (defaults to "month").
// Optional provider and model filters narrow the response.
func (s *Server) handleAdminCosts(c *gin.Context) {
	if s.costTracker == nil || s.budgetTracker == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable",
			"Cost tracking is not configured")
		return
	}

	period := c.DefaultQuery("period", "month")
	if period != "day" && period != "week" && period != "month" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request",
			"Period must be one of: day, week, month")
		return
	}

	providerFilter := c.Query("provider")
	modelFilter := c.Query("model")

	// Known models grouped by provider
	providerModels := map[string][]string{
		"deepseek": {"deepseek-chat", "deepseek-reasoner"},
		"local":    {"llama3.2", "ollama"},
		"mock":     {"mock"},
	}

	// Build per-model cost entries with optional filtering
	type costEntry struct {
		Model        string  `json:"model"`
		Provider     string  `json:"provider"`
		CostPerToken float64 `json:"cost_per_token"`
		IsFree       bool    `json:"is_free"`
	}

	costs := make([]costEntry, 0)
	breakdownByProvider := make(map[string]gin.H)
	breakdownByModel := make(map[string]gin.H)

	for prov, models := range providerModels {
		if providerFilter != "" && prov != providerFilter {
			continue
		}

		providerTotal := 0.0
		providerModelCount := 0

		for _, model := range models {
			if modelFilter != "" && model != modelFilter {
				continue
			}

			price, exists := s.costTracker.GetModelPricing(model)
			if !exists {
				continue
			}

			costs = append(costs, costEntry{
				Model:        model,
				Provider:     prov,
				CostPerToken: price,
				IsFree:       price == 0,
			})

			breakdownByModel[model] = gin.H{
				"provider":       prov,
				"cost_per_token": price,
				"is_free":        price == 0,
			}

			providerTotal += price
			providerModelCount++
		}

		if providerModelCount > 0 {
			breakdownByProvider[prov] = gin.H{
				"model_count":        providerModelCount,
				"avg_cost_per_token": providerTotal / float64(providerModelCount),
			}
		}
	}

	// Budget information
	queryLimit, sessionLimit, monthlyLimit := s.budgetTracker.GetLimits()

	// Determine which limit applies to the selected period
	var periodLimit int
	switch period {
	case "day":
		periodLimit = queryLimit
	case "week":
		periodLimit = sessionLimit
	case "month":
		periodLimit = monthlyLimit
	}

	budgetUsed := 0 // aggregate usage not yet tracked per-period at the service level
	budgetRemaining := periodLimit - budgetUsed
	if budgetRemaining < 0 {
		budgetRemaining = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"period": period,
		"costs":  costs,
		"budget": gin.H{
			"used":      budgetUsed,
			"limit":     periodLimit,
			"remaining": budgetRemaining,
			"period":    period,
			"limits": gin.H{
				"query":   queryLimit,
				"session": sessionLimit,
				"monthly": monthlyLimit,
			},
		},
		"breakdown_by_provider": breakdownByProvider,
		"breakdown_by_model":    breakdownByModel,
		"uptime":                time.Since(s.startTime).String(),
	})
}
