package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/DojoGenesis/gateway/provider"
)

// loadSystemPrompt reads the system prompt from SYSTEM_PROMPT env var or the
// file path specified in SYSTEM_PROMPT_FILE. Returns empty string if neither
// is set or if the file cannot be read.
func loadSystemPrompt() string {
	if content := os.Getenv("SYSTEM_PROMPT"); content != "" {
		return strings.TrimSpace(content)
	}
	if path := os.Getenv("SYSTEM_PROMPT_FILE"); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}

// hasCallerSystemMessage reports whether the request already contains a system
// message from the caller. When true, the env-var system prompt should NOT be
// injected — the caller's system prompt takes precedence.
func hasCallerSystemMessage(msgs []OpenAIChatMessage) bool {
	for _, m := range msgs {
		if m.Role == "system" {
			return true
		}
	}
	return false
}

// ─── OpenAI-Compatible Request/Response Types ────────────────────────────────

// OpenAIChatRequest matches the OpenAI chat completion request format.
type OpenAIChatRequest struct {
	Model            string              `json:"model" binding:"required"`
	Messages         []OpenAIChatMessage `json:"messages" binding:"required"`
	Temperature      *float64            `json:"temperature,omitempty"`
	MaxTokens        *int                `json:"max_tokens,omitempty"`
	Stream           bool                `json:"stream"`
	TopP             *float64            `json:"top_p,omitempty"`
	FrequencyPenalty *float64            `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64            `json:"presence_penalty,omitempty"`
	Stop             interface{}         `json:"stop,omitempty"`
	User             string              `json:"user,omitempty"`
	Metadata         map[string]string   `json:"metadata,omitempty"`
	// Tools and ToolChoice are OpenAI function calling (DGS-141). They used
	// to be dropped on decode, because the struct had no field for them.
	Tools      []OpenAITool    `json:"tools,omitempty"`
	ToolChoice json.RawMessage `json:"tool_choice,omitempty"`
}

// OpenAITool is one entry of a request's `tools` array.
type OpenAITool struct {
	Type     string            `json:"type"`
	Function OpenAIFunctionDef `json:"function"`
}

// OpenAIFunctionDef describes a callable function.
type OpenAIFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// OpenAIToolCall is a function call the model made. On the wire the
// arguments are a JSON-encoded STRING, not an object. Index is set only in
// streaming deltas.
type OpenAIToolCall struct {
	Index    *int               `json:"index,omitempty"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall is the name and JSON-string arguments of a tool call.
type OpenAIFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments"`
}

// OpenAIChatMessage represents a message in OpenAI format.
type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// ToolCalls: an assistant turn that called tools (request history and
	// responses). ToolCallID: a role "tool" turn answering one of them.
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// OpenAIChatResponse is the non-streaming chat completion response (OpenAI format).
type OpenAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenAIChatChoice `json:"choices"`
	Usage   OpenAIUsage        `json:"usage"`
	// DojoInjected is present only when the caller asked for the echo. A
	// pointer so "asked, nothing injected" serializes as [] and "not asked"
	// omits the field.
	DojoInjected *[]InjectedMessage `json:"dojo_injected,omitempty"`
}

// OpenAIChatChoice represents a choice in the response.
type OpenAIChatChoice struct {
	Index        int                `json:"index"`
	Message      *OpenAIChatMessage `json:"message,omitempty"`
	Delta        *OpenAIChatMessage `json:"delta,omitempty"`
	FinishReason *string            `json:"finish_reason"`
}

// InjectedMessage is one message the gateway prepended to a caller's
// conversation (DGS-141). Echoed only when the request asks with
// X-Dojo-Echo-Injected; the X-Dojo-Injected response header names the
// sources on every response.
type InjectedMessage struct {
	Source  string `json:"source"` // "base_system_prompt" or "rag"
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIUsage represents token usage in OpenAI format.
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIStreamChunk is a single SSE chunk in the streaming response.
type OpenAIStreamChunk struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenAIChatChoice `json:"choices"`
	// DojoInjected rides on the first chunk only, when the caller asked.
	DojoInjected *[]InjectedMessage `json:"dojo_injected,omitempty"`
}

