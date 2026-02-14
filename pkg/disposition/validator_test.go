package disposition

import (
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultDisposition()
	err := Validate(cfg)
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got: %v", err)
	}
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *DispositionConfig
		expectedErr string
	}{
		{
			name: "missing pacing",
			cfg: &DispositionConfig{
				Pacing:     "",
				Depth:      "thorough",
				Tone:       "professional",
				Initiative: "responsive",
			},
			expectedErr: "pacing",
		},
		{
			name: "missing depth",
			cfg: &DispositionConfig{
				Pacing:     "measured",
				Depth:      "",
				Tone:       "professional",
				Initiative: "responsive",
			},
			expectedErr: "depth",
		},
		{
			name: "missing tone",
			cfg: &DispositionConfig{
				Pacing:     "measured",
				Depth:      "thorough",
				Tone:       "",
				Initiative: "responsive",
			},
			expectedErr: "tone",
		},
		{
			name: "missing initiative",
			cfg: &DispositionConfig{
				Pacing:     "measured",
				Depth:      "thorough",
				Tone:       "professional",
				Initiative: "",
			},
			expectedErr: "initiative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if err == nil {
				t.Fatal("Expected validation error for missing required field")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error to mention %s, got: %v", tt.expectedErr, err)
			}
		})
	}
}

func TestValidate_InvalidEnumValues(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *DispositionConfig
		expectedErr string
	}{
		{
			name: "invalid pacing",
			cfg: &DispositionConfig{
				Pacing:     "lightning",
				Depth:      "thorough",
				Tone:       "professional",
				Initiative: "responsive",
			},
			expectedErr: "pacing",
		},
		{
			name: "invalid depth",
			cfg: &DispositionConfig{
				Pacing:     "measured",
				Depth:      "ultra-deep",
				Tone:       "professional",
				Initiative: "responsive",
			},
			expectedErr: "depth",
		},
		{
			name: "invalid tone",
			cfg: &DispositionConfig{
				Pacing:     "measured",
				Depth:      "thorough",
				Tone:       "sarcastic",
				Initiative: "responsive",
			},
			expectedErr: "tone",
		},
		{
			name: "invalid initiative",
			cfg: &DispositionConfig{
				Pacing:     "measured",
				Depth:      "thorough",
				Tone:       "professional",
				Initiative: "hyper-autonomous",
			},
			expectedErr: "initiative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if err == nil {
				t.Fatal("Expected validation error for invalid enum value")
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error to mention %s, got: %v", tt.expectedErr, err)
			}
		})
	}
}

func TestValidate_InvalidValidationStrategy(t *testing.T) {
	cfg := DefaultDisposition()
	cfg.Validation.Strategy = "mega-thorough"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Expected validation error for invalid validation strategy")
	}
	if !strings.Contains(err.Error(), "validation.strategy") {
		t.Errorf("Expected error to mention validation.strategy, got: %v", err)
	}
}

func TestValidate_InvalidErrorStrategy(t *testing.T) {
	cfg := DefaultDisposition()
	cfg.ErrorHandling.Strategy = "panic-and-quit"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Expected validation error for invalid error strategy")
	}
	if !strings.Contains(err.Error(), "error_handling.strategy") {
		t.Errorf("Expected error to mention error_handling.strategy, got: %v", err)
	}
}

func TestValidate_InvalidRetryCount(t *testing.T) {
	tests := []struct {
		name       string
		retryCount int
		shouldFail bool
	}{
		{"retry_count_0", 0, false},
		{"retry_count_5", 5, false},
		{"retry_count_10", 10, false},
		{"retry_count_negative", -1, true},
		{"retry_count_too_high", 15, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultDisposition()
			cfg.ErrorHandling.RetryCount = tt.retryCount

			err := Validate(cfg)
			if tt.shouldFail {
				if err == nil {
					t.Fatalf("Expected validation error for retry_count %d", tt.retryCount)
				}
				if !strings.Contains(err.Error(), "retry_count") {
					t.Errorf("Expected error to mention retry_count, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected retry_count %d to be valid, got error: %v", tt.retryCount, err)
				}
			}
		})
	}
}

func TestValidate_InvalidCollaborationStyle(t *testing.T) {
	cfg := DefaultDisposition()
	cfg.Collaboration.Style = "domineering"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Expected validation error for invalid collaboration style")
	}
	if !strings.Contains(err.Error(), "collaboration.style") {
		t.Errorf("Expected error to mention collaboration.style, got: %v", err)
	}
}

func TestValidate_InvalidCheckInFrequency(t *testing.T) {
	cfg := DefaultDisposition()
	cfg.Collaboration.CheckInFrequency = "obsessively"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Expected validation error for invalid check-in frequency")
	}
	if !strings.Contains(err.Error(), "check_in_frequency") {
		t.Errorf("Expected error to mention check_in_frequency, got: %v", err)
	}
}

func TestValidate_InvalidReflectionFrequency(t *testing.T) {
	cfg := DefaultDisposition()
	cfg.Reflection.Frequency = "every-millisecond"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Expected validation error for invalid reflection frequency")
	}
	if !strings.Contains(err.Error(), "reflection.frequency") {
		t.Errorf("Expected error to mention reflection.frequency, got: %v", err)
	}
}

