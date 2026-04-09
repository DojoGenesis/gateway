package specialist

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/DojoGenesis/gateway/server/agent"
)

// defaultConfidenceThreshold is the minimum confidence score required
// for a routing decision to be accepted. Below this, the request falls
// through to the general-purpose agent.
const defaultConfidenceThreshold = 0.7

// Router handles the decision logic for routing requests to specialist agents.
// It sits between the intent classifier and the specialist pool, applying
// confidence thresholds and logging routing decisions.
type Router struct {
	pool             *SpecialistPool
	defaultThreshold float64
	logger           *slog.Logger
}

// RoutingResult captures the outcome of a specialist routing decision.
type RoutingResult struct {
	Specialist   *SpecialistAgent // nil if no specialist matched
	Routed       bool             // true if a specialist was selected
	SpecialistID string           // agent ID of the selected specialist (empty if not routed)
	Reason       string           // human-readable routing reason
}

// RouterOption configures a Router via the functional options pattern.
type RouterOption func(*Router)

// WithThreshold overrides the default confidence threshold (0.7).
// Values are clamped to the [0.0, 1.0] range.
func WithThreshold(t float64) RouterOption {
	return func(r *Router) {
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		r.defaultThreshold = t
	}
}

// WithLogger sets a custom slog.Logger for routing diagnostics.
func WithLogger(l *slog.Logger) RouterOption {
	return func(r *Router) {
		if l != nil {
			r.logger = l
		}
	}
}

// NewRouter creates a Router backed by the given SpecialistPool.
// Options override default configuration (threshold = 0.7, logger = slog.Default()).
func NewRouter(pool *SpecialistPool, opts ...RouterOption) *Router {
	r := &Router{
		pool:             pool,
		defaultThreshold: defaultConfidenceThreshold,
		logger:           slog.Default(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Route takes a RoutingDecision from the intent classifier and returns a
// RoutingResult. The routing logic:
//
//  1. If decision.Confidence < router threshold, reject.
//  2. Look up a specialist via pool.RouteAndGet (which also checks the
//     specialist's own confidence requirement and updates usage tracking).
//  3. If no specialist registered for the category, reject.
//  4. Otherwise, return the matched specialist.
//
// Every decision is logged at Info level for observability.
func (r *Router) Route(decision agent.RoutingDecision) RoutingResult {
	// Gate: confidence below the router-level threshold.
	if decision.Confidence < r.defaultThreshold {
		result := RoutingResult{
			Routed: false,
			Reason: fmt.Sprintf(
				"confidence %.2f below threshold %.2f",
				decision.Confidence, r.defaultThreshold,
			),
		}
		r.logger.Info("specialist routing rejected",
			"category", decision.Category.String(),
			"confidence", decision.Confidence,
			"threshold", r.defaultThreshold,
			"reason", result.Reason,
		)
		return result
	}

	// Pool lookup: finds the best specialist whose own confidence
	// requirement is also satisfied. RouteAndGet atomically updates
	// LastUsed and TaskCount on the matched agent.
	specialist, ok := r.pool.RouteAndGet(decision.Category, decision.Confidence)
	if !ok || specialist == nil {
		result := RoutingResult{
			Routed: false,
			Reason: fmt.Sprintf(
				"no specialist registered for category %s",
				decision.Category.String(),
			),
		}
		r.logger.Info("specialist routing: no match",
			"category", decision.Category.String(),
			"confidence", decision.Confidence,
			"reason", result.Reason,
		)
		return result
	}

	result := RoutingResult{
		Specialist:   specialist,
		Routed:       true,
		SpecialistID: specialist.AgentID,
		Reason: fmt.Sprintf(
			"routed to specialist %q for category %s (confidence %.2f)",
			specialist.Config.Name, decision.Category.String(), decision.Confidence,
		),
	}

	r.logger.Info("specialist routing accepted",
		"specialist", specialist.Config.Name,
		"agent_id", specialist.AgentID,
		"category", decision.Category.String(),
		"confidence", decision.Confidence,
		"task_count", atomic.LoadInt64(&specialist.TaskCount),
	)

	return result
}

// RouteWithOverride selects a specialist by name, bypassing the intent
// classifier entirely. This supports explicit routing where the caller
// (or user) names a specific specialist.
//
// Returns an error if no specialist with the given name exists in the pool.
func (r *Router) RouteWithOverride(specialistName string) (RoutingResult, error) {
	specialist, ok := r.pool.Get(specialistName)
	if !ok || specialist == nil {
		// Try creating on demand via GetOrCreate if registered but not pooled.
		var err error
		specialist, err = r.pool.GetOrCreate(specialistName)
		if err != nil {
			r.logger.Warn("specialist override: not found",
				"requested", specialistName,
				"error", err,
			)
			return RoutingResult{
				Routed: false,
				Reason: fmt.Sprintf("specialist %q not found in pool", specialistName),
			}, fmt.Errorf("specialist %q not found in pool: %w", specialistName, err)
		}
	}

	// Update usage tracking.
	now := time.Now()
	atomic.AddInt64(&specialist.TaskCount, 1)
	r.pool.mu.Lock()
	specialist.LastUsed = now
	r.pool.mu.Unlock()

	result := RoutingResult{
		Specialist:   specialist,
		Routed:       true,
		SpecialistID: specialist.AgentID,
		Reason: fmt.Sprintf(
			"explicit override to specialist %q (agent %s)",
			specialistName, specialist.AgentID,
		),
	}

	r.logger.Info("specialist routing override",
		"specialist", specialistName,
		"agent_id", specialist.AgentID,
		"task_count", atomic.LoadInt64(&specialist.TaskCount),
	)

	return result, nil
}
