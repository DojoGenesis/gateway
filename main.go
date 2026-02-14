package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"

	srv "github.com/TresPies-source/AgenticGatewayByDojoGenesis/server"

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
	for _, r := range providerResults {
		if r.Available {
			loadedCount++
		}
	}
	slog.Info("provider registration complete", "loaded", loadedCount, "total_checked", len(providerResults))

	// ─── Initialize Tool Registry ────────────────────────────────────
	allTools := tools.GetAllTools()
	toolRegistry := tools.NewContextAwareRegistry()
	slog.Info("tool registry initialized", "builtin_tools", len(allTools))

	// ─── Initialize MCP Host (if configured) ─────────────────────────
	var mcpHostManager *mcp.MCPHostManager
	mcpConfigPath := getEnv("MCP_CONFIG_PATH", "config/mcp_servers.yaml")
	if _, err := os.Stat(mcpConfigPath); err == nil {
		slog.Info("loading MCP configuration", "path", mcpConfigPath)

		mcpHostCfg, err := mcp.LoadConfig(mcpConfigPath)
		if err != nil {
			slog.Warn("failed to load MCP config", "error", err)
		} else {
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

	// ─── Initialize Disposition / Agent Initializer ──────────────────
	dispositionCacheTTL := 5 * time.Minute
	agentInitializer := pkgdisposition.NewAgentInitializer(dispositionCacheTTL)
	slog.Info("agent initializer created", "cache_ttl", dispositionCacheTTL)

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

	// Event emitter is nil for now (handled by server)
	var eventEmitter orchestrationpkg.EventEmitterInterface = nil

	// Create the standalone orchestration engine with all adapters
	orchestrationEngine := orchestrationpkg.NewEngine(
		orchestrationpkg.DefaultEngineConfig(),
		planner,
		toolInvoker,
		traceAdapter,
		eventEmitter,
		budgetAdapter,
	)
	slog.Info("orchestration engine initialized", "provider", providerName)

	// ─── Initialize Gateway Interface Implementations ────────────────
	// Wrap existing components to implement gateway interfaces
	orchestrationExecutor := orchestration.NewGatewayOrchestrationExecutor(orchestrationEngine, planner)

	var memoryStore gateway.MemoryStore
	if memoryManager != nil {
		memoryStore = memory.NewGatewayMemoryStore(memoryManager)
		slog.Info("gateway memory store initialized")
	}

	// ─── Create and Start Server ─────────────────────────────────────
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
		MCPHostManager:      mcpHostManager,
		OrchestrationExec:   orchestrationExecutor,
		MemoryStore:         memoryStore,
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
