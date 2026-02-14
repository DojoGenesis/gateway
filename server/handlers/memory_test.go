package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupMemoryTestRouter(t *testing.T) (*gin.Engine, *memory.MemoryManager, func()) {
	gin.SetMode(gin.TestMode)

	dbPath := ".test_memory.db"
	mm, err := memory.NewMemoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}

	InitializeMemoryHandlers(mm)

	router := gin.New()

	router.POST("/api/v1/memory", HandleStoreMemory)
	router.GET("/api/v1/memory/:id", HandleRetrieveMemory)
	router.PUT("/api/v1/memory/:id", HandleUpdateMemory)
	router.POST("/api/v1/memory/search", HandleSearchMemory)
	router.GET("/api/v1/memory/list", HandleListMemories)
	router.DELETE("/api/v1/memory/:id", HandleDeleteMemory)

	cleanup := func() {
		mm.Close()
		os.Remove(dbPath)
	}

	return router, mm, cleanup
}

func TestHandleStoreMemory(t *testing.T) {
	router, _, cleanup := setupMemoryTestRouter(t)
	defer cleanup()

	tests := []struct {
		name            string
		requestBody     interface{}
		expectedStatus  int
		expectedSuccess bool
		checkResponse   func(*testing.T, map[string]interface{})
	}{
		{
			name: "Valid memory storage",
			requestBody: StoreMemoryRequest{
				Type:    "conversation",
				Content: "Test conversation content",
				Metadata: map[string]interface{}{
					"user_id":    "test-user",
					"session_id": "test-session",
				},
			},
			expectedStatus:  http.StatusCreated,
			expectedSuccess: true,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))

				mem := resp["memory"].(map[string]interface{})
				assert.NotEmpty(t, mem["id"])
				assert.Equal(t, "conversation", mem["type"])
				assert.Equal(t, "Test conversation content", mem["content"])

				metadata := mem["metadata"].(map[string]interface{})
				assert.Equal(t, "test-user", metadata["user_id"])
				assert.Equal(t, "test-session", metadata["session_id"])
			},
		},
		{
			name: "Missing type field",
			requestBody: map[string]interface{}{
				"content": "Test content",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
				assert.Contains(t, resp["error"], "Invalid request body")
			},
		},
		{
			name: "Missing content field",
			requestBody: map[string]interface{}{
				"type": "conversation",
			},
			expectedStatus:  http.StatusBadRequest,
			expectedSuccess: false,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
				assert.Contains(t, resp["error"], "Invalid request body")
			},
		},
		{
			name: "Nil metadata defaults to empty map",
			requestBody: StoreMemoryRequest{
				Type:    "plan",
				Content: "Test plan content",
			},
			expectedStatus:  http.StatusCreated,
			expectedSuccess: true,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				mem := resp["memory"].(map[string]interface{})
				metadata := mem["metadata"].(map[string]interface{})
				assert.NotNil(t, metadata)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/memory", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestHandleRetrieveMemory(t *testing.T) {
	router, mm, cleanup := setupMemoryTestRouter(t)
	defer cleanup()

	storedMem := memory.Memory{
		ID:        "test-id-123",
		Type:      "conversation",
		Content:   "Test content for retrieval",
		Metadata:  map[string]interface{}{"key": "value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := mm.Store(context.Background(), storedMem)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		memoryID       string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Retrieve existing memory",
			memoryID:       "test-id-123",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))

				mem := resp["memory"].(map[string]interface{})
				assert.Equal(t, "test-id-123", mem["id"])
				assert.Equal(t, "conversation", mem["type"])
				assert.Equal(t, "Test content for retrieval", mem["content"])
			},
		},
		{
			name:           "Retrieve non-existent memory",
			memoryID:       "non-existent-id",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
				assert.Contains(t, resp["error"], "Memory not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/"+tt.memoryID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestHandleSearchMemory(t *testing.T) {
	router, mm, cleanup := setupMemoryTestRouter(t)
	defer cleanup()

	memories := []memory.Memory{
		{
			ID:        "mem-1",
			Type:      "conversation",
			Content:   "This is about golang programming",
			Metadata:  map[string]interface{}{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "mem-2",
			Type:      "plan",
			Content:   "Planning to learn Python",
			Metadata:  map[string]interface{}{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "mem-3",
			Type:      "conversation",
			Content:   "Advanced golang techniques",
			Metadata:  map[string]interface{}{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, mem := range memories {
		err := mm.Store(context.Background(), mem)
		assert.NoError(t, err)
	}

	tests := []struct {
		name           string
		requestBody    SearchMemoryRequest
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "Search for golang",
			requestBody: SearchMemoryRequest{
				Query:      "golang",
				MaxResults: 10,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				count := int(resp["count"].(float64))
				assert.Equal(t, 2, count)

				results := resp["results"].([]interface{})
				assert.Len(t, results, 2)
			},
		},
		{
			name: "Search for Python",
			requestBody: SearchMemoryRequest{
				Query:      "Python",
				MaxResults: 10,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				count := int(resp["count"].(float64))
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "Search with max results limit",
			requestBody: SearchMemoryRequest{
				Query:      "golang",
				MaxResults: 1,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				results := resp["results"].([]interface{})
				assert.LessOrEqual(t, len(results), 1)
			},
		},
		{
			name: "Search with no results",
			requestBody: SearchMemoryRequest{
				Query:      "nonexistent",
				MaxResults: 10,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				count := int(resp["count"].(float64))
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "Default max results when not specified",
			requestBody: SearchMemoryRequest{
				Query: "golang",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
			},
		},
		{
			name: "Max results capped at 100",
			requestBody: SearchMemoryRequest{
				Query:      "golang",
				MaxResults: 200,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
			},
		},
		{
			name:           "Missing query field",
			requestBody:    SearchMemoryRequest{},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/search", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestHandleDeleteMemory(t *testing.T) {
	router, mm, cleanup := setupMemoryTestRouter(t)
	defer cleanup()

	storedMem := memory.Memory{
		ID:        "delete-test-id",
		Type:      "conversation",
		Content:   "Content to be deleted",
		Metadata:  map[string]interface{}{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := mm.Store(context.Background(), storedMem)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		memoryID       string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Delete existing memory",
			memoryID:       "delete-test-id",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				assert.Equal(t, "Memory deleted successfully", resp["message"])
			},
		},
		{
			name:           "Delete non-existent memory",
			memoryID:       "non-existent-id",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
				assert.Contains(t, resp["error"], "Failed to delete memory")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/memory/"+tt.memoryID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestMemoryHandlersWithoutInitialization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memoryManager = nil

	router := gin.New()
	router.POST("/api/v1/memory", HandleStoreMemory)
	router.GET("/api/v1/memory/:id", HandleRetrieveMemory)
	router.POST("/api/v1/memory/search", HandleSearchMemory)
	router.DELETE("/api/v1/memory/:id", HandleDeleteMemory)

	tests := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"Store without init", http.MethodPost, "/api/v1/memory", StoreMemoryRequest{Type: "test", Content: "test"}},
		{"Retrieve without init", http.MethodGet, "/api/v1/memory/test-id", nil},
		{"Search without init", http.MethodPost, "/api/v1/memory/search", SearchMemoryRequest{Query: "test"}},
		{"Delete without init", http.MethodDelete, "/api/v1/memory/test-id", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.False(t, response["success"].(bool))
			assert.Contains(t, response["error"], "memory manager not initialized")
		})
	}
}

func TestMemoryIntegrationFlow(t *testing.T) {
	router, _, cleanup := setupMemoryTestRouter(t)
	defer cleanup()

	var memoryID string

	t.Run("Store memory", func(t *testing.T) {
		reqBody := StoreMemoryRequest{
			Type:    "conversation",
			Content: "Integration test content",
			Metadata: map[string]interface{}{
				"test": "integration",
			},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		mem := response["memory"].(map[string]interface{})
		memoryID = mem["id"].(string)
		assert.NotEmpty(t, memoryID)
	})

	t.Run("Retrieve stored memory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/"+memoryID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		mem := response["memory"].(map[string]interface{})
		assert.Equal(t, memoryID, mem["id"])
		assert.Equal(t, "Integration test content", mem["content"])
	})

	t.Run("Search for stored memory", func(t *testing.T) {
		reqBody := SearchMemoryRequest{
			Query:      "Integration",
			MaxResults: 10,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		count := int(response["count"].(float64))
		assert.Equal(t, 1, count)
	})

	t.Run("Delete stored memory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/memory/"+memoryID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("Verify deletion", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/"+memoryID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleUpdateMemory(t *testing.T) {
	router, mm, cleanup := setupMemoryTestRouter(t)
	defer cleanup()

	storedMem := memory.Memory{
		ID:        "update-test-id",
		Type:      "conversation",
		Content:   "Original content",
		Metadata:  map[string]interface{}{"key": "value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := mm.Store(context.Background(), storedMem)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		memoryID       string
		requestBody    UpdateMemoryRequest
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:     "Update content only",
			memoryID: "update-test-id",
			requestBody: UpdateMemoryRequest{
				Content: "Updated content",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				mem := resp["memory"].(map[string]interface{})
				assert.Equal(t, "Updated content", mem["content"])
			},
		},
		{
			name:     "Update metadata only",
			memoryID: "update-test-id",
			requestBody: UpdateMemoryRequest{
				Metadata: map[string]interface{}{"newKey": "newValue"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
			},
		},
		{
			name:     "Update both content and metadata",
			memoryID: "update-test-id",
			requestBody: UpdateMemoryRequest{
				Content:  "Both updated",
				Metadata: map[string]interface{}{"both": "updated"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				mem := resp["memory"].(map[string]interface{})
				assert.Equal(t, "Both updated", mem["content"])
			},
		},
		{
			name:           "Update non-existent memory",
			memoryID:       "non-existent-id",
			requestBody:    UpdateMemoryRequest{Content: "Test"},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
			},
		},
		{
			name:           "Empty update request",
			memoryID:       "update-test-id",
			requestBody:    UpdateMemoryRequest{},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.False(t, resp["success"].(bool))
				assert.Contains(t, resp["error"], "At least one")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/memory/"+tt.memoryID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestHandleListMemories(t *testing.T) {
	router, mm, cleanup := setupMemoryTestRouter(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		mem := memory.Memory{
			ID:        fmt.Sprintf("list-test-id-%d", i),
			Type:      "conversation",
			Content:   fmt.Sprintf("Content %d", i),
			Metadata:  map[string]interface{}{"session_id": "test-session"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := mm.Store(context.Background(), mem)
		assert.NoError(t, err)
	}

	tests := []struct {
		name           string
		sessionID      string
		maxResults     int
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "List all memories",
			sessionID:      "",
			maxResults:     0,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				count := int(resp["count"].(float64))
				assert.GreaterOrEqual(t, count, 10)
			},
		},
		{
			name:           "List with session filter",
			sessionID:      "test-session",
			maxResults:     0,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				count := int(resp["count"].(float64))
				assert.GreaterOrEqual(t, count, 10)
			},
		},
		{
			name:           "List with max results limit",
			sessionID:      "",
			maxResults:     5,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
				count := int(resp["count"].(float64))
				assert.LessOrEqual(t, count, 5)
			},
		},
		{
			name:           "List with max results over limit",
			sessionID:      "",
			maxResults:     200,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.True(t, resp["success"].(bool))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/memory/list"
			if tt.sessionID != "" || tt.maxResults > 0 {
				url += "?"
				if tt.sessionID != "" {
					url += "session_id=" + tt.sessionID
				}
				if tt.maxResults > 0 {
					if tt.sessionID != "" {
						url += "&"
					}
					url += fmt.Sprintf("max_results=%d", tt.maxResults)
				}
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func setupGardenTestRouter(t *testing.T) (*gin.Engine, *memory.GardenManager, func()) {
	gin.SetMode(gin.TestMode)

	dbPath := ".test_garden.db"
	mm, err := memory.NewMemoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}

	mockPM := NewMockPluginManager()

	gm, err := memory.NewGardenManager(mm, mockPM)
	if err != nil {
		t.Fatalf("Failed to create garden manager: %v", err)
	}

	InitializeGardenHandlers(gm)

	router := gin.New()

	router.GET("/api/v1/memory/seeds", HandleListSeeds)
	router.POST("/api/v1/memory/seeds", HandleCreateSeed)
	router.GET("/api/v1/memory/snapshots/:session", HandleListSnapshots)
	router.POST("/api/v1/memory/snapshots", HandleCreateSnapshot)
	router.POST("/api/v1/memory/restore/:snapshot", HandleRestoreSnapshot)

	cleanup := func() {
		mm.Close()
		os.Remove(dbPath)
	}

	return router, gm, cleanup
}

func TestHandleListSeeds(t *testing.T) {
	router, gm, cleanup := setupGardenTestRouter(t)
	defer cleanup()

	seed := &memory.Seed{
		ID:          "seed-1",
		Name:        "Test Seed",
		Description: "A test seed",
		Trigger:     "test trigger",
		Content:     "Test content",
		UsageCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := gm.StoreSeed(context.Background(), seed)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/seeds?limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, float64(1), response["count"].(float64))
}

func TestHandleCreateSeed(t *testing.T) {
	router, _, cleanup := setupGardenTestRouter(t)
	defer cleanup()

	reqBody := CreateSeedRequest{
		Name:        "New Seed",
		Description: "A new test seed",
		Trigger:     "new trigger",
		Content:     "New seed content",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/seeds", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))

	seedData := response["seed"].(map[string]interface{})
	assert.NotEmpty(t, seedData["id"])
	assert.Equal(t, "New Seed", seedData["name"])
	assert.Equal(t, "New seed content", seedData["content"])
}

func TestHandleCreateSeed_MissingFields(t *testing.T) {
	router, _, cleanup := setupGardenTestRouter(t)
	defer cleanup()

	tests := []struct {
		name        string
		requestBody map[string]interface{}
	}{
		{
			name: "Missing name",
			requestBody: map[string]interface{}{
				"content": "Test content",
			},
		},
		{
			name: "Missing content",
			requestBody: map[string]interface{}{
				"name": "Test Seed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/seeds", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandleListSnapshots(t *testing.T) {
	router, gm, cleanup := setupGardenTestRouter(t)
	defer cleanup()

	snapshot := &memory.MemorySnapshot{
		ID:           "snapshot-1",
		SessionID:    "session-1",
		SnapshotName: "Test Snapshot",
		SnapshotData: map[string]interface{}{
			"key": "value",
		},
		CreatedAt: time.Now(),
	}

	err := gm.StoreSnapshot(context.Background(), snapshot)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/snapshots/session-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, float64(1), response["count"].(float64))
}

func TestHandleCreateSnapshot(t *testing.T) {
	router, _, cleanup := setupGardenTestRouter(t)
	defer cleanup()

	reqBody := CreateSnapshotRequest{
		SessionID:    "session-1",
		SnapshotName: "My Snapshot",
		SnapshotData: map[string]interface{}{
			"memories": []string{"mem-1", "mem-2"},
			"context":  "important context",
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/snapshots", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))

	snapshotData := response["snapshot"].(map[string]interface{})
	assert.NotEmpty(t, snapshotData["id"])
	assert.Equal(t, "session-1", snapshotData["session_id"])
	assert.Equal(t, "My Snapshot", snapshotData["snapshot_name"])
}

func TestHandleRestoreSnapshot(t *testing.T) {
	router, gm, cleanup := setupGardenTestRouter(t)
	defer cleanup()

	snapshot := &memory.MemorySnapshot{
		ID:           "snapshot-1",
		SessionID:    "session-1",
		SnapshotName: "Test Snapshot",
		SnapshotData: map[string]interface{}{
			"key": "value",
		},
		CreatedAt: time.Now(),
	}

	err := gm.StoreSnapshot(context.Background(), snapshot)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/restore/snapshot-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))

	returnedSnapshot := response["snapshot"].(map[string]interface{})
	assert.Equal(t, "snapshot-1", returnedSnapshot["id"])
}

func TestHandleRestoreSnapshot_NotFound(t *testing.T) {
	router, _, cleanup := setupGardenTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/restore/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGardenHandlersWithoutInitialization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gardenManager = nil

	router := gin.New()
	router.GET("/api/v1/memory/seeds", HandleListSeeds)
	router.POST("/api/v1/memory/seeds", HandleCreateSeed)
	router.GET("/api/v1/memory/snapshots/:session", HandleListSnapshots)
	router.POST("/api/v1/memory/snapshots", HandleCreateSnapshot)
	router.POST("/api/v1/memory/restore/:snapshot", HandleRestoreSnapshot)

	tests := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		expected int
	}{
		{"ListSeeds", "GET", "/api/v1/memory/seeds", nil, http.StatusInternalServerError},
		{"CreateSeed", "POST", "/api/v1/memory/seeds", CreateSeedRequest{Name: "Test", Content: "Test"}, http.StatusInternalServerError},
		{"ListSnapshots", "GET", "/api/v1/memory/snapshots/session-1", nil, http.StatusInternalServerError},
		{"CreateSnapshot", "POST", "/api/v1/memory/snapshots", CreateSnapshotRequest{SessionID: "s1", SnapshotData: map[string]interface{}{}}, http.StatusInternalServerError},
		{"RestoreSnapshot", "POST", "/api/v1/memory/restore/snap-1", nil, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expected, w.Code)
		})
	}
}

func TestValidateContextType(t *testing.T) {
	tests := []struct {
		name        string
		contextType string
		expectError bool
	}{
		{
			name:        "valid private",
			contextType: "private",
			expectError: false,
		},
		{
			name:        "valid group",
			contextType: "group",
			expectError: false,
		},
		{
			name:        "valid public",
			contextType: "public",
			expectError: false,
		},
		{
			name:        "empty string defaults to private",
			contextType: "",
			expectError: false,
		},
		{
			name:        "invalid context type",
			contextType: "invalid",
			expectError: true,
		},
		{
			name:        "invalid context type - shared",
			contextType: "shared",
			expectError: true,
		},
		{
			name:        "invalid context type - personal",
			contextType: "personal",
			expectError: true,
		},
		{
			name:        "case sensitive - Private",
			contextType: "Private",
			expectError: true,
		},
		{
			name:        "case sensitive - PUBLIC",
			contextType: "PUBLIC",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContextType(tt.contextType)
			if tt.expectError && err == nil {
				t.Errorf("expected error for context_type '%s', but got none", tt.contextType)
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error for context_type '%s', but got: %v", tt.contextType, err)
			}
		})
	}
}

func TestHandleStoreMemory_WithContextType(t *testing.T) {
	router, _, cleanup := setupMemoryTestRouter(t)
	defer cleanup()

	tests := []struct {
		name                string
		contextType         string
		expectedStatus      int
		expectedContextType string
	}{
		{
			name:                "store with private context type",
			contextType:         "private",
			expectedStatus:      http.StatusCreated,
			expectedContextType: "private",
		},
		{
			name:                "store with group context type",
			contextType:         "group",
			expectedStatus:      http.StatusCreated,
			expectedContextType: "group",
		},
		{
			name:                "store with public context type",
			contextType:         "public",
			expectedStatus:      http.StatusCreated,
			expectedContextType: "public",
		},
		{
			name:                "store with empty context type defaults to private",
			contextType:         "",
			expectedStatus:      http.StatusCreated,
			expectedContextType: "private",
		},
		{
			name:           "store with invalid context type returns 400",
			contextType:    "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"type":    "user",
				"content": "Test memory content",
			}
			if tt.contextType != "" {
				reqBody["context_type"] = tt.contextType
			}

			jsonBody, _ := json.Marshal(reqBody)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/memory", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response["success"].(bool))

				memory := response["memory"].(map[string]interface{})
				assert.Equal(t, tt.expectedContextType, memory["context_type"])
			}
		})
	}
}
