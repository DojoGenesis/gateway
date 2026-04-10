package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/memory"
	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	providerpkg "github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/artifacts"
	"github.com/DojoGenesis/gateway/server/projects"
	"github.com/DojoGenesis/gateway/server/trace"
	"github.com/DojoGenesis/gateway/tools"
)

const (
	DefaultTemperature  = 0.7
	DefaultMaxTokens    = 2048
	CostEstimateTimeout = 5 * time.Second
	DefaultProviderName = "mock-plugin"
)

var DefaultTimeout = getAgentEnvDuration("AGENT_TIMEOUT", 300*time.Second)

func getAgentEnvDuration(envKey string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(envKey); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultVal
}

// PluginManagerInterface defines the methods required for plugin management.
// This interface allows for easy testing by enabling mock implementations.
type PluginManagerInterface interface {
	GetProvider(name string) (providerpkg.ModelProvider, error)
	GetProviders() map[string]providerpkg.ModelProvider
}

// Response represents a completed response from the Primary Agent.
type Response struct {
	ID          string
	Content     string
	Model       string
	Provider    string
	Usage       providerpkg.Usage
	ToolCalls   []providerpkg.ToolCall
	ToolResults []ToolExecutionResult
	Timestamp   time.Time
}

// StreamChunk represents a single chunk in a streaming response.
type StreamChunk struct {
	ID        string
	Delta     string
	Done      bool
	Usage     *providerpkg.Usage
	Error     error
	Timestamp time.Time
}

// PrimaryAgent is the main reasoning agent that handles complex queries.
// It integrates with the plugin system to route queries to appropriate model providers.
type PrimaryAgent struct {
	pluginManager     PluginManagerInterface
	defaultProvider   string
	guestProvider     string
	authProvider      string
	timeout           time.Duration
	miniAgent         *MiniDelegationAgent
	maxToolIterations int
	memoryManager     *memory.MemoryManager
	// v0.0.17: Memory Garden & Trace integration
	gardenManager  *memory.GardenManager
	contextBuilder *memory.ContextBuilder
	traceLogger    *trace.TraceLogger
	// v0.0.18: Artifact Engine integration
	projectManager  *projects.ProjectManager
	artifactManager *artifacts.ArtifactManager
	// v0.0.30: Orchestration Engine integration
	orchestrationEngine  *orchestrationpkg.Engine
	orchestrationPlanner orchestrationpkg.PlannerInterface
	useOrchestration     bool
}

// NewPrimaryAgent creates a new PrimaryAgent with the given plugin manager.
// Uses default provider name if routing config is not provided.
func NewPrimaryAgent(pm PluginManagerInterface) *PrimaryAgent {
	return NewPrimaryAgentWithConfig(pm, DefaultProviderName, DefaultProviderName, DefaultProviderName)
}

// NewPrimaryAgentWithConfig creates a new PrimaryAgent with specific routing configuration.
func NewPrimaryAgentWithConfig(pm PluginManagerInterface, defaultProvider, guestProvider, authProvider string) *PrimaryAgent {
	// Fallback to defaults if empty strings provided
	if defaultProvider == "" {
		defaultProvider = DefaultProviderName
	}
	if guestProvider == "" {
		guestProvider = defaultProvider
	}
	if authProvider == "" {
		authProvider = defaultProvider
	}

	return &PrimaryAgent{
		pluginManager:     pm,
		defaultProvider:   defaultProvider,
		guestProvider:     guestProvider,
		authProvider:      authProvider,
		timeout:           DefaultTimeout,
		miniAgent:         NewMiniDelegationAgent(),
		maxToolIterations: 5,
	}
}

// SetDefaultProvider sets the default provider name to use when none is specified.
func (pa *PrimaryAgent) SetDefaultProvider(provider string) {
	pa.defaultProvider = provider
}

// SetTimeout sets the timeout duration for completion requests.
func (pa *PrimaryAgent) SetTimeout(timeout time.Duration) {
	pa.timeout = timeout
}

// SetMemoryManager sets the memory manager for conversation history.
func (pa *PrimaryAgent) SetMemoryManager(mm *memory.MemoryManager) {
	pa.memoryManager = mm
}

// SetGardenManager sets the garden manager for intelligent context management.
// v0.0.17: Memory Garden integration
func (pa *PrimaryAgent) SetGardenManager(gm *memory.GardenManager, cb *memory.ContextBuilder) {
	pa.gardenManager = gm
	pa.contextBuilder = cb
}

// SetTraceLogger sets the trace logger for agent instrumentation.
// v0.0.17: Trace integration
func (pa *PrimaryAgent) SetTraceLogger(tl *trace.TraceLogger) {
	pa.traceLogger = tl
}

// SetProjectManager sets the project manager for project context.
// v0.0.18: Artifact Engine integration
func (pa *PrimaryAgent) SetProjectManager(pm *projects.ProjectManager) {
	pa.projectManager = pm
}

// SetArtifactManager sets the artifact manager for artifact creation and versioning.
// v0.0.18: Artifact Engine integration
func (pa *PrimaryAgent) SetArtifactManager(am *artifacts.ArtifactManager) {
	pa.artifactManager = am
}

// SetOrchestrationEngine sets the orchestration engine for autonomous multi-step workflows.
// v0.0.30: Orchestration Engine integration
func (pa *PrimaryAgent) SetOrchestrationEngine(engine *orchestrationpkg.Engine) {
	pa.orchestrationEngine = engine
}

// SetOrchestrationPlanner sets the orchestration planner for task decomposition.
// v0.0.30: Orchestration Engine integration
func (pa *PrimaryAgent) SetOrchestrationPlanner(planner orchestrationpkg.PlannerInterface) {
	pa.orchestrationPlanner = planner
}

// EnableOrchestration enables or disables autonomous orchestration mode.
// v0.0.30: Orchestration Engine integration
func (pa *PrimaryAgent) EnableOrchestration(enabled bool) {
	pa.useOrchestration = enabled
}

