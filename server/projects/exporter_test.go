package projects

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupExporterTestDB(t *testing.T) (*sql.DB, *ProjectManager, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test_exporter_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp db file: %v", err)
	}
	tmpFile.Close()

	db, err := sql.Open("sqlite", tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := initExporterSchema(db); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	pm, err := NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
		if pm.projectBaseDir != "" {
			os.RemoveAll(pm.projectBaseDir)
		}
	}

	return db, pm, cleanup
}

func initExporterSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			template_id TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			last_accessed_at DATETIME NOT NULL,
			settings TEXT,
			metadata TEXT,
			status TEXT DEFAULT 'active'
		);

		CREATE TABLE IF NOT EXISTS project_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			category TEXT,
			structure TEXT NOT NULL,
			default_settings TEXT,
			is_system BOOLEAN DEFAULT FALSE,
			created_at DATETIME NOT NULL
		);

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

		CREATE TABLE IF NOT EXISTS memory_seeds (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			tier INTEGER DEFAULT 3,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			last_accessed_at DATETIME NOT NULL,
			metadata TEXT
		);
	`

	_, err := db.Exec(schema)
	return err
}

func TestExportProjectBasic(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	project, err := pm.CreateProject(ctx, "Test Export Project", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	zipData, err := pm.ExportProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	if len(zipData) == 0 {
		t.Error("Expected non-empty zip data")
	}
}

func TestExportProjectWithArtifacts(t *testing.T) {
	db, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	project, err := pm.CreateProject(ctx, "Test Artifacts Project", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	artifactQuery := `
		INSERT INTO artifacts (id, project_id, type, name, latest_version, created_at, updated_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	artifactID := "test-artifact-1"
	_, err = db.ExecContext(ctx, artifactQuery,
		artifactID,
		project.ID,
		"document",
		"Test Document",
		1,
		now,
		now,
		"{}",
	)
	if err != nil {
		t.Fatalf("Failed to create artifact: %v", err)
	}

	versionQuery := `
		INSERT INTO artifact_versions (id, artifact_id, version, content, commit_message, created_at, created_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.ExecContext(ctx, versionQuery,
		"test-version-1",
		artifactID,
		1,
		"Test content",
		"Initial version",
		now,
		"agent",
		"{}",
	)
	if err != nil {
		t.Fatalf("Failed to create artifact version: %v", err)
	}

	zipData, err := pm.ExportProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	if len(zipData) == 0 {
		t.Error("Expected non-empty zip data")
	}
}

func TestExportProjectWithFiles(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	project, err := pm.CreateProject(ctx, "Test Files Project", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	projectDir := pm.GetProjectDirectory(project.Name)
	testFile := filepath.Join(projectDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	zipData, err := pm.ExportProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	if len(zipData) == 0 {
		t.Error("Expected non-empty zip data")
	}
}

func TestImportProject(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	originalProject, err := pm.CreateProject(ctx, "Test Import Project", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	zipData, err := pm.ExportProject(ctx, originalProject.ID)
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	if err := pm.DeleteProject(ctx, originalProject.ID); err != nil {
		t.Fatalf("Failed to delete original project: %v", err)
	}

	importedProject, err := pm.ImportProject(ctx, zipData)
	if err != nil {
		t.Fatalf("Failed to import project: %v", err)
	}

	if importedProject.Name != originalProject.Name {
		t.Errorf("Expected project name %s, got %s", originalProject.Name, importedProject.Name)
	}

	if importedProject.Description != originalProject.Description {
		t.Errorf("Expected project description %s, got %s", originalProject.Description, importedProject.Description)
	}
}

func TestImportProjectWithArtifacts(t *testing.T) {
	db, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	project, err := pm.CreateProject(ctx, "Test Import Artifacts", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	artifactQuery := `
		INSERT INTO artifacts (id, project_id, type, name, latest_version, created_at, updated_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	artifactID := "test-artifact-1"
	_, err = db.ExecContext(ctx, artifactQuery,
		artifactID,
		project.ID,
		"diagram",
		"Test Diagram",
		2,
		now,
		now,
		"{}",
	)
	if err != nil {
		t.Fatalf("Failed to create artifact: %v", err)
	}

	versionQuery := `
		INSERT INTO artifact_versions (id, artifact_id, version, content, commit_message, created_at, created_by, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.ExecContext(ctx, versionQuery,
		"test-version-1",
		artifactID,
		1,
		"Version 1 content",
		"Initial version",
		now,
		"agent",
		"{}",
	)
	if err != nil {
		t.Fatalf("Failed to create artifact version: %v", err)
	}

	_, err = db.ExecContext(ctx, versionQuery,
		"test-version-2",
		artifactID,
		2,
		"Version 2 content",
		"Updated content",
		now,
		"agent",
		"{}",
	)
	if err != nil {
		t.Fatalf("Failed to create artifact version: %v", err)
	}

	zipData, err := pm.ExportProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	if err := pm.DeleteProject(ctx, project.ID); err != nil {
		t.Fatalf("Failed to delete original project: %v", err)
	}

	importedProject, err := pm.ImportProject(ctx, zipData)
	if err != nil {
		t.Fatalf("Failed to import project: %v", err)
	}

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM artifacts WHERE project_id = ?", importedProject.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count artifacts: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 artifact, got %d", count)
	}

	var versionCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM artifact_versions 
		WHERE artifact_id IN (SELECT id FROM artifacts WHERE project_id = ?)
	`, importedProject.ID).Scan(&versionCount)
	if err != nil {
		t.Fatalf("Failed to count versions: %v", err)
	}

	if versionCount != 2 {
		t.Errorf("Expected 2 artifact versions, got %d", versionCount)
	}
}

func TestImportProjectDuplicateName(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	project, err := pm.CreateProject(ctx, "Test Duplicate", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	zipData, err := pm.ExportProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	_, err = pm.ImportProject(ctx, zipData)
	if err == nil {
		t.Error("Expected error when importing project with duplicate name")
	}
}

func TestExportProjectNotFound(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	_, err := pm.ExportProject(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error when exporting non-existent project")
	}
}

func TestImportProjectInvalidZip(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	invalidZip := []byte("not a zip file")
	_, err := pm.ImportProject(ctx, invalidZip)
	if err == nil {
		t.Error("Expected error when importing invalid zip data")
	}
}
