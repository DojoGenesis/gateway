package wasm

import "time"

// ModuleSource indicates where a WASM module was loaded from.
type ModuleSource int

const (
	// ModuleSourceFile indicates the module was loaded from a file.
	ModuleSourceFile ModuleSource = iota
	// ModuleSourceInline indicates the module was provided as inline bytes.
	ModuleSourceInline
)

// ModuleRef identifies a WASM module.
type ModuleRef struct {
	// Name is the human-readable module name.
	Name string

	// Hash is the SHA-256 of the module bytes (for CAS integration).
	Hash string

	// Source indicates where the module was loaded from.
	Source ModuleSource
}

// ModuleInfo provides metadata about a loaded WASM module.
type ModuleInfo struct {
	Ref          ModuleRef
	Size         int64
	Capabilities CapabilitySet
	LoadedAt     time.Time
}