// HandleQuery processes a user query and returns a complete response.
// It routes the query to the specified provider and model.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - query: The user's query text
//   - providerName: Name of the provider to use (empty string uses default)
//   - modelID: Specific model ID to use (empty string uses first available)
//   - userID: User identifier for routing and budget tracking
//
// NOTE: userID parameter is reserved for post-v1 user-based routing implementation.
// Currently unused, but will enable routing between guest (ollama) and
// authenticated users (cloud providers) based on tier and budget.
func (pa *PrimaryAgent) HandleQuery(ctx context.Context, query string, providerName string, modelID string, userID string) (*Response, error) {
	if providerName == "" {
		providerName = pa.defaultProvider
	}

	provider, err := pa.pluginManager.GetProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %w", providerName, err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, pa.timeout)
	defer cancel()

	systemPrompt := os.Getenv("QWEN3_SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = "You are a helpful AI coding assistant."
	}

	messages := []providerpkg.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: query,
		},
	}

	req := &providerpkg.CompletionRequest{
		Messages:    messages,
		Temperature: DefaultTemperature,
		MaxTokens:   DefaultMaxTokens,
		Stream:      false,
	}

	models, err := provider.ListModels(ctxWithTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("provider %s has no available models", providerName)
	}

	if modelID != "" {
		modelFound := false
		for _, m := range models {
			if m.ID == modelID {
				req.Model = modelID
				modelFound = true
				break
			}
		}
		if !modelFound {
			return nil, fmt.Errorf("model %s not found in provider %s", modelID, providerName)
		}
	} else {
		req.Model = models[0].ID
	}

	resp, err := provider.GenerateCompletion(ctxWithTimeout, req)
	if err != nil {
		slog.Error("failed to generate completion",
			"component", "agent_runner",
			"method", "HandleQuery",
			"error", err,
			"query_preview", truncateQuery(query, 200),
			"model", req.Model,
			"provider", providerName,
		)
		return nil, fmt.Errorf("failed to generate completion: %w", err)
	}

	return &Response{
		ID:        resp.ID,
		Content:   resp.Content,
		Model:     resp.Model,
		Provider:  providerName,
		Usage:     resp.Usage,
		ToolCalls: resp.ToolCalls,
		Timestamp: time.Now(),
	}, nil
}

// HandleStreamingQuery processes a user query and returns a streaming response channel.
// It routes the query to the specified provider and model, returning results as they're generated.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - query: The user's query text
//   - providerName: Name of the provider to use (empty string uses default)
//   - modelID: Specific model ID to use (empty string uses first available)
//   - userID: User identifier for routing and budget tracking
//
// Returns a channel of StreamChunk that must be consumed until closed.
//
// NOTE: userID parameter is reserved for post-v1 user-based routing implementation.
func (pa *PrimaryAgent) HandleStreamingQuery(ctx context.Context, query string, providerName string, modelID string, userID string) (<-chan StreamChunk, error) {
	if providerName == "" {
		providerName = pa.defaultProvider
	}

	provider, err := pa.pluginManager.GetProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %w", providerName, err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, pa.timeout)

	systemPrompt := os.Getenv("QWEN3_SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = "You are a helpful AI coding assistant."
	}

	messages := []providerpkg.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: query,
		},
	}

	req := &providerpkg.CompletionRequest{
		Messages:    messages,
		Temperature: DefaultTemperature,
		MaxTokens:   DefaultMaxTokens,
		Stream:      true,
	}

	models, err := provider.ListModels(ctxWithTimeout)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		cancel()
		return nil, fmt.Errorf("provider %s has no available models", providerName)
	}

	if modelID != "" {
		modelFound := false
		for _, m := range models {
			if m.ID == modelID {
				req.Model = modelID
				modelFound = true
				break
			}
		}
		if !modelFound {
			cancel()
			return nil, fmt.Errorf("model %s not found in provider %s", modelID, providerName)
		}
	} else {
		req.Model = models[0].ID
	}

	pluginStream, err := provider.GenerateCompletionStream(ctxWithTimeout, req)
	if err != nil {
		cancel()
		slog.Error("failed to start streaming completion",
			"component", "agent_runner",
			"method", "HandleStreamingQuery",
			"error", err,
			"query_preview", truncateQuery(query, 200),
			"model", req.Model,
			"provider", providerName,
		)
		return nil, fmt.Errorf("failed to start streaming completion: %w", err)
	}

	outStream := make(chan StreamChunk)

	go func() {
		defer close(outStream)
		defer cancel()

		var totalUsage providerpkg.Usage
		streamID := fmt.Sprintf("stream-%d", time.Now().UnixNano())

		for chunk := range pluginStream {
			outStream <- StreamChunk{
				ID:        streamID,
				Delta:     chunk.Delta,
				Done:      chunk.Done,
				Timestamp: time.Now(),
			}

			if chunk.Done {
				outStream <- StreamChunk{
					ID:        streamID,
					Delta:     "",
					Done:      true,
					Usage:     &totalUsage,
					Timestamp: time.Now(),
				}
				break
			}
		}
	}()

	return outStream, nil
}

