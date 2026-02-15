package apps

import (
	"fmt"
	"sync"
)

// ResourceMeta holds metadata and content for a UI resource.
type ResourceMeta struct {
	URI         string   `json:"uri"`
	MimeType    string   `json:"mime_type"`
	CSP         []string `json:"csp,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Content     []byte   `json:"-"`
	CacheKey    string   `json:"cache_key"`
}

// ResourceRegistry manages UI resources keyed by their URI.
type ResourceRegistry struct {
	mu        sync.RWMutex
	resources map[string]*ResourceMeta
}

// NewResourceRegistry creates an empty resource registry.
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		resources: make(map[string]*ResourceMeta),
	}
}

// Register adds a resource to the registry. Returns an error on duplicate URI.
func (r *ResourceRegistry) Register(meta *ResourceMeta) error {
	if meta == nil {
		return fmt.Errorf("resource meta cannot be nil")
	}
	if meta.URI == "" {
		return fmt.Errorf("resource URI cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[meta.URI]; exists {
		return fmt.Errorf("resource already registered: %s", meta.URI)
	}

	r.resources[meta.URI] = meta
	return nil
}

// Get retrieves a resource by URI. Returns an error if not found.
func (r *ResourceRegistry) Get(uri string) (*ResourceMeta, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, ok := r.resources[uri]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", uri)
	}
	return meta, nil
}

// List returns all registered resource URIs.
func (r *ResourceRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	uris := make([]string, 0, len(r.resources))
	for uri := range r.resources {
		uris = append(uris, uri)
	}
	return uris
}

// Unregister removes a resource by URI. Returns an error if not found.
func (r *ResourceRegistry) Unregister(uri string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[uri]; !exists {
		return fmt.Errorf("resource not found: %s", uri)
	}

	delete(r.resources, uri)
	return nil
}
