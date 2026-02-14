package disposition

import (
	"fmt"
	"strings"
)

// Valid enum values based on Gateway-ADA Contract v1.0.0
var (
	ValidPacingValues     = []string{"deliberate", "measured", "responsive", "rapid"}
	ValidDepthValues      = []string{"surface", "functional", "thorough", "exhaustive"}
	ValidToneValues       = []string{"formal", "professional", "conversational", "casual"}
	ValidInitiativeValues = []string{"reactive", "responsive", "proactive", "autonomous"}

	ValidValidationStrategies = []string{"none", "spot-check", "thorough", "exhaustive"}

	ValidErrorStrategies = []string{"fail-fast", "log-and-continue", "retry", "escalate"}

	ValidCollaborationStyles = []string{"independent", "consultative", "collaborative", "delegating"}
	ValidCheckInFrequencies  = []string{"never", "rarely", "regularly", "constantly"}

	ValidReflectionFrequencies = []string{"never", "session-end", "daily", "weekly"}
	ValidReflectionFormats     = []string{"structured", "narrative", "bullets"}
)

// Validate performs comprehensive validation on a DispositionConfig.
// It checks all required fields, enum values, and constraints defined in the ADA contract.
func Validate(cfg *DispositionConfig) error {
	if cfg == nil {
		return fmt.Errorf("disposition config cannot be nil")
	}

	var errors []string

	// Validate core behavioral dimensions (all required)
	if err := validateEnum("pacing", cfg.Pacing, ValidPacingValues); err != nil {
		errors = append(errors, err.Error())
	}
	if err := validateEnum("depth", cfg.Depth, ValidDepthValues); err != nil {
		errors = append(errors, err.Error())
	}
	if err := validateEnum("tone", cfg.Tone, ValidToneValues); err != nil {
		errors = append(errors, err.Error())
	}
	if err := validateEnum("initiative", cfg.Initiative, ValidInitiativeValues); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate nested configs
	if err := validateValidationConfig(&cfg.Validation); err != nil {
		errors = append(errors, err.Error())
	}
	if err := validateErrorHandlingConfig(&cfg.ErrorHandling); err != nil {
		errors = append(errors, err.Error())
	}
	if err := validateCollaborationConfig(&cfg.Collaboration); err != nil {
		errors = append(errors, err.Error())
	}
	if err := validateReflectionConfig(&cfg.Reflection); err != nil {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateValidationConfig validates the Validation section
func validateValidationConfig(cfg *ValidationConfig) error {
	if cfg.Strategy != "" {
		if err := validateEnum("validation.strategy", cfg.Strategy, ValidValidationStrategies); err != nil {
			return err
		}
	}
	return nil
}

// validateErrorHandlingConfig validates the ErrorHandling section
func validateErrorHandlingConfig(cfg *ErrorHandlingConfig) error {
	var errors []string

	if cfg.Strategy != "" {
		if err := validateEnum("error_handling.strategy", cfg.Strategy, ValidErrorStrategies); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate retry_count is in range 0-10
	if cfg.RetryCount < 0 || cfg.RetryCount > 10 {
		errors = append(errors, fmt.Sprintf("error_handling.retry_count must be between 0 and 10, got %d", cfg.RetryCount))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// validateCollaborationConfig validates the Collaboration section
func validateCollaborationConfig(cfg *CollaborationConfig) error {
	var errors []string

	if cfg.Style != "" {
		if err := validateEnum("collaboration.style", cfg.Style, ValidCollaborationStyles); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if cfg.CheckInFrequency != "" {
		if err := validateEnum("collaboration.check_in_frequency", cfg.CheckInFrequency, ValidCheckInFrequencies); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// validateReflectionConfig validates the Reflection section
func validateReflectionConfig(cfg *ReflectionConfig) error {
	var errors []string

	if cfg.Frequency != "" {
		if err := validateEnum("reflection.frequency", cfg.Frequency, ValidReflectionFrequencies); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if cfg.Format != "" {
		if err := validateEnum("reflection.format", cfg.Format, ValidReflectionFormats); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// validateEnum checks if value is in the valid options list
func validateEnum(fieldName string, value string, validOptions []string) error {
	if value == "" {
		return fmt.Errorf("missing required field: %s", fieldName)
	}

	for _, valid := range validOptions {
		if value == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid value for %s: %q (valid options: %v)", fieldName, value, validOptions)
}
