package disposition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDisposition_BasicFile(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_basic")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	// Create a basic agent.yaml with new schema
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
  strategy: retry
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
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg, err := ResolveDisposition(workspaceRoot, "")
	if err != nil {
		t.Fatalf("ResolveDisposition failed: %v", err)
	}

	// Verify core dimensions
	if cfg.Pacing != "measured" {
		t.Errorf("Expected pacing 'measured', got '%s'", cfg.Pacing)
	}
	if cfg.Depth != "thorough" {
		t.Errorf("Expected depth 'thorough', got '%s'", cfg.Depth)
	}
	if cfg.Tone != "professional" {
		t.Errorf("Expected tone 'professional', got '%s'", cfg.Tone)
	}
	if cfg.Initiative != "responsive" {
		t.Errorf("Expected initiative 'responsive', got '%s'", cfg.Initiative)
	}

	// Verify nested configs
	if cfg.Validation.Strategy != "thorough" {
		t.Errorf("Expected validation.strategy 'thorough', got '%s'", cfg.Validation.Strategy)
	}
	if cfg.ErrorHandling.RetryCount != 3 {
		t.Errorf("Expected error_handling.retry_count 3, got %d", cfg.ErrorHandling.RetryCount)
	}
}

func TestResolveDisposition_WithMode(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_mode")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	// Create agent.yaml with mode overrides
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
error_handling:
  strategy: retry
  retry_count: 3
collaboration:
  style: consultative
  check_in_frequency: regularly
reflection:
  frequency: session-end
  format: structured
  triggers: []
modes:
  debug:
    pacing: deliberate
    depth: exhaustive
    validation:
      strategy: exhaustive
    error_handling:
      retry_count: 1
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg, err := ResolveDisposition(workspaceRoot, "debug")
	if err != nil {
		t.Fatalf("ResolveDisposition with mode failed: %v", err)
	}

	// Verify mode overrides were applied
	if cfg.Pacing != "deliberate" {
		t.Errorf("Expected pacing 'deliberate' (debug mode), got '%s'", cfg.Pacing)
	}
	if cfg.Depth != "exhaustive" {
		t.Errorf("Expected depth 'exhaustive' (debug mode), got '%s'", cfg.Depth)
	}
	if cfg.Validation.Strategy != "exhaustive" {
		t.Errorf("Expected validation.strategy 'exhaustive' (debug mode), got '%s'", cfg.Validation.Strategy)
	}
	if cfg.ErrorHandling.RetryCount != 1 {
		t.Errorf("Expected retry_count 1 (debug mode), got %d", cfg.ErrorHandling.RetryCount)
	}

	// Verify base values that weren't overridden
	if cfg.Tone != "professional" {
		t.Errorf("Expected tone 'professional' (not overridden), got '%s'", cfg.Tone)
	}
}

func TestResolveDisposition_NonexistentMode(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_badmode")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	// Create agent.yaml without the requested mode
	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: measured
depth: thorough
tone: professional
initiative: responsive
validation:
  strategy: thorough
error_handling:
  strategy: retry
  retry_count: 3
collaboration:
  style: consultative
  check_in_frequency: regularly
reflection:
  frequency: session-end
  format: structured
modes:
  debug:
    pacing: deliberate
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Request a mode that doesn't exist - should use base config
	cfg, err := ResolveDisposition(workspaceRoot, "production")
	if err != nil {
		t.Fatalf("ResolveDisposition should not fail for nonexistent mode: %v", err)
	}

	// Should use base config since mode not found
	if cfg.Pacing != "measured" {
		t.Errorf("Expected base pacing 'measured', got '%s'", cfg.Pacing)
	}
}

func TestResolveDisposition_NoFileUsesDefault(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_nofile")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	// Don't create any file
	cfg, err := ResolveDisposition(workspaceRoot, "")
	if err != nil {
		t.Fatalf("ResolveDisposition should not fail when no file found: %v", err)
	}

	// Should return default disposition
	if cfg.Pacing != "measured" {
		t.Errorf("Expected default pacing 'measured', got '%s'", cfg.Pacing)
	}
	if cfg.Depth != "thorough" {
		t.Errorf("Expected default depth 'thorough', got '%s'", cfg.Depth)
	}
}

