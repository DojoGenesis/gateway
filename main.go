package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/apps"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/mcp"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	pkgdisposition "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/config"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/logging"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	svcproviders "github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services/providers"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"

	srv "github.com/TresPies-source/AgenticGatewayByDojoGenesis/server"

	_ "modernc.org/sqlite"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
	// ─── Health Check Mode (for Docker HEALTHCHECK in distroless) ────
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		port := getEnv("PORT", "8080")
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port))
		if err != nil {
			os.Exit(1)
		}
		_ = resp.Body.Close() // Explicitly ignore error in health check
		if resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// ─── Load Configuration ──────────────────────────────────────────
	cfg := config.Load()

	// Initialize structured logging based on environment
	logging.Init(cfg.Environment)

	slog.Info("Agentic Gateway starting", "version", srv.Version)

	if err := cfg.Validate(); err != nil {
		slog.Warn("config validation issue", "error", err)
	}
	slog.Info("configuration loaded", "port", cfg.Port, "environment", cfg.Environment)

	// ─── Initialize OTEL (if enabled) ────────────────────────────────
	var tracerProvider *sdktrace.TracerProvider
	if cfg.OTEL.Enabled {
		slog.Info("initializing OTEL exporter", "endpoint", cfg.OTEL.Endpoint)

		exporter, err := trace.NewOTELExporter(cfg.OTEL.Endpoint)
		if err != nil {
			slog.Warn("OTEL exporter initialization failed", "error", err)
		} else if exporter != nil {
			// Create resource with service name
			res, _ := resource.New(context.Background(),
				resource.WithAttributes(
					semconv.ServiceNameKey.String(cfg.OTEL.ServiceName),
				),
			)

			// Create tracer provider
			tracerProvider = sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(res),
				sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.OTEL.SamplingRate)),
			)

			// Set global tracer provider
			otel.SetTracerProvider(tracerProvider)
			slog.Info("OTEL tracer provider initialized",
				"service", cfg.OTEL.ServiceName,
				"sampling_rate", cfg.OTEL.SamplingRate)
		}
	} else {
		slog.Info("OTEL disabled")
	}

	// ─── Initialize Provider Plugin Manager ──────────────────────────
	pluginManager := provider.NewPluginManager(cfg.PluginDir)
	slog.Info("plugin manager initialized", "dir", cfg.PluginDir)

	// ─── Register In-Process Providers ───────────────────────────────
	providerResults := services.RegisterProviders(context.Background(), pluginManager, cfg, nil)
	loadedCount := 0
	ollamaLoadedAtStartup := false
	for _, r := range providerResults {
		if r.Available {
			loadedCount++
			if r.Name == "ollama" {
				ollamaLoadedAtStartup = true
			}
		}
	}
	slog.Info("provider registration complete", "loaded", loadedCount, "total_checked", len(providerResults))

	// ─── Local Provider Retry Loop ────────────────────────────────────
	// If Ollama wasn't reachable at startup (e.g. still loading, or Tauri app
	// launched before Ollama was ready), poll every 30s and register it as
	// soon as it becomes available.  This prevents the "0 providers" state
	// that blocks all chat requests when the gateway starts before Ollama.
	if !ollamaLoadedAtStartup {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				// Stop retrying once it's loaded (e.g. via plugin or prior tick).
				if pluginManager.IsPluginLoaded("ollama") {
					slog.Info("ollama provider already registered — stopping retry loop")
					return
				}
				ollamaProvider := svcproviders.NewOllamaProvider()
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				available := ollamaProvider.IsAvailable(ctx)
				cancel()
				if available {
					pluginManager.RegisterProvider("ollama", ollamaProvider)
					slog.Info("ollama provider registered (late — was not available at startup)")
					return
				}
				slog.Debug("ollama still not available — will retry in 30s")
			}
		}()
	}

	// ─── Initialize Tool Registry ────────────────────────────────────
	allTools := tools.GetAllTools()
	toolRegistry := tools.NewContextAwareRegistry()
	slog.Info("tool registry initialized", "builtin_tools", len(allTools))

	// ─── Initialize MCP Host (if configured) ─────────────────────────
	var mcpHostManager *mcp.MCPHostManager
	mcpConfigPath := getEnv("MCP_CONFIG_PATH", "gateway-config.yaml")
	if _, err := os.Stat(mcpConfigPath); err == nil {
		slog.Info("loading MCP configuration", "path", mcpConfigPath)

		mcpHostCfg, err := mcp.LoadConfig(mcpConfigPath)
		if err != nil {
			slog.Warn("failed to load MCP config", "error", err)
		} else {
			// Preflight: warn about env vars that expanded to empty strings.
			if w := mcpHostCfg.MCP.ValidateExpansions(); w > 0 {
				slog.Warn("MCP config has missing env var expansions — some servers may fail to start", "warnings", w)
			}
			mcpHostManager, err = mcp.NewMCPHostManager(&mcpHostCfg.MCP, toolRegistry)
			if err != nil {
				slog.Warn("failed to create MCP host manager", "error", err)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := mcpHostManager.Start(ctx); err != nil {
					slog.Warn("failed to start MCP host", "error", err)
					mcpHostManager = nil
				} else {
					mcpStatus := mcpHostManager.Status()
					totalMCPTools := 0
					for _, status := range mcpStatus {
						totalMCPTools += status.ToolCount
					}
					slog.Info("MCP host started", "servers", len(mcpStatus), "tools", totalMCPTools)
				}
				cancel()
			}
		}
	} else {
		slog.Info("MCP configuration not found, skipping", "path", mcpConfigPath)
	}

	// ─── Initialize MCP Apps Manager (v1.1.0) ──────────────────────
	var appManager *apps.AppManager
	if cfg.MCPApps.Enabled {
		allowedOrigins := cfg.MCPApps.AllowedOrigins
		if len(allowedOrigins) == 0 {
			allowedOrigins = cfg.AllowedOrigins
		}
		appManager = apps.NewAppManager(apps.AppManagerConfig{
			AllowedOrigins:     allowedOrigins,
			DefaultToolTimeout: 30 * time.Second,
		}, toolRegistry)
		slog.Info("MCP Apps manager initialized")

		// Register built-in MCP App dashboards
		builtinApps := []struct{ uri, path string }{
			{"ui://dip/dashboard", "apps/dip-dashboard/dashboard.html"},
			{"ui://observability/dashboard", "apps/observability-dashboard/dashboard.html"},
		}
		for _, ba := range builtinApps {
			content, readErr := os.ReadFile(ba.path)
			if readErr != nil {
				slog.Warn("built-in app not found, skipping", "uri", ba.uri, "path", ba.path)
				continue
			}
			regErr := appManager.RegisterResource(&apps.ResourceMeta{
				URI:      ba.uri,
				MimeType: "text/html",
				Content:  content,
				CacheKey: fmt.Sprintf("%s-%d", ba.uri, len(content)),
			})
			if regErr != nil {
				slog.Warn("failed to register built-in app", "uri", ba.uri, "error", regErr)
			} else {
				slog.Info("registered built-in MCP app", "uri", ba.uri, "size", len(content))
			}
		}
	} else {
		slog.Info("MCP Apps disabled (set MCP_APPS_ENABLED=true to enable)")
	}

	// ─── Initialize Disposition / Agent Initializer ──────────────────
	dispositionCacheTTL := 5 * time.Minute
	agentInitializer := pkgdisposition.NewAgentInitializer(dispositionCacheTTL)
	slog.Info("agent initializer created", "cache_ttl", dispositionCacheTTL)

	// ─── Load Default Disposition (ADA) ──────────────────────────────
	workspaceRoot := getEnv("AGENT_WORKSPACE_ROOT", ".")
	defaultDisp, dispErr := disposition.ResolveDisposition(workspaceRoot, "")
	if dispErr != nil {
		slog.Warn("failed to load disposition from workspace, using defaults", "workspace", workspaceRoot, "error", dispErr)
		defaultDisp = disposition.DefaultDisposition()
	}
	slog.Info("disposition loaded",
		"source", defaultDisp.SourceFile,
		"pacing", defaultDisp.Pacing,
		"depth", defaultDisp.Depth,
		"tone", defaultDisp.Tone,
		"initiative", defaultDisp.Initiative)

	// ─── Initialize Memory ───────────────────────────────────────────
	dbPath := getEnv("MEMORY_DB_PATH", "dojo_memory.db")
	memoryManager, err := memory.NewMemoryManager(dbPath)
	if err != nil {
		slog.Warn("memory manager initialization failed", "error", err)
		memoryManager = nil
	} else {
		slog.Info("memory manager initialized", "db", dbPath)
	}

	var gardenManager *memory.GardenManager
	if memoryManager != nil {
		gardenManager, err = memory.NewGardenManager(memoryManager, nil)
		if err != nil {
			slog.Warn("garden manager initialization failed", "error", err)
		} else {
			slog.Info("garden manager initialized")
		}
	}

	// ─── Initialize Services ─────────────────────────────────────────
	costTracker := services.NewCostTracker()
	budgetTracker := services.NewBudgetTracker(
		cfg.Budget.QueryLimit,
		cfg.Budget.SessionLimit,
		cfg.Budget.MonthlyLimit,
	)
	userRouter := services.NewUserRouter(cfg, pluginManager, budgetTracker)
	slog.Info("services initialized",
		"query_limit", cfg.Budget.QueryLimit,
		"session_limit", cfg.Budget.SessionLimit,
		"monthly_limit", cfg.Budget.MonthlyLimit)

	// ─── Initialize Trace Logger ─────────────────────────────────────
	traceLogger := trace.NewTraceLoggerWithoutEvents(nil)

	// ─── Initialize Agent ────────────────────────────────────────────
	intentClassifier := agent.NewIntentClassifier()
	primaryAgent := agent.NewPrimaryAgentWithConfig(
		pluginManager,
		cfg.Routing.DefaultProvider,
		cfg.Routing.GuestProvider,
		cfg.Routing.AuthenticatedProvider,
	)
	slog.Info("agent initialized",
		"default_provider", cfg.Routing.DefaultProvider,
		"guest_provider", cfg.Routing.GuestProvider,
		"auth_provider", cfg.Routing.AuthenticatedProvider)

	// ─── Initialize Orchestration ────────────────────────────────────
	providerName := getEnv("ORCHESTRATION_PROVIDER", cfg.Routing.DefaultProvider)
	modelID := getEnv("ORCHESTRATION_MODEL", "")

	// Create concrete planner implementation (server adapter)
	planner := orchestration.NewPlanner(pluginManager, traceLogger, providerName, modelID)

	// Create adapters for the standalone orchestration engine
	toolInvoker := orchestration.NewToolInvokerAdapter()
	traceAdapter := orchestration.NewTraceLoggerAdapter(traceLogger)
	budgetAdapter := orchestration.NewBudgetTrackerAdapter(budgetTracker)

	// Event emitter starts nil — set per-execution in handle_orchestrate.go
	// via engine.SetEventEmitter() with a per-request EventEmitterAdapter.
	var eventEmitter orchestrationpkg.EventEmitterInterface = nil

	// Create the standalone orchestration engine with all adapters
	orchestrationEngine := orchestrationpkg.NewEngine(
		orchestrationpkg.DefaultEngineConfig(),
		planner,
		toolInvoker,
		traceAdapter,
		eventEmitter,
		budgetAdapter,
		orchestrationpkg.WithDisposition(defaultDisp),
	)
	slog.Info("orchestration engine initialized", "provider", providerName, "pacing", defaultDisp.Pacing)

	// ─── Initialize Gateway Interface Implementations ────────────────
	// Wrap existing components to implement gateway interfaces
	orchestrationExecutor := orchestration.NewGatewayOrchestrationExecutor(orchestrationEngine, planner)

	var memoryStore gateway.MemoryStore
	if memoryManager != nil {
		memoryStore = memory.NewGatewayMemoryStore(memoryManager)
		slog.Info("gateway memory store initialized")
	}

	// ─── Initialize Auth Database (Portal v1.0) ─────────────────────
	var authDB *sql.DB
	authDBDir := getEnv("AUTH_DB_DIR", ".dojo")
	if err := os.MkdirAll(authDBDir, 0o755); err != nil {
		slog.Warn("failed to create auth DB directory", "dir", authDBDir, "error", err)
	}
	authDBPath := filepath.Join(authDBDir, "dojo.db")
	authDB, err = sql.Open("sqlite", authDBPath)
	if err != nil {
		slog.Warn("failed to open auth database", "path", authDBPath, "error", err)
	} else {
		// Apply portal auth migration if not yet applied.
		// Check schema_migrations — if table doesn't exist or row not found, apply migration.
		needsMigration := true
		var migrationVersion string
		row := authDB.QueryRow("SELECT version FROM schema_migrations WHERE version = '20260219_v1.0.0_portal_auth'")
		if scanErr := row.Scan(&migrationVersion); scanErr == nil {
			needsMigration = false
			slog.Info("portal auth migration already applied")
		}
		if needsMigration {
			slog.Info("applying portal auth migration")
			stmts := []string{
				"ALTER TABLE local_users ADD COLUMN email TEXT",
				"ALTER TABLE local_users ADD COLUMN password_hash TEXT",
				"ALTER TABLE local_users ADD COLUMN display_name TEXT",
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_local_users_email ON local_users(email) WHERE email IS NOT NULL",
				"INSERT OR IGNORE INTO schema_migrations (version, applied_at, description) VALUES ('20260219_v1.0.0_portal_auth', datetime('now'), 'Portal auth: add email, password_hash, display_name to local_users')",
			}
			for _, stmt := range stmts {
				if _, execErr := authDB.Exec(stmt); execErr != nil {
					slog.Warn("portal auth migration statement failed (may already exist)", "stmt", stmt, "error", execErr)
				}
			}
			slog.Info("portal auth migration applied")
		}
	}

	// ─── Create and Start Server ─────────────────────────────────────
	// Guard against the Go interface nil trap: assigning a typed nil (*mcp.MCPHostManager)(nil)
	// to MCPStatusProvider produces a non-nil interface, breaking the nil check inside the server.
	// Only set the interface when we have a real manager.
	var mcpProvider srv.MCPStatusProvider
	if mcpHostManager != nil {
		mcpProvider = mcpHostManager
	}

	server := srv.New(srv.ServerDeps{
		Config: &srv.ServerConfig{
			Port:            cfg.Port,
			AllowedOrigins:  cfg.AllowedOrigins,
			AuthMode:        "api_key",
			Environment:     cfg.Environment,
			ShutdownTimeout: 30 * time.Second,
		},
		PluginManager:       pluginManager,
		OrchestrationEngine: orchestrationEngine,
		Planner:             planner,
		MemoryManager:       memoryManager,
		GardenManager:       gardenManager,
		PrimaryAgent:        primaryAgent,
		IntentClassifier:    intentClassifier,
		UserRouter:          userRouter,
		TraceLogger:         traceLogger,
		CostTracker:         costTracker,
		BudgetTracker:       budgetTracker,
		MemoryMaintenance:   nil,
		ToolRegistry:        toolRegistry,
		AgentInitializer:    agentInitializer,
		MCPHostManager:      mcpProvider,
		OrchestrationExec:   orchestrationExecutor,
		MemoryStore:         memoryStore,
		AppManager:          appManager,
		AuthDB:              authDB,
	})

	if err := server.Start(); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}

	slog.Info("server ready",
		"url", "http://localhost:"+cfg.Port,
		"api", "http://localhost:"+cfg.Port+"/v1/chat/completions",
		"health", "http://localhost:"+cfg.Port+"/health")

	// ─── Graceful Shutdown ───────────────────────────────────────────
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("received shutdown signal")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop MCP host if initialized
	if mcpHostManager != nil {
		slog.Info("stopping MCP host")
		if err := mcpHostManager.Stop(ctx); err != nil {
			slog.Warn("MCP host shutdown error", "error", err)
		}
	}

	// Stop server
	if err := server.Stop(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	// Close auth database
	if authDB != nil {
		if err := authDB.Close(); err != nil {
			slog.Warn("auth database close error", "error", err)
		}
	}

	// Shutdown OTEL tracer provider
	if tracerProvider != nil {
		slog.Info("shutting down OTEL tracer provider")
		if err := tracerProvider.Shutdown(ctx); err != nil {
			slog.Warn("OTEL shutdown error", "error", err)
		}
	}

	slog.Info("Agentic Gateway shut down cleanly")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
