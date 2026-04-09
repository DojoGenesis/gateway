package providers

import (
	"github.com/DojoGenesis/gateway/provider"
)

type KimiProvider struct {
	openaiCompatibleProvider
}

func NewKimiProvider(apiKey string) *KimiProvider {
	return &KimiProvider{
		openaiCompatibleProvider: openaiCompatibleProvider{
			BaseProvider: BaseProvider{
				Name:       "kimi",
				BaseURL:    envOrDefault("KIMI_BASE_URL", "https://api.moonshot.ai/v1"),
				APIKey:     apiKey,
				Client:     NewHTTPClient(),
				EnvKeyName: "KIMI_API_KEY",
			},
			defaultModel: "kimi-k2.5",
			models: []provider.ModelInfo{
				{ID: "kimi-k2.5", Name: "Kimi K2.5", Provider: "kimi", ContextSize: 256000, Cost: 1.0},
				{ID: "kimi-k2", Name: "Kimi K2", Provider: "kimi", ContextSize: 256000, Cost: 0.8},
				{ID: "kimi-k2-0905", Name: "Kimi K2 0905", Provider: "kimi", ContextSize: 256000, Cost: 0.8},
				{ID: "kimi-latest", Name: "Kimi Latest", Provider: "kimi", ContextSize: 256000, Cost: 1.0},
				{ID: "moonshot-v1-8k", Name: "Moonshot V1 8K", Provider: "kimi", ContextSize: 8000, Cost: 0.5},
				{ID: "moonshot-v1-32k", Name: "Moonshot V1 32K", Provider: "kimi", ContextSize: 32000, Cost: 0.6},
				{ID: "moonshot-v1-128k", Name: "Moonshot V1 128K", Provider: "kimi", ContextSize: 128000, Cost: 0.8},
			},
			info: &provider.ProviderInfo{
				Name:         "kimi",
				Version:      "1.0.0",
				Description:  "Kimi K2.5 API provider (in-process, Moonshot AI)",
				Capabilities: []string{"completion", "streaming", "tool_calling"},
			},
		},
	}
}
