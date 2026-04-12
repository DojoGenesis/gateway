package cas

import "time"

// Ref is a content-addressed reference (SHA-256 hex string).
type Ref string

// ContentType categorizes stored content.
type ContentType string

const (
	ContentSkill          ContentType = "skill"
	ContentToolModule     ContentType = "tool-module"
	ContentMemorySnapshot ContentType = "memory-snapshot"
	ContentConfig         ContentType = "config"
	ContentAgentIdentity  ContentType = "agent-identity"
)

// ContentMeta holds metadata about stored content.
type ContentMeta struct {
	// Type categorizes the content.
	Type ContentType

	// CreatedAt is when the content was stored.
	CreatedAt time.Time

	// CreatedBy identifies who stored the content (agent ID or user).
	CreatedBy string

	// Size is the content size in bytes.
	Size int64

	// Labels are arbitrary key-value pairs for additional metadata.
	Labels map[string]string
}

// TagEntry represents a named reference to content.
type TagEntry struct {
	Name    string
	Version string
	Ref     Ref
	Meta    ContentMeta
}

// GCResult reports the outcome of a garbage collection run.
type GCResult struct {
	// Removed is the number of content entries removed.
	Removed int

	// Freed is the number of bytes freed.
	Freed int64
}

