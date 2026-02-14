package agent

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/services"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// registerTestTools ensures test tools are registered for orchestration tests.
// Uses idempotent registration to avoid "tool already registered" errors.
func registerTestTools() {
	// Try to register test_tool, ignore error if already registered
	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool for orchestration testing",
		Parameters:  map[string]interface{}{},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"output": "test_tool executed successfully"}, nil
		},
	})

	// Try to register test_tool2, ignore error if already registered
	tools.RegisterTool(&tools.ToolDefinition{
		Name:        "test_tool2",
		Description: "A second test tool for orchestration testing",
		Parameters:  map[string]interface{}{},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"output": "test_tool2 executed successfully"}, nil
		},
	})
}

func init() {
	registerTestTools()
}

// testToolInvoker delegates tool invocations to the global tools registry.
type testToolInvoker struct{}

func (t *testToolInvoker) InvokeTool(ctx context.Context, toolName string, parameters map[string]interface{}) (map[string]interface{}, error) {
	return tools.InvokeTool(ctx, toolName, parameters)
}

type mockPlanner struct {
	generatePlanFunc   func(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error)
	regeneratePlanFunc func(ctx context.Context, task *orchestrationpkg.Task, failedPlan *orchestrationpkg.Plan, errorContext string) (*orchestrationpkg.Plan, error)
}

func (m *mockPlanner) GeneratePlan(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
	if m.generatePlanFunc != nil {
		return m.generatePlanFunc(ctx, task)
	}
	plan := orchestrationpkg.NewPlan(task.ID)
	plan.Nodes = []*orchestrationpkg.PlanNode{
		{
			ID:           uuid.New().String(),
			ToolName:     "test_tool",
			Parameters:   map[string]interface{}{"param1": "value1"},
			Dependencies: []string{},
			State:        orchestrationpkg.NodeStatePending,
		},
	}
	return plan, nil
}

func (m *mockPlanner) RegeneratePlan(ctx context.Context, task *orchestrationpkg.Task, failedPlan *orchestrationpkg.Plan, errorContext string) (*orchestrationpkg.Plan, error) {
	if m.regeneratePlanFunc != nil {
		return m.regeneratePlanFunc(ctx, task, failedPlan, errorContext)
	}
	return nil, errors.New("regenerate not implemented")
}

type mockEngine struct {
	executeFunc func(ctx context.Context, plan *orchestrationpkg.Plan, task *orchestrationpkg.Task, userID string) error
}

func (m *mockEngine) Execute(ctx context.Context, plan *orchestrationpkg.Plan, task *orchestrationpkg.Task, userID string) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, plan, task, userID)
	}
	for _, node := range plan.Nodes {
		node.State = orchestrationpkg.NodeStateSuccess
		node.Result = map[string]interface{}{"output": "success"}
		now := time.Now()
		node.StartTime = &now
		node.EndTime = &now
	}
	return nil
}

func TestOrchestrationEnablement(t *testing.T) {
	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)

	if agent.useOrchestration {
		t.Error("Orchestration should be disabled by default")
	}

	agent.EnableOrchestration(true)

	if !agent.useOrchestration {
		t.Error("Orchestration should be enabled after calling EnableOrchestration(true)")
	}

	agent.EnableOrchestration(false)

	if agent.useOrchestration {
		t.Error("Orchestration should be disabled after calling EnableOrchestration(false)")
	}
}

func TestSetOrchestrationComponents(t *testing.T) {
	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)

	if agent.orchestrationEngine != nil {
		t.Error("Orchestration engine should be nil by default")
	}

	if agent.orchestrationPlanner != nil {
		t.Error("Orchestration planner should be nil by default")
	}

	budgetTracker := services.NewBudgetTracker(1000, 5000, 10000)

	mockPlanner := &mockPlanner{}

	engine := orchestrationpkg.NewEngine(
		orchestrationpkg.DefaultEngineConfig(),
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	if agent.orchestrationEngine == nil {
		t.Error("Orchestration engine should be set")
	}

	if agent.orchestrationPlanner == nil {
		t.Error("Orchestration planner should be set")
	}
}

