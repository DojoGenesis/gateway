package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

type OllamaProvider struct {
	BaseProvider
	defaultModel string
}

func NewOllamaProvider() *OllamaProvider {
	host := envOrDefault("OLLAMA_HOST", "http://localhost:11434")
	return &OllamaProvider{
		BaseProvider: BaseProvider{
			Name:    "ollama",
			BaseURL: host,
			Client: &http.Client{
				Timeout: 10 * time.Minute, // Local models can be slow
			},
		},
		defaultModel: envOrDefault("OLLAMA_DEFAULT_MODEL", "llama3.2"),
	}
}

// IsAvailable checks if the Ollama server is reachable.
func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(checkCtx, "GET", p.BaseURL+"/api/tags", nil)
	resp, err := p.Client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (p *OllamaProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:         "ollama",
		Version:      "1.0.0",
		Description:  "Local Ollama provider (in-process)",
		Capabilities: []string{"completion", "streaming"},
	}, nil
}

func (p *OllamaProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	resp, err := p.DoRequest(ctx, "GET", "/api/tags", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tagsResp struct {
		Models []struct {
			Name    string `json:"name"`
			Size    int64  `json:"size"`
			Details struct {
				ParameterSize string `json:"parameter_size"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama tags: %w", err)
	}

	models := make([]provider.ModelInfo, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = provider.ModelInfo{
			ID:       m.Name,
			Name:     m.Name,
			Provider: "ollama",
			Cost:     0, // Local inference is free
		}
	}
	return models, nil
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Model   string        `json:"model"`
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}

func (p *OllamaProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	oReq := ollamaRequest{
		Model:    model,
		Messages: convertToOllamaMessages(req.Messages),
		Stream:   false,
	}
	if req.Temperature > 0 || req.MaxTokens > 0 {
		oReq.Options = &ollamaOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		}
	}

	body, _ := json.Marshal(oReq)
	resp, err := p.DoRequest(ctx, "POST", "/api/chat", bytes.NewReader(body), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var oResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&oResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return &provider.CompletionResponse{
		Model:   oResp.Model,
		Content: oResp.Message.Content,
	}, nil
}

func (p *OllamaProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	oReq := ollamaRequest{
		Model: model, Messages: convertToOllamaMessages(req.Messages), Stream: true,
	}
	if req.Temperature > 0 || req.MaxTokens > 0 {
		oReq.Options = &ollamaOptions{Temperature: req.Temperature, NumPredict: req.MaxTokens}
	}

	body, _ := json.Marshal(oReq)
	resp, err := p.DoRequest(ctx, "POST", "/api/chat", bytes.NewReader(body), nil)
	if err != nil {
		return nil, err
	}

	ch := make(chan *provider.CompletionChunk, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk ollamaResponse
			if err := decoder.Decode(&chunk); err != nil {
				if err != io.EOF {
					ch <- &provider.CompletionChunk{Done: true}
				}
				return
			}
			if chunk.Message.Content != "" {
				ch <- &provider.CompletionChunk{Delta: chunk.Message.Content, Done: false}
			}
			if chunk.Done {
				ch <- &provider.CompletionChunk{Done: true}
				return
			}
		}
	}()
	return ch, nil
}

func (p *OllamaProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{Error: "tool execution not supported by Ollama provider"}, nil
}

func (p *OllamaProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	type embReq struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	body, _ := json.Marshal(embReq{Model: p.defaultModel, Prompt: text})
	resp, err := p.DoRequest(ctx, "POST", "/api/embeddings", bytes.NewReader(body), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var embResp struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama embedding: %w", err)
	}
	return embResp.Embedding, nil
}

func convertToOllamaMessages(msgs []provider.Message) []ollamaMessage {
	result := make([]ollamaMessage, len(msgs))
	for i, m := range msgs {
		result[i] = ollamaMessage{Role: m.Role, Content: m.Content}
	}
	return result
}
