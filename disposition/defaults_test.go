package disposition

import "testing"

func TestDefaultDisposition(t *testing.T) {
	cfg := DefaultDisposition()
	if cfg == nil {
		t.Fatal("DefaultDisposition returned nil")
	}

	// Check core dimensions are set
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

	// Check nested configs
	if cfg.Validation.Strategy != "thorough" {
		t.Errorf("Expected validation.strategy 'thorough', got '%s'", cfg.Validation.Strategy)
	}
	if cfg.ErrorHandling.Strategy != "log-and-continue" {
		t.Errorf("Expected error_handling.strategy 'log-and-continue', got '%s'", cfg.ErrorHandling.Strategy)
	}
	if cfg.ErrorHandling.RetryCount != 3 {
		t.Errorf("Expected error_handling.retry_count 3, got %d", cfg.ErrorHandling.RetryCount)
	}

	// Should pass validation
	err := Validate(cfg)
	if err != nil {
		t.Errorf("DefaultDisposition should be valid, got: %v", err)
	}
}

func TestMinimalDisposition(t *testing.T) {
	cfg := MinimalDisposition()
	if cfg == nil {
		t.Fatal("MinimalDisposition returned nil")
	}

	// Check core dimensions are set to conservative values
	if cfg.Pacing != "deliberate" {
		t.Errorf("Expected pacing 'deliberate', got '%s'", cfg.Pacing)
	}
	if cfg.Depth != "surface" {
		t.Errorf("Expected depth 'surface', got '%s'", cfg.Depth)
	}
	if cfg.Tone != "formal" {
		t.Errorf("Expected tone 'formal', got '%s'", cfg.Tone)
	}
	if cfg.Initiative != "reactive" {
		t.Errorf("Expected initiative 'reactive', got '%s'", cfg.Initiative)
	}

	// Check nested configs
	if cfg.Validation.Strategy != "none" {
		t.Errorf("Expected validation.strategy 'none', got '%s'", cfg.Validation.Strategy)
	}
	if cfg.ErrorHandling.Strategy != "fail-fast" {
		t.Errorf("Expected error_handling.strategy 'fail-fast', got '%s'", cfg.ErrorHandling.Strategy)
	}
	if cfg.ErrorHandling.RetryCount != 0 {
		t.Errorf("Expected error_handling.retry_count 0, got %d", cfg.ErrorHandling.RetryCount)
	}

	// Should pass validation
	err := Validate(cfg)
	if err != nil {
		t.Errorf("MinimalDisposition should be valid, got: %v", err)
	}
}
