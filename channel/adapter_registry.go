package channel

import (
	"context"
	"fmt"
	"sync"
)

// AdapterRegistry manages registered channel adapters with a shared
// CredentialStore for secure credential retrieval. Each adapter calls
// creds.Get(ctx, platform, key) instead of reading os.Getenv directly.
//
// Era 3 Phase 1 Track A: CredentialStore injection replaces direct env reads.
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]ChannelAdapter
	creds    CredentialStore
}

// NewAdapterRegistry creates a registry backed by the given CredentialStore.
// If creds is nil, a default EnvCredentialStore is used.
func NewAdapterRegistry(creds CredentialStore) *AdapterRegistry {
	if creds == nil {
		creds = NewEnvCredentialStore()
	}
	return &AdapterRegistry{
		adapters: make(map[string]ChannelAdapter),
		creds:    creds,
	}
}

// CredentialStore returns the registry's credential store.
func (r *AdapterRegistry) CredentialStore() CredentialStore {
	return r.creds
}

// Register adds a channel adapter to the registry. Returns an error if
// the adapter is nil, has an empty name, or is already registered.
func (r *AdapterRegistry) Register(adapter ChannelAdapter) error {
	if adapter == nil {
		return fmt.Errorf("channel: adapter cannot be nil")
	}

	name := adapter.Name()
	if name == "" {
		return fmt.Errorf("channel: adapter name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("channel: adapter %q already registered", name)
	}

	r.adapters[name] = adapter
	return nil
}

// Get retrieves an adapter by platform name.
func (r *AdapterRegistry) Get(platform string) (ChannelAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[platform]
	if !exists {
		return nil, fmt.Errorf("channel: adapter %q not found", platform)
	}
	return adapter, nil
}

// List returns all registered adapter platform names.
func (r *AdapterRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// GetCredential retrieves a credential for a platform and key through
// the registry's CredentialStore. This is the canonical entry point for
// adapters to fetch credentials.
func (r *AdapterRegistry) GetCredential(ctx context.Context, platform, key string) (string, error) {
	return r.creds.Get(ctx, platform, key)
}
