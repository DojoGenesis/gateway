package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/DojoGenesis/gateway/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupGuestTestDB(t *testing.T) string {
	return ":memory:"
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
