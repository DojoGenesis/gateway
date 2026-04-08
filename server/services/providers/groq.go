package providers

import (
	"github.com/DojoGenesis/gateway/provider"
)

type GroqProvider struct {
	openaiCompatibleProvider
}

func NewGroqProvider(apiKey string) *GroqProvider {
	return &GroqProvider{
		openaiCompatibleProvider: openaiCompatibleProvider{
			BaseProvider: BaseProvider{
				Name:       "groq",
				BaseURL:    "https://api.groq.com/openai/v1",
				APIKey:     apiKey,
				Client:     NewHTTPClient(),
				EnvKeyName: "GROQ_API_KEY",
			},
			defaultModel: "llama-3.3-70b-versatile",
			models: []provider.ModelInfo{
				{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B", Provider: "groq", ContextSize: 131072, Cost: 0.59},
				{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B", Provider: "groq", ContextSize: 131072, Cost: 0.05},
				{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", Provider: "groq", ContextSize: 32768, Cost: 0.24},
				{ID: "gemma2-9b-it", Name: "Gemma2 9B", Provider: "groq", ContextSize: 8192, Cost: 0.20},
			},
			info: &provider.ProviderInfo{
				Name:         "groq",
				Version:      "1.0.0",
				Description:  "Groq API provider (in-process)",
				Capabilities: []string{"completion", "streaming", "tool_calling"},
			},
		},
	}
}
