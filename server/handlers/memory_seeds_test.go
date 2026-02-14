package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupMemorySeedTest(t *testing.T) (*sql.DB, *memory.SeedManager, func()) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	schema := `
		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);

		CREATE TABLE memory_seeds (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			content TEXT NOT NULL,
			seed_type TEXT NOT NULL,
			source TEXT DEFAULT 'system' CHECK(source IN ('system', 'user', 'calibrated')),
			user_editable BOOLEAN DEFAULT FALSE,
			confidence REAL DEFAULT 1.0,
			usage_count INTEGER DEFAULT 0,
			last_used_at DATETIME,
			deleted_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			created_by TEXT,
			version INTEGER DEFAULT 1
		);
		CREATE INDEX idx_memory_seeds_project ON memory_seeds(project_id);
		CREATE INDEX idx_memory_seeds_type ON memory_seeds(seed_type);
		CREATE INDEX idx_memory_seeds_source ON memory_seeds(source);

		INSERT INTO projects (id, name, created_at, updated_at) 
		VALUES ('test-project', 'Test Project', datetime('now'), datetime('now'));
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	sm, err := memory.NewSeedManager(db)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return db, sm, cleanup
}

func TestHandleGetMemorySeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectID := "test-project"

	tests := []struct {
		name           string
		setupData      bool
		setupManager   bool
		queryParams    string
		expectedStatus int
		expectedError  bool
		minSeedCount   int
	}{
		{
			name:           "successful retrieval with seeds",
			setupData:      true,
			setupManager:   true,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			minSeedCount:   2,
		},
		{
			name:           "successful retrieval without seeds",
			setupData:      false,
			setupManager:   true,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			minSeedCount:   0,
		},
		{
			name:           "filter by user_editable",
			setupData:      true,
			setupManager:   true,
			queryParams:    "user_editable=true",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			minSeedCount:   1,
		},
		{
			name:           "filter by seed_type",
			setupData:      true,
			setupManager:   true,
			queryParams:    "seed_type=test_type",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			minSeedCount:   1,
		},
		{
			name:           "manager not initialized",
			setupData:      false,
			setupManager:   false,
			queryParams:    "",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
			minSeedCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, cleanup := setupMemorySeedTest(t)
			defer cleanup()

			var h *SeedHandler
			if tt.setupManager {
				h = NewSeedHandler(sm)
			} else {
				h = NewSeedHandler(nil)
			}

			if tt.setupData {
				_, err := sm.CreateUserSeed(context.Background(), &projectID, "Test content 1", "test_type", "test_user")
				require.NoError(t, err)
				_, err = sm.CreateUserSeed(context.Background(), &projectID, "Test content 2", "other_type", "test_user")
				require.NoError(t, err)
			}

			router := gin.New()
			router.GET("/api/v1/projects/:project_id/memory/seeds", h.GetMemorySeeds)

			url := "/api/v1/projects/" + projectID + "/memory/seeds"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var errResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResponse)
				require.NoError(t, err)
				assert.NotEmpty(t, errResponse.Error)
			} else {
				var response GetMemorySeedsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(response.Seeds), tt.minSeedCount)
			}
		})
	}
}

func TestHandleUpdateMemorySeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupManager   bool
		seedID         string
		content        string
		userEditable   bool
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "successful update",
			setupManager:   true,
			seedID:         "seed-1",
			content:        "Updated content",
			userEditable:   true,
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "update non-editable seed",
			setupManager:   true,
			seedID:         "seed-2",
			content:        "Updated content",
			userEditable:   false,
			expectedStatus: http.StatusForbidden,
			expectedError:  true,
		},
		{
			name:           "update non-existent seed",
			setupManager:   true,
			seedID:         "non-existent",
			content:        "Updated content",
			userEditable:   true,
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "empty content",
			setupManager:   true,
			seedID:         "seed-1",
			content:        "",
			userEditable:   true,
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "manager not initialized",
			setupManager:   false,
			seedID:         "seed-1",
			content:        "Updated content",
			userEditable:   true,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, sm, cleanup := setupMemorySeedTest(t)
			defer cleanup()

			var h *SeedHandler
			if tt.setupManager {
				h = NewSeedHandler(sm)
			} else {
				h = NewSeedHandler(nil)
			}

			if tt.seedID == "seed-1" || tt.seedID == "seed-2" {
				query := `INSERT INTO memory_seeds 
					(id, content, seed_type, source, user_editable, confidence, usage_count, created_at, updated_at, created_by)
					VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'), ?)`
				_, err := db.Exec(query, tt.seedID, "Original content", "test_type", "user", tt.userEditable, 1.0, 0, "test_user")
				require.NoError(t, err)
			}

			router := gin.New()
			router.PUT("/api/v1/memory/seeds/:id", h.UpdateMemorySeed)

			reqBody := UpdateMemorySeedRequest{Content: tt.content}
			bodyBytes, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("PUT", "/api/v1/memory/seeds/"+tt.seedID, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var errResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResponse)
				require.NoError(t, err)
				assert.NotEmpty(t, errResponse.Error)
			} else {
				var response SeedOperationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Seed)
				assert.Equal(t, tt.content, response.Seed.Content)
			}
		})
	}
}

func TestHandleCreateMemorySeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectID := "test-project"

	tests := []struct {
		name           string
		setupManager   bool
		content        string
		seedType       string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "successful creation",
			setupManager:   true,
			content:        "New seed content",
			seedType:       "test_type",
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name:           "missing content",
			setupManager:   true,
			content:        "",
			seedType:       "test_type",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "missing seed_type",
			setupManager:   true,
			content:        "New seed content",
			seedType:       "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:           "manager not initialized",
			setupManager:   false,
			content:        "New seed content",
			seedType:       "test_type",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, cleanup := setupMemorySeedTest(t)
			defer cleanup()

			var h *SeedHandler
			if tt.setupManager {
				h = NewSeedHandler(sm)
			} else {
				h = NewSeedHandler(nil)
			}

			router := gin.New()
			router.POST("/api/v1/projects/:project_id/memory/seeds", h.CreateMemorySeed)

			reqBody := CreateMemorySeedRequest{
				Content:  tt.content,
				SeedType: tt.seedType,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", "/api/v1/projects/"+projectID+"/memory/seeds", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var errResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResponse)
				require.NoError(t, err)
				assert.NotEmpty(t, errResponse.Error)
			} else {
				var response SeedOperationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Seed)
				assert.Equal(t, tt.content, response.Seed.Content)
				assert.Equal(t, "user", response.Seed.Source)
			}
		})
	}
}

func TestHandleDeleteMemorySeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupManager   bool
		seedID         string
		seedSource     string
		createSeed     bool
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "successful deletion",
			setupManager:   true,
			seedID:         "seed-1",
			seedSource:     "user",
			createSeed:     true,
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "delete system seed",
			setupManager:   true,
			seedID:         "seed-2",
			seedSource:     "system",
			createSeed:     true,
			expectedStatus: http.StatusForbidden,
			expectedError:  true,
		},
		{
			name:           "delete non-existent seed",
			setupManager:   true,
			seedID:         "non-existent",
			seedSource:     "user",
			createSeed:     false,
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "manager not initialized",
			setupManager:   false,
			seedID:         "seed-1",
			seedSource:     "user",
			createSeed:     false,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, sm, cleanup := setupMemorySeedTest(t)
			defer cleanup()

			var h *SeedHandler
			if tt.setupManager {
				h = NewSeedHandler(sm)
			} else {
				h = NewSeedHandler(nil)
			}

			if tt.createSeed {
				query := `INSERT INTO memory_seeds 
					(id, content, seed_type, source, user_editable, confidence, usage_count, created_at, updated_at, created_by)
					VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'), ?)`
				_, err := db.Exec(query, tt.seedID, "Test content", "test_type", tt.seedSource, true, 1.0, 0, "default")
				require.NoError(t, err)
			}

			router := gin.New()
			router.DELETE("/api/v1/memory/seeds/:id", h.DeleteMemorySeed)

			req, _ := http.NewRequest("DELETE", "/api/v1/memory/seeds/"+tt.seedID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var errResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResponse)
				require.NoError(t, err)
				assert.NotEmpty(t, errResponse.Error)
			} else {
				var response SeedOperationResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "success", response.Status)

				var deletedAt sql.NullTime
				err = db.QueryRow("SELECT deleted_at FROM memory_seeds WHERE id = ?", tt.seedID).Scan(&deletedAt)
				require.NoError(t, err)
				assert.True(t, deletedAt.Valid, "seed should be soft-deleted (deleted_at should be set)")
			}
		})
	}
}

