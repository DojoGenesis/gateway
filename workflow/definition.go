// Package workflow implements the Era 3 Workflow Builder foundation:
// composable agent workflow definitions stored as workflow.json in CAS.
//
// The schema follows ADR-019's two-layer split: WorkflowDefinition holds
// execution semantics (steps, triggers, dependencies) while CanvasState
// (canvas.go) holds visual layout. Both are content-addressed independently.
package workflow

// OCI / CAS media types for workflow artifacts.
const (
	// WorkflowArtifactType is the OCI artifact type for workflow definitions.
	WorkflowArtifactType = "application/vnd.dojo.workflow.v1"

	// CanvasArtifactType is the OCI artifact type for canvas layout state.
	CanvasArtifactType = "application/vnd.dojo.workflow.canvas.v1"
)

// WorkflowDefinition is the execution-layer schema for a composable agent workflow.
// It is stored as workflow.json in CAS. Visual layout is separate (CanvasState).
type WorkflowDefinition struct {
	// Version follows semver (e.g. "1.0.0").
	Version string `json:"version"`

	// Name is the unique workflow identifier (e.g. "design-review-pipeline").
	Name string `json:"name"`

	// Description is a one-line summary of what the workflow does.
	Description string `json:"description,omitempty"`

	// ArtifactType must be WorkflowArtifactType for workflow definitions.
	ArtifactType string `json:"artifact_type"`

	// Steps is the ordered list of skill invocations that compose the workflow.
	Steps []Step `json:"steps"`

	// Trigger defines how the workflow is activated. Nil means manual-only.
	Trigger *Trigger `json:"trigger,omitempty"`

	// Metadata holds arbitrary key-value pairs for workflow annotations.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Step is a single skill invocation within a workflow.
type Step struct {
	// ID is the unique step identifier within this workflow.
	ID string `json:"id"`

	// Skill is the CAS skill name to execute (e.g. "strategic-scout").
	Skill string `json:"skill"`

	// Inputs maps port names to values. Values may be literals or
	// template expressions like "{{ steps.X.outputs.Y }}".
	Inputs map[string]string `json:"inputs"`

	// DependsOn lists step IDs that must complete before this step runs.
	DependsOn []string `json:"depends_on"`
}

// Trigger defines how a workflow is activated.
type Trigger struct {
	// Type is the trigger mechanism: "channel_message", "schedule", "manual", "webhook".
	Type string `json:"type"`

	// Platform is the messaging platform for channel triggers (e.g. "slack", "discord").
	Platform string `json:"platform,omitempty"`

	// Pattern is a regex or glob for message matching.
	Pattern string `json:"pattern,omitempty"`

	// Cron is a cron expression for schedule triggers.
	Cron string `json:"cron,omitempty"`
}
