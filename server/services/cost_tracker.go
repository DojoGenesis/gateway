package services

import (
	"sync"
)

// CostTracker tracks costs and token usage across different model providers
type CostTracker struct {
	mu sync.RWMutex

	// Model pricing (cost per token in dollars)
	modelPricing map[string]float64
}

// NewCostTracker creates a new cost tracker with default pricing
func NewCostTracker() *CostTracker {
	return &CostTracker{
		modelPricing: map[string]float64{
			// DeepSeek pricing (from requirements: $3/month = ~10M tokens)
			// $3.00 / 10,000,000 tokens = $0.0000003 per token
			"deepseek-chat":     0.0000003,
			"deepseek-reasoner": 0.0000003,

			// Local models are free
			"llama3.2": 0.0,

			// Ollama models are free (local)
			"ollama": 0.0,

			// Mock provider for testing
			"mock": 0.0,
		},
	}
}

// GetCost calculates the cost in dollars for a given number of tokens and model
func (ct *CostTracker) GetCost(model string, tokens int) float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	pricePerToken, exists := ct.modelPricing[model]
	if !exists {
		// Unknown models default to free (local models are more common than cloud)
		pricePerToken = 0.0
	}

	return float64(tokens) * pricePerToken
}

// GetTokensForBudget calculates how many tokens can be used for a given budget (in dollars)
func (ct *CostTracker) GetTokensForBudget(model string, budgetDollars float64) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	pricePerToken, exists := ct.modelPricing[model]
	if !exists || pricePerToken == 0.0 {
		// Free model or unknown - unlimited tokens
		return int(^uint(0) >> 1) // Max int
	}

	return int(budgetDollars / pricePerToken)
}

// SetModelPricing sets the pricing for a specific model (cost per token in dollars)
func (ct *CostTracker) SetModelPricing(model string, costPerToken float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.modelPricing[model] = costPerToken
}

// GetModelPricing gets the pricing for a specific model
func (ct *CostTracker) GetModelPricing(model string) (float64, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	price, exists := ct.modelPricing[model]
	return price, exists
}

// IsFreeModel checks if a model is free (zero cost)
func (ct *CostTracker) IsFreeModel(model string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	pricePerToken, exists := ct.modelPricing[model]
	if !exists {
		return false
	}

	return pricePerToken == 0.0
}

// GetBudgetRemaining calculates the remaining budget in dollars based on tokens used
func (ct *CostTracker) GetBudgetRemaining(model string, budgetDollars float64, tokensUsed int) float64 {
	costUsed := ct.GetCost(model, tokensUsed)
	remaining := budgetDollars - costUsed

	if remaining < 0 {
		return 0.0
	}

	return remaining
}

// BudgetInfo contains detailed budget information
type BudgetInfo struct {
	TotalBudgetDollars float64
	UsedDollars        float64
	RemainingDollars   float64
	TotalTokens        int
	UsedTokens         int
	RemainingTokens    int
	PercentUsed        float64
	WarningThreshold   float64 // 90% by default
	IsWarning          bool
	IsExceeded         bool
	Model              string
}

// GetBudgetInfo provides comprehensive budget information
func (ct *CostTracker) GetBudgetInfo(model string, budgetDollars float64, tokensUsed int) *BudgetInfo {
	costUsed := ct.GetCost(model, tokensUsed)
	remainingDollars := budgetDollars - costUsed
	if remainingDollars < 0 {
		remainingDollars = 0.0
	}

	totalTokens := ct.GetTokensForBudget(model, budgetDollars)
	remainingTokens := totalTokens - tokensUsed
	if remainingTokens < 0 {
		remainingTokens = 0
	}

	percentUsed := 0.0
	if budgetDollars > 0 {
		percentUsed = (costUsed / budgetDollars) * 100.0
	}

	warningThreshold := 90.0
	isWarning := percentUsed >= warningThreshold && percentUsed < 100.0
	isExceeded := percentUsed >= 100.0

	return &BudgetInfo{
		TotalBudgetDollars: budgetDollars,
		UsedDollars:        costUsed,
		RemainingDollars:   remainingDollars,
		TotalTokens:        totalTokens,
		UsedTokens:         tokensUsed,
		RemainingTokens:    remainingTokens,
		PercentUsed:        percentUsed,
		WarningThreshold:   warningThreshold,
		IsWarning:          isWarning,
		IsExceeded:         isExceeded,
		Model:              model,
	}
}
