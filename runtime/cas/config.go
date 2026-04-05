package cas

import "time"

// Config holds configuration for the content-addressable store.
type Config struct {
	// Enabled controls whether CAS is active.
	Enabled bool

	// DBPath is the path to the SQLite database file.
	DBPath string

	// GC configures garbage collection.
	GC GCConfig
}

// GCConfig configures automatic garbage collection.
type GCConfig struct {
	// Enabled controls whether automatic GC runs.
	Enabled bool

	// Interval is how often GC runs.
	Interval time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled: false,
		DBPath:  "./data/cas.db",
		GC: GCConfig{
			Enabled:  true,
			Interval: 24 * time.Hour,
		},
	}
}
