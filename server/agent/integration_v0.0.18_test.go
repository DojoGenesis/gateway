package agent

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	providerpkg "github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/apptools"
	"github.com/DojoGenesis/gateway/server/artifacts"
	"github.com/DojoGenesis/gateway/server/projects"
	"github.com/DojoGenesis/gateway/tools"
	_ "modernc.org/sqlite"
)

type mockProviderV18 struct{}

func (m *mockProviderV18) GetInfo(ctx context.Context) (*providerpkg.ProviderInfo, error) {
	return &providerpkg.ProviderInfo{
		Name:        "mock-v18",
		Version:     "1.0.0",
		Description: "Mock provider for v0.0.18 testing",
	}, nil
}

func (m *mockProviderV18) ListModels(ctx context.Context) ([]providerpkg.ModelInfo, error) {
	return []providerpkg.ModelInfo{{ID: "test-model", Name: "Test Model", Cost: 0}}, nil
}

func (m *mockProviderV18) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	return &providerpkg.CompletionResponse{
		ID:      "test-response",
		Content: "Test response",
		Model:   req.Model,
		Usage: providerpkg.Usage{
			InputTokens:  10,
			OutputTokens: 20,
		},
		ToolCalls: []providerpkg.ToolCall{},
	}, nil
}

func (m *mockProviderV18) GenerateCompletionStream(ctx context.Context, req *providerpkg.CompletionRequest) (<-chan *providerpkg.CompletionChunk, error) {
	ch := make(chan *providerpkg.CompletionChunk)
	close(ch)
	return ch, nil
}

func (m *mockProviderV18) CallTool(ctx context.Context, req *providerpkg.ToolCallRequest) (*providerpkg.ToolCallResponse, error) {
	return &providerpkg.ToolCallResponse{
		Result: map[string]interface{}{"success": true},
	}, nil
}

func (m *mockProviderV18) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding, nil
}

type mockPluginManagerV18 struct {
	provider *mockProviderV18
}

func (m *mockPluginManagerV18) GetProvider(name string) (providerpkg.ModelProvider, error) {
	return m.provider, nil
}

func (m *mockPluginManagerV18) GetProviders() map[string]providerpkg.ModelProvider {
	return map[string]providerpkg.ModelProvider{
		"mock-v18": m.provider,
	}
}

