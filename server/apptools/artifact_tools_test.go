package apptools

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/DojoGenesis/gateway/server/artifacts"
	_ "modernc.org/sqlite"
)

func setupArtifactTestDB(t *testing.T) (*sql.DB, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_artifacts.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
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
		metadata TEXT DEFAULT '{}'
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
		metadata TEXT DEFAULT '{}',
		FOREIGN KEY (artifact_id) REFERENCES artifacts(id)
	);

	CREATE INDEX IF NOT EXISTS idx_artifacts_project ON artifacts(project_id);
	CREATE INDEX IF NOT EXISTS idx_artifacts_type ON artifacts(type);
	CREATE INDEX IF NOT EXISTS idx_artifact_versions_artifact ON artifact_versions(artifact_id);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestCreateArtifact(t *testing.T) {
	db, cleanup := setupArtifactTestDB(t)
	defer cleanup()

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	InitializeArtifactTools(am)

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name: "successful creation",
			params: map[string]interface{}{
				"project_id": "project-1",
				"type":       "document",
				"name":       "Test Document",
				"content":    "# Hello World",
				"session_id": "session-1",
			},
			expectErr: false,
		},
		{
			name: "missing project_id",
			params: map[string]interface{}{
				"type":    "document",
				"name":    "Test Document",
				"content": "# Hello World",
			},
			expectErr: true,
		},
		{
			name: "missing type",
			params: map[string]interface{}{
				"project_id": "project-1",
				"name":       "Test Document",
				"content":    "# Hello World",
			},
			expectErr: true,
		},
		{
			name: "missing name",
			params: map[string]interface{}{
				"project_id": "project-1",
				"type":       "document",
				"content":    "# Hello World",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreateArtifact(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("CreateArtifact returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("CreateArtifact did not return success field")
			}

			if tt.expectErr && success {
				t.Error("Expected error but got success")
			}

			if !tt.expectErr && !success {
				t.Errorf("Expected success but got error: %v", result["error"])
			}

			if !tt.expectErr && success {
				artifact, ok := result["artifact"]
				if !ok {
					t.Error("Expected artifact in result")
				}
				if artifact == nil {
					t.Error("Artifact should not be nil")
				}
			}
		})
	}
}

func TestUpdateArtifact(t *testing.T) {
	db, cleanup := setupArtifactTestDB(t)
	defer cleanup()

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	InitializeArtifactTools(am)

	artifact, err := am.CreateArtifact(context.Background(), "project-1", "session-1", artifacts.TypeDocument, "Test Doc", "", "Initial content")
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name: "successful update",
			params: map[string]interface{}{
				"artifact_id":    artifact.ID,
				"content":        "Updated content",
				"commit_message": "Updated the document",
			},
			expectErr: false,
		},
		{
			name: "missing artifact_id",
			params: map[string]interface{}{
				"content":        "Updated content",
				"commit_message": "Updated the document",
			},
			expectErr: true,
		},
		{
			name: "missing content",
			params: map[string]interface{}{
				"artifact_id":    artifact.ID,
				"commit_message": "Updated the document",
			},
			expectErr: true,
		},
		{
			name: "non-existent artifact",
			params: map[string]interface{}{
				"artifact_id": "non-existent-id",
				"content":     "New content",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UpdateArtifact(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("UpdateArtifact returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("UpdateArtifact did not return success field")
			}

			if tt.expectErr && success {
				t.Error("Expected error but got success")
			}

			if !tt.expectErr && !success {
				t.Errorf("Expected success but got error: %v", result["error"])
			}

			if !tt.expectErr && success {
				updatedArtifact, ok := result["artifact"]
				if !ok {
					t.Error("Expected artifact in result")
				}
				if updatedArtifact == nil {
					t.Error("Artifact should not be nil")
				}
			}
		})
	}
}

func TestGetArtifact(t *testing.T) {
	db, cleanup := setupArtifactTestDB(t)
	defer cleanup()

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	InitializeArtifactTools(am)

	artifact, err := am.CreateArtifact(context.Background(), "project-1", "session-1", artifacts.TypeDocument, "Test Doc", "", "Initial content")
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name: "get artifact without version",
			params: map[string]interface{}{
				"artifact_id": artifact.ID,
			},
			expectErr: false,
		},
		{
			name: "get artifact with version 1",
			params: map[string]interface{}{
				"artifact_id": artifact.ID,
				"version":     1,
			},
			expectErr: false,
		},
		{
			name: "missing artifact_id",
			params: map[string]interface{}{
				"version": 1,
			},
			expectErr: true,
		},
		{
			name: "non-existent artifact",
			params: map[string]interface{}{
				"artifact_id": "non-existent-id",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetArtifact(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("GetArtifact returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("GetArtifact did not return success field")
			}

			if tt.expectErr && success {
				t.Error("Expected error but got success")
			}

			if !tt.expectErr && !success {
				t.Errorf("Expected success but got error: %v", result["error"])
			}

			if !tt.expectErr && success {
				returnedArtifact, ok := result["artifact"]
				if !ok {
					t.Error("Expected artifact in result")
				}
				if returnedArtifact == nil {
					t.Error("Artifact should not be nil")
				}
			}
		})
	}
}

