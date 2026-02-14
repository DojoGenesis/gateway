package reflection

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
)

func TestShouldReflect_FrequencyNever(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Frequency: "never",
			Triggers:  []string{"error", "milestone"},
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	// Even with matching triggers, "never" should never reflect
	if engine.ShouldReflect(ctx, "error") {
		t.Error("frequency never: should not reflect on error trigger")
	}
	if engine.ShouldReflect(ctx, "milestone") {
		t.Error("frequency never: should not reflect on milestone trigger")
	}
}

func TestShouldReflect_FrequencySessionEnd(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Frequency: "session-end",
			Triggers:  []string{"error", "milestone"},
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	// Should reflect on configured triggers
	if !engine.ShouldReflect(ctx, "error") {
		t.Error("should reflect on configured trigger 'error'")
	}
	if !engine.ShouldReflect(ctx, "milestone") {
		t.Error("should reflect on configured trigger 'milestone'")
	}

	// Should not reflect on non-configured triggers
	if engine.ShouldReflect(ctx, "learning") {
		t.Error("should not reflect on non-configured trigger 'learning'")
	}
}

func TestShouldReflect_FrequencyDaily(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Frequency: "daily",
			Triggers:  []string{"session_end"},
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	// Should reflect on configured trigger
	if !engine.ShouldReflect(ctx, "session_end") {
		t.Error("should reflect on configured trigger 'session_end'")
	}

	// Should not reflect on non-configured trigger
	if engine.ShouldReflect(ctx, "error") {
		t.Error("should not reflect on non-configured trigger 'error'")
	}
}

func TestShouldReflect_FrequencyWeekly(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Frequency: "weekly",
			Triggers:  []string{"milestone", "learning"},
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	// Should reflect on both configured triggers
	if !engine.ShouldReflect(ctx, "milestone") {
		t.Error("should reflect on configured trigger 'milestone'")
	}
	if !engine.ShouldReflect(ctx, "learning") {
		t.Error("should reflect on configured trigger 'learning'")
	}
}

func TestShouldReflect_TriggerMatching(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Frequency: "session-end",
			Triggers:  []string{"error", "milestone"},
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		trigger string
		want    bool
	}{
		{"error", true},
		{"milestone", true},
		{"learning", false},
		{"session_end", false},
		{"ERROR", true},     // Case-insensitive
		{"MILESTONE", true}, // Case-insensitive
	}

	for _, tt := range tests {
		t.Run(tt.trigger, func(t *testing.T) {
			got := engine.ShouldReflect(ctx, tt.trigger)
			if got != tt.want {
				t.Errorf("trigger %q: expected %v, got %v", tt.trigger, tt.want, got)
			}
		})
	}
}

func TestGenerateReflection_Structured(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Format: "structured",
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	sessionData := SessionData{
		TasksCompleted: 5,
		ErrorsHit:      1,
		ToolsUsed:      []string{"tool1", "tool2"},
		Duration:       30 * time.Minute,
		Events: []ReflectionEvent{
			{Type: "error", Context: "test error"},
		},
	}

	output, err := engine.GenerateReflection(ctx, sessionData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Format != "structured" {
		t.Errorf("expected structured format, got %s", output.Format)
	}

	// Verify content contains expected sections
	if !strings.Contains(output.Content, "# Reflection") {
		t.Error("structured output should contain reflection header")
	}
	if !strings.Contains(output.Content, "## Session Summary") {
		t.Error("structured output should contain session summary")
	}
	if !strings.Contains(output.Content, "Tasks Completed: 5") {
		t.Error("structured output should contain task count")
	}
}

func TestGenerateReflection_Narrative(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Format: "narrative",
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	sessionData := SessionData{
		TasksCompleted: 3,
		ErrorsHit:      2,
		ToolsUsed:      []string{"tool1"},
		Duration:       15 * time.Minute,
		Events:         []ReflectionEvent{},
	}

	output, err := engine.GenerateReflection(ctx, sessionData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Format != "narrative" {
		t.Errorf("expected narrative format, got %s", output.Format)
	}

	// Verify narrative style
	if !strings.Contains(output.Content, "During this session") {
		t.Error("narrative output should have narrative introduction")
	}
	if !strings.Contains(output.Content, "3 tasks") {
		t.Error("narrative output should mention task count")
	}
}

func TestGenerateReflection_Bullets(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Format: "bullets",
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	sessionData := SessionData{
		TasksCompleted: 7,
		ErrorsHit:      0,
		ToolsUsed:      []string{"tool1", "tool2", "tool3"},
		Duration:       45 * time.Minute,
		Events:         []ReflectionEvent{},
	}

	output, err := engine.GenerateReflection(ctx, sessionData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Format != "bullets" {
		t.Errorf("expected bullets format, got %s", output.Format)
	}

	// Verify bullet points
	bulletCount := strings.Count(output.Content, "\n- ")
	if bulletCount < 3 {
		t.Errorf("bullets output should have at least 3 bullet points, got %d", bulletCount)
	}
}

func TestEventLog(t *testing.T) {
	engine := NewEngine()

	// Initially empty
	if len(engine.GetEventLog()) != 0 {
		t.Error("initial event log should be empty")
	}

	// Log some events
	event1 := ReflectionEvent{Type: "error", Context: "test error 1"}
	event2 := ReflectionEvent{Type: "milestone", Context: "completed task"}

	engine.LogEvent(event1)
	engine.LogEvent(event2)

	log := engine.GetEventLog()
	if len(log) != 2 {
		t.Errorf("expected 2 events in log, got %d", len(log))
	}

	// Clear log
	engine.ClearEventLog()
	if len(engine.GetEventLog()) != 0 {
		t.Error("event log should be empty after clear")
	}
}

func TestGenerateReflection_DefaultDisposition(t *testing.T) {
	// Engine without explicit disposition should use DefaultDisposition
	engine := NewEngine()
	ctx := context.Background()

	sessionData := SessionData{
		TasksCompleted: 1,
		ErrorsHit:      0,
		ToolsUsed:      []string{},
		Duration:       5 * time.Minute,
		Events:         []ReflectionEvent{},
	}

	output, err := engine.GenerateReflection(ctx, sessionData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DefaultDisposition uses "structured" format
	if output.Format != "structured" {
		t.Errorf("default disposition: expected structured format, got %s", output.Format)
	}
}

func TestGenerateReflection_UnknownFormat(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Reflection: disposition.ReflectionConfig{
			Format: "unknown_format",
		},
	}

	engine := NewEngine(WithDisposition(disp))
	ctx := context.Background()

	sessionData := SessionData{
		TasksCompleted: 1,
		ErrorsHit:      0,
		ToolsUsed:      []string{},
		Duration:       5 * time.Minute,
		Events:         []ReflectionEvent{},
	}

	output, err := engine.GenerateReflection(ctx, sessionData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unknown format should default to structured
	if output.Format != "structured" {
		t.Errorf("unknown format: expected default to structured, got %s", output.Format)
	}
}
