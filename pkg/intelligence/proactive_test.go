package intelligence

import (
	"context"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
)

func TestShouldSuggest_Reactive(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Initiative: "reactive",
	}

	engine := NewProactiveEngine(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name  string
		event TaskEvent
		want  bool
	}{
		{"explicit request", TaskEvent{Type: "explicit_request"}, false},
		{"task complete", TaskEvent{Type: "task_complete"}, false},
		{"error", TaskEvent{Type: "error"}, false},
		{"idle", TaskEvent{Type: "idle"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.ShouldSuggest(ctx, tt.event)
			if got != tt.want {
				t.Errorf("reactive initiative: %s: expected %v, got %v", tt.name, tt.want, got)
			}
		})
	}
}

func TestShouldSuggest_Responsive(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Initiative: "responsive",
	}

	engine := NewProactiveEngine(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name  string
		event TaskEvent
		want  bool
	}{
		{"explicit request", TaskEvent{Type: "explicit_request"}, true},
		{"task complete", TaskEvent{Type: "task_complete"}, false},
		{"error", TaskEvent{Type: "error"}, false},
		{"idle", TaskEvent{Type: "idle"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.ShouldSuggest(ctx, tt.event)
			if got != tt.want {
				t.Errorf("responsive initiative: %s: expected %v, got %v", tt.name, tt.want, got)
			}
		})
	}
}

func TestShouldSuggest_Proactive(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Initiative: "proactive",
	}

	engine := NewProactiveEngine(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name  string
		event TaskEvent
		want  bool
	}{
		{"explicit request", TaskEvent{Type: "explicit_request"}, true},
		{"task complete", TaskEvent{Type: "task_complete"}, true},
		{"error", TaskEvent{Type: "error"}, false},
		{"idle", TaskEvent{Type: "idle"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.ShouldSuggest(ctx, tt.event)
			if got != tt.want {
				t.Errorf("proactive initiative: %s: expected %v, got %v", tt.name, tt.want, got)
			}
		})
	}
}

func TestShouldSuggest_Autonomous(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Initiative: "autonomous",
	}

	engine := NewProactiveEngine(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name  string
		event TaskEvent
		want  bool
	}{
		{"explicit request", TaskEvent{Type: "explicit_request"}, true},
		{"task complete", TaskEvent{Type: "task_complete"}, true},
		{"error", TaskEvent{Type: "error"}, true},
		{"idle", TaskEvent{Type: "idle"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.ShouldSuggest(ctx, tt.event)
			if got != tt.want {
				t.Errorf("autonomous initiative: %s: expected %v, got %v", tt.name, tt.want, got)
			}
		})
	}
}

func TestShouldAutoExecute_OnlyAutonomous(t *testing.T) {
	ctx := context.Background()
	action := ProposedAction{
		Risk:       "low",
		Confidence: 0.9,
	}

	tests := []struct {
		initiative string
		want       bool
	}{
		{"reactive", false},
		{"responsive", false},
		{"proactive", false},
		{"autonomous", true},
	}

	for _, tt := range tests {
		t.Run(tt.initiative, func(t *testing.T) {
			disp := &disposition.DispositionConfig{
				Initiative: tt.initiative,
			}
			engine := NewProactiveEngine(WithDisposition(disp))

			got := engine.ShouldAutoExecute(ctx, action)
			if got != tt.want {
				t.Errorf("%s initiative: expected %v, got %v", tt.initiative, tt.want, got)
			}
		})
	}
}

func TestShouldAutoExecute_RiskFiltering(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Initiative: "autonomous",
	}
	engine := NewProactiveEngine(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name       string
		risk       string
		confidence float64
		want       bool
	}{
		{"low risk, high confidence", "low", 0.9, true},
		{"low risk, medium confidence", "low", 0.7, false},
		{"medium risk, high confidence", "medium", 0.9, false},
		{"high risk, high confidence", "high", 0.9, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := ProposedAction{
				Risk:       tt.risk,
				Confidence: tt.confidence,
			}

			got := engine.ShouldAutoExecute(ctx, action)
			if got != tt.want {
				t.Errorf("%s: expected %v, got %v", tt.name, tt.want, got)
			}
		})
	}
}

func TestGenerateSuggestions_Reactive(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Initiative: "reactive",
	}
	engine := NewProactiveEngine(WithDisposition(disp))
	ctx := context.Background()

	state := AgentState{
		RecentErrors: []error{},
		PendingTasks: []string{"task1", "task2"},
	}

	suggestions, err := engine.GenerateSuggestions(ctx, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reactive should never suggest
	if len(suggestions) != 0 {
		t.Errorf("reactive initiative should not generate suggestions, got %d", len(suggestions))
	}
}

func TestGenerateSuggestions_ProactiveWithErrors(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Initiative: "proactive",
	}
	engine := NewProactiveEngine(WithDisposition(disp))
	ctx := context.Background()

	state := AgentState{
		RecentErrors: []error{errDummy, errDummy},
		PendingTasks: []string{},
	}

	suggestions, err := engine.GenerateSuggestions(ctx, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should suggest error investigation
	if len(suggestions) == 0 {
		t.Error("proactive with errors should generate suggestions")
	}
}

// Dummy error for testing
var errDummy = &dummyError{}

type dummyError struct{}

func (e *dummyError) Error() string {
	return "dummy error"
}
