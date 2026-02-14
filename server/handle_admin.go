package server

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// handleAdminHealth returns detailed health information about the gateway.
// GET /admin/health
func (s *Server) handleAdminHealth(c *gin.Context) {
	health := gin.H{
		"status":  "healthy",
		"version": Version,
		"uptime":  time.Since(s.startTime).String(),
		"system": gin.H{
			"go_version":     runtime.Version(),
			"num_goroutines": runtime.NumGoroutine(),
			"num_cpu":        runtime.NumCPU(),
		},
	}

	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	health["memory"] = gin.H{
		"alloc_mb":       memStats.Alloc / 1024 / 1024,
		"total_alloc_mb": memStats.TotalAlloc / 1024 / 1024,
		"sys_mb":         memStats.Sys / 1024 / 1024,
		"num_gc":         memStats.NumGC,
	}

	// Tool registry stats
	if s.toolRegistry != nil {
		tools, err := s.toolRegistry.List(c.Request.Context())
		if err == nil {
			// Compute namespaces from tool names
			namespaceMap := make(map[string]bool)
			for _, tool := range tools {
				// Extract namespace from tool name (e.g., "mcp.server.tool" -> "mcp")
				parts := strings.SplitN(tool.Name, ".", 2)
				if len(parts) > 1 {
					namespaceMap[parts[0]] = true
				}
			}
			namespaces := make([]string, 0, len(namespaceMap))
			for ns := range namespaceMap {
				namespaces = append(namespaces, ns)
			}
			health["tools"] = gin.H{
				"total":      len(tools),
				"namespaces": namespaces,
			}
		}
	}

	// MCP connections
	if s.mcpHostManager != nil {
		mcpStatus := s.mcpHostManager.Status()
		mcpServers := make([]gin.H, 0)
		totalMCPTools := 0
		for name, status := range mcpStatus {
			mcpServers = append(mcpServers, gin.H{
				"name":         name,
				"connected":    status.Connected,
				"tool_count":   status.ToolCount,
				"last_checked": status.LastChecked,
			})
			totalMCPTools += status.ToolCount
		}
		health["mcp"] = gin.H{
			"enabled":      true,
			"server_count": len(mcpStatus),
			"total_tools":  totalMCPTools,
			"servers":      mcpServers,
		}
	} else {
		health["mcp"] = gin.H{
			"enabled": false,
		}
	}

	// Agent stats
	s.agentMu.RLock()
	agentCount := len(s.agents)
	s.agentMu.RUnlock()
	health["agents"] = gin.H{
		"active_count": agentCount,
	}

	c.JSON(http.StatusOK, health)
}

// handleAdminConfig returns current configuration (sanitized - no secrets).
// GET /admin/config
func (s *Server) handleAdminConfig(c *gin.Context) {
	config := gin.H{
		"server": gin.H{
			"port":             s.cfg.Port,
			"environment":      s.cfg.Environment,
			"auth_mode":        s.cfg.AuthMode,
			"shutdown_timeout": s.cfg.ShutdownTimeout.String(),
		},
		"features": gin.H{
			"otel_enabled":          s.traceLogger != nil,
			"mcp_enabled":           s.mcpHostManager != nil,
			"memory_enabled":        s.memoryManager != nil,
			"orchestration_enabled": s.orchestrationEngine != nil,
		},
	}

	c.JSON(http.StatusOK, config)
}

// handleAdminConfigReload reloads YAML configuration from disk.
// POST /admin/config/reload
func (s *Server) handleAdminConfigReload(c *gin.Context) {
	s.errorResponse(c, http.StatusNotImplemented, "not_implemented", "Configuration reload not yet implemented; requires service restart")
}

