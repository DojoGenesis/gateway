package provider

type ProviderInfo struct {
	Name         string
	Version      string
	Description  string
	Capabilities []string
}

type ModelInfo struct {
	ID          string
	Name        string
	Provider    string
	ContextSize int
	Cost        float64
}

type CompletionRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
	Tools       []Tool
	Stream      bool
	// ToolChoice controls the model's tool-calling behaviour.
	// "required" — model MUST call a tool (first iteration to force action).
	// "auto"     — model decides (default for follow-up iterations).
	// "none"     — model must NOT call any tools.
	// Empty string is treated as "auto".
	ToolChoice string
}

type CompletionResponse struct {
	ID               string
	Model            string
	Content          string
	Usage            Usage
	ToolCalls        []ToolCall
	ReasoningContent string // preserved from providers that emit reasoning (e.g. Kimi) for multi-turn continuity
}

type CompletionChunk struct {
	ID    string
	Delta string
	Done  bool
}

type Message struct {
	Role             string
	Content          string
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // Kimi: must round-trip when present
}

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]interface{}
}

type ToolCallRequest struct {
	ToolCall ToolCall
	Context  map[string]interface{}
}

type ToolCallResponse struct {
	Result interface{}
	Error  string
}
