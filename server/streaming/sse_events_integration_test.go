package streaming

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/agent"
	"github.com/DojoGenesis/gateway/server/apptools"
	"github.com/DojoGenesis/gateway/server/artifacts"
	"github.com/DojoGenesis/gateway/server/projects"
	_ "modernc.org/sqlite"
)

func setupIntegrationTest(t *testing.T) (*sql.DB, *projects.ProjectManager, *artifacts.ArtifactManager, func()) {
	dbPath := "./test_integration.db"

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	projectManager, err := projects.NewProjectManager(db)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create project manager: %v", err)
	}

	// Create artifacts tables (ArtifactManager does not auto-create schema)
	artifactSchema := `
	CREATE TABLE IF NOT EXISTS artifacts (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		session_id TEXT,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		latest_version INTEGER DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		metadata TEXT,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS artifact_versions (
		id TEXT PRIMARY KEY,
		artifact_id TEXT NOT NULL,
		version INTEGER NOT NULL,
		content TEXT NOT NULL,
		diff TEXT,
		commit_message TEXT,
		created_at DATETIME NOT NULL,
		created_by TEXT,
		metadata TEXT,
		FOREIGN KEY (artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE,
		UNIQUE(artifact_id, version)
	);
	`
	if _, err := db.Exec(artifactSchema); err != nil {
		db.Close()
		t.Fatalf("Failed to create artifact tables: %v", err)
	}

	artifactManager, err := artifacts.NewArtifactManager(db)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	apptools.InitializeProjectTools(projectManager)
	apptools.InitializeArtifactTools(artifactManager)

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, projectManager, artifactManager, cleanup
}

type mockProviderForSSE struct {
	toolCalls      []provider.ToolCall
	toolCallsIndex int
}

func (m *mockProviderForSSE) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	var toolCalls []provider.ToolCall
	if m.toolCallsIndex < len(m.toolCalls) {
		toolCalls = []provider.ToolCall{m.toolCalls[m.toolCallsIndex]}
		m.toolCallsIndex++
	}

	return &provider.CompletionResponse{
		ID:      "test-id",
		Content: "Test response",
		Model:   "mock-model",
		Usage: provider.Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
		ToolCalls: toolCalls,
	}, nil
}

func (m *mockProviderForSSE) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	ch := make(chan *provider.CompletionChunk)
	close(ch)
	return ch, nil
}

func (m *mockProviderForSSE) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{{ID: "mock-model", Name: "Mock Model"}}, nil
}

func (m *mockProviderForSSE) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{Name: "Mock Provider", Version: "1.0.0"}, nil
}

func (m *mockProviderForSSE) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Result: map[string]interface{}{"success": true},
	}, nil
}

func (m *mockProviderForSSE) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

type mockPluginManagerForSSE struct {
	provider *mockProviderForSSE
}

func (m *mockPluginManagerForSSE) GetProvider(name string) (provider.ModelProvider, error) {
	return m.provider, nil
}

func (m *mockPluginManagerForSSE) GetProviders() map[string]provider.ModelProvider {
	return map[string]provider.ModelProvider{"mock": m.provider}
}

func (m *mockPluginManagerForSSE) DiscoverPlugins(pluginDir string) error {
	return nil
}

func (m *mockPluginManagerForSSE) Shutdown() error {
	return nil
}

