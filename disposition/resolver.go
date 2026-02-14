package disposition

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ResolveDisposition discovers and loads an agent disposition configuration from the workspace.
// It follows the ADA contract resolution order and applies mode-specific overrides if specified.
//
// Resolution order (per Gateway-ADA Contract v1.0.0):
//  1. Check environment variable AGENT_DISPOSITION_FILE (if set, use that file directly)
//  2. Look for identity.yaml in workspaceRoot (full ADA identity structure)
//  3. Look for disposition.yaml in workspaceRoot (disposition-only file)
//  4. Fall back to agent.yaml in workspaceRoot (ADA bridge file)
//  5. If not found, return DefaultDisposition()
//
// Mode merging:
// If activeMode is not empty, the function looks for mode-specific overrides under the "modes"
// section in the YAML and merges them with the base configuration. Non-empty fields in the mode
// override the corresponding base fields.
//
// This function signature matches the Gateway-ADA Contract v1.0.0 exactly.
func ResolveDisposition(workspaceRoot string, activeMode string) (*DispositionConfig, error) {
	// Determine the file path using resolution order
	filePath, err := resolveFilePath(workspaceRoot)
	if err != nil {
		// If no file found, return default disposition
		fmt.Fprintf(os.Stderr, "Warning: No disposition file found in %s, using default disposition\n", workspaceRoot)
		return DefaultDisposition(), nil
	}

	// Load and parse the YAML file
	cfg, err := loadDispositionFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load disposition from %s: %w", filePath, err)
	}

	// Store metadata
	cfg.SourceFile = filePath

	// Apply mode overrides per ADA contract mode resolution:
	// 1. If activeMode parameter provided, use it
	// 2. Else default to "action" mode
	// 3. If mode not found, log warning and use base disposition
	modeToApply := activeMode
	if modeToApply == "" {
		modeToApply = "action" // Contract default
	}

	// Only apply mode if modes are defined
	if cfg.Modes != nil && len(cfg.Modes) > 0 {
		if err := applyModeOverrides(cfg, modeToApply); err != nil {
			// Mode not found - log warning and continue with base disposition
			fmt.Fprintf(os.Stderr, "Warning: mode '%s' not found, using base disposition\n", modeToApply)
		} else {
			cfg.ActiveMode = modeToApply
		}
	}

	// Validate the final configuration
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("validation failed for %s: %w", filePath, err)
	}

	return cfg, nil
}

// resolveFilePath determines which disposition file to load based on the resolution order.
func resolveFilePath(workspaceRoot string) (string, error) {
	// 1. Check environment variable
	if envPath := os.Getenv("AGENT_DISPOSITION_FILE"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("AGENT_DISPOSITION_FILE points to non-existent file: %s", envPath)
	}

	// 2. Look for identity.yaml (full ADA identity structure)
	identityPath := filepath.Join(workspaceRoot, "identity.yaml")
	if _, err := os.Stat(identityPath); err == nil {
		return identityPath, nil
	}

	// 3. Look for disposition.yaml (disposition-only file)
	dispositionPath := filepath.Join(workspaceRoot, "disposition.yaml")
	if _, err := os.Stat(dispositionPath); err == nil {
		return dispositionPath, nil
	}

	// 4. Fall back to agent.yaml (ADA bridge file)
	bridgePath := filepath.Join(workspaceRoot, "agent.yaml")
	if _, err := os.Stat(bridgePath); err == nil {
		return bridgePath, nil
	}

	// 5. No file found
	return "", fmt.Errorf("no disposition file found in workspace")
}

// loadDispositionFromFile reads and parses a disposition YAML file.
// It handles both bridge format (with disposition: wrapper) and direct format.
func loadDispositionFromFile(filePath string) (*DispositionConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Validate file size (max 1 MB per ADA contract)
	if len(data) > 1_000_000 {
		return nil, fmt.Errorf("file exceeds 1 MB limit (%d bytes)", len(data))
	}

	// First parse into a raw map to detect structure and extract metadata
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var cfg DispositionConfig

	// Check if YAML has "disposition:" wrapper (bridge format)
	if dispositionData, hasWrapper := raw["disposition"]; hasWrapper {
		// Re-marshal just the disposition section and unmarshal into config
		wrappedBytes, err := yaml.Marshal(dispositionData)
		if err != nil {
			return nil, fmt.Errorf("failed to re-marshal disposition section: %w", err)
		}
		if err := yaml.Unmarshal(wrappedBytes, &cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal disposition section: %w", err)
		}
	} else {
		// Direct format - unmarshal entire file
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal disposition: %w", err)
		}
	}

	// Extract schema_version if present (could be at root or inside disposition wrapper)
	if schemaVersion, ok := raw["schema_version"].(string); ok {
		cfg.SchemaVersion = schemaVersion
	} else if dispositionMap, ok := raw["disposition"].(map[string]interface{}); ok {
		if schemaVersion, ok := dispositionMap["schema_version"].(string); ok {
			cfg.SchemaVersion = schemaVersion
		}
	}

	// Validate schema_version is valid semver if present
	if cfg.SchemaVersion != "" {
		if err := validateSemver(cfg.SchemaVersion); err != nil {
			return nil, fmt.Errorf("invalid schema_version: %w", err)
		}
	}

	return &cfg, nil
}

