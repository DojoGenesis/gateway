package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRouter_DefaultProvider(t *testing.T) {
	pm := NewPluginManager(t.TempDir())
	mock := &mockProvider{
		info: &ProviderInfo{Name: "default-provider", Version: "1.0.0"},
	}
	pm.RegisterProvider("default-provider", mock)

	router := NewProviderRouter(pm, RoutingConfig{
		DefaultProvider: "default-provider",
	})

	p, err := router.GetProvider(context.Background())
	require.NoError(t, err)

	info, err := p.GetInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "default-provider", info.Name)
}

func TestProviderRouter_GuestProvider(t *testing.T) {
	pm := NewPluginManager(t.TempDir())
	pm.RegisterProvider("guest-provider", &mockProvider{
		info: &ProviderInfo{Name: "guest-provider", Version: "1.0.0"},
	})
	pm.RegisterProvider("default-provider", &mockProvider{
		info: &ProviderInfo{Name: "default-provider", Version: "1.0.0"},
	})

	router := NewProviderRouter(pm, RoutingConfig{
		DefaultProvider: "default-provider",
		GuestProvider:   "guest-provider",
	})

	ctx := WithGuest(context.Background())
	p, err := router.GetProvider(ctx)
	require.NoError(t, err)

	info, err := p.GetInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, "guest-provider", info.Name)
}

func TestProviderRouter_AuthenticatedProvider(t *testing.T) {
	pm := NewPluginManager(t.TempDir())
	pm.RegisterProvider("user-provider", &mockProvider{
		info: &ProviderInfo{Name: "user-provider", Version: "1.0.0"},
	})
	pm.RegisterProvider("default-provider", &mockProvider{
		info: &ProviderInfo{Name: "default-provider", Version: "1.0.0"},
	})

	router := NewProviderRouter(pm, RoutingConfig{
		DefaultProvider: "default-provider",
		AuthProviders: map[string]string{
			"user-123": "user-provider",
		},
	})

	ctx := WithUserID(context.Background(), "user-123")
	p, err := router.GetProvider(ctx)
	require.NoError(t, err)

	info, err := p.GetInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, "user-provider", info.Name)
}

func TestProviderRouter_FallbackProvider(t *testing.T) {
	pm := NewPluginManager(t.TempDir())
	pm.RegisterProvider("fallback", &mockProvider{
		info: &ProviderInfo{Name: "fallback", Version: "1.0.0"},
	})

	router := NewProviderRouter(pm, RoutingConfig{
		DefaultProvider:  "non-existent",
		FallbackProvider: "fallback",
	})

	p, err := router.GetProvider(context.Background())
	require.NoError(t, err)

	info, err := p.GetInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "fallback", info.Name)
}

func TestProviderRouter_NoProviderAvailable(t *testing.T) {
	pm := NewPluginManager(t.TempDir())

	router := NewProviderRouter(pm, RoutingConfig{
		DefaultProvider: "non-existent",
	})

	_, err := router.GetProvider(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no provider available")
}

func TestProviderRouter_AuthOverridesGuest(t *testing.T) {
	pm := NewPluginManager(t.TempDir())
	pm.RegisterProvider("auth-provider", &mockProvider{
		info: &ProviderInfo{Name: "auth-provider", Version: "1.0.0"},
	})
	pm.RegisterProvider("guest-provider", &mockProvider{
		info: &ProviderInfo{Name: "guest-provider", Version: "1.0.0"},
	})

	router := NewProviderRouter(pm, RoutingConfig{
		GuestProvider: "guest-provider",
		AuthProviders: map[string]string{
			"user-1": "auth-provider",
		},
	})

	// Even with guest flag set, auth mapping takes priority
	ctx := WithGuest(WithUserID(context.Background(), "user-1"))
	p, err := router.GetProvider(ctx)
	require.NoError(t, err)

	info, err := p.GetInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, "auth-provider", info.Name)
}

func TestProviderRouter_UnknownUserFallsToDefault(t *testing.T) {
	pm := NewPluginManager(t.TempDir())
	pm.RegisterProvider("default-provider", &mockProvider{
		info: &ProviderInfo{Name: "default-provider", Version: "1.0.0"},
	})

	router := NewProviderRouter(pm, RoutingConfig{
		DefaultProvider: "default-provider",
		AuthProviders: map[string]string{
			"user-1": "some-provider",
		},
	})

	ctx := WithUserID(context.Background(), "unknown-user")
	p, err := router.GetProvider(ctx)
	require.NoError(t, err)

	info, err := p.GetInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, "default-provider", info.Name)
}

func TestWithUserID(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-42")
	val, ok := ctx.Value(userIDKey).(string)
	assert.True(t, ok)
	assert.Equal(t, "user-42", val)
}

func TestWithGuest(t *testing.T) {
	ctx := WithGuest(context.Background())
	val, ok := ctx.Value(isGuestKey).(bool)
	assert.True(t, ok)
	assert.True(t, val)
}
