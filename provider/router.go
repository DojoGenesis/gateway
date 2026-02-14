package provider

import (
	"context"
	"fmt"
)

// routerContextKey is a custom type for router context keys.
type routerContextKey string

const (
	userIDKey  routerContextKey = "user_id"
	isGuestKey routerContextKey = "is_guest"
)

// ProviderRouter routes requests to the appropriate provider based on context.
type ProviderRouter struct {
	manager       *PluginManager
	routingConfig RoutingConfig
}

// NewProviderRouter creates a ProviderRouter backed by the given manager and config.
func NewProviderRouter(manager *PluginManager, config RoutingConfig) *ProviderRouter {
	return &ProviderRouter{
		manager:       manager,
		routingConfig: config,
	}
}

// GetProvider selects a provider based on context (userID, isGuest).
// Priority: authenticated user mapping > guest provider > default provider > fallback.
func (r *ProviderRouter) GetProvider(ctx context.Context) (ModelProvider, error) {
	// Check for authenticated user mapping
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		if providerName, exists := r.routingConfig.AuthProviders[userID]; exists {
			if p, err := r.manager.GetProvider(providerName); err == nil {
				return p, nil
			}
		}
	}

	// Check for guest context
	if isGuest, ok := ctx.Value(isGuestKey).(bool); ok && isGuest {
		if r.routingConfig.GuestProvider != "" {
			if p, err := r.manager.GetProvider(r.routingConfig.GuestProvider); err == nil {
				return p, nil
			}
		}
	}

	// Default provider
	if r.routingConfig.DefaultProvider != "" {
		if p, err := r.manager.GetProvider(r.routingConfig.DefaultProvider); err == nil {
			return p, nil
		}
	}

	// Fallback provider
	if r.routingConfig.FallbackProvider != "" {
		if p, err := r.manager.GetProvider(r.routingConfig.FallbackProvider); err == nil {
			return p, nil
		}
	}

	return nil, fmt.Errorf("no provider available for the given context")
}

// WithUserID adds a user ID to the context for provider routing.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithGuest marks the context as a guest (unauthenticated) request.
func WithGuest(ctx context.Context) context.Context {
	return context.WithValue(ctx, isGuestKey, true)
}
