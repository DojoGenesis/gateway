// Package disposition provides the AgentInitializerImpl which implements the
// gateway.AgentInitializer interface.
//
// It bridges the disposition resolution logic (from the disposition package)
// to the gateway's public interface, converting DispositionConfig into
// gateway.AgentConfig with TTL-based caching.
//
// Usage:
//
//	init := disposition.NewAgentInitializer(5 * time.Minute)
//	config, err := init.Initialize(ctx, "/workspace", "code-review")
package disposition