// GetCostEstimate calculates the estimated cost for a query with the given token counts.
// Returns 0 for providers with free models (cost == 0).
//
// TODO(post-v1): Enhance to support per-model cost estimation instead of using first model.
func (pa *PrimaryAgent) GetCostEstimate(ctx context.Context, providerName string, inputTokens, outputTokens int) (float64, error) {
	provider, err := pa.pluginManager.GetProvider(providerName)
	if err != nil {
		return 0, fmt.Errorf("failed to get provider %s: %w", providerName, err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, CostEstimateTimeout)
	defer cancel()

	models, err := provider.ListModels(ctxWithTimeout)
	if err != nil {
		return 0, fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		return 0, nil
	}

	costPerToken := models[0].Cost
	totalTokens := float64(inputTokens + outputTokens)

	return costPerToken * totalTokens, nil
}

// ListAvailableProviders returns a list of all available provider names.
func (pa *PrimaryAgent) ListAvailableProviders() []string {
	providers := pa.pluginManager.GetProviders()
	names := make([]string, 0, len(providers))

	for name := range providers {
		names = append(names, name)
	}

	return names
}

// selectProvider chooses the appropriate provider based on user tier and intent.
func (pa *PrimaryAgent) selectProvider(userTier string, intent Intent) string {
	switch userTier {
	case "guest":
		return pa.guestProvider
	case "authenticated", "premium":
		return pa.authProvider
	default:
		return pa.defaultProvider
	}
}

// SelectProvider exposes the provider selection logic for external use.
func (pa *PrimaryAgent) SelectProvider(userTier string, intent Intent) string {
	return pa.selectProvider(userTier, intent)
}

// ClassifyIntent exposes the mini agent's intent classification for external use.
func (pa *PrimaryAgent) ClassifyIntent(ctx context.Context, query string) (Intent, float64) {
	if pa.miniAgent == nil {
		return IntentGeneral, 0.0
	}
	return pa.miniAgent.ClassifyIntent(ctx, query)
}

// buildSystemPrompt creates an intent-specific system prompt.
func (pa *PrimaryAgent) buildSystemPrompt(intent Intent) string {
	basePrompt := "You are a helpful AI coding assistant."

	envPrompt := os.Getenv("QWEN3_SYSTEM_PROMPT")
	if envPrompt != "" {
		basePrompt = envPrompt
	}

	switch intent {
	case IntentThink:
		return basePrompt + "\n\nFocus on deep analysis, reasoning, and explaining complex concepts. Take time to think through problems systematically."
	case IntentSearch:
		return basePrompt + "\n\nFocus on finding and retrieving accurate information. Use search tools to gather comprehensive data before answering."
	case IntentBuild:
		return basePrompt + "\n\nFocus on generating clean, efficient, and well-documented code. Follow best practices and established patterns."
	case IntentDebug:
		return basePrompt + "\n\nFocus on identifying issues, validating code, and providing clear solutions. Be thorough in error analysis."
	case IntentGeneral:
		return basePrompt
	default:
		return basePrompt
	}
}

// convertToolsToPluginFormat converts tool definitions to plugin Tool format.
func convertToolsToPluginFormat(toolDefs []*tools.ToolDefinition) []providerpkg.Tool {
	pluginTools := make([]providerpkg.Tool, len(toolDefs))
	for i, td := range toolDefs {
		pluginTools[i] = providerpkg.Tool{
			Name:        td.Name,
			Description: td.Description,
			Parameters:  td.Parameters,
		}
	}
	return pluginTools
}

// ToolExecutionResult represents the result of executing a single tool call.
type ToolExecutionResult struct {
	ToolCallID string
	ToolName   string
	Result     map[string]interface{}
	Error      error
}

// executeToolCalls executes multiple tool calls in parallel.
func (pa *PrimaryAgent) executeToolCalls(ctx context.Context, toolCalls []providerpkg.ToolCall) []ToolExecutionResult {
	results := make([]ToolExecutionResult, len(toolCalls))
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, call providerpkg.ToolCall) {
			defer wg.Done()

			result, err := tools.InvokeTool(ctx, call.Name, call.Arguments)
			results[idx] = ToolExecutionResult{
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Result:     result,
				Error:      err,
			}
		}(i, tc)
	}

	wg.Wait()
	return results
}

// toolResultMaxChars caps tool result content to this many characters before
// injecting it into the message history. Long file reads or command outputs
// grow the context window fast — at 4 chars/token this is ~1K tokens per result.
const toolResultMaxChars = 4096

// formatToolResults formats tool execution results as JSON strings for the model.
// Results are truncated to toolResultMaxChars to prevent unbounded context growth
// across multiple tool-calling iterations.
func formatToolResults(results []ToolExecutionResult) []providerpkg.Message {
	messages := make([]providerpkg.Message, len(results))

	for i, result := range results {
		var content string
		if result.Error != nil {
			content = fmt.Sprintf(`{"error": "%s"}`, result.Error.Error())
		} else {
			jsonData, err := json.Marshal(result.Result)
			if err != nil {
				content = fmt.Sprintf(`{"error": "failed to marshal result: %s"}`, err.Error())
			} else {
				content = string(jsonData)
			}
		}

		// Truncate to prevent runaway context growth across iterations.
		if len(content) > toolResultMaxChars {
			content = content[:toolResultMaxChars] + `... [truncated]`
		}

		messages[i] = providerpkg.Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: result.ToolCallID,
		}
	}

	return messages
}

// QueryRequest represents a structured query request with all parameters.
type QueryRequest struct {
	Query         string
	UserID        string
	UserTier      string
	ProviderName  string
	ModelID       string
	Temperature   float64
	MaxTokens     int
	UseMemory     bool
	ProjectID     string // v0.0.18: Project context for scoped tools and memory
	WorkspaceRoot string // Absolute path to the user's workspace; file tools resolve relative paths against this
}

// ConversationMemory represents a stored conversation turn.
type ConversationMemory struct {
	UserMessage      string    `json:"user_message"`
	AssistantMessage string    `json:"assistant_message"`
	Timestamp        time.Time `json:"timestamp"`
	Model            string    `json:"model"`
	Provider         string    `json:"provider"`
}

// buildMessagesWithContext builds a message array that includes conversation history
// from memory manager if available and UseMemory is enabled.
func (pa *PrimaryAgent) buildMessagesWithContext(ctx context.Context, systemPrompt string, query string, userID string, useMemory bool) ([]providerpkg.Message, error) {
	messages := []providerpkg.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	if !useMemory || pa.memoryManager == nil || userID == "" {
		messages = append(messages, providerpkg.Message{
			Role:    "user",
			Content: query,
		})
		return messages, nil
	}

	memories, err := pa.memoryManager.SearchByType(ctx, fmt.Sprintf("conversation:%s", userID), 5)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conversation history: %w", err)
	}

	for _, mem := range memories {
		var convMem ConversationMemory
		if err := json.Unmarshal([]byte(mem.Content), &convMem); err != nil {
			continue
		}

		messages = append(messages, providerpkg.Message{
			Role:    "user",
			Content: convMem.UserMessage,
		})
		messages = append(messages, providerpkg.Message{
			Role:    "assistant",
			Content: convMem.AssistantMessage,
		})
	}

	messages = append(messages, providerpkg.Message{
		Role:    "user",
		Content: query,
	})

	return messages, nil
}

// buildContextWithGarden builds context using the Memory Garden if available, otherwise falls back to buildMessagesWithContext
func (pa *PrimaryAgent) buildContextWithGarden(ctx context.Context, systemPrompt string, query string, userID string, useMemory bool) ([]providerpkg.Message, map[memory.ContextTier]int, error) {
	if pa.gardenManager != nil && pa.contextBuilder != nil && useMemory {
		sessionID := userID
		if sessionID == "" {
			sessionID = "default"
		}

		buildResult, err := pa.contextBuilder.BuildContext(ctx, query, sessionID, systemPrompt)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build context with garden manager: %w", err)
		}
		// Convert memory.Message to provider.Message
		provMessages := make([]providerpkg.Message, len(buildResult.Messages))
		for i, m := range buildResult.Messages {
			provMessages[i] = providerpkg.Message{
				Role:    m.Role,
				Content: m.Content,
			}
		}
		return provMessages, buildResult.TiersUsed, nil
	}

	messages, err := pa.buildMessagesWithContext(ctx, systemPrompt, query, userID, useMemory)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build messages with context: %w", err)
	}
	return messages, nil, nil
}

