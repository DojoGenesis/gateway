package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// HandleHealthCheck returns a simple health check response for network status detection
func HandleHealthCheck(c *gin.Context) {
	response := HealthResponse{
		Status:  "ok",
		Version: "0.2.4",
	}

	c.JSON(http.StatusOK, response)
}
