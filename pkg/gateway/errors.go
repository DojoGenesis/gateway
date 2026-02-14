package gateway

import "errors"

// ─── Sentinel Errors ────────────────────────────────────────────────────────
//
// Integrators should check these with errors.Is() to handle specific failures.

var (
	// ErrToolNotFound is returned when a requested tool does not exist in the registry.
	ErrToolNotFound = errors.New("tool not found")

	// ErrToolNotRegistered is returned when attempting to invoke a tool that has
	// not been registered with the gateway (distinct from ErrToolNotFound which
	// covers lookup failures — this covers registration gaps).
	ErrToolNotRegistered = errors.New("tool not registered")

	// ErrAgentNotFound is returned when a requested agent ID does not exist.
	ErrAgentNotFound = errors.New("agent not found")

	// ErrAgentInitFailed is returned when agent initialization fails.
	// Wrap the underlying error using fmt.Errorf("%w: ...", ErrAgentInitFailed) to preserve context.
	ErrAgentInitFailed = errors.New("agent initialization failed")

	// ErrProviderUnavailable is returned when no provider is configured or
	// the requested model's provider cannot be reached.
	ErrProviderUnavailable = errors.New("provider unavailable")

	// ErrMemoryUnavailable is returned when the memory store encounters an error.
	ErrMemoryUnavailable = errors.New("memory store unavailable")

	// ErrExecutionCancelled is returned when an execution is cancelled by request.
	ErrExecutionCancelled = errors.New("execution cancelled")

	// ErrInvalidPlan is returned when an execution plan is malformed or invalid.
	// For example, if the DAG contains cycles or references non-existent tools.
	ErrInvalidPlan = errors.New("invalid execution plan")

	// ErrOrchestrationNotFound is returned when a requested orchestration ID
	// does not exist in the store.
	ErrOrchestrationNotFound = errors.New("orchestration not found")

	// ErrTraceNotFound is returned when a requested trace ID does not exist.
	ErrTraceNotFound = errors.New("trace not found")

	// ErrServiceUnavailable is returned when a required service (orchestration
	// engine, trace logger, memory manager, etc.) has not been initialized.
	ErrServiceUnavailable = errors.New("service unavailable")
)
