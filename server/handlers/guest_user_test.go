package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/artifacts"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/middleware"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/projects"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupGuestTestDB(t *testing.T) string {
	return ":memory:"
}

func setupProjectArtifactDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
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

	CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	CREATE INDEX IF NOT EXISTS idx_projects_last_accessed ON projects(last_accessed_at DESC);
	CREATE INDEX IF NOT EXISTS idx_projects_updated ON projects(updated_at DESC);

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

	CREATE INDEX IF NOT EXISTS idx_templates_category ON project_templates(category);

	CREATE TABLE IF NOT EXISTS project_files (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		filename TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_size INTEGER,
		mime_type TEXT,
		uploaded_at DATETIME NOT NULL,
		metadata TEXT,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_project_files_project ON project_files(project_id);

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

	CREATE INDEX IF NOT EXISTS idx_artifacts_project ON artifacts(project_id);
	CREATE INDEX IF NOT EXISTS idx_artifacts_type ON artifacts(type);
	CREATE INDEX IF NOT EXISTS idx_artifacts_updated ON artifacts(updated_at DESC);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}

func TestGuestUserCanStoreMemory(t *testing.T) {
	dbPath := setupGuestTestDB(t)

	mm, err := memory.NewMemoryManager(dbPath)
	require.NoError(t, err)

	h := NewMemoryHandler(mm, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/memory", middleware.OptionalAuthMiddleware(), h.StoreMemory)

	reqBody := map[string]interface{}{
		"type":    "note",
		"content": "Test memory from guest user",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/memory", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["memory"])
}

func TestGuestUserCanListMemories(t *testing.T) {
	dbPath := setupGuestTestDB(t)

	mm, err := memory.NewMemoryManager(dbPath)
	require.NoError(t, err)

	h := NewMemoryHandler(mm, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/memory/list", middleware.OptionalAuthMiddleware(), h.ListMemories)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/memory/list", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestGuestUserCanCreateProject(t *testing.T) {
	db := setupProjectArtifactDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	require.NoError(t, err)

	h := NewProjectHandler(pm)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/projects", middleware.OptionalAuthMiddleware(), h.CreateProject)

	reqBody := map[string]interface{}{
		"name":        "test-guest-project",
		"description": "A test project created by a guest user",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/projects", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["project"])
}

func TestGuestUserCanListProjects(t *testing.T) {
	db := setupProjectArtifactDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	require.NoError(t, err)

	h := NewProjectHandler(pm)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/projects", middleware.OptionalAuthMiddleware(), h.ListProjects)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/projects", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestGuestUserCanCreateArtifact(t *testing.T) {
	db := setupProjectArtifactDB(t)
	defer db.Close()

	pm, err := projects.NewProjectManager(db)
	require.NoError(t, err)

	am, err := artifacts.NewArtifactManager(db)
	require.NoError(t, err)

	ph := NewProjectHandler(pm)
	ah := NewArtifactHandler(am)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/projects", middleware.OptionalAuthMiddleware(), ph.CreateProject)
	router.POST("/api/v1/artifacts", middleware.OptionalAuthMiddleware(), ah.CreateArtifact)

	projectReqBody := map[string]interface{}{
		"name":        "test-project-for-artifact",
		"description": "A test project",
	}
	projectBody, _ := json.Marshal(projectReqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/projects", bytes.NewBuffer(projectBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var projectResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &projectResponse)
	require.NoError(t, err)

	project := projectResponse["project"].(map[string]interface{})
	projectID := project["id"].(string)

	artifactReqBody := map[string]interface{}{
		"project_id":  projectID,
		"type":        "code",
		"name":        "main.go",
		"description": "Main application file",
		"content":     "package main\n\nfunc main() {}\n",
	}
	artifactBody, _ := json.Marshal(artifactReqBody)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/artifacts", bytes.NewBuffer(artifactBody))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Logf("Response body: %s", w.Body.String())
	}

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["artifact"])
}

func TestGuestUserCanListArtifacts(t *testing.T) {
	db := setupProjectArtifactDB(t)
	defer db.Close()

	_, err := projects.NewProjectManager(db)
	require.NoError(t, err)

	am, err := artifacts.NewArtifactManager(db)
	require.NoError(t, err)

	ah := NewArtifactHandler(am)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/artifacts", middleware.OptionalAuthMiddleware(), ah.ListArtifacts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/artifacts", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
	}

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestAuthenticatedUserCanAccessMemory(t *testing.T) {
	dbPath := setupGuestTestDB(t)

	mm, err := memory.NewMemoryManager(dbPath)
	require.NoError(t, err)

	h := NewMemoryHandler(mm, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/memory", middleware.OptionalAuthMiddleware(), h.StoreMemory)

	reqBody := map[string]interface{}{
		"type":    "note",
		"content": "Test memory from authenticated user",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/memory", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["memory"])
}
