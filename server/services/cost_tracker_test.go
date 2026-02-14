package services

import (
	"math"
	"testing"
)

func TestNewCostTracker(t *testing.T) {
	ct := NewCostTracker()

	if ct == nil {
		t.Fatal("NewCostTracker returned nil")
	}

	// Check default pricing for DeepSeek
	price, exists := ct.GetModelPricing("deepseek-chat")
	if !exists {
		t.Error("Expected deepseek-chat to have default pricing")
	}
	if price != 0.0000003 {
		t.Errorf("Expected deepseek-chat price 0.0000003, got %f", price)
	}

	// Check free models
	if !ct.IsFreeModel("embedded-qwen3") {
		t.Error("Expected embedded-qwen3 to be free")
	}
	if !ct.IsFreeModel("qwen3-8b") {
		t.Error("Expected qwen3-8b to be free")
	}
	if !ct.IsFreeModel("ollama") {
		t.Error("Expected ollama to be free")
	}
}

func TestGetCost(t *testing.T) {
	ct := NewCostTracker()

	tests := []struct {
		name     string
		model    string
		tokens   int
		expected float64
	}{
		{
			name:     "DeepSeek chat 1000 tokens",
			model:    "deepseek-chat",
			tokens:   1000,
			expected: 0.0003,
		},
		{
			name:     "DeepSeek chat 10M tokens ($3)",
			model:    "deepseek-chat",
			tokens:   10000000,
			expected: 3.0,
		},
		{
			name:     "Embedded model (free)",
			model:    "embedded-qwen3",
			tokens:   1000000,
			expected: 0.0,
		},
		{
			name:     "Zero tokens",
			model:    "deepseek-chat",
			tokens:   0,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := ct.GetCost(tt.model, tt.tokens)
			if math.Abs(cost-tt.expected) > 0.000001 {
				t.Errorf("Expected cost %f, got %f", tt.expected, cost)
			}
		})
	}
}

func TestGetCostUnknownModel(t *testing.T) {
	ct := NewCostTracker()

	// Unknown models default to free (0 cost) — local models are the common case
	cost := ct.GetCost("unknown-model", 10000000)
	expected := 0.0
	if math.Abs(cost-expected) > 0.000001 {
		t.Errorf("Expected unknown model to default to free, got %f", cost)
	}
}

func TestGetTokensForBudget(t *testing.T) {
	ct := NewCostTracker()

	tests := []struct {
		name          string
		model         string
		budgetDollars float64
		expectedMin   int // Allow some rounding tolerance
		expectedMax   int
	}{
		{
			name:          "DeepSeek $3 budget",
			model:         "deepseek-chat",
			budgetDollars: 3.0,
			expectedMin:   9999999,
			expectedMax:   10000001,
		},
		{
			name:          "DeepSeek $0.30 budget",
			model:         "deepseek-chat",
			budgetDollars: 0.30,
			expectedMin:   999999,
			expectedMax:   1000001,
		},
		{
			name:          "Free model (unlimited)",
			model:         "embedded-qwen3",
			budgetDollars: 1.0,
			expectedMin:   math.MaxInt32, // Very large number
			expectedMax:   math.MaxInt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := ct.GetTokensForBudget(tt.model, tt.budgetDollars)
			if tokens < tt.expectedMin || tokens > tt.expectedMax {
				t.Errorf("Expected tokens between %d and %d, got %d", tt.expectedMin, tt.expectedMax, tokens)
			}
		})
	}
}

func TestSetModelPricing(t *testing.T) {
	ct := NewCostTracker()

	// Set custom pricing
	ct.SetModelPricing("custom-model", 0.00001)

	price, exists := ct.GetModelPricing("custom-model")
	if !exists {
		t.Fatal("Expected custom-model to exist after setting")
	}

	if price != 0.00001 {
		t.Errorf("Expected price 0.00001, got %f", price)
	}

	// Verify cost calculation uses custom pricing
	cost := ct.GetCost("custom-model", 1000)
	expected := 0.01
	if math.Abs(cost-expected) > 0.000001 {
		t.Errorf("Expected cost %f, got %f", expected, cost)
	}
}

func TestIsFreeModel(t *testing.T) {
	ct := NewCostTracker()

	freeModels := []string{"embedded-qwen3", "qwen3-8b", "ollama", "mock"}
	for _, model := range freeModels {
		if !ct.IsFreeModel(model) {
			t.Errorf("Expected %s to be free", model)
		}
	}

	paidModels := []string{"deepseek-chat", "deepseek-reasoner"}
	for _, model := range paidModels {
		if ct.IsFreeModel(model) {
			t.Errorf("Expected %s to be paid", model)
		}
	}

	// Unknown model should not be considered free
	if ct.IsFreeModel("unknown-model") {
		t.Error("Expected unknown-model to not be free")
	}
}

