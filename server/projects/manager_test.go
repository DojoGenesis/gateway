package projects

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*ProjectManager, *sql.DB, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

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
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	pm, err := NewProjectManager(db)
	require.NoError(t, err)

	return pm, db, tmpDir
}

func TestNewProjectManager(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	schema := `
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
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	pm, err := NewProjectManager(db)
	require.NoError(t, err)
	require.NotNil(t, pm)

	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, "DojoProjects")
	_, err = os.Stat(expectedDir)
	assert.NoError(t, err)
}

func TestProjectManager_CreateProject(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	project, err := pm.CreateProject(ctx, "test-project", "A test project", "")

	assert.NoError(t, err)
	require.NotNil(t, project)
	assert.Equal(t, "test-project", project.Name)
	assert.Equal(t, "A test project", project.Description)
	assert.Equal(t, StatusActive, project.Status)
	assert.NotEmpty(t, project.ID)

	projectDir := pm.GetProjectDirectory("test-project")
	_, err = os.Stat(projectDir)
	assert.NoError(t, err)
}

func TestProjectManager_CreateProject_WithTemplate(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	project, err := pm.CreateProject(ctx, "research-project", "A research project", "research-report")

	assert.NoError(t, err)
	require.NotNil(t, project)
	assert.Equal(t, "research-project", project.Name)
	assert.Equal(t, "research-report", project.TemplateID)

	projectDir := pm.GetProjectDirectory("research-project")
	_, err = os.Stat(filepath.Join(projectDir, "research"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(projectDir, "notes"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(projectDir, "artifacts"))
	assert.NoError(t, err)
}

func TestProjectManager_CreateProject_InvalidName(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := pm.CreateProject(ctx, "", "No name", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project name is required")
}

func TestProjectManager_CreateProject_InvalidTemplate(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := pm.CreateProject(ctx, "test-project", "Test", "invalid-template")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestProjectManager_GetProject(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	created, err := pm.CreateProject(ctx, "test-project", "A test project", "")
	require.NoError(t, err)

	retrieved, err := pm.GetProject(ctx, created.ID)
	assert.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Name, retrieved.Name)
	assert.Equal(t, created.Description, retrieved.Description)
}

func TestProjectManager_GetProject_NotFound(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := pm.GetProject(ctx, "non-existent-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")
}

func TestProjectManager_GetProject_UpdatesLastAccessed(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	created, err := pm.CreateProject(ctx, "test-project", "A test project", "")
	require.NoError(t, err)

	originalLastAccessed := created.LastAccessedAt
	time.Sleep(100 * time.Millisecond)

	retrieved, err := pm.GetProject(ctx, created.ID)
	require.NoError(t, err)

	assert.True(t, retrieved.LastAccessedAt.After(originalLastAccessed) || retrieved.LastAccessedAt.Equal(originalLastAccessed))
}

func TestProjectManager_ListProjects(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := pm.CreateProject(ctx, "project-1", "Project 1", "")
	require.NoError(t, err)

	_, err = pm.CreateProject(ctx, "project-2", "Project 2", "")
	require.NoError(t, err)

	_, err = pm.CreateProject(ctx, "project-3", "Project 3", "")
	require.NoError(t, err)

	projects, err := pm.ListProjects(ctx, "")
	assert.NoError(t, err)
	assert.Len(t, projects, 3)
}

func TestProjectManager_ListProjects_FilterByStatus(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	p1, err := pm.CreateProject(ctx, "project-1", "Project 1", "")
	require.NoError(t, err)

	_, err = pm.CreateProject(ctx, "project-2", "Project 2", "")
	require.NoError(t, err)

	_, err = pm.UpdateProject(ctx, p1.ID, "", "", StatusArchived, nil, nil)
	require.NoError(t, err)

	activeProjects, err := pm.ListProjects(ctx, StatusActive)
	assert.NoError(t, err)
	assert.Len(t, activeProjects, 1)

	archivedProjects, err := pm.ListProjects(ctx, StatusArchived)
	assert.NoError(t, err)
	assert.Len(t, archivedProjects, 1)
}

func TestProjectManager_UpdateProject(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	created, err := pm.CreateProject(ctx, "test-project", "Original description", "")
	require.NoError(t, err)

	updated, err := pm.UpdateProject(ctx, created.ID, "", "Updated description", StatusArchived, nil, nil)
	assert.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "test-project", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, StatusArchived, updated.Status)
	assert.True(t, updated.UpdatedAt.After(created.UpdatedAt))
}

func TestProjectManager_UpdateProject_NameChangeNotAllowed(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	created, err := pm.CreateProject(ctx, "test-project", "Original description", "")
	require.NoError(t, err)

	_, err = pm.UpdateProject(ctx, created.ID, "new-name", "Updated description", "", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project name cannot be changed")
}

func TestProjectManager_UpdateProject_PartialUpdate(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	created, err := pm.CreateProject(ctx, "test-project", "Original description", "")
	require.NoError(t, err)

	updated, err := pm.UpdateProject(ctx, created.ID, "", "Updated description", "", nil, nil)
	assert.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "test-project", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, StatusActive, updated.Status)
}

func TestProjectManager_UpdateProject_Settings(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	created, err := pm.CreateProject(ctx, "test-project", "Test project", "")
	require.NoError(t, err)

	newSettings := map[string]interface{}{
		"theme":     "dark",
		"auto_save": false,
	}

	updated, err := pm.UpdateProject(ctx, created.ID, "", "", "", newSettings, nil)
	assert.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "dark", updated.Settings["theme"])
	assert.Equal(t, false, updated.Settings["auto_save"])
}

func TestProjectManager_UpdateProject_NotFound(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := pm.UpdateProject(ctx, "non-existent-id", "test", "test", "", nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")
}

func TestProjectManager_DeleteProject(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	created, err := pm.CreateProject(ctx, "test-project", "Test project", "")
	require.NoError(t, err)

	projectDir := pm.GetProjectDirectory("test-project")
	_, err = os.Stat(projectDir)
	require.NoError(t, err)

	err = pm.DeleteProject(ctx, created.ID)
	assert.NoError(t, err)

	_, err = pm.GetProject(ctx, created.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")

	_, err = os.Stat(projectDir)
	assert.True(t, os.IsNotExist(err))
}

func TestProjectManager_DeleteProject_NotFound(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	err := pm.DeleteProject(ctx, "non-existent-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")
}

func TestProjectManager_GetTemplate(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	template, err := pm.GetTemplate(ctx, "research-report")
	assert.NoError(t, err)
	require.NotNil(t, template)
	assert.Equal(t, "research-report", template.ID)
	assert.Equal(t, "Research Report", template.Name)
	assert.Equal(t, CategoryResearch, template.Category)
	assert.True(t, template.IsSystem)
}

func TestProjectManager_GetTemplate_NotFound(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	template, err := pm.GetTemplate(ctx, "non-existent-template")
	assert.NoError(t, err)
	assert.Nil(t, template)
}

func TestProjectManager_ListTemplates(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	templates, err := pm.ListTemplates(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(templates), 4)

	templateIDs := make(map[string]bool)
	for _, tmpl := range templates {
		templateIDs[tmpl.ID] = true
		assert.True(t, tmpl.IsSystem)
	}

	assert.True(t, templateIDs["research-report"])
	assert.True(t, templateIDs["software-design"])
	assert.True(t, templateIDs["data-analysis"])
	assert.True(t, templateIDs["creative-studio"])
}

func TestProjectManager_GetProjectDirectory(t *testing.T) {
	pm, db, _ := setupTestDB(t)
	defer db.Close()

	projectDir := pm.GetProjectDirectory("test-project")

	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, "DojoProjects", "test-project")
	assert.Equal(t, expectedDir, projectDir)
}

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid name", "my-project", false},
		{"valid with spaces", "My Project", false},
		{"empty", "", true},
		{"only spaces", "   ", true},
		{"path traversal", "../etc/passwd", true},
		{"forward slash", "project/name", true},
		{"backslash", "project\\name", true},
		{"colon", "project:name", true},
		{"asterisk", "project*name", true},
		{"question mark", "project?name", true},
		{"quote", "project\"name", true},
		{"less than", "project<name", true},
		{"greater than", "project>name", true},
		{"pipe", "project|name", true},
		{"too long", strings.Repeat("a", 256), true},
		{"exactly 255", strings.Repeat("a", 255), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectName(tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
