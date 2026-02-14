package server

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var requestsProcessed int64

// incrementRequests is called to track total requests (used by metrics).
func incrementRequests() {
	atomic.AddInt64(&requestsProcessed, 1)
}

// HealthResponse is the response for GET /health.
type HealthResponse struct {
	Status            string                     `json:"status"`
	Version           string                     `json:"version"`
	Timestamp         string                     `json:"timestamp"`
	Providers         map[string]string           `json:"providers"`
	Dependencies      map[string]string           `json:"dependencies"`
	UptimeSeconds     int64                       `json:"uptime_seconds"`
	RequestsProcessed int64                       `json:"requests_processed"`
}

// handleHealth handles GET /health.
func (s *Server) handleHealth(c *gin.Context) {
	incrementRequests()

	providers := make(map[string]string)
	overallStatus := "healthy"

	if s.pluginManager != nil {
		for name, prov := range s.pluginManager.GetProviders() {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			_, err := prov.GetInfo(ctx)
			cancel()
			if err != nil {
				providers[name] = "degraded"
				if overallStatus == "healthy" {
					overallStatus = "degraded"
				}
			} else {
				providers[name] = "healthy"
			}
		}
	}

	deps := make(map[string]string)

	// Memory store
	if s.memoryManager != nil {
		deps["memory_store"] = "healthy"
	} else {
		deps["memory_store"] = "not_configured"
	}

	// Tool registry
	deps["tool_registry"] = "healthy"

	// Orchestration engine
	if s.orchestrationEngine != nil {
		deps["orchestration_engine"] = "healthy"
	} else {
		deps["orchestration_engine"] = "not_configured"
	}

	uptime := int64(0)
	if !s.startTime.IsZero() {
		uptime = int64(time.Since(s.startTime).Seconds())
	}

	status := http.StatusOK

	c.JSON(status, HealthResponse{
		Status:            overallStatus,
		Version:           Version,
		Timestamp:         time.Now().Format(time.RFC3339),
		Providers:         providers,
		Dependencies:      deps,
		UptimeSeconds:     uptime,
		RequestsProcessed: atomic.LoadInt64(&requestsProcessed),
	})
}

// handleMetrics handles GET /metrics (Prometheus-style).
func (s *Server) handleMetrics(c *gin.Context) {
	incrementRequests()

	total := atomic.LoadInt64(&requestsProcessed)
	uptime := int64(0)
	if !s.startTime.IsZero() {
		uptime = int64(time.Since(s.startTime).Seconds())
	}

	providerCount := 0
	if s.pluginManager != nil {
		providerCount = len(s.pluginManager.GetProviders())
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, `# HELP server_requests_total Total number of requests processed
# TYPE server_requests_total counter
server_requests_total %d

# HELP server_uptime_seconds Server uptime in seconds
# TYPE server_uptime_seconds gauge
server_uptime_seconds %d

# HELP server_providers_active Number of active providers
# TYPE server_providers_active gauge
server_providers_active %d

# HELP server_info Server version information
# TYPE server_info gauge
server_info{version="%s"} 1
`, total, uptime, providerCount, Version)
}
