package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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
		// Canonical date-stamped IDs
		{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Provider: "anthropic", ContextSize: 200000, Cost: 3.0},
		{ID: "claude-haiku-4-20250414", Name: "Claude Haiku 4", Provider: "anthropic", ContextSize: 200000, Cost: 0.25},
		{ID: "claude-opus-4-20250514", Name: "Claude Opus 4", Provider: "anthropic", ContextSize: 200000, Cost: 15.0},
		// Short-form aliases that users actually type (routed to Anthropic API which resolves them)
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Provider: "anthropic", ContextSize: 200000, Cost: 3.0},
		{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", Provider: "anthropic", ContextSize: 200000, Cost: 0.25},
		{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", Provider: "anthropic", ContextSize: 200000, Cost: 15.0},
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
	ToolChoice  interface{}        `json:"tool_choice,omitempty"`
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

	system, messages := convertToAnthropicMessages(req.Messages)

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
		// Anthropic tool_choice: {"type": "any"} = model must call a tool;
		// omit entirely for "auto" behaviour (Anthropic's default).
		switch req.ToolChoice {
		case "required":
			aReq.ToolChoice = map[string]string{"type": "any"}
		case "none":
			aReq.ToolChoice = map[string]string{"type": "none"}
		// "auto" or "" → omit (Anthropic defaults to auto when tools are present)
		}
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

	system, messages := convertToAnthropicMessages(req.Messages)

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
	return nil, fmt.Errorf("%s: Anthropic does not offer an embeddings API; set DOJO_EMBEDDING_PROVIDER to openai, google, voyage, or ollama", p.Name)
}

// convertToAnthropicMessages converts provider messages into Anthropic API format.
// It extracts the system prompt, converts "tool" role messages into user messages
// with tool_result content blocks, and merges consecutive tool results into a single
// user message (Anthropic requires alternating user/assistant roles).
func convertToAnthropicMessages(msgs []provider.Message) (system string, messages []anthropicMessage) {
	for _, m := range msgs {
		if m.Role == "system" {
			if system == "" {
				system = m.Content
			} else {
				system = system + "\n\n" + m.Content
			}
			continue
		}
		// Convert OpenAI-style "tool" role into Anthropic tool_result user message.
		if m.Role == "tool" {
			block := map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": m.ToolCallID,
				"content":     m.Content,
			}
			// Merge into the previous user message if it already contains tool_result blocks
			// to avoid consecutive user messages (which Anthropic rejects).
			if n := len(messages); n > 0 && messages[n-1].Role == "user" {
				if blocks, ok := messages[n-1].Content.([]map[string]interface{}); ok {
					messages[n-1].Content = append(blocks, block)
					continue
				}
			}
			messages = append(messages, anthropicMessage{
				Role:    "user",
				Content: []map[string]interface{}{block},
			})
			continue
		}
		// Convert assistant messages that carry tool_use calls into content-block format.
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			var blocks []map[string]interface{}
			if m.Content != "" {
				blocks = append(blocks, map[string]interface{}{
					"type": "text",
					"text": m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				input := tc.Arguments
				if input == nil {
					input = map[string]interface{}{}
				}
				blocks = append(blocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": input,
				})
			}
			messages = append(messages, anthropicMessage{
				Role:    "assistant",
				Content: blocks,
			})
			continue
		}
		// Safety: reject any remaining "tool" role that somehow slipped through.
		if m.Role != "user" && m.Role != "assistant" {
			slog.Warn("dropping message with unexpected role for Anthropic",
				"role", m.Role, "content_preview", truncateStr(m.Content, 80))
			continue
		}
		messages = append(messages, anthropicMessage{Role: m.Role, Content: m.Content})
	}
	return system, messages
}

// truncateStr truncates a string to maxLen runes.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
