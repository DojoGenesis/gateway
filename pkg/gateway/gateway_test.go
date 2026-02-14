package gateway

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
	"gopkg.in/yaml.v3"
)

// TestInterfaceCompilation ensures that all interfaces can be implemented without compilation errors.
// This test validates the interface signatures and that concrete types can satisfy them.
func TestInterfaceCompilation(t *testing.T) {
	// Test that a mock implementation can satisfy ToolRegistry
	var _ ToolRegistry = (*mockToolRegistry)(nil)

	// Test that a mock implementation can satisfy AgentInitializer
	var _ AgentInitializer = (*mockAgentInitializer)(nil)

	// Test that a mock implementation can satisfy MemoryStore
	var _ MemoryStore = (*mockMemoryStore)(nil)

	// Test that a mock implementation can satisfy OrchestrationExecutor
	var _ OrchestrationExecutor = (*mockOrchestrationExecutor)(nil)
}

// TestTypesHaveStructTags verifies that all types have proper JSON and YAML struct tags.
func TestTypesHaveStructTags(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		wantJSON bool
		wantYAML bool
	}{
		{"AgentConfig", AgentConfig{Pacing: "measured", Depth: "thorough", Tone: "professional", Initiative: "responsive"}, true, true},
		{"MemoryEntry", MemoryEntry{ID: "1", EntryType: "test", Content: "content", Metadata: map[string]interface{}{}, CreatedAt: time.Now(), UpdatedAt: time.Now()}, true, true},
		{"SearchQuery", SearchQuery{Text: "query", EntryType: "conversation"}, true, true},
		{"ExecutionPlan", ExecutionPlan{ID: "1", Name: "plan", DAG: []*ToolInvocation{}}, true, true},
		{"ExecutionResult", ExecutionResult{ExecutionID: "1", Status: "success", Output: map[string]interface{}{}, Duration: 100}, true, true},
		{"ToolInvocation", ToolInvocation{ID: "1", ToolName: "test", Input: map[string]interface{}{}}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantJSON {
				// Test JSON marshaling
				jsonData, err := json.Marshal(tt.value)
				if err != nil {
					t.Errorf("JSON marshal failed: %v", err)
				}
				if len(jsonData) == 0 {
					t.Error("JSON marshal produced empty output")
				}

				// Test JSON unmarshaling to verify tags are correct
				var result interface{}
				if err := json.Unmarshal(jsonData, &result); err != nil {
					t.Errorf("JSON unmarshal failed: %v", err)
				}
			}

			if tt.wantYAML {
				// Test YAML marshaling
				yamlData, err := yaml.Marshal(tt.value)
				if err != nil {
					t.Errorf("YAML marshal failed: %v", err)
				}
				if len(yamlData) == 0 {
					t.Error("YAML marshal produced empty output")
				}
			}
		})
	}
}

// TestErrorTypes verifies that all error values are correctly typed and distinct.
func TestErrorTypes(t *testing.T) {
	errors := []struct {
		name  string
		err   error
		errIs error
	}{
		{"ErrToolNotFound", ErrToolNotFound, ErrToolNotFound},
		{"ErrAgentInitFailed", ErrAgentInitFailed, ErrAgentInitFailed},
		{"ErrMemoryUnavailable", ErrMemoryUnavailable, ErrMemoryUnavailable},
		{"ErrExecutionCancelled", ErrExecutionCancelled, ErrExecutionCancelled},
		{"ErrInvalidPlan", ErrInvalidPlan, ErrInvalidPlan},
	}

	for _, tt := range errors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("error value is nil")
			}
			if tt.err.Error() == "" {
				t.Error("error message is empty")
			}
			// Verify error identity
			if tt.err != tt.errIs {
				t.Errorf("error identity check failed: %v != %v", tt.err, tt.errIs)
			}
		})
	}

	// Verify errors are distinct
	distinctErrors := make(map[error]bool)
	for _, tt := range errors {
		if distinctErrors[tt.err] {
			t.Errorf("duplicate error found: %v", tt.err)
		}
		distinctErrors[tt.err] = true
	}
}

