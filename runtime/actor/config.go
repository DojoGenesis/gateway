package actor

import "time"

// Config holds configuration for the actor supervisor.
type Config struct {
	// Enabled controls whether actor supervision is active.
	Enabled bool

	// DefaultMailboxSize is the default channel buffer size for agent mailboxes.
	DefaultMailboxSize int

	// DefaultStrategy is the default supervision strategy for new agents.
	DefaultStrategy SupervisionStrategy
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:            false,
		DefaultMailboxSize: 100,
		DefaultStrategy: SupervisionStrategy{
			MaxRestarts:   3,
			RestartWindow: 60 * time.Second,
			Backoff: BackoffConfig{
				Initial:    100 * time.Millisecond,
				Max:        10 * time.Second,
				Multiplier: 2.0,
			},
			OnExhausted: PolicyLogAndContinue,
		},
	}
}
