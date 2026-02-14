package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/mcp"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/config"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"

	srv "github.com/TresPies-source/AgenticGatewayByDojoGenesis/server"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("[Main] Agentic Gateway by Dojo Genesis v%s starting...", srv.Version)

	// ─── Load Configuration ──────────────────────────────────────────
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Printf("[Main] WARNING: Config validation: %v", err)
	}
	log.Printf("[Main] Configuration loaded (port: %s, env: %s)", cfg.Port, cfg.Environment)

	// ─── Initialize OTEL (if enabled) ────────────────────────────────
	var tracerProvider *sdktrace.TracerProvider
	if cfg.OTEL.Enabled {
		log.Printf("[Main] OTEL enabled, initializing exporter (endpoint: %s)", cfg.OTEL.Endpoint)

		exporter, err := trace.NewOTELExporter(cfg.OTEL.Endpoint)
		if err != nil {
			log.Printf("[Main] WARNING: OTEL exporter initialization failed: %v", err)
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
			log.Printf("[Main] OTEL tracer provider initialized (service: %s, sampling: %.2f)",
				cfg.OTEL.ServiceName, cfg.OTEL.SamplingRate)
		}
	} else {
		log.Printf("[Main] OTEL disabled")
	}

	// ─── Initialize Provider Plugin Manager ──────────────────────────
	pluginManager := provider.NewPluginManager(cfg.PluginDir)
	log.Printf("[Main] Plugin manager initialized (dir: %s)", cfg.PluginDir)

	// ─── Initialize Tool Registry ────────────────────────────────────
	allTools := tools.GetAllTools()
	toolRegistry := tools.NewContextAwareRegistry()
	log.Printf("[Main] Tool registry initialized (%d built-in tools)", len(allTools))

	// ─── Initialize MCP Host (if configured) ─────────────────────────
	var mcpHostManager *mcp.MCPHostManager
	mcpConfigPath := getEnv("MCP_CONFIG_PATH", "config/mcp_servers.yaml")
	if _, err := os.Stat(mcpConfigPath); err == nil {
		log.Printf("[Main] Loading MCP configuration from %s", mcpConfigPath)

		mcpHostCfg, err := mcp.LoadConfig(mcpConfigPath)
		if err != nil {
			log.Printf("[Main] WARNING: Failed to load MCP config: %v", err)
		} else {
			mcpHostManager, err = mcp.NewMCPHostManager(&mcpHostCfg.MCP, toolRegistry)
			if err != nil {
				log.Printf("[Main] WARNING: Failed to create MCP host manager: %v", err)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := mcpHostManager.Start(ctx); err != nil {
					log.Printf("[Main] WARNING: Failed to start MCP host: %v", err)
					mcpHostManager = nil
				} else {
					mcpStatus := mcpHostManager.Status()
					totalMCPTools := 0
					for _, status := range mcpStatus {
						totalMCPTools += status.ToolCount
					}
					log.Printf("[Main] MCP host started (%d servers, %d tools)", len(mcpStatus), totalMCPTools)
				}
				cancel()
			}
		}
	} else {
		log.Printf("[Main] MCP configuration not found at %s, skipping MCP initialization", mcpConfigPath)
	}

	// ─── Initialize Disposition / Agent Initializer ──────────────────
	dispositionCacheTTL := 5 * time.Minute
	agentInitializer := disposition.NewAgentInitializer(dispositionCacheTTL)
	log.Printf("[Main] Agent initializer created (cache TTL: %v)", dispositionCacheTTL)

	// ─── Initialize Memory ───────────────────────────────────────────
	dbPath := getEnv("MEMORY_DB_PATH", "dojo_memory.db")
	memoryManager, err := memory.NewMemoryManager(dbPath)
	if err != nil {
		log.Printf("[Main] WARNING: Memory manager initialization failed: %v", err)
		memoryManager = nil
	} else {
		log.Printf("[Main] Memory manager initialized (db: %s)", dbPath)
	}

	var gardenManager *memory.GardenManager
	if memoryManager != nil {
		gardenManager, err = memory.NewGardenManager(memoryManager, nil)
		if err != nil {
			log.Printf("[Main] WARNING: Garden manager initialization failed: %v", err)
		} else {
			log.Printf("[Main] Garden manager initialized")
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
	log.Printf("[Main] Services initialized (budget: query=%d, session=%d, monthly=%d)",
		cfg.Budget.QueryLimit, cfg.Budget.SessionLimit, cfg.Budget.MonthlyLimit)

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
	log.Printf("[Main] Agent initialized (default: %s, guest: %s, auth: %s)",
		cfg.Routing.DefaultProvider, cfg.Routing.GuestProvider, cfg.Routing.AuthenticatedProvider)

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
	log.Printf("[Main] Orchestration engine initialized (provider: %s)", providerName)

	// ─── Initialize Gateway Interface Implementations ────────────────
	// Wrap existing components to implement gateway interfaces
	var orchestrationExecutor gateway.OrchestrationExecutor
	orchestrationExecutor = orchestration.NewGatewayOrchestrationExecutor(orchestrationEngine, planner)

	var memoryStore gateway.MemoryStore
	if memoryManager != nil {
		memoryStore = memory.NewGatewayMemoryStore(memoryManager)
		log.Printf("[Main] Gateway memory store initialized")
	}

	// ─── Create and Start Server ─────────────────────────────────────
	server := srv.NewFromConfig(
		cfg,
		pluginManager,
		orchestrationEngine,
		planner,
		memoryManager,
		gardenManager,
		primaryAgent,
		intentClassifier,
		userRouter,
		traceLogger,
		costTracker,
		budgetTracker,
		nil,                   // memory maintenance (optional)
		toolRegistry,          // Phase 2: Gateway tool registry
		agentInitializer,      // Phase 2: Agent disposition initializer
		mcpHostManager,        // Phase 2: MCP host manager (optional)
		orchestrationExecutor, // Phase 3: Orchestration executor (gateway interface)
		memoryStore,           // Phase 3: Memory store (gateway interface)
	)

	if err := server.Start(); err != nil {
		log.Fatalf("[Main] Failed to start server: %v", err)
	}

	log.Printf("[Main] Server ready at http://localhost:%s", cfg.Port)
	log.Printf("[Main] OpenAI-compatible API: http://localhost:%s/v1/chat/completions", cfg.Port)
	log.Printf("[Main] Health check: http://localhost:%s/health", cfg.Port)

	// ─── Graceful Shutdown ───────────────────────────────────────────
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\n[Main] Received shutdown signal")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop MCP host if initialized
	if mcpHostManager != nil {
		log.Println("[Main] Stopping MCP host...")
		if err := mcpHostManager.Stop(ctx); err != nil {
			log.Printf("[Main] WARNING: MCP host shutdown error: %v", err)
		}
	}

	// Stop server
	if err := server.Stop(ctx); err != nil {
		log.Fatalf("[Main] Shutdown error: %v", err)
	}

	// Shutdown OTEL tracer provider
	if tracerProvider != nil {
		log.Println("[Main] Shutting down OTEL tracer provider...")
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Printf("[Main] WARNING: OTEL shutdown error: %v", err)
		}
	}

	log.Printf("[Main] Agentic Gateway shut down cleanly")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
