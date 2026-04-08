package collaboration

import (
	"context"
	"testing"

	"github.com/DojoGenesis/gateway/disposition"
)

func TestShouldCheckIn_Independent(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Collaboration: disposition.CollaborationConfig{
			Style:            "independent",
			CheckInFrequency: "constantly", // Even with constant frequency, independent never checks in
		},
	}

	manager := NewManager(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name  string
		event CollabEvent
	}{
		{"decision point", CollabEvent{Type: "decision_point", IsSignificant: true}},
		{"major milestone", CollabEvent{Type: "major_milestone", IsSignificant: true}},
		{"action", CollabEvent{Type: "action", IsSignificant: true}},
		{"agent handoff", CollabEvent{Type: "agent_handoff", IsSignificant: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCheck := manager.ShouldCheckIn(ctx, tt.event)
			if shouldCheck {
				t.Errorf("independent style: %s: should never check in, got true", tt.name)
			}
		})
	}
}

func TestShouldCheckIn_Consultative(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Collaboration: disposition.CollaborationConfig{
			Style:            "consultative",
			CheckInFrequency: "constantly",
		},
	}

	manager := NewManager(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name  string
		event CollabEvent
		want  bool
	}{
		{"decision point (significant)", CollabEvent{Type: "decision_point", IsSignificant: true}, true},
		{"decision point (not significant)", CollabEvent{Type: "decision_point", IsSignificant: false}, false},
		{"action (significant)", CollabEvent{Type: "action", IsSignificant: true}, false},
		{"milestone (significant)", CollabEvent{Type: "major_milestone", IsSignificant: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.ResetActionCount()
			shouldCheck := manager.ShouldCheckIn(ctx, tt.event)
			if shouldCheck != tt.want {
				t.Errorf("consultative style: %s: expected %v, got %v", tt.name, tt.want, shouldCheck)
			}
		})
	}
}

func TestShouldCheckIn_Collaborative(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Collaboration: disposition.CollaborationConfig{
			Style:            "collaborative",
			CheckInFrequency: "constantly",
		},
	}

	manager := NewManager(WithDisposition(disp))
	ctx := context.Background()

	// With "constantly" frequency, should check in on significant events
	event := CollabEvent{Type: "action", IsSignificant: true}
	shouldCheck := manager.ShouldCheckIn(ctx, event)
	if !shouldCheck {
		t.Error("collaborative + constantly: should check in on significant action")
	}

	// Non-significant event should not trigger check-in
	manager.ResetActionCount()
	event = CollabEvent{Type: "action", IsSignificant: false}
	shouldCheck = manager.ShouldCheckIn(ctx, event)
	if shouldCheck {
		t.Error("collaborative + constantly: should not check in on non-significant action")
	}
}

func TestShouldCheckIn_Delegating(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Collaboration: disposition.CollaborationConfig{
			Style:            "delegating",
			CheckInFrequency: "constantly",
		},
	}

	manager := NewManager(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name  string
		event CollabEvent
		want  bool
	}{
		{"agent handoff (significant)", CollabEvent{Type: "agent_handoff", IsSignificant: true}, true},
		{"decision point (significant)", CollabEvent{Type: "decision_point", IsSignificant: true}, true},
		{"action (significant)", CollabEvent{Type: "action", IsSignificant: true}, false},
		{"milestone (significant)", CollabEvent{Type: "major_milestone", IsSignificant: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.ResetActionCount()
			shouldCheck := manager.ShouldCheckIn(ctx, tt.event)
			if shouldCheck != tt.want {
				t.Errorf("delegating style: %s: expected %v, got %v", tt.name, tt.want, shouldCheck)
			}
		})
	}
}

func TestShouldCheckIn_FrequencyNever(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Collaboration: disposition.CollaborationConfig{
			Style:            "collaborative",
			CheckInFrequency: "never",
		},
	}

	manager := NewManager(WithDisposition(disp))
	ctx := context.Background()

	event := CollabEvent{Type: "decision_point", IsSignificant: true}
	shouldCheck := manager.ShouldCheckIn(ctx, event)
	if shouldCheck {
		t.Error("frequency never: should never check in")
	}
}