func TestListArtifacts(t *testing.T) {
	db, cleanup := setupArtifactTestDB(t)
	defer cleanup()

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	InitializeArtifactTools(am)

	_, err = am.CreateArtifact(context.Background(), "project-1", "session-1", artifacts.TypeDocument, "Doc 1", "", "Content 1")
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	_, err = am.CreateArtifact(context.Background(), "project-1", "session-1", artifacts.TypeDiagram, "Diagram 1", "", "graph TD")
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	_, err = am.CreateArtifact(context.Background(), "project-2", "session-1", artifacts.TypeDocument, "Doc 2", "", "Content 2")
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	tests := []struct {
		name          string
		params        map[string]interface{}
		expectedCount int
	}{
		{
			name:          "list all artifacts",
			params:        map[string]interface{}{},
			expectedCount: 3,
		},
		{
			name: "list artifacts for project-1",
			params: map[string]interface{}{
				"project_id": "project-1",
			},
			expectedCount: 2,
		},
		{
			name: "list document artifacts",
			params: map[string]interface{}{
				"type": "document",
			},
			expectedCount: 2,
		},
		{
			name: "list diagram artifacts for project-1",
			params: map[string]interface{}{
				"project_id": "project-1",
				"type":       "diagram",
			},
			expectedCount: 1,
		},
		{
			name: "list with limit",
			params: map[string]interface{}{
				"limit": 2,
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ListArtifacts(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("ListArtifacts returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("ListArtifacts did not return success field")
			}

			if !success {
				t.Errorf("Expected success but got error: %v", result["error"])
			}

			count, ok := result["count"].(int)
			if !ok {
				t.Fatal("ListArtifacts did not return count field")
			}

			if count != tt.expectedCount {
				t.Errorf("Expected %d artifacts, got %d", tt.expectedCount, count)
			}
		})
	}
}

func TestListArtifactVersions(t *testing.T) {
	db, cleanup := setupArtifactTestDB(t)
	defer cleanup()

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	InitializeArtifactTools(am)

	artifact, err := am.CreateArtifact(context.Background(), "project-1", "session-1", artifacts.TypeDocument, "Test Doc", "", "Initial content")
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	_, err = am.UpdateArtifact(context.Background(), artifact.ID, "Updated content", "Update 1")
	if err != nil {
		t.Fatalf("Failed to update artifact: %v", err)
	}

	_, err = am.UpdateArtifact(context.Background(), artifact.ID, "Updated content again", "Update 2")
	if err != nil {
		t.Fatalf("Failed to update artifact: %v", err)
	}

	tests := []struct {
		name          string
		params        map[string]interface{}
		expectedCount int
		expectErr     bool
	}{
		{
			name: "list versions",
			params: map[string]interface{}{
				"artifact_id": artifact.ID,
			},
			expectedCount: 3,
			expectErr:     false,
		},
		{
			name:          "missing artifact_id",
			params:        map[string]interface{}{},
			expectedCount: 0,
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ListArtifactVersions(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("ListArtifactVersions returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("ListArtifactVersions did not return success field")
			}

			if tt.expectErr && success {
				t.Error("Expected error but got success")
			}

			if !tt.expectErr && !success {
				t.Errorf("Expected success but got error: %v", result["error"])
			}

			if !tt.expectErr && success {
				count, ok := result["count"].(int)
				if !ok {
					t.Fatal("ListArtifactVersions did not return count field")
				}

				if count != tt.expectedCount {
					t.Errorf("Expected %d versions, got %d", tt.expectedCount, count)
				}
			}
		})
	}
}

func TestExportArtifact(t *testing.T) {
	db, cleanup := setupArtifactTestDB(t)
	defer cleanup()

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	InitializeArtifactTools(am)

	artifact, err := am.CreateArtifact(context.Background(), "project-1", "session-1", artifacts.TypeDocument, "Test Doc", "", "# Hello World")
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name: "export as raw",
			params: map[string]interface{}{
				"artifact_id": artifact.ID,
				"format":      "raw",
			},
			expectErr: false,
		},
		{
			name: "export as base64",
			params: map[string]interface{}{
				"artifact_id": artifact.ID,
				"format":      "base64",
			},
			expectErr: false,
		},
		{
			name: "export as json",
			params: map[string]interface{}{
				"artifact_id": artifact.ID,
				"format":      "json",
			},
			expectErr: false,
		},
		{
			name: "export with default format",
			params: map[string]interface{}{
				"artifact_id": artifact.ID,
			},
			expectErr: false,
		},
		{
			name: "missing artifact_id",
			params: map[string]interface{}{
				"format": "raw",
			},
			expectErr: true,
		},
		{
			name: "unsupported format",
			params: map[string]interface{}{
				"artifact_id": artifact.ID,
				"format":      "unsupported",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExportArtifact(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("ExportArtifact returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("ExportArtifact did not return success field")
			}

			if tt.expectErr && success {
				t.Error("Expected error but got success")
			}

			if !tt.expectErr && !success {
				t.Errorf("Expected success but got error: %v", result["error"])
			}

			if !tt.expectErr && success {
				_, ok := result["content"]
				if !ok {
					t.Error("Expected content in result")
				}

				_, ok = result["filename"]
				if !ok {
					t.Error("Expected filename in result")
				}

				_, ok = result["mime_type"]
				if !ok {
					t.Error("Expected mime_type in result")
				}
			}
		})
	}
}

func TestArtifactManagerNotInitialized(t *testing.T) {
	artifactManager = nil

	result, err := CreateArtifact(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("CreateArtifact returned error: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok {
		t.Fatal("CreateArtifact did not return success field")
	}

	if success {
		t.Error("Expected failure when artifact manager not initialized")
	}

	errorMsg, ok := result["error"].(string)
	if !ok {
		t.Fatal("Expected error message when artifact manager not initialized")
	}

	if errorMsg != "artifact manager not initialized" {
		t.Errorf("Expected 'artifact manager not initialized', got '%s'", errorMsg)
	}
}
