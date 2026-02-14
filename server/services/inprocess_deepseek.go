package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

// APIKeyResolver is a function that dynamically resolves the API key at request time.
// This allows the provider to pick up API keys added via the Dev Mode UI
// without requiring a restart.
type APIKeyResolver func(ctx context.Context) string

// InProcessDeepSeekProvider implements provider.ModelProvider for the DeepSeek API
// without requiring an external plugin binary. This serves as a fallback when
// the plugin subprocess cannot be loaded (missing binary, crashed, etc.).
//
// It uses the same DeepSeek API as the plugin version but runs in-process,
// avoiding gRPC overhead and subprocess management complexity.
//
// The provider supports dynamic API key resolution: if an APIKeyResolver is set,
// it will be called at each request to get the current API key. This allows
// keys added through the Dev Mode UI to be picked up immediately.
type InProcessDeepSeekProvider struct {
	apiKey      string
	keyResolver APIKeyResolver
	baseURL     string
	client      *http.Client
}

// NewInProcessDeepSeekProvider creates a new in-process DeepSeek API provider.
// It reads the API key from the environment if not provided.
func NewInProcessDeepSeekProvider(apiKey string) *InProcessDeepSeekProvider {
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}

	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	return &InProcessDeepSeekProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// SetKeyResolver sets a dynamic API key resolver that is called at each request.
// This allows the provider to pick up API keys added via the Dev Mode GUI
// without requiring a server restart.
func (p *InProcessDeepSeekProvider) SetKeyResolver(resolver APIKeyResolver) {
	p.keyResolver = resolver
}

// resolveAPIKey returns the current API key, checking the dynamic resolver first,
// then falling back to the static key, then environment variable.
func (p *InProcessDeepSeekProvider) resolveAPIKey(ctx context.Context) string {
	// 1. Try dynamic resolver (picks up keys from secure storage / Dev Mode UI)
	if p.keyResolver != nil {
		if key := p.keyResolver(ctx); key != "" {
			return key
		}
	}

	// 2. Use static key (from config or environment)
	if p.apiKey != "" {
		return p.apiKey
	}

	// 3. Last resort: check environment again (may have been set after startup)
	return os.Getenv("DEEPSEEK_API_KEY")
}

func (p *InProcessDeepSeekProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:        "deepseek-api",
		Version:     "1.0.0-inprocess",
		Description: "DeepSeek API provider (in-process fallback)",
		Capabilities: []string{
			"completion",
			"streaming",
			"tool_calling",
		},
	}, nil
}

func (p *InProcessDeepSeekProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{
			ID:          "deepseek-chat",
			Name:        "DeepSeek Chat",
			Provider:    "deepseek-api",
			ContextSize: 32768,
			Cost:        0.14,
		},
		{
			ID:          "deepseek-reasoner",
			Name:        "DeepSeek Reasoner",
			Provider:    "deepseek-api",
			ContextSize: 32768,
			Cost:        0.55,
		},
	}, nil
}

// deepseekRequest is the API request format for DeepSeek.
type deepseekRequest struct {
	Model       string            `json:"model"`
	Messages    []deepseekMessage `json:"messages"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream"`
	Tools       []deepseekTool    `json:"tools,omitempty"`
}

