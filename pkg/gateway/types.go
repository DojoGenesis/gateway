package gateway

import "time"

// AgentConfig represents the effective agent configuration after resolving disposition.
// It includes core behavioral dimensions and nested configuration sections.
// This aligns with the Gateway-ADA Contract v1.0.0.
type AgentConfig struct {
	// Core behavioral dimensions (required)
	Pacing     string `json:"pacing" yaml:"pacing"`         // deliberate | measured | responsive | rapid
	Depth      string `json:"depth" yaml:"depth"`           // surface | functional | thorough | exhaustive
	Tone       string `json:"tone" yaml:"tone"`             // formal | professional | conversational | casual
	Initiative string `json:"initiative" yaml:"initiative"` // reactive | responsive | proactive | autonomous

	// Validation contains validation rules for agent outputs.
	Validation ValidationConfig `json:"validation" yaml:"validation"`

	// ErrorHandling defines how the agent handles errors and retries.
	ErrorHandling ErrorHandlingConfig `json:"error_handling" yaml:"error_handling"`

	// Collaboration controls multi-agent collaboration settings.
	Collaboration CollaborationConfig `json:"collaboration" yaml:"collaboration"`

	// Reflection defines self-reflection and improvement behavior.
	Reflection ReflectionConfig `json:"reflection" yaml:"reflection"`
}

// ValidationConfig defines quality assurance preferences.
// Fields match the gateway-ada.md contract specification.
type ValidationConfig struct {
	// Strategy defines the validation approach.
	// Valid values: "none", "spot-check", "thorough", "exhaustive"
	Strategy string `json:"strategy" yaml:"strategy"`

	// RequireTests indicates whether tests must be present for code changes.
	// Default: true
	RequireTests bool `json:"require_tests" yaml:"require_tests"`

	// RequireDocs indicates whether documentation must be present for code changes.
	// Default: false
	RequireDocs bool `json:"require_docs" yaml:"require_docs"`
}

// ErrorHandlingConfig defines error response strategy.
// Fields match the gateway-ada.md contract specification.
type ErrorHandlingConfig struct {
	// Strategy defines how errors are handled.
	// Valid values: "fail-fast", "log-and-continue", "retry", "escalate"
	Strategy string `json:"strategy" yaml:"strategy"`

	// RetryCount is the maximum number of retry attempts (0-10).
	// Default: 3
	RetryCount int `json:"retry_count" yaml:"retry_count"`
}

// CollaborationConfig defines multi-agent/human interaction preferences.
// Fields match the gateway-ada.md contract specification.
type CollaborationConfig struct {
	// Style defines the collaboration approach.
	// Valid values: "independent", "consultative", "collaborative", "delegating"
	Style string `json:"style" yaml:"style"`

	// CheckInFrequency defines how often the agent checks in with collaborators.
	// Valid values: "never", "rarely", "regularly", "constantly"
	CheckInFrequency string `json:"check_in_frequency" yaml:"check_in_frequency"`
}

// ReflectionConfig defines introspection behavior.
// Fields match the gateway-ada.md contract specification.
type ReflectionConfig struct {
	// Frequency defines how often reflection occurs.
	// Valid values: "never", "session-end", "daily", "weekly"
	Frequency string `json:"frequency" yaml:"frequency"`

	// Format defines the structure of reflection outputs.
	// Valid values: "structured", "narrative", "bullets"
	Format string `json:"format" yaml:"format"`

	// Triggers is a list of events that trigger reflection.
	// Examples: "error", "milestone", "learning"
	Triggers []string `json:"triggers" yaml:"triggers"`
}

// MemoryEntry represents a single entry in the memory store.
// Entries can represent conversations, tool calls, reflections, or other
// memory types that need to be stored and searched.
type MemoryEntry struct {
	// ID is the unique identifier for this memory entry.
	ID string `json:"id" yaml:"id"`

	// EntryType categorizes the memory entry (e.g., "conversation", "tool_call", "reflection").
	EntryType string `json:"type" yaml:"type"`

	// Content is the textual content of the memory entry.
	Content string `json:"content" yaml:"content"`

	// Metadata holds additional structured data about the entry.
	Metadata map[string]interface{} `json:"metadata" yaml:"metadata"`

	// CreatedAt is the timestamp when the entry was created.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`

	// UpdatedAt is the timestamp when the entry was last modified.
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`

	// Embedding is the vector representation of the content for similarity search.
	// Omitted from JSON/YAML if empty.
	Embedding []float64 `json:"embedding,omitempty" yaml:"embedding,omitempty"`
}

// SearchQuery defines the parameters for searching memory entries.
type SearchQuery struct {
	// Text is the search query text for text-based or semantic search.
	Text string `json:"text" yaml:"text"`

	// EntryType filters results by memory entry type (e.g., "conversation").
	// If empty, all types are included.
	EntryType string `json:"type,omitempty" yaml:"type,omitempty"`
}

// ExecutionPlan represents an orchestration plan as a Directed Acyclic Graph (DAG)
// of tool invocations. Each tool invocation can depend on the outputs of previous
// invocations, enabling complex multi-step workflows.
type ExecutionPlan struct {
	// ID is the unique identifier for this execution plan.
	ID string `json:"id" yaml:"id"`

	// Name is a human-readable name for the plan.
	Name string `json:"name" yaml:"name"`

	// DAG is the list of tool invocations that make up the execution plan.
	// Invocations are executed in dependency order.
	DAG []*ToolInvocation `json:"dag" yaml:"dag"`
}

// ExecutionResult represents the result of executing an orchestration plan.
type ExecutionResult struct {
	// ExecutionID is the unique identifier for this execution.
	ExecutionID string `json:"execution_id" yaml:"execution_id"`

	// Status indicates whether the execution succeeded, failed, or was cancelled.
	// Valid values: "success", "failed", "cancelled"
	Status string `json:"status" yaml:"status"`

	// Output contains the final output data from the execution.
	Output map[string]interface{} `json:"output" yaml:"output"`

	// Error contains the error message if the execution failed.
	Error string `json:"error,omitempty" yaml:"error,omitempty"`

	// Duration is the total execution time in milliseconds.
	Duration int64 `json:"duration_ms" yaml:"duration_ms"`
}

// ToolInvocation represents a single tool call within an orchestration DAG.
type ToolInvocation struct {
	// ID is the unique identifier for this invocation within the plan.
	ID string `json:"id" yaml:"id"`

	// ToolName is the name of the tool to invoke.
	ToolName string `json:"tool_name" yaml:"tool_name"`

	// Input is the input data to pass to the tool.
	Input map[string]interface{} `json:"input" yaml:"input"`

	// DependsOn lists the IDs of invocations that must complete before this one.
	// If empty, this invocation has no dependencies and can run immediately.
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
}
