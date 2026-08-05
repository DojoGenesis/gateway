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
		chatHandler.SetOrchestrator(&handlers.OrchestratorAdapter{
			Planner:     s.planner,
			StartOrchFn: s.StartOrchestrationForChat,
		})
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
	// DGS-100 item 5. Gated: it streams broadcaster events to anyone who names
	// a client_id, and it has no caller anywhere in this repo or the CLI — only
	// this registration. /health and /metrics stay public: the deploy liveness
	// probe in deploy/provision.sh hits /health unauthenticated by design.
	s.router.GET("/events", middleware.AuthMiddleware(), handlers.HandleSSE)

	// ─── OpenAI-Compatible API (/v1) ─────────────────────────────────
	v1 := s.router.Group("/v1")
	// DGS-88: every route under /v1 either spends money (LLM completions,
	// agent chat, orchestration), mutates durable state (memory, seeds,
	// snapshots, CAS-backed settings), or reads private data. None of it is
	// public. Before this line the only middleware reaching /v1 was the global
	// OptionalAuthMiddleware, which never rejects — it assigns a random guest
	// UUID and calls c.Next() — so an anonymous caller from the public internet
	// could POST /v1/gateway/agents and /v1/gateway/agents/:id/chat and bill
	// real provider spend.
	//
	// gin snapshots the handler chain at route/subgroup creation time
	// (RouterGroup.combineHandlers), so this Use() MUST stay above every route
	// and every v1.Group(...) below — including the /gateway and /settings
	// subgroups. Moving it down silently un-protects everything above it.
	//
	// /health, /metrics, /auth/* and /.well-known/did.json are registered on
	// s.router outside this group and stay public — the deploy liveness probe
	// (deploy/provision.sh) hits /health unauthenticated by design.
	v1.Use(middleware.AuthMiddleware())
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

	// ─── /api/* authentication (DGS-100) ────────────────────────────────
	//
	// Every /api/* route below carries AuthMiddleware explicitly, on BOTH
	// handler paths. There is no /api group to hang it on: these routes are
	// registered directly on s.router at several points, and the workflow CRUD
	// routes are a bare http.ServeMux mounted through gin.WrapH — that mux
	// cannot take gin middleware itself, but the gin route that WRAPS it can,
	// which is why the middleware sits at the registration site rather than
	// inside workflow/api/handler.go.
	//
	// WHY THIS COULD FINALLY LAND: DGS-100 was blocked from 2026-07-24 on the
	// belief that gating these routes would break the workflow-builder SPA,
	// which calls /api/workflows* from the browser with no token and streams
	// execution over EventSource (which cannot send an Authorization header at
	// all). That consumer does not exist in any deployed gateway: the embedded
	// dist directories hold only .gitkeep, goreleaser's only before-hook is
	// `go mod download`, and production answers 503 on /workflow and /chat. No
	// released binary has ever contained the SPA.
	//
	// So the SPA's auth story is now a PREREQUISITE FOR SHIPPING IT, not a
	// blocker on closing a live hole. When it does ship it needs a
	// Secure/HttpOnly/SameSite cookie — see docs/api-route-disposition.md — and
	// this is deliberately NOT weakened with an "allow anonymous /api" flag in
	// the meantime: a switch that turns authentication off is the exact shape
	// of defect DGS-112 was.
	//
	// Known cost, accepted: the `dojo` CLI attaches a bearer token only when
	// one is configured, and that is not the default. A CLI with no token now
	// gets 401 on /api/cas/* and /api/skills. The credential already exists
	// (svc:dojo-cli); it has to be configured before this reaches a host the
	// CLI talks to.
	//
	// /mesh/* and /.well-known/did.json are deliberately NOT covered: federation
	// peers cannot hold a gateway JWT, so they need DID-signature auth, which is
	// a separate design.

	// ─── ADA Validation (Gap 20) ────────────────────────────────────
	// Was registered bare with a comment claiming the frontend called it
	// without auth. Nothing calls it — no SPA, no CLI, no script, and it has no
	// test file. Gated with everything else rather than left as the one open
	// POST on the surface.
	s.router.POST("/api/ada/validate", middleware.AuthMiddleware(), s.handleADAValidate)

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
		s.router.Any("/api/workflows", middleware.AuthMiddleware(), ginMux)
		s.router.GET("/api/workflows/:name", middleware.AuthMiddleware(), ginMux)
		s.router.GET("/api/workflows/:name/canvas", middleware.AuthMiddleware(), ginMux)
		s.router.PUT("/api/workflows/:name/canvas", middleware.AuthMiddleware(), ginMux)
		s.router.POST("/api/workflows/:name/validate", middleware.AuthMiddleware(), ginMux)
		s.router.GET("/api/skills", middleware.AuthMiddleware(), ginMux)
	}

	// Execution endpoints (always registered; handler returns 501 when CAS absent).
	// POST /execute is the route DGS-108 reached a shell through; it is now
	// authenticated as well as capability-gated, and those are independent
	// controls — neither replaces the other.
	s.router.POST("/api/workflows/:name/execute", middleware.AuthMiddleware(), s.handleWorkflowExecute)
	s.router.GET("/api/workflows/:name/execution", middleware.AuthMiddleware(), s.handleWorkflowExecutionStream)

	// ─── WebSocket: real-time workflow execution events (Era 3) ──────
	// No production caller: neither SPA opens a WebSocket, and the CLI never
	// dials it. A browser WebSocket cannot send an Authorization header either,
	// so if the Phase 2 bridge is ever built it needs the same cookie the SPA
	// will need.
	s.router.GET("/api/ws/workflow", middleware.AuthMiddleware(), s.wsHub.HandleWS)

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
	// DGS-100: the group carries AuthMiddleware now. It previously had no
	// .Use() at all, so every route below — including durable writes via
	// /content, /tags, /import and /batch — served anonymous callers.
	//
	// The only live consumer is the `dojo` CLI (cli/internal/commands/
	// cmd_skill.go), which sends a bearer token only when one is configured.
	// That is not the default, so a CLI without a configured token gets 401
	// here. The svc:dojo-cli credential already exists and has to be wired up
	// before this reaches a host the CLI talks to.
	if s.workflowCAS != nil {
		casGroup := s.router.Group("/api/cas", middleware.AuthMiddleware())
		{
			// Existing content/tags endpoints
			casGroup.GET("/tags", s.handleCASListTags)
			casGroup.GET("/tags/:name/:version", s.handleCASResolveTag)
			casGroup.POST("/tags", s.handleCASCreateTag)
			casGroup.DELETE("/tags/:name/:version", s.handleCASDeleteTag)
			casGroup.GET("/content/:ref", s.handleCASGetContent)
			casGroup.POST("/content", s.handleCASPutContent)

			// DGS-100: /gc calls workflowCAS.GC(), which DELETES content, and
			// the /api/cas group carries no middleware — so this was an
			// unauthenticated destructive operation on a publicly reachable
			// gateway. Admin-gated here rather than left open pending the
			// broader /api auth design, because it has no caller to break:
			// no SPA (chat-ui and workflow-builder call only /api/workflows*
			// and /api/skills), no `dojo` CLI (it uses /content and /tags),
			// no MCP server, no worker. The path is deliberately unchanged so
			// any undiscovered caller gets a diagnosable 401 rather than a 404.
			casGroup.POST("/gc", middleware.AdminAuthMiddleware(), s.handleCASGarbageCollect)

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
		admin.POST("/mcp/tools/invoke", s.handleAdminInvokeMCPTool)

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