func TestResolveDisposition_DispositionWrapper(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_wrapper")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	// Create agent.yaml with disposition: wrapper (bridge format)
	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
disposition:
  pacing: rapid
  depth: functional
  tone: casual
  initiative: autonomous
  validation:
    strategy: spot-check
    require_tests: false
  error_handling:
    strategy: fail-fast
    retry_count: 0
  collaboration:
    style: independent
    check_in_frequency: never
  reflection:
    frequency: never
    format: bullets
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg, err := ResolveDisposition(workspaceRoot, "")
	if err != nil {
		t.Fatalf("ResolveDisposition failed with wrapper format: %v", err)
	}

	// Verify values from within disposition: wrapper
	if cfg.Pacing != "rapid" {
		t.Errorf("Expected pacing 'rapid', got '%s'", cfg.Pacing)
	}
	if cfg.Depth != "functional" {
		t.Errorf("Expected depth 'functional', got '%s'", cfg.Depth)
	}
	if cfg.Tone != "casual" {
		t.Errorf("Expected tone 'casual', got '%s'", cfg.Tone)
	}
}

func TestResolveDisposition_SemverValidation(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_semver")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	tests := []struct {
		name       string
		version    string
		shouldFail bool
	}{
		{"valid_1.0.0", "1.0.0", false},
		{"valid_2.1.3", "2.1.3", false},
		{"valid_with_prerelease", "1.0.0-beta", false},
		{"invalid_no_patch", "1.0", true},
		{"invalid_text", "latest", true},
		{"invalid_v_prefix", "v1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentFile := filepath.Join(workspaceRoot, "agent.yaml")
			content := `
schema_version: "` + tt.version + `"
pacing: measured
depth: thorough
tone: professional
initiative: responsive
validation:
  strategy: thorough
error_handling:
  strategy: retry
  retry_count: 3
collaboration:
  style: consultative
  check_in_frequency: regularly
reflection:
  frequency: session-end
  format: structured
`
			err := os.WriteFile(agentFile, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			_, err = ResolveDisposition(workspaceRoot, "")
			if tt.shouldFail && err == nil {
				t.Errorf("Expected error for invalid semver %q", tt.version)
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Expected valid semver %q to pass, got error: %v", tt.version, err)
			}
		})
	}
}

