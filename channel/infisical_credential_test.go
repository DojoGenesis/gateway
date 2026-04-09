package channel

import (
	"context"
	"fmt"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock Infisical client for unit tests
// ---------------------------------------------------------------------------

type mockInfisicalClient struct {
	secrets map[string]string // key -> value
}

func newMockInfisicalClient() *mockInfisicalClient {
	return &mockInfisicalClient{
		secrets: make(map[string]string),
	}
}

func (m *mockInfisicalClient) GetSecret(_ context.Context, key, _, _ string) (string, error) {
	v, ok := m.secrets[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found", key)
	}
	return v, nil
}

func (m *mockInfisicalClient) ListSecrets(_ context.Context, _, _ string) ([]string, error) {
	keys := make([]string, 0, len(m.secrets))
	for k := range m.secrets {
		keys = append(keys, k)
	}
	return keys, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestInfisicalCredentialStore_Get(t *testing.T) {
	mock := newMockInfisicalClient()
	mock.secrets["SLACK_TOKEN"] = "xoxb-infisical-token"

	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})
	ctx := context.Background()

	val, err := store.Get(ctx, "slack", "TOKEN")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "xoxb-infisical-token" {
		t.Errorf("got %q, want %q", val, "xoxb-infisical-token")
	}
}

func TestInfisicalCredentialStore_GetCached(t *testing.T) {
	mock := newMockInfisicalClient()
	mock.secrets["SLACK_TOKEN"] = "original"

	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})
	ctx := context.Background()

	// First call populates cache.
	store.Get(ctx, "slack", "TOKEN")

	// Change the underlying secret.
	mock.secrets["SLACK_TOKEN"] = "rotated"

	// Should return cached value.
	val, err := store.Get(ctx, "slack", "TOKEN")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "original" {
		t.Errorf("got %q, want cached %q", val, "original")
	}

	// Invalidate cache, should get new value.
	store.InvalidateCache()
	val, err = store.Get(ctx, "slack", "TOKEN")
	if err != nil {
		t.Fatalf("Get after invalidate: %v", err)
	}
	if val != "rotated" {
		t.Errorf("got %q, want %q", val, "rotated")
	}
}

func TestInfisicalCredentialStore_GetMissing(t *testing.T) {
	mock := newMockInfisicalClient()
	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})

	_, err := store.Get(context.Background(), "slack", "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for missing secret, got nil")
	}
}

func TestInfisicalCredentialStore_Set(t *testing.T) {
	mock := newMockInfisicalClient()
	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})
	ctx := context.Background()

	// Set a local override.
	if err := store.Set(ctx, "discord", "TOKEN", "local-override"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Get should return the override.
	val, err := store.Get(ctx, "discord", "TOKEN")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "local-override" {
		t.Errorf("got %q, want %q", val, "local-override")
	}
}

func TestInfisicalCredentialStore_SetOverridesRemote(t *testing.T) {
	mock := newMockInfisicalClient()
	mock.secrets["SLACK_TOKEN"] = "remote-value"

	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})
	ctx := context.Background()

	// Set a local override for the same key.
	store.Set(ctx, "slack", "TOKEN", "local-value")

	// Local override takes precedence.
	val, err := store.Get(ctx, "slack", "TOKEN")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "local-value" {
		t.Errorf("got %q, want %q", val, "local-value")
	}
}

func TestInfisicalCredentialStore_List(t *testing.T) {
	mock := newMockInfisicalClient()
	mock.secrets["SLACK_TOKEN"] = "tok"
	mock.secrets["SLACK_SIGNING_SECRET"] = "sec"
	mock.secrets["DISCORD_TOKEN"] = "dtok"

	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})
	ctx := context.Background()

	keys, err := store.List(ctx, "slack")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	sort.Strings(keys)
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2: %v", len(keys), keys)
	}
	if keys[0] != "SIGNING_SECRET" {
		t.Errorf("keys[0] = %q, want %q", keys[0], "SIGNING_SECRET")
	}
	if keys[1] != "TOKEN" {
		t.Errorf("keys[1] = %q, want %q", keys[1], "TOKEN")
	}
}

func TestInfisicalCredentialStore_ListIncludesCache(t *testing.T) {
	mock := newMockInfisicalClient()
	mock.secrets["SLACK_TOKEN"] = "tok"

	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})
	ctx := context.Background()

	// Add a cached-only key.
	store.Set(ctx, "slack", "APP_TOKEN", "cached-only")

	keys, err := store.List(ctx, "slack")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	sort.Strings(keys)
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2: %v", len(keys), keys)
	}
	if keys[0] != "APP_TOKEN" {
		t.Errorf("keys[0] = %q, want %q", keys[0], "APP_TOKEN")
	}
}

func TestInfisicalCredentialStore_DefaultConfig(t *testing.T) {
	mock := newMockInfisicalClient()
	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})

	if store.config.Environment != "prod" {
		t.Errorf("default environment = %q, want %q", store.config.Environment, "prod")
	}
	if store.config.SecretPath != "/channel" {
		t.Errorf("default secret path = %q, want %q", store.config.SecretPath, "/channel")
	}
}

func TestInfisicalCredentialStore_InvalidateCache(t *testing.T) {
	mock := newMockInfisicalClient()
	mock.secrets["SLACK_TOKEN"] = "v1"

	store := NewInfisicalCredentialStore(mock, InfisicalConfig{})
	ctx := context.Background()

	// Populate cache.
	store.Get(ctx, "slack", "TOKEN")

	// Verify cache is populated.
	store.mu.RLock()
	cached := len(store.cache)
	store.mu.RUnlock()
	if cached != 1 {
		t.Fatalf("cache size = %d, want 1", cached)
	}

	// Invalidate.
	store.InvalidateCache()

	store.mu.RLock()
	cached = len(store.cache)
	store.mu.RUnlock()
	if cached != 0 {
		t.Fatalf("cache size after invalidate = %d, want 0", cached)
	}
}
