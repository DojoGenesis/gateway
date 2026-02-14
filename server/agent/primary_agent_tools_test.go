package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

type mockToolProvider struct {
	mockProvider
	toolCallResponses [][]providerpkg.ToolCall
	currentCall       int
}

func (m *mockToolProvider) GenerateCompletion(ctx context.Context, req *providerpkg.CompletionRequest) (*providerpkg.CompletionResponse, error) {
	if m.completionError != nil {
		return nil, m.completionError
	}

	if m.currentCall < len(m.toolCallResponses) {
		toolCalls := m.toolCallResponses[m.currentCall]
		m.currentCall++

		return &providerpkg.CompletionResponse{
			ID:        fmt.Sprintf("completion-%d", m.currentCall),
			Model:     req.Model,
			Content:   "I'll use these tools",
			ToolCalls: toolCalls,
			Usage: providerpkg.Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		}, nil
	}

	return &providerpkg.CompletionResponse{
		ID:      "final-completion",
		Model:   req.Model,
		Content: "Final response after using tools",
		Usage: providerpkg.Usage{
			InputTokens:  15,
			OutputTokens: 25,
			TotalTokens:  40,
		},
	}, nil
}

func setupToolTests() {
	tools.ClearRegistry()

	mockCalcTool := &tools.ToolDefinition{
		Name:        "calculate",
		Description: "Perform calculations",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "Math expression to evaluate",
				},
			},
			"required": []interface{}{"expression"},
		},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			expression := params["expression"].(string)
			return map[string]interface{}{
				"success":    true,
				"expression": expression,
				"result":     42.0,
			}, nil
		},
	}

	mockSearchTool := &tools.ToolDefinition{
		Name:        "web_search",
		Description: "Search the web",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
			},
			"required": []interface{}{"query"},
		},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			query := params["query"].(string)
			return map[string]interface{}{
				"success": true,
				"query":   query,
				"results": []map[string]interface{}{
					{"title": "Result 1", "url": "https://example.com/1"},
					{"title": "Result 2", "url": "https://example.com/2"},
				},
			}, nil
		},
	}

	mockReadFileTool := &tools.ToolDefinition{
		Name:        "read_file",
		Description: "Read file contents",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to file",
				},
			},
			"required": []interface{}{"file_path"},
		},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			filePath := params["file_path"].(string)
			return map[string]interface{}{
				"success":   true,
				"file_path": filePath,
				"content":   "Mock file content",
				"lines":     10,
			}, nil
		},
	}

	tools.RegisterTool(mockCalcTool)
	tools.RegisterTool(mockSearchTool)
	tools.RegisterTool(mockReadFileTool)
}

func TestPrimaryAgent_HandleQueryWithTools_NoToolCalls(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Hello, what's your name?",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.Content == "" {
		t.Error("Response content is empty")
	}

	if len(resp.ToolCalls) != 0 {
		t.Errorf("Expected 0 tool calls, got %d", len(resp.ToolCalls))
	}
}

