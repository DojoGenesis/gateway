// Package specialist manages the mapping from intent categories to specialist
// agents. Each specialist is an opinionated agent configuration combining a
// CoworkPlugin, an ADA disposition preset, and a preloaded skill set. The
// registry is the foundation that all other specialist components depend on.
package specialist

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/DojoGenesis/gateway/server/agent"
)

// SpecialistConfig defines a specialist agent's identity and capabilities.
type SpecialistConfig struct {
	Name        string                 // e.g., "researcher", "forger", "scout"
	Plugin      string                 // e.g., "continuous-learning", "skill-forge"
	Disposition string                 // ADA preset name, e.g., "deliberate", "measured", "rapid"
	Skills      []string               // skill names to preload from CAS
	Categories  []agent.IntentCategory // which intents route here
	Confidence  float64                // minimum confidence to route here (default 0.7)
}

// SpecialistRegistry manages the mapping from intent categories to specialist agents.
type SpecialistRegistry struct {
	mu          sync.RWMutex
	specialists map[string]*SpecialistConfig              // name -> config
	categoryMap map[agent.IntentCategory]*SpecialistConfig // category -> config
}

// NewSpecialistRegistry returns an empty, ready-to-use registry.
func NewSpecialistRegistry() *SpecialistRegistry {
	return &SpecialistRegistry{
		specialists: make(map[string]*SpecialistConfig),
		categoryMap: make(map[agent.IntentCategory]*SpecialistConfig),
	}
}

// Register validates and adds a specialist configuration to the registry.
// It returns an error if the name is empty or already registered.
func (r *SpecialistRegistry) Register(config SpecialistConfig) error {
	if config.Name == "" {
		return fmt.Errorf("specialist config name must not be empty")
	}

	// Apply default confidence threshold.
	if config.Confidence <= 0 {
		config.Confidence = 0.7
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.specialists[config.Name]; exists {
		return fmt.Errorf("specialist %q is already registered", config.Name)
	}

	stored := config // copy
	r.specialists[config.Name] = &stored

	for _, cat := range config.Categories {
		if prev, exists := r.categoryMap[cat]; exists {
			slog.Warn("overwriting category mapping",
				"category", cat.String(),
				"previous", prev.Name,
				"new", config.Name,
			)
		}
		r.categoryMap[cat] = &stored
	}

	slog.Info("registered specialist",
		"name", config.Name,
		"plugin", config.Plugin,
		"disposition", config.Disposition,
		"categories", len(config.Categories),
		"confidence", config.Confidence,
	)

	return nil
}

// Lookup returns the specialist config for the given name.
// The second return value is false if no specialist with that name exists.
func (r *SpecialistRegistry) Lookup(name string) (*SpecialistConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, ok := r.specialists[name]
	return cfg, ok
}

// Get is an alias for Lookup, used by SpecialistPool.
func (r *SpecialistRegistry) Get(name string) (*SpecialistConfig, bool) {
	return r.Lookup(name)
}

// RouteToSpecialist returns the specialist mapped to the given intent category,
// but only if the provided confidence meets or exceeds the specialist's threshold.
// Returns nil and false when no specialist matches or confidence is too low.
func (r *SpecialistRegistry) RouteToSpecialist(category agent.IntentCategory, confidence float64) (*SpecialistConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, ok := r.categoryMap[category]
	if !ok {
		return nil, false
	}

	if confidence < cfg.Confidence {
		slog.Debug("confidence below specialist threshold",
			"category", category.String(),
			"specialist", cfg.Name,
			"confidence", confidence,
			"threshold", cfg.Confidence,
		)
		return nil, false
	}

	return cfg, true
}

// List returns all registered specialist configurations.
func (r *SpecialistRegistry) List() []*SpecialistConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*SpecialistConfig, 0, len(r.specialists))
	for _, cfg := range r.specialists {
		result = append(result, cfg)
	}
	return result
}

// DefaultSpecialists returns the seven CoworkPlugin agent definitions mapped
// to intent categories, plus a generalist fallback for anything that falls
// below threshold. Greeting and Calculation are intentionally omitted — they
// use template/fast handlers, not specialist agents.
func DefaultSpecialists() []SpecialistConfig {
	return []SpecialistConfig{
		{
			Name:        "forger",
			Plugin:      "skill-forge",
			Disposition: "rapid",
			Skills:      []string{"skill-creation", "skill-extraction"},
			Categories:  []agent.IntentCategory{agent.CodeGeneration},
			Confidence:  0.7,
		},
		{
			Name:        "researcher",
			Plugin:      "continuous-learning",
			Disposition: "deliberate",
			Skills:      []string{"deep-research", "wide-research"},
			Categories:  []agent.IntentCategory{agent.Debugging},
			Confidence:  0.7,
		},
		{
			Name:        "specifier",
			Plugin:      "specification-driven-development",
			Disposition: "measured",
			Skills:      []string{"release-spec", "frontend-spec"},
			Categories:  []agent.IntentCategory{agent.Planning},
			Confidence:  0.7,
		},
		{
			Name:        "coordinator",
			Plugin:      "agent-orchestration",
			Disposition: "measured",
			Skills:      []string{"agent-dispatch-playbook", "workflow-router"},
			Categories:  []agent.IntentCategory{agent.MetaQuery},
			Confidence:  0.7,
		},
		{
			Name:        "gardener",
			Plugin:      "wisdom-garden",
			Disposition: "measured",
			Skills:      []string{"memory-garden", "seed-extraction"},
			Categories:  []agent.IntentCategory{agent.Explanation},
			Confidence:  0.7,
		},
		{
			Name:        "factual-researcher",
			Plugin:      "continuous-learning",
			Disposition: "responsive",
			Skills:      []string{"deep-research"},
			Categories:  []agent.IntentCategory{agent.Factual},
			Confidence:  0.7,
		},
		// Generalist fallback — no category binding, used when confidence
		// is below threshold for any specialist.
		{
			Name:        "generalist",
			Plugin:      "",
			Disposition: "responsive",
			Skills:      []string{},
			Categories:  nil,
			Confidence:  0.0,
		},
	}
}
