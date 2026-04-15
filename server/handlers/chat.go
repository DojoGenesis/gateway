package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/agent"
	"github.com/DojoGenesis/gateway/server/database"
	"github.com/DojoGenesis/gateway/server/services"
	"github.com/DojoGenesis/gateway/server/streaming"
	"github.com/gin-gonic/gin"
)

const (
	ResponseTypeChunk    = "chunk"
	ResponseTypeComplete = "complete"
	ResponseTypeError    = "error"
)

// defaultTemplateResponse is the response used when no template matches.
// Comparing against it detects a cache miss and triggers the LLM fallback.
const defaultTemplateResponse = "I'm here to help! Could you please provide more details about what you'd like to work on?"

// cloudProviders lists provider names considered "cloud" (API-based, higher quality).
// Order determines preference — first loaded wins for reasoning tasks.
var cloudProviders = []string{"deepseek-api", "openai", "anthropic", "google", "groq", "mistral"}

// localProviders lists provider names considered "local" (low latency, free).
// Order determines preference — first loaded wins for fast tasks.
var localProviders = []string{"ollama"}

// ChatHandler handles chat-related HTTP requests.
type ChatHandler struct {
	classifier     *agent.IntentClassifier
	semanticRouter *agent.SemanticRouter
	agent          *agent.PrimaryAgent
	router         *services.UserRouter
	cache          *agent.ResponseCache
	streaming      *streaming.StreamingAgentWithEvents
	pluginMgr      *provider.PluginManager
	db             database.DatabaseAdapter

	// specialistRouter dispatches to specialist agents. Nil disables specialist routing.
	specialistRouter SpecialistRouter
	// orchestrator decomposes multi-step queries into plan DAGs. Nil disables orchestrate routing.
	orchestrator Orchestrator
}

// NewChatHandler creates a new ChatHandler with the specified dependencies.
// It initializes derived fields (cache, streaming) internally.
func NewChatHandler(ic *agent.IntentClassifier, pa *agent.PrimaryAgent, ur *services.UserRouter, pm *provider.PluginManager) *ChatHandler {
	return &ChatHandler{
		classifier: ic,
		agent:      pa,
		router:     ur,
		cache:      agent.NewResponseCache(1*time.Hour, 1000),
		streaming:  streaming.NewStreamingAgentWithEvents(pa),
		pluginMgr:  pm,
	}
}

// SpecialistRouter is the interface the chat handler uses to dispatch to specialist agents.
// Satisfied by *specialist.Router.
type SpecialistRouter interface {
	Route(decision agent.RoutingDecision) SpecialistRoutingResult
}

// SpecialistRoutingResult captures the outcome of a specialist routing decision.
type SpecialistRoutingResult struct {
	Routed       bool
	SpecialistID string
	Reason       string
}

// Orchestrator is the interface the chat handler uses for multi-step plan generation.
// Satisfied by the orchestration planner in server/orchestration.
type Orchestrator interface {
	GeneratePlanForChat(ctx context.Context, userID, query string) (planSummary string, err error)
}

// SetSemanticRouter injects the semantic router. When set, the chat handler
// uses it instead of the keyword-based IntentClassifier.
func (h *ChatHandler) SetSemanticRouter(sr *agent.SemanticRouter) {
	h.semanticRouter = sr
}

// SetSpecialistRouter injects the specialist router for domain-specific dispatch.
func (h *ChatHandler) SetSpecialistRouter(sr SpecialistRouter) {
	h.specialistRouter = sr
}

// SetOrchestrator injects the orchestration planner for multi-step queries.
func (h *ChatHandler) SetOrchestrator(o Orchestrator) {
	h.orchestrator = o
}

// SetDB sets the database adapter for user tier lookups.
func (h *ChatHandler) SetDB(db database.DatabaseAdapter) {
	h.db = db
}

type ChatRequest struct {
	Message       string `json:"message"`
	Model         string `json:"model,omitempty"`
	Provider      string `json:"provider,omitempty"`
	Stream        bool   `json:"stream"`
	SessionID     string `json:"session_id"`
	UserID        string `json:"user_id,omitempty"`
	ProjectID     string `json:"project_id,omitempty"`
	WorkspaceRoot string `json:"workspace_root,omitempty"` // User's CWD; file tools resolve relative paths against this
}

