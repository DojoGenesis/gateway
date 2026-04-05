package actor

import "time"

// SupervisionStrategy defines how an agent restarts on failure.
type SupervisionStrategy struct {
	// MaxRestarts is the maximum number of restarts within RestartWindow.
	MaxRestarts int

	// RestartWindow is the time window for counting restarts.
	RestartWindow time.Duration

	// Backoff configures exponential backoff between restarts.
	Backoff BackoffConfig

	// OnExhausted defines behavior when max restarts are exceeded.
	OnExhausted ExhaustionPolicy
}

// BackoffConfig defines exponential backoff parameters.
type BackoffConfig struct {
	Initial    time.Duration
	Max        time.Duration
	Multiplier float64
}

// ExhaustionPolicy defines behavior when max restarts are exceeded.
type ExhaustionPolicy int

const (
	// PolicyStop stops the agent permanently.
	PolicyStop ExhaustionPolicy = iota
	// PolicyEscalate propagates the failure to the parent supervisor.
	PolicyEscalate
	// PolicyLogAndContinue logs the failure and keeps the agent stopped.
	PolicyLogAndContinue
)