func TestValidate_InvalidReflectionFormat(t *testing.T) {
	cfg := DefaultDisposition()
	cfg.Reflection.Format = "haiku"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Expected validation error for invalid reflection format")
	}
	if !strings.Contains(err.Error(), "reflection.format") {
		t.Errorf("Expected error to mention reflection.format, got: %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &DispositionConfig{
		Pacing:     "lightning-fast", // invalid
		Depth:      "",                // missing
		Tone:       "professional",
		Initiative: "responsive",
		ErrorHandling: ErrorHandlingConfig{
			Strategy:   "invalid", // invalid
			RetryCount: 20,        // out of range
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Expected validation errors for multiple invalid fields")
	}

	errMsg := err.Error()
	// Should contain multiple error messages
	if !strings.Contains(errMsg, "pacing") {
		t.Errorf("Expected error to mention pacing, got: %v", err)
	}
	if !strings.Contains(errMsg, "depth") {
		t.Errorf("Expected error to mention depth, got: %v", err)
	}
}

func TestValidate_NilConfig(t *testing.T) {
	err := Validate(nil)
	if err == nil {
		t.Fatal("Expected error for nil config")
	}
	if !strings.Contains(err.Error(), "cannot be nil") {
		t.Errorf("Expected 'cannot be nil' in error, got: %v", err)
	}
}

func TestValidate_AllValidEnumValues(t *testing.T) {
	// Test all valid enum combinations to ensure validators accept them
	pacingValues := []string{"deliberate", "measured", "responsive", "rapid"}
	depthValues := []string{"surface", "functional", "thorough", "exhaustive"}
	toneValues := []string{"formal", "professional", "conversational", "casual"}
	initiativeValues := []string{"reactive", "responsive", "proactive", "autonomous"}

	for _, pacing := range pacingValues {
		for _, depth := range depthValues {
			for _, tone := range toneValues {
				for _, initiative := range initiativeValues {
					cfg := &DispositionConfig{
						Pacing:     pacing,
						Depth:      depth,
						Tone:       tone,
						Initiative: initiative,
						Validation: ValidationConfig{
							Strategy:     "thorough",
							RequireTests: true,
							RequireDocs:  false,
						},
						ErrorHandling: ErrorHandlingConfig{
							Strategy:   "retry",
							RetryCount: 3,
						},
						Collaboration: CollaborationConfig{
							Style:            "consultative",
							CheckInFrequency: "regularly",
						},
						Reflection: ReflectionConfig{
							Frequency: "session-end",
							Format:    "structured",
							Triggers:  []string{"error"},
						},
					}

					err := Validate(cfg)
					if err != nil {
						t.Errorf("Valid combination failed validation: pacing=%s, depth=%s, tone=%s, initiative=%s, err=%v",
							pacing, depth, tone, initiative, err)
					}
				}
			}
		}
	}
}

func TestValidate_AllValidationStrategies(t *testing.T) {
	strategies := []string{"none", "spot-check", "thorough", "exhaustive"}

	for _, strategy := range strategies {
		cfg := DefaultDisposition()
		cfg.Validation.Strategy = strategy

		err := Validate(cfg)
		if err != nil {
			t.Errorf("Valid validation strategy %q failed: %v", strategy, err)
		}
	}
}

func TestValidate_AllErrorStrategies(t *testing.T) {
	strategies := []string{"fail-fast", "log-and-continue", "retry", "escalate"}

	for _, strategy := range strategies {
		cfg := DefaultDisposition()
		cfg.ErrorHandling.Strategy = strategy

		err := Validate(cfg)
		if err != nil {
			t.Errorf("Valid error strategy %q failed: %v", strategy, err)
		}
	}
}

func TestValidate_AllCollaborationStyles(t *testing.T) {
	styles := []string{"independent", "consultative", "collaborative", "delegating"}

	for _, style := range styles {
		cfg := DefaultDisposition()
		cfg.Collaboration.Style = style

		err := Validate(cfg)
		if err != nil {
			t.Errorf("Valid collaboration style %q failed: %v", style, err)
		}
	}
}

func TestValidate_AllCheckInFrequencies(t *testing.T) {
	frequencies := []string{"never", "rarely", "regularly", "constantly"}

	for _, freq := range frequencies {
		cfg := DefaultDisposition()
		cfg.Collaboration.CheckInFrequency = freq

		err := Validate(cfg)
		if err != nil {
			t.Errorf("Valid check-in frequency %q failed: %v", freq, err)
		}
	}
}

func TestValidate_AllReflectionFrequencies(t *testing.T) {
	frequencies := []string{"never", "session-end", "daily", "weekly"}

	for _, freq := range frequencies {
		cfg := DefaultDisposition()
		cfg.Reflection.Frequency = freq

		err := Validate(cfg)
		if err != nil {
			t.Errorf("Valid reflection frequency %q failed: %v", freq, err)
		}
	}
}

func TestValidate_AllReflectionFormats(t *testing.T) {
	formats := []string{"structured", "narrative", "bullets"}

	for _, format := range formats {
		cfg := DefaultDisposition()
		cfg.Reflection.Format = format

		err := Validate(cfg)
		if err != nil {
			t.Errorf("Valid reflection format %q failed: %v", format, err)
		}
	}
}