// handleChatCompletions handles POST /v1/chat/completions (OpenAI-compatible).
func (s *Server) handleChatCompletions(c *gin.Context) {
	var req OpenAIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Invalid request format: "+err.Error())
		return
	}

	if len(req.Messages) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Messages array must not be empty")
		return
	}

	// Default 5 minutes for local LLM inference (large models on consumer GPUs)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Second)
	defer cancel()

	if req.Stream {
		s.streamChatCompletions(c, ctx, &req)
	} else {
		s.nonStreamChatCompletions(c, ctx, &req)
	}
}

func (s *Server) nonStreamChatCompletions(c *gin.Context, ctx context.Context, req *OpenAIChatRequest) {
	completionReq, provName, prov, ok := s.prepareCompletion(c, ctx, req, false)
	if !ok {
		return
	}

	callStart := time.Now()
	resp, err := prov.GenerateCompletion(ctx, completionReq)
	latencyMs := time.Since(callStart).Milliseconds()

	// Record latency for provider history tracking
	if s.latencyTracker != nil {
		s.latencyTracker.Record(provName, latencyMs, err != nil)
	}

	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "provider_error", "Completion failed: "+err.Error())
		return
	}

	completionID := "chatcmpl-" + uuid.New().String()[:12]
	finishReason := "stop"
	if len(resp.ToolCalls) > 0 {
		finishReason = "tool_calls"
	}

	openAIResp := OpenAIChatResponse{
		ID:      completionID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []OpenAIChatChoice{
			{
				Index: 0,
				Message: &OpenAIChatMessage{
					Role:      "assistant",
					Content:   resp.Content,
					ToolCalls: toOpenAIToolCalls(resp.ToolCalls, false),
				},
				FinishReason: &finishReason,
			},
		},
		Usage: OpenAIUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	openAIResp.DojoInjected = injectedEcho(c)
	c.JSON(http.StatusOK, openAIResp)
}

func (s *Server) streamChatCompletions(c *gin.Context, ctx context.Context, req *OpenAIChatRequest) {
	completionReq, _, prov, ok := s.prepareCompletion(c, ctx, req, true)
	if !ok {
		return
	}
	// Provider stream chunks carry text only (provider.CompletionChunk has no
	// tool calls), so a streamed request with tools would lose every call the
	// model makes. Serve it with one non-streaming call and emit the result
	// as spec-valid SSE instead. It arrives all at once, not token by token;
	// that is the honest trade for not dropping the calls.
	if len(completionReq.Tools) > 0 {
		s.streamToolCompletion(c, ctx, req, completionReq, prov)
		return
	}

	chunkChan, err := prov.GenerateCompletionStream(ctx, completionReq)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "provider_error", "Stream failed: "+err.Error())
		return
	}

	completionID := "chatcmpl-" + uuid.New().String()[:12]
	created := time.Now().Unix()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Streaming not supported")
		return
	}

	// Send initial role chunk
	initialChunk := OpenAIStreamChunk{
		ID:      completionID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   req.Model,
		Choices: []OpenAIChatChoice{
			{
				Index: 0,
				Delta: &OpenAIChatMessage{
					Role:    "assistant",
					Content: "",
				},
				FinishReason: nil,
			},
		},
	}
	initialChunk.DojoInjected = injectedEcho(c)
	s.writeSSEChunk(c.Writer, flusher, initialChunk)

	for chunk := range chunkChan {
		if chunk.Done {
			finishReason := "stop"
			finalChunk := OpenAIStreamChunk{
				ID:      completionID,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   req.Model,
				Choices: []OpenAIChatChoice{
					{
						Index:        0,
						Delta:        &OpenAIChatMessage{Content: ""},
						FinishReason: &finishReason,
					},
				},
			}
			s.writeSSEChunk(c.Writer, flusher, finalChunk)
			break
		}

		contentChunk := OpenAIStreamChunk{
			ID:      completionID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   req.Model,
			Choices: []OpenAIChatChoice{
				{
					Index:        0,
					Delta:        &OpenAIChatMessage{Content: chunk.Delta},
					FinishReason: nil,
				},
			},
		}
		s.writeSSEChunk(c.Writer, flusher, contentChunk)
	}

	// Send [DONE] terminator
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