func TestPrimaryAgentV18Integration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	schema, err := os.ReadFile("../migrations/20260131_v0.0.17_and_v0.0.18_schemas.sql")
	if err != nil {
		t.Fatalf("Failed to read schema: %v", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create ProjectManager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create ArtifactManager: %v", err)
	}

	mockPM := &mockPluginManagerV18{provider: &mockProviderV18{}}
	agent := NewPrimaryAgent(mockPM)

	agent.SetProjectManager(pm)
	agent.SetArtifactManager(am)

	if agent.projectManager == nil {
		t.Error("ProjectManager not set on agent")
	}

	if agent.artifactManager == nil {
		t.Error("ArtifactManager not set on agent")
	}

	ctx := context.Background()
	project, err := pm.CreateProject(ctx, "test-project", "Test project for v0.0.18", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	req := QueryRequest{
		Query:     "Create a test artifact",
		UserID:    "test-user",
		UserTier:  "authenticated",
		ProjectID: project.ID,
	}

	if req.ProjectID == "" {
		t.Error("ProjectID not set in QueryRequest")
	}

	if req.ProjectID != project.ID {
		t.Errorf("Expected ProjectID %s, got %s", project.ID, req.ProjectID)
	}
}

// TestContextProjectID verifies the context-based project ID management
func TestContextProjectID(t *testing.T) {
	ctx := context.Background()

	// Test WithProjectID
	ctx = tools.WithProjectID(ctx, "test-project-123")

	projectID := tools.GetProjectIDFromContext(ctx)
	if projectID != "test-project-123" {
		t.Errorf("Expected project ID 'test-project-123', got '%s'", projectID)
	}

	// Test empty context
	emptyCtx := context.Background()
	emptyProjectID := tools.GetProjectIDFromContext(emptyCtx)
	if emptyProjectID != "" {
		t.Errorf("Expected empty project ID from empty context, got '%s'", emptyProjectID)
	}

	// Test that context isolation works for two different contexts
	ctx1 := tools.WithProjectID(context.Background(), "project-alpha")
	ctx2 := tools.WithProjectID(context.Background(), "project-beta")

	id1 := tools.GetProjectIDFromContext(ctx1)
	id2 := tools.GetProjectIDFromContext(ctx2)

	if id1 != "project-alpha" {
		t.Errorf("Context 1: expected 'project-alpha', got '%s'", id1)
	}
	if id2 != "project-beta" {
		t.Errorf("Context 2: expected 'project-beta', got '%s'", id2)
	}

	// Verify they didn't interfere with each other
	id1Again := tools.GetProjectIDFromContext(ctx1)
	if id1Again != "project-alpha" {
		t.Errorf("Context 1 changed: expected 'project-alpha', got '%s'", id1Again)
	}
}

// TestConcurrentRequestsWithDifferentProjects simulates concurrent HTTP requests
// with different project IDs to verify thread-safe project isolation
func TestConcurrentRequestsWithDifferentProjects(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-concurrent.db")

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	schema, err := os.ReadFile("../migrations/20260131_v0.0.17_and_v0.0.18_schemas.sql")
	if err != nil {
		t.Fatalf("Failed to read schema: %v", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("Failed to apply schema: %v", err)
	}

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create ProjectManager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create ArtifactManager: %v", err)
	}

	// RegisterArtifactTools sets the artifact manager AND re-registers all
	// artifact tool definitions in the global registry. This is safe to call on
	// every test invocation (including -count>1 runs) because it unregisters any
	// stale entries before re-registering, ensuring the registry is consistent
	// regardless of what previous tests did with ClearRegistry().
	apptools.RegisterArtifactTools(am)
	defer tools.ClearRegistry()

	ctx := context.Background()

	// Create two different projects
	project1, err := pm.CreateProject(ctx, "concurrent-project-1", "Project 1", "")
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}

	project2, err := pm.CreateProject(ctx, "concurrent-project-2", "Project 2", "")
	if err != nil {
		t.Fatalf("Failed to create project 2: %v", err)
	}

	// Use channels to coordinate concurrent requests and collect results
	type result struct {
		projectID  string
		artifactID string
		err        error
	}

	results := make(chan result, 2)

	// Simulate Request 1: Create artifact in project 1
	go func() {
		ctx1 := tools.WithProjectID(context.Background(), project1.ID)

		params := map[string]interface{}{
			"type":    "document",
			"name":    "artifact-from-request-1",
			"content": "Content from request 1",
		}

		res, err := tools.InvokeTool(ctx1, "create_artifact", params)

		if err != nil {
			results <- result{projectID: project1.ID, err: err}
			return
		}

		// Extract artifact from result
		artifactData, ok := res["artifact"]
		if !ok {
			results <- result{projectID: project1.ID, err: fmt.Errorf("no artifact returned: %v", res)}
			return
		}

		artifactMap, ok := artifactData.(map[string]interface{})
		if !ok {
			results <- result{projectID: project1.ID, err: fmt.Errorf("artifact is not of correct type")}
			return
		}

		artifactID, _ := artifactMap["id"].(string)
		results <- result{projectID: project1.ID, artifactID: artifactID, err: nil}
	}()

	// Simulate Request 2: Create artifact in project 2
	go func() {
		ctx2 := tools.WithProjectID(context.Background(), project2.ID)

		params := map[string]interface{}{
			"type":    "diagram",
			"name":    "artifact-from-request-2",
			"content": "Content from request 2",
		}

		res, err := tools.InvokeTool(ctx2, "create_artifact", params)

		if err != nil {
			results <- result{projectID: project2.ID, err: err}
			return
		}

		// Extract artifact from result
		artifactData, ok := res["artifact"]
		if !ok {
			results <- result{projectID: project2.ID, err: fmt.Errorf("no artifact returned: %v", res)}
			return
		}

		artifactMap, ok := artifactData.(map[string]interface{})
		if !ok {
			results <- result{projectID: project2.ID, err: fmt.Errorf("artifact is not of correct type")}
			return
		}

		artifactID, _ := artifactMap["id"].(string)
		results <- result{projectID: project2.ID, artifactID: artifactID, err: nil}
	}()

	// Collect results (order is non-deterministic)
	r1 := <-results
	r2 := <-results

	// Map results to correct projects based on projectID field
	var resultForProject1, resultForProject2 result
	if r1.projectID == project1.ID {
		resultForProject1 = r1
		resultForProject2 = r2
	} else {
		resultForProject1 = r2
		resultForProject2 = r1
	}

	// Verify both requests succeeded
	if resultForProject1.err != nil {
		t.Errorf("Request 1 failed: %v", resultForProject1.err)
	}
	if resultForProject2.err != nil {
		t.Errorf("Request 2 failed: %v", resultForProject2.err)
	}

	// Verify artifacts were created in correct projects
	if resultForProject1.artifactID != "" {
		artifact1, err := am.GetArtifact(ctx, resultForProject1.artifactID)
		if err != nil {
			t.Errorf("Failed to retrieve artifact 1: %v", err)
		} else if artifact1.ProjectID != project1.ID {
			t.Errorf("Artifact 1 in wrong project: expected %s, got %s", project1.ID, artifact1.ProjectID)
		} else if artifact1.Name != "artifact-from-request-1" {
			t.Errorf("Artifact 1 has wrong name: expected 'artifact-from-request-1', got '%s'", artifact1.Name)
		}
	}

	if resultForProject2.artifactID != "" {
		artifact2, err := am.GetArtifact(ctx, resultForProject2.artifactID)
		if err != nil {
			t.Errorf("Failed to retrieve artifact 2: %v", err)
		} else if artifact2.ProjectID != project2.ID {
			t.Errorf("Artifact 2 in wrong project: expected %s, got %s", project2.ID, artifact2.ProjectID)
		} else if artifact2.Name != "artifact-from-request-2" {
			t.Errorf("Artifact 2 has wrong name: expected 'artifact-from-request-2', got '%s'", artifact2.Name)
		}
	}
}
