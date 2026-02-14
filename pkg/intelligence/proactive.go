package intelligence

import (
	"context"
	"strings"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
)

// TaskEvent represents an event that might trigger proactive suggestions.
type TaskEvent struct {
	Type     string      // "task_complete", "error", "explicit_request", "idle"
	TaskID   string      // Identifier for the task
	Result   interface{} // Result of the task (if applicable)
	Metadata map[string]interface{}
}

// ProposedAction represents an action the agent could take proactively.
type ProposedAction struct {
	Description string                 // Human-readable description
	ToolName    string                 // Name of tool to invoke
	Args        map[string]interface{} // Arguments for the tool
	Risk        string                 // "low", "medium", "high"
	Confidence  float64                // 0.0-1.0
}

// Suggestion represents a suggestion the agent can make to the user.
type Suggestion struct {
	Description string          // What the agent suggests
	Confidence  float64         // How confident the agent is (0.0-1.0)
	Action      *ProposedAction // Optional action to execute (nil for informational)
	Reasoning   string          // Why the agent suggests this
}

// AgentState represents the current state of the agent for suggestion generation.
type AgentState struct {
	CompletedTasks  []string
	PendingTasks    []string
	RecentErrors    []error
	ConversationLog []string
}

// ProactiveEngine decides when and how to make suggestions based on disposition.Initiative.
//
// Per Gateway-ADA Contract §3.3:
//   - reactive: no suggestions, wait for explicit commands
//   - responsive: suggest only when explicitly asked
//   - proactive: suggest next steps after task completion
//   - autonomous: execute anticipated tasks without approval
type ProactiveEngine struct {
	disp *disposition.DispositionConfig
}

// EngineOption is a functional option for configuring the ProactiveEngine.
type EngineOption func(*ProactiveEngine)

// WithDisposition sets the disposition configuration.
func WithDisposition(disp *disposition.DispositionConfig) EngineOption {
	return func(e *ProactiveEngine) {
		e.disp = disp
	}
}

// NewProactiveEngine creates a new proactive intelligence engine.
func NewProactiveEngine(opts ...EngineOption) *ProactiveEngine {
	e := &ProactiveEngine{
		disp: disposition.DefaultDisposition(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ShouldSuggest determines whether the engine should suggest next steps
// based on the current event and initiative level.
func (e *ProactiveEngine) ShouldSuggest(ctx context.Context, event TaskEvent) bool {
	if e.disp == nil {
		return false
	}

	initiative := strings.ToLower(e.disp.Initiative)

	switch initiative {
	case "reactive":
		// Never suggest - only respond to explicit commands
		return false

	case "responsive":
		// Only suggest when explicitly asked
		return event.Type == "explicit_request"

	case "proactive":
		// Suggest after task completion or when asked
		return event.Type == "task_complete" || event.Type == "explicit_request"

	case "autonomous":
		// Always suggest (will auto-execute in some cases)
		return true

	default:
		// Unknown initiative - default to responsive
		return event.Type == "explicit_request"
	}
}

// ShouldAutoExecute determines whether to execute an action without user approval.
// Only autonomous initiative allows auto-execution, and only for low-risk actions.
func (e *ProactiveEngine) ShouldAutoExecute(ctx context.Context, action ProposedAction) bool {
	if e.disp == nil {
		return false
	}

	initiative := strings.ToLower(e.disp.Initiative)

	// Only autonomous initiative can auto-execute
	if initiative != "autonomous" {
		return false
	}

	// Only execute low-risk actions automatically
	if action.Risk != "low" {
		return false
	}

	// Require high confidence for auto-execution
	if action.Confidence < 0.8 {
		return false
	}

	return true
}

// GenerateSuggestions produces next-step suggestions based on current agent state.
// Returns empty list for reactive initiative.
func (e *ProactiveEngine) GenerateSuggestions(ctx context.Context, state AgentState) ([]Suggestion, error) {
	if e.disp == nil {
		return nil, nil
	}

	initiative := strings.ToLower(e.disp.Initiative)

	// Reactive agents don't suggest
	if initiative == "reactive" {
		return nil, nil
	}

	// This is a simplified heuristic implementation.
	// A production version would use LLM-based reasoning.
	suggestions := make([]Suggestion, 0)

	// If there are recent errors, suggest investigation
	if len(state.RecentErrors) > 0 && (initiative == "proactive" || initiative == "autonomous") {
		suggestions = append(suggestions, Suggestion{
			Description: "Investigate recent errors to prevent recurrence",
			Confidence:  0.7,
			Action: &ProposedAction{
				Description: "Analyze error patterns",
				ToolName:    "error_analysis",
				Args:        map[string]interface{}{"errors": state.RecentErrors},
				Risk:        "low",
				Confidence:  0.7,
			},
			Reasoning: "Multiple errors detected in recent tasks",
		})
	}

	// If tasks are pending, suggest prioritization
	if len(state.PendingTasks) > 3 && (initiative == "proactive" || initiative == "autonomous") {
		suggestions = append(suggestions, Suggestion{
			Description: "Prioritize pending tasks based on dependencies",
			Confidence:  0.6,
			Reasoning:   "Large backlog of pending tasks detected",
		})
	}

	return suggestions, nil
}
