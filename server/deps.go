package server

import (
	"database/sql"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/apps"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/maintenance"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
)

// ServerDeps holds all injectable dependencies for the Server.
// This replaces the 18-parameter New() constructor with a single struct.
type ServerDeps struct {
	Config              *ServerConfig
	PluginManager       *provider.PluginManager
	OrchestrationEngine *orchestrationpkg.Engine
	Planner             orchestrationpkg.PlannerInterface
	MemoryManager       *memory.MemoryManager
	GardenManager       *memory.GardenManager
	PrimaryAgent        *agent.PrimaryAgent
	IntentClassifier    *agent.IntentClassifier
	UserRouter          *services.UserRouter
	TraceLogger         *trace.TraceLogger
	CostTracker         *services.CostTracker
	BudgetTracker       *services.BudgetTracker
	MemoryMaintenance   *maintenance.MemoryMaintenance
	ToolRegistry        gateway.ToolRegistry
	AgentInitializer    gateway.AgentInitializer
	MCPHostManager      MCPStatusProvider
	OrchestrationExec   gateway.OrchestrationExecutor
	MemoryStore         gateway.MemoryStore
	AppManager          *apps.AppManager
	AuthDB              *sql.DB
}
