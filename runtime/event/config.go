package event

import "time"

// Config holds configuration for the event bus.
type Config struct {
	// Enabled controls whether the event bus is active.
	Enabled bool

	// WAL configures the write-ahead log for event durability.
	WAL WALConfig
}

// WALConfig configures the write-ahead log.
type WALConfig struct {
	// Enabled controls whether events are persisted to WAL.
	Enabled bool

	// DBPath is the path to the SQLite WAL database.
	DBPath string

	// Retention is how long events are kept in the WAL.
	Retention time.Duration

	// FlushInterval is how often events are flushed to disk.
	FlushInterval time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled: false,
		WAL: WALConfig{
			Enabled:       true,
			DBPath:        "./data/event-wal.db",
			Retention:     7 * 24 * time.Hour,
			FlushInterval: time.Second,
		},
	}
}
