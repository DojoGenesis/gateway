package channel

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// InfisicalClient is the minimal interface for interacting with the Infisical
// secrets API. Production wiring uses the github.com/Infisical/go-sdk client.
// Tests inject a mock.
type InfisicalClient interface {
	// GetSecret retrieves a single secret by key from the given project
	// environment and path.
	GetSecret(ctx context.Context, key, environment, secretPath string) (string, error)

	// ListSecrets returns all secret keys under the given project
	// environment and path.
	ListSecrets(ctx context.Context, environment, secretPath string) ([]string, error)
}

// InfisicalConfig holds configuration for the InfisicalCredentialStore.
// Mapped from gateway-config.yaml under channel.credentials.
type InfisicalConfig struct {
	// SiteURL is the Infisical instance URL (default: "https://app.infisical.com").
	SiteURL string

	// ClientID is the machine identity client ID.
	ClientID string

	// ClientSecret is the machine identity client secret.
	ClientSecret string

	// ProjectID is the Infisical project containing channel credentials.
	ProjectID string

	// Environment is the Infisical environment slug (e.g. "prod", "dev").
	Environment string

	// SecretPath is the path prefix in Infisical (e.g. "/channel").
	SecretPath string
}

// InfisicalCredentialStore reads credentials from Infisical using the
// InfisicalClient interface. Secret keys follow the convention
// {PLATFORM}_{KEY} (e.g. "SLACK_TOKEN") stored under SecretPath.
//
// A local cache reduces API calls. Use Set() to add test overrides that
// take precedence over remote lookups.
type InfisicalCredentialStore struct {
	client InfisicalClient
	config InfisicalConfig

	mu    sync.RWMutex
	cache map[string]string
}

// NewInfisicalCredentialStore creates a store backed by the given client.
func NewInfisicalCredentialStore(client InfisicalClient, config InfisicalConfig) *InfisicalCredentialStore {
	if config.Environment == "" {
		config.Environment = "prod"
	}
	if config.SecretPath == "" {
		config.SecretPath = "/channel"
	}
	return &InfisicalCredentialStore{
		client: client,
		config: config,
		cache:  make(map[string]string),
	}
}

// infisicalKey builds the secret key: {PLATFORM}_{KEY}.
func infisicalKey(platform, key string) string {
	return fmt.Sprintf("%s_%s",
		strings.ToUpper(platform),
		strings.ToUpper(key),
	)
}

// Get retrieves a credential. It checks the local cache first, then
// queries the Infisical API.
func (s *InfisicalCredentialStore) Get(ctx context.Context, platform, key string) (string, error) {
	k := infisicalKey(platform, key)

	s.mu.RLock()
	if v, ok := s.cache[k]; ok {
		s.mu.RUnlock()
		return v, nil
	}
	s.mu.RUnlock()

	val, err := s.client.GetSecret(ctx, k, s.config.Environment, s.config.SecretPath)
	if err != nil {
		return "", fmt.Errorf("channel: infisical get %s: %w", k, err)
	}

	// Cache the result.
	s.mu.Lock()
	s.cache[k] = val
	s.mu.Unlock()

	return val, nil
}

// Set stores a credential in the local cache. This does NOT write to
// Infisical — it provides a local override for testing and hot-reload
// scenarios.
func (s *InfisicalCredentialStore) Set(_ context.Context, platform, key, value string) error {
	k := infisicalKey(platform, key)
	s.mu.Lock()
	s.cache[k] = value
	s.mu.Unlock()
	return nil
}

// List returns all known keys for a given platform by querying the
// Infisical API and filtering by the platform prefix.
func (s *InfisicalCredentialStore) List(ctx context.Context, platform string) ([]string, error) {
	prefix := strings.ToUpper(platform) + "_"

	allKeys, err := s.client.ListSecrets(ctx, s.config.Environment, s.config.SecretPath)
	if err != nil {
		return nil, fmt.Errorf("channel: infisical list: %w", err)
	}

	var keys []string
	for _, k := range allKeys {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, strings.TrimPrefix(k, prefix))
		}
	}

	// Also include cached overrides.
	s.mu.RLock()
	for k := range s.cache {
		if strings.HasPrefix(k, prefix) {
			short := strings.TrimPrefix(k, prefix)
			found := false
			for _, existing := range keys {
				if existing == short {
					found = true
					break
				}
			}
			if !found {
				keys = append(keys, short)
			}
		}
	}
	s.mu.RUnlock()

	return keys, nil
}

// InvalidateCache clears the local credential cache, forcing subsequent
// Get calls to re-fetch from the Infisical API. Useful for credential
// rotation without restart.
func (s *InfisicalCredentialStore) InvalidateCache() {
	s.mu.Lock()
	s.cache = make(map[string]string)
	s.mu.Unlock()
}