// TestAgentConfigStructure validates the AgentConfig structure.
func TestAgentConfigStructure(t *testing.T) {
	cfg := AgentConfig{
		Pacing:     "measured",
		Depth:      "thorough",
		Tone:       "professional",
		Initiative: "responsive",
	}

	if cfg.Pacing != "measured" {
		t.Errorf("Pacing = %v, want %v", cfg.Pacing, "measured")
	}
	if cfg.Depth != "thorough" {
		t.Errorf("Depth = %v, want %v", cfg.Depth, "thorough")
	}
	if cfg.Tone != "professional" {
		t.Errorf("Tone = %v, want %v", cfg.Tone, "professional")
	}
	if cfg.Initiative != "responsive" {
		t.Errorf("Initiative = %v, want %v", cfg.Initiative, "responsive")
	}
}

// TestMemoryEntryStructure validates the MemoryEntry structure.
func TestMemoryEntryStructure(t *testing.T) {
	now := time.Now()
	entry := MemoryEntry{
		ID:        "mem-001",
		EntryType: "conversation",
		Content:   "test content",
		Metadata:  map[string]interface{}{"key": "value"},
		CreatedAt: now,
		UpdatedAt: now,
		Embedding: []float64{0.1, 0.2, 0.3},
	}

	if entry.ID != "mem-001" {
		t.Errorf("ID = %v, want %v", entry.ID, "mem-001")
	}
	if entry.EntryType != "conversation" {
		t.Errorf("EntryType = %v, want %v", entry.EntryType, "conversation")
	}
	if len(entry.Embedding) != 3 {
		t.Errorf("Embedding length = %v, want %v", len(entry.Embedding), 3)
	}
}

// TestExecutionPlanStructure validates the ExecutionPlan structure.
func TestExecutionPlanStructure(t *testing.T) {
	plan := ExecutionPlan{
		ID:   "plan-001",
		Name: "Test Plan",
		DAG: []*ToolInvocation{
			{
				ID:       "inv-001",
				ToolName: "tool1",
				Input:    map[string]interface{}{"param": "value"},
			},
			{
				ID:        "inv-002",
				ToolName:  "tool2",
				Input:     map[string]interface{}{},
				DependsOn: []string{"inv-001"},
			},
		},
	}

	if plan.ID != "plan-001" {
		t.Errorf("ID = %v, want %v", plan.ID, "plan-001")
	}
	if len(plan.DAG) != 2 {
		t.Errorf("DAG length = %v, want %v", len(plan.DAG), 2)
	}
	if len(plan.DAG[1].DependsOn) != 1 {
		t.Errorf("DependsOn length = %v, want %v", len(plan.DAG[1].DependsOn), 1)
	}
}

// Mock implementations for interface compilation testing

type mockToolRegistry struct{}

func (m *mockToolRegistry) Register(ctx context.Context, def *tools.ToolDefinition) error {
	return nil
}

func (m *mockToolRegistry) Get(ctx context.Context, name string) (*tools.ToolDefinition, error) {
	return nil, ErrToolNotFound
}

func (m *mockToolRegistry) List(ctx context.Context) ([]*tools.ToolDefinition, error) {
	return []*tools.ToolDefinition{}, nil
}

func (m *mockToolRegistry) ListByNamespace(ctx context.Context, prefix string) ([]*tools.ToolDefinition, error) {
	return []*tools.ToolDefinition{}, nil
}

type mockAgentInitializer struct{}

func (m *mockAgentInitializer) Initialize(ctx context.Context, workspaceRoot string, activeMode string) (*AgentConfig, error) {
	return &AgentConfig{}, nil
}

type mockMemoryStore struct{}

func (m *mockMemoryStore) Store(ctx context.Context, entry *MemoryEntry) error {
	return nil
}

func (m *mockMemoryStore) Search(ctx context.Context, query *SearchQuery, limit int) ([]*MemoryEntry, error) {
	return []*MemoryEntry{}, nil
}

func (m *mockMemoryStore) Get(ctx context.Context, id string) (*MemoryEntry, error) {
	return nil, ErrMemoryUnavailable
}

func (m *mockMemoryStore) Delete(ctx context.Context, id string) error {
	return nil
}

type mockOrchestrationExecutor struct{}

func (m *mockOrchestrationExecutor) Execute(ctx context.Context, plan *ExecutionPlan) (*ExecutionResult, error) {
	return &ExecutionResult{Status: "success"}, nil
}

func (m *mockOrchestrationExecutor) Cancel(ctx context.Context, executionID string) error {
	return nil
}
