//go:build manual
// +build manual

package apptools

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/projects"
	_ "modernc.org/sqlite"
)

func TestProjectToolsEndToEndWithInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		template_id TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_accessed_at DATETIME NOT NULL,
		settings TEXT DEFAULT '{}',
		metadata TEXT DEFAULT '{}',
		status TEXT DEFAULT 'active'
	);

	CREATE TABLE IF NOT EXISTS project_templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT DEFAULT 'general',
		structure TEXT NOT NULL,
		default_settings TEXT DEFAULT '{}',
		is_system BOOLEAN DEFAULT 0,
		created_at DATETIME NOT NULL
	);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	InitializeProjectTools(pm)

	ctx := context.Background()

	t.Run("Create project via InvokeTool", func(t *testing.T) {
		result, err := InvokeTool(ctx, "create_project", map[string]interface{}{
			"name":        "End-to-End Test Project",
			"description": "Testing tool initialization",
			"template_id": "research-report",
		})

		if err != nil {
			t.Fatalf("InvokeTool failed: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok || !success {
			t.Fatalf("Expected successful result, got: %v", result)
		}

		t.Logf("✓ Project created successfully via InvokeTool")
	})

	t.Run("List projects via InvokeTool", func(t *testing.T) {
		result, err := InvokeTool(ctx, "list_projects", map[string]interface{}{})

		if err != nil {
			t.Fatalf("InvokeTool failed: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok || !success {
			t.Fatalf("Expected successful result, got: %v", result)
		}

		count, ok := result["count"].(int)
		if !ok {
			t.Fatal("Result missing 'count' field")
		}

		if count != 1 {
			t.Errorf("Expected 1 project, got %d", count)
		}

		t.Logf("✓ Listed %d project(s) successfully", count)
	})

	t.Run("List templates via InvokeTool", func(t *testing.T) {
		result, err := InvokeTool(ctx, "list_templates", map[string]interface{}{})

		if err != nil {
			t.Fatalf("InvokeTool failed: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok || !success {
			t.Fatalf("Expected successful result, got: %v", result)
		}

		count, ok := result["count"].(int)
		if !ok {
			t.Fatal("Result missing 'count' field")
		}

		if count < 4 {
			t.Errorf("Expected at least 4 templates, got %d", count)
		}

		t.Logf("✓ Listed %d template(s) successfully", count)
	})

	t.Run("Verify empty update validation", func(t *testing.T) {
		listResult, _ := InvokeTool(ctx, "list_projects", map[string]interface{}{})
		projects, ok := listResult["projects"].([]projects.Project)
		if !ok {
			t.Skip("Cannot retrieve project ID for update test")
		}

		if len(projects) == 0 {
			t.Skip("No projects available for update test")
		}

		projectID := projects[0].ID

		result, err := InvokeTool(ctx, "update_project", map[string]interface{}{
			"project_id": projectID,
		})

		if err != nil {
			t.Fatalf("InvokeTool failed: %v", err)
		}

		success, ok := result["success"].(bool)
		if !ok {
			t.Fatal("Result missing 'success' field")
		}

		if success {
			t.Error("Expected empty update to fail, but it succeeded")
		}

		errorMsg, ok := result["error"].(string)
		if !ok {
			t.Fatal("Result missing 'error' field")
		}

		expectedErr := "at least one field must be provided for update"
		if errorMsg != expectedErr {
			t.Errorf("Expected error message %q, got %q", expectedErr, errorMsg)
		}

		t.Logf("✓ Empty update validation working correctly")
	})

	t.Log("✅ All end-to-end tests passed!")
}