func TestResolveDisposition_FileSizeLimit(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_large")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	// Create a file larger than 1 MB
	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	largeContent := strings.Repeat("# comment\n", 120000) // > 1 MB (1,200,000 bytes)
	err := os.WriteFile(agentFile, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	_, err = ResolveDisposition(workspaceRoot, "")
	if err == nil {
		t.Error("Expected error for file exceeding 1 MB limit")
	}
	if !strings.Contains(err.Error(), "1 MB") {
		t.Errorf("Expected error to mention 1 MB limit, got: %v", err)
	}
}

func TestResolveDisposition_FileResolutionOrder(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_resolution")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	// Create identity.yaml (should be preferred over agent.yaml)
	identityFile := filepath.Join(workspaceRoot, "identity.yaml")
	err := os.WriteFile(identityFile, []byte(`
schema_version: "1.0.0"
pacing: rapid
depth: functional
tone: casual
initiative: autonomous
validation:
  strategy: none
error_handling:
  strategy: fail-fast
  retry_count: 0
collaboration:
  style: independent
  check_in_frequency: never
reflection:
  frequency: never
  format: bullets
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create identity.yaml: %v", err)
	}

	// Also create agent.yaml (should be ignored)
	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err = os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: deliberate
depth: exhaustive
tone: formal
initiative: reactive
validation:
  strategy: exhaustive
error_handling:
  strategy: escalate
  retry_count: 10
collaboration:
  style: collaborative
  check_in_frequency: constantly
reflection:
  frequency: daily
  format: narrative
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create agent.yaml: %v", err)
	}

	cfg, err := ResolveDisposition(workspaceRoot, "")
	if err != nil {
		t.Fatalf("ResolveDisposition failed: %v", err)
	}

	// Should use identity.yaml values, not agent.yaml
	if cfg.Pacing != "rapid" {
		t.Errorf("Expected pacing 'rapid' from identity.yaml, got '%s'", cfg.Pacing)
	}
	if cfg.Validation.Strategy != "none" {
		t.Errorf("Expected validation strategy 'none' from identity.yaml, got '%s'", cfg.Validation.Strategy)
	}
}

func TestMergeValidation(t *testing.T) {
	base := &ValidationConfig{
		Strategy:     "thorough",
		RequireTests: true,
		RequireDocs:  false,
	}

	override := &ValidationConfig{
		Strategy:     "exhaustive",
		RequireTests: false,
		RequireDocs:  true,
	}

	mergeValidation(base, override)

	if base.Strategy != "exhaustive" {
		t.Errorf("Expected strategy 'exhaustive', got '%s'", base.Strategy)
	}
	if base.RequireTests != false {
		t.Error("Expected require_tests false")
	}
	if base.RequireDocs != true {
		t.Error("Expected require_docs true")
	}
}

func TestMergeErrorHandling(t *testing.T) {
	base := &ErrorHandlingConfig{
		Strategy:   "retry",
		RetryCount: 3,
	}

	override := &ErrorHandlingConfig{
		Strategy:   "fail-fast",
		RetryCount: 0,
	}

	mergeErrorHandling(base, override)

	if base.Strategy != "fail-fast" {
		t.Errorf("Expected strategy 'fail-fast', got '%s'", base.Strategy)
	}
	// Note: RetryCount of 0 is valid, but won't override because > 0 check
	// This is expected behavior for distinguishing zero value from explicit
}

func TestMergeCollaboration(t *testing.T) {
	base := &CollaborationConfig{
		Style:            "consultative",
		CheckInFrequency: "regularly",
	}

	override := &CollaborationConfig{
		Style:            "independent",
		CheckInFrequency: "never",
	}

	mergeCollaboration(base, override)

	if base.Style != "independent" {
		t.Errorf("Expected style 'independent', got '%s'", base.Style)
	}
	if base.CheckInFrequency != "never" {
		t.Errorf("Expected check_in_frequency 'never', got '%s'", base.CheckInFrequency)
	}
}

func TestMergeReflection(t *testing.T) {
	base := &ReflectionConfig{
		Frequency: "session-end",
		Format:    "structured",
		Triggers:  []string{"error"},
	}

	override := &ReflectionConfig{
		Frequency: "daily",
		Format:    "narrative",
		Triggers:  []string{"milestone", "learning"},
	}

	mergeReflection(base, override)

	if base.Frequency != "daily" {
		t.Errorf("Expected frequency 'daily', got '%s'", base.Frequency)
	}
	if base.Format != "narrative" {
		t.Errorf("Expected format 'narrative', got '%s'", base.Format)
	}
	if len(base.Triggers) != 2 {
		t.Errorf("Expected 2 triggers, got %d", len(base.Triggers))
	}
}

func TestResolveDisposition_ActionModeDefault(t *testing.T) {
	workspaceRoot := filepath.Join(os.TempDir(), "test_workspace_action")
	os.MkdirAll(workspaceRoot, 0755)
	defer os.RemoveAll(workspaceRoot)

	// Create agent.yaml with action mode defined
	agentFile := filepath.Join(workspaceRoot, "agent.yaml")
	err := os.WriteFile(agentFile, []byte(`
schema_version: "1.0.0"
pacing: measured
depth: thorough
tone: professional
initiative: responsive
validation:
  strategy: thorough
error_handling:
  strategy: retry
  retry_count: 3
collaboration:
  style: consultative
  check_in_frequency: regularly
reflection:
  frequency: session-end
  format: structured
modes:
  action:
    pacing: rapid
    initiative: proactive
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Call with empty mode - should default to "action" mode
	cfg, err := ResolveDisposition(workspaceRoot, "")
	if err != nil {
		t.Fatalf("ResolveDisposition failed: %v", err)
	}

	// Should have action mode overrides applied
	if cfg.Pacing != "rapid" {
		t.Errorf("Expected pacing 'rapid' from action mode, got '%s'", cfg.Pacing)
	}
	if cfg.Initiative != "proactive" {
		t.Errorf("Expected initiative 'proactive' from action mode, got '%s'", cfg.Initiative)
	}
}
