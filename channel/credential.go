package channel

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

// CredentialStore is a pluggable interface for secure credential retrieval.
// ADR-018 specifies two implementations: EnvCredentialStore (default) and
// InfisicalCredentialStore (production). Phase 0 ships EnvCredentialStore.
type CredentialStore interface {
	// Get retrieves a credential value for a platform and key.
	Get(ctx context.Context, platform, key string) (string, error)

	// Set stores a credential value for a platform and key.
	Set(ctx context.Context, platform, key, value string) error

	// List returns all known keys for a platform.
	List(ctx context.Context, platform string) ([]string, error)
}

// EnvCredentialStore reads credentials from environment variables following
// the convention DOJO_{PLATFORM}_{KEY} (e.g. DOJO_SLACK_TOKEN). The Set
// method stores values in a local map overlay that shadows env lookups,
// enabling testing without modifying the process environment.
type EnvCredentialStore struct {
	mu       sync.RWMutex
	override map[string]string
}

// NewEnvCredentialStore returns an EnvCredentialStore ready for use.
func NewEnvCredentialStore() *EnvCredentialStore {
	return &EnvCredentialStore{
		override: make(map[string]string),
	}
}

// envKey builds the DOJO_{PLATFORM}_{KEY} environment variable name.
func envKey(platform, key string) string {
	return fmt.Sprintf("DOJO_%s_%s",
		strings.ToUpper(platform),
		strings.ToUpper(key),
	)
}

// Get retrieves a credential. It checks the local override map first, then
// falls back to the process environment.
func (s *EnvCredentialStore) Get(_ context.Context, platform, key string) (string, error) {
	k := envKey(platform, key)

	s.mu.RLock()
	if v, ok := s.override[k]; ok {
		s.mu.RUnlock()
		return v, nil
	}
	s.mu.RUnlock()

	v := os.Getenv(k)
	if v == "" {
		return "", fmt.Errorf("channel: credential %s not found", k)
	}
	return v, nil
}

// Set stores a credential in the local override map. This does not modify
// the process environment — the override shadows the env lookup.
func (s *EnvCredentialStore) Set(_ context.Context, platform, key, value string) error {
	k := envKey(platform, key)

	s.mu.Lock()
	s.override[k] = value
	s.mu.Unlock()
	return nil
}

// List returns all keys for a given platform by scanning both the override
// map and the process environment.
func (s *EnvCredentialStore) List(_ context.Context, platform string) ([]string, error) {
	prefix := fmt.Sprintf("DOJO_%s_", strings.ToUpper(platform))

	seen := make(map[string]struct{})
	var keys []string

	// Scan overrides.
	s.mu.RLock()
	for k := range s.override {
		if strings.HasPrefix(k, prefix) {
			short := strings.TrimPrefix(k, prefix)
			if _, ok := seen[short]; !ok {
				seen[short] = struct{}{}
				keys = append(keys, short)
			}
		}
	}
	s.mu.RUnlock()

	// Scan environment.
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.HasPrefix(parts[0], prefix) {
			short := strings.TrimPrefix(parts[0], prefix)
			if _, ok := seen[short]; !ok {
				seen[short] = struct{}{}
				keys = append(keys, short)
			}
		}
	}

	return keys, nil
}
