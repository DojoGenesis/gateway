package disposition

import (
	"context"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

// AgentInitializerImpl implements the gateway.AgentInitializer interface.
// It loads agent disposition configurations from disk and caches them to avoid
// repeated file I/O operations.
type AgentInitializerImpl struct {
	cache *DispositionCache
}

// NewAgentInitializer creates a new AgentInitializer with a cache of the specified TTL.
// A typical TTL is 5 minutes, which balances freshness with performance.
func NewAgentInitializer(cacheTTL time.Duration) *AgentInitializerImpl {
	return &AgentInitializerImpl{
		cache: NewDispositionCache(cacheTTL),
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
	disposition, err := ResolveDisposition(workspaceRoot, activeMode)
	if err != nil {
		return nil, err
	}

	// Cache the result
	a.cache.Set(cacheKey, disposition)

	// Convert to gateway.AgentConfig
	return convertToAgentConfig(disposition), nil
}

// ClearCache clears all cached disposition configurations.
// This is useful for testing or when you want to force a reload from disk.
func (a *AgentInitializerImpl) ClearCache() {
	a.cache.Clear()
}

// convertToAgentConfig converts a DispositionConfig to a gateway.AgentConfig.
// The gateway.AgentConfig is a simplified view used by the gateway's core logic,
// while DispositionConfig is the full ADA configuration.
// This conversion follows the Gateway-ADA Contract v1.0.0.
func convertToAgentConfig(disposition *DispositionConfig) *gateway.AgentConfig {
	return &gateway.AgentConfig{
		// Core behavioral dimensions
		Pacing:     disposition.Pacing,
		Depth:      disposition.Depth,
		Tone:       disposition.Tone,
		Initiative: disposition.Initiative,

		// Nested configurations
		Validation: gateway.ValidationConfig{
			Strategy:     disposition.Validation.Strategy,
			RequireTests: disposition.Validation.RequireTests,
			RequireDocs:  disposition.Validation.RequireDocs,
		},
		ErrorHandling: gateway.ErrorHandlingConfig{
			Strategy:   disposition.ErrorHandling.Strategy,
			RetryCount: disposition.ErrorHandling.RetryCount,
		},
		Collaboration: gateway.CollaborationConfig{
			Style:            disposition.Collaboration.Style,
			CheckInFrequency: disposition.Collaboration.CheckInFrequency,
		},
		Reflection: gateway.ReflectionConfig{
			Frequency: disposition.Reflection.Frequency,
			Format:    disposition.Reflection.Format,
			Triggers:  disposition.Reflection.Triggers,
		},
	}
}
