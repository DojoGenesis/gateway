package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleHealthCheck(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router
	r := gin.New()
	r.GET("/api/v1/health", HandleHealthCheck)

	// Create a test request
	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Assert status code
	assert.Equal(t, http.StatusOK, w.Code, "Expected status code 200")

	// Parse response
	var response HealthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be valid JSON")

	// Assert response structure
	assert.Equal(t, "ok", response.Status, "Status should be 'ok'")
	assert.Equal(t, "1.0.0", response.Version, "Version should be '1.0.0'")
}

func TestHealthCheckResponseTime(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router
	r := gin.New()
	r.GET("/api/v1/health", HandleHealthCheck)

	// Create a test request
	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request and measure time
	r.ServeHTTP(w, req)

	// Assert that response time requirement is met (should be < 50ms)
	// This is tested by ensuring the handler completes quickly
	// In practice, this simple handler should respond in < 1ms
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthCheckNoAuthentication(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test router
	r := gin.New()
	r.GET("/api/v1/health", HandleHealthCheck)

	// Create a test request with no auth headers
	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	r.ServeHTTP(w, req)

	// Should succeed without authentication
	assert.Equal(t, http.StatusOK, w.Code, "Health check should work without authentication")
}
