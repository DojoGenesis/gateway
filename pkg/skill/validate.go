package skill

import (
	"errors"
	"fmt"
	"strings"
)

// multiErr collects multiple errors and supports errors.Is traversal.
type multiErr struct {
	msg  string
	errs []error
}

func (m *multiErr) Error() string { return m.msg }

func (m *multiErr) Is(target error) bool {
	for _, e := range m.errs {
		if errors.Is(e, target) {
			return true
		}
	}
	return false
}

func (m *multiErr) Unwrap() []error { return m.errs }

// validPortTypes is the exhaustive set of allowed port type values.
var validPortTypes = map[string]bool{
	"string":  true,
	"string[]": true,
	"number":  true,
	"boolean": true,
	"object":  true,
	"ref":     true,
	"any":     true,
}

// ErrInvalidPortType is returned when a port type is not in validPortTypes.
var ErrInvalidPortType = errors.New("skill: invalid port type")

// ErrRequiredPortHasDefault is returned when a required:true port also declares a default value.
var ErrRequiredPortHasDefault = errors.New("skill: required port must not have a default")

// ErrRefPortMissingFormat is returned when a ref-typed port lacks format: cas-ref.
var ErrRefPortMissingFormat = errors.New("skill: ref-typed port must carry format: cas-ref")

// PortValidationError wraps a port-level validation error with context.
type PortValidationError struct {
	Direction string // "input" or "output"
	Port      string // port name
	Err       error
}

func (e *PortValidationError) Error() string {
	return fmt.Sprintf("port %s[%s]: %v", e.Direction, e.Port, e.Err)
}

func (e *PortValidationError) Unwrap() error {
	return e.Err
}

// ValidateSkillManifest checks all port definitions in a SkillManifest.
//
// Rules enforced:
//   - port.Type must be a known value (string, string[], number, boolean, object, ref, any)
//   - required:true ports must not carry a default value
//   - ref-typed ports must carry Format == "cas-ref"
//
// Returns nil when all ports are valid. Returns a combined error that supports
// errors.Is traversal when one or more violations are found; all violations are
// reported, not just the first.
func ValidateSkillManifest(m *SkillManifest) error {
	if m == nil {
		return fmt.Errorf("skill: manifest is nil")
	}

	var portErrs []error

	for _, p := range m.Inputs {
		if err := validatePort("input", p); err != nil {
			portErrs = append(portErrs, err)
		}
	}

	for _, p := range m.Outputs {
		if err := validatePort("output", p); err != nil {
			portErrs = append(portErrs, err)
		}
	}

	if len(portErrs) == 0 {
		return nil
	}

	msgs := make([]string, len(portErrs))
	for i, e := range portErrs {
		msgs[i] = e.Error()
	}

	return &multiErr{
		msg:  fmt.Sprintf("manifest %q has %d port error(s): %s", m.Name, len(portErrs), strings.Join(msgs, "; ")),
		errs: portErrs,
	}
}

// validatePort checks a single PortDefinition.
func validatePort(direction string, p PortDefinition) error {
	if !validPortTypes[p.Type] {
		return &PortValidationError{
			Direction: direction,
			Port:      p.Name,
			Err:       fmt.Errorf("%w: %q (must be one of: string, string[], number, boolean, object, ref, any)", ErrInvalidPortType, p.Type),
		}
	}

	if p.Required && p.Default != nil {
		return &PortValidationError{
			Direction: direction,
			Port:      p.Name,
			Err:       ErrRequiredPortHasDefault,
		}
	}

	if p.Type == "ref" && p.Format != "cas-ref" {
		return &PortValidationError{
			Direction: direction,
			Port:      p.Name,
			Err:       fmt.Errorf("%w (got %q)", ErrRefPortMissingFormat, p.Format),
		}
	}

	return nil
}
