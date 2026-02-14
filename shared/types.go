package shared

import "time"

// Message is a conversation message (used across provider, memory, orchestration).
type Message struct {
	Role       string     `json:"role"`                   // "system", "user", "assistant", "tool"
	Content    string     `json:"content"`                // Message text
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // Assistant tool calls (role == "assistant")
	ToolCallID string     `json:"tool_call_id,omitempty"` // Tool call ID (role == "tool")
}

// ToolCall represents a tool invocation in a message.
type ToolCall struct {
	ID        string                 `json:"id"`        // Unique call ID
	Name      string                 `json:"name"`      // Tool name
	Arguments map[string]interface{} `json:"arguments"` // Parsed arguments
}

// ToolResult is the output of tool execution.
type ToolResult struct {
	ToolName string      `json:"tool_name"`
	Success  bool        `json:"success"`
	Content  interface{} `json:"content"`
	Error    string      `json:"error,omitempty"`
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// TaskStatus enumeration (used by orchestration).
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskSkipped   TaskStatus = "skipped"
)

// NodeState enumeration (used by orchestration).
type NodeState string

const (
	NodeStatePending NodeState = "pending"
	NodeStateRunning NodeState = "running"
	NodeStateSuccess NodeState = "success"
	NodeStateFailed  NodeState = "failed"
	NodeStateSkipped NodeState = "skipped"
)

// CallRequest is used to call a model provider.
type CallRequest struct {
	Messages    []Message `json:"messages"`
	Tools       []ToolUse `json:"tools,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
}

// ToolUse describes a tool available for a model call.
type ToolUse struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// CallResponse is the result of a model provider call.
type CallResponse struct {
	Message    Message `json:"message"`
	TokensUsed int     `json:"tokens_used"`
	StopReason string  `json:"stop_reason"` // "stop", "tool_use", "length", etc.
}

// ProviderConfig holds provider configuration.
type ProviderConfig struct {
	Name        string  `json:"name"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float32 `json:"temperature"`
	APIKey      string  `json:"api_key,omitempty"`
	Endpoint    string  `json:"endpoint,omitempty"`
}

// Timestamp is a convenience alias.
type Timestamp = time.Time
