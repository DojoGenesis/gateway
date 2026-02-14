package reflection

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
)

// ReflectionEvent represents an event that might trigger reflection.
type ReflectionEvent struct {
	Type      string    // matches triggers: "error", "milestone", "learning", "session_end"
	Timestamp time.Time // When the event occurred
	Context   string    // Description of what happened
	Metadata  map[string]interface{}
}

// SessionData contains information about a session for reflection.
type SessionData struct {
	TasksCompleted int           // Number of tasks completed
	ErrorsHit      int           // Number of errors encountered
	ToolsUsed      []string      // List of tools used
	Duration       time.Duration // Session duration
	Events         []ReflectionEvent
	StartTime      time.Time
	EndTime        time.Time
}

// ReflectionOutput represents the generated reflection content.
type ReflectionOutput struct {
	Format  string // "structured", "narrative", "bullets"
	Content string // Markdown-formatted reflection content
}

// Engine generates reflections based on disposition.Reflection.
//
// Per Gateway-ADA Contract §3.7:
//
// Frequency:
//   - never: Disable automatic reflection
//   - session-end: Trigger reflection at session completion
//   - daily: Trigger reflection at end of day
//   - weekly: Trigger reflection at end of week
//
// Format:
//   - structured: YAML template with sections
//   - narrative: Markdown freeform
//   - bullets: Concise bullet points
//
// Triggers: Array of event types triggering reflection
type Engine struct {
	disp     *disposition.DispositionConfig
	eventLog []ReflectionEvent
}

// EngineOption is a functional option for configuring the Engine.
type EngineOption func(*Engine)

// WithDisposition sets the disposition configuration.
func WithDisposition(disp *disposition.DispositionConfig) EngineOption {
	return func(e *Engine) {
		e.disp = disp
	}
}

