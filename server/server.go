package server

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/mcp"
	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/config"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/maintenance"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/middleware"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
)

const Version = "0.1.0"

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
	gardenManager      *memory.GardenManager
	primaryAgent       *agent.PrimaryAgent
	intentClassifier   *agent.IntentClassifier
	userRouter         *services.UserRouter
	traceLogger        *trace.TraceLogger
	costTracker        *services.CostTracker
	budgetTracker      *services.BudgetTracker
	memoryMaintenance  *maintenance.MemoryMaintenance

	// Phase 2 dependencies (v0.2.0)
	toolRegistry      gateway.ToolRegistry
	agentInitializer  gateway.AgentInitializer
	mcpHostManager    MCPStatusProvider

	// Phase 3 gateway interface implementations
	orchestrationExecutor gateway.OrchestrationExecutor
	memoryStore           gateway.MemoryStore

	// Orchestration state
	orchestrations *OrchestrationStore

	// Agent state (in-memory for v0.2.0)
	agents  map[string]*gateway.AgentConfig
	agentMu sync.RWMutex

	// Server start time for uptime tracking
	startTime time.Time
}

// New creates a new Server with all dependencies injected.
func New(
	cfg *ServerConfig,
	pm *provider.PluginManager,
	orch *orchestrationpkg.Engine,
	plan orchestrationpkg.PlannerInterface,
	mm *memory.MemoryManager,
	gm *memory.GardenManager,
	pa *agent.PrimaryAgent,
	ic *agent.IntentClassifier,
	ur *services.UserRouter,
	tl *trace.TraceLogger,
	ct *services.CostTracker,
	bt *services.BudgetTracker,
	maint *maintenance.MemoryMaintenance,
	toolReg gateway.ToolRegistry,
	agentInit gateway.AgentInitializer,
	mcpMgr MCPStatusProvider,
	orchExec gateway.OrchestrationExecutor,
	memStore gateway.MemoryStore,
) *Server {
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
		pluginManager:         pm,
		orchestrationEngine:   orch,
		planner:               plan,
		memoryManager:         mm,
		gardenManager:         gm,
		primaryAgent:          pa,
		intentClassifier:      ic,
		userRouter:            ur,
		traceLogger:           tl,
		costTracker:           ct,
		budgetTracker:         bt,
		memoryMaintenance:     maint,
		toolRegistry:          toolReg,
		agentInitializer:      agentInit,
		mcpHostManager:        mcpMgr,
		orchestrationExecutor: orchExec,
		memoryStore:           memStore,
		orchestrations:        NewOrchestrationStore(),
		agents:                make(map[string]*gateway.AgentConfig),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// NewFromConfig creates a Server from the application Config.
func NewFromConfig(
	appCfg *config.Config,
	pm *provider.PluginManager,
	orch *orchestrationpkg.Engine,
	plan orchestrationpkg.PlannerInterface,
	mm *memory.MemoryManager,
	gm *memory.GardenManager,
	pa *agent.PrimaryAgent,
	ic *agent.IntentClassifier,
	ur *services.UserRouter,
	tl *trace.TraceLogger,
	ct *services.CostTracker,
	bt *services.BudgetTracker,
	maint *maintenance.MemoryMaintenance,
	toolReg gateway.ToolRegistry,
	agentInit gateway.AgentInitializer,
	mcpMgr MCPStatusProvider,
	orchExec gateway.OrchestrationExecutor,
	memStore gateway.MemoryStore,
) *Server {
	serverCfg := &ServerConfig{
		Port:            appCfg.Port,
		AllowedOrigins:  appCfg.AllowedOrigins,
		AuthMode:        "api_key",
		Environment:     appCfg.Environment,
		ShutdownTimeout: 30 * time.Second,
	}

	return New(serverCfg, pm, orch, plan, mm, gm, pa, ic, ur, tl, ct, bt, maint, toolReg, agentInit, mcpMgr, orchExec, memStore)
}

func (s *Server) setupMiddleware() {
	// Recovery middleware (catch panics)
	s.router.Use(gin.Recovery())

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
		Addr:           ":" + s.cfg.Port,
		Handler:        s.router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   0, // Disable for SSE streaming
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("[Server] Starting Agentic Gateway v%s on %s (environment: %s)", Version, s.httpServer.Addr, s.cfg.Environment)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Server] ListenAndServe error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	log.Printf("[Server] Shutting down gracefully (timeout: %s)...", s.cfg.ShutdownTimeout)

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Server] Shutdown error: %v", err)
		return err
	}

	log.Printf("[Server] Shutdown complete")
	return nil
}