func TestShouldCheckIn_FrequencyRarely(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Collaboration: disposition.CollaborationConfig{
			Style:            "collaborative",
			CheckInFrequency: "rarely",
		},
	}

	manager := NewManager(WithDisposition(disp))
	ctx := context.Background()

	tests := []struct {
		name  string
		event CollabEvent
		want  bool
	}{
		{"major milestone", CollabEvent{Type: "major_milestone", IsSignificant: true}, true},
		{"decision point", CollabEvent{Type: "decision_point", IsSignificant: true}, false},
		{"action", CollabEvent{Type: "action", IsSignificant: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.ResetActionCount()
			shouldCheck := manager.ShouldCheckIn(ctx, tt.event)
			if shouldCheck != tt.want {
				t.Errorf("frequency rarely: %s: expected %v, got %v", tt.name, tt.want, shouldCheck)
			}
		})
	}
}

func TestShouldCheckIn_FrequencyRegularly(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Collaboration: disposition.CollaborationConfig{
			Style:            "collaborative",
			CheckInFrequency: "regularly",
		},
	}

	manager := NewManager(WithDisposition(disp))
	ctx := context.Background()

	event := CollabEvent{Type: "action", IsSignificant: false}

	// Should check in every 7 actions (action count starts at 0)
	// Actions 1-6 should not trigger
	for i := 1; i <= 6; i++ {
		shouldCheck := manager.ShouldCheckIn(ctx, event)
		if shouldCheck {
			t.Errorf("frequency regularly: action %d should not trigger check-in", i)
		}
	}

	// Action 7 should trigger
	shouldCheck := manager.ShouldCheckIn(ctx, event)
	if !shouldCheck {
		t.Error("frequency regularly: action 7 should trigger check-in")
	}
}

func TestShouldCheckIn_FrequencyConstantly(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Collaboration: disposition.CollaborationConfig{
			Style:            "collaborative",
			CheckInFrequency: "constantly",
		},
	}

	manager := NewManager(WithDisposition(disp))
	ctx := context.Background()

	// Significant events should trigger
	event := CollabEvent{Type: "action", IsSignificant: true}
	shouldCheck := manager.ShouldCheckIn(ctx, event)
	if !shouldCheck {
		t.Error("frequency constantly: significant event should trigger check-in")
	}

	// Non-significant events should not trigger
	manager.ResetActionCount()
	event = CollabEvent{Type: "action", IsSignificant: false}
	shouldCheck = manager.ShouldCheckIn(ctx, event)
	if shouldCheck {
		t.Error("frequency constantly: non-significant event should not trigger check-in")
	}
}

func TestResetActionCount(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	event := CollabEvent{Type: "action", IsSignificant: false}

	// Trigger some actions
	for i := 0; i < 5; i++ {
		manager.ShouldCheckIn(ctx, event)
	}

	if manager.GetActionCount() != 5 {
		t.Errorf("expected action count 5, got %d", manager.GetActionCount())
	}

	// Reset
	manager.ResetActionCount()

	if manager.GetActionCount() != 0 {
		t.Errorf("expected action count 0 after reset, got %d", manager.GetActionCount())
	}
}

func TestShouldCheckIn_DefaultDisposition(t *testing.T) {
	// Manager without explicit disposition should use DefaultDisposition
	manager := NewManager()
	ctx := context.Background()

	// DefaultDisposition uses "consultative" + "regularly"
	// Should only check in at decision points, and only when action count is multiple of 7

	// Non-decision event should not trigger
	event := CollabEvent{Type: "action", IsSignificant: true}
	shouldCheck := manager.ShouldCheckIn(ctx, event)
	if shouldCheck {
		t.Error("default disposition: non-decision event should not trigger check-in")
	}
}
