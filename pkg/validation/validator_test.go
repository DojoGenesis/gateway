package validation

import (
	"context"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
)

func TestValidate_None(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Validation: disposition.ValidationConfig{
			Strategy:     "none",
			RequireTests: true,
			RequireDocs:  true,
		},
	}

	validator := NewValidator(WithDisposition(disp))
	ctx := context.Background()

	artifact := Artifact{Type: "code", Language: "go"}
	result, err := validator.Validate(ctx, artifact)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Passed {
		t.Error("none strategy should always pass")
	}

	if len(result.Checks) != 0 {
		t.Errorf("none strategy should run 0 checks, got %d", len(result.Checks))
	}
}

func TestValidate_SpotCheck(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Validation: disposition.ValidationConfig{
			Strategy:     "spot-check",
			RequireTests: false,
			RequireDocs:  false,
		},
	}

	validator := NewValidator(WithDisposition(disp))
	ctx := context.Background()

	artifact := Artifact{Type: "code", Language: "go"}
	result, err := validator.Validate(ctx, artifact)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spot-check without tests should run syntax only
	expectedChecks := 1 // syntax
	if len(result.Checks) != expectedChecks {
		t.Errorf("spot-check (no tests): expected %d checks, got %d", expectedChecks, len(result.Checks))
	}

	// Verify syntax check was run
	found := false
	for _, check := range result.Checks {
		if check.Name == "syntax" {
			found = true
		}
	}
	if !found {
		t.Error("spot-check should include syntax check")
	}
}

func TestValidate_SpotCheck_WithTests(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Validation: disposition.ValidationConfig{
			Strategy:     "spot-check",
			RequireTests: true,
			RequireDocs:  false,
		},
	}

	validator := NewValidator(WithDisposition(disp))
	ctx := context.Background()

	artifact := Artifact{Type: "code", Language: "go"}
	result, err := validator.Validate(ctx, artifact)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spot-check with tests should run syntax + sample tests
	expectedChecks := 2
	if len(result.Checks) != expectedChecks {
		t.Errorf("spot-check (with tests): expected %d checks, got %d", expectedChecks, len(result.Checks))
	}

	// Verify both checks were run
	checkNames := make(map[string]bool)
	for _, check := range result.Checks {
		checkNames[check.Name] = true
	}
	if !checkNames["syntax"] {
		t.Error("spot-check should include syntax check")
	}
	if !checkNames["sample_tests"] {
		t.Error("spot-check with RequireTests should include sample_tests")
	}
}

func TestValidate_Thorough(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Validation: disposition.ValidationConfig{
			Strategy:     "thorough",
			RequireTests: false,
			RequireDocs:  false,
		},
	}

	validator := NewValidator(WithDisposition(disp))
	ctx := context.Background()

	artifact := Artifact{Type: "code", Language: "go"}
	result, err := validator.Validate(ctx, artifact)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Thorough without flags should run syntax + lint + type
	expectedChecks := 3
	if len(result.Checks) != expectedChecks {
		t.Errorf("thorough (no flags): expected %d checks, got %d", expectedChecks, len(result.Checks))
	}
}

func TestValidate_Thorough_WithFlags(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Validation: disposition.ValidationConfig{
			Strategy:     "thorough",
			RequireTests: true,
			RequireDocs:  true,
		},
	}

	validator := NewValidator(WithDisposition(disp))
	ctx := context.Background()

	artifact := Artifact{Type: "code", Language: "go"}
	result, err := validator.Validate(ctx, artifact)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Thorough with all flags should run syntax + lint + type + full_tests + docs
	expectedChecks := 5
	if len(result.Checks) != expectedChecks {
		t.Errorf("thorough (with flags): expected %d checks, got %d", expectedChecks, len(result.Checks))
	}

	// Verify all expected checks
	checkNames := make(map[string]bool)
	for _, check := range result.Checks {
		checkNames[check.Name] = true
	}
	expected := []string{"syntax", "lint", "type", "full_tests", "documentation"}
	for _, name := range expected {
		if !checkNames[name] {
			t.Errorf("thorough with flags should include %s check", name)
		}
	}
}

