package errors

import (
	"context"
	"errors"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
)

var errTest = errors.New("test error")

func TestHandleError_FailFast(t *testing.T) {
	disp := &disposition.DispositionConfig{
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "fail-fast",
			RetryCount: 3,
		},
	}

	handler := NewHandler(WithDisposition(disp))
	ctx := context.Background()

	decision := handler.HandleError(ctx, errTest, 0)

	if decision.Action != ActionStop {
		t.Errorf("fail-fast: expected ActionStop, got %v", decision.Action)
	}
}

func TestHandleError_LogAndContinue(t *testing.T) {
	disp := &disposition.DispositionConfig{
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "log-and-continue",
			RetryCount: 3,
		},
	}

	handler := NewHandler(WithDisposition(disp))
	ctx := context.Background()

	decision := handler.HandleError(ctx, errTest, 0)

	if decision.Action != ActionContinue {
		t.Errorf("log-and-continue: expected ActionContinue, got %v", decision.Action)
	}
}

func TestHandleError_Retry_WithinLimit(t *testing.T) {
	disp := &disposition.DispositionConfig{
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "retry",
			RetryCount: 3,
		},
	}

	handler := NewHandler(WithDisposition(disp))
	ctx := context.Background()

	// First attempt (attemptCount = 0)
	decision := handler.HandleError(ctx, errTest, 0)
	if decision.Action != ActionRetry {
		t.Errorf("retry attempt 0: expected ActionRetry, got %v", decision.Action)
	}
	if decision.RetriesLeft != 3 {
		t.Errorf("retry attempt 0: expected 3 retries left, got %d", decision.RetriesLeft)
	}

	// Second attempt (attemptCount = 1)
	decision = handler.HandleError(ctx, errTest, 1)
	if decision.Action != ActionRetry {
		t.Errorf("retry attempt 1: expected ActionRetry, got %v", decision.Action)
	}
	if decision.RetriesLeft != 2 {
		t.Errorf("retry attempt 1: expected 2 retries left, got %d", decision.RetriesLeft)
	}
}

func TestHandleError_Retry_ExhaustedRetries(t *testing.T) {
	disp := &disposition.DispositionConfig{
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "retry",
			RetryCount: 3,
		},
	}

	handler := NewHandler(WithDisposition(disp))
	ctx := context.Background()

	// Attempt beyond retry limit (attemptCount = 3)
	decision := handler.HandleError(ctx, errTest, 3)

	if decision.Action != ActionStop {
		t.Errorf("exhausted retries: expected ActionStop, got %v", decision.Action)
	}
}

func TestHandleError_Retry_ZeroRetries(t *testing.T) {
	disp := &disposition.DispositionConfig{
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "retry",
			RetryCount: 0,
		},
	}

	handler := NewHandler(WithDisposition(disp))
	ctx := context.Background()

	// First attempt with 0 retry count should stop immediately
	decision := handler.HandleError(ctx, errTest, 0)

	if decision.Action != ActionStop {
		t.Errorf("zero retries: expected ActionStop, got %v", decision.Action)
	}
}

func TestHandleError_Retry_OneRetry(t *testing.T) {
	disp := &disposition.DispositionConfig{
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "retry",
			RetryCount: 1,
		},
	}

	handler := NewHandler(WithDisposition(disp))
	ctx := context.Background()

	// First attempt should retry
	decision := handler.HandleError(ctx, errTest, 0)
	if decision.Action != ActionRetry {
		t.Errorf("one retry - attempt 0: expected ActionRetry, got %v", decision.Action)
	}

	// Second attempt should stop
	decision = handler.HandleError(ctx, errTest, 1)
	if decision.Action != ActionStop {
		t.Errorf("one retry - attempt 1: expected ActionStop, got %v", decision.Action)
	}
}

func TestHandleError_Retry_MaxRetries(t *testing.T) {
	disp := &disposition.DispositionConfig{
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "retry",
			RetryCount: 10, // Maximum per contract
		},
	}

	handler := NewHandler(WithDisposition(disp))
	ctx := context.Background()

	// Attempt 9 should still retry
	decision := handler.HandleError(ctx, errTest, 9)
	if decision.Action != ActionRetry {
		t.Errorf("max retries - attempt 9: expected ActionRetry, got %v", decision.Action)
	}

	// Attempt 10 should stop
	decision = handler.HandleError(ctx, errTest, 10)
	if decision.Action != ActionStop {
		t.Errorf("max retries - attempt 10: expected ActionStop, got %v", decision.Action)
	}
}

func TestHandleError_Escalate(t *testing.T) {
	disp := &disposition.DispositionConfig{
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "escalate",
			RetryCount: 3,
		},
	}

	handler := NewHandler(WithDisposition(disp))
	ctx := context.Background()

	decision := handler.HandleError(ctx, errTest, 0)

	if decision.Action != ActionEscalate {
		t.Errorf("escalate: expected ActionEscalate, got %v", decision.Action)
	}
}

func TestErrorDecision_ConvenienceMethods(t *testing.T) {
	tests := []struct {
		action         ErrorAction
		shouldRetry    bool
		shouldStop     bool
		shouldContinue bool
		shouldEscalate bool
	}{
		{ActionRetry, true, false, false, false},
		{ActionStop, false, true, false, false},
		{ActionContinue, false, false, true, false},
		{ActionEscalate, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			decision := ErrorDecision{Action: tt.action}

			if decision.ShouldRetry() != tt.shouldRetry {
				t.Errorf("ShouldRetry: expected %v, got %v", tt.shouldRetry, decision.ShouldRetry())
			}
			if decision.ShouldStop() != tt.shouldStop {
				t.Errorf("ShouldStop: expected %v, got %v", tt.shouldStop, decision.ShouldStop())
			}
			if decision.ShouldContinue() != tt.shouldContinue {
				t.Errorf("ShouldContinue: expected %v, got %v", tt.shouldContinue, decision.ShouldContinue())
			}
			if decision.ShouldEscalate() != tt.shouldEscalate {
				t.Errorf("ShouldEscalate: expected %v, got %v", tt.shouldEscalate, decision.ShouldEscalate())
			}
		})
	}
}

func TestHandleError_DefaultDisposition(t *testing.T) {
	// Handler without explicit disposition should use DefaultDisposition
	handler := NewHandler()
	ctx := context.Background()

	// DefaultDisposition uses "log-and-continue" strategy
	decision := handler.HandleError(ctx, errTest, 0)

	if decision.Action != ActionContinue {
		t.Errorf("default disposition: expected ActionContinue, got %v", decision.Action)
	}
}
