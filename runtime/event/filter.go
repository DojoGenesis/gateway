package event

// EventFilter specifies criteria for matching events.
type EventFilter struct {
	// Types filters by CloudEvent type (e.g., "dojo.tool.completed").
	Types []string

	// Sources filters by CloudEvent source (e.g., "agent/scout-1").
	Sources []string

	// Subjects filters by CloudEvent subject (e.g., "workflow/abc123").
	Subjects []string
}

// Subscription represents an active event subscription.
type Subscription interface {
	// Unsubscribe removes this subscription.
	Unsubscribe() error

	// ID returns the subscription identifier.
	ID() string
}