// prepareCompletion is the one place a /v1/chat/completions request becomes
// model input, for both the streaming and the non-streaming path. It used to
// be written twice, and the copies had drifted: only the non-stream copy
// refused a request with no user message. On failure it has already written
// the error response and returns ok=false.
//
// Injection rules (DGS-141 documents why they matter to callers that log
// every model input):
//   - X-Route: direct skips every injection below.
//   - The base system prompt (SYSTEM_PROMPT / SYSTEM_PROMPT_FILE) is
//     prepended only when the caller sent no system message.
//   - RAG context is prepended whenever the request carries a user ID and
//     the auth DB is available, even when the caller sent a system message.
func (s *Server) prepareCompletion(c *gin.Context, ctx context.Context, req *OpenAIChatRequest, stream bool) (*provider.CompletionRequest, string, provider.ModelProvider, bool) {
	if s.pluginManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "No providers configured")
		return nil, "", nil, false
	}

	lastUserMsg := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMsg = req.Messages[i].Content
			break
		}
	}
	if lastUserMsg == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "No user message found")
		return nil, "", nil, false
	}

	// injected mirrors, in message order, every message prepended below.
	injected := []InjectedMessage{}
	if c.GetHeader("X-Route") != "direct" {
		if !hasCallerSystemMessage(req.Messages) {
			if sysPrompt := loadSystemPrompt(); sysPrompt != "" {
				req.Messages = append([]OpenAIChatMessage{{Role: "system", Content: sysPrompt}}, req.Messages...)
				injected = append([]InjectedMessage{{Source: "base_system_prompt", Role: "system", Content: sysPrompt}}, injected...)
				slog.Debug("base system prompt injected", "chars", len(sysPrompt))
			}
		}
		if userID, ok := getUserIDFromContext(c); ok && s.authDB != nil {
			ragCtx, ragErr := s.BuildRAGContext(ctx, userID, lastUserMsg, 5)
			if ragErr != nil {
				slog.Warn("rag context retrieval failed", "error", ragErr)
			} else if ragCtx != "" {
				req.Messages = append([]OpenAIChatMessage{{Role: "system", Content: ragCtx}}, req.Messages...)
				injected = append([]InjectedMessage{{Source: "rag", Role: "system", Content: ragCtx}}, injected...)
				slog.Debug("rag context injected", "user_id", userID, "chars", len(ragCtx))
			}
		}
	}
	// Every response names what was injected, so any caller can tell the
	// model saw text it never sent. The full text is echoed on request.
	sources := make([]string, len(injected))
	for i, m := range injected {
		sources[i] = m.Source
	}
	if len(sources) == 0 {
		c.Header("X-Dojo-Injected", "none")
	} else {
		c.Header("X-Dojo-Injected", strings.Join(sources, ","))
	}
	if echoRequested(c) {
		c.Set(injectedEchoKey, &injected)
	}

	messages, tools, toolChoice, convErr := convertToolFields(req)
	if convErr != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", convErr.Error())
		return nil, "", nil, false
	}
	temp := 0.7
	if req.Temperature != nil {
		temp = *req.Temperature
	}
	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	name, prov, err := s.resolveProvider(req.Model)
	if err != nil {
		s.errorResponse(c, http.StatusBadRequest, "model_not_found", "Model not available: "+err.Error())
		return nil, "", nil, false
	}
	return &provider.CompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
		Stream:      stream,
		Tools:       tools,
		ToolChoice:  toolChoice,
	}, name, prov, true
}

