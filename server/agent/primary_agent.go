package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/DojoGenesis/gateway/memory"
	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	providerpkg "github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/artifacts"
	"github.com/DojoGenesis/gateway/server/projects"
	"github.com/DojoGenesis/gateway/server/trace"
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

	systemPrompt := resolveBaseSystemPrompt()

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

	systemPrompt := resolveBaseSystemPrompt()

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