// generateID generates a random unique identifier.
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// storeConversation stores a conversation turn in memory.
func (pa *PrimaryAgent) storeConversation(ctx context.Context, userID string, query string, response *Response) error {
	if pa.memoryManager == nil || userID == "" {
		return nil
	}

	convMem := ConversationMemory{
		UserMessage:      query,
		AssistantMessage: response.Content,
		Timestamp:        response.Timestamp,
		Model:            response.Model,
		Provider:         response.Provider,
	}

	contentJSON, err := json.Marshal(convMem)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	mem := memory.Memory{
		ID:      generateID(),
		Type:    fmt.Sprintf("conversation:%s", userID),
		Content: string(contentJSON),
		Metadata: map[string]interface{}{
			"user_id":  userID,
			"model":    response.Model,
			"provider": response.Provider,
		},
		CreatedAt: response.Timestamp,
		UpdatedAt: response.Timestamp,
	}

	return pa.memoryManager.Store(ctx, mem)
}

// HandleQueryWithTools processes a query with full tool-calling support.
// This is the main method that orchestrates intent classification, tool execution,
// and iterative model interactions.
func (pa *PrimaryAgent) HandleQueryWithTools(ctx context.Context, req QueryRequest) (*Response, error) {
	var rootSpan *trace.Span
	var traceID string

	if pa.traceLogger != nil {
		sessionID := req.UserID
		if sessionID == "" {
			sessionID = "default"
		}

		tid, err := pa.traceLogger.StartTrace(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to start trace: %w", err)
		}
		traceID = tid

		rootSpan, err = pa.traceLogger.StartSpan(ctx, traceID, "HandleQueryWithTools", map[string]interface{}{
			"query":     req.Query,
			"user_id":   req.UserID,
			"user_tier": req.UserTier,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start root span: %w", err)
		}
		ctx = trace.WithSpan(ctx, rootSpan)

		defer func() {
			if rootSpan != nil {
				if err := pa.traceLogger.EndTrace(ctx, traceID, "completed"); err == nil {
					pa.traceLogger.EndSpan(ctx, rootSpan, nil)
				}
			}
		}()
	}

	// Step 1: Fast intent classification via Mini Agent
	var intent Intent
	var confidence float64

	if pa.traceLogger != nil {
		classifySpan, err := pa.traceLogger.StartSpan(ctx, traceID, "intent_classification", map[string]interface{}{
			"query": req.Query,
		})
		if err == nil {
			ctx = trace.WithSpan(ctx, classifySpan)
			intent, confidence = pa.miniAgent.ClassifyIntent(ctx, req.Query)
			pa.traceLogger.EndSpan(ctx, classifySpan, map[string]interface{}{
				"intent":           string(intent),
				"confidence":       confidence,
				"confidence_level": getConfidenceLevel(confidence),
			})
			ctx = trace.WithSpan(ctx, rootSpan)
		}
	} else {
		intent, confidence = pa.miniAgent.ClassifyIntent(ctx, req.Query)
	}

	// v0.0.30: Check if query should use orchestration
	// Route complex multi-step queries to orchestration engine if enabled
	if pa.useOrchestration && pa.shouldUseOrchestration(intent, confidence, req.Query, req.UserTier) {
		if pa.traceLogger != nil && rootSpan != nil {
			rootSpan.AddMetadata("routing_decision", "orchestration")
			rootSpan.AddMetadata("orchestration_reason", pa.getOrchestrationReason(intent, confidence, req.Query))
			pa.traceLogger.EndSpan(ctx, rootSpan, nil)
		}
		return pa.HandleQueryWithOrchestration(ctx, req)
	}

	// Step 2: Select appropriate provider based on intent and user tier
	var providerName string

	if pa.traceLogger != nil {
		providerSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "provider_selection", map[string]interface{}{
			"intent":             intent,
			"user_tier":          req.UserTier,
			"requested_provider": req.ProviderName,
		})
		if err == nil {
			ctx = trace.WithSpan(ctx, providerSpan)

			providerName = req.ProviderName
			if providerName == "" {
				providerName = pa.selectProvider(req.UserTier, intent)
			}

			pa.traceLogger.EndSpan(ctx, providerSpan, map[string]interface{}{
				"selected_provider": providerName,
			})
			ctx = trace.WithSpan(ctx, rootSpan)
		}
	} else {
		providerName = req.ProviderName
		if providerName == "" {
			providerName = pa.selectProvider(req.UserTier, intent)
		}
	}

	// Step 3: Get model provider plugin
	provider, err := pa.pluginManager.GetProvider(providerName)
	if err != nil {
		if pa.traceLogger != nil && rootSpan != nil {
			pa.traceLogger.FailSpan(ctx, rootSpan, fmt.Sprintf("failed to get provider: %v", err))
		}
		return nil, fmt.Errorf("failed to get provider %s: %w", providerName, err)
	}

	// Step 4: Get available models
	ctxWithTimeout, cancel := context.WithTimeout(ctx, pa.timeout)
	defer cancel()

	models, err := provider.ListModels(ctxWithTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("provider %s has no available models", providerName)
	}

	// Select model
	modelID := req.ModelID
	if modelID == "" {
		modelID = models[0].ID
	} else {
		modelFound := false
		for _, m := range models {
			if m.ID == modelID {
				modelFound = true
				break
			}
		}
		if !modelFound {
			return nil, fmt.Errorf("model %s not found in provider %s", modelID, providerName)
		}
	}

	// Step 5: Build messages with intent-based system prompt and conversation history
	systemPrompt := pa.buildSystemPrompt(intent)
	// Append workspace context so the model uses correct paths with file tools.
	if req.WorkspaceRoot != "" {
		systemPrompt += fmt.Sprintf("\n\nWorkspace root: %s\nWhen using file tools, use paths relative to this workspace root or absolute paths.", req.WorkspaceRoot)
	}

	var messages []providerpkg.Message

	if pa.traceLogger != nil {
		contextSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "context_building", map[string]interface{}{
			"use_memory": req.UseMemory,
		})
		if err == nil {
			ctx = trace.WithSpan(ctx, contextSpan)

			var tiersUsed map[memory.ContextTier]int
			messages, tiersUsed, err = pa.buildContextWithGarden(ctxWithTimeout, systemPrompt, req.Query, req.UserID, req.UseMemory)
			if err != nil {
				pa.traceLogger.FailSpan(ctx, contextSpan, fmt.Sprintf("failed to build context: %v", err))
				ctx = trace.WithSpan(ctx, rootSpan)
				return nil, err
			}

			spanMetadata := map[string]interface{}{
				"message_count": len(messages),
			}
			if tiersUsed != nil {
				spanMetadata["tiers_used"] = tiersUsed
			}
			pa.traceLogger.EndSpan(ctx, contextSpan, spanMetadata)

			ctx = trace.WithSpan(ctx, rootSpan)
		}
	} else {
		var err error
		messages, _, err = pa.buildContextWithGarden(ctxWithTimeout, systemPrompt, req.Query, req.UserID, req.UseMemory)
		if err != nil {
			return nil, err
		}
	}

	// v0.0.18: Add project_id to context for thread-safe project scoping
	// Using context.Context ensures concurrent requests don't interfere with each other
	if req.ProjectID != "" {
		ctxWithTimeout = tools.WithProjectID(ctxWithTimeout, req.ProjectID)
	}
	// Inject workspace root so file tools can resolve relative paths correctly.
	// The CLI sends os.Getwd() from the user's session; without it the gateway
	// would resolve relative paths against its own process CWD.
	if req.WorkspaceRoot != "" {
		ctxWithTimeout = tools.WithWorkspaceRoot(ctxWithTimeout, req.WorkspaceRoot)
	}

	// Step 6: Get available tools filtered by intent.
	// Sending only relevant tools reduces input token count per call, which is
	// critical for staying under provider TPM rate limits. The full tool set (~33)
	// is only sent when intent is unrecognised.
	availableTools := tools.GetToolsForIntent(string(intent))
	pluginTools := convertToolsToPluginFormat(availableTools)

	// Set defaults for temperature and max tokens
	temperature := req.Temperature
	if temperature == 0 {
		temperature = DefaultTemperature
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultMaxTokens
	}

	// Step 7: Iterative tool calling loop
	var finalResponse *providerpkg.CompletionResponse
	var allToolCalls []providerpkg.ToolCall
	var allToolResults []ToolExecutionResult

	for iteration := 0; iteration < pa.maxToolIterations; iteration++ {
		// On the first iteration, force a tool call so the model acts rather than
		// just writing a plan. On subsequent iterations, let it decide — it may
		// want to return a final text response once tool results are available.
		toolChoice := "auto"
		if iteration == 0 && len(pluginTools) > 0 {
			toolChoice = "required"
		}

		completionReq := &providerpkg.CompletionRequest{
			Model:       modelID,
			Messages:    messages,
			Temperature: temperature,
			MaxTokens:   maxTokens,
			Tools:       pluginTools,
			Stream:      false,
			ToolChoice:  toolChoice,
		}

		var response *providerpkg.CompletionResponse

		if pa.traceLogger != nil {
			modelSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "model_invocation", map[string]interface{}{
				"model":     modelID,
				"provider":  providerName,
				"iteration": iteration,
			})
			if err == nil {
				ctx = trace.WithSpan(ctx, modelSpan)
				response, err = provider.GenerateCompletion(ctxWithTimeout, completionReq)
				if err != nil {
					pa.traceLogger.FailSpan(ctx, modelSpan, fmt.Sprintf("completion failed: %v", err))
					ctx = trace.WithSpan(ctx, rootSpan)
					slog.Error("completion failed",
						"component", "agent_runner",
						"method", "HandleQueryWithTools",
						"error", err,
						"query_preview", truncateQuery(req.Query, 200),
						"model", modelID,
						"provider", providerName,
						"iteration", iteration,
					)
					return nil, fmt.Errorf("completion failed: %w", err)
				}

				pa.traceLogger.EndSpan(ctx, modelSpan, map[string]interface{}{
					"input_tokens":  response.Usage.InputTokens,
					"output_tokens": response.Usage.OutputTokens,
					"tool_calls":    len(response.ToolCalls),
				})
				ctx = trace.WithSpan(ctx, rootSpan)
			}
		} else {
			var err error
			response, err = provider.GenerateCompletion(ctxWithTimeout, completionReq)
			if err != nil {
				slog.Error("completion failed",
					"component", "agent_runner",
					"method", "HandleQueryWithTools",
					"error", err,
					"query_preview", truncateQuery(req.Query, 200),
					"model", modelID,
					"provider", providerName,
					"iteration", iteration,
				)
				return nil, fmt.Errorf("completion failed: %w", err)
			}
		}

		// If no tool calls, we're done
		if len(response.ToolCalls) == 0 {
			finalResponse = response
			break
		}

		// Execute tool calls in parallel
		allToolCalls = append(allToolCalls, response.ToolCalls...)

		var toolResults []ToolExecutionResult

		// v0.0.17: Trace tool execution
		if pa.traceLogger != nil {
			toolExecSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "tool_execution", map[string]interface{}{
				"tool_call_count": len(response.ToolCalls),
			})
			if err == nil {
				ctx = trace.WithSpan(ctx, toolExecSpan)
				toolResults = pa.executeToolCalls(ctxWithTimeout, response.ToolCalls)

				successCount := 0
				for _, result := range toolResults {
					if result.Error == nil {
						successCount++
					}
				}

				pa.traceLogger.EndSpan(ctx, toolExecSpan, map[string]interface{}{
					"success_count": successCount,
					"total_count":   len(toolResults),
				})
				ctx = trace.WithSpan(ctx, rootSpan)
			}
		} else {
			toolResults = pa.executeToolCalls(ctxWithTimeout, response.ToolCalls)
		}

		// v0.0.18: Accumulate tool results
		allToolResults = append(allToolResults, toolResults...)

		// Add assistant message with tool calls
		messages = append(messages, providerpkg.Message{
			Role:      "assistant",
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})

		// Add tool results as messages
		toolMessages := formatToolResults(toolResults)
		messages = append(messages, toolMessages...)

		// If this is the last iteration, get final response without tools
		if iteration == pa.maxToolIterations-1 {
			completionReq.Tools = []providerpkg.Tool{}

			if pa.traceLogger != nil {
				finalSpan, err := pa.traceLogger.StartSpan(ctx, traceID, "final_model_invocation", map[string]interface{}{
					"model":    modelID,
					"provider": providerName,
				})
				if err == nil {
					ctx = trace.WithSpan(ctx, finalSpan)
					finalResponse, err = provider.GenerateCompletion(ctxWithTimeout, completionReq)
					if err != nil {
						pa.traceLogger.FailSpan(ctx, finalSpan, fmt.Sprintf("final completion failed: %v", err))
						ctx = trace.WithSpan(ctx, rootSpan)
						return nil, fmt.Errorf("final completion failed: %w", err)
					}

					pa.traceLogger.EndSpan(ctx, finalSpan, map[string]interface{}{
						"input_tokens":  finalResponse.Usage.InputTokens,
						"output_tokens": finalResponse.Usage.OutputTokens,
					})
					ctx = trace.WithSpan(ctx, rootSpan)
				}
			} else {
				var err error
				finalResponse, err = provider.GenerateCompletion(ctxWithTimeout, completionReq)
				if err != nil {
					return nil, fmt.Errorf("final completion failed: %w", err)
				}
			}
			break
		}
	}

	if finalResponse == nil {
		if pa.traceLogger != nil && rootSpan != nil {
			pa.traceLogger.FailSpan(ctx, rootSpan, "no response generated")
		}
		return nil, fmt.Errorf("no response generated after %d iterations", pa.maxToolIterations)
	}

	response := &Response{
		ID:          finalResponse.ID,
		Content:     finalResponse.Content,
		Model:       finalResponse.Model,
		Provider:    providerName,
		Usage:       finalResponse.Usage,
		ToolCalls:   allToolCalls,
		ToolResults: allToolResults,
		Timestamp:   time.Now(),
	}

	if req.UseMemory {
		if pa.traceLogger != nil {
			memorySpan, err := pa.traceLogger.StartSpan(ctx, traceID, "memory_storage", map[string]interface{}{
				"user_id": req.UserID,
			})
			if err == nil {
				ctx = trace.WithSpan(ctx, memorySpan)
				if err := pa.storeConversation(ctxWithTimeout, req.UserID, req.Query, response); err != nil {
					pa.traceLogger.FailSpan(ctx, memorySpan, fmt.Sprintf("failed to store conversation: %v", err))
					ctx = trace.WithSpan(ctx, rootSpan)
					return response, fmt.Errorf("warning: failed to store conversation: %w", err)
				}
				pa.traceLogger.EndSpan(ctx, memorySpan, nil)
				ctx = trace.WithSpan(ctx, rootSpan)
			}
		} else {
			if err := pa.storeConversation(ctxWithTimeout, req.UserID, req.Query, response); err != nil {
				return response, fmt.Errorf("warning: failed to store conversation: %w", err)
			}
		}
	}

	if pa.traceLogger != nil && rootSpan != nil {
		rootSpan.AddMetadata("total_tool_calls", len(allToolCalls))
		rootSpan.AddMetadata("total_tokens", response.Usage.InputTokens+response.Usage.OutputTokens)
	}

	return response, nil
}