type deepseekMessage struct {
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	ToolCalls  []deepseekToolCall `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
}

type deepseekTool struct {
	Type     string           `json:"type"`
	Function deepseekFunction `json:"function"`
}

type deepseekFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type deepseekToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

// deepseekResponse is the API response format for DeepSeek.
type deepseekResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role      string             `json:"role"`
			Content   string             `json:"content"`
			ToolCalls []deepseekToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// deepseekStreamChunk is a single chunk from the streaming API.
type deepseekStreamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Role      string             `json:"role"`
			Content   string             `json:"content"`
			ToolCalls []deepseekToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

func (p *InProcessDeepSeekProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	apiKey := p.resolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY is not set. Please set it in your environment or via the Dev Mode settings UI (Settings > Dev Mode > Add DeepSeek API Key)")
	}

	model := req.Model
	if model == "" {
		model = "deepseek-chat"
	}

	// Build request
	dsReq := deepseekRequest{
		Model:       model,
		Messages:    convertToDeepSeekMessages(req.Messages),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      false,
	}

	if len(req.Tools) > 0 {
		dsReq.Tools = convertToDeepSeekTools(req.Tools)
	}

	body, err := json.Marshal(dsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var dsResp deepseekResponse
	if err := json.NewDecoder(resp.Body).Decode(&dsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	content := ""
	var toolCalls []provider.ToolCall

	if len(dsResp.Choices) > 0 {
		content = dsResp.Choices[0].Message.Content
		toolCalls = convertFromDeepSeekToolCalls(dsResp.Choices[0].Message.ToolCalls)
	}

	return &provider.CompletionResponse{
		ID:      dsResp.ID,
		Model:   dsResp.Model,
		Content: content,
		Usage: provider.Usage{
			InputTokens:  dsResp.Usage.PromptTokens,
			OutputTokens: dsResp.Usage.CompletionTokens,
			TotalTokens:  dsResp.Usage.TotalTokens,
		},
		ToolCalls: toolCalls,
	}, nil
}

func (p *InProcessDeepSeekProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	apiKey := p.resolveAPIKey(ctx)
	if apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY is not set. Please set it in your environment or via the Dev Mode settings UI (Settings > Dev Mode > Add DeepSeek API Key)")
	}

	model := req.Model
	if model == "" {
		model = "deepseek-chat"
	}

	dsReq := deepseekRequest{
		Model:       model,
		Messages:    convertToDeepSeekMessages(req.Messages),
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	}

	if len(req.Tools) > 0 {
		dsReq.Tools = convertToDeepSeekTools(req.Tools)
	}

	body, err := json.Marshal(dsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	ch := make(chan *provider.CompletionChunk, 100)
	completionID := fmt.Sprintf("deepseek-stream-%d", time.Now().UnixNano())

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		scanner := newSSEScanner(resp.Body)

		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- &provider.CompletionChunk{
					ID:    completionID,
					Delta: "",
					Done:  true,
				}
				return
			}

			var chunk deepseekStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				slog.Error("failed to decode stream chunk", "error", err)
				continue
			}

			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta.Content
				if delta != "" {
					select {
					case ch <- &provider.CompletionChunk{
						ID:    completionID,
						Delta: delta,
						Done:  false,
					}:
					case <-ctx.Done():
						return
					}
				}

				if chunk.Choices[0].FinishReason != nil {
					ch <- &provider.CompletionChunk{
						ID:    completionID,
						Delta: "",
						Done:  true,
					}
					return
				}
			}
		}

		_ = decoder // suppress unused warning
	}()

	return ch, nil
}

func (p *InProcessDeepSeekProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	// DeepSeek API doesn't execute tools — it generates tool call requests.
	// Tool execution happens in the agent layer. Return unsupported.
	return &provider.ToolCallResponse{
		Result: nil,
		Error:  fmt.Sprintf("tool execution not supported by DeepSeek API provider; tool '%s' should be executed by the agent layer", req.ToolCall.Name),
	}, nil
}

func (p *InProcessDeepSeekProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// DeepSeek doesn't have a dedicated embeddings endpoint.
	// Return a simple hash-based embedding as placeholder.
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	embeddingDim := 768
	embedding := make([]float32, embeddingDim)

	for i := 0; i < embeddingDim; i++ {
		hash := 0
		for j, ch := range text {
			hash = (hash*31 + int(ch) + i + j) % 100000
		}
		embedding[i] = float32(hash%10000)/10000.0 - 0.5
	}

	// L2 normalize
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	if norm > 0 {
		invNorm := float32(1.0 / float64(norm))
		for i := range embedding {
			embedding[i] *= invNorm
		}
	}

	return embedding, nil
}

// HasAPIKey returns true if the provider has a valid API key configured
// either statically or via the dynamic resolver.
func (p *InProcessDeepSeekProvider) HasAPIKey() bool {
	if p.apiKey != "" {
		return true
	}
	if p.keyResolver != nil {
		if key := p.keyResolver(context.Background()); key != "" {
			return true
		}
	}
	return os.Getenv("DEEPSEEK_API_KEY") != ""
}

// --- Helper functions ---

func convertToDeepSeekMessages(messages []provider.Message) []deepseekMessage {
	result := make([]deepseekMessage, len(messages))
	for i, m := range messages {
		msg := deepseekMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]deepseekToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				msg.ToolCalls[j] = deepseekToolCall{
					ID:   tc.ID,
					Type: "function",
				}
				msg.ToolCalls[j].Function.Name = tc.Name
				msg.ToolCalls[j].Function.Arguments = argsJSON
			}
		}
		result[i] = msg
	}
	return result
}

func convertToDeepSeekTools(tools []provider.Tool) []deepseekTool {
	result := make([]deepseekTool, len(tools))
	for i, t := range tools {
		result[i] = deepseekTool{
			Type: "function",
			Function: deepseekFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}
	return result
}

func convertFromDeepSeekToolCalls(toolCalls []deepseekToolCall) []provider.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]provider.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		var args map[string]interface{}
		if len(tc.Function.Arguments) > 0 {
			raw := tc.Function.Arguments
			if err := json.Unmarshal(raw, &args); err != nil {
				// Try as double-encoded string
				var s string
				if err2 := json.Unmarshal(raw, &s); err2 == nil {
					if err3 := json.Unmarshal([]byte(s), &args); err3 != nil {
						args = map[string]interface{}{"raw": s}
					}
				} else {
					args = map[string]interface{}{"raw": string(raw)}
				}
			}
		}

		result[i] = provider.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		}
	}
	return result
}

// sseScanner is a simple line scanner for SSE streams.
type sseScanner struct {
	reader *io.Reader
	buf    []byte
	text   string
	err    error
}

func newSSEScanner(r io.Reader) *sseScanner {
	return &sseScanner{
		reader: &r,
		buf:    make([]byte, 0, 4096),
	}
}

func (s *sseScanner) Scan() bool {
	buf := make([]byte, 1)
	var line []byte

	for {
		n, err := (*s.reader).Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				s.text = string(line)
				return true
			}
			if buf[0] != '\r' {
				line = append(line, buf[0])
			}
		}
		if err != nil {
			if len(line) > 0 {
				s.text = string(line)
				return true
			}
			s.err = err
			return false
		}
	}
}

func (s *sseScanner) Text() string {
	return s.text
}