// NewEngine creates a new reflection engine.
func NewEngine(opts ...EngineOption) *Engine {
	e := &Engine{
		disp:     disposition.DefaultDisposition(),
		eventLog: make([]ReflectionEvent, 0),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ShouldReflect checks whether reflection should be triggered now.
// Returns true if the frequency setting allows it AND the trigger matches configured triggers.
func (e *Engine) ShouldReflect(ctx context.Context, trigger string) bool {
	if e.disp == nil {
		return false
	}

	frequency := strings.ToLower(e.disp.Reflection.Frequency)

	// Never reflect if frequency is "never"
	if frequency == "never" {
		return false
	}

	// Check if trigger matches configured triggers
	triggerMatches := false
	for _, configuredTrigger := range e.disp.Reflection.Triggers {
		if strings.EqualFold(configuredTrigger, trigger) {
			triggerMatches = true
			break
		}
	}

	return triggerMatches
}

// LogEvent adds an event to the reflection log.
func (e *Engine) LogEvent(event ReflectionEvent) {
	e.eventLog = append(e.eventLog, event)
}

// ClearEventLog clears all logged events (e.g., after generating reflection).
func (e *Engine) ClearEventLog() {
	e.eventLog = make([]ReflectionEvent, 0)
}

// GetEventLog returns a copy of the current event log.
func (e *Engine) GetEventLog() []ReflectionEvent {
	logCopy := make([]ReflectionEvent, len(e.eventLog))
	copy(logCopy, e.eventLog)
	return logCopy
}

// GenerateReflection produces a reflection artifact in the configured format.
func (e *Engine) GenerateReflection(ctx context.Context, sessionData SessionData) (*ReflectionOutput, error) {
	if e.disp == nil {
		return nil, fmt.Errorf("reflection engine requires a disposition configuration: call NewEngine(WithDisposition(cfg)) to provide one")
	}

	format := strings.ToLower(e.disp.Reflection.Format)

	switch format {
	case "structured":
		return e.generateStructured(sessionData)
	case "narrative":
		return e.generateNarrative(sessionData)
	case "bullets":
		return e.generateBullets(sessionData)
	default:
		// Unknown format - default to structured
		return e.generateStructured(sessionData)
	}
}

// generateStructured creates a YAML-like structured reflection.
func (e *Engine) generateStructured(sessionData SessionData) (*ReflectionOutput, error) {
	var content strings.Builder

	content.WriteString("# Reflection\n\n")
	content.WriteString("## Session Summary\n")
	content.WriteString(fmt.Sprintf("- Duration: %v\n", sessionData.Duration))
	content.WriteString(fmt.Sprintf("- Tasks Completed: %d\n", sessionData.TasksCompleted))
	content.WriteString(fmt.Sprintf("- Errors Encountered: %d\n", sessionData.ErrorsHit))
	content.WriteString(fmt.Sprintf("- Tools Used: %d\n\n", len(sessionData.ToolsUsed)))

	content.WriteString("## Key Events\n")
	if len(sessionData.Events) == 0 {
		content.WriteString("- No significant events logged\n\n")
	} else {
		for _, event := range sessionData.Events {
			content.WriteString(fmt.Sprintf("- **%s**: %s\n", event.Type, event.Context))
		}
		content.WriteString("\n")
	}

	content.WriteString("## Insights\n")
	content.WriteString("- (Insights would be generated by LLM in production)\n\n")

	content.WriteString("## Next Steps\n")
	content.WriteString("- (Next steps would be generated by LLM in production)\n")

	return &ReflectionOutput{
		Format:  "structured",
		Content: content.String(),
	}, nil
}

// generateNarrative creates a freeform narrative reflection.
func (e *Engine) generateNarrative(sessionData SessionData) (*ReflectionOutput, error) {
	var content strings.Builder

	content.WriteString("# Session Reflection\n\n")
	content.WriteString(fmt.Sprintf("During this session, which lasted %v, I completed %d tasks and encountered %d errors. ",
		sessionData.Duration, sessionData.TasksCompleted, sessionData.ErrorsHit))

	if len(sessionData.ToolsUsed) > 0 {
		content.WriteString(fmt.Sprintf("I used %d different tools throughout the session. ", len(sessionData.ToolsUsed)))
	}

	content.WriteString("\n\n")

	if len(sessionData.Events) > 0 {
		content.WriteString("Notable events during this session:\n\n")
		for _, event := range sessionData.Events {
			content.WriteString(fmt.Sprintf("- At %s, %s: %s\n",
				event.Timestamp.Format("15:04:05"), event.Type, event.Context))
		}
		content.WriteString("\n")
	}

	content.WriteString("_Note: In production, this narrative would be generated by an LLM with deeper analysis._\n")

	return &ReflectionOutput{
		Format:  "narrative",
		Content: content.String(),
	}, nil
}

// generateBullets creates a concise bullet-point reflection.
func (e *Engine) generateBullets(sessionData SessionData) (*ReflectionOutput, error) {
	var content strings.Builder

	content.WriteString("# Session Reflection\n\n")
	content.WriteString("## Summary\n")
	content.WriteString(fmt.Sprintf("- Duration: %v\n", sessionData.Duration))
	content.WriteString(fmt.Sprintf("- %d tasks completed\n", sessionData.TasksCompleted))
	content.WriteString(fmt.Sprintf("- %d errors encountered\n", sessionData.ErrorsHit))
	content.WriteString(fmt.Sprintf("- %d tools used\n\n", len(sessionData.ToolsUsed)))

	if len(sessionData.Events) > 0 {
		content.WriteString("## Key Events\n")
		for _, event := range sessionData.Events {
			content.WriteString(fmt.Sprintf("- %s: %s\n", event.Type, event.Context))
		}
		content.WriteString("\n")
	}

	content.WriteString("## Takeaways\n")
	content.WriteString("- (Generated by LLM in production)\n")

	return &ReflectionOutput{
		Format:  "bullets",
		Content: content.String(),
	}, nil
}
