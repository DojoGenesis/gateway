package shared

import "errors"

// Standard error types for the Agentic Gateway.
var (
	// ErrToolNotFound is returned when a tool cannot be found in the registry.
	ErrToolNotFound = errors.New("tool not found")

	// ErrToolTimeout is returned when a tool execution exceeds its timeout.
	ErrToolTimeout = errors.New("tool execution timed out")

	// ErrToolExecution is returned when a tool fails during execution.
	ErrToolExecution = errors.New("tool execution failed")

	// ErrProviderNotFound is returned when a provider cannot be found.
	ErrProviderNotFound = errors.New("provider not found")

	// ErrProviderUnavailable is returned when a provider is not reachable.
	ErrProviderUnavailable = errors.New("provider unavailable")

	// ErrTaskFailed is returned when a task fails during orchestration.
	ErrTaskFailed = errors.New("task failed")

	// ErrTaskCancelled is returned when a task is cancelled.
	ErrTaskCancelled = errors.New("task cancelled")

	// ErrCircuitOpen is returned when a circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker open")

	// ErrMaxRetriesExceeded is returned when the maximum number of retries is exceeded.
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")

	// ErrMemoryNotFound is returned when a memory entry cannot be found.
	ErrMemoryNotFound = errors.New("memory not found")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned for authentication failures.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrBudgetExceeded is returned when a budget limit is exceeded.
	ErrBudgetExceeded = errors.New("budget exceeded")
)