type ChatResponse struct {
	Type    string          `json:"type"`
	Content string          `json:"content"`
	Usage   *provider.Usage `json:"usage,omitempty"`
}

// Chat handles chat requests.
func (h *ChatHandler) Chat(c *gin.Context) {
	if h.classifier == nil || h.agent == nil {
		respondInternalError(c, "chat handler not initialized")
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid request format")
		return
	}

	if req.Message == "" {
		respondBadRequest(c, "message is required")
		return
	}

	// Validate message length to prevent context overflow
	// Rough estimate: 1 token ≈ 4 characters
	// Max context: 8192 tokens ≈ 32,768 characters
	// Leave room for system prompt and response (use 50% of context)
	const maxInputChars = 16384
	if len(req.Message) > maxInputChars {
		respondBadRequest(c, "message too long",
			fmt.Sprintf("Message exceeds maximum length of %d characters (got %d). Please shorten your message.", maxInputChars, len(req.Message)))
		return
	}

	if req.SessionID == "" {
		respondBadRequest(c, "session_id is required")
		return
	}

	// Extract user ID from context (set by auth middleware if present)
	// If not present, userID will be empty string (guest user)
	if req.UserID == "" {
		req.UserID = c.GetString("user_id")
	}

	// Store user ID in context for budget middleware
	c.Set("user_id", req.UserID)

	// Route using semantic router when available, falling back to legacy classifier.
	var decision agent.RoutingDecision
	if h.semanticRouter != nil {
		var err error
		decision, err = h.semanticRouter.Route(c.Request.Context(), req.Message)
		if err != nil {
			slog.Warn("semantic router error, falling back to legacy classifier", "error", err)
			decision = h.classifier.Route(req.Message)
		}
	} else {
		decision = h.classifier.Route(req.Message)
	}

	// If the request explicitly specifies a provider, honour it — this overrides the
	// intent classifier's provider selection while preserving its handler/category logic.
	if req.Provider != "" && h.isProviderLoaded(req.Provider) {
		slog.Info("explicit provider from request", "provider", req.Provider)
		decision.Provider = req.Provider
	}

	// Structured logging for routing decisions
	slog.Info("intent classified",
		"query", req.Message,
		"category", decision.Category.String(),
		"handler", decision.Handler,
		"provider", decision.Provider,
		"confidence", decision.Confidence,
		"reasoning", decision.Reasoning,
	)

	// Emit per-route similarity scores as a debug header when Tier 2 was used.
	if len(decision.Scores) > 0 {
		c.Header("X-Route-Scores", formatRouteScores(decision.Scores))
	}

	// Route based on decision
	switch decision.Handler {
	case "template":
		h.handleTemplateQuery(c, &req, decision)
	case "llm-fast", "llm-reasoning":
		if req.Stream {
			h.handleStreamingQuery(c, &req, decision)
		} else {
			h.handleNonStreamingQuery(c, &req, decision)
		}
	case "specialist":
		// Try specialist router; degrade to llm-reasoning if no match or router not wired.
		if h.specialistRouter != nil {
			result := h.specialistRouter.Route(decision)
			if result.Routed {
				slog.Info("specialist dispatch",
					"specialist_id", result.SpecialistID,
					"reason", result.Reason,
				)
				c.Header("X-Specialist-ID", result.SpecialistID)
				decision.SpecialistAgentID = result.SpecialistID
				decision.Handler = "llm-reasoning"
				decision.Provider = "llm-reasoning"
			} else {
				slog.Info("specialist routing: no match, degrading to llm-reasoning",
					"reason", result.Reason,
				)
				c.Header("X-Route-Degraded", "specialist->llm-reasoning")
				decision.Handler = "llm-reasoning"
				decision.Provider = "llm-reasoning"
			}
		} else {
			slog.Warn("specialist router not configured, degrading to llm-reasoning")
			c.Header("X-Route-Degraded", "specialist->llm-reasoning")
			decision.Handler = "llm-reasoning"
			decision.Provider = "llm-reasoning"
		}
		if req.Stream {
			h.handleStreamingQuery(c, &req, decision)
		} else {
			h.handleNonStreamingQuery(c, &req, decision)
		}
	case "orchestrate":
		// Try orchestration planner; degrade to llm-reasoning if not wired or plan fails.
		if h.orchestrator != nil {
			planCtx, planCancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
			planSummary, err := h.orchestrator.GeneratePlanForChat(planCtx, req.UserID, req.Message)
			planCancel()
			if err != nil {
				slog.Warn("orchestration planning failed, degrading to llm-reasoning",
					"error", err,
				)
				c.Header("X-Route-Degraded", "orchestrate->llm-reasoning")
				decision.Handler = "llm-reasoning"
				decision.Provider = "llm-reasoning"
			} else {
				slog.Info("orchestration plan generated", "summary_len", len(planSummary))
				c.Header("X-Orchestrated", "true")
				// Prepend the plan context to the query so the LLM can execute it
				decision.Handler = "llm-reasoning"
				decision.Provider = "llm-reasoning"
				req.Message = fmt.Sprintf("[Orchestration Plan]\n%s\n\n[Original Query]\n%s", planSummary, req.Message)
			}
		} else {
			slog.Warn("orchestrator not configured, degrading to llm-reasoning")
			c.Header("X-Route-Degraded", "orchestrate->llm-reasoning")
			decision.Handler = "llm-reasoning"
			decision.Provider = "llm-reasoning"
		}
		if req.Stream {
			h.handleStreamingQuery(c, &req, decision)
		} else {
			h.handleNonStreamingQuery(c, &req, decision)
		}
	default:
		// Fallback to llm-fast for unknown handlers
		slog.Warn("unknown handler, falling back to llm-fast", "handler", decision.Handler)
		decision.Handler = "llm-fast"
		decision.Provider = "" // Let selectProviderWithRouting resolve to a real provider
		if req.Stream {
			h.handleStreamingQuery(c, &req, decision)
		} else {
			h.handleNonStreamingQuery(c, &req, decision)
		}
	}
}

