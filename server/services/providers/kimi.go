package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/DojoGenesis/gateway/provider"
)

type KimiProvider struct {
	openaiCompatibleProvider
}

// kimiRequest is the Kimi-specific request body.
// It extends the standard OpenAI format with:
//   - enable_thinking: false  — disables thinking mode so reasoning_content is
//     not injected into assistant messages; without this, multi-turn tool-calling
//     loops fail because the gateway doesn't preserve reasoning_content between
//     iterations (Kimi returns 400 "reasoning_content is missing").
type kimiRequest struct {
	Model          string       `json:"model"`
	Messages       []oaiMessage `json:"messages"`
	Temperature    float64      `json:"temperature,omitempty"`
	MaxTokens      int          `json:"max_tokens,omitempty"`
	Stream         bool         `json:"stream"`
	Tools          []oaiTool    `json:"tools,omitempty"`
	ToolChoice     interface{}  `json:"tool_choice,omitempty"`
	EnableThinking bool         `json:"enable_thinking"`
}

// buildKimiRequest converts a CompletionRequest into a kimiRequest with
// Kimi-specific constraints applied.
func buildKimiRequest(req *provider.CompletionRequest, stream bool) kimiRequest {
	model := req.Model
	if model == "" {
		model = "kimi-k2.5"
	}
	kr := kimiRequest{
		Model:          model,
		Messages:       convertToOAIMessages(req.Messages),
		Temperature:    1, // K2 models require temperature=1
		MaxTokens:      req.MaxTokens,
		Stream:         stream,
		EnableThinking: false, // disable thinking to prevent reasoning_content in tool-call turns
	}
	if len(req.Tools) > 0 {
		kr.Tools = convertToOAITools(req.Tools)
		// "required" is incompatible with thinking; cap to "auto" regardless
		switch req.ToolChoice {
		case "none":
			kr.ToolChoice = "none"
		default:
			kr.ToolChoice = "auto"
		}
	}
	return kr
}

// GenerateCompletion sends a non-streaming request with Kimi-specific constraints.
func (p *KimiProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	apiKey := p.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("kimi API key is not set")
	}
	kr := buildKimiRequest(req, false)
	body, _ := json.Marshal(kr)
	resp, err := p.DoRequest(ctx, "POST", "/chat/completions", bytes.NewReader(body), map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var oResp oaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&oResp); err != nil {
		return nil, fmt.Errorf("failed to decode kimi response: %w", err)
	}

	var content, reasoningContent string
	var toolCalls []provider.ToolCall
	if len(oResp.Choices) > 0 {
		content = oResp.Choices[0].Message.Content
		reasoningContent = oResp.Choices[0].Message.ReasoningContent
		toolCalls = convertFromOAIToolCalls(oResp.Choices[0].Message.ToolCalls)
	}
	return &provider.CompletionResponse{
		ID: oResp.ID, Model: oResp.Model, Content: content,
		ReasoningContent: reasoningContent,
		Usage: provider.Usage{
			InputTokens: oResp.Usage.PromptTokens, OutputTokens: oResp.Usage.CompletionTokens,
			TotalTokens: oResp.Usage.TotalTokens,
		},
		ToolCalls: toolCalls,
	}, nil
}

// GenerateCompletionStream sends a streaming request with Kimi-specific constraints.
func (p *KimiProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	apiKey := p.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("kimi API key is not set")
	}
	kr := buildKimiRequest(req, true)
	body, _ := json.Marshal(kr)
	resp, err := p.DoRequest(ctx, "POST", "/chat/completions", bytes.NewReader(body), map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Accept":        "text/event-stream",
	})
	if err != nil {
		return nil, err
	}

	ch := make(chan *provider.CompletionChunk, 100)
	go func() {
		defer close(ch)
		dataCh := make(chan string, 100)
		go p.StreamSSE(ctx, resp, dataCh)

		for data := range dataCh {
			var chunk oaiStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta.Content
				if delta != "" {
					ch <- &provider.CompletionChunk{ID: chunk.ID, Delta: delta, Done: false}
				}
				if chunk.Choices[0].FinishReason != nil {
					ch <- &provider.CompletionChunk{ID: chunk.ID, Done: true}
					return
				}
			}
		}
		ch <- &provider.CompletionChunk{Done: true}
	}()
	return ch, nil
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
				{ID: "kimi-k2-0905-preview", Name: "Kimi K2 0905 Preview", Provider: "kimi", ContextSize: 256000, Cost: 0.8},
				{ID: "kimi-k2-0711-preview", Name: "Kimi K2 0711 Preview", Provider: "kimi", ContextSize: 256000, Cost: 0.8},
				{ID: "kimi-k2-turbo-preview", Name: "Kimi K2 Turbo Preview", Provider: "kimi", ContextSize: 256000, Cost: 0.7},
				{ID: "kimi-k2-thinking", Name: "Kimi K2 Thinking", Provider: "kimi", ContextSize: 256000, Cost: 1.0},
				{ID: "kimi-k2-thinking-turbo", Name: "Kimi K2 Thinking Turbo", Provider: "kimi", ContextSize: 256000, Cost: 0.8},
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
