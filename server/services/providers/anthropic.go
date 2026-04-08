package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/DojoGenesis/gateway/provider"
)

type AnthropicProvider struct {
	BaseProvider
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		BaseProvider: BaseProvider{
			Name:       "anthropic",
			BaseURL:    envOrDefault("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
			APIKey:     apiKey,
			Client:     NewHTTPClient(),
			EnvKeyName: "ANTHROPIC_API_KEY",
		},
	}
}

func (p *AnthropicProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:        "anthropic",
		Version:     "1.0.0",
		Description: "Anthropic Claude API provider (in-process)",
		Capabilities: []string{
			"completion", "streaming", "tool_calling",
		},
	}, nil
}

func (p *AnthropicProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Provider: "anthropic", ContextSize: 200000, Cost: 3.0},
		{ID: "claude-haiku-4-20250414", Name: "Claude Haiku 4", Provider: "anthropic", ContextSize: 200000, Cost: 0.25},
		{ID: "claude-opus-4-20250514", Name: "Claude Opus 4", Provider: "anthropic", ContextSize: 200000, Cost: 15.0},
	}, nil
}

// anthropicRequest is the Anthropic Messages API request format.
type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	System      string             `json:"system,omitempty"`
}

type anthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentBlock
}

type anthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// anthropicResponse is the Anthropic Messages API response format.
type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Model   string `json:"model"`
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *AnthropicProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	apiKey := p.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Build Anthropic-format messages -- extract system message
	var system string
	var messages []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		messages = append(messages, anthropicMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	aReq := anthropicRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		Stream:      false,
		System:      system,
	}
	if len(req.Tools) > 0 {
		aReq.Tools = convertToAnthropicTools(req.Tools)
	}

	body, err := json.Marshal(aReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := p.DoRequest(ctx, "POST", "/v1/messages", bytes.NewReader(body), map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var aResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&aResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text content and tool calls
	var content string
	var toolCalls []provider.ToolCall
	for _, block := range aResp.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "tool_use":
			var args map[string]interface{}
			json.Unmarshal(block.Input, &args)
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: args,
			})
		}
	}

	return &provider.CompletionResponse{
		ID:      aResp.ID,
		Model:   aResp.Model,
		Content: content,
		Usage: provider.Usage{
			InputTokens:  aResp.Usage.InputTokens,
			OutputTokens: aResp.Usage.OutputTokens,
			TotalTokens:  aResp.Usage.InputTokens + aResp.Usage.OutputTokens,
		},
		ToolCalls: toolCalls,
	}, nil
}

func (p *AnthropicProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	apiKey := p.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	var system string
	var messages []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		messages = append(messages, anthropicMessage{Role: m.Role, Content: m.Content})
	}

	aReq := anthropicRequest{
		Model: model, Messages: messages, MaxTokens: maxTokens,
		Temperature: req.Temperature, Stream: true, System: system,
	}
	if len(req.Tools) > 0 {
		aReq.Tools = convertToAnthropicTools(req.Tools)
	}

	body, _ := json.Marshal(aReq)
	resp, err := p.DoRequest(ctx, "POST", "/v1/messages", bytes.NewReader(body), map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
		"Accept":            "text/event-stream",
	})
	if err != nil {
		return nil, err
	}

	ch := make(chan *provider.CompletionChunk, 100)
	go func() {
		defer close(ch)
		// Note: resp.Body is closed by StreamSSE, do not close here.
		dataCh := make(chan string, 100)
		go p.StreamSSE(ctx, resp, dataCh)

		for data := range dataCh {
			var event struct {
				Type  string `json:"type"`
				Index int    `json:"index"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			switch event.Type {
			case "content_block_delta":
				if event.Delta.Text != "" {
					ch <- &provider.CompletionChunk{Delta: event.Delta.Text, Done: false}
				}
			case "message_stop":
				ch <- &provider.CompletionChunk{Done: true}
				return
			}
		}
		ch <- &provider.CompletionChunk{Done: true}
	}()
	return ch, nil
}

func (p *AnthropicProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Error: fmt.Sprintf("tool execution not supported by Anthropic provider; tool '%s' should be executed by the agent layer", req.ToolCall.Name),
	}, nil
}

func (p *AnthropicProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("Anthropic does not provide an embeddings endpoint")
}

func convertToAnthropicTools(tools []provider.Tool) []anthropicTool {
	result := make([]anthropicTool, len(tools))
	for i, t := range tools {
		result[i] = anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		}
	}
	return result
}