func (h *ChatHandler) handleTemplateQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) {
	normalizedQuery := normalizeQuery(req.Message)

	cached, found := h.cache.Get(normalizedQuery)
	var response string

	if found {
		response = cached
		c.Header("X-Cache-Hit", "true")
	} else {
		response = getTemplateResponse(req.Message)
		h.cache.Set(normalizedQuery, response)
		c.Header("X-Cache-Hit", "false")
	}

	// Add routing decision to response headers for debugging
	c.Header("X-Intent-Category", decision.Category.String())
	c.Header("X-Intent-Handler", decision.Handler)
	c.Header("X-Intent-Confidence", fmt.Sprintf("%.2f", decision.Confidence))

	// If template fallback is configured and response is empty, fallback to LLM
	if decision.Fallback == "llm-fast" && response == defaultTemplateResponse {
		slog.Info("template response not found, falling back", "fallback", decision.Fallback)
		// Update decision for fallback
		decision.Handler = decision.Fallback
		decision.Provider = decision.Fallback
		if req.Stream {
			h.handleStreamingQuery(c, req, decision)
		} else {
			h.handleNonStreamingQuery(c, req, decision)
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

func (h *ChatHandler) handleNonStreamingQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) {
	ctx := c.Request.Context()

	// Select provider based on routing decision, user status, and model preference
	providerName, resolvedModel, err := h.selectProviderWithRouting(req.UserID, req.Model, decision)
	if err != nil {
		slog.Error("failed to select provider", "error", err)
		respondInternalError(c, "failed to select provider")
		return
	}

	slog.Info("selected provider", "provider", providerName, "model", resolvedModel, "handler", decision.Handler)

	response, err := h.agent.HandleQuery(ctx, req.Message, providerName, resolvedModel, req.UserID)
	if err != nil {
		slog.Error("failed to generate response", "error", err)
		respondInternalError(c, "failed to generate response")
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

func (h *ChatHandler) handleStreamingQuery(c *gin.Context, req *ChatRequest, decision agent.RoutingDecision) {
	// Detach from HTTP request context so client disconnect/response flush
	// doesn't cancel in-flight LLM calls.
	detachedCtx, agentCancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Minute)
	defer agentCancel()
	ctx := detachedCtx

	// Select provider based on routing decision, user status, and model preference
	providerName, resolvedModel, err := h.selectProviderWithRouting(req.UserID, req.Model, decision)
	if err != nil {
		slog.Error("failed to select provider", "error", err)
		respondInternalError(c, "failed to select provider")
		return
	}

	slog.Info("selected provider", "provider", providerName, "model", resolvedModel, "handler", decision.Handler, "streaming", true)

	// Build query request for streaming agent
	queryReq := agent.QueryRequest{
		Query:         req.Message,
		ProviderName:  providerName,
		ModelID:       resolvedModel,
		UserID:        req.UserID,
		UserTier:      h.getUserTier(req.UserID),
		UseMemory:     false,
		Temperature:   agent.DefaultTemperature,
		MaxTokens:     agent.DefaultMaxTokens,
		ProjectID:     req.ProjectID,
		WorkspaceRoot: req.WorkspaceRoot,
	}

	// Use StreamingAgent with detailed events
	stream, err := h.streaming.HandleQueryStreamingWithEvents(ctx, queryReq)
	if err != nil {
		slog.Error("failed to start streaming", "error", err)
		respondInternalError(c, "failed to start streaming")
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
				content, _ := event.Data["content"].(string)
				c.SSEvent(string(streaming.ResponseChunk), ChatResponse{
					Type:    ResponseTypeChunk,
					Content: content,
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
				errorMsg := "unknown error"
				if err, ok := event.Data["error"]; ok && err != nil {
					if errStr, ok := err.(string); ok {
						errorMsg = errStr
					} else if errVal, ok := err.(error); ok {
						errorMsg = errVal.Error()
					}
				}
				c.SSEvent(string(streaming.Error), gin.H{
					"error": errorMsg,
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

	return defaultTemplateResponse
}

func normalizeQuery(query string) string {
	query = strings.TrimSpace(query)
	query = strings.Trim(query, ".,:;!?")
	return strings.ToLower(query)
}

// resolveProviderAlias maps abstract intent-classifier provider names (like "llm-fast"
// and "llm-reasoning") to actual registered plugin names. It dynamically checks which
// providers are actually loaded (i.e., have valid API keys and running plugin processes)
// and picks the best available one — no hardcoded fallbacks.
func (h *ChatHandler) resolveProviderAlias(provider string) string {
	if h.pluginMgr == nil {
		return provider
	}

	switch provider {
	case "llm-fast":
		// Fast inference: prefer local providers, then fall back to any cloud provider
		if name := h.firstLoadedProvider(localProviders); name != "" {
			return name
		}
		if name := h.firstLoadedProvider(cloudProviders); name != "" {
			return name
		}
	case "llm-reasoning":
		// Reasoning: prefer cloud providers, then fall back to any local provider
		if name := h.firstLoadedProvider(cloudProviders); name != "" {
			return name
		}
		if name := h.firstLoadedProvider(localProviders); name != "" {
			return name
		}
	default:
		// Direct provider name — check if loaded, otherwise find any alternative
		if _, err := h.pluginMgr.GetProvider(provider); err == nil {
			return provider
		}
		// Try cloud first, then local
		if name := h.firstLoadedProvider(cloudProviders); name != "" {
			return name
		}
		if name := h.firstLoadedProvider(localProviders); name != "" {
			return name
		}
	}

	// Absolute last resort: return original name and let caller handle the error
	return provider
}

// firstLoadedProvider returns the first provider from the given list that is
// currently loaded in the plugin manager, or "" if none are loaded.
func (h *ChatHandler) firstLoadedProvider(candidates []string) string {
	for _, name := range candidates {
		if _, err := h.pluginMgr.GetProvider(name); err == nil {
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
func (h *ChatHandler) selectProviderWithRouting(userID, model string, decision agent.RoutingDecision) (string, string, error) {
	// If specific model requested by user, honor that (highest priority)
	if model != "" {
		if h.router != nil {
			providerName, err := h.router.SelectProviderForModel(userID, model)
			if err == nil {
				return providerName, model, nil
			}
			// Model not found in any provider's list — try prefix-based provider inference
			// so users can pass short-form IDs like "claude-sonnet-4-6" without needing
			// them in the ListModels registry.
			slog.Warn("model not found in any provider, attempting prefix inference", "model", model)

			lowerModel := strings.ToLower(model)
			prefixToProvider := map[string]string{
				"claude-":     "anthropic",
				"gpt-":        "openai",
				"o1-":         "openai",
				"o3":          "openai",
				"o4-":         "openai",
				"chatgpt-":    "openai",
				"gemini-":     "google",
				"llama-":      "groq",
				"mixtral-":    "groq",
				"mistral-":    "mistral",
				"codestral-":  "mistral",
				"deepseek-":   "deepseek",
				"moonshot-":   "kimi",
				"kimi-":       "kimi",
			}
			for prefix, provName := range prefixToProvider {
				if strings.HasPrefix(lowerModel, prefix) {
					if h.isProviderLoaded(provName) {
						slog.Info("model matched by prefix inference", "model", model, "provider", provName)
						return provName, model, nil
					}
				}
			}
			slog.Warn("prefix inference found no loaded provider, falling through to intent classifier", "model", model)
		}
	}

	// If routing decision specifies a provider, resolve aliases and use it
	if decision.Provider != "" {
		resolved := h.resolveProviderAlias(decision.Provider)
		slog.Info("resolved provider alias", "from", decision.Provider, "to", resolved)
		return resolved, "", nil
	}

	// Fallback to user-based routing
	if h.router != nil {
		provider, err := h.router.SelectProvider(userID)
		return provider, "", err
	}

	// Final fallback: empty string (default provider)
	return "", "", nil
}

// isProviderLoaded returns true if the named provider is currently registered
// and reachable in the plugin manager.
func (h *ChatHandler) isProviderLoaded(name string) bool {
	if h.pluginMgr == nil {
		return false
	}
	_, err := h.pluginMgr.GetProvider(name)
	return err == nil
}

// getUserTier determines the user tier based on user ID.
// Empty user ID means guest tier.
func (h *ChatHandler) getUserTier(userID string) string {
	if userID == "" {
		return "guest"
	}

	if h.db != nil {
		if tier := h.lookupUserTier(context.Background(), userID); tier != "" {
			return tier
		}
	}

	// Falls back to "free" tier when DB unavailable or user has no settings
	return "free"
}

// lookupUserTier attempts to resolve the user's tier from database settings.
// Returns the tier string if found, or empty string on any error.
func (h *ChatHandler) lookupUserTier(ctx context.Context, userID string) string {
	settings, err := h.db.GetSettings(ctx, userID)
	if err != nil {
		return ""
	}

	if settings.Preferences == nil {
		return ""
	}

	var prefs map[string]interface{}
	if err := json.Unmarshal([]byte(*settings.Preferences), &prefs); err != nil {
		return ""
	}

	if tier, ok := prefs["tier"].(string); ok && tier != "" {
		return tier
	}

	return ""
}

// formatRouteScores produces a compact "route=score" string for the top-3
// routes by descending similarity, e.g. "fast_inference=0.82,direct_response=0.71,deep_inference=0.65".
// Intended for the X-Route-Scores debug header.
func formatRouteScores(scores map[string]float64) string {
	type routeScore struct {
		name  string
		score float64
	}
	pairs := make([]routeScore, 0, len(scores))
	for name, score := range scores {
		pairs = append(pairs, routeScore{name, score})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].score > pairs[j].score
	})
	if len(pairs) > 3 {
		pairs = pairs[:3]
	}
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = fmt.Sprintf("%s=%.4f", p.name, p.score)
	}
	return strings.Join(parts, ",")
}
