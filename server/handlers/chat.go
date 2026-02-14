package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/streaming"
	"github.com/gin-gonic/gin"
)

const (
	ResponseTypeChunk    = "chunk"
	ResponseTypeComplete = "complete"
	ResponseTypeError    = "error"
)

var (
	intentClassifier *agent.IntentClassifier
	primaryAgent     *agent.PrimaryAgent
	userRouter       *services.UserRouter
	responseCache    *agent.ResponseCache
	streamingAgent   *streaming.StreamingAgentWithEvents
	chatPluginMgr    *provider.PluginManager
)

func InitializeChatHandlers(ic *agent.IntentClassifier, pa *agent.PrimaryAgent, ur *services.UserRouter, pm *provider.PluginManager) {
	intentClassifier = ic
	primaryAgent = pa
	userRouter = ur
	chatPluginMgr = pm
	responseCache = agent.NewResponseCache(1*time.Hour, 1000)
	streamingAgent = streaming.NewStreamingAgentWithEvents(pa)
}

type ChatRequest struct {
	Message   string `json:"message"`
	Model     string `json:"model,omitempty"`
	Stream    bool   `json:"stream"`
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

type ChatResponse struct {
	Type    string        `json:"type"`
	Content string        `json:"content"`
	Usage   *provider.Usage `json:"usage,omitempty"`
}

func HandleChat(c *gin.Context) {
	if intentClassifier == nil || primaryAgent == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "chat handler not initialized",
		})
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request format",
			"details": err.Error(),
		})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "message is required",
		})
		return
	}

	// Validate message length to prevent context overflow
	// Rough estimate: 1 token ≈ 4 characters
	// Max context: 8192 tokens ≈ 32,768 characters
	// Leave room for system prompt and response (use 50% of context)
	const maxInputChars = 16384
	if len(req.Message) > maxInputChars {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "message too long",
			"details":       fmt.Sprintf("Message exceeds maximum length of %d characters (got %d). Please shorten your message.", maxInputChars, len(req.Message)),
			"max_length":    maxInputChars,
			"actual_length": len(req.Message),
		})
		return
	}

	if req.SessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "session_id is required",
		})
		return
	}

	// Extract user ID from context (set by auth middleware if present)
	// If not present, userID will be empty string (guest user)
	if req.UserID == "" {
		req.UserID = c.GetString("user_id")
	}

	// Store user ID in context for budget middleware
	c.Set("user_id", req.UserID)

	// Use new routing system
	decision := intentClassifier.Route(req.Message)

	// Structured logging for routing decisions
	log.Printf("[IntentRouter] Query: %q | Category: %s | Handler: %s | Provider: %s | Confidence: %.2f | Reasoning: %v",
		req.Message,
		decision.Category.String(),
		decision.Handler,
		decision.Provider,
		decision.Confidence,
		decision.Reasoning,
	)

	// Route based on decision
	switch decision.Handler {
	case "template":
		handleTemplateQuery(c, &req, decision)
	case "llm-fast", "llm-reasoning":
		if req.Stream {
			handleStreamingQuery(c, &req, decision)
		} else {
			handleNonStreamingQuery(c, &req, decision)
		}
	default:
		// Fallback to llm-fast for unknown handlers
		log.Printf("[IntentRouter] WARNING: Unknown handler %s, falling back to llm-fast", decision.Handler)
		decision.Handler = "llm-fast"
		decision.Provider = "" // Let selectProviderWithRouting resolve to a real provider
		if req.Stream {
			handleStreamingQuery(c, &req, decision)
		} else {
			handleNonStreamingQuery(c, &req, decision)
		}
	}
}

func handleTemplateQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) {
	normalizedQuery := normalizeQuery(req.Message)

	cached, found := responseCache.Get(normalizedQuery)
	var response string

	if found {
		response = cached
		c.Header("X-Cache-Hit", "true")
	} else {
		response = getTemplateResponse(req.Message)
		responseCache.Set(normalizedQuery, response)
		c.Header("X-Cache-Hit", "false")
	}

	// Add routing decision to response headers for debugging
	c.Header("X-Intent-Category", decision.Category.String())
	c.Header("X-Intent-Handler", decision.Handler)
	c.Header("X-Intent-Confidence", fmt.Sprintf("%.2f", decision.Confidence))

	// If template fallback is configured and response is empty, fallback to LLM
	if decision.Fallback == "llm-fast" && response == "I'm here to help! Could you please provide more details about what you'd like to work on?" {
		log.Printf("[IntentRouter] Template response not found, falling back to %s", decision.Fallback)
		// Update decision for fallback
		decision.Handler = decision.Fallback
		decision.Provider = decision.Fallback
		if req.Stream {
			handleStreamingQuery(c, req, decision)
		} else {
			handleNonStreamingQuery(c, req, decision)
		}
		return
	}

	if req.Stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		c.SSEvent(ResponseTypeChunk, ChatResponse{
			Type:    ResponseTypeChunk,
			Content: response,
		})
		c.SSEvent(ResponseTypeComplete, ChatResponse{
			Type: ResponseTypeComplete,
		})
		c.Writer.Flush()
	} else {
		c.JSON(http.StatusOK, ChatResponse{
			Type:    ResponseTypeComplete,
			Content: response,
		})
	}
}

func handleNonStreamingQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) {
	ctx := c.Request.Context()

	// Select provider based on routing decision, user status, and model preference
	providerName, resolvedModel, err := selectProviderWithRouting(req.UserID, req.Model, decision)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to select provider",
			"details": err.Error(),
		})
		return
	}

	log.Printf("[IntentRouter] Selected provider: %s (model: %s) for handler: %s", providerName, resolvedModel, decision.Handler)

	response, err := primaryAgent.HandleQuery(ctx, req.Message, providerName, resolvedModel, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate response",
			"details": err.Error(),
		})
		return
	}

	// Track token usage for budget middleware
	c.Set("token_usage", response.Usage.TotalTokens)
	c.Set("model", response.Model)

	// Add routing decision to response headers for debugging
	c.Header("X-Intent-Category", decision.Category.String())
	c.Header("X-Intent-Handler", decision.Handler)
	c.Header("X-Intent-Confidence", fmt.Sprintf("%.2f", decision.Confidence))

	c.JSON(http.StatusOK, ChatResponse{
		Type:    ResponseTypeComplete,
		Content: response.Content,
		Usage:   &response.Usage,
	})
}

func handleStreamingQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) {
	ctx := c.Request.Context()

	// Select provider based on routing decision, user status, and model preference
	providerName, resolvedModel, err := selectProviderWithRouting(req.UserID, req.Model, decision)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to select provider",
			"details": err.Error(),
		})
		return
	}

	log.Printf("[IntentRouter] Selected provider: %s (model: %s) for handler: %s (streaming)", providerName, resolvedModel, decision.Handler)

	// Build query request for streaming agent
	queryReq := agent.QueryRequest{
		Query:        req.Message,
		ProviderName: providerName,
		ModelID:      resolvedModel,
		UserID:       req.UserID,
		UserTier:     getUserTier(req.UserID),
		UseMemory:    false,
		Temperature:  agent.DefaultTemperature,
		MaxTokens:    agent.DefaultMaxTokens,
	}

	// Use StreamingAgent with detailed events
	stream, err := streamingAgent.HandleQueryStreamingWithEvents(ctx, queryReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to start streaming",
			"details": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("X-Intent-Category", decision.Category.String())
	c.Header("X-Intent-Handler", decision.Handler)
	c.Header("X-Intent-Confidence", fmt.Sprintf("%.2f", decision.Confidence))

	var totalTokens int

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-stream:
			if !ok {
				return false
			}

			// Convert streaming event to SSE format
			switch event.Type {
			case streaming.IntentClassified:
				c.SSEvent(string(streaming.IntentClassified), event.Data)
			case streaming.ProviderSelected:
				c.SSEvent(string(streaming.ProviderSelected), event.Data)
			case streaming.MemoryRetrieved:
				c.SSEvent(string(streaming.MemoryRetrieved), event.Data)
			case streaming.Thinking:
				c.SSEvent(string(streaming.Thinking), event.Data)
			case streaming.ToolInvoked:
				c.SSEvent(string(streaming.ToolInvoked), event.Data)
			case streaming.ToolCompleted:
				c.SSEvent(string(streaming.ToolCompleted), event.Data)
			case streaming.ArtifactCreated:
				c.SSEvent(string(streaming.ArtifactCreated), event.Data)
			case streaming.ArtifactUpdated:
				c.SSEvent(string(streaming.ArtifactUpdated), event.Data)
			case streaming.ProjectSwitched:
				c.SSEvent(string(streaming.ProjectSwitched), event.Data)
			case streaming.DiagramRendered:
				c.SSEvent(string(streaming.DiagramRendered), event.Data)
			case streaming.OrchestrationPlanCreated:
				c.SSEvent(string(streaming.OrchestrationPlanCreated), event.Data)
			case streaming.OrchestrationNodeStart:
				c.SSEvent(string(streaming.OrchestrationNodeStart), event.Data)
			case streaming.OrchestrationNodeEnd:
				c.SSEvent(string(streaming.OrchestrationNodeEnd), event.Data)
			case streaming.OrchestrationReplanning:
				c.SSEvent(string(streaming.OrchestrationReplanning), event.Data)
			case streaming.OrchestrationComplete:
				c.SSEvent(string(streaming.OrchestrationComplete), event.Data)
			case streaming.OrchestrationFailed:
				c.SSEvent(string(streaming.OrchestrationFailed), event.Data)
			case streaming.ResponseChunk:
				// Use "response_chunk" as SSE event name (matches streaming.ResponseChunk constant)
				// so the frontend can match on currentEvent === "response_chunk"
				c.SSEvent(string(streaming.ResponseChunk), ChatResponse{
					Type:    ResponseTypeChunk,
					Content: event.Data["content"].(string),
				})
			case streaming.Complete:
				if usage, ok := event.Data["usage"].(map[string]interface{}); ok {
					if total, ok := usage["total_tokens"].(int); ok {
						totalTokens = total
						c.Set("token_usage", totalTokens)
					}
				}
				// Use "complete" as SSE event name (matches streaming.Complete constant)
				c.SSEvent(string(streaming.Complete), ChatResponse{
					Type: ResponseTypeComplete,
					Usage: &provider.Usage{
						TotalTokens: totalTokens,
					},
				})
				return false
			case streaming.Error:
				// Use "error" as SSE event name (matches streaming.Error constant)
				c.SSEvent(string(streaming.Error), gin.H{
					"error": event.Data["error"],
				})
				return false
			}
			return true
		case <-ctx.Done():
			return false
		}
	})
}

func getTemplateResponse(message string) string {
	message = normalizeQuery(message)

	greetings := map[string]string{
		"hello":          "Hello! I'm Dojo Genesis. How can I help?",
		"hi":             "Hi! What are you working on?",
		"hey":            "Hey! What can I help with?",
		"good morning":   "Good morning! What's on your agenda?",
		"good afternoon": "Good afternoon! How can I help?",
		"good evening":   "Good evening! What are you building?",
	}

	help := map[string]string{
		"help":            "I'm Dojo Genesis, an AI coding assistant. I can write and debug code, explain concepts, design architectures, refactor, and generate tests. What do you need?",
		"what can you do": "I help with code generation, debugging, architecture, code review, testing, and documentation across 25+ languages. What are you working on?",
		"capabilities":    "I have advanced reasoning, support 25+ languages, tool calling, context-aware help, and streaming responses. What would you like to build?",
	}

	farewell := map[string]string{
		"bye":       "Goodbye! Happy coding!",
		"goodbye":   "See you! Happy coding!",
		"thanks":    "You're welcome! Ask anytime!",
		"thank you": "My pleasure! Ask anytime!",
	}

	for key, response := range greetings {
		if message == key {
			return response
		}
	}

	for key, response := range help {
		if message == key {
			return response
		}
	}

	for key, response := range farewell {
		if message == key {
			return response
		}
	}

	return "I'm here to help! Could you please provide more details about what you'd like to work on?"
}

func normalizeQuery(query string) string {
	query = strings.TrimSpace(query)
	query = strings.Trim(query, ".,:;!?")
	return strings.ToLower(query)
}

