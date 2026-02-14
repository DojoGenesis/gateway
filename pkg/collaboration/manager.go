package collaboration

import (
	"context"
	"strings"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
)

// CollabEvent represents an event that might trigger a check-in.
type CollabEvent struct {
	Type          string // "action", "decision_point", "major_milestone", "agent_handoff"
	IsSignificant bool   // Whether this event is significant enough to warrant check-in
	Description   string // Human-readable description of the event
	Metadata      map[string]interface{}
}

// Manager decides when to check in with the user based on disposition.Collaboration.
//
// Per Gateway-ADA Contract §3.5:
//
// Style:
//   - independent: Complete tasks without check-ins
//   - consultative: Check in at decision points
//   - collaborative: Frequent dialogue with user
//   - delegating: Coordinate with other agents
//
// CheckInFrequency:
//   - never: No automatic check-ins
//   - rarely: At major milestones
//   - regularly: Every 5-10 actions
//   - constantly: Before significant actions
type Manager struct {
	disp        *disposition.DispositionConfig
	actionCount int
}

// ManagerOption is a functional option for configuring the Manager.
type ManagerOption func(*Manager)

// WithDisposition sets the disposition configuration.
func WithDisposition(disp *disposition.DispositionConfig) ManagerOption {
	return func(m *Manager) {
		m.disp = disp
	}
}

// NewManager creates a new collaboration manager.
func NewManager(opts ...ManagerOption) *Manager {
	m := &Manager{
		disp:        disposition.DefaultDisposition(),
		actionCount: 0,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ShouldCheckIn decides whether to check in with the user at this event.
// This implements two-layer filtering: style then frequency.
func (m *Manager) ShouldCheckIn(ctx context.Context, event CollabEvent) bool {
	if m.disp == nil {
		return false
	}

	// Increment action counter for frequency tracking
	m.actionCount++

	// Layer 1: Style-based filtering
	style := strings.ToLower(m.disp.Collaboration.Style)

	switch style {
	case "independent":
		// Never check in regardless of frequency
		return false

	case "consultative":
		// Only check in at decision points (style filter)
		if event.Type != "decision_point" {
			return false
		}
		// Fall through to frequency check

	case "collaborative":
		// Check frequency below (allow most events)
		// Fall through to frequency check

	case "delegating":
		// Only check in for agent handoffs or decision points
		if event.Type != "agent_handoff" && event.Type != "decision_point" {
			return false
		}
		// Fall through to frequency check

	default:
		// Unknown style - default to consultative
		if event.Type != "decision_point" {
			return false
		}
	}

	// Layer 2: Frequency-based filtering
	frequency := strings.ToLower(m.disp.Collaboration.CheckInFrequency)

	switch frequency {
	case "never":
		return false

	case "rarely":
		// Only at major milestones
		return event.Type == "major_milestone"

	case "regularly":
		// Every ~7 actions (using 7 as midpoint of 5-10)
		return m.actionCount%7 == 0

	case "constantly":
		// Before all significant actions
		return event.IsSignificant

	default:
		// Unknown frequency - default to "regularly"
		return m.actionCount%7 == 0
	}
}

// ResetActionCount resets the action counter (e.g., at session start).
func (m *Manager) ResetActionCount() {
	m.actionCount = 0
}

// GetActionCount returns the current action count.
func (m *Manager) GetActionCount() int {
	return m.actionCount
}
