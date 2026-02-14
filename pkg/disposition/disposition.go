package disposition

// DispositionConfig represents the effective disposition configuration
// after resolving the active mode and applying overrides.
// This aligns with the Gateway-ADA Contract v1.0.0.
type DispositionConfig struct {
	// Core behavioral dimensions (required)
	Pacing     string `json:"pacing" yaml:"pacing"`         // deliberate | measured | responsive | rapid
	Depth      string `json:"depth" yaml:"depth"`           // surface | functional | thorough | exhaustive
	Tone       string `json:"tone" yaml:"tone"`             // formal | professional | conversational | casual
	Initiative string `json:"initiative" yaml:"initiative"` // reactive | responsive | proactive | autonomous

	// Validation preferences
	Validation ValidationConfig `json:"validation" yaml:"validation"`

	// Error handling configuration
	ErrorHandling ErrorHandlingConfig `json:"error_handling" yaml:"error_handling"`

	// Collaboration settings
	Collaboration CollaborationConfig `json:"collaboration" yaml:"collaboration"`

	// Reflection configuration
	Reflection ReflectionConfig `json:"reflection" yaml:"reflection"`

	// Metadata (for introspection and debugging)
	SourceFile    string `json:"-" yaml:"-"` // Path to the loaded YAML file
	SchemaVersion string `json:"-" yaml:"-"` // e.g., "1.0.0"
	ActiveMode    string `json:"-" yaml:"-"` // Which mode was active when loaded

	// Mode-specific overrides (used during resolution, not in final config)
	Modes map[string]*ModeOverride `json:"modes,omitempty" yaml:"modes,omitempty"`
}

// ValidationConfig defines quality assurance preferences
type ValidationConfig struct {
	Strategy     string `json:"strategy" yaml:"strategy"`           // none | spot-check | thorough | exhaustive
	RequireTests bool   `json:"require_tests" yaml:"require_tests"` // default: true
	RequireDocs  bool   `json:"require_docs" yaml:"require_docs"`   // default: false
}

// ErrorHandlingConfig defines error response strategy
type ErrorHandlingConfig struct {
	Strategy   string `json:"strategy" yaml:"strategy"`       // fail-fast | log-and-continue | retry | escalate
	RetryCount int    `json:"retry_count" yaml:"retry_count"` // 0-10, default: 3
}

// CollaborationConfig defines multi-agent/human interaction
type CollaborationConfig struct {
	Style            string `json:"style" yaml:"style"`                           // independent | consultative | collaborative | delegating
	CheckInFrequency string `json:"check_in_frequency" yaml:"check_in_frequency"` // never | rarely | regularly | constantly
}

// ReflectionConfig defines introspection behavior
type ReflectionConfig struct {
	Frequency string   `json:"frequency" yaml:"frequency"` // never | session-end | daily | weekly
	Format    string   `json:"format" yaml:"format"`       // structured | narrative | bullets
	Triggers  []string `json:"triggers" yaml:"triggers"`   // e.g., ["error", "milestone", "learning"]
}

// ModeOverride represents a mode-specific configuration override
type ModeOverride struct {
	Pacing        string                   `json:"pacing,omitempty" yaml:"pacing,omitempty"`
	Depth         string                   `json:"depth,omitempty" yaml:"depth,omitempty"`
	Tone          string                   `json:"tone,omitempty" yaml:"tone,omitempty"`
	Initiative    string                   `json:"initiative,omitempty" yaml:"initiative,omitempty"`
	Validation    *ValidationConfig        `json:"validation,omitempty" yaml:"validation,omitempty"`
	ErrorHandling *ErrorHandlingConfig     `json:"error_handling,omitempty" yaml:"error_handling,omitempty"`
	Collaboration *CollaborationConfig     `json:"collaboration,omitempty" yaml:"collaboration,omitempty"`
	Reflection    *ReflectionConfig        `json:"reflection,omitempty" yaml:"reflection,omitempty"`
}
