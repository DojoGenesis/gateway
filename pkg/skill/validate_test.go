package skill

import (
	"errors"
	"testing"
)

func TestValidateSkillManifest_NilManifest(t *testing.T) {
	err := ValidateSkillManifest(nil)
	if err == nil {
		t.Fatal("expected error for nil manifest")
	}
}

func TestValidateSkillManifest_EmptyPorts(t *testing.T) {
	m := &SkillManifest{
		Name:    "test-skill",
		Version: "1.0.0",
		Inputs:  []PortDefinition{},
		Outputs: []PortDefinition{},
	}
	if err := ValidateSkillManifest(m); err != nil {
		t.Fatalf("expected no error for empty ports, got: %v", err)
	}
}

func TestValidateSkillManifest_ValidTypes(t *testing.T) {
	types := []string{"string", "string[]", "number", "boolean", "object", "any"}
	for _, typ := range types {
		m := &SkillManifest{
			Name: "test",
			Inputs: []PortDefinition{
				{Name: "in", Type: typ},
			},
			Outputs: []PortDefinition{
				{Name: "out", Type: typ},
			},
		}
		if err := ValidateSkillManifest(m); err != nil {
			t.Errorf("type %q should be valid, got: %v", typ, err)
		}
	}
}

func TestValidateSkillManifest_InvalidPortType(t *testing.T) {
	m := &SkillManifest{
		Name: "test",
		Inputs: []PortDefinition{
			{Name: "bad", Type: "file"},
		},
	}
	err := ValidateSkillManifest(m)
	if err == nil {
		t.Fatal("expected error for invalid port type")
	}
	if !errors.Is(err, ErrInvalidPortType) {
		t.Errorf("expected ErrInvalidPortType wrapped in error, got: %v", err)
	}
}

func TestValidateSkillManifest_RequiredPortHasDefault(t *testing.T) {
	m := &SkillManifest{
		Name: "test",
		Inputs: []PortDefinition{
			{Name: "ctx", Type: "string", Required: true, Default: "hello"},
		},
	}
	err := ValidateSkillManifest(m)
	if err == nil {
		t.Fatal("expected error for required port with default")
	}
	if !errors.Is(err, ErrRequiredPortHasDefault) {
		t.Errorf("expected ErrRequiredPortHasDefault, got: %v", err)
	}
}

func TestValidateSkillManifest_RequiredPortNoDefault_Valid(t *testing.T) {
	m := &SkillManifest{
		Name: "test",
		Inputs: []PortDefinition{
			{Name: "ctx", Type: "string", Required: true},
		},
	}
	if err := ValidateSkillManifest(m); err != nil {
		t.Fatalf("required port with no default should be valid, got: %v", err)
	}
}

func TestValidateSkillManifest_RefPortMissingFormat(t *testing.T) {
	m := &SkillManifest{
		Name: "test",
		Outputs: []PortDefinition{
			{Name: "artifact", Type: "ref"},
		},
	}
	err := ValidateSkillManifest(m)
	if err == nil {
		t.Fatal("expected error for ref port without format: cas-ref")
	}
	if !errors.Is(err, ErrRefPortMissingFormat) {
		t.Errorf("expected ErrRefPortMissingFormat, got: %v", err)
	}
}

func TestValidateSkillManifest_RefPortWithCasRef_Valid(t *testing.T) {
	m := &SkillManifest{
		Name: "test",
		Outputs: []PortDefinition{
			{Name: "artifact", Type: "ref", Format: "cas-ref"},
		},
	}
	if err := ValidateSkillManifest(m); err != nil {
		t.Fatalf("ref port with format: cas-ref should be valid, got: %v", err)
	}
}

func TestValidateSkillManifest_MultipleErrors(t *testing.T) {
	m := &SkillManifest{
		Name: "test",
		Inputs: []PortDefinition{
			{Name: "bad-type", Type: "file"},
			{Name: "req-default", Type: "string", Required: true, Default: "oops"},
		},
		Outputs: []PortDefinition{
			{Name: "no-format", Type: "ref"}, // missing format: cas-ref
		},
	}
	err := ValidateSkillManifest(m)
	if err == nil {
		t.Fatal("expected error for multiple violations")
	}
	// Should report 3 errors in one message.
	errStr := err.Error()
	for _, substr := range []string{"bad-type", "req-default", "no-format"} {
		if len(errStr) == 0 {
			t.Errorf("error string is empty but should contain %q", substr)
		}
	}
}

func TestPortValidationError_Unwrap(t *testing.T) {
	pve := &PortValidationError{
		Direction: "input",
		Port:      "ctx",
		Err:       ErrInvalidPortType,
	}
	if !errors.Is(pve, ErrInvalidPortType) {
		t.Error("expected errors.Is to traverse Unwrap chain")
	}
}
