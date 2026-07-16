package providers

import "github.com/DojoGenesis/gateway/provider"

// OpenRouterProvider routes to OpenRouter (https://openrouter.ai) — an
// OpenAI-compatible aggregator that fronts hundreds of models from many
// vendors. Its /models endpoint is the source of truth for the catalog (far too
// large and fast-moving to hardcode), so the embedded openaiCompatibleProvider's
// dynamic ListModels does the work; the static list below is only a minimal
// fallback used when the live endpoint is unreachable.
//
// Model IDs are vendor-namespaced (e.g. "openai/gpt-4o",
// "anthropic/claude-sonnet-4.5", "google/gemini-2.5-pro").
type OpenRouterProvider struct {
	openaiCompatibleProvider
}

func NewOpenRouterProvider(apiKey string) *OpenRouterProvider {
	return &OpenRouterProvider{
		openaiCompatibleProvider: openaiCompatibleProvider{
			BaseProvider: BaseProvider{
				Name:       "openrouter",
				BaseURL:    envOrDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
				APIKey:     apiKey,
				Client:     NewHTTPClient(),
				EnvKeyName: "OPENROUTER_API_KEY",
			},
			defaultModel: "openai/gpt-4o",
			models: []provider.ModelInfo{
				// Minimal fallback only — the live /models list is authoritative.
				{ID: "openai/gpt-4o", Name: "GPT-4o (OpenRouter)", Provider: "openrouter", ContextSize: 128000},
				{ID: "anthropic/claude-sonnet-4.5", Name: "Claude Sonnet 4.5 (OpenRouter)", Provider: "openrouter", ContextSize: 200000},
			},
			info: &provider.ProviderInfo{
				Name:         "openrouter",
				Version:      "1.0.0",
				Description:  "OpenRouter aggregator (OpenAI-compatible, in-process)",
				Capabilities: []string{"completion", "streaming", "tool_calling"},
			},
		},
	}
}
