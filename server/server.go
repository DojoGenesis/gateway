package server

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/apps"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/mcp"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/maintenance"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/middleware"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
)

const Version = "1.1.0"

// MCPStatusProvider is the interface used by the server to query MCP status.
// *mcp.MCPHostManager satisfies this interface.
type MCPStatusProvider interface {
	Status() map[string]mcp.ServerStatus
}

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
	Port            string
	AllowedOrigins  []string
	AuthMode        string // "none", "api_key", "custom"
	Environment     string // "development", "production"
	ShutdownTimeout time.Duration
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
	agents  map[string]*gateway.AgentConfig
	agentMu sync.RWMutex

	// Server start time for uptime tracking
	startTime time.Time
}

// New creates a new Server with all dependencies injected.
func New(deps ServerDeps) *Server {
	cfg := deps.Config
	if cfg == nil {
		cfg = &ServerConfig{
			Port:            "8080",
			AllowedOrigins:  []string{"http://localhost:3000"},
			AuthMode:        "api_key",
			Environment:     "production",
			ShutdownTimeout: 30 * time.Second,
		}
	}

	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 30 * time.Second
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
		agents:                make(map[string]*gateway.AgentConfig),
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

	// CORS middleware
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     s.cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

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
