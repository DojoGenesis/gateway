package memory

import (
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/disposition"
)

func TestRetentionPolicy_Surface(t *testing.T) {
	policy := RetentionPolicy("surface")

	if !policy[CategoryDecision] {
		t.Error("surface should retain decisions")
	}
	if !policy[CategoryFinalOutput] {
		t.Error("surface should retain final outputs")
	}
	if policy[CategoryAction] {
		t.Error("surface should not retain actions")
	}
	if policy[CategoryObservation] {
		t.Error("surface should not retain observations")
	}
	if policy[CategoryAlternative] {
		t.Error("surface should not retain alternatives")
	}
	if policy[CategoryReasoning] {
		t.Error("surface should not retain reasoning")
	}
}

func TestRetentionPolicy_Functional(t *testing.T) {
	policy := RetentionPolicy("functional")

	if !policy[CategoryDecision] {
		t.Error("functional should retain decisions")
	}
	if !policy[CategoryAction] {
		t.Error("functional should retain actions")
	}
	if !policy[CategoryObservation] {
		t.Error("functional should retain observations")
	}
	if policy[CategoryAlternative] {
		t.Error("functional should not retain alternatives")
	}
	if policy[CategoryReasoning] {
		t.Error("functional should not retain reasoning")
	}
}

func TestRetentionPolicy_Thorough(t *testing.T) {
	policy := RetentionPolicy("thorough")

	if !policy[CategoryDecision] {
		t.Error("thorough should retain decisions")
	}
	if !policy[CategoryAction] {
		t.Error("thorough should retain actions")
	}
	if !policy[CategoryObservation] {
		t.Error("thorough should retain observations")
	}
	if !policy[CategoryAlternative] {
		t.Error("thorough should retain alternatives")
	}
	if policy[CategoryReasoning] {
		t.Error("thorough should not retain reasoning")
	}
}

func TestRetentionPolicy_Exhaustive(t *testing.T) {
	policy := RetentionPolicy("exhaustive")

	if !policy[CategoryDecision] {
		t.Error("exhaustive should retain decisions")
	}
	if !policy[CategoryAction] {
		t.Error("exhaustive should retain actions")
	}
	if !policy[CategoryObservation] {
		t.Error("exhaustive should retain observations")
	}
	if !policy[CategoryAlternative] {
		t.Error("exhaustive should retain alternatives")
	}
	if !policy[CategoryReasoning] {
		t.Error("exhaustive should retain reasoning")
	}
	if !policy[CategoryFinalOutput] {
		t.Error("exhaustive should retain final outputs")
	}
}

func TestCategorizeMemory_Decision(t *testing.T) {
	mem := Memory{
		Content: "We decided to use PostgreSQL for the database",
		Type:    "message",
	}

	category := CategorizeMemory(mem)
	if category != CategoryDecision {
		t.Errorf("expected decision category, got %v", category)
	}
}

func TestCategorizeMemory_Action(t *testing.T) {
	mem := Memory{
		Content: "Executed the database migration",
		Type:    "tool_call",
	}

	category := CategorizeMemory(mem)
	if category != CategoryAction {
		t.Errorf("expected action category, got %v", category)
	}
}

func TestCategorizeMemory_Alternative(t *testing.T) {
	mem := Memory{
		Content: "Alternatively, we could use MySQL instead",
		Type:    "message",
	}

	category := CategorizeMemory(mem)
	if category != CategoryAlternative {
		t.Errorf("expected alternative category, got %v", category)
	}
}

func TestCategorizeMemory_Reasoning(t *testing.T) {
	mem := Memory{
		Content: "We chose PostgreSQL because it has better JSON support",
		Type:    "message",
	}

	category := CategorizeMemory(mem)
	if category != CategoryReasoning {
		t.Errorf("expected reasoning category, got %v", category)
	}
}

func TestCategorizeMemory_FinalOutput(t *testing.T) {
	mem := Memory{
		Content: "Final result: Migration completed successfully",
		Type:    "message",
	}

	category := CategorizeMemory(mem)
	if category != CategoryFinalOutput {
		t.Errorf("expected final_output category, got %v", category)
	}
}

