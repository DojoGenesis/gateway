package server

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/DojoGenesis/gateway/apps"
	"github.com/DojoGenesis/gateway/disposition"
	"github.com/DojoGenesis/gateway/mcp"
	"github.com/DojoGenesis/gateway/memory"
	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/DojoGenesis/gateway/pkg/collaboration"
	pkgerrors "github.com/DojoGenesis/gateway/pkg/errors"
	"github.com/DojoGenesis/gateway/pkg/gateway"
	"github.com/DojoGenesis/gateway/pkg/intelligence"
	"github.com/DojoGenesis/gateway/pkg/reflection"
	"github.com/DojoGenesis/gateway/pkg/validation"
	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/runtime/cas"
	"github.com/DojoGenesis/gateway/runtime/mesh"
	"github.com/DojoGenesis/gateway/server/agent"
	"github.com/DojoGenesis/gateway/server/maintenance"
	"github.com/DojoGenesis/gateway/server/middleware"
	"github.com/DojoGenesis/gateway/server/services"
	"github.com/DojoGenesis/gateway/server/trace"
	"github.com/DojoGenesis/gateway/specialist"
)

// Version is the server version. The default is the development version string;
// goreleaser overrides this at build time via ldflags:
//
//	-X github.com/DojoGenesis/gateway/server.Version={{.Version}}
var Version = "1.1.0"

// MCPStatusProvider is the interface used by the server to query MCP status.
// *mcp.MCPHostManager satisfies this interface.
type MCPStatusProvider interface {
	Status() map[string]mcp.ServerStatus
}

// AgentRuntime holds per-agent disposition config and instantiated consumer modules.
// Created when an agent is initialized via POST /v1/gateway/agents.
type AgentRuntime struct {
	Config        *gateway.AgentConfig
	Disposition   *disposition.DispositionConfig
	ErrorHandler  *pkgerrors.Handler
	CollabManager *collaboration.Manager
	Validator     *validation.Validator
	Reflection    *reflection.Engine
	Proactive     *intelligence.ProactiveEngine

	// Channels lists the channel IDs this agent is bound to.
	Channels []string
}

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
	Port            string
	AllowedOrigins  []string
	AuthMode        string // "none", "api_key", "custom"
	Environment     string // "development", "production"
	ShutdownTimeout time.Duration
	// Auth token TTLs (configurable, defaults: access=24h, refresh=7d)
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	// AdminAPIKey is reserved for a future simple bearer-token fallback on admin
	// endpoints. Currently unused — AdminAuthMiddleware uses JWT validation only.
	AdminAPIKey string
	// RegistrationEnabled controls whether POST /auth/register is open.
	// When false, the endpoint returns 403. Defaults to true.
	RegistrationEnabled bool
}

// Server is the main HTTP server that ties all framework modules together.
type Server struct {
	router     *gin.Engine
	cfg        *ServerConfig
	httpServer *http.Server

	// Injected dependencies
	pluginManager       *provider.PluginManager
	orchestrationEngine *orchestrationpkg.Engine
	planner             orchestrationpkg.PlannerInterface
	memoryManager       *memory.MemoryManager
	gardenManager       *memory.GardenManager
	primaryAgent        *agent.PrimaryAgent
	intentClassifier    *agent.IntentClassifier
	userRouter          *services.UserRouter
	traceLogger         *trace.TraceLogger
	costTracker         *services.CostTracker
	budgetTracker       *services.BudgetTracker
	memoryMaintenance   *maintenance.MemoryMaintenance

	// Gateway interface dependencies
	toolRegistry     gateway.ToolRegistry
	agentInitializer gateway.AgentInitializer
	mcpHostManager   MCPStatusProvider

	// Orchestration and memory interfaces
	orchestrationExecutor gateway.OrchestrationExecutor
	memoryStore           gateway.MemoryStore

	// MCP Apps (v1.1.0)
	appManager *apps.AppManager

	// Auth database (Portal v1.0)
	authDB *sql.DB

	// Orchestration state
	orchestrations *OrchestrationStore

	// Agent state (in-memory for v1.0.0)
	agents  map[string]*AgentRuntime
	agentMu sync.RWMutex

	// Server start time for uptime tracking
	startTime time.Time

	// Workflow execution (Era 3)
	workflowCAS cas.Store
	execBus     *ExecutionBus

	// D1 sync loop (Era 4 Phase 1). Nil when DOJO_D1_* env vars are not set.
	d1Syncer *cas.D1Syncer

	// Provider latency tracking (Gap 13)
	latencyTracker *services.ProviderLatencyTracker

	// Telemetry tap: optional forwarder to Cloudflare Worker (Phase 1).
	// Nil when DOJO_TELEMETRY_WORKER_URL is not set.
	telemetryTap *services.TelemetryTap

	// WebSocket hub for real-time workflow execution events (Era 3)
	wsHub *WorkflowWSHub

	// SemanticRouter replaces the deprecated IntentClassifier with embedding-based
	// routing. When non-nil, the chat handler delegates to it instead of the
	// keyword-based classifier. Hot-switchable between cascade/llm/embedding modes.
	semanticRouter *agent.SemanticRouter
	// semanticRouterInitOnce ensures that lazy init is only kicked off once on
	// rapid concurrent POST /v1/gateway/providers calls.
	semanticRouterInitOnce sync.Once

	// Specialist dispatch (Phase 2): routes requests to specialist agents
	// based on intent classification. Nil means specialist dispatch is disabled.
	specialistRouter *specialist.Router

	// Federated agent mesh (Era 4 Phase 0). Nil when mesh is not configured.
	mesh *mesh.Mesh
}

