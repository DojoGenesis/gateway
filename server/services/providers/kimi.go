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
