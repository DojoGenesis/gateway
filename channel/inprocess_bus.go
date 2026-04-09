package channel

import "sync"

// InProcessBus is a synchronous in-process event bus retained for test and
// subpackage smoke-test use ONLY. It is NOT used in any production code path.
//
// Production code uses NATSBus (nats_bus.go) backed by the runtime/event
// NATS + JetStream bus. The cmd/dojo bridge subcommand exclusively uses
// NATSBus. See Era 3 Phase 1 Track A.
//
// Subscribers are called synchronously in the Publish goroutine, which
// guarantees ordering in tests and keeps the flywheel deterministic.
type InProcessBus struct {
	mu          sync.RWMutex
	subscribers []func(string, Event)
}

// Publish calls all registered subscribers with subject and event.
// Implements EventPublisher.
func (b *InProcessBus) Publish(subject string, evt Event) error {
	b.mu.RLock()
	subs := make([]func(string, Event), len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, sub := range subs {
		sub(subject, evt)
	}
	return nil
}

// Subscribe registers fn to be called for every published event.
func (b *InProcessBus) Subscribe(fn func(subject string, evt Event)) {
	b.mu.Lock()
	b.subscribers = append(b.subscribers, fn)
	b.mu.Unlock()
}
