package providers

import (
	"github.com/DojoGenesis/gateway/provider"
)

type MistralProvider struct {
	openaiCompatibleProvider
}

func NewMistralProvider(apiKey string) *MistralProvider {
	return &MistralProvider{
		openaiCompatibleProvider: openaiCompatibleProvider{
			BaseProvider: BaseProvider{
				Name:       "mistral",
				BaseURL:    "https://api.mistral.ai/v1",
				APIKey:     apiKey,
				Client:     NewHTTPClient(),
				EnvKeyName: "MISTRAL_API_KEY",
			},
			defaultModel: "mistral-large-latest",
			models: []provider.ModelInfo{
				{ID: "mistral-large-latest", Name: "Mistral Large", Provider: "mistral", ContextSize: 128000, Cost: 2.0},
				{ID: "mistral-small-latest", Name: "Mistral Small", Provider: "mistral", ContextSize: 128000, Cost: 0.2},
				{ID: "codestral-latest", Name: "Codestral", Provider: "mistral", ContextSize: 256000, Cost: 0.3},
				{ID: "open-mistral-nemo", Name: "Mistral Nemo", Provider: "mistral", ContextSize: 128000, Cost: 0.15},
			},
			info: &provider.ProviderInfo{
				Name:         "mistral",
				Version:      "1.0.0",
				Description:  "Mistral AI API provider (in-process)",
				Capabilities: []string{"completion", "streaming", "tool_calling"},
			},
		},
	}
}
