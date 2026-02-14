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
}

type CompletionResponse struct {
	ID        string
	Model     string
	Content   string
	Usage     Usage
	ToolCalls []ToolCall
}

type CompletionChunk struct {
	ID    string
	Delta string
	Done  bool
}

type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
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