// handleAdminMetrics returns Prometheus-format metrics.
// GET /admin/metrics/prometheus
func (s *Server) handleAdminMetrics(c *gin.Context) {
	// Generate Prometheus text format metrics
	metrics := ""

	// Tool execution metrics
	if s.toolRegistry != nil {
		tools, err := s.toolRegistry.List(c.Request.Context())
		toolCount := 0
		if err == nil {
			toolCount = len(tools)
		}
		metrics += fmt.Sprintf("# HELP gateway_tools_total Total number of registered tools\n")
		metrics += fmt.Sprintf("# TYPE gateway_tools_total gauge\n")
		metrics += fmt.Sprintf("gateway_tools_total %d\n", toolCount)
	}

	// Agent metrics
	s.agentMu.RLock()
	agentCount := len(s.agents)
	s.agentMu.RUnlock()
	metrics += fmt.Sprintf("# HELP gateway_agents_active Active agent count\n")
	metrics += fmt.Sprintf("# TYPE gateway_agents_active gauge\n")
	metrics += fmt.Sprintf("gateway_agents_active %d\n", agentCount)

	// Memory metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	metrics += fmt.Sprintf("# HELP gateway_memory_alloc_bytes Allocated memory in bytes\n")
	metrics += fmt.Sprintf("# TYPE gateway_memory_alloc_bytes gauge\n")
	metrics += fmt.Sprintf("gateway_memory_alloc_bytes %d\n", memStats.Alloc)

	metrics += fmt.Sprintf("# HELP gateway_goroutines Number of goroutines\n")
	metrics += fmt.Sprintf("# TYPE gateway_goroutines gauge\n")
	metrics += fmt.Sprintf("gateway_goroutines %d\n", runtime.NumGoroutine())

	// Uptime
	uptime := time.Since(s.startTime).Seconds()
	metrics += fmt.Sprintf("# HELP gateway_uptime_seconds Gateway uptime in seconds\n")
	metrics += fmt.Sprintf("# TYPE gateway_uptime_seconds counter\n")
	metrics += fmt.Sprintf("gateway_uptime_seconds %.0f\n", uptime)

	c.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(metrics))
}

// handleAdminMCPServers lists MCP server connections and their status.
// GET /admin/mcp/servers
func (s *Server) handleAdminMCPServers(c *gin.Context) {
	if s.mcpHostManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"servers": []interface{}{},
		})
		return
	}

	mcpStatus := s.mcpHostManager.Status()
	servers := make([]gin.H, 0)

	for name, status := range mcpStatus {
		servers = append(servers, gin.H{
			"name":         name,
			"connected":    status.Connected,
			"tool_count":   status.ToolCount,
			"last_error":   status.LastError,
			"last_checked": status.LastChecked,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":      true,
		"server_count": len(servers),
		"servers":      servers,
	})
}

// handleAdminMCPStatus returns MCP server connection status and overall health.
// GET /admin/mcp/status
// Response format:
//
//	{
//	  "servers": { "server_id": { "server_id": "...", "state": "connected", "tool_count": 14, ... } },
//	  "total_servers": 1,
//	  "total_tools": 14,
//	  "healthy": true
//	}
func (s *Server) handleAdminMCPStatus(c *gin.Context) {
	if s.mcpHostManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"servers":       map[string]interface{}{},
			"total_servers": 0,
			"total_tools":   0,
			"healthy":       false,
		})
		return
	}

	// Get status from MCP host manager
	mcpStatus := s.mcpHostManager.Status()

	// Build response structure per spec
	servers := make(map[string]gin.H)
	totalTools := 0
	healthy := true

	for serverID, status := range mcpStatus {
		// Convert "Connected" boolean to "state" string per spec
		state := "disconnected"
		if status.Connected {
			state = "connected"
		}

		// Build server status object
		servers[serverID] = gin.H{
			"server_id":         serverID,
			"display_name":      status.Name,
			"state":             state,
			"last_health_check": status.LastChecked,
			"tool_count":        status.ToolCount,
			"last_error":        status.LastError,
		}

		// Aggregate metrics
		totalTools += status.ToolCount

		// Overall health is false if ANY server is not connected
		if !status.Connected {
			healthy = false
		}
	}

	// Return response per spec format
	c.JSON(http.StatusOK, gin.H{
		"servers":       servers,
		"total_servers": len(servers),
		"total_tools":   totalTools,
		"healthy":       healthy,
	})
}
