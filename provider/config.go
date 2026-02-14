package provider

import "time"

// ProviderConfig defines the configuration for a single model provider plugin.
type ProviderConfig struct {
	Name       string                 // Provider name (e.g., "openai")
	Enabled    bool                   // Whether to load this provider
	Priority   int                    // Load priority (lower = earlier)
	PluginPath string                 // Path to plugin binary (relative or absolute)
	Config     map[string]interface{} // Provider-specific config (API keys, etc.)
}

// PluginManagerConfig defines the configuration for the PluginManager.
type PluginManagerConfig struct {
	PluginDir          string           // Directory containing plugin binaries
	Providers          []ProviderConfig // Explicit provider configs (optional)
	MonitorInterval    time.Duration    // Health check interval (default 5s)
	RestartDelay       time.Duration    // Delay before restarting crashed plugin (default 1s)
	MaxRestartAttempts int              // Max restarts before giving up (default 3)
}

// RoutingConfig defines how requests are routed to providers.
type RoutingConfig struct {
	DefaultProvider  string            // Provider name for default requests
	GuestProvider    string            // Provider name for unauthenticated requests
	AuthProviders    map[string]string // userID -> provider name mappings
	FallbackProvider string            // If routing fails, use this
}
