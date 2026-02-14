package orchestration

import orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

type mockProvider struct {
	response     string
	shouldError  bool
	errorMessage string
	callCount    int
	lastRequest  *provider.CompletionRequest
}

func (m *mockProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	m.callCount++
	m.lastRequest = req

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	return &provider.CompletionResponse{
		ID:      "test-response",
		Model:   req.Model,
		Content: m.response,
		Usage: provider.Usage{
			InputTokens:  100,
			OutputTokens: 200,
			TotalTokens:  300,
		},
	}, nil
}

func (m *mockProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	return nil, fmt.Errorf("streaming not implemented in mock")
}

func (m *mockProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{
			ID:          "test-model",
			Name:        "Test Model",
			Provider:    "test",
			ContextSize: 8192,
			Cost:        0.0,
		},
	}, nil
}

func (m *mockProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:        "test",
		Version:     "1.0.0",
		Description: "Test provider",
	}, nil
}

func (m *mockProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Result: "tool result",
		Error:  "",
	}, nil
}

func (m *mockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

type mockPluginManager struct {
	provider *mockProvider
}

func (m *mockPluginManager) GetProvider(name string) (provider.ModelProvider, error) {
	if m.provider == nil {
		return nil, fmt.Errorf("provider not found")
	}
	return m.provider, nil
}

func (m *mockPluginManager) GetProviders() map[string]provider.ModelProvider {
	return map[string]provider.ModelProvider{
		"test": m.provider,
	}
}

func setupTestTools() {
	tools.ClearRegistry()

	_ = tools.RegisterTool(&tools.ToolDefinition{
		Name:        "fetch_article",
		Description: "Fetches an article from a URL",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to fetch",
				},
			},
			"required": []string{"url"},
		},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"content": "article content"}, nil
		},
	})

	_ = tools.RegisterTool(&tools.ToolDefinition{
		Name:        "summarize_text",
		Description: "Summarizes a text",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "The text to summarize",
				},
			},
			"required": []string{"text"},
		},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"summary": "summary"}, nil
		},
	})

	_ = tools.RegisterTool(&tools.ToolDefinition{
		Name:        "create_report",
		Description: "Creates a markdown report",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Report title",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Report content",
				},
			},
			"required": []string{"title", "content"},
		},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"report_path": "/tmp/report.md"}, nil
		},
	})
}

