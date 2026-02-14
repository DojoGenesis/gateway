package services

import (
	"context"
	"fmt"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/config"
)

type UserRouter struct {
	config        *config.Config
	pluginManager *provider.PluginManager
	budgetTracker *BudgetTracker
}

func NewUserRouter(cfg *config.Config, pm *provider.PluginManager, bt *BudgetTracker) *UserRouter {
	return &UserRouter{
		config:        cfg,
		pluginManager: pm,
		budgetTracker: bt,
	}
}

// cloudProviderPreference lists cloud providers in preference order for "auto" routing.
var cloudProviderPreference = []string{"deepseek-api", "openai", "anthropic", "google", "groq", "mistral", "kimi"}

// localProviderPreference lists local providers in preference order for "auto" routing.
var localProviderPreference = []string{"ollama"}

func (ur *UserRouter) SelectProvider(userID string) (string, error) {
	if userID == "" {
		return ur.resolveProvider(ur.config.Routing.GuestProvider, false), nil
	}

	budget, err := ur.budgetTracker.GetRemaining(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get budget: %w", err)
	}

	if budget > 0 {
		provider := ur.resolveProvider(ur.config.Routing.AuthenticatedProvider, true)

		_, err := ur.pluginManager.GetProvider(provider)
		if err != nil {
			return ur.resolveProvider(ur.config.Routing.GuestProvider, false), nil
		}

		return provider, nil
	}

	return ur.resolveProvider(ur.config.Routing.GuestProvider, false), nil
}

// resolveProvider resolves "auto" to a concrete provider name by scanning loaded plugins.
// If preferCloud is true, cloud providers are tried first; otherwise local providers first.
func (ur *UserRouter) resolveProvider(name string, preferCloud bool) string {
	if name != "auto" {
		return name
	}

	if preferCloud {
		// Cloud first, then local
		if p := ur.firstLoaded(cloudProviderPreference); p != "" {
			return p
		}
		if p := ur.firstLoaded(localProviderPreference); p != "" {
			return p
		}
	} else {
		// Local first, then cloud
		if p := ur.firstLoaded(localProviderPreference); p != "" {
			return p
		}
		if p := ur.firstLoaded(cloudProviderPreference); p != "" {
			return p
		}
	}

	// Nothing loaded — return "auto" and let caller deal with the error
	return name
}

// firstLoaded returns the first provider name from candidates that is currently loaded.
func (ur *UserRouter) firstLoaded(candidates []string) string {
	for _, name := range candidates {
		if _, err := ur.pluginManager.GetProvider(name); err == nil {
			return name
		}
	}
	return ""
}

func (ur *UserRouter) SelectProviderForModel(userID, model string) (string, error) {
	if model == "" {
		return ur.SelectProvider(userID)
	}

	providers := ur.pluginManager.GetProviders()
	for name, provider := range providers {
		models, err := provider.ListModels(context.Background())
		if err != nil {
			continue
		}

		for _, m := range models {
			if m.ID == model || m.Name == model {
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("model %s not found in any provider", model)
}

func (ur *UserRouter) GetDefaultProvider() string {
	return ur.resolveProvider(ur.config.Routing.DefaultProvider, false)
}

func (ur *UserRouter) GetGuestProvider() string {
	return ur.resolveProvider(ur.config.Routing.GuestProvider, false)
}

func (ur *UserRouter) GetAuthenticatedProvider() string {
	return ur.resolveProvider(ur.config.Routing.AuthenticatedProvider, true)
}

// RoutingInfo describes the current routing configuration with resolved values.
type RoutingInfo struct {
	Default       string `json:"default"`
	Guest         string `json:"guest"`
	Authenticated string `json:"authenticated"`
}

// GetRoutingInfo returns the current routing config with auto-resolved provider names.
func (ur *UserRouter) GetRoutingInfo() RoutingInfo {
	format := func(raw, resolved string) string {
		if raw == "auto" {
			return fmt.Sprintf("auto (resolved: %s)", resolved)
		}
		return resolved
	}
	return RoutingInfo{
		Default:       format(ur.config.Routing.DefaultProvider, ur.GetDefaultProvider()),
		Guest:         format(ur.config.Routing.GuestProvider, ur.GetGuestProvider()),
		Authenticated: format(ur.config.Routing.AuthenticatedProvider, ur.GetAuthenticatedProvider()),
	}
}

// ResolveHandler maps a logical handler name (e.g. "llm-fast") to a real provider name
// using the config's handler_mapping, with fallback to user-based provider selection.
func (ur *UserRouter) ResolveHandler(handler, userID string) (string, error) {
	// First check handler_mapping in config
	if ur.config.Routing.HandlerMapping != nil {
		if provider, ok := ur.config.Routing.HandlerMapping[handler]; ok {
			// Validate the mapped provider actually exists and is available
			if _, err := ur.pluginManager.GetProvider(provider); err == nil {
				return provider, nil
			}
			// Provider from mapping not available, fall through to user-based routing
		}
	}

	// Fall back to user-based routing
	return ur.SelectProvider(userID)
}
