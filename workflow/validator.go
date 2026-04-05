package workflow

import "fmt"

// PortDefinition describes a typed input or output port on a skill.
// Used for connection validation in the workflow builder canvas.
type PortDefinition struct {
	// Name is the port identifier (e.g. "sources", "format").
	Name string `json:"name"`

	// Type is the port data type: "string", "number", "boolean", "string[]", "object", "any".
	Type string `json:"type"`

	// Description is a human-readable explanation of the port.
	Description string `json:"description,omitempty"`

	// Required indicates whether this input must be connected.
	Required bool `json:"required,omitempty"`

	// Default is the default value when no connection is made.
	Default any `json:"default,omitempty"`

	// Enum lists allowed values for string-typed ports.
	Enum []string `json:"enum,omitempty"`
}

// ValidateConnection checks if an output port can connect to an input port.
// Returns nil if compatible, an error describing the mismatch if not.
//
// Type compatibility rules (from ADR-019):
//   - Exact type match: always valid
//   - "any" on either side: always valid
//   - string -> string[]: valid (auto-wrap in array)
//   - string[] -> string: invalid (lossy)
//   - number -> string: valid (auto-coerce)
//   - string -> number: invalid (may fail)
func ValidateConnection(output, input PortDefinition) error {
	outType := output.Type
	inType := input.Type

	// Exact match is always valid.
	if outType == inType {
		return nil
	}

	// "any" on either side is always valid.
	if outType == "any" || inType == "any" {
		return nil
	}

	// string -> string[]: valid (auto-wrap in array).
	if outType == "string" && inType == "string[]" {
		return nil
	}

	// number -> string: valid (auto-coerce).
	if outType == "number" && inType == "string" {
		return nil
	}

	return fmt.Errorf("workflow: incompatible connection: output %q (type %s) cannot connect to input %q (type %s)",
		output.Name, outType, input.Name, inType)
}