func TestGetBudgetRemaining(t *testing.T) {
	ct := NewCostTracker()

	tests := []struct {
		name          string
		model         string
		budgetDollars float64
		tokensUsed    int
		expected      float64
	}{
		{
			name:          "Half budget used",
			model:         "deepseek-chat",
			budgetDollars: 3.0,
			tokensUsed:    5000000,
			expected:      1.5,
		},
		{
			name:          "No usage",
			model:         "deepseek-chat",
			budgetDollars: 3.0,
			tokensUsed:    0,
			expected:      3.0,
		},
		{
			name:          "Over budget (should return 0)",
			model:         "deepseek-chat",
			budgetDollars: 3.0,
			tokensUsed:    15000000,
			expected:      0.0,
		},
		{
			name:          "Free model (budget unchanged)",
			model:         "embedded-qwen3",
			budgetDollars: 3.0,
			tokensUsed:    1000000,
			expected:      3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := ct.GetBudgetRemaining(tt.model, tt.budgetDollars, tt.tokensUsed)
			if math.Abs(remaining-tt.expected) > 0.000001 {
				t.Errorf("Expected remaining %f, got %f", tt.expected, remaining)
			}
		})
	}
}

func TestGetBudgetInfo(t *testing.T) {
	ct := NewCostTracker()

	tests := []struct {
		name          string
		model         string
		budgetDollars float64
		tokensUsed    int
		checkFunc     func(*testing.T, *BudgetInfo)
	}{
		{
			name:          "Half budget used",
			model:         "deepseek-chat",
			budgetDollars: 3.0,
			tokensUsed:    5000000,
			checkFunc: func(t *testing.T, info *BudgetInfo) {
				if math.Abs(info.PercentUsed-50.0) > 0.1 {
					t.Errorf("Expected 50%% used, got %f%%", info.PercentUsed)
				}
				if info.IsWarning {
					t.Error("Expected no warning at 50%")
				}
				if info.IsExceeded {
					t.Error("Expected not exceeded at 50%")
				}
			},
		},
		{
			name:          "Warning threshold (95%)",
			model:         "deepseek-chat",
			budgetDollars: 3.0,
			tokensUsed:    9500000,
			checkFunc: func(t *testing.T, info *BudgetInfo) {
				if math.Abs(info.PercentUsed-95.0) > 0.1 {
					t.Errorf("Expected 95%% used, got %f%%", info.PercentUsed)
				}
				if !info.IsWarning {
					t.Error("Expected warning at 95%")
				}
				if info.IsExceeded {
					t.Error("Expected not exceeded at 95%")
				}
			},
		},
		{
			name:          "Budget exceeded",
			model:         "deepseek-chat",
			budgetDollars: 3.0,
			tokensUsed:    15000000,
			checkFunc: func(t *testing.T, info *BudgetInfo) {
				if info.PercentUsed < 100.0 {
					t.Errorf("Expected >100%% used, got %f%%", info.PercentUsed)
				}
				if !info.IsExceeded {
					t.Error("Expected budget exceeded")
				}
				if info.RemainingDollars != 0.0 {
					t.Errorf("Expected $0 remaining, got $%f", info.RemainingDollars)
				}
				if info.RemainingTokens != 0 {
					t.Errorf("Expected 0 tokens remaining, got %d", info.RemainingTokens)
				}
			},
		},
		{
			name:          "Free model",
			model:         "embedded-qwen3",
			budgetDollars: 3.0,
			tokensUsed:    1000000,
			checkFunc: func(t *testing.T, info *BudgetInfo) {
				if info.UsedDollars != 0.0 {
					t.Errorf("Expected $0 used for free model, got $%f", info.UsedDollars)
				}
				if math.Abs(info.RemainingDollars-3.0) > 0.000001 {
					t.Errorf("Expected $3 remaining, got $%f", info.RemainingDollars)
				}
			},
		},
		{
			name:          "No usage",
			model:         "deepseek-chat",
			budgetDollars: 3.0,
			tokensUsed:    0,
			checkFunc: func(t *testing.T, info *BudgetInfo) {
				if info.PercentUsed != 0.0 {
					t.Errorf("Expected 0%% used, got %f%%", info.PercentUsed)
				}
				if math.Abs(info.RemainingDollars-3.0) > 0.000001 {
					t.Errorf("Expected $3 remaining, got $%f", info.RemainingDollars)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ct.GetBudgetInfo(tt.model, tt.budgetDollars, tt.tokensUsed)

			if info == nil {
				t.Fatal("GetBudgetInfo returned nil")
			}

			// Check common fields
			if info.TotalBudgetDollars != tt.budgetDollars {
				t.Errorf("Expected total budget %f, got %f", tt.budgetDollars, info.TotalBudgetDollars)
			}
			if info.UsedTokens != tt.tokensUsed {
				t.Errorf("Expected used tokens %d, got %d", tt.tokensUsed, info.UsedTokens)
			}
			if info.Model != tt.model {
				t.Errorf("Expected model %s, got %s", tt.model, info.Model)
			}
			if info.WarningThreshold != 90.0 {
				t.Errorf("Expected warning threshold 90.0, got %f", info.WarningThreshold)
			}

			// Run custom checks
			tt.checkFunc(t, info)
		})
	}
}

func TestCostTrackerConcurrentAccess(t *testing.T) {
	ct := NewCostTracker()

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			ct.GetCost("deepseek-chat", 1000)
			ct.IsFreeModel("embedded-qwen3")
			ct.GetModelPricing("deepseek-chat")
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			ct.SetModelPricing("test-model", float64(id)*0.0001)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify no race conditions occurred
	_, exists := ct.GetModelPricing("test-model")
	if !exists {
		t.Error("Expected test-model to exist after concurrent writes")
	}
}
