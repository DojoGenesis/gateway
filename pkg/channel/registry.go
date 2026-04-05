package channel

import (
	"fmt"
	"sync"
)

// AdapterRegistry manages registered channel adapters.
type AdapterRegistry struct {
	adapters map[string]ChannelAdapter
	mu       sync.RWMutex
}

// NewAdapterRegistry creates a new adapter registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]ChannelAdapter),
	}
}

// Register adds a channel adapter to the registry.
func (r *AdapterRegistry) Register(adapter ChannelAdapter) error {
	if adapter == nil {
		return fmt.Errorf("adapter cannot be nil")
	}

	id := adapter.ID()
	if id == "" {
		return fmt.Errorf("adapter ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[id]; exists {
		return fmt.Errorf("adapter %q already registered", id)
	}

	r.adapters[id] = adapter
	return nil
}

// Get retrieves an adapter by platform ID.
func (r *AdapterRegistry) Get(platformID string) (ChannelAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[platformID]
	if !exists {
		return nil, fmt.Errorf("adapter %q not found", platformID)
	}
	return adapter, nil
}

// List returns all registered adapter IDs.
func (r *AdapterRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.adapters))
	for id := range r.adapters {
		ids = append(ids, id)
	}
	return ids
}
