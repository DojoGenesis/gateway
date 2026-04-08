package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// handleAdminProviderHistory returns latency history for a specific provider.
// GET /admin/providers/:name/history?points=60
func (s *Server) handleAdminProviderHistory(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", "Provider name is required")
		return
	}

	points := 60
	if raw := c.Query("points"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			s.errorResponse(c, http.StatusBadRequest, "invalid_request", "points must be a positive integer")
			return
		}
		points = n
	}

	if s.latencyTracker == nil {
		c.JSON(http.StatusOK, gin.H{
			"name":        name,
			"data_points": []interface{}{},
		})
		return
	}

	history := s.latencyTracker.GetHistory(name, points)
	c.JSON(http.StatusOK, history)
}
