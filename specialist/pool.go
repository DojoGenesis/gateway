package specialist

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DojoGenesis/gateway/server/agent"
	"github.com/google/uuid"
)

// SpecialistAgent represents a pre-warmed specialist agent instance.
// Agents are long-lived — created once at startup, reused per request.
type SpecialistAgent struct {
	Config    *SpecialistConfig
	AgentID   string    // unique agent identifier (UUID)
	SessionID string    // persistent session for this specialist
	CreatedAt time.Time
	LastUsed  time.Time
	TaskCount int64 // number of tasks routed to this specialist
}

// PoolStats contains pool statistics for observability.
type PoolStats struct {
	TotalSpecialists  int   `json:"total_specialists"`
	ActiveSpecialists int   `json:"active_specialists"` // agents with TaskCount > 0
	TotalTasks        int64 `json:"total_tasks"`
}

// SpecialistPool manages pre-created specialist agent sessions.
// Agents are long-lived — created once at startup, reused per request.
type SpecialistPool struct {
	mu       sync.RWMutex
	agents   map[string]*SpecialistAgent // specialist name → agent
	registry *SpecialistRegistry
	logger   *slog.Logger
}

// NewSpecialistPool creates a new pool backed by the given registry.
func NewSpecialistPool(registry *SpecialistRegistry) *SpecialistPool {
	return &SpecialistPool{
		agents:   make(map[string]*SpecialistAgent),
		registry: registry,
		logger:   slog.Default().With("component", "specialist-pool"),
	}
}

// InitializeAll creates agent instances for all registered specialists.
// Called once at startup. Each specialist gets a UUID-based AgentID and SessionID.
func (p *SpecialistPool) InitializeAll() error {
	configs := p.registry.List()
	if len(configs) == 0 {
		p.logger.Info("no specialists registered, pool is empty")
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for _, cfg := range configs {
		if _, exists := p.agents[cfg.Name]; exists {
			p.logger.Warn("specialist already initialized, skipping",
				"name", cfg.Name,
			)
			continue
		}

		sa, err := newSpecialistAgent(cfg)
		if err != nil {
			p.logger.Error("failed to initialize specialist",
				"name", cfg.Name,
				"error", err,
			)
			errs = append(errs, fmt.Errorf("init %s: %w", cfg.Name, err))
			continue
		}

		p.agents[cfg.Name] = sa
		p.logger.Info("specialist agent initialized",
			"name", cfg.Name,
			"agent_id", sa.AgentID,
			"session_id", sa.SessionID,
			"plugin", cfg.Plugin,
			"skills", len(cfg.Skills),
		)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to initialize %d/%d specialists", len(errs), len(configs))
	}

	p.logger.Info("specialist pool initialized",
		"total", len(p.agents),
	)
	return nil
}

// GetOrCreate returns an existing specialist agent, or creates one if missing.
// The specialist must be registered in the registry.
func (p *SpecialistPool) GetOrCreate(name string) (*SpecialistAgent, error) {
	// Fast path: read lock check.
	p.mu.RLock()
	if sa, ok := p.agents[name]; ok {
		p.mu.RUnlock()
		return sa, nil
	}
	p.mu.RUnlock()

	// Slow path: look up config in registry and create.
	cfg, ok := p.registry.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("specialist %q not found in registry", name)
	}

	sa, err := newSpecialistAgent(cfg)
	if err != nil {
		return nil, fmt.Errorf("create specialist %q: %w", name, err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have created it).
	if existing, ok := p.agents[name]; ok {
		return existing, nil
	}

	p.agents[name] = sa
	p.logger.Info("specialist agent created on demand",
		"name", name,
		"agent_id", sa.AgentID,
	)

	return sa, nil
}

// Get returns a specialist agent by name, or nil if not initialized.
func (p *SpecialistPool) Get(name string) (*SpecialistAgent, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	sa, ok := p.agents[name]
	return sa, ok
}

// RouteAndGet combines registry lookup with pool retrieval.
// Given an IntentCategory + confidence, finds the right specialist and returns its agent.
// Returns nil, false if no specialist matches or confidence is below threshold.
func (p *SpecialistPool) RouteAndGet(category agent.IntentCategory, confidence float64) (*SpecialistAgent, bool) {
	cfg, ok := p.registry.RouteToSpecialist(category, confidence)
	if !ok || cfg == nil {
		return nil, false
	}

	sa, ok := p.Get(cfg.Name)
	if !ok {
		// Specialist registered but not yet in pool — try creating it.
		created, err := p.GetOrCreate(cfg.Name)
		if err != nil {
			p.logger.Error("failed to create routed specialist",
				"name", cfg.Name,
				"category", category.String(),
				"error", err,
			)
			return nil, false
		}
		sa = created
	}

	// Record usage atomically.
	now := time.Now()
	atomic.AddInt64(&sa.TaskCount, 1)
	// LastUsed update is best-effort under the read lock.
	p.mu.Lock()
	sa.LastUsed = now
	p.mu.Unlock()

	return sa, true
}

// Stats returns pool statistics for observability.
func (p *SpecialistPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalSpecialists: len(p.agents),
	}

	for _, sa := range p.agents {
		tasks := atomic.LoadInt64(&sa.TaskCount)
		stats.TotalTasks += tasks
		if tasks > 0 {
			stats.ActiveSpecialists++
		}
	}

	return stats
}

// newSpecialistAgent creates a new SpecialistAgent with generated IDs.
func newSpecialistAgent(cfg *SpecialistConfig) (*SpecialistAgent, error) {
	agentID := uuid.New().String()
	sessionID := uuid.New().String()

	now := time.Now()
	return &SpecialistAgent{
		Config:    cfg,
		AgentID:   agentID,
		SessionID: sessionID,
		CreatedAt: now,
		LastUsed:  now,
		TaskCount: 0,
	}, nil
}
