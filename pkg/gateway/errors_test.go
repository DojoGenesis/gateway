package gateway

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSentinelErrors_AreDistinct verifies that all sentinel errors are unique
// and can be reliably matched with errors.Is().
func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []error{
		ErrToolNotFound,
		ErrToolNotRegistered,
		ErrAgentNotFound,
		ErrAgentInitFailed,
		ErrProviderUnavailable,
		ErrMemoryUnavailable,
		ErrExecutionCancelled,
		ErrInvalidPlan,
		ErrOrchestrationNotFound,
		ErrTraceNotFound,
		ErrServiceUnavailable,
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, err := range sentinels {
		msg := err.Error()
		assert.False(t, seen[msg], "duplicate sentinel error message: %q", msg)
		seen[msg] = true
	}

	assert.Len(t, sentinels, 11, "expected 11 sentinel errors")
}

// TestSentinelErrors_IsMatching verifies that wrapped errors can be matched
// with errors.Is() — the primary way integrators check error types.
func TestSentinelErrors_IsMatching(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
	}{
		{"ErrToolNotFound", ErrToolNotFound},
		{"ErrToolNotRegistered", ErrToolNotRegistered},
		{"ErrAgentNotFound", ErrAgentNotFound},
		{"ErrAgentInitFailed", ErrAgentInitFailed},
		{"ErrProviderUnavailable", ErrProviderUnavailable},
		{"ErrMemoryUnavailable", ErrMemoryUnavailable},
		{"ErrExecutionCancelled", ErrExecutionCancelled},
		{"ErrInvalidPlan", ErrInvalidPlan},
		{"ErrOrchestrationNotFound", ErrOrchestrationNotFound},
		{"ErrTraceNotFound", ErrTraceNotFound},
		{"ErrServiceUnavailable", ErrServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Direct match
			assert.True(t, errors.Is(tt.sentinel, tt.sentinel))

			// Wrapped match (how integrators would use these in practice)
			wrapped := fmt.Errorf("additional context: %w", tt.sentinel)
			assert.True(t, errors.Is(wrapped, tt.sentinel),
				"wrapped error should match sentinel via errors.Is()")

			// Non-match against other sentinels
			for _, other := range tests {
				if other.name != tt.name {
					assert.False(t, errors.Is(tt.sentinel, other.sentinel),
						"%s should not match %s", tt.name, other.name)
				}
			}
		})
	}
}

// TestSentinelErrors_Messages verifies that error messages are descriptive
// and actionable for integrators.
func TestSentinelErrors_Messages(t *testing.T) {
	tests := []struct {
		err     error
		wantMsg string
	}{
		{ErrToolNotFound, "tool not found"},
		{ErrToolNotRegistered, "tool not registered"},
		{ErrAgentNotFound, "agent not found"},
		{ErrAgentInitFailed, "agent initialization failed"},
		{ErrProviderUnavailable, "provider unavailable"},
		{ErrMemoryUnavailable, "memory store unavailable"},
		{ErrExecutionCancelled, "execution cancelled"},
		{ErrInvalidPlan, "invalid execution plan"},
		{ErrOrchestrationNotFound, "orchestration not found"},
		{ErrTraceNotFound, "trace not found"},
		{ErrServiceUnavailable, "service unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.wantMsg, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}
