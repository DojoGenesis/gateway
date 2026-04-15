package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/DojoGenesis/gateway/provider"
)

// voyageProvider implements the Voyage AI embeddings API.
// Voyage uses an OpenAI-compatible /v1/embeddings endpoint.
// Only GenerateEmbedding is supported; all other methods return descriptive errors.
type voyageProvider struct {
	BaseProvider
}

// NewVoyageProvider creates a Voyage AI provider. apiKey may be empty;
// the provider will fall back to the VOYAGE_API_KEY environment variable.
func NewVoyageProvider(apiKey string) *voyageProvider {
	if apiKey == "" {
		apiKey = os.Getenv("VOYAGE_API_KEY")
	}
	return &voyageProvider{
		BaseProvider: BaseProvider{
			Name:       "voyage",
			BaseURL:    "https://api.voyageai.com",
			APIKey:     apiKey,
			Client:     NewHTTPClient(),
			EnvKeyName: "VOYAGE_API_KEY",
		},
	}
}

func (p *voyageProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:         "voyage",
		Version:      "1.0.0",
		Description:  "Voyage AI embeddings provider",
		Capabilities: []string{"embedding"},
	}, nil
}

func (p *voyageProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{ID: "voyage-3-large", Name: "Voyage 3 Large", Provider: "voyage"},
		{ID: "voyage-3", Name: "Voyage 3", Provider: "voyage"},
		{ID: "voyage-3-lite", Name: "Voyage 3 Lite", Provider: "voyage"},
		{ID: "voyage-code-3", Name: "Voyage Code 3", Provider: "voyage"},
	}, nil
}

func (p *voyageProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return nil, fmt.Errorf("voyage provider supports embeddings only; use a full-capability provider for chat completions")
}

func (p *voyageProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	return nil, fmt.Errorf("voyage provider supports embeddings only; use a full-capability provider for chat completions")
}

func (p *voyageProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return nil, fmt.Errorf("voyage provider supports embeddings only; use a full-capability provider for tool calling")
}

func (p *voyageProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	apiKey := p.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("voyage: VOYAGE_API_KEY is not set")
	}

	reqBody := map[string]interface{}{
		"input": text,
		"model": "voyage-3-large",
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("voyage: marshal embedding request: %w", err)
	}

	resp, err := p.DoRequest(ctx, "POST", "/v1/embeddings", bytes.NewReader(bodyBytes), map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("voyage: embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("voyage: decode embedding response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("voyage: empty embedding response")
	}
	return result.Data[0].Embedding, nil
}
