package apptools

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/DojoGenesis/gateway/server/projects"
	"github.com/DojoGenesis/gateway/tools"
	_ "modernc.org/sqlite"
)

func setupProjectToolsTest(t *testing.T) (*sql.DB, *projects.ProjectManager) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

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

	CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at);
	CREATE INDEX IF NOT EXISTS idx_projects_last_accessed_at ON projects(last_accessed_at);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
	})

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	InitializeProjectTools(pm)

	return db, pm
}

func TestCreateProjectTool(t *testing.T) {
	db, _ := setupProjectToolsTest(t)
	defer db.Close()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
		wantError   string
	}{
		{
			name: "Valid project creation",
			params: map[string]interface{}{
				"name":        "Test Project",
				"description": "A test project",
			},
			wantSuccess: true,
		},
		{
			name: "Valid project with template",
			params: map[string]interface{}{
				"name":        "Research Project",
				"description": "A research project",
				"template_id": "research-report",
			},
			wantSuccess: true,
		},
		{
			name:        "Missing name",
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantError:   "project name is required",
		},
		{
			name: "Empty name",
			params: map[string]interface{}{
				"name": "",
			},
			wantSuccess: false,
			wantError:   "project name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreateProject(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("CreateProject returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("Result missing 'success' field")
			}

			if success != tt.wantSuccess {
				t.Errorf("Expected success=%v, got %v", tt.wantSuccess, success)
			}

			if !tt.wantSuccess {
				errMsg, ok := result["error"].(string)
				if !ok {
					t.Fatal("Result missing 'error' field")
				}
				if tt.wantError != "" && errMsg != tt.wantError {
					t.Errorf("Expected error=%q, got %q", tt.wantError, errMsg)
				}
			} else {
				if _, ok := result["project"]; !ok {
					t.Error("Expected 'project' field in successful result")
				}
			}
		})
	}
}

func TestListProjectsTool(t *testing.T) {
	db, pm := setupProjectToolsTest(t)
	defer db.Close()

	ctx := context.Background()
	_, err := pm.CreateProject(ctx, "Project 1", "Description 1", "")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	_, err = pm.CreateProject(ctx, "Project 2", "Description 2", "")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantCount int
	}{
		{
			name:      "List all projects",
			params:    map[string]interface{}{},
			wantCount: 2,
		},
		{
			name: "List active projects",
			params: map[string]interface{}{
				"status": "active",
			},
			wantCount: 2,
		},
		{
			name: "List archived projects",
			params: map[string]interface{}{
				"status": "archived",
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ListProjects(ctx, tt.params)
			if err != nil {
				t.Fatalf("ListProjects returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok || !success {
				t.Fatal("Expected successful result")
			}

			count, ok := result["count"].(int)
			if !ok {
				t.Fatal("Result missing 'count' field")
			}

			if count != tt.wantCount {
				t.Errorf("Expected count=%d, got %d", tt.wantCount, count)
			}
		})
	}
}

func TestGetProjectTool(t *testing.T) {
	db, pm := setupProjectToolsTest(t)
	defer db.Close()

	ctx := context.Background()
	project, err := pm.CreateProject(ctx, "Test Project", "Description", "")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
	}{
		{
			name: "Valid project ID",
			params: map[string]interface{}{
				"project_id": project.ID,
			},
			wantSuccess: true,
		},
		{
			name:        "Missing project ID",
			params:      map[string]interface{}{},
			wantSuccess: false,
		},
		{
			name: "Invalid project ID",
			params: map[string]interface{}{
				"project_id": "nonexistent",
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetProject(ctx, tt.params)
			if err != nil {
				t.Fatalf("GetProject returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("Result missing 'success' field")
			}

			if success != tt.wantSuccess {
				t.Errorf("Expected success=%v, got %v", tt.wantSuccess, success)
			}

			if tt.wantSuccess {
				if _, ok := result["project"]; !ok {
					t.Error("Expected 'project' field in successful result")
				}
			}
		})
	}
}

func TestSwitchProjectTool(t *testing.T) {
	db, pm := setupProjectToolsTest(t)
	defer db.Close()

	ctx := context.Background()
	project, err := pm.CreateProject(ctx, "Test Project", "Description", "")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
	}{
		{
			name: "Valid switch",
			params: map[string]interface{}{
				"project_id": project.ID,
			},
			wantSuccess: true,
		},
		{
			name:        "Missing project ID",
			params:      map[string]interface{}{},
			wantSuccess: false,
		},
		{
			name: "Invalid project ID",
			params: map[string]interface{}{
				"project_id": "nonexistent",
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SwitchProject(ctx, tt.params)
			if err != nil {
				t.Fatalf("SwitchProject returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("Result missing 'success' field")
			}

			if success != tt.wantSuccess {
				t.Errorf("Expected success=%v, got %v", tt.wantSuccess, success)
			}

			if tt.wantSuccess {
				if _, ok := result["project_id"]; !ok {
					t.Error("Expected 'project_id' field in successful result")
				}
			}
		})
	}
}

func TestListTemplates(t *testing.T) {
	db, _ := setupProjectToolsTest(t)
	defer db.Close()

	result, err := ListTemplates(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Fatal("Expected successful result")
	}

	count, ok := result["count"].(int)
	if !ok {
		t.Fatal("Result missing 'count' field")
	}

	if count < 4 {
		t.Errorf("Expected at least 4 default templates, got %d", count)
	}
}

