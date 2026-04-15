package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/server/handlers"
	"github.com/DojoGenesis/gateway/server/middleware"
	wfapi "github.com/DojoGenesis/gateway/workflow/api"
)

// setupRoutes registers all HTTP routes on the Gin engine.
func (s *Server) setupRoutes() {
	// Initialize all handler structs with injected dependencies
	modelHandler := handlers.NewModelHandler(s.pluginManager)
	chatHandler := handlers.NewChatHandler(s.intentClassifier, s.primaryAgent, s.userRouter, s.pluginManager)
	if s.semanticRouter != nil {
		chatHandler.SetSemanticRouter(s.semanticRouter)
	}
	if s.specialistRouter != nil {
		chatHandler.SetSpecialistRouter(&handlers.SpecialistRouterAdapter{Router: s.specialistRouter})
	}
	if s.planner != nil {
		chatHandler.SetOrchestrator(&handlers.OrchestratorAdapter{Planner: s.planner})
	}
	memoryHandler := handlers.NewMemoryHandler(s.memoryManager, s.gardenManager, s.memoryMaintenance)

	// ─── Auth (Portal v1.0) ──────────────────────────────────────────────────────
	// Routes are public — no AuthMiddleware applied.
	// Rate limiting is provided by the global RateLimitMiddleware.
	auth := s.router.Group("/auth")
	{
		auth.POST("/register", s.handleAuthRegister)
		auth.POST("/login", s.handleAuthLogin)
		auth.POST("/refresh", s.handleAuthRefresh)

		// OAuth2 (Wave 1 — GitHub)
		auth.GET("/github", s.handleOAuthGitHubStart)
		auth.GET("/github/callback", s.handleOAuthGitHubCallback)
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

		// ─── Gateway Routes (v1.0.0) ─────────────────────────────────
		gateway := v1.Group("/gateway")
		{
			// Tool discovery with MCP namespace support
			gateway.GET("/tools", s.handleGatewayListTools)

			// Agent management with disposition
			gateway.POST("/agents", s.handleGatewayCreateAgent)
			gateway.GET("/agents", s.handleGatewayListAgents)
			gateway.GET("/agents/:id", s.handleGatewayGetAgent)
			gateway.POST("/agents/:id/chat", s.handleGatewayAgentChat)

			// Agent-channel binding (Gap 5)
			gateway.POST("/agents/:id/channels", s.handleGatewayBindAgentChannels)
			gateway.GET("/agents/:id/channels", s.handleGatewayListAgentChannels)
			gateway.DELETE("/agents/:id/channels/:channel", s.handleGatewayUnbindAgentChannel)

			// Orchestration DAG execution
			gateway.POST("/orchestrate", s.handleGatewayOrchestrate)
			gateway.GET("/orchestrate/:id/dag", s.handleGatewayOrchestrationDAG)

			// Trace inspection (if OTEL enabled)
			gateway.GET("/traces/:id", s.handleGatewayGetTrace)

			// ─── Document fetch (v1.1.0) ───────────────────────────
			gateway.GET("/documents/:id", s.handleGetDocument)

			// ─── MCP Apps (v1.1.0) ─────────────────────────────────
			appsGroup := gateway.Group("/apps")
			{
				appsGroup.POST("/launch", s.handleLaunchApp)
				appsGroup.POST("/close", s.handleCloseApp)
				appsGroup.GET("", s.handleListApps)
				appsGroup.POST("/tool-call", s.handleProxyToolCall)
				appsGroup.GET("/status", s.handleAppStatus)
			}
			// Resource serving (separate from apps group for cleaner URLs)
			gateway.GET("/resources", s.handleGetResource)
		}

		// ─── Legacy endpoints (preserving existing routes) ───────────
		v1.POST("/chat", chatHandler.Chat)
		v1.GET("/providers", modelHandler.ListProviders)
		v1.POST("/tools/search", handlers.HandleSearchTools)
		v1.POST("/tools/invoke", handlers.HandleInvokeTool)

		// Memory garden endpoints (called from server methods via handle_memory.go)
		v1.GET("/garden/context", memoryHandler.GetGardenContext)
		v1.GET("/garden/stats", memoryHandler.GetGardenStats)
		v1.GET("/seeds", memoryHandler.ListSeeds)
		v1.POST("/seeds", memoryHandler.CreateSeed)
		v1.DELETE("/seeds/:id", memoryHandler.DeleteSeed)
		v1.GET("/snapshots/:session", memoryHandler.ListSnapshots)
		v1.POST("/snapshots", memoryHandler.CreateSnapshot)
		v1.GET("/snapshots/restore/:snapshot", memoryHandler.RestoreSnapshot)
		v1.DELETE("/snapshots/:id", memoryHandler.DeleteSnapshot)
		v1.GET("/snapshots/export/:id", memoryHandler.ExportSnapshot)
		v1.POST("/maintenance/run", memoryHandler.RunMaintenance)
	}

	// ─── Settings (v1.1.0) ───────────────────────────────────────────
	settings := v1.Group("/settings")
	{
		settings.POST("/providers", s.handleSetProviderKey)
		settings.GET("/providers", s.handleGetProviderSettings)
	}

	// ─── ADA Validation (Gap 20) ────────────────────────────────────
	// Public endpoint — accessible from frontend without admin auth.
	s.router.POST("/api/ada/validate", s.handleADAValidate)

	// ─── Workflow API (Era 3) ────────────────────────────────────────
	// CRUD: POST/GET /api/workflows, PUT/GET /api/workflows/:name/canvas,
	// POST /api/workflows/:name/validate, GET /api/skills.
	// Requires WorkflowCAS dep; omitted when nil.
	// Note: explicit sub-routes instead of /*path wildcard to avoid conflict
	// with the always-registered execution endpoints below.
	if s.workflowCAS != nil {
		wfHandler := wfapi.NewWorkflowHandler(s.workflowCAS)
		mux := http.NewServeMux()
		wfHandler.RegisterRoutes(mux)
		ginMux := gin.WrapH(mux)
		s.router.Any("/api/workflows", ginMux)
		s.router.GET("/api/workflows/:name", ginMux)
		s.router.GET("/api/workflows/:name/canvas", ginMux)
		s.router.PUT("/api/workflows/:name/canvas", ginMux)
		s.router.POST("/api/workflows/:name/validate", ginMux)
		s.router.GET("/api/skills", ginMux)
	}

	// Execution endpoints (always registered; handler returns 501 when CAS absent).
	s.router.POST("/api/workflows/:name/execute", s.handleWorkflowExecute)
	s.router.GET("/api/workflows/:name/execution", s.handleWorkflowExecutionStream)

	// ─── WebSocket: real-time workflow execution events (Era 3) ──────
	s.router.GET("/api/ws/workflow", s.wsHub.HandleWS)

	// ─── Workflow Builder SPA (Era 3) ────────────────────────────────
	// Served from embedded dist/ compiled by `make build-spa`.
	// SvelteKit base path is /workflow, so all internal links resolve here.
	wbHandler := s.workflowBuilderHandler()
	s.router.GET("/workflow", wbHandler)
	s.router.GET("/workflow/*filepath", wbHandler)

	// ─── Chat UI SPA (Wave 2) ─────────────────────────────────────────
	// Served from embedded dist/ compiled by `make build-chat-spa`.
	// SvelteKit base path is /chat, so all internal links resolve here.
	chatSPAHandler := s.chatUIHandler()
	s.router.GET("/chat", chatSPAHandler)
	s.router.GET("/chat/*filepath", chatSPAHandler)

	// ─── CAS API ────────────────────────────────────────────────────────
	if s.workflowCAS != nil {
		casGroup := s.router.Group("/api/cas")
		{
			// Existing content/tags endpoints
			casGroup.GET("/tags", s.handleCASListTags)
			casGroup.GET("/tags/:name/:version", s.handleCASResolveTag)
			casGroup.POST("/tags", s.handleCASCreateTag)
			casGroup.DELETE("/tags/:name/:version", s.handleCASDeleteTag)
			casGroup.GET("/content/:ref", s.handleCASGetContent)
			casGroup.POST("/content", s.handleCASPutContent)
			casGroup.POST("/gc", s.handleCASGarbageCollect)

			// Gap 1: /api/cas/refs/* endpoints
			casGroup.GET("/refs", s.handleCASListRefs)
			casGroup.GET("/refs/:ref", s.handleCASGetRef)
			casGroup.HEAD("/refs/:ref", s.handleCASHeadRef)
			casGroup.POST("/refs", s.handleCASStoreRef)
			casGroup.GET("/export", s.handleCASExport)
			casGroup.POST("/import", s.handleCASImport)

			// Era 4 Phase 1: D1 sync endpoints
			casGroup.GET("/delta", s.handleCASDelta)
			casGroup.PUT("/batch", s.handleCASBatch)
			casGroup.GET("/status", s.handleCASSyncStatus)
		}
	}

	// ─── Era 4 Phase 0: Federated mesh ──────────────────────────────────────────
	s.router.GET("/.well-known/did.json", s.handleMeshDID)
	meshGroup := s.router.Group("/mesh")
	{
		meshGroup.POST("/announce", s.handleMeshAnnounce)
		meshGroup.GET("/peers", s.handleMeshPeers)
		meshGroup.POST("/delegate", s.handleMeshDelegate)
		meshGroup.GET("/health", s.handleMeshHealth)
	}

	// ─── Admin Routes (v1.0.0) ───────────────────────────────────────
	admin := s.router.Group("/admin")
	admin.Use(middleware.AdminAuthMiddleware())
	{
		// Health and diagnostics
		admin.GET("/health", s.handleAdminHealth)
		admin.GET("/config", s.handleAdminConfig)
		admin.POST("/config/reload", s.handleAdminConfigReload)

		// Provider status
		admin.GET("/providers", s.handleAdminProviders)
		admin.GET("/providers/:name/history", s.handleAdminProviderHistory)

		// Metrics
		admin.GET("/metrics/prometheus", s.handleAdminMetrics)

		// MCP server management
		admin.GET("/mcp/servers", s.handleAdminMCPServers)
		admin.GET("/mcp/status", s.handleAdminMCPStatus)

		// Cost aggregation
		admin.GET("/costs", s.handleAdminCosts)

		// Routing mode control (semantic router hot-switching)
		admin.GET("/routing/mode", s.handleAdminRoutingMode)
		admin.POST("/routing/mode", s.handleAdminSetRoutingMode)
		admin.GET("/routing/stats", s.handleAdminRoutingStats)
		admin.POST("/routing/threshold", s.handleAdminSetRouteThreshold)

		// User management (Wave 1)
		admin.GET("/users", s.handleAdminListUsers)
		admin.POST("/users/:id/deactivate", s.handleAdminDeactivateUser)
		admin.POST("/users/:id/activate", s.handleAdminActivateUser)
	}

	// ─── Conversations API (Wave 1) ─────────────────────────────────
	convGroup := v1.Group("/conversations")
	convGroup.Use(middleware.AuthMiddleware())
	{
		convGroup.GET("", s.handleListConversations)
		convGroup.POST("", s.handleCreateConversation)
		convGroup.GET("/:id", s.handleGetConversation)
		convGroup.DELETE("/:id", s.handleDeleteConversation)
		convGroup.GET("/:id/messages", s.handleListMessages)
		convGroup.POST("/:id/messages", s.handleCreateMessage)
	}

	// ─── Prompt Templates API (Wave 2) ──────────────────────────────
	tmplGroup := v1.Group("/templates")
	tmplGroup.Use(middleware.AuthMiddleware())
	{
		tmplGroup.GET("", s.handleListTemplates)
		tmplGroup.GET("/:id", s.handleGetTemplate)
		tmplGroup.POST("", s.handleCreateTemplate)
		tmplGroup.PUT("/:id", s.handleUpdateTemplate)
		tmplGroup.DELETE("/:id", s.handleDeleteTemplate)
	}

	// ─── Documents / RAG API (Wave 2) ───────────────────────────────
	docGroup := v1.Group("/documents")
	docGroup.Use(middleware.AuthMiddleware())
	{
		docGroup.POST("", s.handleUploadDocument)
		docGroup.GET("", s.handleListRAGDocuments)
		docGroup.GET("/:id", s.handleGetRAGDocument)
		docGroup.DELETE("/:id", s.handleDeleteRAGDocument)
		docGroup.POST("/search", s.handleSearchRAGDocuments)
	}
}