func TestHandleQueryWithOrchestration_ComponentsNotInitialized(t *testing.T) {
	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	ctx := context.Background()
	req := QueryRequest{
		Query:  "Test query",
		UserID: "user1",
	}

	_, err := agent.HandleQueryWithOrchestration(ctx, req)

	if err == nil {
		t.Error("Expected error when orchestration components not initialized")
	}

	if err.Error() != "orchestration components not initialized" {
		t.Errorf("Expected 'orchestration components not initialized' error, got: %v", err)
	}
}

func TestHandleQueryWithOrchestration_Success(t *testing.T) {
	registerTestTools() // Ensure test tools are available

	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	budgetTracker := services.NewBudgetTracker(10000, 50000, 100000)

	mockPlanner := &mockPlanner{
		generatePlanFunc: func(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
			plan := orchestrationpkg.NewPlan(task.ID)
			plan.Metadata["reasoning"] = "Test reasoning"
			plan.Nodes = []*orchestrationpkg.PlanNode{
				{
					ID:           "node1",
					ToolName:     "test_tool",
					Parameters:   map[string]interface{}{"param1": "value1"},
					Dependencies: []string{},
					State:        orchestrationpkg.NodeStatePending,
				},
				{
					ID:           "node2",
					ToolName:     "test_tool2",
					Parameters:   map[string]interface{}{"param2": "value2"},
					Dependencies: []string{"node1"},
					State:        orchestrationpkg.NodeStatePending,
				},
			}
			return plan, nil
		},
	}

	config := orchestrationpkg.DefaultEngineConfig()
	config.EnableAutoReplanning = false

	engine := orchestrationpkg.NewEngine(
		config,
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	ctx := context.Background()
	req := QueryRequest{
		Query:  "Test complex query",
		UserID: "user1",
	}

	response, err := agent.HandleQueryWithOrchestration(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Provider != "orchestration" {
		t.Errorf("Expected provider 'orchestration', got: %s", response.Provider)
	}

	if response.Model != "orchestration-engine" {
		t.Errorf("Expected model 'orchestration-engine', got: %s", response.Model)
	}

	if len(response.ToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls, got: %d", len(response.ToolCalls))
	}

	if len(response.ToolResults) != 2 {
		t.Errorf("Expected 2 tool results, got: %d", len(response.ToolResults))
	}

	if response.Content == "" {
		t.Error("Expected non-empty response content")
	}
}

func TestHandleQueryWithOrchestration_PlanGenerationFailure(t *testing.T) {
	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	budgetTracker := services.NewBudgetTracker(10000, 50000, 100000)

	mockPlanner := &mockPlanner{
		generatePlanFunc: func(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
			return nil, errors.New("plan generation failed")
		},
	}

	engine := orchestrationpkg.NewEngine(
		orchestrationpkg.DefaultEngineConfig(),
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	ctx := context.Background()
	req := QueryRequest{
		Query:  "Test query",
		UserID: "user1",
	}

	_, err := agent.HandleQueryWithOrchestration(ctx, req)

	if err == nil {
		t.Fatal("Expected error when plan generation fails")
	}

	if !errors.Is(err, errors.New("failed to generate plan: plan generation failed")) && err.Error() != "failed to generate plan: plan generation failed" {
		t.Errorf("Expected plan generation error, got: %v", err)
	}
}

func TestHandleQueryWithOrchestration_ExecutionFailure(t *testing.T) {
	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	budgetTracker := services.NewBudgetTracker(10000, 50000, 100000)

	mockPlanner := &mockPlanner{
		generatePlanFunc: func(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
			plan := orchestrationpkg.NewPlan(task.ID)
			plan.Nodes = []*orchestrationpkg.PlanNode{
				{
					ID:           "node1",
					ToolName:     "nonexistent_tool",
					Parameters:   map[string]interface{}{"param1": "value1"},
					Dependencies: []string{},
					State:        orchestrationpkg.NodeStatePending,
				},
			}
			return plan, nil
		},
	}

	config := orchestrationpkg.DefaultEngineConfig()
	config.EnableAutoReplanning = false

	engine := orchestrationpkg.NewEngine(
		config,
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	ctx := context.Background()
	req := QueryRequest{
		Query:  "Test query",
		UserID: "user1",
	}

	_, err := agent.HandleQueryWithOrchestration(ctx, req)

	if err == nil {
		t.Fatal("Expected error when execution fails")
	}
}

func TestBuildResponseFromPlan(t *testing.T) {
	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)

	task := orchestrationpkg.NewTask("user1", "Test task description")
	plan := orchestrationpkg.NewPlan(task.ID)
	plan.Metadata["reasoning"] = "This is the reasoning"

	now := time.Now()
	end := now.Add(500 * time.Millisecond)

	plan.Nodes = []*orchestrationpkg.PlanNode{
		{
			ID:         "node1",
			ToolName:   "tool1",
			Parameters: map[string]interface{}{"key": "value"},
			State:      orchestrationpkg.NodeStateSuccess,
			Result:     map[string]interface{}{"output": "result1"},
			StartTime:  &now,
			EndTime:    &end,
		},
		{
			ID:         "node2",
			ToolName:   "tool2",
			Parameters: map[string]interface{}{"key2": "value2"},
			State:      orchestrationpkg.NodeStateFailed,
			Error:      "Tool error",
			StartTime:  &now,
			EndTime:    &end,
		},
	}

	response := agent.buildResponseFromPlan(plan, task)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.ID != plan.ID {
		t.Errorf("Expected response ID to match plan ID, got: %s", response.ID)
	}

	if response.Provider != "orchestration" {
		t.Errorf("Expected provider 'orchestration', got: %s", response.Provider)
	}

	if response.Model != "orchestration-engine" {
		t.Errorf("Expected model 'orchestration-engine', got: %s", response.Model)
	}

	if len(response.ToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls, got: %d", len(response.ToolCalls))
	}

	if len(response.ToolResults) != 2 {
		t.Errorf("Expected 2 tool results, got: %d", len(response.ToolResults))
	}

	if response.Content == "" {
		t.Error("Expected non-empty content")
	}

	if !contains(response.Content, "Test task description") {
		t.Error("Expected content to contain task description")
	}

	if !contains(response.Content, "This is the reasoning") {
		t.Error("Expected content to contain reasoning")
	}

	if !contains(response.Content, "tool1") {
		t.Error("Expected content to contain tool1")
	}

	if !contains(response.Content, "tool2") {
		t.Error("Expected content to contain tool2")
	}

	if !contains(response.Content, "Tool error") {
		t.Error("Expected content to contain error message")
	}
}

func TestHandleQueryWithOrchestration_WithTraceLogger(t *testing.T) {
	registerTestTools() // Ensure test tools are available

	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	traceStorage, err := trace.NewTraceStorage(db)
	if err != nil {
		t.Fatalf("Failed to create trace storage: %v", err)
	}

	traceLogger := trace.NewTraceLoggerWithoutEvents(traceStorage)
	agent.SetTraceLogger(traceLogger)

	budgetTracker := services.NewBudgetTracker(10000, 50000, 100000)

	mockPlanner := &mockPlanner{}

	config := orchestrationpkg.DefaultEngineConfig()
	config.EnableAutoReplanning = false

	engine := orchestrationpkg.NewEngine(
		config,
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	ctx := context.Background()
	req := QueryRequest{
		Query:  "Test query with trace",
		UserID: "user1",
	}

	response, err := agent.HandleQueryWithOrchestration(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}
}

func TestHandleQueryWithOrchestration_WithMemory(t *testing.T) {
	registerTestTools() // Ensure test tools are available

	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	memManager, err := memory.NewMemoryManager(":memory:")
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}
	agent.SetMemoryManager(memManager)

	budgetTracker := services.NewBudgetTracker(10000, 50000, 100000)

	mockPlanner := &mockPlanner{}

	config := orchestrationpkg.DefaultEngineConfig()
	config.EnableAutoReplanning = false

	engine := orchestrationpkg.NewEngine(
		config,
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	ctx := context.Background()
	req := QueryRequest{
		Query:     "Test query with memory",
		UserID:    "user1",
		UseMemory: true,
	}

	response, err := agent.HandleQueryWithOrchestration(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	ctx2 := context.Background()
	memories, err := memManager.SearchByType(ctx2, "conversation:user1", 10)
	if err != nil {
		t.Fatalf("Expected no error retrieving memories, got: %v", err)
	}

	if len(memories) != 1 {
		t.Errorf("Expected 1 memory entry, got: %d", len(memories))
	}
}

// Edge case tests

func TestHandleQueryWithOrchestration_EmptyPlan(t *testing.T) {
	registerTestTools()

	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	budgetTracker := services.NewBudgetTracker(10000, 50000, 100000)

	mockPlanner := &mockPlanner{
		generatePlanFunc: func(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
			plan := orchestrationpkg.NewPlan(task.ID)
			// Empty plan with no nodes
			plan.Nodes = []*orchestrationpkg.PlanNode{}
			return plan, nil
		},
	}

	config := orchestrationpkg.DefaultEngineConfig()
	config.EnableAutoReplanning = false

	engine := orchestrationpkg.NewEngine(
		config,
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	ctx := context.Background()
	req := QueryRequest{
		Query:  "Test query with empty plan",
		UserID: "user1",
	}

	_, err := agent.HandleQueryWithOrchestration(ctx, req)

	// Empty plan should result in an error from the engine
	// (no executable nodes means the plan is invalid or incomplete)
	if err == nil {
		t.Fatal("Expected error with empty plan, got nil")
	}

	expectedErr := "orchestration execution failed"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErr, err)
	}
}

func TestHandleQueryWithOrchestration_AllNodesFailed(t *testing.T) {
	registerTestTools()

	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	budgetTracker := services.NewBudgetTracker(10000, 50000, 100000)

	mockPlanner := &mockPlanner{
		generatePlanFunc: func(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
			plan := orchestrationpkg.NewPlan(task.ID)
			plan.Nodes = []*orchestrationpkg.PlanNode{
				{
					ID:           "node1",
					ToolName:     "nonexistent_tool1",
					Parameters:   map[string]interface{}{},
					Dependencies: []string{},
					State:        orchestrationpkg.NodeStatePending,
				},
				{
					ID:           "node2",
					ToolName:     "nonexistent_tool2",
					Parameters:   map[string]interface{}{},
					Dependencies: []string{},
					State:        orchestrationpkg.NodeStatePending,
				},
			}
			return plan, nil
		},
	}

	config := orchestrationpkg.DefaultEngineConfig()
	config.EnableAutoReplanning = false

	engine := orchestrationpkg.NewEngine(
		config,
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	ctx := context.Background()
	req := QueryRequest{
		Query:  "Test query with all nodes failing",
		UserID: "user1",
	}

	_, err := agent.HandleQueryWithOrchestration(ctx, req)

	// Should return error when all nodes fail
	if err == nil {
		t.Fatal("Expected error when all nodes fail")
	}
}

func TestHandleQueryWithOrchestration_ContextCancellation(t *testing.T) {
	registerTestTools()

	pm := &MockPluginManager{
		providers: make(map[string]providerpkg.ModelProvider),
	}

	agent := NewPrimaryAgent(pm)
	agent.EnableOrchestration(true)

	budgetTracker := services.NewBudgetTracker(10000, 50000, 100000)

	mockPlanner := &mockPlanner{
		generatePlanFunc: func(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
			plan := orchestrationpkg.NewPlan(task.ID)
			plan.Nodes = []*orchestrationpkg.PlanNode{
				{
					ID:           "node1",
					ToolName:     "test_tool",
					Parameters:   map[string]interface{}{},
					Dependencies: []string{},
					State:        orchestrationpkg.NodeStatePending,
				},
			}
			return plan, nil
		},
	}

	config := orchestrationpkg.DefaultEngineConfig()
	config.EnableAutoReplanning = false

	engine := orchestrationpkg.NewEngine(
		config,
		mockPlanner,
		&testToolInvoker{},
		nil,
		nil,
		budgetTracker,
	)

	agent.SetOrchestrationEngine(engine)
	agent.SetOrchestrationPlanner(mockPlanner)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := QueryRequest{
		Query:  "Test query with cancelled context",
		UserID: "user1",
	}

	_, err := agent.HandleQueryWithOrchestration(ctx, req)

	// Should fail due to cancelled context
	if err == nil {
		t.Fatal("Expected error due to cancelled context")
	}
}
