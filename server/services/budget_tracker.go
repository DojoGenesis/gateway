package services

import (
	"sync"
	"time"
)

type BudgetTracker struct {
	mu sync.RWMutex

	queryUsage   map[string]int
	sessionUsage map[string]int
	monthlyUsage map[string]int

	queryLimit   int
	sessionLimit int
	monthlyLimit int

	monthlyResetTime map[string]time.Time
}

func NewBudgetTracker(queryLimit, sessionLimit, monthlyLimit int) *BudgetTracker {
	return &BudgetTracker{
		queryUsage:       make(map[string]int),
		sessionUsage:     make(map[string]int),
		monthlyUsage:     make(map[string]int),
		queryLimit:       queryLimit,
		sessionLimit:     sessionLimit,
		monthlyLimit:     monthlyLimit,
		monthlyResetTime: make(map[string]time.Time),
	}
}

func (bt *BudgetTracker) GetRemaining(userID string) (int, error) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.checkMonthlyReset(userID)

	monthly := bt.monthlyUsage[userID]
	remaining := bt.monthlyLimit - monthly

	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

func (bt *BudgetTracker) TrackUsage(userID string, tokens int) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.checkMonthlyReset(userID)

	bt.queryUsage[userID] = tokens
	bt.sessionUsage[userID] += tokens
	bt.monthlyUsage[userID] += tokens

	return nil
}

func (bt *BudgetTracker) GetQueryUsage(userID string) int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.queryUsage[userID]
}

func (bt *BudgetTracker) GetSessionUsage(userID string) int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.sessionUsage[userID]
}

func (bt *BudgetTracker) GetMonthlyUsage(userID string) int {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.monthlyUsage[userID]
}

func (bt *BudgetTracker) ResetSession(userID string) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.sessionUsage[userID] = 0
	bt.queryUsage[userID] = 0
}

func (bt *BudgetTracker) IsQueryLimitExceeded(userID string, tokens int) bool {
	return tokens > bt.queryLimit
}

func (bt *BudgetTracker) IsSessionLimitExceeded(userID string) bool {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.sessionUsage[userID] >= bt.sessionLimit
}

func (bt *BudgetTracker) IsMonthlyLimitExceeded(userID string) bool {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.checkMonthlyReset(userID)
	return bt.monthlyUsage[userID] >= bt.monthlyLimit
}

func (bt *BudgetTracker) checkMonthlyReset(userID string) {
	resetTime, exists := bt.monthlyResetTime[userID]
	if !exists {
		bt.monthlyResetTime[userID] = time.Now().AddDate(0, 1, 0)
		return
	}

	if time.Now().After(resetTime) {
		bt.monthlyUsage[userID] = 0
		bt.monthlyResetTime[userID] = time.Now().AddDate(0, 1, 0)
	}
}

func (bt *BudgetTracker) GetLimits() (query, session, monthly int) {
	return bt.queryLimit, bt.sessionLimit, bt.monthlyLimit
}
