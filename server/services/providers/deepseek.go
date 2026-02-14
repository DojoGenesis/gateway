package providers

import (
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

// DeepSeekProvider implements the DeepSeek API using the shared OpenAI-compatible base.
// DeepSeek's API follows the OpenAI Chat Completions format.
type DeepSeekProvider struct {
	openaiCompatibleProvider
}

func NewDeepSeekProvider(apiKey string) *DeepSeekProvider {
	return &DeepSeekProvider{
		openaiCompatibleProvider: openaiCompatibleProvider{
			BaseProvider: BaseProvider{
				Name:       "deepseek-api",
				BaseURL:    envOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com") + "/v1",
				APIKey:     apiKey,
				Client:     NewHTTPClient(),
				EnvKeyName: "DEEPSEEK_API_KEY",
			},
			defaultModel: "deepseek-chat",
			models: []provider.ModelInfo{
				{ID: "deepseek-chat", Name: "DeepSeek Chat", Provider: "deepseek-api", ContextSize: 32768, Cost: 0.14},
				{ID: "deepseek-reasoner", Name: "DeepSeek Reasoner", Provider: "deepseek-api", ContextSize: 32768, Cost: 0.55},
			},
			info: &provider.ProviderInfo{
				Name:         "deepseek-api",
				Version:      "1.0.0",
				Description:  "DeepSeek API provider (in-process)",
				Capabilities: []string{"completion", "streaming", "tool_calling"},
			},
		},
	}
}
