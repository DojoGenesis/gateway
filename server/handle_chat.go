package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
}

// OpenAIChatMessage represents a message in OpenAI format.
type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIChatResponse is the non-streaming chat completion response (OpenAI format).
type OpenAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []OpenAIChatChoice `json:"choices"`
	Usage   OpenAIUsage        `json:"usage"`
}

// OpenAIChatChoice represents a choice in the response.
type OpenAIChatChoice struct {
	Index        int                `json:"index"`
	Message      *OpenAIChatMessage `json:"message,omitempty"`
	Delta        *OpenAIChatMessage `json:"delta,omitempty"`
	FinishReason *string            `json:"finish_reason"`
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
	if s.pluginManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "No providers configured")
		return
	}

	// Get the last user message for the agent query
	lastUserMsg := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMsg = req.Messages[i].Content
			break
		}
	}

	if lastUserMsg == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "No user message found")
		return
	}

	// ─── Route bypass: X-Route: direct skips all injection ───────────
	directRoute := c.GetHeader("X-Route") == "direct"

	// ─── Base system prompt injection ──────────────────────────────────
	// Prepends a system message from SYSTEM_PROMPT or SYSTEM_PROMPT_FILE env vars.
	// Skipped when: (a) caller already sent a system message (Route E), or
	// (b) X-Route: direct header is set (Route A).
	if !directRoute && !hasCallerSystemMessage(req.Messages) {
		if sysPrompt := loadSystemPrompt(); sysPrompt != "" {
			sysMsg := OpenAIChatMessage{Role: "system", Content: sysPrompt}
			req.Messages = append([]OpenAIChatMessage{sysMsg}, req.Messages...)
			slog.Debug("base system prompt injected", "chars", len(sysPrompt))
		}
	}

	// ─── RAG context injection ──────────────────────────────────
	if !directRoute {
		if userID, ok := getUserIDFromContext(c); ok && s.authDB != nil {
			ragCtx, ragErr := s.BuildRAGContext(ctx, userID, lastUserMsg, 5)
			if ragErr != nil {
				slog.Warn("rag context retrieval failed", "error", ragErr)
			} else if ragCtx != "" {
				// Prepend RAG context as a system message
				ragMsg := OpenAIChatMessage{Role: "system", Content: ragCtx}
				req.Messages = append([]OpenAIChatMessage{ragMsg}, req.Messages...)
				slog.Debug("rag context injected", "user_id", userID, "chars", len(ragCtx))
			}
		}
	}

	// Build provider completion request directly
	messages := make([]provider.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = provider.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	temp := 0.7
	if req.Temperature != nil {
		temp = *req.Temperature
	}
	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	completionReq := &provider.CompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
		Stream:      false,
	}

	// Try to find the right provider for the model
	prov, err := s.resolveProvider(req.Model)
	if err != nil {
		s.errorResponse(c, http.StatusBadRequest, "model_not_found", "Model not available: "+err.Error())
		return
	}

	callStart := time.Now()
	resp, err := prov.GenerateCompletion(ctx, completionReq)
	latencyMs := time.Since(callStart).Milliseconds()

	// Record latency for provider history tracking
	if s.latencyTracker != nil {
		provName := s.resolveProviderName(req.Model)
		s.latencyTracker.Record(provName, latencyMs, err != nil)
	}

	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "provider_error", "Completion failed: "+err.Error())
		return
	}

	completionID := "chatcmpl-" + uuid.New().String()[:12]
	finishReason := "stop"

	openAIResp := OpenAIChatResponse{
		ID:      completionID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []OpenAIChatChoice{
			{
				Index: 0,
				Message: &OpenAIChatMessage{
					Role:    "assistant",
					Content: resp.Content,
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

	c.JSON(http.StatusOK, openAIResp)
}

func (s *Server) streamChatCompletions(c *gin.Context, ctx context.Context, req *OpenAIChatRequest) {
	if s.pluginManager == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "server_error", "No providers configured")
		return
	}

	// ─── Route bypass: X-Route: direct skips all injection ───────────
	directRoute := c.GetHeader("X-Route") == "direct"

	// ─── Base system prompt injection ──────────────────────────────────
	// Skipped when: (a) caller already sent a system message (Route E), or
	// (b) X-Route: direct header is set (Route A).
	if !directRoute && !hasCallerSystemMessage(req.Messages) {
		if sysPrompt := loadSystemPrompt(); sysPrompt != "" {
			sysMsg := OpenAIChatMessage{Role: "system", Content: sysPrompt}
			req.Messages = append([]OpenAIChatMessage{sysMsg}, req.Messages...)
			slog.Debug("base system prompt injected", "chars", len(sysPrompt))
		}
	}

	// ─── RAG context injection ──────────────────────────────────
	// Extract last user message for RAG query
	lastUserMsg := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMsg = req.Messages[i].Content
			break
		}
	}
	if !directRoute {
		if userID, ok := getUserIDFromContext(c); ok && s.authDB != nil && lastUserMsg != "" {
			ragCtx, ragErr := s.BuildRAGContext(ctx, userID, lastUserMsg, 5)
			if ragErr != nil {
				slog.Warn("rag context retrieval failed", "error", ragErr)
			} else if ragCtx != "" {
				ragMsg := OpenAIChatMessage{Role: "system", Content: ragCtx}
				req.Messages = append([]OpenAIChatMessage{ragMsg}, req.Messages...)
				slog.Debug("rag context injected", "user_id", userID, "chars", len(ragCtx))
			}
		}
	}

	messages := make([]provider.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = provider.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	temp := 0.7
	if req.Temperature != nil {
		temp = *req.Temperature
	}
	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	completionReq := &provider.CompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
		Stream:      true,
	}

	prov, err := s.resolveProvider(req.Model)
	if err != nil {
		s.errorResponse(c, http.StatusBadRequest, "model_not_found", "Model not available: "+err.Error())
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

func (s *Server) writeSSEChunk(w http.ResponseWriter, flusher http.Flusher, chunk OpenAIStreamChunk) {
	data, _ := json.Marshal(chunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// resolveProvider finds the appropriate ModelProvider for a given model name.
func (s *Server) resolveProvider(model string) (provider.ModelProvider, error) {
	if s.pluginManager == nil {
		return nil, fmt.Errorf("no plugin manager configured")
	}

	// If model is empty, use the first available provider.
	if model == "" {
		providers := s.pluginManager.GetProviders()
		for _, prov := range providers {
			return prov, nil
		}
		return nil, fmt.Errorf("no providers available")
	}

	// Step 1: Exact model match — ask each provider if it has this model.
	providers := s.pluginManager.GetProviders()
	for _, prov := range providers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		models, err := prov.ListModels(ctx)
		cancel()
		if err != nil {
			continue
		}
		for _, m := range models {
			if m.ID == model || m.Name == model {
				return prov, nil
			}
		}
	}

	// Step 2: Model-prefix-to-provider inference.
	lowerModel := strings.ToLower(model)
	prefixMap := map[string][]string{
		"anthropic": {"claude-"},
		"openai":    {"gpt-", "o1-", "o3", "o4-", "chatgpt-"},
		"google":    {"gemini-"},
		"groq":      {"llama-", "mixtral-"},
		"mistral":   {"mistral-", "codestral-"},
		"deepseek":  {"deepseek-"},
		"kimi":      {"moonshot-", "kimi-"},
	}
	for providerName, prefixes := range prefixMap {
		for _, prefix := range prefixes {
			if strings.HasPrefix(lowerModel, prefix) {
				if prov, ok := providers[providerName]; ok {
					return prov, nil
				}
			}
		}
	}

	// Step 3: Fallback — try the first available provider.
	for _, prov := range providers {
		return prov, nil
	}

	return nil, fmt.Errorf("no provider available for model %q", model)
}

// resolveProviderName returns the provider name for a given model string,
// used for latency tracking attribution. Falls back to "unknown" if the model
// cannot be matched to a specific provider.
func (s *Server) resolveProviderName(model string) string {
	if s.pluginManager == nil {
		return "unknown"
	}
	providers := s.pluginManager.GetProviders()
	for name, prov := range providers {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		models, err := prov.ListModels(ctx)
		cancel()
		if err != nil {
			continue
		}
		for _, m := range models {
			if m.ID == model || m.Name == model {
				return name
			}
		}
	}
	return "unknown"
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
