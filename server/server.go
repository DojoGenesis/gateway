package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
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

// MCPStatusProvider is the interface used by the server to query MCP status
// and invoke MCP-bridged tools directly by server+tool name.
// *mcp.MCPHostManager satisfies this interface.
type MCPStatusProvider interface {
	Status() map[string]mcp.ServerStatus
	CallTool(ctx context.Context, serverName string, toolName string, args map[string]interface{}) (map[string]interface{}, error)
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
	Port string
	// BindHost is the interface the listener binds (DGS-113). Empty means
	// DefaultBindHost — loopback. Widening it ("0.0.0.0", a LAN address) is an
	// explicit operator act: main.go fills it from GATEWAY_BIND_HOST or the
	// config file's bind_host, and container images set it because inside a
	// container loopback is unreachable from the published port.
	BindHost        string
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
	// When false, the endpoint returns 403. Defaults to true — this gateway
	// deliberately lets anyone sign up and use the chat.
	//
	// main.go sets this from config.Config.RegistrationEnabled, which reads
	// `registration_enabled` in the config file and REGISTRATION_ENABLED in
	// the environment. It was hardcoded true here and in main.go until the
	// config key was made real; a deployment that wrote the key got nothing.
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
		mesh:                  deps.Mesh,
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

// DefaultBindHost is where the gateway listens when nothing says otherwise.
//
// It used to be every interface: Start built its address as ":" + port, so a
// gateway run as a bare binary, via `go run`, or from any unit that did not
// think about it answered on the LAN (DGS-113). That matters more here than
// usual because /health, /metrics and /auth/* are public by design and the
// development JWT secret is publicly known — its only protection off
// production is not being reachable. Unset must be the safe branch.
const DefaultBindHost = "127.0.0.1"

// listenAddr resolves the address the HTTP server binds. An empty or
// whitespace-only BindHost is treated as unset, so a blank line in an env file
// cannot widen the bind.
func (s *Server) listenAddr() string {
	host := strings.TrimSpace(s.cfg.BindHost)
	if host == "" {
		host = DefaultBindHost
	}
	return net.JoinHostPort(host, s.cfg.Port)
}

// Start begins listening for HTTP requests.
//
// The listener is opened before Start returns, so a bind failure (port in use,
// a bind host that is not an address on this machine) is returned to the
// caller instead of being logged from a goroutine while the process keeps
// running with nothing listening.
func (s *Server) Start() error {
	s.startTime = time.Now()

	addr := s.listenAddr()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	if host, _, _ := net.SplitHostPort(addr); !isLoopbackHost(host) {
		slog.Warn("gateway is listening beyond loopback — reachable from other machines on this network",
			"addr", addr,
			"hint", "intended inside a container or behind a firewall; unset GATEWAY_BIND_HOST to bind 127.0.0.1 only")
	}

	s.httpServer = &http.Server{
		Addr:              addr,
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
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// isLoopbackHost reports whether host names only the loopback interface.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
