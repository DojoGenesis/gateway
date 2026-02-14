package providers

import (
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

type OpenAIProvider struct {
	openaiCompatibleProvider
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		openaiCompatibleProvider: openaiCompatibleProvider{
			BaseProvider: BaseProvider{
				Name:       "openai",
				BaseURL:    envOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
				APIKey:     apiKey,
				Client:     NewHTTPClient(),
				EnvKeyName: "OPENAI_API_KEY",
			},
			defaultModel: "gpt-4o",
			models: []provider.ModelInfo{
				{ID: "gpt-4o", Name: "GPT-4o", Provider: "openai", ContextSize: 128000, Cost: 5.0},
				{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai", ContextSize: 128000, Cost: 0.15},
				{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "openai", ContextSize: 128000, Cost: 10.0},
				{ID: "o1", Name: "o1", Provider: "openai", ContextSize: 200000, Cost: 15.0},
				{ID: "o3-mini", Name: "o3-mini", Provider: "openai", ContextSize: 200000, Cost: 1.1},
			},
			info: &provider.ProviderInfo{
				Name:         "openai",
				Version:      "1.0.0",
				Description:  "OpenAI API provider (in-process)",
				Capabilities: []string{"completion", "streaming", "tool_calling"},
			},
		},
	}
}
