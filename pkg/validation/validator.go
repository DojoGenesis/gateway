package validation

import (
	"context"
	"strings"

	"github.com/DojoGenesis/gateway/disposition"
)

// Artifact represents code or content to be validated.
type Artifact struct {
	Type     string // "code", "config", "document"
	Path     string // File path
	Language string // Programming language (for code)
	Content  []byte // Raw content
	Metadata map[string]interface{}
}

// CheckResult represents the result of a single validation check.
type CheckResult struct {
	Name    string // Name of the check (e.g., "syntax", "lint", "tests")
	Passed  bool   // Whether the check passed
	Message string // Human-readable message
	Details map[string]interface{}
}

// ValidationResult represents the overall validation result.
type ValidationResult struct {
	Passed  bool          // Whether all checks passed
	Checks  []CheckResult // Individual check results
	Summary string        // Overall summary message
}

// Validator runs validation checks based on disposition.Validation.
//
// Per Gateway-ADA Contract §3.6:
//   - none: Skip validation
//   - spot-check: Quick syntax validation, sample tests
//   - thorough: Full test suite, linting, type checking
//   - exhaustive: Tests + coverage + security scanning
//
// Flags:
//   - RequireTests: Fail validation if tests missing/failing
//   - RequireDocs: Warn if docs missing, don't fail
type Validator struct {
	disp *disposition.DispositionConfig
}

// ValidatorOption is a functional option for configuring the Validator.
type ValidatorOption func(*Validator)

// WithDisposition sets the disposition configuration.
func WithDisposition(disp *disposition.DispositionConfig) ValidatorOption {
	return func(v *Validator) {
		v.disp = disp
	}
}

// NewValidator creates a new validator.
func NewValidator(opts ...ValidatorOption) *Validator {
	v := &Validator{
		disp: disposition.DefaultDisposition(),
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// Validate runs validation appropriate to the disposition strategy.
func (v *Validator) Validate(ctx context.Context, artifact Artifact) (*ValidationResult, error) {
	if v.disp == nil {
		return &ValidationResult{
			Passed:  true,
			Summary: "no disposition configured, skipping validation",
		}, nil
	}

	strategy := strings.ToLower(v.disp.Validation.Strategy)
	checks := make([]CheckResult, 0)

	switch strategy {
	case "none":
		return &ValidationResult{
			Passed:  true,
			Summary: "validation skipped (strategy: none)",
		}, nil

	case "spot-check":
		// Quick syntax check
		checks = append(checks, v.syntaxCheck(artifact))

		// Sample test check if required
		if v.disp.Validation.RequireTests {
			checks = append(checks, v.sampleTestCheck(artifact))
		}

	case "thorough":
		// Comprehensive checks
		checks = append(checks, v.syntaxCheck(artifact))
		checks = append(checks, v.lintCheck(artifact))
		checks = append(checks, v.typeCheck(artifact))

		// Full test suite if required
		if v.disp.Validation.RequireTests {
			checks = append(checks, v.fullTestSuite(artifact))
		}

		// Documentation check if required
		if v.disp.Validation.RequireDocs {
			checks = append(checks, v.docCheck(artifact))
		}

	case "exhaustive":
		// All checks
		checks = append(checks, v.syntaxCheck(artifact))
		checks = append(checks, v.lintCheck(artifact))
		checks = append(checks, v.typeCheck(artifact))
		checks = append(checks, v.fullTestSuite(artifact))
		checks = append(checks, v.coverageCheck(artifact))
		checks = append(checks, v.securityScan(artifact))

		// Documentation check if required
		if v.disp.Validation.RequireDocs {
			checks = append(checks, v.docCheck(artifact))
		}

	default:
		// Unknown strategy - default to thorough
		checks = append(checks, v.syntaxCheck(artifact))
		checks = append(checks, v.lintCheck(artifact))
		checks = append(checks, v.typeCheck(artifact))
		if v.disp.Validation.RequireTests {
			checks = append(checks, v.fullTestSuite(artifact))
		}
	}

	// Determine overall pass/fail
	passed := true
	for _, check := range checks {
		if !check.Passed {
			passed = false
			break
		}
	}

	summary := v.buildSummary(passed, len(checks), strategy)

	return &ValidationResult{
		Passed:  passed,
		Checks:  checks,
		Summary: summary,
	}, nil
}

// Stub check methods - these would be implemented with real validation logic
// For now, they return passing results to focus on routing logic

func (v *Validator) syntaxCheck(artifact Artifact) CheckResult {
	// Stub: real implementation would parse and check syntax
	return CheckResult{
		Name:    "syntax",
		Passed:  true,
		Message: "syntax check passed (stub)",
	}
}

func (v *Validator) lintCheck(artifact Artifact) CheckResult {
	// Stub: real implementation would run linter
	return CheckResult{
		Name:    "lint",
		Passed:  true,
		Message: "lint check passed (stub)",
	}
}

func (v *Validator) typeCheck(artifact Artifact) CheckResult {
	// Stub: real implementation would run type checker
	return CheckResult{
		Name:    "type",
		Passed:  true,
		Message: "type check passed (stub)",
	}
}

func (v *Validator) sampleTestCheck(artifact Artifact) CheckResult {
	// Stub: real implementation would run a subset of tests
	return CheckResult{
		Name:    "sample_tests",
		Passed:  true,
		Message: "sample tests passed (stub)",
	}
}

func (v *Validator) fullTestSuite(artifact Artifact) CheckResult {
	// Stub: real implementation would run full test suite
	return CheckResult{
		Name:    "full_tests",
		Passed:  true,
		Message: "full test suite passed (stub)",
	}
}

func (v *Validator) coverageCheck(artifact Artifact) CheckResult {
	// Stub: real implementation would measure test coverage
	return CheckResult{
		Name:    "coverage",
		Passed:  true,
		Message: "coverage check passed (stub)",
	}
}

func (v *Validator) securityScan(artifact Artifact) CheckResult {
	// Stub: real implementation would run security scanner
	return CheckResult{
		Name:    "security",
		Passed:  true,
		Message: "security scan passed (stub)",
	}
}

func (v *Validator) docCheck(artifact Artifact) CheckResult {
	// Stub: real implementation would check for documentation
	return CheckResult{
		Name:    "documentation",
		Passed:  true,
		Message: "documentation check passed (stub)",
	}
}

func (v *Validator) buildSummary(passed bool, checkCount int, strategy string) string {
	if passed {
		return strings.Join([]string{
			"validation passed:",
			strategy,
			"strategy,",
			strings.Join([]string{string(rune(checkCount + '0')), "checks"}, " "), //nolint:gosec // G115: intentional ASCII digit conversion; checkCount is an internal counter bounded by the number of registered checks, not external input
		}, " ")
	}
	return strings.Join([]string{
		"validation failed:",
		strategy,
		"strategy,",
		strings.Join([]string{string(rune(checkCount + '0')), "checks"}, " "), //nolint:gosec // G115: intentional ASCII digit conversion; checkCount is an internal counter bounded by the number of registered checks, not external input
	}, " ")
}