// streamToolCompletion answers a streamed request that carries tools: one
// non-streaming provider call, emitted as the OpenAI SSE sequence (role chunk,
// content and/or tool_calls delta, finish chunk, [DONE]).
func (s *Server) streamToolCompletion(c *gin.Context, ctx context.Context, req *OpenAIChatRequest, completionReq *provider.CompletionRequest, prov provider.ModelProvider) {
	completionReq.Stream = false
	resp, err := prov.GenerateCompletion(ctx, completionReq)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "provider_error", "Completion failed: "+err.Error())
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", "Streaming not supported")
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	id := "chatcmpl-" + uuid.New().String()[:12]
	created := time.Now().Unix()
	chunk := func(delta *OpenAIChatMessage, finish *string) OpenAIStreamChunk {
		return OpenAIStreamChunk{ID: id, Object: "chat.completion.chunk", Created: created, Model: req.Model,
			Choices: []OpenAIChatChoice{{Index: 0, Delta: delta, FinishReason: finish}}}
	}
	first := chunk(&OpenAIChatMessage{Role: "assistant"}, nil)
	first.DojoInjected = injectedEcho(c)
	s.writeSSEChunk(c.Writer, flusher, first)
	if resp.Content != "" || len(resp.ToolCalls) > 0 {
		s.writeSSEChunk(c.Writer, flusher, chunk(&OpenAIChatMessage{
			Content:   resp.Content,
			ToolCalls: toOpenAIToolCalls(resp.ToolCalls, true),
		}, nil))
	}
	finish := "stop"
	if len(resp.ToolCalls) > 0 {
		finish = "tool_calls"
	}
	s.writeSSEChunk(c.Writer, flusher, chunk(&OpenAIChatMessage{}, &finish))
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

// convertToolFields maps the OpenAI wire's tool fields onto the provider
// layer, which already supports all of them. Anything it cannot carry
// faithfully is a 400 naming the field, never a silent drop (DGS-141).
func convertToolFields(req *OpenAIChatRequest) ([]provider.Message, []provider.Tool, string, error) {
	messages := make([]provider.Message, len(req.Messages))
	for i, m := range req.Messages {
		pm := provider.Message{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for j, tc := range m.ToolCalls {
			args := map[string]interface{}{}
			if strings.TrimSpace(tc.Function.Arguments) != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					return nil, nil, "", fmt.Errorf("messages[%d].tool_calls[%d].function.arguments must be a JSON object encoded as a string: %v", i, j, err)
				}
			}
			pm.ToolCalls = append(pm.ToolCalls, provider.ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: args})
		}
		messages[i] = pm
	}

	var tools []provider.Tool
	for i, t := range req.Tools {
		if t.Type != "" && t.Type != "function" {
			return nil, nil, "", fmt.Errorf("tools[%d].type %q is not supported; only \"function\"", i, t.Type)
		}
		if strings.TrimSpace(t.Function.Name) == "" {
			return nil, nil, "", fmt.Errorf("tools[%d].function.name is required", i)
		}
		tools = append(tools, provider.Tool{Name: t.Function.Name, Description: t.Function.Description, Parameters: t.Function.Parameters})
	}

	toolChoice := ""
	if raw := strings.TrimSpace(string(req.ToolChoice)); raw != "" && raw != "null" {
		var choice string
		if err := json.Unmarshal(req.ToolChoice, &choice); err != nil {
			return nil, nil, "", fmt.Errorf("tool_choice naming a specific function is not supported by this gateway; use \"auto\", \"none\" or \"required\"")
		}
		switch choice {
		case "auto", "none", "required":
			toolChoice = choice
		default:
			return nil, nil, "", fmt.Errorf("tool_choice %q is not one of \"auto\", \"none\", \"required\"", choice)
		}
	}
	return messages, tools, toolChoice, nil
}

