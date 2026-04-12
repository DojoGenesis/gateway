package specialist

import (
	"testing"

	"github.com/DojoGenesis/gateway/server/agent"
)

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestNewSpecialistRegistry(t *testing.T) {
	r := NewSpecialistRegistry()
	if r == nil {
		t.Fatal("NewSpecialistRegistry returned nil")
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewSpecialistRegistry()
	cfg := SpecialistConfig{
		Name:       "test-specialist",
		Plugin:     "test-plugin",
		Confidence: 0.8,
	}
	if err := r.Register(cfg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	got, ok := r.Lookup("test-specialist")
	if !ok {
		t.Fatal("Lookup returned false after Register")
	}
	if got.Name != "test-specialist" {
		t.Errorf("expected name %q, got %q", "test-specialist", got.Name)
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	r := NewSpecialistRegistry()
	cfg := SpecialistConfig{Name: "dup", Confidence: 0.7}
	if err := r.Register(cfg); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}
	if err := r.Register(cfg); err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}
}

func TestRegistryRegisterEmptyName(t *testing.T) {
	r := NewSpecialistRegistry()
	cfg := SpecialistConfig{Name: "", Confidence: 0.7}
	if err := r.Register(cfg); err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestRegistryRegisterDefaultConfidence(t *testing.T) {
	r := NewSpecialistRegistry()
	cfg := SpecialistConfig{Name: "zero-conf", Confidence: 0}
	if err := r.Register(cfg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	got, ok := r.Lookup("zero-conf")
	if !ok {
		t.Fatal("Lookup returned false")
	}
	if got.Confidence != 0.7 {
		t.Errorf("expected default confidence 0.7, got %f", got.Confidence)
	}
}

func TestRegistryRouteToSpecialist(t *testing.T) {
	r := NewSpecialistRegistry()
	cfg := SpecialistConfig{
		Name:       "coder",
		Categories: []agent.IntentCategory{agent.CodeGeneration},
		Confidence: 0.7,
	}
	if err := r.Register(cfg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	got, ok := r.RouteToSpecialist(agent.CodeGeneration, 0.9)
	if !ok {
		t.Fatal("RouteToSpecialist returned false for confidence above threshold")
	}
	if got == nil {
		t.Fatal("RouteToSpecialist returned nil config")
	}
	if got.Name != "coder" {
		t.Errorf("expected name %q, got %q", "coder", got.Name)
	}
}

func TestRegistryRouteToSpecialistBelowThreshold(t *testing.T) {
	r := NewSpecialistRegistry()
	cfg := SpecialistConfig{
		Name:       "coder",
		Categories: []agent.IntentCategory{agent.CodeGeneration},
		Confidence: 0.8,
	}
	if err := r.Register(cfg); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	got, ok := r.RouteToSpecialist(agent.CodeGeneration, 0.5)
	if ok || got != nil {
		t.Fatal("expected RouteToSpecialist to return false below threshold")
	}
}

func TestRegistryRouteToSpecialistUnknownCategory(t *testing.T) {
	r := NewSpecialistRegistry()
	got, ok := r.RouteToSpecialist(agent.Planning, 0.9)
	if ok || got != nil {
		t.Fatal("expected nil, false for unknown category")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewSpecialistRegistry()
	names := []string{"alpha", "beta", "gamma"}
	for _, n := range names {
		if err := r.Register(SpecialistConfig{Name: n, Confidence: 0.7}); err != nil {
			t.Fatalf("Register(%q) failed: %v", n, err)
		}
	}
	list := r.List()
	if len(list) != len(names) {
		t.Errorf("List returned %d entries, want %d", len(list), len(names))
	}
}

func TestDefaultSpecialists(t *testing.T) {
	defaults := DefaultSpecialists()
	if len(defaults) != 7 {
		t.Errorf("DefaultSpecialists returned %d entries, want 7", len(defaults))
	}
	for i, d := range defaults {
		if d.Name == "" {
			t.Errorf("DefaultSpecialists[%d] has empty Name", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Pool tests
// ---------------------------------------------------------------------------

func newFilledRegistry(t *testing.T) *SpecialistRegistry {
	t.Helper()
	r := NewSpecialistRegistry()
	configs := []SpecialistConfig{
		{
			Name:       "coder",
			Categories: []agent.IntentCategory{agent.CodeGeneration},
			Confidence: 0.7,
		},
		{
			Name:       "planner",
			Categories: []agent.IntentCategory{agent.Planning},
			Confidence: 0.7,
		},
	}
	for _, cfg := range configs {
		if err := r.Register(cfg); err != nil {
			t.Fatalf("Register(%q) failed: %v", cfg.Name, err)
		}
	}
	return r
}

func TestNewSpecialistPool(t *testing.T) {
	r := NewSpecialistRegistry()
	p := NewSpecialistPool(r)
	if p == nil {
		t.Fatal("NewSpecialistPool returned nil")
	}
}

func TestPoolInitializeAll(t *testing.T) {
	r := newFilledRegistry(t)
	p := NewSpecialistPool(r)
	if err := p.InitializeAll(); err != nil {
		t.Fatalf("InitializeAll failed: %v", err)
	}
	// Both registered specialists should be present with non-empty IDs.
	for _, name := range []string{"coder", "planner"} {
		sa, ok := p.Get(name)
		if !ok {
			t.Fatalf("Get(%q) returned false after InitializeAll", name)
		}
		if sa.AgentID == "" {
			t.Errorf("specialist %q has empty AgentID", name)
		}
		if sa.SessionID == "" {
			t.Errorf("specialist %q has empty SessionID", name)
		}
	}
}

func TestPoolInitializeAllEmpty(t *testing.T) {
	r := NewSpecialistRegistry()
	p := NewSpecialistPool(r)
	if err := p.InitializeAll(); err != nil {
		t.Fatalf("InitializeAll on empty registry returned error: %v", err)
	}
}

func TestPoolGet(t *testing.T) {
	r := newFilledRegistry(t)
	p := NewSpecialistPool(r)
	if err := p.InitializeAll(); err != nil {
		t.Fatalf("InitializeAll failed: %v", err)
	}
	sa, ok := p.Get("coder")
	if !ok {
		t.Fatal("Get(\"coder\") returned false")
	}
	if sa == nil {
		t.Fatal("Get(\"coder\") returned nil agent")
	}
}

func TestPoolGetUnknown(t *testing.T) {
	r := NewSpecialistRegistry()
	p := NewSpecialistPool(r)
	sa, ok := p.Get("nobody")
	if ok || sa != nil {
		t.Fatal("Get of unknown specialist should return nil, false")
	}
}

func TestPoolStats(t *testing.T) {
	r := newFilledRegistry(t)
	p := NewSpecialistPool(r)
	if err := p.InitializeAll(); err != nil {
		t.Fatalf("InitializeAll failed: %v", err)
	}
	stats := p.Stats()
	if stats.TotalSpecialists != 2 {
		t.Errorf("Stats.TotalSpecialists = %d, want 2", stats.TotalSpecialists)
	}
}

// ---------------------------------------------------------------------------
// Router tests
// ---------------------------------------------------------------------------

func newRouterSetup(t *testing.T) (*SpecialistPool, *Router) {
	t.Helper()
	r := newFilledRegistry(t)
	p := NewSpecialistPool(r)
	if err := p.InitializeAll(); err != nil {
		t.Fatalf("InitializeAll failed: %v", err)
	}
	router := NewRouter(p)
	return p, router
}

func TestNewRouter(t *testing.T) {
	r := NewSpecialistRegistry()
	p := NewSpecialistPool(r)
	router := NewRouter(p)
	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
	if router.defaultThreshold != 0.7 {
		t.Errorf("expected default threshold 0.7, got %f", router.defaultThreshold)
	}
}

func TestRouterWithThreshold(t *testing.T) {
	r := NewSpecialistRegistry()
	p := NewSpecialistPool(r)
	router := NewRouter(p, WithThreshold(0.5))
	if router.defaultThreshold != 0.5 {
		t.Errorf("expected threshold 0.5, got %f", router.defaultThreshold)
	}
}

func TestRouterRoute(t *testing.T) {
	_, router := newRouterSetup(t)
	decision := agent.RoutingDecision{
		Category:   agent.CodeGeneration,
		Confidence: 0.9,
	}
	result := router.Route(decision)
	if !result.Routed {
		t.Errorf("expected Routed=true, reason: %s", result.Reason)
	}
	if result.Specialist == nil {
		t.Fatal("expected non-nil Specialist")
	}
	if result.SpecialistID == "" {
		t.Error("expected non-empty SpecialistID")
	}
}

func TestRouterRouteBelowThreshold(t *testing.T) {
	_, router := newRouterSetup(t)
	decision := agent.RoutingDecision{
		Category:   agent.CodeGeneration,
		Confidence: 0.3,
	}
	result := router.Route(decision)
	if result.Routed {
		t.Errorf("expected Routed=false for confidence 0.3 below threshold 0.7")
	}
}

func TestRouterRouteNoSpecialist(t *testing.T) {
	_, router := newRouterSetup(t)
	// Debugging has no specialist registered in newFilledRegistry.
	decision := agent.RoutingDecision{
		Category:   agent.Debugging,
		Confidence: 0.95,
	}
	result := router.Route(decision)
	if result.Routed {
		t.Errorf("expected Routed=false for unregistered category Debugging")
	}
}

func TestRouterRouteWithOverride(t *testing.T) {
	_, router := newRouterSetup(t)
	result, err := router.RouteWithOverride("coder")
	if err != nil {
		t.Fatalf("RouteWithOverride(\"coder\") returned error: %v", err)
	}
	if !result.Routed {
		t.Errorf("expected Routed=true, reason: %s", result.Reason)
	}
	if result.Specialist == nil {
		t.Fatal("expected non-nil Specialist")
	}
}

func TestRouterRouteWithOverrideUnknown(t *testing.T) {
	_, router := newRouterSetup(t)
	_, err := router.RouteWithOverride("does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown specialist override, got nil")
	}
}
