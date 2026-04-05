package wasm

import "time"

// Config holds configuration for the WASM runtime.
type Config struct {
	// Enabled controls whether the WASM runtime is active.
	Enabled bool

	// ModuleDir is the directory to load WASM modules from.
	ModuleDir string

	// Defaults specifies default capability restrictions for modules.
	Defaults CapabilitySet

	// Modules lists explicitly configured WASM modules.
	Modules []ModuleConfig
}

// ModuleConfig describes a single WASM module to load.
type ModuleConfig struct {
	Name         string
	Source       string
	Capabilities CapabilitySet
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:   false,
		ModuleDir: "./wasm-modules",
		Defaults: CapabilitySet{
			MaxMemory:   16 * 1024 * 1024, // 16MB
			MaxDuration: 30 * time.Second,
		},
	}
}
