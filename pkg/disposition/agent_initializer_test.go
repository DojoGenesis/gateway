package disposition

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/disposition"
)

func TestAgentInitializer_Initialize(t *testing.T) {
	initializer := NewAgentInitializer(5 * time.Minute)
	ctx := context.Background()

	// Create a temporary workspace with agent.yaml
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace")
	if err := os.MkdirAll(workspaceRoot, 0750); err != nil {
		t.Fatalf("failed to create test workspace: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspaceRoot) // Cleanup best-effort
	}()

	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: measured
depth: thorough
tone: professional
initiative: responsive
validation:
  strategy: thorough
  require_tests: true
  require_docs: false
error_handling:
  strategy: log-and-continue
  retry_count: 3
collaboration:
  style: consultative
  check_in_frequency: regularly
reflection:
  frequency: session-end
  format: structured
  triggers:
    - error
    - milestone
`), 0600)
	if err != nil {
		t.Fatalf("Failed to create test agent.yaml: %v", err)
	}

	agentConfig, err := initializer.Initialize(ctx, workspaceRoot, "")
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if agentConfig.Pacing != "measured" {
		t.Errorf("Expected pacing 'measured', got '%s'", agentConfig.Pacing)
	}
	if agentConfig.Depth != "thorough" {
		t.Errorf("Expected depth 'thorough', got '%s'", agentConfig.Depth)
	}
	if agentConfig.Tone != "professional" {
		t.Errorf("Expected tone 'professional', got '%s'", agentConfig.Tone)
	}
	if agentConfig.Initiative != "responsive" {
		t.Errorf("Expected initiative 'responsive', got '%s'", agentConfig.Initiative)
	}
}

func TestAgentInitializer_CacheHit(t *testing.T) {
	initializer := NewAgentInitializer(5 * time.Minute)
	ctx := context.Background()

	// Create a temporary workspace
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_cache")
	if err := os.MkdirAll(workspaceRoot, 0750); err != nil {
		t.Fatalf("failed to create test workspace: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspaceRoot) // Cleanup best-effort
	}()

	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: deliberate
depth: surface
tone: formal
initiative: reactive
validation:
  strategy: none
  require_tests: false
  require_docs: false
error_handling:
  strategy: fail-fast
  retry_count: 0
collaboration:
  style: independent
  check_in_frequency: never
reflection:
  frequency: never
  format: structured
  triggers: []
`), 0600)
	if err != nil {
		t.Fatalf("Failed to create test agent.yaml: %v", err)
	}

	// First call - should load from disk
	_, err = initializer.Initialize(ctx, workspaceRoot, "")
	if err != nil {
		t.Fatalf("First Initialize failed: %v", err)
	}

	// Modify the file on disk
	err = os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: rapid
depth: exhaustive
tone: casual
initiative: autonomous
validation:
  strategy: exhaustive
  require_tests: true
  require_docs: true
error_handling:
  strategy: escalate
  retry_count: 10
collaboration:
  style: delegating
  check_in_frequency: constantly
reflection:
  frequency: daily
  format: narrative
  triggers:
    - error
`), 0600)
	if err != nil {
		t.Fatalf("Failed to modify agent.yaml: %v", err)
	}

	// Second call - should return cached value (not the modified one)
	agentConfig, err := initializer.Initialize(ctx, workspaceRoot, "")
	if err != nil {
		t.Fatalf("Second Initialize failed: %v", err)
	}

	if agentConfig.Pacing != "deliberate" {
		t.Errorf("Expected cached pacing 'deliberate', got '%s' (cache miss?)", agentConfig.Pacing)
	}
}

func TestAgentInitializer_ClearCache(t *testing.T) {
	initializer := NewAgentInitializer(5 * time.Minute)
	ctx := context.Background()

	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_clear")
	if err := os.MkdirAll(workspaceRoot, 0750); err != nil {
		t.Fatalf("failed to create test workspace: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspaceRoot) // Cleanup best-effort
	}()

	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: measured
depth: thorough
tone: professional
initiative: responsive
validation:
  strategy: thorough
  require_tests: true
  require_docs: false
error_handling:
  strategy: log-and-continue
  retry_count: 3
collaboration:
  style: consultative
  check_in_frequency: regularly
reflection:
  frequency: session-end
  format: structured
  triggers:
    - error
`), 0600)
	if err != nil {
		t.Fatalf("Failed to create test agent.yaml: %v", err)
	}

	// First call - load into cache
	_, err = initializer.Initialize(ctx, workspaceRoot, "")
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Clear cache
	initializer.ClearCache()

	// Modify file
	err = os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: rapid
depth: functional
tone: conversational
initiative: proactive
validation:
  strategy: spot-check
  require_tests: false
  require_docs: false
