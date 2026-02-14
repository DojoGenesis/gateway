package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/artifacts"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/projects"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func setupArtifactTestDB(t *testing.T) *sql.DB {
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

func TestHandleCreateArtifact(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.POST("/api/v1/artifacts", h.CreateArtifact)

	reqBody := CreateArtifactRequest{
		ProjectID: project.ID,
		Type:      "document",
		Name:      "Test Document",
		Content:   "This is test content",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/artifacts", bytes.NewBuffer(body))
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

func TestHandleListArtifacts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")

	am.CreateArtifact(ctx, project.ID, "", artifacts.ArtifactType("document"), "Doc 1", "", "Content 1")
	am.CreateArtifact(ctx, project.ID, "", artifacts.ArtifactType("document"), "Doc 2", "", "Content 2")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.GET("/api/v1/artifacts", h.ListArtifacts)

	req, _ := http.NewRequest("GET", "/api/v1/artifacts?project_id="+project.ID, nil)
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
		t.Errorf("Expected 2 artifacts, got %v", response["count"])
	}
}

func TestHandleGetArtifact(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")
	artifact, _ := am.CreateArtifact(ctx, project.ID, "", artifacts.ArtifactType("document"), "Test Doc", "", "Content")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.GET("/api/v1/artifacts/:id", h.GetArtifact)

	req, _ := http.NewRequest("GET", "/api/v1/artifacts/"+artifact.ID, nil)
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

func TestHandleUpdateArtifact(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")
	artifact, _ := am.CreateArtifact(ctx, project.ID, "", artifacts.ArtifactType("document"), "Test Doc", "", "Original content")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.PUT("/api/v1/artifacts/:id", h.UpdateArtifact)

	reqBody := UpdateArtifactRequest{
		Content:       "Updated content",
		CommitMessage: "Updated the content",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/api/v1/artifacts/"+artifact.ID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}

	artifactData := response["artifact"].(map[string]interface{})
	if int(artifactData["latest_version"].(float64)) != 2 {
		t.Errorf("Expected version 2, got %v", artifactData["latest_version"])
	}
}

func TestHandleListArtifactVersions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")
	artifact, _ := am.CreateArtifact(ctx, project.ID, "", artifacts.ArtifactType("document"), "Test Doc", "", "Original content")
	am.UpdateArtifact(ctx, artifact.ID, "Updated content", "Update 1")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.GET("/api/v1/artifacts/:id/versions", h.ListArtifactVersions)

	req, _ := http.NewRequest("GET", "/api/v1/artifacts/"+artifact.ID+"/versions", nil)
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
		t.Errorf("Expected 2 versions, got %v", response["count"])
	}
}

func TestHandleDeleteArtifact(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")
	artifact, _ := am.CreateArtifact(ctx, project.ID, "", artifacts.ArtifactType("document"), "Test Doc", "", "Content")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.DELETE("/api/v1/artifacts/:id", h.DeleteArtifact)

	req, _ := http.NewRequest("DELETE", "/api/v1/artifacts/"+artifact.ID, nil)
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

func TestHandleExportArtifact_JSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")
	artifact, _ := am.CreateArtifact(ctx, project.ID, "", artifacts.ArtifactType("document"), "Test Doc", "", "Content")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.GET("/api/v1/artifacts/:id/export", h.ExportArtifact)

	req, _ := http.NewRequest("GET", "/api/v1/artifacts/"+artifact.ID+"/export?format=json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestHandleExportArtifact_Markdown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")
	artifact, _ := am.CreateArtifact(ctx, project.ID, "", artifacts.ArtifactType("markdown"), "Test Doc", "", "# Markdown Content")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.GET("/api/v1/artifacts/:id/export", h.ExportArtifact)

	req, _ := http.NewRequest("GET", "/api/v1/artifacts/"+artifact.ID+"/export?format=md", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/markdown" {
		t.Errorf("Expected Content-Type text/markdown, got %s", contentType)
	}
}

func TestHandleCreateArtifact_EmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.POST("/api/v1/artifacts", h.CreateArtifact)

	reqBody := CreateArtifactRequest{
		ProjectID: project.ID,
		Type:      "document",
		Name:      "   ",
		Content:   "Test content",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/artifacts", bytes.NewBuffer(body))
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

func TestHandleCreateArtifact_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	if err != nil {
		t.Fatalf("Failed to create project manager: %v", err)
	}

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	ctx := context.Background()
	project, _ := pm.CreateProject(ctx, "Test Project", "Description", "")

	h := NewArtifactHandler(am)

	router := gin.New()
	router.POST("/api/v1/artifacts", h.CreateArtifact)

	reqBody := CreateArtifactRequest{
		ProjectID: project.ID,
		Type:      "invalid_type",
		Name:      "Test",
		Content:   "Test content",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/artifacts", bytes.NewBuffer(body))
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

func TestHandleGetArtifact_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupArtifactTestDB(t)
	defer db.Close()

	am, err := artifacts.NewArtifactManager(db)
	if err != nil {
		t.Fatalf("Failed to create artifact manager: %v", err)
	}

	h := NewArtifactHandler(am)

	router := gin.New()
	router.GET("/api/v1/artifacts/:id", h.GetArtifact)

	req, _ := http.NewRequest("GET", "/api/v1/artifacts/nonexistent-id", nil)
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
