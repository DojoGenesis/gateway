package skill

import (
	"context"
	"fmt"
	"sync"
)

// budgetKey is the context key for budget tracking
const budgetKey contextKey = "skill_budget_tracker"

// BudgetTracker manages token budget for meta-skill chains.
// It tracks total, consumed, and reserved tokens with thread-safe operations.
type BudgetTracker struct {
	TotalTokens    int
	ConsumedTokens int
	ReservedTokens int // Tokens reserved but not yet consumed
	mu             sync.Mutex
}

// NewBudgetTracker creates a new budget tracker with the specified total token budget.
func NewBudgetTracker(totalTokens int) *BudgetTracker {
	return &BudgetTracker{
		TotalTokens:    totalTokens,
		ConsumedTokens: 0,
		ReservedTokens: 0,
	}
}

// Remaining returns the number of tokens available (not consumed or reserved).
func (b *BudgetTracker) Remaining() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.TotalTokens - b.ConsumedTokens - b.ReservedTokens
}

// Reserve pre-allocates the estimated number of tokens.
// Returns ErrBudgetExhausted if insufficient tokens are available.
func (b *BudgetTracker) Reserve(estimate int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	remaining := b.TotalTokens - b.ConsumedTokens - b.ReservedTokens
	if remaining < estimate {
		return fmt.Errorf("%w: requested %d tokens, only %d remaining", ErrBudgetExhausted, estimate, remaining)
	}

	b.ReservedTokens += estimate
	return nil
}

// Consume records the actual token usage.
// The actual usage may be less than the reserved amount.
// Any difference between reserved and actual is automatically released.
func (b *BudgetTracker) Consume(reserved, actual int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Release the reservation
	b.ReservedTokens -= reserved

	// Record actual consumption
	b.ConsumedTokens += actual
}

// Release returns unused reserved tokens to the pool.
// This is useful when a skill invocation is cancelled or fails before consuming tokens.
func (b *BudgetTracker) Release(reserved int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ReservedTokens -= reserved
}

// WithBudgetTracker adds a budget tracker to the context.
func WithBudgetTracker(ctx context.Context, tracker *BudgetTracker) context.Context {
	return context.WithValue(ctx, budgetKey, tracker)
}

// GetBudgetTracker retrieves the budget tracker from the context.
// Returns nil if no tracker is set.
func GetBudgetTracker(ctx context.Context) *BudgetTracker {
	tracker, ok := ctx.Value(budgetKey).(*BudgetTracker)
	if !ok {
		return nil
	}
	return tracker
}
