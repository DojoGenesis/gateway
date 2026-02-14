package gateway

import "errors"

var (
	// ErrToolNotFound is returned when a requested tool does not exist in the registry.
	ErrToolNotFound = errors.New("tool not found")

	// ErrAgentInitFailed is returned when agent initialization fails.
	// Wrap the underlying error using fmt.Errorf("%w", err) to preserve context.
	ErrAgentInitFailed = errors.New("agent initialization failed")

	// ErrMemoryUnavailable is returned when the memory store encounters an error.
	ErrMemoryUnavailable = errors.New("memory store unavailable")

	// ErrExecutionCancelled is returned when an execution is cancelled by request.
	ErrExecutionCancelled = errors.New("execution cancelled")

	// ErrInvalidPlan is returned when an execution plan is malformed or invalid.
	// For example, if the DAG contains cycles or references non-existent tools.
	ErrInvalidPlan = errors.New("invalid execution plan")
)
