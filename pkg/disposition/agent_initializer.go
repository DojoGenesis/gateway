package disposition

import (
	"context"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

// AgentInitializerImpl implements the gateway.AgentInitializer interface.
// It loads agent disposition configurations from disk and caches them to avoid
// repeated file I/O operations.
type AgentInitializerImpl struct {
	cache *disposition.DispositionCache
}

// NewAgentInitializer creates a new AgentInitializer with a cache of the specified TTL.
// A typical TTL is 5 minutes, which balances freshness with performance.
func NewAgentInitializer(cacheTTL time.Duration) *AgentInitializerImpl {
	return &AgentInitializerImpl{
		cache: disposition.NewDispositionCache(cacheTTL),
	}
}

// Initialize loads the agent configuration for the given workspace and mode.
// It implements the gateway.AgentInitializer interface.
//
// The method first checks the cache for an existing configuration. If not found,
// it calls ResolveDisposition to load from disk, then caches the result.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - workspaceRoot: Absolute path to the workspace directory
//   - activeMode: Optional mode name for applying mode-specific overrides (empty string for base config)
//
// Returns:
//   - *gateway.AgentConfig: The loaded and merged agent configuration
//   - error: An error if initialization fails
func (a *AgentInitializerImpl) Initialize(ctx context.Context, workspaceRoot string, activeMode string) (*gateway.AgentConfig, error) {
	// Check cache first
	cacheKey := makeCacheKey(workspaceRoot, activeMode)
	if cached, found := a.cache.Get(cacheKey); found {
		return convertToAgentConfig(cached), nil
	}

	// Load from disk (note: ResolveDisposition doesn't use context per ADA contract)
	disp, err := disposition.ResolveDisposition(workspaceRoot, activeMode)
	if err != nil {
		return nil, err
	}

	// Cache the result
	a.cache.Set(cacheKey, disp)

	// Convert to gateway.AgentConfig
	return convertToAgentConfig(disp), nil
}

// ClearCache clears all cached disposition configurations.
// This is useful for testing or when you want to force a reload from disk.
func (a *AgentInitializerImpl) ClearCache() {
	a.cache.Clear()
}

// makeCacheKey creates a cache key from workspace root and active mode.
func makeCacheKey(workspaceRoot, activeMode string) string {
	if activeMode == "" {
		return workspaceRoot + ":base"
	}
	return workspaceRoot + ":" + activeMode
}

// convertToAgentConfig converts a DispositionConfig to a gateway.AgentConfig.
// The gateway.AgentConfig is a simplified view used by the gateway's core logic,
// while DispositionConfig is the full ADA configuration.
// This conversion follows the Gateway-ADA Contract v1.0.0.
func convertToAgentConfig(disp *disposition.DispositionConfig) *gateway.AgentConfig {
	return &gateway.AgentConfig{
		// Core behavioral dimensions
		Pacing:     disp.Pacing,
		Depth:      disp.Depth,
		Tone:       disp.Tone,
		Initiative: disp.Initiative,

		// Nested configurations
		Validation: gateway.ValidationConfig{
			Strategy:     disp.Validation.Strategy,
			RequireTests: disp.Validation.RequireTests,
			RequireDocs:  disp.Validation.RequireDocs,
		},
		ErrorHandling: gateway.ErrorHandlingConfig{
			Strategy:   disp.ErrorHandling.Strategy,
			RetryCount: disp.ErrorHandling.RetryCount,
		},
		Collaboration: gateway.CollaborationConfig{
			Style:            disp.Collaboration.Style,
			CheckInFrequency: disp.Collaboration.CheckInFrequency,
		},
		Reflection: gateway.ReflectionConfig{
			Frequency: disp.Reflection.Frequency,
			Format:    disp.Reflection.Format,
			Triggers:  disp.Reflection.Triggers,
		},
	}
}
