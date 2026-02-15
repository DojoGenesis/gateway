package apps

import (
	"fmt"
	"sync"
	"time"
)

// AppInstance represents an active MCP App instance.
type AppInstance struct {
	ID          string                 `json:"id"`
	ResourceURI string                 `json:"resource_uri"`
	SessionID   string                 `json:"session_id"`
	LaunchedAt  time.Time              `json:"launched_at"`
	LastActive  time.Time              `json:"last_active"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AppRegistry manages active app instances.
type AppRegistry struct {
	mu        sync.RWMutex
	instances map[string]*AppInstance
}

// NewAppRegistry creates an empty app registry.
func NewAppRegistry() *AppRegistry {
	return &AppRegistry{
		instances: make(map[string]*AppInstance),
	}
}

// Launch creates a new app instance for the given resource and session.
func (r *AppRegistry) Launch(resourceURI, sessionID string) (*AppInstance, error) {
	if resourceURI == "" {
		return nil, fmt.Errorf("resource URI cannot be empty")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	now := time.Now()
	instance := &AppInstance{
		ID:          fmt.Sprintf("%s-%d", resourceURI, now.UnixNano()),
		ResourceURI: resourceURI,
		SessionID:   sessionID,
		LaunchedAt:  now,
		LastActive:  now,
		Metadata:    make(map[string]interface{}),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.instances[instance.ID] = instance
	return instance, nil
}

// Get retrieves an app instance by ID. Returns an error if not found.
func (r *AppRegistry) Get(instanceID string) (*AppInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, ok := r.instances[instanceID]
	if !ok {
		return nil, fmt.Errorf("app instance not found: %s", instanceID)
	}
	return instance, nil
}

// Close removes an app instance. Returns an error if not found.
func (r *AppRegistry) Close(instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.instances[instanceID]; !exists {
		return fmt.Errorf("app instance not found: %s", instanceID)
	}

	delete(r.instances, instanceID)
	return nil
}

// ListBySession returns all app instances for the given session.
func (r *AppRegistry) ListBySession(sessionID string) []*AppInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*AppInstance
	for _, inst := range r.instances {
		if inst.SessionID == sessionID {
			result = append(result, inst)
		}
	}
	return result
}

// UpdateActivity updates the LastActive timestamp for an instance.
func (r *AppRegistry) UpdateActivity(instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, ok := r.instances[instanceID]
	if !ok {
		return fmt.Errorf("app instance not found: %s", instanceID)
	}

	instance.LastActive = time.Now()
	return nil
}

// Count returns the total number of active app instances.
func (r *AppRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.instances)
}