// HandleQueryWithOrchestration processes a query using the orchestration engine for autonomous multi-step workflows.
// v0.0.30: Orchestration Engine integration
func (pa *PrimaryAgent) HandleQueryWithOrchestration(ctx context.Context, req QueryRequest) (*Response, error) {
	if pa.orchestrationPlanner == nil || pa.orchestrationEngine == nil {
		return nil, fmt.Errorf("orchestration components not initialized")
	}

	var rootSpan *trace.Span
	var traceID string

	if pa.traceLogger != nil {
		sessionID := req.UserID
		if sessionID == "" {
			sessionID = "default"
		}

		tid, err := pa.traceLogger.StartTrace(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to start trace: %w", err)
		}
		traceID = tid

		rootSpan, err = pa.traceLogger.StartSpan(ctx, traceID, "HandleQueryWithOrchestration", map[string]interface{}{
			"query":     req.Query,
			"user_id":   req.UserID,
			"user_tier": req.UserTier,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start root span: %w", err)
		}
		ctx = trace.WithSpan(ctx, rootSpan)

		defer func() {
			if rootSpan != nil {
				if err := pa.traceLogger.EndTrace(ctx, traceID, "completed"); err == nil {
					pa.traceLogger.EndSpan(ctx, rootSpan, nil)
				}
			}
		}()
	}

	task := orchestrationpkg.NewTask(req.UserID, req.Query)

	// Set project context before plan generation so planner can use it
	if req.ProjectID != "" {
		ctx = tools.WithProjectID(ctx, req.ProjectID)
	}

	var plan *orchestrationpkg.Plan
	var err error

	if pa.traceLogger != nil {
		planSpan, spanErr := pa.traceLogger.StartSpan(ctx, traceID, "plan_generation", map[string]interface{}{
			"task_id":    task.ID,
			"project_id": req.ProjectID,
		})
		if spanErr == nil {
			ctx = trace.WithSpan(ctx, planSpan)
			plan, err = pa.orchestrationPlanner.GeneratePlan(ctx, task)
			if err != nil {
				pa.traceLogger.FailSpan(ctx, planSpan, fmt.Sprintf("failed to generate plan: %v", err))
				ctx = trace.WithSpan(ctx, rootSpan)
				return nil, fmt.Errorf("failed to generate plan: %w", err)
			}
			pa.traceLogger.EndSpan(ctx, planSpan, map[string]interface{}{
				"plan_id":    plan.ID,
				"node_count": len(plan.Nodes),
			})
			ctx = trace.WithSpan(ctx, rootSpan)
		}
	} else {
		plan, err = pa.orchestrationPlanner.GeneratePlan(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("failed to generate plan: %w", err)
		}
	}

	// Create separate timeout context for execution (5x default timeout for complex workflows)
	executionCtx, executionCancel := context.WithTimeout(ctx, pa.timeout*5)
	defer executionCancel()

	if err := pa.orchestrationEngine.Execute(executionCtx, plan, task, req.UserID); err != nil {
		if pa.traceLogger != nil && rootSpan != nil {
			pa.traceLogger.FailSpan(ctx, rootSpan, fmt.Sprintf("orchestration execution failed: %v", err))
		}
		return nil, fmt.Errorf("orchestration execution failed: %w", err)
	}

	response := pa.buildResponseFromPlan(plan, task)

	if req.UseMemory {
		// Create separate timeout context for memory storage (standard timeout)
		memoryCtx, memoryCancel := context.WithTimeout(ctx, pa.timeout)
		defer memoryCancel()

		if pa.traceLogger != nil {
			memorySpan, spanErr := pa.traceLogger.StartSpan(ctx, traceID, "memory_storage", map[string]interface{}{
				"user_id": req.UserID,
			})
			if spanErr == nil {
				ctx = trace.WithSpan(ctx, memorySpan)
				if err := pa.storeConversation(memoryCtx, req.UserID, req.Query, response); err != nil {
					// Log the error but don't fail the request since orchestration succeeded
					pa.traceLogger.FailSpan(ctx, memorySpan, fmt.Sprintf("failed to store conversation: %v", err))
					slog.Warn("orchestration succeeded but failed to store conversation", "user_id", req.UserID, "error", err)
					ctx = trace.WithSpan(ctx, rootSpan)
				} else {
					pa.traceLogger.EndSpan(ctx, memorySpan, nil)
					ctx = trace.WithSpan(ctx, rootSpan)
				}
			}
		} else {
			if err := pa.storeConversation(memoryCtx, req.UserID, req.Query, response); err != nil {
				// Log the error but don't fail the request since orchestration succeeded
				slog.Warn("orchestration succeeded but failed to store conversation", "user_id", req.UserID, "error", err)
			}
		}
	}

	if pa.traceLogger != nil && rootSpan != nil {
		rootSpan.AddMetadata("completed_nodes", len(plan.Nodes))
		rootSpan.AddMetadata("plan_version", plan.Version)
	}

	return response, nil
}

// synthesizePlanSummary creates a natural language summary from plan results using LLM.
// v0.0.30: Orchestration Engine integration - sophisticated response synthesis
func (pa *PrimaryAgent) synthesizePlanSummary(ctx context.Context, plan *orchestrationpkg.Plan, task *orchestrationpkg.Task) string {
	// Build context for LLM synthesis
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Original Task: %s\n\n", task.Description))
	contextBuilder.WriteString("Execution Results:\n")

	for i, node := range plan.Nodes {
		contextBuilder.WriteString(fmt.Sprintf("%d. Tool: %s (Status: %s)\n", i+1, node.ToolName, node.State))
		if node.State == orchestrationpkg.NodeStateSuccess && node.Result != nil {
			resultJSON, _ := json.Marshal(node.Result)
			contextBuilder.WriteString(fmt.Sprintf("   Result: %s\n", string(resultJSON)))
		} else if node.State == orchestrationpkg.NodeStateFailed {
			contextBuilder.WriteString(fmt.Sprintf("   Error: %s\n", node.Error))
		}
	}

	// Create synthesis prompt
	synthesisPrompt := fmt.Sprintf(`You are an AI assistant that synthesizes technical execution results into clear, user-friendly summaries.

Given the following task execution details, create a natural language summary that:
1. Directly addresses what the user asked for
2. Highlights key findings and results
3. Mentions any important caveats or failures
4. Is concise and easy to understand

%s

Provide only the summary, without preamble or meta-commentary.`, contextBuilder.String())

	// Attempt to synthesize using default provider
	provider, err := pa.pluginManager.GetProvider(pa.defaultProvider)
	if err != nil {
		return "" // Fallback to technical response
	}

	messages := []providerpkg.Message{
		{Role: "system", Content: "You are a helpful AI assistant that creates clear, concise summaries."},
		{Role: "user", Content: synthesisPrompt},
	}

	req := &providerpkg.CompletionRequest{
		Model:       "",
		Messages:    messages,
		Temperature: 0.3, // Lower temperature for more focused summaries
	}

	// Use a short timeout for synthesis to avoid delays
	synthesisCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := provider.GenerateCompletion(synthesisCtx, req)
	if err != nil {
		return "" // Fallback to technical response
	}

	return resp.Content
}

// buildResponseFromPlan aggregates results from a completed plan into a coherent response.
// v0.0.30: Orchestration Engine integration with LLM-powered synthesis
func (pa *PrimaryAgent) buildResponseFromPlan(plan *orchestrationpkg.Plan, task *orchestrationpkg.Task) *Response {
	var contentBuilder strings.Builder

	successfulNodes := 0
	failedNodes := 0
	for _, node := range plan.Nodes {
		if node.State == orchestrationpkg.NodeStateSuccess {
			successfulNodes++
		} else if node.State == orchestrationpkg.NodeStateFailed {
			failedNodes++
		}
	}

	// Try to generate natural language summary if synthesis is enabled
	enableSynthesis := true // Can be made configurable
	var naturalSummary string

	if enableSynthesis && pa.pluginManager != nil && successfulNodes > 0 {
		naturalSummary = pa.synthesizePlanSummary(context.Background(), plan, task)
	}

	// If we got a natural summary, use it as the primary content
	if naturalSummary != "" {
		contentBuilder.WriteString(naturalSummary)
		contentBuilder.WriteString("\n\n---\n\n")
		contentBuilder.WriteString("## Execution Details\n\n")
	}

	contentBuilder.WriteString(fmt.Sprintf("**Task**: %s\n\n", task.Description))
	contentBuilder.WriteString(fmt.Sprintf("**Plan Execution Summary**:\n"))
	contentBuilder.WriteString(fmt.Sprintf("- Total nodes: %d\n", len(plan.Nodes)))
	contentBuilder.WriteString(fmt.Sprintf("- Successful: %d\n", successfulNodes))
	contentBuilder.WriteString(fmt.Sprintf("- Failed: %d\n\n", failedNodes))

	if reasoning, ok := plan.Metadata["reasoning"].(string); ok && reasoning != "" {
		contentBuilder.WriteString(fmt.Sprintf("**Planning Strategy**:\n%s\n\n", reasoning))
	}

	contentBuilder.WriteString("**Node Results**:\n")
	for i, node := range plan.Nodes {
		contentBuilder.WriteString(fmt.Sprintf("\n%d. %s (%s)\n", i+1, node.ToolName, node.State))
		if node.State == orchestrationpkg.NodeStateSuccess && node.Result != nil {
			resultJSON, err := json.MarshalIndent(node.Result, "   ", "  ")
			if err == nil {
				contentBuilder.WriteString(fmt.Sprintf("   Result: %s\n", string(resultJSON)))
			}
		} else if node.State == orchestrationpkg.NodeStateFailed {
			contentBuilder.WriteString(fmt.Sprintf("   Error: %s\n", node.Error))
		}
		if node.StartTime != nil && node.EndTime != nil {
			duration := node.EndTime.Sub(*node.StartTime)
			contentBuilder.WriteString(fmt.Sprintf("   Duration: %v\n", duration))
		}
	}

	toolCalls := make([]providerpkg.ToolCall, len(plan.Nodes))
	for i, node := range plan.Nodes {
		toolCalls[i] = providerpkg.ToolCall{
			ID:        node.ID,
			Name:      node.ToolName,
			Arguments: node.Parameters,
		}
	}

	toolResults := make([]ToolExecutionResult, len(plan.Nodes))
	for i, node := range plan.Nodes {
		var err error
		if node.Error != "" {
			err = fmt.Errorf("%s", node.Error)
		}
		toolResults[i] = ToolExecutionResult{
			ToolCallID: node.ID,
			ToolName:   node.ToolName,
			Result:     node.Result,
			Error:      err,
		}
	}

	// Aggregate usage from plan metadata if available
	usage := providerpkg.Usage{}
	if planUsage, ok := plan.Metadata["total_usage"].(map[string]interface{}); ok {
		if inputTokens, ok := planUsage["input_tokens"].(float64); ok {
			usage.InputTokens = int(inputTokens)
		}
		if outputTokens, ok := planUsage["output_tokens"].(float64); ok {
			usage.OutputTokens = int(outputTokens)
		}
		if totalTokens, ok := planUsage["total_tokens"].(float64); ok {
			usage.TotalTokens = int(totalTokens)
		}
	}

	// If total_usage not found, try to extract from individual node metadata
	if usage.TotalTokens == 0 {
		for _, node := range plan.Nodes {
			if nodeUsage, ok := node.Result["usage"].(map[string]interface{}); ok {
				if inputTokens, ok := nodeUsage["input_tokens"].(float64); ok {
					usage.InputTokens += int(inputTokens)
				}
				if outputTokens, ok := nodeUsage["output_tokens"].(float64); ok {
					usage.OutputTokens += int(outputTokens)
				}
			}
		}
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}

	return &Response{
		ID:          plan.ID,
		Content:     contentBuilder.String(),
		Model:       "orchestration-engine",
		Provider:    "orchestration",
		Usage:       usage,
		ToolCalls:   toolCalls,
		ToolResults: toolResults,
		Timestamp:   time.Now(),
	}
}

// truncateQuery truncates a query string for safe inclusion in log messages.
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "..."
}

