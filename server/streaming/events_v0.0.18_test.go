package streaming

import (
	"testing"
	"time"
)

func TestNewArtifactCreatedEvent(t *testing.T) {
	event := NewArtifactCreatedEvent("artifact-123", "My Diagram", "diagram", "project-456")

	if event.Type != ArtifactCreated {
		t.Errorf("expected event type %s, got %s", ArtifactCreated, event.Type)
	}

	if event.Data["artifact_id"] != "artifact-123" {
		t.Errorf("expected artifact_id 'artifact-123', got %v", event.Data["artifact_id"])
	}

	if event.Data["artifact_name"] != "My Diagram" {
		t.Errorf("expected artifact_name 'My Diagram', got %v", event.Data["artifact_name"])
	}

	if event.Data["artifact_type"] != "diagram" {
		t.Errorf("expected artifact_type 'diagram', got %v", event.Data["artifact_type"])
	}

	if event.Data["project_id"] != "project-456" {
		t.Errorf("expected project_id 'project-456', got %v", event.Data["project_id"])
	}

	if event.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}

	if time.Since(event.Timestamp) > time.Second {
		t.Error("timestamp should be recent")
	}
}

func TestNewArtifactUpdatedEvent(t *testing.T) {
	event := NewArtifactUpdatedEvent("artifact-123", "My Diagram", 3, "Updated colors")

	if event.Type != ArtifactUpdated {
		t.Errorf("expected event type %s, got %s", ArtifactUpdated, event.Type)
	}

	if event.Data["artifact_id"] != "artifact-123" {
		t.Errorf("expected artifact_id 'artifact-123', got %v", event.Data["artifact_id"])
	}

	if event.Data["artifact_name"] != "My Diagram" {
		t.Errorf("expected artifact_name 'My Diagram', got %v", event.Data["artifact_name"])
	}

	if event.Data["version"] != 3 {
		t.Errorf("expected version 3, got %v", event.Data["version"])
	}

	if event.Data["commit_message"] != "Updated colors" {
		t.Errorf("expected commit_message 'Updated colors', got %v", event.Data["commit_message"])
	}
}

func TestNewProjectSwitchedEvent(t *testing.T) {
	event := NewProjectSwitchedEvent("project-789", "Research Project")

	if event.Type != ProjectSwitched {
		t.Errorf("expected event type %s, got %s", ProjectSwitched, event.Type)
	}

	if event.Data["project_id"] != "project-789" {
		t.Errorf("expected project_id 'project-789', got %v", event.Data["project_id"])
	}

	if event.Data["project_name"] != "Research Project" {
		t.Errorf("expected project_name 'Research Project', got %v", event.Data["project_name"])
	}
}

func TestNewDiagramRenderedEvent(t *testing.T) {
	event := NewDiagramRenderedEvent("diagram-abc", "mermaid", "svg")

	if event.Type != DiagramRendered {
		t.Errorf("expected event type %s, got %s", DiagramRendered, event.Type)
	}

	if event.Data["diagram_id"] != "diagram-abc" {
		t.Errorf("expected diagram_id 'diagram-abc', got %v", event.Data["diagram_id"])
	}

	if event.Data["diagram_type"] != "mermaid" {
		t.Errorf("expected diagram_type 'mermaid', got %v", event.Data["diagram_type"])
	}

	if event.Data["format"] != "svg" {
		t.Errorf("expected format 'svg', got %v", event.Data["format"])
	}
}

func TestV0_0_18_EventTypes(t *testing.T) {
	tests := []struct {
		name     string
		event    EventType
		expected string
	}{
		{"ArtifactCreated", ArtifactCreated, "artifact_created"},
		{"ArtifactUpdated", ArtifactUpdated, "artifact_updated"},
		{"ProjectSwitched", ProjectSwitched, "project_switched"},
		{"DiagramRendered", DiagramRendered, "diagram_rendered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.event) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.event))
			}
		})
	}
}

func TestV0_0_18_EventJSONSerialization(t *testing.T) {
	event := NewArtifactCreatedEvent("test-id", "test-name", "document", "test-project")

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON data should not be empty")
	}

	parsed, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if parsed.Type != ArtifactCreated {
		t.Errorf("expected type %s, got %s", ArtifactCreated, parsed.Type)
	}

	if parsed.Data["artifact_id"] != "test-id" {
		t.Errorf("expected artifact_id 'test-id', got %v", parsed.Data["artifact_id"])
	}
}
