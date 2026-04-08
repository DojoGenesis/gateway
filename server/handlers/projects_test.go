package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DojoGenesis/gateway/server/projects"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func setupProjectTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
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
		settings TEXT,
		metadata TEXT,
		status TEXT
	);

	CREATE TABLE IF NOT EXISTS project_templates (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		category TEXT,
		structure TEXT NOT NULL,
		default_settings TEXT,
		is_system BOOLEAN,
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
		FOREIGN KEY (project_id) REFERENCES projects(id)
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
		FOREIGN KEY (artifact_id) REFERENCES artifacts(id)
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func TestHandleCreateProject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	h := NewProjectHandler(pm)

	router := gin.New()
	router.POST("/api/v1/projects", h.CreateProject)

	reqBody := CreateProjectRequest{
		Name:        "Test Project",
		Description: "A test project",
		TemplateID:  "",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/projects", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}
}

func TestHandleListProjects(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	ctx := context.Background()
	pm.CreateProject(ctx, "Project 1", "Description 1", "")
	pm.CreateProject(ctx, "Project 2", "Description 2", "")

	h := NewProjectHandler(pm)

	router := gin.New()
	router.GET("/api/v1/projects", h.ListProjects)

	req, _ := http.NewRequest("GET", "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}

	if int(response["count"].(float64)) != 2 {
		t.Errorf("Expected 2 projects, got %v", response["count"])
	}
}

func TestHandleGetProject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")

	h := NewProjectHandler(pm)

	router := gin.New()
	router.GET("/api/v1/projects/:id", h.GetProject)

	req, _ := http.NewRequest("GET", "/api/v1/projects/"+project.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}
}

func TestHandleUpdateProject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")

	h := NewProjectHandler(pm)

	router := gin.New()
	router.PUT("/api/v1/projects/:id", h.UpdateProject)

	reqBody := UpdateProjectRequest{
		Description: "Updated description",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/api/v1/projects/"+project.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}
}

func TestHandleDeleteProject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")

	h := NewProjectHandler(pm)

	router := gin.New()
	router.DELETE("/api/v1/projects/:id", h.DeleteProject)

	req, _ := http.NewRequest("DELETE", "/api/v1/projects/"+project.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}
}

func TestHandleListProjectTemplates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	h := NewProjectHandler(pm)

	router := gin.New()
	router.GET("/api/v1/projects/templates", h.ListProjectTemplates)

	req, _ := http.NewRequest("GET", "/api/v1/projects/templates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}

	if int(response["count"].(float64)) < 1 {
		t.Errorf("Expected at least 1 template, got %v", response["count"])
	}
}

func TestHandleExportProject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Export Test Project", "Test export", "")

	h := NewProjectHandler(pm)

	router := gin.New()
	router.GET("/api/v1/projects/:id/export", h.ExportProject)

	req, _ := http.NewRequest("GET", "/api/v1/projects/"+project.ID+"/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/zip" {
		t.Errorf("Expected Content-Type application/zip, got %s", contentType)
	}

	if w.Body.Len() == 0 {
		t.Errorf("Expected non-empty zip data")
	}
}

func TestHandleCreateProject_EmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	h := NewProjectHandler(pm)

	router := gin.New()
	router.POST("/api/v1/projects", h.CreateProject)

	reqBody := CreateProjectRequest{
		Name:        "   ",
		Description: "A test project",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/projects", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["success"].(bool) {
		t.Errorf("Expected success to be false")
	}
}

func TestHandleGetProject_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupProjectTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	h := NewProjectHandler(pm)

	router := gin.New()
	router.GET("/api/v1/projects/:id", h.GetProject)

	req, _ := http.NewRequest("GET", "/api/v1/projects/nonexistent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["success"].(bool) {
		t.Errorf("Expected success to be false")
	}
}
