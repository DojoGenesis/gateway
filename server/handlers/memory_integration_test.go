package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/DojoGenesis/gateway/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupMemoryIntegrationTestRouter(t *testing.T) (*gin.Engine, *memory.MemoryManager, func()) {
	gin.SetMode(gin.TestMode)

	dbPath := ".test_memory_integration.db"
	mm, err := memory.NewMemoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}

	h := NewMemoryHandler(mm, nil, nil)

	router := gin.New()

	router.POST("/api/v1/memory", middleware.AuthMiddleware(), h.StoreMemory)
	router.GET("/api/v1/memory/:id", middleware.AuthMiddleware(), h.RetrieveMemory)
	router.POST("/api/v1/memory/search", middleware.AuthMiddleware(), h.SearchMemory)
	router.DELETE("/api/v1/memory/:id", middleware.AuthMiddleware(), h.DeleteMemory)

	cleanup := func() {
		mm.Close()
		os.Remove(dbPath)
	}

	return router, mm, cleanup
}

func TestMemoryAPIWithAuthentication(t *testing.T) {
	router, _, cleanup := setupMemoryIntegrationTestRouter(t)
	defer cleanup()

	var memoryID string

	t.Run("Store memory without auth - should fail", func(t *testing.T) {
		reqBody := StoreMemoryRequest{
			Type:    "conversation",
			Content: "Test without auth",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Store memory with valid auth - should succeed", func(t *testing.T) {
		reqBody := StoreMemoryRequest{
			Type:    "conversation",
			Content: "Test with auth",
			Metadata: map[string]interface{}{
				"authenticated": true,
			},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mem := response["memory"].(map[string]interface{})
		memoryID = mem["id"].(string)
		assert.NotEmpty(t, memoryID)
	})

	t.Run("Retrieve memory without auth - should fail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/"+memoryID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Retrieve memory with valid auth - should succeed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/"+memoryID, nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mem := response["memory"].(map[string]interface{})
		assert.Equal(t, "Test with auth", mem["content"])
	})

	t.Run("Search memory without auth - should fail", func(t *testing.T) {
		reqBody := SearchMemoryRequest{
			Query:      "Test",
			MaxResults: 10,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Search memory with valid auth - should succeed", func(t *testing.T) {
		reqBody := SearchMemoryRequest{
			Query:      "auth",
			MaxResults: 10,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		count := int(response["count"].(float64))
		assert.Equal(t, 1, count)
	})

	t.Run("Delete memory without auth - should fail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/memory/"+memoryID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Delete memory with valid auth - should succeed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/memory/"+memoryID, nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	t.Run("Verify memory deleted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/"+memoryID, nil)
		req.Header.Set("Authorization", "Bearer test-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestMemoryAPIWithDifferentTokens(t *testing.T) {
	router, _, cleanup := setupMemoryIntegrationTestRouter(t)
	defer cleanup()

	validTokens := []string{
		"Bearer test-token",
		"Bearer user-12345",
		"Bearer user-abc-def",
	}

	invalidTokens := []string{
		"",
		"Bearer",
		"Bearer ",
		"Basic test-token",
		"Bearer invalid-token-123",
	}

	reqBody := StoreMemoryRequest{
		Type:    "test",
		Content: "Test content",
	}
	body, _ := json.Marshal(reqBody)

	for _, token := range validTokens {
		t.Run("Valid token: "+token, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/memory", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", token)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)
		})
	}

	for _, token := range invalidTokens {
		t.Run("Invalid token: "+token, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/memory", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if token != "" {
				req.Header.Set("Authorization", token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