func TestCompressionStore_Surface(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Depth: "surface",
	}

	store := NewCompressionStore(WithDepthStrategy(disp))

	// Add various memory types
	store.Add(Memory{Content: "decided to use Go", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "ran the tests", Type: "tool_call", CreatedAt: time.Now()})
	store.Add(Memory{Content: "observed high memory usage", Type: "observation", CreatedAt: time.Now()})
	store.Add(Memory{Content: "alternatively could use Rust", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "Final result: System configured successfully", Type: "message", CreatedAt: time.Now()})

	compressed := store.Compress()

	// Surface should only retain decisions and final outputs (2 items)
	if len(compressed) != 2 {
		t.Errorf("surface depth: expected 2 retained memories (decision + final output), got %d", len(compressed))
	}

	// Verify we have both categories
	hasDecision := false
	hasFinalOutput := false
	for _, cm := range compressed {
		if cm.Category == CategoryDecision {
			hasDecision = true
		}
		if cm.Category == CategoryFinalOutput {
			hasFinalOutput = true
		}
	}

	if !hasDecision {
		t.Error("surface depth: missing decision category")
	}
	if !hasFinalOutput {
		t.Error("surface depth: missing final output category")
	}
}

func TestCompressionStore_Functional(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Depth: "functional",
	}

	store := NewCompressionStore(WithDepthStrategy(disp))

	// Add various memory types
	store.Add(Memory{Content: "decided to use Go", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "ran the tests", Type: "tool_call", CreatedAt: time.Now()})
	store.Add(Memory{Content: "observed high memory usage", Type: "observation", CreatedAt: time.Now()})
	store.Add(Memory{Content: "alternatively could use Rust", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "because Go has better concurrency", Type: "message", CreatedAt: time.Now()})

	compressed := store.Compress()

	// Functional should retain decisions, actions, observations
	if len(compressed) != 3 {
		t.Errorf("functional depth: expected 3 retained memories, got %d", len(compressed))
	}
}

func TestCompressionStore_Thorough(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Depth: "thorough",
	}

	store := NewCompressionStore(WithDepthStrategy(disp))

	// Add various memory types
	store.Add(Memory{Content: "decided to use Go", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "ran the tests", Type: "tool_call", CreatedAt: time.Now()})
	store.Add(Memory{Content: "observed high memory usage", Type: "observation", CreatedAt: time.Now()})
	store.Add(Memory{Content: "alternatively could use Rust", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "because Go has better concurrency", Type: "message", CreatedAt: time.Now()})

	compressed := store.Compress()

	// Thorough should retain decisions, actions, observations, alternatives
	if len(compressed) != 4 {
		t.Errorf("thorough depth: expected 4 retained memories, got %d", len(compressed))
	}
}

func TestCompressionStore_Exhaustive(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Depth: "exhaustive",
	}

	store := NewCompressionStore(WithDepthStrategy(disp))

	// Add various memory types
	store.Add(Memory{Content: "decided to use Go", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "ran the tests", Type: "tool_call", CreatedAt: time.Now()})
	store.Add(Memory{Content: "observed high memory usage", Type: "observation", CreatedAt: time.Now()})
	store.Add(Memory{Content: "alternatively could use Rust", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "because Go has better concurrency", Type: "message", CreatedAt: time.Now()})

	compressed := store.Compress()

	// Exhaustive should retain everything
	if len(compressed) != 5 {
		t.Errorf("exhaustive depth: expected 5 retained memories, got %d", len(compressed))
	}
}

func TestCompressionStore_DefaultDisposition(t *testing.T) {
	// Store without explicit disposition should use DefaultDisposition (thorough)
	store := NewCompressionStore()

	store.Add(Memory{Content: "decided to use Go", Type: "message", CreatedAt: time.Now()})
	store.Add(Memory{Content: "ran the tests", Type: "tool_call", CreatedAt: time.Now()})
	store.Add(Memory{Content: "because Go has better concurrency", Type: "message", CreatedAt: time.Now()})

	compressed := store.Compress()

	// Default disposition uses "thorough" depth, which retains decisions, actions, observations, alternatives
	// Should filter out reasoning
	if len(compressed) > 3 {
		t.Errorf("default disposition: expected <= 3 retained memories, got %d", len(compressed))
	}
}