func TestSSEEventEmission_CreateArtifact(t *testing.T) {
	_, projectManager, artifactManager, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	project, err := projectManager.CreateProject(ctx, "Test Project", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	mockProvider := &mockProviderForSSE{
		toolCalls: []provider.ToolCall{
			{
				ID:   "call-1",
				Name: "create_artifact",
				Arguments: map[string]interface{}{
					"project_id": project.ID,
					"type":       "diagram",
					"name":       "Test Diagram",
					"content":    "graph TD\nA-->B",
				},
			},
		},
	}

	pm := &mockPluginManagerForSSE{provider: mockProvider}
	pa := agent.NewPrimaryAgent(pm)
	sa := NewStreamingAgentWithEvents(pa)

	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query:     "Create a diagram",
		UserTier:  "guest",
		ProjectID: project.ID,
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	artifactCreatedFound := false
	for _, event := range events {
		if event.Type == ArtifactCreated {
			artifactCreatedFound = true

			if event.Data["artifact_name"] != "Test Diagram" {
				t.Errorf("Expected artifact_name 'Test Diagram', got %v", event.Data["artifact_name"])
			}
			if event.Data["artifact_type"] != "diagram" {
				t.Errorf("Expected artifact_type 'diagram', got %v", event.Data["artifact_type"])
			}
			if event.Data["project_id"] != project.ID {
				t.Errorf("Expected project_id '%s', got %v", project.ID, event.Data["project_id"])
			}
		}
	}

	if !artifactCreatedFound {
		t.Error("ArtifactCreated event not emitted")
	}

	artifacts, err := artifactManager.ListArtifacts(ctx, project.ID, "", 10)
	if err != nil {
		t.Fatalf("Failed to list artifacts: %v", err)
	}
	if len(artifacts) != 1 {
		t.Errorf("Expected 1 artifact, got %d", len(artifacts))
	}
}

func TestSSEEventEmission_UpdateArtifact(t *testing.T) {
	_, projectManager, artifactManager, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	project, err := projectManager.CreateProject(ctx, "Test Project", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	artifact, err := artifactManager.CreateArtifact(ctx, project.ID, "", artifacts.TypeDiagram, "Test Diagram", "Test Description", "graph TD\nA-->B")
	if err != nil {
		t.Fatalf("Failed to create artifact: %v", err)
	}

	mockProvider := &mockProviderForSSE{
		toolCalls: []provider.ToolCall{
			{
				ID:   "call-1",
				Name: "update_artifact",
				Arguments: map[string]interface{}{
					"artifact_id":    artifact.ID,
					"content":        "graph TD\nA-->B\nB-->C",
					"commit_message": "Added node C",
				},
			},
		},
	}

	pm := &mockPluginManagerForSSE{provider: mockProvider}
	pa := agent.NewPrimaryAgent(pm)
	sa := NewStreamingAgentWithEvents(pa)

	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query:     "Update the diagram",
		UserTier:  "guest",
		ProjectID: project.ID,
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	artifactUpdatedFound := false
	for _, event := range events {
		if event.Type == ArtifactUpdated {
			artifactUpdatedFound = true

			if event.Data["artifact_name"] != "Test Diagram" {
				t.Errorf("Expected artifact_name 'Test Diagram', got %v", event.Data["artifact_name"])
			}
			if event.Data["version"] != 2 {
				t.Errorf("Expected version 2, got %v", event.Data["version"])
			}
			if event.Data["commit_message"] != "Added node C" {
				t.Errorf("Expected commit_message 'Added node C', got %v", event.Data["commit_message"])
			}
		}
	}

	if !artifactUpdatedFound {
		t.Error("ArtifactUpdated event not emitted")
	}
}

func TestSSEEventEmission_SwitchProject(t *testing.T) {
	_, projectManager, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	project1, err := projectManager.CreateProject(ctx, "Project 1", "Description 1", "")
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}

	mockProvider := &mockProviderForSSE{
		toolCalls: []provider.ToolCall{
			{
				ID:   "call-1",
				Name: "switch_project",
				Arguments: map[string]interface{}{
					"project_id": project1.ID,
				},
			},
		},
	}

	pm := &mockPluginManagerForSSE{provider: mockProvider}
	pa := agent.NewPrimaryAgent(pm)
	sa := NewStreamingAgentWithEvents(pa)

	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query:    "Switch to Project 1",
		UserTier: "guest",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	projectSwitchedFound := false
	for _, event := range events {
		if event.Type == ProjectSwitched {
			projectSwitchedFound = true

			if event.Data["project_id"] != project1.ID {
				t.Errorf("Expected project_id '%s', got %v", project1.ID, event.Data["project_id"])
			}
			if event.Data["project_name"] != "Project 1" {
				t.Errorf("Expected project_name 'Project 1', got %v", event.Data["project_name"])
			}
		}
	}

	if !projectSwitchedFound {
		t.Error("ProjectSwitched event not emitted")
	}
}

func TestSSEEventEmission_RenderDiagram(t *testing.T) {
	mockProvider := &mockProviderForSSE{
		toolCalls: []provider.ToolCall{
			{
				ID:   "call-1",
				Name: "prepare_diagram",
				Arguments: map[string]interface{}{
					"syntax": "graph TD\nA-->B",
					"type":   "mermaid",
					"format": "svg",
				},
			},
		},
	}

	pm := &mockPluginManagerForSSE{provider: mockProvider}
	pa := agent.NewPrimaryAgent(pm)
	sa := NewStreamingAgentWithEvents(pa)

	ctx := context.Background()
	eventChan, err := sa.HandleQueryStreamingWithEvents(ctx, agent.QueryRequest{
		Query:    "Render this diagram",
		UserTier: "guest",
	})

	if err != nil {
		t.Fatalf("HandleQueryStreamingWithEvents failed: %v", err)
	}

	events := collectEvents(eventChan, 5*time.Second)

	diagramRenderedFound := false
	for _, event := range events {
		if event.Type == DiagramRendered {
			diagramRenderedFound = true

			if event.Data["diagram_type"] != "mermaid" {
				t.Errorf("Expected diagram_type 'mermaid', got %v", event.Data["diagram_type"])
			}
			if event.Data["format"] != "svg" {
				t.Errorf("Expected format 'svg', got %v", event.Data["format"])
			}
		}
	}

	if !diagramRenderedFound {
		t.Error("DiagramRendered event not emitted")
	}
}
