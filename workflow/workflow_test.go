package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// validLinear returns a simple A -> B -> C workflow definition.
func validLinear() *WorkflowDefinition {
	return &WorkflowDefinition{
		Version:      "1.0.0",
		Name:         "linear-pipeline",
		ArtifactType: WorkflowArtifactType,
		Steps: []Step{
			{ID: "a", Skill: "scout", Inputs: map[string]string{"topic": "testing"}, DependsOn: nil},
			{ID: "b", Skill: "synthesize", Inputs: map[string]string{"src": "{{ steps.a.outputs.findings }}"}, DependsOn: []string{"a"}},
			{ID: "c", Skill: "report", Inputs: map[string]string{"data": "{{ steps.b.outputs.synthesis }}"}, DependsOn: []string{"b"}},
		},
	}
}

// ---------------------------------------------------------------------------
// 1. Roundtrip
// ---------------------------------------------------------------------------

func TestMarshalUnmarshal_Roundtrip(t *testing.T) {
	original := validLinear()
	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	got, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.Name != original.Name {
		t.Errorf("Name: got %q, want %q", got.Name, original.Name)
	}
	if got.Version != original.Version {
		t.Errorf("Version: got %q, want %q", got.Version, original.Version)
	}
	if len(got.Steps) != len(original.Steps) {
		t.Fatalf("Steps: got %d, want %d", len(got.Steps), len(original.Steps))
	}
	for i, s := range got.Steps {
		if s.ID != original.Steps[i].ID {
			t.Errorf("Step[%d].ID: got %q, want %q", i, s.ID, original.Steps[i].ID)
		}
		if s.Skill != original.Steps[i].Skill {
			t.Errorf("Step[%d].Skill: got %q, want %q", i, s.Skill, original.Steps[i].Skill)
		}
	}
}

// ---------------------------------------------------------------------------
// 2-3. Valid workflows
// ---------------------------------------------------------------------------

func TestValidate_ValidLinearWorkflow(t *testing.T) {
	if err := Validate(validLinear()); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidate_ValidParallelWorkflow(t *testing.T) {
	def := &WorkflowDefinition{
		Name:         "parallel-pipeline",
		ArtifactType: WorkflowArtifactType,
		Steps: []Step{
			{ID: "a", Skill: "scout", Inputs: map[string]string{}},
			{ID: "b", Skill: "research", Inputs: map[string]string{}},
			{ID: "c", Skill: "merge", Inputs: map[string]string{}, DependsOn: []string{"a", "b"}},
		},
	}
	if err := Validate(def); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 4-6. Cycle detection
// ---------------------------------------------------------------------------

func TestValidate_CycleDetected(t *testing.T) {
	def := &WorkflowDefinition{
		Name:         "cycle-ab",
		ArtifactType: WorkflowArtifactType,
		Steps: []Step{
			{ID: "a", Skill: "scout", Inputs: map[string]string{}, DependsOn: []string{"b"}},
			{ID: "b", Skill: "synthesize", Inputs: map[string]string{}, DependsOn: []string{"a"}},
		},
	}
	err := Validate(def)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("expected cycle error, got: %v", err)
	}
}

func TestValidate_ComplexCycle(t *testing.T) {
	def := &WorkflowDefinition{
		Name:         "cycle-abc",
		ArtifactType: WorkflowArtifactType,
		Steps: []Step{
			{ID: "a", Skill: "scout", Inputs: map[string]string{}, DependsOn: []string{"c"}},
			{ID: "b", Skill: "synthesize", Inputs: map[string]string{}, DependsOn: []string{"a"}},
			{ID: "c", Skill: "report", Inputs: map[string]string{}, DependsOn: []string{"b"}},
		},
	}
	err := Validate(def)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("expected cycle error, got: %v", err)
	}
}