// New creates a new Server with all dependencies injected.
func New(deps ServerDeps) *Server {
	cfg := deps.Config
	if cfg == nil {
		cfg = &ServerConfig{
			Port:                "7340",
			AllowedOrigins:      []string{"http://localhost:3000"},
			AuthMode:            "api_key",
			Environment:         "production",
			ShutdownTimeout:     30 * time.Second,
			RegistrationEnabled: true,
		}
	}

	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 30 * time.Second
	}
	if cfg.AccessTokenTTL == 0 {
		cfg.AccessTokenTTL = 24 * time.Hour
	}
	if cfg.RefreshTokenTTL == 0 {
		cfg.RefreshTokenTTL = 7 * 24 * time.Hour
	}

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	s := &Server{
		router:                gin.New(),
		cfg:                   cfg,
		pluginManager:         deps.PluginManager,
		orchestrationEngine:   deps.OrchestrationEngine,
		planner:               deps.Planner,
		memoryManager:         deps.MemoryManager,
		gardenManager:         deps.GardenManager,
		primaryAgent:          deps.PrimaryAgent,
		intentClassifier:      deps.IntentClassifier,
		userRouter:            deps.UserRouter,
		traceLogger:           deps.TraceLogger,
		costTracker:           deps.CostTracker,
		budgetTracker:         deps.BudgetTracker,
		memoryMaintenance:     deps.MemoryMaintenance,
		toolRegistry:          deps.ToolRegistry,
		agentInitializer:      deps.AgentInitializer,
		mcpHostManager:        deps.MCPHostManager,
		orchestrationExecutor: deps.OrchestrationExec,
		memoryStore:           deps.MemoryStore,
		appManager:            deps.AppManager,
		authDB:                deps.AuthDB,
		orchestrations:        NewOrchestrationStore(),
		agents:                make(map[string]*AgentRuntime),
		workflowCAS:           deps.WorkflowCAS,
		d1Syncer:              deps.D1Syncer,
		semanticRouter:        deps.SemanticRouter,
		specialistRouter:      deps.SpecialistRouter,
		execBus:               newExecutionBus(),
		latencyTracker:        services.NewProviderLatencyTracker(60),
		wsHub:                 NewWorkflowWSHub(),
	}

	// Start WebSocket broadcast loop in background.
	go s.wsHub.Run()

	// Telemetry tap: forward SSE events to CF Worker if configured.
	if url := os.Getenv("DOJO_TELEMETRY_WORKER_URL"); url != "" {
		tap := services.NewTelemetryTap(url)
		tap.Start()
		s.telemetryTap = tap
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	// Recovery middleware (catch panics)
	s.router.Use(gin.Recovery())

	// Inject environment into context for middleware that needs it (e.g. HSTS)
	env := s.cfg.Environment
	s.router.Use(func(c *gin.Context) {
		c.Set("environment", env)
		c.Next()
	})

	// Security headers middleware
	s.router.Use(middleware.SecurityHeadersMiddleware())

	// Rate limiting middleware (per-IP token bucket)
	s.router.Use(middleware.RateLimitMiddleware(middleware.DefaultRateLimitConfig()))

	// CORS middleware — supports wildcard "*" for development
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           3600,
	}
	if len(s.cfg.AllowedOrigins) == 1 && s.cfg.AllowedOrigins[0] == "*" {
		// Dynamic origin: reflect the requesting origin (dev mode)
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowCredentials = false // AllowAllOrigins + credentials is invalid
	} else {
		corsConfig.AllowOrigins = s.cfg.AllowedOrigins
	}
	s.router.Use(cors.New(corsConfig))

	// Request ID middleware
	s.router.Use(requestIDMiddleware())

	// Request logging middleware
	s.router.Use(middleware.Logger())

	// Auth middleware (optional, based on config)
	if s.cfg.AuthMode == "api_key" {
		// Use optional auth — allows unauthenticated requests for public endpoints
		s.router.Use(middleware.OptionalAuthMiddleware())
	}

	// Budget middleware (optional)
	if s.budgetTracker != nil && s.costTracker != nil {
		s.router.Use(middleware.BudgetMiddleware(s.budgetTracker, s.costTracker))
	}
}

// requestIDMiddleware assigns a unique request ID to each request.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	s.startTime = time.Now()

	s.httpServer = &http.Server{
		Addr:              ":" + s.cfg.Port,
		Handler:           s.router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      0, // Disable for SSE streaming
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	slog.Info("starting Agentic Gateway",
		"version", Version,
		"addr", s.httpServer.Addr,
		"environment", s.cfg.Environment)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("ListenAndServe error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	slog.Info("shutting down gracefully", "timeout", s.cfg.ShutdownTimeout)

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		return err
	}

	slog.Info("shutdown complete")
	return nil
}
