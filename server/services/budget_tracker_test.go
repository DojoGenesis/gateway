package services

import (
	"sync"
	"testing"
	"time"
)

func TestNewBudgetTracker(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	if bt.queryLimit != 1000 {
		t.Errorf("Expected query limit 1000, got %d", bt.queryLimit)
	}
	if bt.sessionLimit != 5000 {
		t.Errorf("Expected session limit 5000, got %d", bt.sessionLimit)
	}
	if bt.monthlyLimit != 10000 {
		t.Errorf("Expected monthly limit 10000, got %d", bt.monthlyLimit)
	}
}

func TestGetRemainingForNewUser(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	remaining, err := bt.GetRemaining("user1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if remaining != 10000 {
		t.Errorf("Expected remaining 10000, got %d", remaining)
	}
}

func TestTrackUsage(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	err := bt.TrackUsage("user1", 500)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	queryUsage := bt.GetQueryUsage("user1")
	if queryUsage != 500 {
		t.Errorf("Expected query usage 500, got %d", queryUsage)
	}

	sessionUsage := bt.GetSessionUsage("user1")
	if sessionUsage != 500 {
		t.Errorf("Expected session usage 500, got %d", sessionUsage)
	}

	monthlyUsage := bt.GetMonthlyUsage("user1")
	if monthlyUsage != 500 {
		t.Errorf("Expected monthly usage 500, got %d", monthlyUsage)
	}

	remaining, _ := bt.GetRemaining("user1")
	if remaining != 9500 {
		t.Errorf("Expected remaining 9500, got %d", remaining)
	}
}

func TestTrackMultipleUsages(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	bt.TrackUsage("user1", 100)
	bt.TrackUsage("user1", 200)
	bt.TrackUsage("user1", 300)

	sessionUsage := bt.GetSessionUsage("user1")
	if sessionUsage != 600 {
		t.Errorf("Expected session usage 600, got %d", sessionUsage)
	}

	queryUsage := bt.GetQueryUsage("user1")
	if queryUsage != 300 {
		t.Errorf("Expected query usage 300 (last query), got %d", queryUsage)
	}
}

func TestIsQueryLimitExceeded(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	if bt.IsQueryLimitExceeded("user1", 500) {
		t.Error("Expected query limit not exceeded for 500 tokens")
	}

	if !bt.IsQueryLimitExceeded("user1", 1500) {
		t.Error("Expected query limit exceeded for 1500 tokens")
	}

	if bt.IsQueryLimitExceeded("user1", 1000) {
		t.Error("Expected query limit not exceeded for exactly 1000 tokens (at limit)")
	}

	if !bt.IsQueryLimitExceeded("user1", 1001) {
		t.Error("Expected query limit exceeded for 1001 tokens")
	}
}

func TestIsSessionLimitExceeded(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	bt.TrackUsage("user1", 3000)
	if bt.IsSessionLimitExceeded("user1") {
		t.Error("Expected session limit not exceeded after 3000 tokens")
	}

	bt.TrackUsage("user1", 2000)
	if !bt.IsSessionLimitExceeded("user1") {
		t.Error("Expected session limit exceeded after 5000 tokens")
	}
}

func TestIsMonthlyLimitExceeded(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	bt.TrackUsage("user1", 8000)
	if bt.IsMonthlyLimitExceeded("user1") {
		t.Error("Expected monthly limit not exceeded after 8000 tokens")
	}

	bt.TrackUsage("user1", 3000)
	if !bt.IsMonthlyLimitExceeded("user1") {
		t.Error("Expected monthly limit exceeded after 11000 tokens")
	}
}

func TestResetSession(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	bt.TrackUsage("user1", 500)
	bt.TrackUsage("user1", 300)

	sessionUsage := bt.GetSessionUsage("user1")
	if sessionUsage != 800 {
		t.Errorf("Expected session usage 800, got %d", sessionUsage)
	}

	bt.ResetSession("user1")

	sessionUsage = bt.GetSessionUsage("user1")
	if sessionUsage != 0 {
		t.Errorf("Expected session usage 0 after reset, got %d", sessionUsage)
	}

	queryUsage := bt.GetQueryUsage("user1")
	if queryUsage != 0 {
		t.Errorf("Expected query usage 0 after reset, got %d", queryUsage)
	}

	monthlyUsage := bt.GetMonthlyUsage("user1")
	if monthlyUsage != 800 {
		t.Errorf("Expected monthly usage 800 (not reset), got %d", monthlyUsage)
	}
}

func TestMultipleUsers(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	bt.TrackUsage("user1", 500)
	bt.TrackUsage("user2", 700)
	bt.TrackUsage("user3", 300)

	user1Usage := bt.GetSessionUsage("user1")
	user2Usage := bt.GetSessionUsage("user2")
	user3Usage := bt.GetSessionUsage("user3")

	if user1Usage != 500 {
		t.Errorf("Expected user1 usage 500, got %d", user1Usage)
	}
	if user2Usage != 700 {
		t.Errorf("Expected user2 usage 700, got %d", user2Usage)
	}
	if user3Usage != 300 {
		t.Errorf("Expected user3 usage 300, got %d", user3Usage)
	}
}

func TestConcurrentAccess(t *testing.T) {
	bt := NewBudgetTracker(10000, 50000, 100000)

	var wg sync.WaitGroup
	users := []string{"user1", "user2", "user3", "user4", "user5"}

	for _, user := range users {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				bt.TrackUsage(u, 10)
				bt.GetRemaining(u)
			}
		}(user)
	}

	wg.Wait()

	for _, user := range users {
		usage := bt.GetSessionUsage(user)
		if usage != 1000 {
			t.Errorf("Expected usage 1000 for %s, got %d", user, usage)
		}
	}
}

func TestGetLimits(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	query, session, monthly := bt.GetLimits()

	if query != 1000 {
		t.Errorf("Expected query limit 1000, got %d", query)
	}
	if session != 5000 {
		t.Errorf("Expected session limit 5000, got %d", session)
	}
	if monthly != 10000 {
		t.Errorf("Expected monthly limit 10000, got %d", monthly)
	}
}

func TestMonthlyReset(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	bt.TrackUsage("user1", 5000)

	monthlyUsage := bt.GetMonthlyUsage("user1")
	if monthlyUsage != 5000 {
		t.Errorf("Expected monthly usage 5000, got %d", monthlyUsage)
	}

	bt.monthlyResetTime["user1"] = time.Now().Add(-time.Hour)

	remaining, _ := bt.GetRemaining("user1")
	if remaining != 10000 {
		t.Errorf("Expected remaining 10000 after reset, got %d", remaining)
	}

	monthlyUsage = bt.GetMonthlyUsage("user1")
	if monthlyUsage != 0 {
		t.Errorf("Expected monthly usage 0 after reset, got %d", monthlyUsage)
	}
}

func TestGetRemainingNegativeCase(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	bt.TrackUsage("user1", 12000)

	remaining, err := bt.GetRemaining("user1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if remaining != 0 {
		t.Errorf("Expected remaining 0 when over budget, got %d", remaining)
	}
}

func TestEmptyUserID(t *testing.T) {
	bt := NewBudgetTracker(1000, 5000, 10000)

	remaining, err := bt.GetRemaining("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if remaining != 10000 {
		t.Errorf("Expected remaining 10000 for empty user, got %d", remaining)
	}

	bt.TrackUsage("", 500)

	remaining, _ = bt.GetRemaining("")
	if remaining != 9500 {
		t.Errorf("Expected remaining 9500 for empty user, got %d", remaining)
	}
}
