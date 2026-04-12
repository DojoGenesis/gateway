package server

import (
	"database/sql"

	"github.com/DojoGenesis/gateway/apps"
	"github.com/DojoGenesis/gateway/memory"
	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/DojoGenesis/gateway/pkg/gateway"
	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/runtime/cas"
	"github.com/DojoGenesis/gateway/server/agent"
	"github.com/DojoGenesis/gateway/server/maintenance"
	"github.com/DojoGenesis/gateway/server/services"
	"github.com/DojoGenesis/gateway/server/trace"
	"github.com/DojoGenesis/gateway/specialist"
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

	// SpecialistRouter dispatches requests to specialist agents based on
	// intent classification. If nil, specialist dispatch is disabled.
	SpecialistRouter *specialist.Router

	// WorkflowCAS is the CAS store used for workflow definition lookup and execution.
	// If nil, the /api/workflows/* and execution endpoints are disabled.
	WorkflowCAS cas.Store

	// D1Syncer is the optional local-to-D1 CAS sync loop (Era 4).
	// If nil, GET /api/cas/status reports sync as disabled.
	D1Syncer *cas.D1Syncer
}
