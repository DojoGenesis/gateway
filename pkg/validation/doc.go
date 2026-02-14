// Package validation provides disposition-driven validation for artifacts
// produced during agent task execution.
//
// It supports four validation strategies (none, spot-check, thorough, exhaustive)
// and can optionally require tests and documentation.
//
// Usage:
//
//	v := validation.NewValidator(validation.WithDisposition(cfg))
//	result, err := v.Validate(ctx, artifact)
//	if !result.Passed {
//	    // handle validation failures
//	}
package validation
