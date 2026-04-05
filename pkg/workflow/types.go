// Package workflow defines the workflow execution model for Era 3.
// Workflows are DAGs of skill invocations with typed ports, disposition-aware
// steps, trigger conditions, and human approval gates.
package workflow

import "time"

// WorkflowDefinition is the top-level workflow configuration.
type WorkflowDefinition struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string            `json:"version,omitempty" yaml:"version,omitempty"`
	Trigger     *TriggerConfig    `json:"trigger,omitempty" yaml:"trigger,omitempty"`
	Steps       []WorkflowStep    `json:"steps" yaml:"steps"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// WorkflowStep is a single step in a workflow DAG.
// Each step invokes a skill with a specific disposition preset.
type WorkflowStep struct {
	ID          string            `json:"id" yaml:"id"`
	Skill       string            `json:"skill" yaml:"skill"`
	Disposition string            `json:"disposition,omitempty" yaml:"disposition,omitempty"` // preset name
	DependsOn   []string          `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Inputs      map[string]string `json:"inputs,omitempty" yaml:"inputs,omitempty"` // supports {{ template }} refs
	Gate        string            `json:"gate,omitempty" yaml:"gate,omitempty"`     // "approval-required", "auto"
	Timeout     string            `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Condition   string            `json:"condition,omitempty" yaml:"condition,omitempty"` // skip condition expression
}

// TriggerConfig defines what starts a workflow.
type TriggerConfig struct {
	Channel string            `json:"channel" yaml:"channel"` // platform ID from ChannelAdapter
	Event   string            `json:"event" yaml:"event"`     // e.g., "release.published", "message.created"
	Filter  map[string]string `json:"filter,omitempty" yaml:"filter,omitempty"`
}

// WorkflowExecution tracks a running workflow instance.
type WorkflowExecution struct {
	ID          string                   `json:"id"`
	WorkflowID  string                   `json:"workflow_id"`
	Status      ExecutionStatus          `json:"status"`
	Steps       map[string]*StepExecution `json:"steps"`
	StartedAt   time.Time                `json:"started_at"`
	CompletedAt *time.Time               `json:"completed_at,omitempty"`
	TriggerData map[string]interface{}   `json:"trigger_data,omitempty"`
}

// StepExecution tracks a single step's execution state.
type StepExecution struct {
	StepID      string          `json:"step_id"`
	Status      ExecutionStatus `json:"status"`
	Disposition string          `json:"disposition,omitempty"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
	Error       string          `json:"error,omitempty"`
}

// ExecutionStatus represents the state of a workflow or step.
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusWaiting   ExecutionStatus = "waiting_approval" // blocked on human gate
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusSkipped   ExecutionStatus = "skipped"
)

// PortDefinition defines a typed input or output for a skill.
type PortDefinition struct {
	Name        string                 `json:"name" yaml:"name"`
	Type        string                 `json:"type" yaml:"type"` // string, integer, number, boolean, object, array, ref
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                   `json:"required,omitempty" yaml:"required,omitempty"`
	Format      string                 `json:"format,omitempty" yaml:"format,omitempty"` // markdown, json, yaml, cas-ref
	Schema      map[string]interface{} `json:"schema,omitempty" yaml:"schema,omitempty"` // JSON Schema for object/array types
	Default     interface{}            `json:"default,omitempty" yaml:"default,omitempty"`
}
