package gateway

import (
	"context"

	"github.com/DojoGenesis/gateway/tools"
)

// ToolRegistry is the interface for managing tool registration and lookup.
// It provides a unified abstraction over the Gateway's tool registry, allowing
// external modules (such as MCP hosts) to register and query tools dynamically.
type ToolRegistry interface {
	// Register adds a new tool to the registry.
	// Returns an error if the tool already exists or the definition is invalid.
	Register(ctx context.Context, def *tools.ToolDefinition) error

	// Get retrieves a tool by its exact name.
	// Returns an error if the tool is not found.
	Get(ctx context.Context, name string) (*tools.ToolDefinition, error)

	// List returns all registered tools in the registry.
	// The returned slice is a snapshot and can be safely iterated.
	List(ctx context.Context) ([]*tools.ToolDefinition, error)

	// ListByNamespace returns all tools matching a namespace prefix.
	// For example, "composio." returns all tools registered by the Composio MCP server.
	// The prefix match is case-sensitive.
	ListByNamespace(ctx context.Context, prefix string) ([]*tools.ToolDefinition, error)
}

// AgentInitializer is the interface for loading agent configuration based on workspace context.
// It abstracts agent disposition loading, allowing the Gateway to initialize agents from
// YAML files, environment variables, or other configuration sources.
type AgentInitializer interface {
	// Initialize loads the agent configuration for the given workspace and mode.
	// workspaceRoot is the absolute path to the workspace directory.
	// activeMode is an optional mode name (e.g., "debug", "prod") for applying mode-specific overrides.
	// If activeMode is empty, the base configuration is returned without overrides.
	// Returns the merged AgentConfig or an error if initialization fails.
	Initialize(ctx context.Context, workspaceRoot string, activeMode string) (*AgentConfig, error)
}

// MemoryStore is the interface for abstracting memory backend operations.
// It provides a unified API for storing, searching, and retrieving memory entries,
// allowing different implementations (in-memory, SQLite, vector databases) to be swapped.
type MemoryStore interface {
	// Store persists a memory entry to the backend.
	// Returns an error if the storage operation fails.
	Store(ctx context.Context, entry *MemoryEntry) error

	// Search queries memory entries using text or embedding similarity.
	// limit specifies the maximum number of results to return.
	// Returns matching entries ordered by relevance.
	Search(ctx context.Context, query *SearchQuery, limit int) ([]*MemoryEntry, error)

	// Get retrieves a specific memory entry by its unique ID.
	// Returns an error if the entry is not found.
	Get(ctx context.Context, id string) (*MemoryEntry, error)

	// Delete removes a memory entry by its unique ID.
	// Returns an error if the entry is not found or deletion fails.
	Delete(ctx context.Context, id string) error
}

// OrchestrationExecutor is the interface for running orchestration DAGs (Directed Acyclic Graphs).
// It allows external orchestration engines to be plugged into the Gateway, enabling
// complex multi-tool workflows with dependency management.
type OrchestrationExecutor interface {
	// Execute runs an orchestration plan represented as a DAG of tool invocations.
	// The executor respects dependency ordering defined in the plan.
	// Returns the execution result or an error if execution fails or is cancelled.
	Execute(ctx context.Context, plan *ExecutionPlan) (*ExecutionResult, error)

	// Cancel terminates a running execution by its unique execution ID.
	// Returns an error if the execution is not found or cancellation fails.
	Cancel(ctx context.Context, executionID string) error
}