func TestValidate_Exhaustive(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Validation: disposition.ValidationConfig{
			Strategy:     "exhaustive",
			RequireTests: false,
			RequireDocs:  false,
		},
	}

	validator := NewValidator(WithDisposition(disp))
	ctx := context.Background()

	artifact := Artifact{Type: "code", Language: "go"}
	result, err := validator.Validate(ctx, artifact)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Exhaustive without docs should run 6 checks
	expectedChecks := 6 // syntax, lint, type, full_tests, coverage, security
	if len(result.Checks) != expectedChecks {
		t.Errorf("exhaustive (no docs): expected %d checks, got %d", expectedChecks, len(result.Checks))
	}
}

func TestValidate_Exhaustive_WithDocs(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Validation: disposition.ValidationConfig{
			Strategy:     "exhaustive",
			RequireTests: false,
			RequireDocs:  true,
		},
	}

	validator := NewValidator(WithDisposition(disp))
	ctx := context.Background()

	artifact := Artifact{Type: "code", Language: "go"}
	result, err := validator.Validate(ctx, artifact)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Exhaustive with docs should run 7 checks
	expectedChecks := 7 // syntax, lint, type, full_tests, coverage, security, docs
	if len(result.Checks) != expectedChecks {
		t.Errorf("exhaustive (with docs): expected %d checks, got %d", expectedChecks, len(result.Checks))
	}

	// Verify documentation check was included
	found := false
	for _, check := range result.Checks {
		if check.Name == "documentation" {
			found = true
		}
	}
	if !found {
		t.Error("exhaustive with RequireDocs should include documentation check")
	}
}

func TestValidate_FlagCombinations(t *testing.T) {
	tests := []struct {
		name         string
		strategy     string
		requireTests bool
		requireDocs  bool
		minChecks    int
		maxChecks    int
	}{
		{"spot-check, no flags", "spot-check", false, false, 1, 1},
		{"spot-check, tests only", "spot-check", true, false, 2, 2},
		{"spot-check, docs only (ignored)", "spot-check", false, true, 1, 1},
		{"thorough, no flags", "thorough", false, false, 3, 3},
		{"thorough, tests only", "thorough", true, false, 4, 4},
		{"thorough, docs only", "thorough", false, true, 4, 4},
		{"thorough, both flags", "thorough", true, true, 5, 5},
		{"exhaustive, no flags", "exhaustive", false, false, 6, 6},
		{"exhaustive, docs only", "exhaustive", false, true, 7, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disp := &disposition.DispositionConfig{
				Validation: disposition.ValidationConfig{
					Strategy:     tt.strategy,
					RequireTests: tt.requireTests,
					RequireDocs:  tt.requireDocs,
				},
			}

			validator := NewValidator(WithDisposition(disp))
			ctx := context.Background()

			artifact := Artifact{Type: "code", Language: "go"}
			result, err := validator.Validate(ctx, artifact)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			checkCount := len(result.Checks)
			if checkCount < tt.minChecks || checkCount > tt.maxChecks {
				t.Errorf("%s: expected %d-%d checks, got %d", tt.name, tt.minChecks, tt.maxChecks, checkCount)
			}
		})
	}
}

func TestValidate_DefaultDisposition(t *testing.T) {
	// Validator without explicit disposition should use DefaultDisposition
	validator := NewValidator()
	ctx := context.Background()

	artifact := Artifact{Type: "code", Language: "go"}
	result, err := validator.Validate(ctx, artifact)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// DefaultDisposition uses "thorough" + RequireTests=true
	// Should run: syntax, lint, type, full_tests
	if len(result.Checks) < 4 {
		t.Errorf("default disposition should run at least 4 checks, got %d", len(result.Checks))
	}
}
