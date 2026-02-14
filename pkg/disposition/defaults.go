package disposition

// DefaultDisposition returns a sensible default DispositionConfig
// when no explicit configuration is found. Represents a "balanced" agent.
// This aligns with the Gateway-ADA Contract v1.0.0.
func DefaultDisposition() *DispositionConfig {
	return &DispositionConfig{
		// Core behavioral dimensions
		Pacing:     "measured",
		Depth:      "thorough",
		Tone:       "professional",
		Initiative: "responsive",

		// Validation preferences
		Validation: ValidationConfig{
			Strategy:     "thorough",
			RequireTests: true,
			RequireDocs:  false,
		},

		// Error handling
		ErrorHandling: ErrorHandlingConfig{
			Strategy:   "log-and-continue",
			RetryCount: 3,
		},

		// Collaboration
		Collaboration: CollaborationConfig{
			Style:            "consultative",
			CheckInFrequency: "regularly",
		},

		// Reflection
		Reflection: ReflectionConfig{
			Frequency: "session-end",
			Format:    "structured",
			Triggers:  []string{"error", "milestone"},
		},

		// Metadata
		SourceFile:    "",
		SchemaVersion: "1.0.0",
		ActiveMode:    "",
		Modes:         nil,
	}
}

// MinimalDisposition returns a minimal valid DispositionConfig with conservative defaults.
// This is useful for testing or when you want a very cautious agent configuration.
func MinimalDisposition() *DispositionConfig {
	return &DispositionConfig{
		// Core behavioral dimensions
		Pacing:     "deliberate",
		Depth:      "surface",
		Tone:       "formal",
		Initiative: "reactive",

		// Validation preferences
		Validation: ValidationConfig{
			Strategy:     "none",
			RequireTests: false,
			RequireDocs:  false,
		},

		// Error handling
		ErrorHandling: ErrorHandlingConfig{
			Strategy:   "fail-fast",
			RetryCount: 0,
		},

		// Collaboration
		Collaboration: CollaborationConfig{
			Style:            "independent",
			CheckInFrequency: "never",
		},

		// Reflection
		Reflection: ReflectionConfig{
			Frequency: "never",
			Format:    "structured",
			Triggers:  []string{},
		},

		// Metadata
		SourceFile:    "",
		SchemaVersion: "1.0.0",
		ActiveMode:    "",
		Modes:         nil,
	}
}
