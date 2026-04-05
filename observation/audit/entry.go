package audit

import "time"

// Action categorizes the type of audited event.
type Action string

const (
	// ActionToolExecution records a tool being executed.
	ActionToolExecution Action = "tool_execution"
	// ActionCapabilityGrant records a capability being granted.
	ActionCapabilityGrant Action = "capability_grant"
	// ActionAgentSpawn records an agent being spawned.
	ActionAgentSpawn Action = "agent_spawn"
	// ActionAgentStop records an agent being stopped.
	ActionAgentStop Action = "agent_stop"
	// ActionMemoryAccess records a memory read/write.
	ActionMemoryAccess Action = "memory_access"
	// ActionSkillInvocation records a skill being invoked.
	ActionSkillInvocation Action = "skill_invocation"
)

// ExportFormat specifies the output format for audit exports.
type ExportFormat string

const (
	ExportJSON ExportFormat = "json"
	ExportCSV  ExportFormat = "csv"
)

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	// ID is the unique entry identifier.
	ID string

	// Timestamp is when the event occurred.
	Timestamp time.Time

	// AgentID identifies which agent performed the action.
	AgentID string

	// Action categorizes the event.
	Action Action

	// Tool is the tool name (for tool execution events).
	Tool string

	// ToolArgs are the tool arguments (for tool execution events).
	ToolArgs map[string]interface{}

	// CapabilitiesGranted lists capabilities granted (for capability events).
	CapabilitiesGranted []string

	// ResultHash is the SHA-256 of the result (for tamper evidence).
	ResultHash string

	// Duration is how long the action took.
	Duration time.Duration

	// Metadata holds additional context.
	Metadata map[string]interface{}
}

// AuditFilter specifies criteria for querying audit entries.
type AuditFilter struct {
	// AgentID filters by agent.
	AgentID string

	// Actions filters by action type.
	Actions []Action

	// From filters entries after this time.
	From time.Time

	// To filters entries before this time.
	To time.Time

	// Tool filters by tool name.
	Tool string

	// Limit caps the number of results.
	Limit int

	// Offset skips the first N results (for pagination).
	Offset int
}
