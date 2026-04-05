package wasm

import "time"

// CapabilitySet defines what a WASM module is allowed to do.
// Deny-by-default: nil/zero values mean access is denied.
type CapabilitySet struct {
	// FileSystem grants restricted filesystem access. nil = no FS access.
	FileSystem *FSCapability

	// Network grants restricted network access. nil = no network.
	Network *NetCapability

	// Environment lists allowed environment variable names. Empty = none.
	Environment []string

	// MaxMemory is the maximum memory in bytes (0 = default 16MB).
	MaxMemory uint32

	// MaxDuration is the execution timeout.
	MaxDuration time.Duration
}

// FSCapability defines filesystem access restrictions.
type FSCapability struct {
	ReadPaths  []string
	WritePaths []string
}

// NetCapability defines network access restrictions.
type NetCapability struct {
	AllowedHosts []string
	AllowDNS     bool
}