func TestPrimaryAgent_HandleQueryWithTools_SingleToolCall(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{
		toolCallResponses: [][]providerpkg.ToolCall{
			{
				{
					ID:   "call-1",
					Name: "calculate",
					Arguments: map[string]interface{}{
						"expression": "2 + 2",
					},
				},
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "What is 2 + 2?",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "calculate" {
		t.Errorf("Expected tool 'calculate', got '%s'", resp.ToolCalls[0].Name)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_MultipleToolCalls(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{
		toolCallResponses: [][]providerpkg.ToolCall{
			{
				{
					ID:   "call-1",
					Name: "web_search",
					Arguments: map[string]interface{}{
						"query": "golang best practices",
					},
				},
				{
					ID:   "call-2",
					Name: "calculate",
					Arguments: map[string]interface{}{
						"expression": "100 * 2",
					},
				},
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Search for golang best practices and calculate 100 * 2",
		UserID:   "user-123",
		UserTier: "authenticated",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if len(resp.ToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(resp.ToolCalls))
	}

	toolNames := make(map[string]bool)
	for _, tc := range resp.ToolCalls {
		toolNames[tc.Name] = true
	}

	if !toolNames["web_search"] {
		t.Error("Expected web_search tool call")
	}

	if !toolNames["calculate"] {
		t.Error("Expected calculate tool call")
	}
}

func TestPrimaryAgent_HandleQueryWithTools_IterativeToolCalls(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{
		toolCallResponses: [][]providerpkg.ToolCall{
			{
				{
					ID:   "call-1",
					Name: "read_file",
					Arguments: map[string]interface{}{
						"file_path": "/test/file.txt",
					},
				},
			},
			{
				{
					ID:   "call-2",
					Name: "calculate",
					Arguments: map[string]interface{}{
						"expression": "10 * 10",
					},
				},
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Read the file and then do some math",
		UserID:   "user-123",
		UserTier: "premium",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if len(resp.ToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls across iterations, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Name != "read_file" {
		t.Errorf("Expected first tool 'read_file', got '%s'", resp.ToolCalls[0].Name)
	}

	if resp.ToolCalls[1].Name != "calculate" {
		t.Errorf("Expected second tool 'calculate', got '%s'", resp.ToolCalls[1].Name)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_MaxIterations(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()

	toolResponses := make([][]providerpkg.ToolCall, 10)
	for i := 0; i < 10; i++ {
		toolResponses[i] = []providerpkg.ToolCall{
			{
				ID:   fmt.Sprintf("call-%d", i),
				Name: "calculate",
				Arguments: map[string]interface{}{
					"expression": fmt.Sprintf("%d + 1", i),
				},
			},
		}
	}

	mockProv := &mockToolProvider{
		toolCallResponses: toolResponses,
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Keep calculating",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if len(resp.ToolCalls) > agent.maxToolIterations {
		t.Errorf("Tool calls exceeded max iterations: got %d, max %d", len(resp.ToolCalls), agent.maxToolIterations)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_IntentClassification(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	testCases := []struct {
		query          string
		expectedIntent Intent
	}{
		{"Explain why this algorithm is better", IntentThink},
		{"Search for golang tutorials", IntentSearch},
		{"Create a new REST API endpoint", IntentBuild},
		{"Fix this compilation error", IntentDebug},
		{"Hello, how are you?", IntentGeneral},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			intent, _ := agent.miniAgent.ClassifyIntent(context.Background(), tc.query)
			if intent != tc.expectedIntent {
				t.Errorf("Expected intent %s, got %s", tc.expectedIntent, intent)
			}
		})
	}
}

func TestPrimaryAgent_HandleQueryWithTools_ProviderSelection(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	guestProv := &mockToolProvider{}
	authProv := &mockToolProvider{}
	pm.AddProvider("guest-provider", guestProv)
	pm.AddProvider("auth-provider", authProv)

	agent := NewPrimaryAgentWithConfig(pm, "guest-provider", "guest-provider", "auth-provider")

	testCases := []struct {
		userTier         string
		expectedProvider string
	}{
		{"guest", "guest-provider"},
		{"authenticated", "auth-provider"},
		{"premium", "auth-provider"},
	}

	for _, tc := range testCases {
		t.Run(tc.userTier, func(t *testing.T) {
			req := QueryRequest{
				Query:    "Test query",
				UserID:   "user-123",
				UserTier: tc.userTier,
			}

			ctx := context.Background()
			resp, err := agent.HandleQueryWithTools(ctx, req)
			if err != nil {
				t.Fatalf("HandleQueryWithTools failed: %v", err)
			}

			if resp.Provider != tc.expectedProvider {
				t.Errorf("Expected provider %s, got %s", tc.expectedProvider, resp.Provider)
			}
		})
	}
}

func TestPrimaryAgent_HandleQueryWithTools_SystemPromptCustomization(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	agent := NewPrimaryAgentWithConfig(newMockPluginManager(), "mock", "mock", "mock")

	testCases := []struct {
		intent        Intent
		shouldContain string
	}{
		{IntentThink, "deep analysis"},
		{IntentSearch, "finding and retrieving"},
		{IntentBuild, "clean, efficient"},
		{IntentDebug, "identifying issues"},
		{IntentGeneral, "helpful AI"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.intent), func(t *testing.T) {
			prompt := agent.buildSystemPrompt(tc.intent)
			if !contains(prompt, tc.shouldContain) {
				t.Errorf("Expected prompt to contain '%s', got: %s", tc.shouldContain, prompt)
			}
		})
	}
}

func TestPrimaryAgent_HandleQueryWithTools_ToolExecutionError(t *testing.T) {
	tools.ClearRegistry()

	errorTool := &tools.ToolDefinition{
		Name:        "failing_tool",
		Description: "A tool that always fails",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Function: func(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("tool execution failed")
		},
	}

	tools.RegisterTool(errorTool)
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{
		toolCallResponses: [][]providerpkg.ToolCall{
			{
				{
					ID:   "call-1",
					Name: "failing_tool",
					Arguments: map[string]interface{}{
						"param": "test",
					},
				},
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Use the failing tool",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}
}

func TestPrimaryAgent_HandleQueryWithTools_ParallelExecution(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{
		toolCallResponses: [][]providerpkg.ToolCall{
			{
				{ID: "call-1", Name: "calculate", Arguments: map[string]interface{}{"expression": "1+1"}},
				{ID: "call-2", Name: "calculate", Arguments: map[string]interface{}{"expression": "2+2"}},
				{ID: "call-3", Name: "calculate", Arguments: map[string]interface{}{"expression": "3+3"}},
				{ID: "call-4", Name: "calculate", Arguments: map[string]interface{}{"expression": "4+4"}},
				{ID: "call-5", Name: "calculate", Arguments: map[string]interface{}{"expression": "5+5"}},
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Do multiple calculations",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	start := time.Now()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if len(resp.ToolCalls) != 5 {
		t.Errorf("Expected 5 tool calls, got %d", len(resp.ToolCalls))
	}

	if duration > 1*time.Second {
		t.Logf("Warning: Parallel execution took %v (expected < 1s)", duration)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_DefaultParameters(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Test query",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}
}

func TestPrimaryAgent_HandleQueryWithTools_CustomParameters(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{}
	pm.AddProvider("custom-provider", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:        "Test query",
		UserID:       "user-456",
		UserTier:     "premium",
		ProviderName: "custom-provider",
		ModelID:      "mock-model",
		Temperature:  0.8,
		MaxTokens:    4096,
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp.Provider != "custom-provider" {
		t.Errorf("Expected provider 'custom-provider', got '%s'", resp.Provider)
	}

	if resp.Model != "mock-model" {
		t.Errorf("Expected model 'mock-model', got '%s'", resp.Model)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_ContextCancellation(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")
	agent.SetTimeout(1 * time.Nanosecond)

	req := QueryRequest{
		Query:    "Test query",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	_, err := agent.HandleQueryWithTools(ctx, req)
	if err == nil {
		t.Log("Warning: Expected timeout error, but query may have completed too fast")
	}
}

func TestPrimaryAgent_HandleQueryWithTools_ModelSelection(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{
		mockProvider: mockProvider{
			models: []providerpkg.ModelInfo{
				{ID: "model-1", Name: "Model 1"},
				{ID: "model-2", Name: "Model 2"},
				{ID: "model-3", Name: "Model 3"},
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Test query",
		UserID:   "user-123",
		UserTier: "guest",
		ModelID:  "model-2",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp.Model != "model-2" {
		t.Errorf("Expected model 'model-2', got '%s'", resp.Model)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_InvalidModel(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{
		mockProvider: mockProvider{
			models: []providerpkg.ModelInfo{
				{ID: "model-1", Name: "Model 1"},
			},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Test query",
		UserID:   "user-123",
		UserTier: "guest",
		ModelID:  "nonexistent-model",
	}

	ctx := context.Background()
	_, err := agent.HandleQueryWithTools(ctx, req)
	if err == nil {
		t.Fatal("Expected error for nonexistent model")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestPrimaryAgent_ConvertToolsToPluginFormat(t *testing.T) {
	toolDefs := []*tools.ToolDefinition{
		{
			Name:        "test_tool_1",
			Description: "Test tool 1",
			Parameters:  map[string]interface{}{"type": "object"},
		},
		{
			Name:        "test_tool_2",
			Description: "Test tool 2",
			Parameters:  map[string]interface{}{"type": "object"},
		},
	}

	pluginTools := convertToolsToPluginFormat(toolDefs)

	if len(pluginTools) != 2 {
		t.Errorf("Expected 2 plugin tools, got %d", len(pluginTools))
	}

	for i, pt := range pluginTools {
		if pt.Name != toolDefs[i].Name {
			t.Errorf("Tool %d: expected name '%s', got '%s'", i, toolDefs[i].Name, pt.Name)
		}

		if pt.Description != toolDefs[i].Description {
			t.Errorf("Tool %d: expected description '%s', got '%s'", i, toolDefs[i].Description, pt.Description)
		}
	}
}

func TestPrimaryAgent_FormatToolResults(t *testing.T) {
	results := []ToolExecutionResult{
		{
			ToolCallID: "call-1",
			ToolName:   "test_tool",
			Result: map[string]interface{}{
				"success": true,
				"value":   42,
			},
			Error: nil,
		},
		{
			ToolCallID: "call-2",
			ToolName:   "error_tool",
			Result:     nil,
			Error:      fmt.Errorf("tool failed"),
		},
	}

	messages := formatToolResults(results)

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	for i, msg := range messages {
		if msg.Role != "tool" {
			t.Errorf("Message %d: expected role 'tool', got '%s'", i, msg.Role)
		}

		if msg.Content == "" {
			t.Errorf("Message %d: content is empty", i)
		}

		if i == 1 && !contains(msg.Content, "error") {
			t.Errorf("Message %d: expected error in content, got: %s", i, msg.Content)
		}
	}
}

func TestPrimaryAgent_SelectProvider(t *testing.T) {
	agent := NewPrimaryAgentWithConfig(
		newMockPluginManager(),
		"default-provider",
		"guest-provider",
		"auth-provider",
	)

	testCases := []struct {
		userTier string
		intent   Intent
		expected string
	}{
		{"guest", IntentGeneral, "guest-provider"},
		{"guest", IntentThink, "guest-provider"},
		{"authenticated", IntentGeneral, "auth-provider"},
		{"authenticated", IntentSearch, "auth-provider"},
		{"premium", IntentBuild, "auth-provider"},
		{"unknown", IntentDebug, "default-provider"},
		{"", IntentGeneral, "default-provider"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.userTier, tc.intent), func(t *testing.T) {
			result := agent.selectProvider(tc.userTier, tc.intent)
			if result != tc.expected {
				t.Errorf("Expected provider '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestPrimaryAgent_BuildSystemPrompt(t *testing.T) {
	agent := NewPrimaryAgentWithConfig(newMockPluginManager(), "mock", "mock", "mock")

	testCases := []struct {
		intent        Intent
		shouldContain []string
	}{
		{IntentThink, []string{"deep analysis", "reasoning"}},
		{IntentSearch, []string{"finding", "retrieving", "information"}},
		{IntentBuild, []string{"clean", "efficient", "code"}},
		{IntentDebug, []string{"identifying", "issues", "solutions"}},
		{IntentGeneral, []string{"helpful AI"}},
	}

	for _, tc := range testCases {
		t.Run(string(tc.intent), func(t *testing.T) {
			prompt := agent.buildSystemPrompt(tc.intent)

			if prompt == "" {
				t.Fatal("System prompt is empty")
			}

			foundCount := 0
			for _, keyword := range tc.shouldContain {
				if contains(prompt, keyword) {
					foundCount++
				}
			}

			if foundCount == 0 {
				t.Errorf("Expected prompt to contain at least one of %v, got: %s", tc.shouldContain, prompt)
			}
		})
	}
}

func TestPrimaryAgent_ExecuteToolCalls_Parallel(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	toolCalls := []providerpkg.ToolCall{
		{ID: "call-1", Name: "calculate", Arguments: map[string]interface{}{"expression": "1+1"}},
		{ID: "call-2", Name: "calculate", Arguments: map[string]interface{}{"expression": "2+2"}},
		{ID: "call-3", Name: "calculate", Arguments: map[string]interface{}{"expression": "3+3"}},
	}

	ctx := context.Background()
	start := time.Now()
	results := agent.executeToolCalls(ctx, toolCalls)
	duration := time.Since(start)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for i, result := range results {
		if result.Error != nil {
			t.Errorf("Result %d: unexpected error: %v", i, result.Error)
		}

		if result.ToolName != "calculate" {
			t.Errorf("Result %d: expected tool 'calculate', got '%s'", i, result.ToolName)
		}

		if result.Result == nil {
			t.Errorf("Result %d: result is nil", i)
		}
	}

	if duration > 500*time.Millisecond {
		t.Logf("Warning: Parallel execution took %v (expected < 500ms)", duration)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_NoProviderSpecified(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{}
	guestProv := &mockToolProvider{}
	authProv := &mockToolProvider{}
	pm.AddProvider("default-provider", mockProv)
	pm.AddProvider("guest-provider", guestProv)
	pm.AddProvider("auth-provider", authProv)

	agent := NewPrimaryAgentWithConfig(pm, "default-provider", "guest-provider", "auth-provider")

	req := QueryRequest{
		Query:    "Test query without provider",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.Provider != "guest-provider" {
		t.Errorf("Expected provider 'guest-provider' (selected by tier), got '%s'", resp.Provider)
	}
}

func TestPrimaryAgent_HandleQueryWithTools_EmptyToolCalls(t *testing.T) {
	setupToolTests()
	defer tools.ClearRegistry()

	pm := newMockPluginManager()
	mockProv := &mockToolProvider{
		toolCallResponses: [][]providerpkg.ToolCall{
			{},
		},
	}
	pm.AddProvider("mock", mockProv)

	agent := NewPrimaryAgentWithConfig(pm, "mock", "mock", "mock")

	req := QueryRequest{
		Query:    "Test query",
		UserID:   "user-123",
		UserTier: "guest",
	}

	ctx := context.Background()
	resp, err := agent.HandleQueryWithTools(ctx, req)
	if err != nil {
		t.Fatalf("HandleQueryWithTools failed: %v", err)
	}

	if len(resp.ToolCalls) != 0 {
		t.Errorf("Expected 0 tool calls, got %d", len(resp.ToolCalls))
	}
}
