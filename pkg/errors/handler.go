package errors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
)

// ErrorAction represents the action to take in response to an error.
type ErrorAction string

const (
	ActionStop     ErrorAction = "stop"     // Stop execution and return error
	ActionContinue ErrorAction = "continue" // Log and continue with remaining tasks
	ActionRetry    ErrorAction = "retry"    // Retry the failed operation
	ActionEscalate ErrorAction = "escalate" // Ask user for guidance
)

// ErrorDecision represents the decision made about how to handle an error.
type ErrorDecision struct {
	Action      ErrorAction // What action to take
	RetriesLeft int         // How many retries remain (for ActionRetry)
	Message     string      // Human-readable message about the decision
}

// Handler decides how to handle errors based on disposition.ErrorHandling.
//
// Per Gateway-ADA Contract §3.4:
//   - fail-fast: stop on first error
//   - log-and-continue: log error, continue with remaining tasks
//   - retry: retry N times (from RetryCount), then fail
//   - escalate: ask user for guidance on error
type Handler struct {
	disp *disposition.DispositionConfig
}

// HandlerOption is a functional option for configuring the Handler.
type HandlerOption func(*Handler)

// WithDisposition sets the disposition configuration.
func WithDisposition(disp *disposition.DispositionConfig) HandlerOption {
	return func(h *Handler) {
		h.disp = disp
	}
}

// NewHandler creates a new error handler.
func NewHandler(opts ...HandlerOption) *Handler {
	h := &Handler{
		disp: disposition.DefaultDisposition(),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// HandleError decides what to do with an error based on disposition.
// attemptCount is the number of times this operation has been attempted (0 = first attempt).
func (h *Handler) HandleError(ctx context.Context, err error, attemptCount int) ErrorDecision {
	if h.disp == nil {
		return ErrorDecision{
			Action:  ActionStop,
			Message: err.Error(),
		}
	}

	strategy := strings.ToLower(h.disp.ErrorHandling.Strategy)

	switch strategy {
	case "fail-fast":
		return ErrorDecision{
			Action:  ActionStop,
			Message: fmt.Sprintf("fail-fast: %v", err),
		}

	case "log-and-continue":
		slog.WarnContext(ctx, "error occurred, continuing", "error", err, "attempt", attemptCount)
		return ErrorDecision{
			Action:  ActionContinue,
			Message: fmt.Sprintf("logged and continuing: %v", err),
		}

	case "retry":
		maxRetries := h.disp.ErrorHandling.RetryCount
		if attemptCount < maxRetries {
			retriesLeft := maxRetries - attemptCount
			return ErrorDecision{
				Action:      ActionRetry,
				RetriesLeft: retriesLeft,
				Message:     fmt.Sprintf("retrying (%d/%d): %v", attemptCount+1, maxRetries, err),
			}
		}
		// Exhausted retries
		return ErrorDecision{
			Action:  ActionStop,
			Message: fmt.Sprintf("exhausted %d retries: %v", maxRetries, err),
		}

	case "escalate":
		return ErrorDecision{
			Action:  ActionEscalate,
			Message: fmt.Sprintf("error needs user guidance: %v", err),
		}

	default:
		// Unknown strategy - default to fail-fast
		slog.WarnContext(ctx, "unknown error handling strategy, defaulting to fail-fast", "strategy", strategy)
		return ErrorDecision{
			Action:  ActionStop,
			Message: fmt.Sprintf("unknown strategy '%s': %v", strategy, err),
		}
	}
}

// ShouldRetry is a convenience method that returns true if the decision is to retry.
func (d ErrorDecision) ShouldRetry() bool {
	return d.Action == ActionRetry
}

// ShouldStop is a convenience method that returns true if the decision is to stop.
func (d ErrorDecision) ShouldStop() bool {
	return d.Action == ActionStop
}

// ShouldContinue is a convenience method that returns true if the decision is to continue.
func (d ErrorDecision) ShouldContinue() bool {
	return d.Action == ActionContinue
}

// ShouldEscalate is a convenience method that returns true if the decision is to escalate.
func (d ErrorDecision) ShouldEscalate() bool {
	return d.Action == ActionEscalate
}
