// Package disposition provides agent disposition configuration management for the Agentic Gateway.
//
// The disposition package implements the ADA (Agent Disposition Architecture) contract,
// which defines how agents are configured, initialized, and adapted at runtime. It provides:
//
//   - Configuration parsing from YAML files (agent.yaml)
//   - Mode-based configuration overrides (debug, prod, etc.)
//   - Validation of configuration constraints
//   - TTL-based caching for performance
//   - Implementation of the gateway.AgentInitializer interface
//
// # Configuration Structure
//
// A disposition configuration defines an agent's behavior across multiple dimensions:
//
//   - Validation: Output quality and schema constraints
//   - Error Handling: Retry policies, timeouts, and circuit breakers
//   - Collaboration: Multi-agent coordination settings
//   - Reflection: Self-improvement and learning capabilities
//   - Pacing: Execution throttling and rate limiting
//   - Depth: Memory compression and retention policies
//
// # File Resolution
//
// The package discovers agent.yaml files using the following priority order:
//
//  1. AGENT_DISPOSITION_PATH environment variable (if set)
//  2. {agent_id}.agent.yaml in workspace root (if .agent_bridge file exists)
//  3. agent.yaml in workspace root
//  4. Default disposition (if no file found)
//
// # Mode Overrides
//
// Modes allow runtime adaptation of agent behavior. For example, a "debug" mode might:
//
//   - Increase error verbosity
//   - Reduce retry limits for faster failure
//   - Enable additional validation checks
//
// Mode overrides are defined in the YAML under a "modes" section and are merged
// with the base configuration at initialization time.
//
// # Example Usage
//
//	// Create an agent initializer with 5-minute cache TTL
//	initializer := disposition.NewAgentInitializer(5 * time.Minute)
//
//	// Initialize an agent for a workspace in production mode
//	agentConfig, err := initializer.Initialize(
//	    ctx,
//	    "/path/to/workspace",
//	    "prod",
//	)
//	if err != nil {
//	    log.Fatalf("Failed to initialize agent: %v", err)
//	}
//
//	// Use the configuration
//	fmt.Printf("Agent: %s (ID: %s)\n", agentConfig.Name, agentConfig.AgentID)
//
// # Performance
//
// The package uses a TTL-based cache to avoid repeated disk I/O. Typical file parsing
// completes in <100ms for files under 100KB. The cache is thread-safe and automatically
// evicts expired entries in a background goroutine.
//
// # Validation
//
// All configurations are validated before being returned to ensure:
//
//   - Required fields are present (agent_id, name)
//   - Enum fields contain valid values
//   - Numeric ranges are respected (e.g., confidence_threshold: 0.0-1.0)
//   - Cross-field constraints are satisfied (e.g., min < max)
//
// Validation errors include the field path, invalid value, and valid options
// to make debugging easier.
package disposition