error_handling:
  strategy: retry
  retry_count: 5
collaboration:
  style: collaborative
  check_in_frequency: rarely
reflection:
  frequency: weekly
  format: bullets
  triggers:
    - milestone
`), 0600)
	if err != nil {
		t.Fatalf("Failed to modify agent.yaml: %v", err)
	}

	// Second call - should reload from disk
	agentConfig, err := initializer.Initialize(ctx, workspaceRoot, "")
	if err != nil {
		t.Fatalf("Initialize after clear failed: %v", err)
	}

	if agentConfig.Pacing != "rapid" {
		t.Errorf("Expected reloaded pacing 'rapid', got '%s'", agentConfig.Pacing)
	}
	if agentConfig.Depth != "functional" {
		t.Errorf("Expected reloaded depth 'functional', got '%s'", agentConfig.Depth)
	}
}

func TestAgentInitializer_DifferentModes(t *testing.T) {
	initializer := NewAgentInitializer(5 * time.Minute)
	ctx := context.Background()

	// Create workspace with modes
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_modes")
	if err := os.MkdirAll(workspaceRoot, 0750); err != nil {
		t.Fatalf("failed to create test workspace: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(workspaceRoot) // Cleanup best-effort
	}()

	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: measured
depth: thorough
tone: professional
initiative: responsive
validation:
  strategy: thorough
  require_tests: true
  require_docs: false
error_handling:
  strategy: log-and-continue
  retry_count: 3
collaboration:
  style: consultative
  check_in_frequency: regularly
reflection:
  frequency: session-end
  format: structured
  triggers:
    - error
modes:
  debug:
    pacing: deliberate
    depth: exhaustive
    validation:
      strategy: exhaustive
    error_handling:
      retry_count: 1
  prod:
    pacing: rapid
    depth: functional
    validation:
      strategy: spot-check
    error_handling:
      retry_count: 5
`), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Initialize with debug mode
	debugConfig, err := initializer.Initialize(ctx, workspaceRoot, "debug")
	if err != nil {
		t.Fatalf("Initialize with debug mode failed: %v", err)
	}

	// Initialize with prod mode
	prodConfig, err := initializer.Initialize(ctx, workspaceRoot, "prod")
	if err != nil {
		t.Fatalf("Initialize with prod mode failed: %v", err)
	}

	// The configs should have different settings due to mode overrides
	if debugConfig.Pacing != "deliberate" {
		t.Errorf("Expected debug pacing 'deliberate', got '%s'", debugConfig.Pacing)
	}
	if prodConfig.Pacing != "rapid" {
		t.Errorf("Expected prod pacing 'rapid', got '%s'", prodConfig.Pacing)
	}
	if debugConfig.Depth != "exhaustive" {
		t.Errorf("Expected debug depth 'exhaustive', got '%s'", debugConfig.Depth)
	}
	if prodConfig.Depth != "functional" {
		t.Errorf("Expected prod depth 'functional', got '%s'", prodConfig.Depth)
	}
	if debugConfig.ErrorHandling.RetryCount == prodConfig.ErrorHandling.RetryCount {
		t.Error("Expected different retry_count for debug vs prod modes")
	}
}

func TestConvertToAgentConfig(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Pacing:     "measured",
		Depth:      "thorough",
		Tone:       "professional",
		Initiative: "responsive",
		Validation: disposition.ValidationConfig{
			Strategy:     "thorough",
			RequireTests: true,
			RequireDocs:  false,
		},
		ErrorHandling: disposition.ErrorHandlingConfig{
			Strategy:   "log-and-continue",
			RetryCount: 3,
		},
		Collaboration: disposition.CollaborationConfig{
			Style:            "consultative",
			CheckInFrequency: "regularly",
		},
		Reflection: disposition.ReflectionConfig{
			Frequency: "session-end",
			Format:    "structured",
			Triggers:  []string{"error", "milestone"},
		},
	}

	agentConfig := convertToAgentConfig(disp)

	if agentConfig.Pacing != "measured" {
		t.Errorf("Expected pacing 'measured', got '%s'", agentConfig.Pacing)
	}
	if agentConfig.Depth != "thorough" {
		t.Errorf("Expected depth 'thorough', got '%s'", agentConfig.Depth)
	}
	if agentConfig.Tone != "professional" {
		t.Errorf("Expected tone 'professional', got '%s'", agentConfig.Tone)
	}
	if agentConfig.Initiative != "responsive" {
		t.Errorf("Expected initiative 'responsive', got '%s'", agentConfig.Initiative)
	}
	if agentConfig.Validation.Strategy != "thorough" {
		t.Errorf("Expected validation strategy 'thorough', got '%s'", agentConfig.Validation.Strategy)
	}
}