func TestValidate_SelfCycle(t *testing.T) {
	def := &WorkflowDefinition{
		Name:         "self-cycle",
		ArtifactType: WorkflowArtifactType,
		Steps: []Step{
			{ID: "a", Skill: "scout", Inputs: map[string]string{}, DependsOn: []string{"a"}},
		},
	}
	err := Validate(def)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("expected cycle error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 7-10. Structural validation
// ---------------------------------------------------------------------------

func TestValidate_EmptyName(t *testing.T) {
	def := validLinear()
	def.Name = ""
	err := Validate(def)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !strings.Contains(err.Error(), "name must not be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_EmptySteps(t *testing.T) {
	def := &WorkflowDefinition{
		Name:         "empty",
		ArtifactType: WorkflowArtifactType,
		Steps:        []Step{},
	}
	err := Validate(def)
	if err == nil {
		t.Fatal("expected error for empty steps, got nil")
	}
	if !strings.Contains(err.Error(), "at least one step") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_DuplicateStepIDs(t *testing.T) {
	def := &WorkflowDefinition{
		Name:         "dup-ids",
		ArtifactType: WorkflowArtifactType,
		Steps: []Step{
			{ID: "a", Skill: "scout", Inputs: map[string]string{}},
			{ID: "a", Skill: "synthesize", Inputs: map[string]string{}},
		},
	}
	err := Validate(def)
	if err == nil {
		t.Fatal("expected error for duplicate IDs, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate step ID") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_MissingDependency(t *testing.T) {
	def := &WorkflowDefinition{
		Name:         "missing-dep",
		ArtifactType: WorkflowArtifactType,
		Steps: []Step{
			{ID: "a", Skill: "scout", Inputs: map[string]string{}, DependsOn: []string{"ghost"}},
		},
	}
	err := Validate(def)
	if err == nil {
		t.Fatal("expected error for missing dependency, got nil")
	}
	if !strings.Contains(err.Error(), "non-existent step") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 11-16. Port connection validation
// ---------------------------------------------------------------------------

func TestValidateConnection_ExactMatch(t *testing.T) {
	out := PortDefinition{Name: "findings", Type: "string"}
	in := PortDefinition{Name: "sources", Type: "string"}
	if err := ValidateConnection(out, in); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateConnection_StringToArray(t *testing.T) {
	out := PortDefinition{Name: "finding", Type: "string"}
	in := PortDefinition{Name: "sources", Type: "string[]"}
	if err := ValidateConnection(out, in); err != nil {
		t.Errorf("expected valid (auto-wrap), got: %v", err)
	}
}

func TestValidateConnection_ArrayToString(t *testing.T) {
	out := PortDefinition{Name: "items", Type: "string[]"}
	in := PortDefinition{Name: "text", Type: "string"}
	err := ValidateConnection(out, in)
	if err == nil {
		t.Fatal("expected error for lossy string[]->string, got nil")
	}
	if !strings.Contains(err.Error(), "incompatible") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConnection_NumberToString(t *testing.T) {
	out := PortDefinition{Name: "score", Type: "number"}
	in := PortDefinition{Name: "label", Type: "string"}
	if err := ValidateConnection(out, in); err != nil {
		t.Errorf("expected valid (auto-coerce), got: %v", err)
	}
}

func TestValidateConnection_StringToNumber(t *testing.T) {
	out := PortDefinition{Name: "label", Type: "string"}
	in := PortDefinition{Name: "count", Type: "number"}
	err := ValidateConnection(out, in)
	if err == nil {
		t.Fatal("expected error for string->number, got nil")
	}
	if !strings.Contains(err.Error(), "incompatible") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConnection_AnyCompatibility(t *testing.T) {
	tests := []struct {
		name string
		out  PortDefinition
		in   PortDefinition
	}{
		{
			name: "any output to string input",
			out:  PortDefinition{Name: "data", Type: "any"},
			in:   PortDefinition{Name: "text", Type: "string"},
		},
		{
			name: "string output to any input",
			out:  PortDefinition{Name: "text", Type: "string"},
			in:   PortDefinition{Name: "data", Type: "any"},
		},
		{
			name: "any output to any input",
			out:  PortDefinition{Name: "a", Type: "any"},
			in:   PortDefinition{Name: "b", Type: "any"},
		},
		{
			name: "number output to any input",
			out:  PortDefinition{Name: "score", Type: "number"},
			in:   PortDefinition{Name: "data", Type: "any"},
		},
		{
			name: "any output to string[] input",
			out:  PortDefinition{Name: "data", Type: "any"},
			in:   PortDefinition{Name: "items", Type: "string[]"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateConnection(tt.out, tt.in); err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 17-18. Canvas state
// ---------------------------------------------------------------------------

func TestCanvasState_MarshalUnmarshal(t *testing.T) {
	original := &CanvasState{
		WorkflowRef: "sha256:abc123",
		Viewport:    Viewport{X: 10, Y: 20, Zoom: 1.5},
		NodePositions: map[string]Position{
			"step-1": {X: 100, Y: 200},
			"step-2": {X: 400, Y: 200},
		},
		EdgeStyles: map[string]EdgeStyle{
			"step-1->step-2": {Color: "#00ff00", Animated: true},
		},
		Selection: []string{"step-1"},
	}

	data, err := MarshalCanvas(original)
	if err != nil {
		t.Fatalf("MarshalCanvas failed: %v", err)
	}

	got, err := UnmarshalCanvas(data)
	if err != nil {
		t.Fatalf("UnmarshalCanvas failed: %v", err)
	}

	if got.WorkflowRef != original.WorkflowRef {
		t.Errorf("WorkflowRef: got %q, want %q", got.WorkflowRef, original.WorkflowRef)
	}
	if got.Viewport.Zoom != original.Viewport.Zoom {
		t.Errorf("Zoom: got %v, want %v", got.Viewport.Zoom, original.Viewport.Zoom)
	}
	if len(got.NodePositions) != len(original.NodePositions) {
		t.Errorf("NodePositions: got %d, want %d", len(got.NodePositions), len(original.NodePositions))
	}
	if pos, ok := got.NodePositions["step-1"]; !ok || pos.X != 100 || pos.Y != 200 {
		t.Errorf("NodePositions[step-1]: got %+v, want {X:100 Y:200}", pos)
	}
}

func TestCanvasState_IndependentFromWorkflow(t *testing.T) {
	// Two different canvas states referencing the same workflow produce
	// different JSON (and therefore different CAS hashes).
	canvas1 := &CanvasState{
		WorkflowRef: "sha256:samehash",
		Viewport:    Viewport{X: 0, Y: 0, Zoom: 1.0},
		NodePositions: map[string]Position{
			"a": {X: 100, Y: 100},
		},
	}
	canvas2 := &CanvasState{
		WorkflowRef: "sha256:samehash",
		Viewport:    Viewport{X: 50, Y: 50, Zoom: 2.0},
		NodePositions: map[string]Position{
			"a": {X: 300, Y: 400},
		},
	}

	data1, err := MarshalCanvas(canvas1)
	if err != nil {
		t.Fatalf("MarshalCanvas canvas1 failed: %v", err)
	}
	data2, err := MarshalCanvas(canvas2)
	if err != nil {
		t.Fatalf("MarshalCanvas canvas2 failed: %v", err)
	}

	// Verify same workflow_ref.
	var raw1, raw2 map[string]json.RawMessage
	json.Unmarshal(data1, &raw1)
	json.Unmarshal(data2, &raw2)

	if string(raw1["workflow_ref"]) != string(raw2["workflow_ref"]) {
		t.Errorf("expected same workflow_ref, got %s vs %s",
			string(raw1["workflow_ref"]), string(raw2["workflow_ref"]))
	}

	// But different overall JSON.
	if string(data1) == string(data2) {
		t.Error("expected different JSON for different canvas layouts, got identical")
	}
}
