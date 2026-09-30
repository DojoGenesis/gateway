package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/provider"
)

type OllamaProvider struct {
	BaseProvider
	defaultModel string

	// toolCapable caches, per model, whether Ollama reports the "tools"
	// capability (/api/show). See modelSupportsTools.
	toolCapable sync.Map // model name -> bool
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
		defaultModel: os.Getenv("OLLAMA_DEFAULT_MODEL"), // empty = auto-detect first available
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
	// Tools are Ollama's native function calling. This field did not exist,
	// so the "native" path silently dropped every tool (DGS-141).
	Tools []ollamaTool `json:"tools,omitempty"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	// ToolName names the tool a role "tool" message answers; Ollama matches
	// results by name.
	ToolName string `json:"tool_name,omitempty"`
}

type ollamaTool struct {
	Type     string             `json:"type"`
	Function ollamaToolFunction `json:"function"`
}

type ollamaToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ollamaToolCall is Ollama's tool-call shape: arguments are a JSON OBJECT,
// unlike OpenAI's JSON-encoded string.
type ollamaToolCall struct {
	ID       string `json:"id,omitempty"`
	Function struct {
		Index     int                    `json:"index,omitempty"`
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	} `json:"function"`
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

// resolveModel returns a model name, falling back to the first available Ollama model
// if the configured default is empty.
func (p *OllamaProvider) resolveModel(ctx context.Context, requested string) string {
	if requested != "" {
		return requested
	}
	if p.defaultModel != "" {
		return p.defaultModel
	}
	// Auto-detect: pick the first available model
	models, err := p.ListModels(ctx)
	if err == nil && len(models) > 0 {
		return models[0].ID
	}
	return "llama3.2" // last-resort fallback
}

func (p *OllamaProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	model := p.resolveModel(ctx, req.Model)

	// Route through text-mode tool fallback for models without native tool support
	if len(req.Tools) > 0 && !p.modelSupportsTools(ctx, model) {
		return p.generateWithTextToolFallback(ctx, req)
	}

	oReq := ollamaRequest{
		Model:    model,
		Messages: convertToOllamaMessages(req.Messages),
		Stream:   false,
		Tools:    convertToOllamaTools(req.Tools),
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
		Model:     oResp.Model,
		Content:   oResp.Message.Content,
		ToolCalls: convertFromOllamaToolCalls(oResp.Message.ToolCalls),
	}, nil
}

func (p *OllamaProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	model := p.resolveModel(ctx, req.Model)

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

// modelSupportsTools reports whether Ollama takes native tools for a model.
// Ollama's own answer comes first: /api/show lists a model's capabilities,
// and "tools" means native function calling. It's cached per model. Only an
// Ollama too old to report capabilities (or unreachable for /api/show) falls
// back to the name list below, which used to be the whole decision and could
// only ever be stale.
func (p *OllamaProvider) modelSupportsTools(ctx context.Context, modelName string) bool {
	if v, ok := p.toolCapable.Load(modelName); ok {
		return v.(bool)
	}
	if capable, known := p.showToolCapability(ctx, modelName); known {
		p.toolCapable.Store(modelName, capable)
		return capable
	}
	return modelNameSuggestsTools(modelName)
}

// showToolCapability asks /api/show. known is false when Ollama gave no
// capabilities list, so the caller must fall back.
func (p *OllamaProvider) showToolCapability(ctx context.Context, modelName string) (capable, known bool) {
	body, _ := json.Marshal(map[string]string{"model": modelName})
	resp, err := p.DoRequest(ctx, "POST", "/api/show", bytes.NewReader(body), nil)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()
	var show struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&show); err != nil || show.Capabilities == nil {
		return false, false
	}
	for _, c := range show.Capabilities {
		if c == "tools" {
			return true, true
		}
	}
	return false, true
}

// modelNameSuggestsTools is the fallback name list for Ollama versions that
// don't report capabilities.
func modelNameSuggestsTools(modelName string) bool {
	supported := []string{
		"llama3.1", "llama3.2", "llama3.3",
		"mistral-nemo", "mistral-large",
		"firefunction-v2",
		"command-r", "command-r-plus",
		"smollm2",
		"qwen2.5", "qwen2.5-coder",
		"hermes3",
	}
	for _, s := range supported {
		if strings.Contains(strings.ToLower(modelName), s) {
			return true
		}
	}
	return false
}

// generateWithTextToolFallback builds a text description of available tools and
// appends it to the system prompt so models without native tool-calling can still
// use tools through a structured text protocol.  The model is asked to respond
// with a JSON block when it wants to invoke a tool:
//
//	```tool_call
//	{"name":"<tool>","arguments":{...}}
//	```
//
// The caller (runAgentLoop) already handles native tool calls; for Ollama models
// that don't support them natively, GenerateCompletion is called without a Tools
// slice but with this augmented system message.
func (p *OllamaProvider) generateWithTextToolFallback(
	ctx context.Context,
	req *provider.CompletionRequest,
) (*provider.CompletionResponse, error) {
	if len(req.Tools) == 0 {
		return p.GenerateCompletion(ctx, req)
	}

	// Build human-readable tool catalogue
	var sb strings.Builder
	sb.WriteString("\n\n--- AVAILABLE TOOLS ---\n")
	sb.WriteString("You can invoke tools by responding with a JSON block in this format:\n")
	sb.WriteString("```tool_call\n{\"name\":\"<tool_name>\",\"arguments\":{...}}\n```\n\n")
	sb.WriteString("Tools:\n")
	for _, t := range req.Tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.Description))
	}
	sb.WriteString("--- END TOOLS ---")

	// Inject into the system message (or prepend one)
	augmented := make([]provider.Message, 0, len(req.Messages))
	injected := false
	for _, m := range req.Messages {
		if m.Role == "system" && !injected {
			augmented = append(augmented, provider.Message{
				Role:    "system",
				Content: m.Content + sb.String(),
			})
			injected = true
		} else {
			augmented = append(augmented, m)
		}
	}
	if !injected {
		augmented = append([]provider.Message{{Role: "system", Content: sb.String()}}, augmented...)
	}

	// Strip native tools — model doesn't support them
	fallbackReq := *req
	fallbackReq.Tools = nil
	fallbackReq.Messages = augmented

	return p.GenerateCompletion(ctx, &fallbackReq)
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
	// Ollama matches a tool result to its call by tool NAME, while the
	// provider layer (and OpenAI) carry the call's ID. Resolve IDs to names
	// from the assistant turns that made the calls.
	nameByID := map[string]string{}
	for _, m := range msgs {
		for _, tc := range m.ToolCalls {
			if tc.ID != "" {
				nameByID[tc.ID] = tc.Name
			}
		}
	}
	result := make([]ollamaMessage, len(msgs))
	for i, m := range msgs {
		om := ollamaMessage{Role: m.Role, Content: m.Content}
		for j, tc := range m.ToolCalls {
			var call ollamaToolCall
			call.ID = tc.ID
			call.Function.Index = j
			call.Function.Name = tc.Name
			call.Function.Arguments = tc.Arguments
			if call.Function.Arguments == nil {
				call.Function.Arguments = map[string]interface{}{}
			}
			om.ToolCalls = append(om.ToolCalls, call)
		}
		if m.Role == "tool" {
			om.ToolName = nameByID[m.ToolCallID]
		}
		result[i] = om
	}
	return result
}

func convertToOllamaTools(tools []provider.Tool) []ollamaTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]ollamaTool, len(tools))
	for i, t := range tools {
		out[i] = ollamaTool{Type: "function", Function: ollamaToolFunction{Name: t.Name, Description: t.Description, Parameters: t.Parameters}}
	}
	return out
}

// convertFromOllamaToolCalls maps Ollama's tool calls to the provider layer.
// Older Ollama versions send no ID; one is minted so a caller can answer
// the call with a tool_call_id.
func convertFromOllamaToolCalls(calls []ollamaToolCall) []provider.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]provider.ToolCall, len(calls))
	for i, c := range calls {
		id := c.ID
		if id == "" {
			id = fmt.Sprintf("call_%d", i)
		}
		out[i] = provider.ToolCall{ID: id, Name: c.Function.Name, Arguments: c.Function.Arguments}
	}
	return out
}