func TestHandleUpdateMemorySeed_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, sm, cleanup := setupMemorySeedTest(t)
	defer cleanup()

	h := NewSeedHandler(sm)

	tests := []struct {
		name           string
		seedID         string
		requestBody    string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:           "invalid json",
			seedID:         "seed-1",
			requestBody:    `{"content": invalid}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse:  true,
		},
		{
			name:           "empty seed id",
			seedID:         "",
			requestBody:    `{"content": "Valid content"}`,
			expectedStatus: http.StatusNotFound,
			checkResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.PUT("/api/v1/memory/seeds/:id", h.UpdateMemorySeed)

			url := "/api/v1/memory/seeds/" + tt.seedID
			req, _ := http.NewRequest("PUT", url, bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse {
				var errResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResponse)
				require.NoError(t, err)
				assert.NotEmpty(t, errResponse.Error)
			}
		})
	}
}

func TestHandleCreateMemorySeed_WithProjectID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, sm, cleanup := setupMemorySeedTest(t)
	defer cleanup()

	h := NewSeedHandler(sm)

	// Add project-123 to the database
	projectID := "project-123"
	_, err := db.Exec("INSERT INTO projects (id, name, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))", projectID, "Test Project 123")
	require.NoError(t, err)

	router := gin.New()
	router.POST("/api/v1/projects/:project_id/memory/seeds", h.CreateMemorySeed)
	reqBody := CreateMemorySeedRequest{
		Content:  "Project-specific seed",
		SeedType: "project_type",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/projects/"+projectID+"/memory/seeds", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response SeedOperationResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response.Status)
	assert.NotNil(t, response.Seed)
	assert.Equal(t, projectID, response.Seed.ProjectID)
}

func TestHandleGetMemorySeeds_WithProjectIDFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, sm, cleanup := setupMemorySeedTest(t)
	defer cleanup()

	h := NewSeedHandler(sm)

	projectID1 := "project-1"
	projectID2 := "project-2"

	// Add projects to database
	_, err := db.Exec("INSERT INTO projects (id, name, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))", projectID1, "Project 1")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO projects (id, name, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))", projectID2, "Project 2")
	require.NoError(t, err)

	_, err = sm.CreateUserSeed(context.Background(), &projectID1, "Project 1 seed", "test_type", "test_user")
	require.NoError(t, err)
	_, err = sm.CreateUserSeed(context.Background(), &projectID2, "Project 2 seed", "test_type", "test_user")
	require.NoError(t, err)
	_, err = sm.CreateUserSeed(context.Background(), nil, "Global seed", "test_type", "test_user")
	require.NoError(t, err)

	router := gin.New()
	router.GET("/api/v1/projects/:project_id/memory/seeds", h.GetMemorySeeds)

	req, _ := http.NewRequest("GET", "/api/v1/projects/"+projectID1+"/memory/seeds", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response GetMemorySeedsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 1, len(response.Seeds))
	assert.Equal(t, projectID1, response.Seeds[0].ProjectID)
}

func TestHandleDeleteMemorySeed_PermissionCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, sm, cleanup := setupMemorySeedTest(t)
	defer cleanup()

	h := NewSeedHandler(sm)

	query := `INSERT INTO memory_seeds
		(id, content, seed_type, source, user_editable, confidence, usage_count, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'), ?)`
	_, err := db.Exec(query, "seed-1", "Test content", "test_type", "user", true, 1.0, 0, "other_user")
	require.NoError(t, err)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test_user")
		c.Next()
	})
	router.DELETE("/api/v1/memory/seeds/:id", h.DeleteMemorySeed)

	req, _ := http.NewRequest("DELETE", "/api/v1/memory/seeds/seed-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResponse ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errResponse)
	require.NoError(t, err)

	assert.Contains(t, errResponse.Error, "Permission denied")
}

func TestHandleBulkDeleteMemorySeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		setupManager      bool
		seedIDs           []string
		createSeeds       bool
		expectedStatus    int
		expectedDeleted   int
		expectedFailed    int
		expectedError     bool
		expectMultiStatus bool
	}{
		{
			name:              "successful bulk deletion",
			setupManager:      true,
			seedIDs:           []string{"seed-1", "seed-2"},
			createSeeds:       true,
			expectedStatus:    http.StatusOK,
			expectedDeleted:   2,
			expectedFailed:    0,
			expectedError:     false,
			expectMultiStatus: false,
		},
		{
			name:              "partial success",
			setupManager:      true,
			seedIDs:           []string{"seed-1", "non-existent"},
			createSeeds:       true,
			expectedStatus:    http.StatusMultiStatus,
			expectedDeleted:   1,
			expectedFailed:    1,
			expectedError:     false,
			expectMultiStatus: true,
		},
		{
			name:              "all failed",
			setupManager:      true,
			seedIDs:           []string{"non-existent-1", "non-existent-2"},
			createSeeds:       false,
			expectedStatus:    http.StatusBadRequest,
			expectedDeleted:   0,
			expectedFailed:    2,
			expectedError:     false,
			expectMultiStatus: false,
		},
		{
			name:              "empty seed list",
			setupManager:      true,
			seedIDs:           []string{},
			createSeeds:       false,
			expectedStatus:    http.StatusBadRequest,
			expectedDeleted:   0,
			expectedFailed:    0,
			expectedError:     true,
			expectMultiStatus: false,
		},
		{
			name:              "manager not initialized",
			setupManager:      false,
			seedIDs:           []string{"seed-1"},
			createSeeds:       false,
			expectedStatus:    http.StatusInternalServerError,
			expectedDeleted:   0,
			expectedFailed:    0,
			expectedError:     true,
			expectMultiStatus: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, sm, cleanup := setupMemorySeedTest(t)
			defer cleanup()

			var h *SeedHandler
			if tt.setupManager {
				h = NewSeedHandler(sm)
			} else {
				h = NewSeedHandler(nil)
			}

			if tt.createSeeds {
				for _, seedID := range tt.seedIDs {
					if seedID == "seed-1" || seedID == "seed-2" {
						query := `INSERT INTO memory_seeds 
							(id, content, seed_type, source, user_editable, confidence, usage_count, created_at, updated_at, created_by)
							VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'), ?)`
						_, err := db.Exec(query, seedID, "Test content", "test_type", "user", true, 1.0, 0, "default")
						require.NoError(t, err)
					}
				}
			}

			router := gin.New()
			router.POST("/api/v1/memory/seeds/bulk-delete", h.BulkDeleteMemorySeeds)

			reqBody := BulkDeleteSeedsRequest{SeedIDs: tt.seedIDs}
			bodyBytes, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", "/api/v1/memory/seeds/bulk-delete", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var errResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResponse)
				require.NoError(t, err)
				assert.NotEmpty(t, errResponse.Error)
			} else {
				var response BulkDeleteSeedsResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedDeleted, len(response.Deleted))
				assert.Equal(t, tt.expectedFailed, len(response.Failed))
			}
		})
	}
}

func TestHandleSearchMemorySeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	projectID := "test-project"

	tests := []struct {
		name           string
		setupManager   bool
		setupData      bool
		query          string
		limit          string
		expectedStatus int
		expectedError  bool
		minResults     int
		maxResults     int
	}{
		{
			name:           "successful search with results",
			setupManager:   true,
			setupData:      true,
			query:          "apple",
			limit:          "",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			minResults:     1,
			maxResults:     100,
		},
		{
			name:           "successful search no results",
			setupManager:   true,
			setupData:      true,
			query:          "nonexistent",
			limit:          "",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			minResults:     0,
			maxResults:     0,
		},
		{
			name:           "search with limit",
			setupManager:   true,
			setupData:      true,
			query:          "content",
			limit:          "5",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			minResults:     0,
			maxResults:     5,
		},
		{
			name:           "query too short",
			setupManager:   true,
			setupData:      false,
			query:          "a",
			limit:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			minResults:     0,
			maxResults:     0,
		},
		{
			name:           "missing query",
			setupManager:   true,
			setupData:      false,
			query:          "",
			limit:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			minResults:     0,
			maxResults:     0,
		},
		{
			name:           "manager not initialized",
			setupManager:   false,
			setupData:      false,
			query:          "test",
			limit:          "",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
			minResults:     0,
			maxResults:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, cleanup := setupMemorySeedTest(t)
			defer cleanup()

			var h *SeedHandler
			if tt.setupManager {
				h = NewSeedHandler(sm)
			} else {
				h = NewSeedHandler(nil)
			}

			if tt.setupData {
				// Create test seeds with different content
				_, err := sm.CreateUserSeed(context.Background(), &projectID, "This is apple content", "test_type", "test_user")
				require.NoError(t, err)
				_, err = sm.CreateUserSeed(context.Background(), &projectID, "This is banana content", "test_type", "test_user")
				require.NoError(t, err)
				_, err = sm.CreateUserSeed(context.Background(), &projectID, "This is orange content", "test_type", "test_user")
				require.NoError(t, err)
			}

			router := gin.New()
			router.GET("/api/v1/projects/:project_id/memory/seeds/search", h.SearchMemorySeeds)

			url := "/api/v1/projects/" + projectID + "/memory/seeds/search?q=" + tt.query
			if tt.limit != "" {
				url += "&limit=" + tt.limit
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var errResponse ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResponse)
				require.NoError(t, err)
				assert.NotEmpty(t, errResponse.Error)
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				seeds, ok := response["seeds"].([]interface{})
				require.True(t, ok)

				count := int(response["count"].(float64))
				assert.GreaterOrEqual(t, count, tt.minResults)
				assert.LessOrEqual(t, count, tt.maxResults)
				assert.Equal(t, count, len(seeds))

				queryVal, ok := response["query"].(string)
				require.True(t, ok)
				assert.Equal(t, tt.query, queryVal)
			}
		})
	}
}