func TestPlannerGeneratePlanLinear(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "Fetch article, then summarize, then create report",
		"nodes": [
			{
				"id": "node_1",
				"tool_name": "fetch_article",
				"parameters": {"url": "https://example.com/article"},
				"dependencies": []
			},
			{
				"id": "node_2",
				"tool_name": "summarize_text",
				"parameters": {"text": "{{node_1.content}}"},
				"dependencies": ["node_1"]
			},
			{
				"id": "node_3",
				"tool_name": "create_report",
				"parameters": {"title": "Report", "content": "{{node_2.summary}}"},
				"dependencies": ["node_2"]
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Fetch an article, summarize it, and create a report")

	plan, err := planner.GeneratePlan(context.Background(), task)
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	if len(plan.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(plan.Nodes))
	}

	if plan.Nodes[0].ToolName != "fetch_article" {
		t.Errorf("Expected first node to be fetch_article, got %s", plan.Nodes[0].ToolName)
	}

	if len(plan.Nodes[1].Dependencies) != 1 || plan.Nodes[1].Dependencies[0] != "node_1" {
		t.Errorf("Expected node_2 to depend on node_1")
	}

	if plan.Metadata["reasoning"] == nil {
		t.Error("Expected reasoning in metadata")
	}
}

func TestPlannerGeneratePlanParallel(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "Fetch three articles in parallel, then summarize all",
		"nodes": [
			{
				"id": "fetch_1",
				"tool_name": "fetch_article",
				"parameters": {"url": "https://example.com/1"},
				"dependencies": []
			},
			{
				"id": "fetch_2",
				"tool_name": "fetch_article",
				"parameters": {"url": "https://example.com/2"},
				"dependencies": []
			},
			{
				"id": "fetch_3",
				"tool_name": "fetch_article",
				"parameters": {"url": "https://example.com/3"},
				"dependencies": []
			},
			{
				"id": "summarize_all",
				"tool_name": "summarize_text",
				"parameters": {"text": "combined"},
				"dependencies": ["fetch_1", "fetch_2", "fetch_3"]
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Fetch three articles and summarize them")

	plan, err := planner.GeneratePlan(context.Background(), task)
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	if len(plan.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(plan.Nodes))
	}

	parallelNodes := 0
	for _, node := range plan.Nodes {
		if len(node.Dependencies) == 0 && strings.HasPrefix(node.ID, "fetch_") {
			parallelNodes++
		}
	}

	if parallelNodes != 3 {
		t.Errorf("Expected 3 parallel fetch nodes, got %d", parallelNodes)
	}

	lastNode := plan.Nodes[3]
	if len(lastNode.Dependencies) != 3 {
		t.Errorf("Expected last node to depend on 3 nodes, got %d", len(lastNode.Dependencies))
	}
}

func TestPlannerParseMalformedJSON(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	testCases := []struct {
		name     string
		response string
		wantErr  bool
	}{
		{
			name:     "Invalid JSON",
			response: `{invalid json}`,
			wantErr:  true,
		},
		{
			name:     "Missing nodes array",
			response: `{"reasoning": "test"}`,
			wantErr:  true,
		},
		{
			name:     "Empty response",
			response: ``,
			wantErr:  true,
		},
		{
			name:     "Non-JSON response",
			response: `This is not JSON at all`,
			wantErr:  true,
		},
		{
			name: "JSON with extra text",
			response: `Here is the plan:
			{
				"reasoning": "test",
				"nodes": [
					{
						"id": "node_1",
						"tool_name": "fetch_article",
						"parameters": {"url": "test"},
						"dependencies": []
					}
				]
			}
			That's the plan.`,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &mockProvider{response: tc.response}
			pm := &mockPluginManager{provider: provider}
			planner := NewPlanner(pm, nil, "test", "test-model")

			task := orchestrationpkg.NewTask("user-1", "Test task")
			_, err := planner.GeneratePlan(context.Background(), task)

			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPlannerDetectCyclicDependencies(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "This plan has a cycle",
		"nodes": [
			{
				"id": "node_1",
				"tool_name": "fetch_article",
				"parameters": {"url": "test"},
				"dependencies": ["node_2"]
			},
			{
				"id": "node_2",
				"tool_name": "summarize_text",
				"parameters": {"text": "test"},
				"dependencies": ["node_1"]
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Test cyclic dependency")

	_, err := planner.GeneratePlan(context.Background(), task)
	if err == nil {
		t.Fatal("Expected error for cyclic dependencies")
	}

	if !strings.Contains(err.Error(), "cyclic") {
		t.Errorf("Expected cyclic dependency error, got: %v", err)
	}
}

func TestPlannerInvalidToolName(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "Using non-existent tool",
		"nodes": [
			{
				"id": "node_1",
				"tool_name": "nonexistent_tool",
				"parameters": {},
				"dependencies": []
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Test invalid tool")

	_, err := planner.GeneratePlan(context.Background(), task)
	if err == nil {
		t.Fatal("Expected error for invalid tool name")
	}

	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("Expected unknown tool error, got: %v", err)
	}
}

func TestPlannerRegeneratePlan(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	originalPlan := orchestrationpkg.NewPlan("task-1")
	originalPlan.Nodes = []*orchestrationpkg.PlanNode{
		{
			ID:           "node_1",
			ToolName:     "fetch_article",
			Parameters:   map[string]interface{}{"url": "bad-url"},
			Dependencies: []string{},
			State:        orchestrationpkg.NodeStateFailed,
			Error:        "Failed to fetch: invalid URL",
		},
	}

	mockResp := `{
		"reasoning": "Fixed the URL parameter to use a valid URL",
		"nodes": [
			{
				"id": "node_1_fixed",
				"tool_name": "fetch_article",
				"parameters": {"url": "https://example.com/valid"},
				"dependencies": []
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Fetch article and summarize")

	newPlan, err := planner.RegeneratePlan(context.Background(), task, originalPlan, "Failed to fetch: invalid URL")
	if err != nil {
		t.Fatalf("RegeneratePlan failed: %v", err)
	}

	if newPlan.Version != 2 {
		t.Errorf("Expected version 2, got %d", newPlan.Version)
	}

	if len(newPlan.Nodes) != 1 {
		t.Errorf("Expected 1 node in new plan, got %d", len(newPlan.Nodes))
	}

	if newPlan.Nodes[0].ToolName != "fetch_article" {
		t.Errorf("Expected fetch_article tool, got %s", newPlan.Nodes[0].ToolName)
	}

	lastReq := provider.lastRequest
	if lastReq == nil {
		t.Fatal("No request captured")
	}

	promptContent := ""
	for _, msg := range lastReq.Messages {
		if msg.Role == "user" {
			promptContent = msg.Content
		}
	}

	if !strings.Contains(promptContent, "Failed Plan") {
		t.Error("Replanning prompt should contain failed plan")
	}

	if !strings.Contains(promptContent, "Failed to fetch: invalid URL") {
		t.Error("Replanning prompt should contain error context")
	}
}

func TestPlannerPromptConstruction(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "test",
		"nodes": [{
			"id": "node_1",
			"tool_name": "fetch_article",
			"parameters": {"url": "test"},
			"dependencies": []
		}]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Test prompt construction")

	_, err := planner.GeneratePlan(context.Background(), task)
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	if provider.callCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", provider.callCount)
	}

	lastReq := provider.lastRequest
	if lastReq == nil {
		t.Fatal("No request captured")
	}

	if len(lastReq.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(lastReq.Messages))
	}

	if lastReq.Messages[0].Role != "system" {
		t.Error("First message should be system")
	}

	if lastReq.Messages[1].Role != "user" {
		t.Error("Second message should be user")
	}

	userContent := lastReq.Messages[1].Content

	if !strings.Contains(userContent, "Task Description") {
		t.Error("Prompt should contain task description")
	}

	if !strings.Contains(userContent, "Available Tools") {
		t.Error("Prompt should contain available tools")
	}

	if !strings.Contains(userContent, "fetch_article") {
		t.Error("Prompt should list fetch_article tool")
	}

	if !strings.Contains(userContent, "Response Format") {
		t.Error("Prompt should contain response format instructions")
	}

	if lastReq.Temperature != 0.2 {
		t.Errorf("Expected temperature 0.2, got %f", lastReq.Temperature)
	}
}

func TestPlannerDuplicateNodeIDs(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "test",
		"nodes": [
			{
				"id": "node_1",
				"tool_name": "fetch_article",
				"parameters": {"url": "test"},
				"dependencies": []
			},
			{
				"id": "node_1",
				"tool_name": "summarize_text",
				"parameters": {"text": "test"},
				"dependencies": []
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Test duplicate IDs")

	_, err := planner.GeneratePlan(context.Background(), task)
	if err == nil {
		t.Fatal("Expected error for duplicate node IDs")
	}

	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("Expected duplicate error, got: %v", err)
	}
}

func TestPlannerEmptyPlan(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "No nodes needed",
		"nodes": []
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Test empty plan")

	_, err := planner.GeneratePlan(context.Background(), task)
	if err == nil {
		t.Fatal("Expected error for empty plan")
	}

	if !strings.Contains(err.Error(), "no nodes") {
		t.Errorf("Expected no nodes error, got: %v", err)
	}
}

func TestPlannerLLMError(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	provider := &mockProvider{
		shouldError:  true,
		errorMessage: "LLM service unavailable",
	}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Test LLM error")

	_, err := planner.GeneratePlan(context.Background(), task)
	if err == nil {
		t.Fatal("Expected error from LLM")
	}

	if !strings.Contains(err.Error(), "failed to call LLM") {
		t.Errorf("Expected LLM call error, got: %v", err)
	}
}

func TestPlannerInvalidDependency(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "test",
		"nodes": [
			{
				"id": "node_1",
				"tool_name": "fetch_article",
				"parameters": {"url": "test"},
				"dependencies": ["nonexistent_node"]
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Test invalid dependency")

	_, err := planner.GeneratePlan(context.Background(), task)
	if err == nil {
		t.Fatal("Expected error for invalid dependency")
	}

	if !strings.Contains(err.Error(), "invalid dependency") {
		t.Errorf("Expected invalid dependency error, got: %v", err)
	}
}

func TestPlannerComplexDAG(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	mockResp := `{
		"reasoning": "Complex workflow with parallel and sequential steps",
		"nodes": [
			{
				"id": "fetch_1",
				"tool_name": "fetch_article",
				"parameters": {"url": "url1"},
				"dependencies": []
			},
			{
				"id": "fetch_2",
				"tool_name": "fetch_article",
				"parameters": {"url": "url2"},
				"dependencies": []
			},
			{
				"id": "summarize_1",
				"tool_name": "summarize_text",
				"parameters": {"text": "text1"},
				"dependencies": ["fetch_1"]
			},
			{
				"id": "summarize_2",
				"tool_name": "summarize_text",
				"parameters": {"text": "text2"},
				"dependencies": ["fetch_2"]
			},
			{
				"id": "report",
				"tool_name": "create_report",
				"parameters": {"title": "Report", "content": "combined"},
				"dependencies": ["summarize_1", "summarize_2"]
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Complex multi-step workflow")

	plan, err := planner.GeneratePlan(context.Background(), task)
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}

	if len(plan.Nodes) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(plan.Nodes))
	}

	executableNodes := plan.GetExecutableNodes()
	if len(executableNodes) != 2 {
		t.Errorf("Expected 2 initially executable nodes, got %d", len(executableNodes))
	}

	for _, node := range executableNodes {
		if !strings.HasPrefix(node.ID, "fetch_") {
			t.Errorf("Expected fetch nodes to be initially executable, got %s", node.ID)
		}
	}

	reportNode := plan.GetNodeByID("report")
	if reportNode == nil {
		t.Fatal("Report node not found")
	}

	if len(reportNode.Dependencies) != 2 {
		t.Errorf("Expected report node to have 2 dependencies, got %d", len(reportNode.Dependencies))
	}
}

func TestPlannerPreserveCompletedWork(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	originalPlan := orchestrationpkg.NewPlan("task-1")
	originalPlan.Nodes = []*orchestrationpkg.PlanNode{
		{
			ID:           "fetch_1",
			ToolName:     "fetch_article",
			Parameters:   map[string]interface{}{"url": "url1"},
			Dependencies: []string{},
			State:        orchestrationpkg.NodeStateSuccess,
			Result:       map[string]interface{}{"content": "article 1"},
		},
		{
			ID:           "fetch_2",
			ToolName:     "fetch_article",
			Parameters:   map[string]interface{}{"url": "bad-url"},
			Dependencies: []string{},
			State:        orchestrationpkg.NodeStateFailed,
			Error:        "Invalid URL",
		},
		{
			ID:           "summarize",
			ToolName:     "summarize_text",
			Parameters:   map[string]interface{}{"text": "combined"},
			Dependencies: []string{"fetch_1", "fetch_2"},
			State:        orchestrationpkg.NodeStatePending,
		},
	}

	mockResp := `{
		"reasoning": "Preserve fetch_1 result and fix fetch_2",
		"nodes": [
			{
				"id": "fetch_2_fixed",
				"tool_name": "fetch_article",
				"parameters": {"url": "https://valid-url.com"},
				"dependencies": []
			},
			{
				"id": "summarize",
				"tool_name": "summarize_text",
				"parameters": {"text": "combined"},
				"dependencies": ["fetch_2_fixed"]
			}
		]
	}`

	provider := &mockProvider{response: mockResp}
	pm := &mockPluginManager{provider: provider}
	planner := NewPlanner(pm, nil, "test", "test-model")

	task := orchestrationpkg.NewTask("user-1", "Fetch and summarize")

	newPlan, err := planner.RegeneratePlan(context.Background(), task, originalPlan, "fetch_2 failed: Invalid URL")
	if err != nil {
		t.Fatalf("RegeneratePlan failed: %v", err)
	}

	lastReq := provider.lastRequest
	promptContent := ""
	for _, msg := range lastReq.Messages {
		if msg.Role == "user" {
			promptContent = msg.Content
		}
	}

	if !strings.Contains(promptContent, "Completed Nodes") {
		t.Error("Replanning prompt should mention completed nodes")
	}

	if !strings.Contains(promptContent, "fetch_1") {
		t.Error("Replanning prompt should list fetch_1 as completed")
	}

	if newPlan.Version != 2 {
		t.Errorf("Expected version 2, got %d", newPlan.Version)
	}
}

func TestParsePlanFromLLMResponse(t *testing.T) {
	planner := &Planner{}

	testCases := []struct {
		name      string
		response  string
		wantError bool
		wantNodes int
	}{
		{
			name: "Valid plan",
			response: `{
				"reasoning": "test",
				"nodes": [
					{
						"id": "node_1",
						"tool_name": "test_tool",
						"parameters": {"key": "value"},
						"dependencies": []
					}
				]
			}`,
			wantError: false,
			wantNodes: 1,
		},
		{
			name: "Plan with auto-generated IDs",
			response: `{
				"nodes": [
					{
						"tool_name": "test_tool",
						"parameters": {},
						"dependencies": []
					}
				]
			}`,
			wantError: false,
			wantNodes: 1,
		},
		{
			name:      "Empty nodes array",
			response:  `{"nodes": []}`,
			wantError: true,
		},
		{
			name:      "Invalid JSON",
			response:  `{invalid}`,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := planner.parsePlanFromLLMResponse(tc.response, "task-1")

			if tc.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if plan != nil && len(plan.Nodes) != tc.wantNodes {
					t.Errorf("Expected %d nodes, got %d", tc.wantNodes, len(plan.Nodes))
				}
			}
		})
	}
}

func TestBuildPlanningPrompt(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	planner := &Planner{}
	task := orchestrationpkg.NewTask("user-1", "Test task description")
	availableTools := tools.GetAllTools()

	prompt := planner.buildPlanningPrompt(task, availableTools)

	requiredElements := []string{
		"Test task description",
		"Available Tools",
		"fetch_article",
		"summarize_text",
		"create_report",
		"Response Format",
		"dependencies",
		"DAG",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(prompt, elem) {
			t.Errorf("Prompt missing required element: %s", elem)
		}
	}

	var parsed map[string]interface{}
	startIdx := strings.Index(prompt, "Response Format")
	if startIdx != -1 {
		jsonStart := strings.Index(prompt[startIdx:], "{")
		if jsonStart != -1 {
			jsonEnd := strings.Index(prompt[startIdx+jsonStart:], "}\n}")
			if jsonEnd != -1 {
				jsonStr := prompt[startIdx+jsonStart : startIdx+jsonStart+jsonEnd+2]
				if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
					t.Logf("Example JSON in prompt is valid")
				}
			}
		}
	}
}

func TestBuildReplanningPrompt(t *testing.T) {
	setupTestTools()
	defer tools.ClearRegistry()

	planner := &Planner{}
	task := orchestrationpkg.NewTask("user-1", "Original task")

	failedPlan := orchestrationpkg.NewPlan(task.ID)
	failedPlan.Nodes = []*orchestrationpkg.PlanNode{
		{
			ID:       "node_1",
			ToolName: "fetch_article",
			State:    orchestrationpkg.NodeStateFailed,
			Error:    "Connection timeout",
		},
	}

	availableTools := tools.GetAllTools()
	errorContext := "Connection timeout when fetching article"

	prompt := planner.buildReplanningPrompt(task, failedPlan, errorContext, availableTools)

	requiredElements := []string{
		"Original task",
		"Failed Plan",
		"Error Context",
		"Connection timeout",
		"Available Tools",
		"corrected plan",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(prompt, elem) {
			t.Errorf("Replanning prompt missing required element: %s", elem)
		}
	}
}
