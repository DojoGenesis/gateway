package server

import (
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/handlers"
)

// setupRoutes registers all HTTP routes on the Gin engine.
func (s *Server) setupRoutes() {
	// Initialize existing handler modules with injected dependencies
	if s.pluginManager != nil {
		handlers.InitializeModelHandlers(s.pluginManager)
	}
	if s.intentClassifier != nil && s.primaryAgent != nil {
		handlers.InitializeChatHandlers(s.intentClassifier, s.primaryAgent, s.userRouter, s.pluginManager)
	}
	if s.memoryManager != nil {
		handlers.InitializeMemoryHandlers(s.memoryManager)
	}
	if s.gardenManager != nil {
		handlers.InitializeGardenHandlers(s.gardenManager)
	}
	if s.memoryMaintenance != nil {
		handlers.InitializeMaintenanceHandlers(s.memoryMaintenance)
	}

	// ─── Infrastructure ──────────────────────────────────────────────
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/metrics", s.handleMetrics)

	// ─── SSE (existing broadcaster) ──────────────────────────────────
	s.router.GET("/events", handlers.HandleSSE)

	// ─── OpenAI-Compatible API (/v1) ─────────────────────────────────
	v1 := s.router.Group("/v1")
	{
		// Chat completions (OpenAI-compatible)
		v1.POST("/chat/completions", s.handleChatCompletions)

		// Models (OpenAI-compatible)
		v1.GET("/models", s.handleListModels)

		// ─── Tool Management ─────────────────────────────────────────
		v1.GET("/tools", s.handleListTools)
		v1.GET("/tools/:name", s.handleGetTool)
		v1.POST("/tools/:name/invoke", s.handleInvokeTool)

		// ─── Memory Management ───────────────────────────────────────
		v1.POST("/memory", s.handleStoreMemory)
		v1.GET("/memory", s.handleListMemories)
		v1.GET("/memory/:id", s.handleGetMemory)
		v1.PUT("/memory/:id", s.handleUpdateMemory)
		v1.DELETE("/memory/:id", s.handleDeleteMemory)
		v1.POST("/memory/search", s.handleSearchMemory)

		// ─── Orchestration ───────────────────────────────────────────
		v1.POST("/orchestrate", s.handleOrchestrate)
		v1.GET("/orchestrate/:id/events", s.handleOrchestrationEvents)

		// ─── Gateway Routes (v0.2.0 - New) ───────────────────────────
		gateway := v1.Group("/gateway")
		{
			// Tool discovery with MCP namespace support
			gateway.GET("/tools", s.handleGatewayListTools)

			// Agent management with disposition
			gateway.POST("/agents", s.handleGatewayCreateAgent)
			gateway.GET("/agents/:id", s.handleGatewayGetAgent)
			gateway.POST("/agents/:id/chat", s.handleGatewayAgentChat)

			// Orchestration DAG execution
			gateway.POST("/orchestrate", s.handleGatewayOrchestrate)
			gateway.GET("/orchestrate/:id/dag", s.handleGatewayOrchestrationDAG)

			// Trace inspection (if OTEL enabled)
			gateway.GET("/traces/:id", s.handleGatewayGetTrace)
		}

		// ─── Legacy endpoints (preserving existing routes) ───────────
		// These delegate to existing handler functions for backward compatibility
		v1.POST("/chat", handlers.HandleChat)
		v1.GET("/providers", handlers.HandleListProviders)
		v1.POST("/tools/search", handlers.HandleSearchTools)
		v1.POST("/tools/invoke", handlers.HandleInvokeTool)

		// Memory garden
		v1.GET("/garden/context", handlers.HandleGetGardenContext)
		v1.GET("/garden/stats", handlers.HandleGetGardenStats)
		v1.GET("/seeds", handlers.HandleListSeeds)
		v1.POST("/seeds", handlers.HandleCreateSeed)
		v1.DELETE("/seeds/:id", handlers.HandleDeleteSeed)
		v1.GET("/snapshots/:session", handlers.HandleListSnapshots)
		v1.POST("/snapshots", handlers.HandleCreateSnapshot)
		v1.GET("/snapshots/restore/:snapshot", handlers.HandleRestoreSnapshot)
		v1.DELETE("/snapshots/:id", handlers.HandleDeleteSnapshot)
		v1.GET("/snapshots/export/:id", handlers.HandleExportSnapshot)
		v1.POST("/maintenance/run", handlers.HandleRunMaintenance)
	}

	// ─── Admin Routes (v0.2.0 - New) ─────────────────────────────────
	admin := s.router.Group("/admin")
	{
		// Health and diagnostics
		admin.GET("/health", s.handleAdminHealth)
		admin.GET("/config", s.handleAdminConfig)
		admin.POST("/config/reload", s.handleAdminConfigReload)

		// Metrics
		admin.GET("/metrics/prometheus", s.handleAdminMetrics)

		// MCP server management
		admin.GET("/mcp/servers", s.handleAdminMCPServers)
		admin.GET("/mcp/status", s.handleAdminMCPStatus)
	}
}
