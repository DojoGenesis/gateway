package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/config"
	"github.com/DojoGenesis/gateway/server/services/providers"
)

// RegisterProviders discovers and registers all available providers.
// Order of operations:
//  1. DiscoverPlugins() -- load external plugin binaries from PluginDir
//  2. Register in-process cloud providers (if API key available)
//  3. Register in-process local providers (if service reachable)
//
// Returns a summary of what was loaded and what was skipped.
func RegisterProviders(
	ctx context.Context,
	pm *provider.PluginManager,
	cfg *config.Config,
	keyResolver providers.APIKeyResolver,
) []providers.ProviderRegistrationInfo {
	var results []providers.ProviderRegistrationInfo

	// 1. External plugins
	if err := pm.DiscoverPlugins(); err != nil {
		slog.Warn("plugin discovery failed", "error", err)
	}

	// 2. Cloud providers -- register if API key is available
	cloudProviders := []struct {
		name    string
		factory func(string) provider.ModelProvider
		envKey  string
	}{
		{"anthropic", func(k string) provider.ModelProvider { return providers.NewAnthropicProvider(k) }, "ANTHROPIC_API_KEY"},
		{"openai", func(k string) provider.ModelProvider { return providers.NewOpenAIProvider(k) }, "OPENAI_API_KEY"},
		{"google", func(k string) provider.ModelProvider { return providers.NewGoogleProvider(k) }, "GOOGLE_API_KEY"},
		{"groq", func(k string) provider.ModelProvider { return providers.NewGroqProvider(k) }, "GROQ_API_KEY"},
		{"mistral", func(k string) provider.ModelProvider { return providers.NewMistralProvider(k) }, "MISTRAL_API_KEY"},
		{"deepseek-api", func(k string) provider.ModelProvider { return providers.NewDeepSeekProvider(k) }, "DEEPSEEK_API_KEY"},
		{"kimi", func(k string) provider.ModelProvider { return providers.NewKimiProvider(k) }, "KIMI_API_KEY"},
		{"voyage", func(k string) provider.ModelProvider { return providers.NewVoyageProvider(k) }, "VOYAGE_API_KEY"},
	}

	for _, cp := range cloudProviders {
		// Skip if already loaded via plugin discovery
		if pm.IsPluginLoaded(cp.name) {
			results = append(results, providers.ProviderRegistrationInfo{
				Name: cp.name, Available: true, Reason: "loaded via plugin",
			})
			continue
		}

		p := cp.factory("")
		if setter, ok := p.(interface{ SetKeyResolver(providers.APIKeyResolver) }); ok && keyResolver != nil {
			setter.SetKeyResolver(keyResolver)
		}

		// Check if provider has a usable key
		if checker, ok := p.(interface{ HasAPIKey(context.Context) bool }); ok && checker.HasAPIKey(ctx) {
			pm.RegisterProvider(cp.name, p)
			results = append(results, providers.ProviderRegistrationInfo{
				Name: cp.name, Available: true, Provider: p,
			})
			slog.Info("registered in-process provider", "name", cp.name)
		} else {
			results = append(results, providers.ProviderRegistrationInfo{
				Name: cp.name, Available: false, Reason: "no API key (" + cp.envKey + ")",
			})
		}
	}

	// 3. Local providers -- register if service is reachable
	ollamaProvider := providers.NewOllamaProvider()
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if ollamaProvider.IsAvailable(checkCtx) {
		pm.RegisterProvider("ollama", ollamaProvider)
		results = append(results, providers.ProviderRegistrationInfo{
			Name: "ollama", Available: true, Provider: ollamaProvider,
		})
		slog.Info("registered local provider", "name", "ollama")
	} else {
		results = append(results, providers.ProviderRegistrationInfo{
			Name: "ollama", Available: false, Reason: "Ollama server not reachable",
		})
	}

	return results
}