// toOpenAIToolCalls renders provider tool calls on the OpenAI wire: type
// "function" and arguments as a JSON string. withIndex adds the index a
// streaming delta requires.
func toOpenAIToolCalls(calls []provider.ToolCall, withIndex bool) []OpenAIToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]OpenAIToolCall, len(calls))
	for i, tc := range calls {
		args := []byte("{}")
		if tc.Arguments != nil {
			if b, err := json.Marshal(tc.Arguments); err == nil {
				args = b
			}
		}
		out[i] = OpenAIToolCall{ID: tc.ID, Type: "function", Function: OpenAIFunctionCall{Name: tc.Name, Arguments: string(args)}}
		if withIndex {
			idx := i
			out[i].Index = &idx
		}
	}
	return out
}

const injectedEchoKey = "dojo_injected_echo"

// echoRequested reports whether the caller asked for the injected text.
func echoRequested(c *gin.Context) bool {
	v := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Dojo-Echo-Injected")))
	return v == "1" || v == "true"
}

// injectedEcho returns the echo prepareCompletion stored, or nil when the
// caller did not ask for one.
func injectedEcho(c *gin.Context) *[]InjectedMessage {
	if v, ok := c.Get(injectedEchoKey); ok {
		return v.(*[]InjectedMessage)
	}
	return nil
}

func (s *Server) writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk OpenAIStreamChunk) {
	data, _ := json.Marshal(chunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// modelPrefixes maps a registered provider name to the model-id prefixes it
// serves. The names must be the ones services.RegisterProviders registers —
// DeepSeek is "deepseek-api", and this table used to say "deepseek", so the
// prefix never matched and deepseek-* requests fell through to the fallback.
var modelPrefixes = []struct {
	provider string
	prefixes []string
}{
	{"anthropic", []string{"claude-"}},
	{"openai", []string{"gpt-", "o1-", "o3", "o4-", "chatgpt-"}},
	{"google", []string{"gemini-"}},
	{"groq", []string{"llama-", "mixtral-"}},
	{"mistral", []string{"mistral-", "codestral-"}},
	{"deepseek-api", []string{"deepseek-"}},
	{"kimi", []string{"moonshot-", "kimi-"}},
}

// resolveProvider finds the provider for a model and returns its registered
// name with it (the name is what latency tracking records). Order:
//  1. a provider whose ListModels lists the model exactly;
//  2. the model-id prefix table above;
//  3. the first provider by name.
//
// Every step walks providers in name order. They used to walk a Go map, so
// with two candidates the answer could change from one request to the next.
func (s *Server) resolveProvider(model string) (string, provider.ModelProvider, error) {
	if s.pluginManager == nil {
		return "", nil, fmt.Errorf("no plugin manager configured")
	}
	providers := s.pluginManager.GetProviders()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", nil, fmt.Errorf("no providers available")
	}

	if model != "" {
		for _, name := range names {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			models, err := providers[name].ListModels(ctx)
			cancel()
			if err != nil {
				continue
			}
			for _, m := range models {
				if m.ID == model || m.Name == model {
					return name, providers[name], nil
				}
			}
		}

		lowerModel := strings.ToLower(model)
		for _, row := range modelPrefixes {
			prov, ok := providers[row.provider]
			if !ok {
				continue
			}
			for _, prefix := range row.prefixes {
				if strings.HasPrefix(lowerModel, prefix) {
					return row.provider, prov, nil
				}
			}
		}
	}

	return names[0], providers[names[0]], nil
}

// errorResponse sends a consistent error response.
// All HTTP endpoints MUST use this function for error responses to ensure
// integrators see a uniform JSON shape:
//
//	{"error": {"code": "string", "message": "string", "details": {}}}
func (s *Server) errorResponse(c *gin.Context, status int, code, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":       code,
			"message":    message,
			"details":    gin.H{},
			"request_id": requestID,
		},
	})
}

// errorResponseWithDetails sends a consistent error response with additional details.
func (s *Server) errorResponseWithDetails(c *gin.Context, status int, code, message string, details gin.H) {
	requestID, _ := c.Get("request_id")
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":       code,
			"message":    message,
			"details":    details,
			"request_id": requestID,
		},
	})
}
