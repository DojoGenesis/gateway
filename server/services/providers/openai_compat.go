package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/DojoGenesis/gateway/provider"
)

// openaiCompatibleProvider implements the ModelProvider interface for any
// provider that uses the OpenAI Chat Completions API format.
// OpenAI, Groq, Mistral, Kimi, and DeepSeek all use this base.
type openaiCompatibleProvider struct {
	BaseProvider
	defaultModel string
	models       []provider.ModelInfo
	info         *provider.ProviderInfo
}

// --- OpenAI-format request/response types ---

type oaiRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Stream      bool         `json:"stream"`
	Tools       []oaiTool    `json:"tools,omitempty"`
	ToolChoice  interface{}  `json:"tool_choice,omitempty"`
}

type oaiMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	ToolCalls  []oaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type oaiTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Parameters  interface{} `json:"parameters,omitempty"`
	} `json:"function"`
}

type oaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type oaiResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role      string        `json:"role"`
			Content   string        `json:"content"`
			ToolCalls []oaiToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type oaiStreamChunk struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content   string        `json:"content"`
			ToolCalls []oaiToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func (p *openaiCompatibleProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return p.info, nil
}

func (p *openaiCompatibleProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return p.models, nil
}

func (p *openaiCompatibleProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	apiKey := p.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("%s API key is not set", p.Name)
	}

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	oReq := oaiRequest{
		Model:       model,
		Messages:    convertToOAIMessages(req.Messages),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
	}
	if len(req.Tools) > 0 {
		oReq.Tools = convertToOAITools(req.Tools)
		switch req.ToolChoice {
		case "required":
			oReq.ToolChoice = "required"
		case "none":
			oReq.ToolChoice = "none"
		default:
			oReq.ToolChoice = "auto"
		}
	}

	body, _ := json.Marshal(oReq)
	resp, err := p.DoRequest(ctx, "POST", "/chat/completions", bytes.NewReader(body), map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var oResp oaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&oResp); err != nil {
		return nil, fmt.Errorf("failed to decode %s response: %w", p.Name, err)
	}

	var content string
	var toolCalls []provider.ToolCall
	if len(oResp.Choices) > 0 {
		content = oResp.Choices[0].Message.Content
		toolCalls = convertFromOAIToolCalls(oResp.Choices[0].Message.ToolCalls)
	}

	return &provider.CompletionResponse{
		ID: oResp.ID, Model: oResp.Model, Content: content,
		Usage: provider.Usage{
			InputTokens: oResp.Usage.PromptTokens, OutputTokens: oResp.Usage.CompletionTokens,
			TotalTokens: oResp.Usage.TotalTokens,
		},
		ToolCalls: toolCalls,
	}, nil
}

func (p *openaiCompatibleProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	apiKey := p.ResolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("%s API key is not set", p.Name)
	}

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	oReq := oaiRequest{
		Model: model, Messages: convertToOAIMessages(req.Messages),
		Temperature: req.Temperature, MaxTokens: req.MaxTokens, Stream: true,
	}
	if len(req.Tools) > 0 {
		oReq.Tools = convertToOAITools(req.Tools)
	}

	body, _ := json.Marshal(oReq)
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

func (p *openaiCompatibleProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Error: fmt.Sprintf("tool execution not supported by %s provider", p.Name),
	}, nil
}

func (p *openaiCompatibleProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("%s: embedding not implemented via this provider", p.Name)
}

// --- Shared conversion helpers ---

func convertToOAIMessages(msgs []provider.Message) []oaiMessage {
	result := make([]oaiMessage, len(msgs))
	for i, m := range msgs {
		msg := oaiMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		// Convert provider.ToolCall to OpenAI format so assistant messages
		// that invoked tools include the tool_calls array. Without this,
		// subsequent "tool" role messages are rejected by the API.
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]oaiToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				msg.ToolCalls[j] = oaiToolCall{
					ID:   tc.ID,
					Type: "function",
				}
				msg.ToolCalls[j].Function.Name = tc.Name
				msg.ToolCalls[j].Function.Arguments = string(argsJSON)
			}
		}
		result[i] = msg
	}
	return result
}

func convertToOAITools(tools []provider.Tool) []oaiTool {
	result := make([]oaiTool, len(tools))
	for i, t := range tools {
		result[i] = oaiTool{Type: "function"}
		result[i].Function.Name = t.Name
		result[i].Function.Description = t.Description
		result[i].Function.Parameters = t.Parameters
	}
	return result
}

func convertFromOAIToolCalls(toolCalls []oaiToolCall) []provider.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}
	result := make([]provider.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		result[i] = provider.ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: args}
	}
	return result
}