// selectProvider selects the appropriate provider based on user status and model preference.
// If a specific model is requested, it routes to the provider that has that model.
// Otherwise, it uses user-based routing (guest vs authenticated).
func selectProvider(userID, model string) (string, error) {
	// If no UserRouter initialized, use empty string (default provider)
	if userRouter == nil {
		return "", nil
	}

	// If specific model requested, find provider for that model
	if model != "" {
		providerName, err := userRouter.SelectProviderForModel(userID, model)
		if err != nil {
			// If model not found, fallback to user-based routing
			return userRouter.SelectProvider(userID)
		}
		return providerName, nil
	}

	// Use user-based routing (guest vs authenticated)
	return userRouter.SelectProvider(userID)
}

// cloudProviders lists provider names considered "cloud" (API-based, higher quality).
// Order determines preference — first loaded wins for reasoning tasks.
var cloudProviders = []string{"deepseek-api", "openai", "anthropic", "google", "groq", "mistral"}

// localProviders lists provider names considered "local" (low latency, free).
// Order determines preference — first loaded wins for fast tasks.
var localProviders = []string{"ollama", "embedded-qwen3"}

// resolveProviderAlias maps abstract intent-classifier provider names (like "llm-fast"
// and "llm-reasoning") to actual registered plugin names. It dynamically checks which
// providers are actually loaded (i.e., have valid API keys and running plugin processes)
// and picks the best available one — no hardcoded fallbacks.
func resolveProviderAlias(provider string) string {
	if chatPluginMgr == nil {
		return provider
	}

	switch provider {
	case "llm-fast":
		// Fast inference: prefer local providers, then fall back to any cloud provider
		if name := firstLoadedProvider(localProviders); name != "" {
			return name
		}
		if name := firstLoadedProvider(cloudProviders); name != "" {
			return name
		}
	case "llm-reasoning":
		// Reasoning: prefer cloud providers, then fall back to any local provider
		if name := firstLoadedProvider(cloudProviders); name != "" {
			return name
		}
		if name := firstLoadedProvider(localProviders); name != "" {
			return name
		}
	default:
		// Direct provider name — check if loaded, otherwise find any alternative
		if _, err := chatPluginMgr.GetProvider(provider); err == nil {
			return provider
		}
		// Try cloud first, then local
		if name := firstLoadedProvider(cloudProviders); name != "" {
			return name
		}
		if name := firstLoadedProvider(localProviders); name != "" {
			return name
		}
	}

	// Absolute last resort: return original name and let caller handle the error
	return provider
}

// firstLoadedProvider returns the first provider from the given list that is
// currently loaded in the plugin manager, or "" if none are loaded.
func firstLoadedProvider(candidates []string) string {
	for _, name := range candidates {
		if _, err := chatPluginMgr.GetProvider(name); err == nil {
			return name
		}
	}
	return ""
}

// selectProviderWithRouting selects the appropriate provider based on routing decision,
// user status, and model preference. Respects the provider specified by the intent classifier.
// Returns the provider name and the resolved model ID (which may be empty if the requested
// model was not found and the provider should use its default).
//
// Priority order:
//  1. Explicit model requested by user (req.Model)
//  2. Provider from routing decision (decision.Provider), resolved through alias mapping
//  3. User-based routing (guest vs authenticated)
func selectProviderWithRouting(userID, model string, decision agent.RoutingDecision) (string, string, error) {
	// If specific model requested by user, honor that (highest priority)
	if model != "" {
		if userRouter != nil {
			providerName, err := userRouter.SelectProviderForModel(userID, model)
			if err == nil {
				return providerName, model, nil
			}
			// Model not found in any provider — clear model so provider uses its default
			log.Printf("[chat] Model %q not found in any provider, will use provider default", model)
		}
	}

	// If routing decision specifies a provider, resolve aliases and use it
	if decision.Provider != "" {
		resolved := resolveProviderAlias(decision.Provider)
		log.Printf("[chat] Resolved provider alias: %q → %q", decision.Provider, resolved)
		return resolved, "", nil
	}

	// Fallback to user-based routing
	if userRouter != nil {
		provider, err := userRouter.SelectProvider(userID)
		return provider, "", err
	}

	// Final fallback: empty string (default provider)
	return "", "", nil
}

// getUserTier determines the user tier based on user ID.
// Empty user ID means guest tier.
func getUserTier(userID string) string {
	if userID == "" {
		return "guest"
	}
	// TODO: Implement proper user tier lookup from database
	// For now, all authenticated users are considered "authenticated" tier
	return "authenticated"
}
