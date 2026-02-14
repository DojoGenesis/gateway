package skill

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBudgetTracker(t *testing.T) {
	tracker := NewBudgetTracker(10000)
	assert.NotNil(t, tracker)
	assert.Equal(t, 10000, tracker.TotalTokens)
	assert.Equal(t, 0, tracker.ConsumedTokens)
	assert.Equal(t, 0, tracker.ReservedTokens)
}

func TestBudgetTracker_Remaining(t *testing.T) {
	tracker := NewBudgetTracker(10000)
	assert.Equal(t, 10000, tracker.Remaining())

	// Reserve some tokens
	err := tracker.Reserve(3000)
	require.NoError(t, err)
	assert.Equal(t, 7000, tracker.Remaining())

	// Consume with exact match
	tracker.Consume(3000, 3000)
	assert.Equal(t, 7000, tracker.Remaining())
}

func TestBudgetTracker_Reserve_Success(t *testing.T) {
	tracker := NewBudgetTracker(10000)

	err := tracker.Reserve(5000)
	assert.NoError(t, err)
	assert.Equal(t, 5000, tracker.Remaining())
	assert.Equal(t, 5000, tracker.ReservedTokens)
}

func TestBudgetTracker_Reserve_Exhausted(t *testing.T) {
	tracker := NewBudgetTracker(10000)

	// Reserve most of the budget
	err := tracker.Reserve(9000)
	require.NoError(t, err)

	// Try to reserve more than remaining
	err = tracker.Reserve(2000)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrBudgetExhausted)
	assert.Contains(t, err.Error(), "requested 2000 tokens, only 1000 remaining")
}

func TestBudgetTracker_Consume_ExactMatch(t *testing.T) {
	tracker := NewBudgetTracker(10000)

	// Reserve 3000
	err := tracker.Reserve(3000)
	require.NoError(t, err)

	// Consume exact amount
	tracker.Consume(3000, 3000)

	assert.Equal(t, 0, tracker.ReservedTokens)
	assert.Equal(t, 3000, tracker.ConsumedTokens)
	assert.Equal(t, 7000, tracker.Remaining())
}

func TestBudgetTracker_Consume_LessThanReserved(t *testing.T) {
	tracker := NewBudgetTracker(10000)

	// Reserve 3000
	err := tracker.Reserve(3000)
	require.NoError(t, err)

	// Consume less (actual was 2000)
	tracker.Consume(3000, 2000)

	assert.Equal(t, 0, tracker.ReservedTokens)
	assert.Equal(t, 2000, tracker.ConsumedTokens)
	assert.Equal(t, 8000, tracker.Remaining())
}

func TestBudgetTracker_Release(t *testing.T) {
	tracker := NewBudgetTracker(10000)

	// Reserve 3000
	err := tracker.Reserve(3000)
	require.NoError(t, err)
	assert.Equal(t, 7000, tracker.Remaining())

	// Release the reservation (e.g., due to error)
	tracker.Release(3000)

	assert.Equal(t, 0, tracker.ReservedTokens)
	assert.Equal(t, 0, tracker.ConsumedTokens)
	assert.Equal(t, 10000, tracker.Remaining())
}

func TestBudgetTracker_MultipleReservations(t *testing.T) {
	tracker := NewBudgetTracker(10000)

	// Reserve multiple times
	err := tracker.Reserve(2000)
	require.NoError(t, err)
	err = tracker.Reserve(3000)
	require.NoError(t, err)
	err = tracker.Reserve(4000)
	require.NoError(t, err)

	assert.Equal(t, 9000, tracker.ReservedTokens)
	assert.Equal(t, 1000, tracker.Remaining())

	// Attempt to exceed budget
	err = tracker.Reserve(2000)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrBudgetExhausted)
}

func TestBudgetTracker_ThreadSafety(t *testing.T) {
	tracker := NewBudgetTracker(100000)
	var wg sync.WaitGroup

	// Launch 100 goroutines that each reserve, consume, and release
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Reserve
			err := tracker.Reserve(500)
			if err != nil {
				return // Budget exhausted, skip
			}

			// Consume
			tracker.Consume(500, 400)
		}()
	}

	wg.Wait()

	// Check that consumed <= total
	assert.LessOrEqual(t, tracker.ConsumedTokens, tracker.TotalTokens)
	assert.Equal(t, 0, tracker.ReservedTokens)
}

func TestWithBudgetTracker(t *testing.T) {
	ctx := context.Background()
	tracker := NewBudgetTracker(10000)

	ctx = WithBudgetTracker(ctx, tracker)

	retrieved := GetBudgetTracker(ctx)
	assert.NotNil(t, retrieved)
	assert.Equal(t, tracker, retrieved)
}

func TestGetBudgetTracker_NotSet(t *testing.T) {
	ctx := context.Background()

	retrieved := GetBudgetTracker(ctx)
	assert.Nil(t, retrieved)
}