func getConfidenceLevel(confidence float64) string {
	if confidence >= 0.9 {
		return "high"
	} else if confidence >= 0.7 {
		return "medium"
	} else if confidence >= 0.5 {
		return "low"
	}
	return "very_low"
}

// shouldUseOrchestration determines if a query should be routed to the orchestration engine.
// v0.0.30: Orchestration routing logic
func (pa *PrimaryAgent) shouldUseOrchestration(intent Intent, confidence float64, query string, userTier string) bool {
	// Don't use orchestration if components not initialized
	if pa.orchestrationPlanner == nil || pa.orchestrationEngine == nil {
		return false
	}

	// Check for multi-step indicators in the query
	queryLower := strings.ToLower(query)
	multiStepIndicators := []string{
		"and then",
		"then",
		"after that",
		"next",
		"finally",
		"first", "second", "third",
		"step 1", "step 2", "step 3",
		"1.", "2.", "3.",
	}

	hasMultiStep := false
	for _, indicator := range multiStepIndicators {
		if strings.Contains(queryLower, indicator) {
			hasMultiStep = true
			break
		}
	}

	// Check for research + creation pattern
	hasResearch := strings.Contains(queryLower, "research") ||
		strings.Contains(queryLower, "find") ||
		strings.Contains(queryLower, "search for")

	hasCreation := strings.Contains(queryLower, "create") ||
		strings.Contains(queryLower, "generate") ||
		strings.Contains(queryLower, "write") ||
		strings.Contains(queryLower, "make") ||
		strings.Contains(queryLower, "build")

	hasResearchAndCreation := hasResearch && hasCreation

	// Check for multiple tool-like actions
	actionWords := []string{"analyze", "summarize", "extract", "fetch", "download", "compare", "combine"}
	actionCount := 0
	for _, action := range actionWords {
		if strings.Contains(queryLower, action) {
			actionCount++
		}
	}
	hasMultipleActions := actionCount >= 2

	// Route to orchestration if:
	// 1. Query has multi-step indicators
	// 2. Query has research + creation pattern
	// 3. Query has multiple tool-like actions
	// 4. Intent is complex and confidence is high
	if hasMultiStep || hasResearchAndCreation || hasMultipleActions {
		return true
	}

	// For think/build intents with high confidence, use orchestration for long queries
	if (intent == IntentThink || intent == IntentBuild) && confidence >= 0.8 {
		// Check query length as proxy for complexity
		if len(strings.Split(query, " ")) >= 15 {
			return true
		}
	}

	return false
}