func TestUpdateProjectTool(t *testing.T) {
	db, pm := setupProjectToolsTest(t)
	defer db.Close()

	ctx := context.Background()
	project, err := pm.CreateProject(ctx, "Test Project", "Original description", "")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
	}{
		{
			name: "Update description",
			params: map[string]interface{}{
				"project_id":  project.ID,
				"description": "Updated description",
			},
			wantSuccess: true,
		},
		{
			name: "Update status",
			params: map[string]interface{}{
				"project_id": project.ID,
				"status":     "archived",
			},
			wantSuccess: true,
		},
		{
			name:        "Missing project ID",
			params:      map[string]interface{}{},
			wantSuccess: false,
		},
		{
			name: "Empty update (no fields provided)",
			params: map[string]interface{}{
				"project_id": project.ID,
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UpdateProject(ctx, tt.params)
			if err != nil {
				t.Fatalf("UpdateProject returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("Result missing 'success' field")
			}

			if success != tt.wantSuccess {
				t.Errorf("Expected success=%v, got %v", tt.wantSuccess, success)
			}
		})
	}
}

func TestDeleteProjectTool(t *testing.T) {
	db, pm := setupProjectToolsTest(t)
	defer db.Close()

	ctx := context.Background()
	project, err := pm.CreateProject(ctx, "Test Project", "Description", "")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantSuccess bool
	}{
		{
			name: "Valid deletion",
			params: map[string]interface{}{
				"project_id": project.ID,
			},
			wantSuccess: true,
		},
		{
			name:        "Missing project ID",
			params:      map[string]interface{}{},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeleteProject(ctx, tt.params)
			if err != nil {
				t.Fatalf("DeleteProject returned error: %v", err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("Result missing 'success' field")
			}

			if success != tt.wantSuccess {
				t.Errorf("Expected success=%v, got %v", tt.wantSuccess, success)
			}
		})
	}
}

func TestProjectToolsNotInitialized(t *testing.T) {
	InitializeProjectTools(nil)
	defer func() {
		db, pm := setupProjectToolsTest(t)
		defer db.Close()
		InitializeProjectTools(pm)
	}()

	ctx := context.Background()

	tests := []struct {
		name string
		fn   func(context.Context, map[string]interface{}) (map[string]interface{}, error)
	}{
		{"CreateProject", CreateProject},
		{"ListProjects", ListProjects},
		{"GetProject", GetProject},
		{"SwitchProject", SwitchProject},
		{"ListTemplates", ListTemplates},
		{"UpdateProject", UpdateProject},
		{"DeleteProject", DeleteProject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.fn(ctx, map[string]interface{}{})
			if err != nil {
				t.Fatalf("%s returned error: %v", tt.name, err)
			}

			success, ok := result["success"].(bool)
			if !ok {
				t.Fatal("Result missing 'success' field")
			}

			if success {
				t.Error("Expected failure when project manager not initialized")
			}

			errMsg, ok := result["error"].(string)
			if !ok {
				t.Fatal("Result missing 'error' field")
			}

			expectedErr := "project manager not initialized"
			if errMsg != expectedErr {
				t.Errorf("Expected error=%q, got %q", expectedErr, errMsg)
			}
		})
	}
}

func TestProjectToolsRegistration(t *testing.T) {
	expectedTools := []string{
		"create_project",
		"list_projects",
		"get_project",
		"switch_project",
		"list_templates",
		"update_project",
		"delete_project",
	}

	for _, toolName := range expectedTools {
		t.Run(toolName, func(t *testing.T) {
			tool, err := tools.GetTool(toolName)
			if err != nil {
				t.Fatalf("Tool %s not registered: %v", toolName, err)
			}

			if tool.Name != toolName {
				t.Errorf("Expected tool name %s, got %s", toolName, tool.Name)
			}

			if tool.Description == "" {
				t.Error("Tool description is empty")
			}

			if tool.Parameters == nil {
				t.Error("Tool parameters are nil")
			}

			if tool.Function == nil {
				t.Error("Tool function is nil")
			}
		})
	}
}

func TestInvokeToolIntegration(t *testing.T) {
	db, _ := setupProjectToolsTest(t)
	defer db.Close()

	ctx := context.Background()

	result, err := tools.InvokeTool(ctx, "create_project", map[string]interface{}{
		"name":        "Integration Test Project",
		"description": "Created via InvokeTool",
	})

	if err != nil {
		t.Fatalf("InvokeTool failed: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Fatalf("Expected successful result, got: %v", result)
	}

	projectData, ok := result["project"]
	if !ok {
		t.Fatal("Result missing 'project' field")
	}

	projectMap, ok := projectData.(map[string]interface{})
	if !ok {
		project, ok := projectData.(*projects.Project)
		if !ok {
			t.Fatalf("Expected project to be *projects.Project or map, got %T", projectData)
		}
		projectMap = map[string]interface{}{
			"id":   project.ID,
			"name": project.Name,
		}
	}

	projectID, ok := projectMap["id"].(string)
	if !ok {
		t.Fatal("Project ID not found")
	}

	listResult, err := tools.InvokeTool(ctx, "list_projects", map[string]interface{}{})
	if err != nil {
		t.Fatalf("InvokeTool for list_projects failed: %v", err)
	}

	success, ok = listResult["success"].(bool)
	if !ok || !success {
		t.Fatalf("Expected successful list result, got: %v", listResult)
	}

	getResult, err := tools.InvokeTool(ctx, "get_project", map[string]interface{}{
		"project_id": projectID,
	})

	if err != nil {
		t.Fatalf("InvokeTool for get_project failed: %v", err)
	}

	success, ok = getResult["success"].(bool)
	if !ok || !success {
		t.Fatalf("Expected successful get result, got: %v", getResult)
	}
}
