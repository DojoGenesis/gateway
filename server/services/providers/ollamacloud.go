package providers

import "github.com/DojoGenesis/gateway/provider"

// OllamaCloudProvider routes to Ollama Cloud / Turbo (https://ollama.com) — a
// hosted, API-key-authenticated service exposing large models (gpt-oss:120b,
// deepseek, kimi, qwen, glm, …) via an OpenAI-compatible /v1 endpoint.
//
// It is DISTINCT from the local `ollama` provider (OllamaProvider), which talks
// to an unauthenticated ollama daemon at OLLAMA_HOST/localhost:11434. Kept
// separate so a box can serve BOTH a local ollama and the cloud at once. Its
// live /v1/models list is authoritative; the static list below is a fallback.
type OllamaCloudProvider struct {
	openaiCompatibleProvider
}

func NewOllamaCloudProvider(apiKey string) *OllamaCloudProvider {
	return &OllamaCloudProvider{
		openaiCompatibleProvider: openaiCompatibleProvider{
			BaseProvider: BaseProvider{
				Name:       "ollama-cloud",
				BaseURL:    envOrDefault("OLLAMA_CLOUD_BASE_URL", "https://ollama.com/v1"),
				APIKey:     apiKey,
				Client:     NewHTTPClient(),
				EnvKeyName: "OLLAMA_CLOUD_API_KEY",
			},
			defaultModel: "gpt-oss:120b",
			models: []provider.ModelInfo{
				// Minimal fallback only — the live /v1/models list is authoritative.
				{ID: "gpt-oss:120b", Name: "GPT-OSS 120B (Ollama Cloud)", Provider: "ollama-cloud"},
				{ID: "gpt-oss:20b", Name: "GPT-OSS 20B (Ollama Cloud)", Provider: "ollama-cloud"},
			},
			info: &provider.ProviderInfo{
				Name:         "ollama-cloud",
				Version:      "1.0.0",
				Description:  "Ollama Cloud/Turbo (OpenAI-compatible, in-process)",
				Capabilities: []string{"completion", "streaming", "tool_calling"},
			},
		},
	}
}