// getOrchestrationReason returns a human-readable reason for routing to orchestration.
// v0.0.30: Orchestration routing metadata
func (pa *PrimaryAgent) getOrchestrationReason(intent Intent, confidence float64, query string) string {
	queryLower := strings.ToLower(query)

	if strings.Contains(queryLower, "and then") || strings.Contains(queryLower, "then") {
		return "multi_step_sequence_detected"
	}

	hasResearch := strings.Contains(queryLower, "research") || strings.Contains(queryLower, "find")
	hasCreation := strings.Contains(queryLower, "create") || strings.Contains(queryLower, "generate")
	if hasResearch && hasCreation {
		return "research_and_creation_workflow"
	}

	if strings.Contains(queryLower, "1.") || strings.Contains(queryLower, "step 1") {
		return "numbered_steps_detected"
	}

	actionWords := []string{"analyze", "summarize", "extract", "fetch", "compare"}
	actionCount := 0
	for _, action := range actionWords {
		if strings.Contains(queryLower, action) {
			actionCount++
		}
	}
	if actionCount >= 2 {
		return "multiple_actions_detected"
	}

	if (intent == IntentThink || intent == IntentBuild) && confidence >= 0.8 {
		return "complex_intent_high_confidence"
	}

	return "complex_query_heuristic"
}
