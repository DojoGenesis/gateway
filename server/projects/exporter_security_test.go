package projects

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestImportProjectPathTraversal(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	manifest := `{
		"dojo_packet_version": "1.0",
		"exported_at": "2026-01-31T10:00:00Z",
		"project": {
			"id": "test-id",
			"name": "PathTraversalTest",
			"description": "Test",
			"settings": {},
			"metadata": {},
			"created_at": "2026-01-31T10:00:00Z"
		},
		"artifacts": [],
		"files": [],
		"metadata": {}
	}`

	manifestWriter, _ := zipWriter.Create("dojo_packet_manifest.json")
	manifestWriter.Write([]byte(manifest))

	maliciousFile, _ := zipWriter.Create("files/../../../../../../tmp/malicious.txt")
	maliciousFile.Write([]byte("This should not be written"))

	zipWriter.Close()

	_, err := pm.ImportProject(ctx, buf.Bytes())
	if err == nil {
		t.Error("Expected error for path traversal attempt")
	}

	if !strings.Contains(err.Error(), "path traversal") && !strings.Contains(err.Error(), "invalid file path") {
		t.Errorf("Expected path traversal error, got: %v", err)
	}

	if _, err := os.Stat("/tmp/malicious.txt"); err == nil {
		t.Error("Malicious file was created outside project directory")
		os.Remove("/tmp/malicious.txt")
	}
}

func TestImportProjectAbsolutePath(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	manifest := `{
		"dojo_packet_version": "1.0",
		"exported_at": "2026-01-31T10:00:00Z",
		"project": {
			"id": "test-id",
			"name": "AbsolutePathTest",
			"description": "Test",
			"settings": {},
			"metadata": {},
			"created_at": "2026-01-31T10:00:00Z"
		},
		"artifacts": [],
		"files": [],
		"metadata": {}
	}`

	manifestWriter, _ := zipWriter.Create("dojo_packet_manifest.json")
	manifestWriter.Write([]byte(manifest))

	absolutePathFile, _ := zipWriter.Create("files//etc/passwd")
	absolutePathFile.Write([]byte("malicious content"))

	zipWriter.Close()

	_, err := pm.ImportProject(ctx, buf.Bytes())
	if err == nil {
		t.Error("Expected error for absolute path attempt")
	}

	if !strings.Contains(err.Error(), "invalid file path") {
		t.Errorf("Expected invalid file path error, got: %v", err)
	}
}

func TestImportProjectOrphanedDirectoryCleanup(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	manifest := `{
		"dojo_packet_version": "1.0",
		"exported_at": "2026-01-31T10:00:00Z",
		"project": {
			"id": "test-id",
			"name": "CleanupTest",
			"description": "Test",
			"settings": {},
			"metadata": {},
			"created_at": "2026-01-31T10:00:00Z"
		},
		"artifacts": [
			{
				"id": "bad-artifact",
				"type": "document",
				"name": "Test",
				"versions": []
			}
		],
		"files": [],
		"metadata": {}
	}`

	manifestWriter, _ := zipWriter.Create("dojo_packet_manifest.json")
	manifestWriter.Write([]byte(manifest))

	zipWriter.Close()

	projectDir := filepath.Join(pm.projectBaseDir, "CleanupTest")

	_, err := pm.ImportProject(ctx, buf.Bytes())
	if err == nil {
		t.Error("Expected error due to artifact with no versions")
	}

	if _, err := os.Stat(projectDir); err == nil {
		t.Error("Expected project directory to be cleaned up on import failure")
	}
}

func TestExportProjectWithMemorySeeds(t *testing.T) {
	db, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	project, err := pm.CreateProject(ctx, "Memory Test Project", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	seedQuery := `
		INSERT INTO memory_seeds (id, project_id, type, content, tier, created_at, updated_at, last_accessed_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.ExecContext(ctx, seedQuery,
		"test-seed-1",
		project.ID,
		"fact",
		"Test memory content",
		3,
		"2026-01-31T10:00:00Z",
		"2026-01-31T10:00:00Z",
		"2026-01-31T10:00:00Z",
		"{}",
	)
	if err != nil {
		t.Fatalf("Failed to create memory seed: %v", err)
	}

	zipData, err := pm.ExportProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	if len(zipData) == 0 {
		t.Error("Expected non-empty zip data")
	}

	if err := pm.DeleteProject(ctx, project.ID); err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	importedProject, err := pm.ImportProject(ctx, zipData)
	if err != nil {
		t.Fatalf("Failed to import project: %v", err)
	}

	var seedCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memory_seeds WHERE project_id = ?", importedProject.ID).Scan(&seedCount)
	if err != nil {
		t.Fatalf("Failed to count memory seeds: %v", err)
	}

	if seedCount != 1 {
		t.Errorf("Expected 1 memory seed, got %d", seedCount)
	}
}

func TestExportProjectWithNestedFiles(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	project, err := pm.CreateProject(ctx, "Nested Files Test", "Test Description", "")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	projectDir := pm.GetProjectDirectory(project.Name)

	nestedDir := filepath.Join(projectDir, "subdir1", "subdir2")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	testFile := filepath.Join(nestedDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Nested content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	zipData, err := pm.ExportProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	if err := pm.DeleteProject(ctx, project.ID); err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	importedProject, err := pm.ImportProject(ctx, zipData)
	if err != nil {
		t.Fatalf("Failed to import project: %v", err)
	}

	importedProjectDir := pm.GetProjectDirectory(importedProject.Name)
	importedFile := filepath.Join(importedProjectDir, "subdir1", "subdir2", "test.txt")

	if _, err := os.Stat(importedFile); os.IsNotExist(err) {
		t.Error("Expected nested file to exist in imported project")
	} else {
		content, err := os.ReadFile(importedFile)
		if err != nil {
			t.Fatalf("Failed to read imported file: %v", err)
		}

		if string(content) != "Nested content" {
			t.Errorf("Expected content 'Nested content', got %q", string(content))
		}
	}
}

func TestDojoPacketVersionValidation(t *testing.T) {
	_, pm, cleanup := setupExporterTestDB(t)
	defer cleanup()

	ctx := context.Background()

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	manifest := `{
		"dojo_packet_version": "2.0",
		"exported_at": "2026-01-31T10:00:00Z",
		"project": {
			"id": "test-id",
			"name": "VersionTest",
			"description": "Test",
			"settings": {},
			"metadata": {},
			"created_at": "2026-01-31T10:00:00Z"
		},
		"artifacts": [],
		"files": [],
		"metadata": {}
	}`

	manifestWriter, _ := zipWriter.Create("dojo_packet_manifest.json")
	manifestWriter.Write([]byte(manifest))

	zipWriter.Close()

	_, err := pm.ImportProject(ctx, buf.Bytes())
	if err == nil {
		t.Error("Expected error for unsupported DojoPacket version")
	}

	if !strings.Contains(err.Error(), "unsupported DojoPacket version") {
		t.Errorf("Expected version error, got: %v", err)
	}
}