// applyModeOverrides merges mode-specific configuration overrides into the base configuration.
// Non-empty fields in the mode override the corresponding base fields.
func applyModeOverrides(cfg *DispositionConfig, activeMode string) error {
	if cfg.Modes == nil {
		return fmt.Errorf("mode '%s' not found: no modes defined in configuration", activeMode)
	}

	modeOverride, exists := cfg.Modes[activeMode]
	if !exists {
		return fmt.Errorf("mode '%s' not found in configuration", activeMode)
	}

	// Merge core behavioral dimensions
	if modeOverride.Pacing != "" {
		cfg.Pacing = modeOverride.Pacing
	}
	if modeOverride.Depth != "" {
		cfg.Depth = modeOverride.Depth
	}
	if modeOverride.Tone != "" {
		cfg.Tone = modeOverride.Tone
	}
	if modeOverride.Initiative != "" {
		cfg.Initiative = modeOverride.Initiative
	}

	// Merge nested configurations
	if modeOverride.Validation != nil {
		mergeValidation(&cfg.Validation, modeOverride.Validation)
	}
	if modeOverride.ErrorHandling != nil {
		mergeErrorHandling(&cfg.ErrorHandling, modeOverride.ErrorHandling)
	}
	if modeOverride.Collaboration != nil {
		mergeCollaboration(&cfg.Collaboration, modeOverride.Collaboration)
	}
	if modeOverride.Reflection != nil {
		mergeReflection(&cfg.Reflection, modeOverride.Reflection)
	}

	return nil
}

// mergeValidation merges validation configuration, with override taking precedence for non-empty values.
func mergeValidation(base *ValidationConfig, override *ValidationConfig) {
	if override.Strategy != "" {
		base.Strategy = override.Strategy
	}
	// Note: booleans always merge (can't distinguish zero value from explicit false)
	base.RequireTests = override.RequireTests
	base.RequireDocs = override.RequireDocs
}

// mergeErrorHandling merges error handling configuration, with override taking precedence for non-zero values.
func mergeErrorHandling(base *ErrorHandlingConfig, override *ErrorHandlingConfig) {
	if override.Strategy != "" {
		base.Strategy = override.Strategy
	}
	if override.RetryCount > 0 {
		base.RetryCount = override.RetryCount
	}
}

// mergeCollaboration merges collaboration configuration, with override taking precedence for non-empty values.
func mergeCollaboration(base *CollaborationConfig, override *CollaborationConfig) {
	if override.Style != "" {
		base.Style = override.Style
	}
	if override.CheckInFrequency != "" {
		base.CheckInFrequency = override.CheckInFrequency
	}
}

// mergeReflection merges reflection configuration, with override taking precedence for non-empty values.
func mergeReflection(base *ReflectionConfig, override *ReflectionConfig) {
	if override.Frequency != "" {
		base.Frequency = override.Frequency
	}
	if override.Format != "" {
		base.Format = override.Format
	}
	if override.Triggers != nil && len(override.Triggers) > 0 {
		base.Triggers = override.Triggers
	}
}

// validateSemver validates that a version string follows semantic versioning.
// Per ADA contract, schema_version must be valid semver (e.g., "1.0.0", "2.1.3-beta").
func validateSemver(version string) error {
	// Basic semver regex pattern: MAJOR.MINOR.PATCH with optional pre-release and build metadata
	semverPattern := `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	matched, err := regexp.MatchString(semverPattern, version)
	if err != nil {
		return fmt.Errorf("regex error: %w", err)
	}
	if !matched {
		return fmt.Errorf("version %q is not valid semver (expected format: MAJOR.MINOR.PATCH)", version)
	}
	return nil
}
