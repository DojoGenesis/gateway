package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/trace"
	"github.com/google/uuid"
)

type PluginManagerInterface interface {
	GetProvider(name string) (providerpkg.ModelProvider, error)
	GetProviders() map[string]providerpkg.ModelProvider
}

type Planner struct {
	pluginManager PluginManagerInterface
	traceLogger   *trace.TraceLogger
	providerName  string
	modelID       string
}

func NewPlanner(pluginManager PluginManagerInterface, traceLogger *trace.TraceLogger, providerName, modelID string) *Planner {
	return &Planner{
		pluginManager: pluginManager,
		traceLogger:   traceLogger,
		providerName:  providerName,
		modelID:       modelID,
	}
}

func (p *Planner) GeneratePlan(ctx context.Context, task *orchestrationpkg.Task) (*orchestrationpkg.Plan, error) {
	if p.traceLogger != nil {
		span, err := p.traceLogger.StartSpan(ctx, task.ID, "planner.generate_plan", map[string]interface{}{
			"task_id":     task.ID,
			"description": task.Description,
		})
		if err == nil {
			defer func() {
				_ = p.traceLogger.EndSpan(ctx, span, map[string]interface{}{})
			}()
			ctx = trace.WithSpan(ctx, span)
		}
	}

	availableTools := tools.GetAllTools()
	prompt := p.buildPlanningPrompt(task, availableTools)

	response, err := p.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	plan, err := p.parsePlanFromLLMResponse(response, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	if err := plan.ValidateDAG(); err != nil {
		return nil, fmt.Errorf("invalid plan DAG: %w", err)
	}

	if err := p.validateToolNames(plan, availableTools); err != nil {
		return nil, fmt.Errorf("invalid tool names in plan: %w", err)
	}

	return plan, nil
}

func (p *Planner) RegeneratePlan(ctx context.Context, task *orchestrationpkg.Task, failedPlan *orchestrationpkg.Plan, errorContext string) (*orchestrationpkg.Plan, error) {
	if p.traceLogger != nil {
		span, err := p.traceLogger.StartSpan(ctx, task.ID, "planner.regenerate_plan", map[string]interface{}{
			"task_id":       task.ID,
			"failed_plan":   failedPlan.ID,
			"error_context": errorContext,
		})
		if err == nil {
			defer func() {
				_ = p.traceLogger.EndSpan(ctx, span, map[string]interface{}{})
			}()
			ctx = trace.WithSpan(ctx, span)
		}
	}

	availableTools := tools.GetAllTools()
	prompt := p.buildReplanningPrompt(task, failedPlan, errorContext, availableTools)

	response, err := p.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM for replanning: %w", err)
	}

	newPlan, err := p.parsePlanFromLLMResponse(response, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse replanned plan: %w", err)
	}

	newPlan.Version = failedPlan.Version + 1

	if err := newPlan.ValidateDAG(); err != nil {
		return nil, fmt.Errorf("invalid replanned DAG: %w", err)
	}

	if err := p.validateToolNames(newPlan, availableTools); err != nil {
		return nil, fmt.Errorf("invalid tool names in replanned: %w", err)
	}

	return newPlan, nil
}

func (p *Planner) buildPlanningPrompt(task *orchestrationpkg.Task, availableTools []*tools.ToolDefinition) string {
	var sb strings.Builder

	sb.WriteString("You are an expert orchestration planner. Your task is to decompose a user's request into a Directed Acyclic Graph (DAG) of tool calls.\n\n")
	sb.WriteString("Task Description:\n")
	sb.WriteString(task.Description)
	sb.WriteString("\n\n")

	sb.WriteString("Available Tools:\n")
	for _, tool := range availableTools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		if paramsJSON, err := json.Marshal(tool.Parameters); err == nil {
			sb.WriteString(fmt.Sprintf("  Parameters: %s\n", string(paramsJSON)))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("Requirements:\n")
	sb.WriteString("1. Create a plan that breaks down the task into discrete tool calls\n")
	sb.WriteString("2. Identify dependencies between nodes - a node can only run after its dependencies complete\n")
	sb.WriteString("3. Nodes with no dependencies on each other can run in parallel\n")
	sb.WriteString("4. Each node must have a unique ID, tool name, parameters, and list of dependency node IDs\n")
	sb.WriteString("5. The DAG must not contain cycles\n")
	sb.WriteString("6. Only use tools from the available tools list\n\n")

	sb.WriteString("Response Format (JSON):\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"Brief explanation of your planning strategy\",\n")
	sb.WriteString("  \"nodes\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"id\": \"node_1\",\n")
	sb.WriteString("      \"tool_name\": \"tool_name_here\",\n")
	sb.WriteString("      \"parameters\": {\"param1\": \"value1\"},\n")
	sb.WriteString("      \"dependencies\": []\n")
	sb.WriteString("    },\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"id\": \"node_2\",\n")
	sb.WriteString("      \"tool_name\": \"another_tool\",\n")
	sb.WriteString("      \"parameters\": {\"param1\": \"value1\"},\n")
	sb.WriteString("      \"dependencies\": [\"node_1\"]\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n\n")
	sb.WriteString("Respond with ONLY the JSON, no additional text.")

	return sb.String()
}

func (p *Planner) buildReplanningPrompt(task *orchestrationpkg.Task, failedPlan *orchestrationpkg.Plan, errorContext string, availableTools []*tools.ToolDefinition) string {
	var sb strings.Builder

	sb.WriteString("You are an expert orchestration planner performing error recovery. A previous plan failed and you need to generate a corrected plan.\n\n")
	
	sb.WriteString("Original Task Description:\n")
	sb.WriteString(task.Description)
	sb.WriteString("\n\n")

	sb.WriteString("Failed Plan:\n")
	planJSON, _ := json.MarshalIndent(failedPlan, "", "  ")
	sb.WriteString(string(planJSON))
	sb.WriteString("\n\n")

	sb.WriteString("Error Context:\n")
	sb.WriteString(errorContext)
	sb.WriteString("\n\n")

	completedNodes := make([]*orchestrationpkg.PlanNode, 0)
	for _, node := range failedPlan.Nodes {
		if node.State == orchestrationpkg.NodeStateSuccess {
			completedNodes = append(completedNodes, node)
		}
	}

	if len(completedNodes) > 0 {
		sb.WriteString("Completed Nodes (preserve their work):\n")
		for _, node := range completedNodes {
			sb.WriteString(fmt.Sprintf("- %s (%s): %v\n", node.ID, node.ToolName, node.Result))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Available Tools:\n")
	for _, tool := range availableTools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		if paramsJSON, err := json.Marshal(tool.Parameters); err == nil {
			sb.WriteString(fmt.Sprintf("  Parameters: %s\n", string(paramsJSON)))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("Requirements:\n")
	sb.WriteString("1. Analyze why the previous plan failed\n")
	sb.WriteString("2. Create a corrected plan that addresses the failure\n")
	sb.WriteString("3. Preserve work from completed nodes where possible\n")
	sb.WriteString("4. Each node must have a unique ID, tool name, parameters, and dependencies\n")
	sb.WriteString("5. The DAG must not contain cycles\n")
	sb.WriteString("6. Only use tools from the available tools list\n\n")

	sb.WriteString("Response Format (JSON):\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"Explanation of what went wrong and how the new plan fixes it\",\n")
	sb.WriteString("  \"nodes\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"id\": \"node_1\",\n")
	sb.WriteString("      \"tool_name\": \"tool_name_here\",\n")
	sb.WriteString("      \"parameters\": {\"param1\": \"value1\"},\n")
	sb.WriteString("      \"dependencies\": []\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n\n")
	sb.WriteString("Respond with ONLY the JSON, no additional text.")

	return sb.String()
}

func (p *Planner) parsePlanFromLLMResponse(response string, taskID string) (*orchestrationpkg.Plan, error) {
	response = strings.TrimSpace(response)
	
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	
	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	
	jsonStr := response[startIdx : endIdx+1]

	var planData struct {
		Reasoning string `json:"reasoning"`
		Nodes     []struct {
			ID           string                 `json:"id"`
			ToolName     string                 `json:"tool_name"`
			Parameters   map[string]interface{} `json:"parameters"`
			Dependencies []string               `json:"dependencies"`
		} `json:"nodes"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &planData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	plan := orchestrationpkg.NewPlan(taskID)

	if planData.Reasoning != "" {
		plan.Metadata["reasoning"] = planData.Reasoning
	}

	nodeIDMap := make(map[string]bool)
	for _, nodeData := range planData.Nodes {
		if nodeData.ID == "" {
			nodeData.ID = uuid.New().String()
		}

		if nodeIDMap[nodeData.ID] {
			return nil, fmt.Errorf("duplicate node ID: %s", nodeData.ID)
		}
		nodeIDMap[nodeData.ID] = true

		if nodeData.Parameters == nil {
			nodeData.Parameters = make(map[string]interface{})
		}

		if nodeData.Dependencies == nil {
			nodeData.Dependencies = []string{}
		}

		node := &orchestrationpkg.PlanNode{
			ID:           nodeData.ID,
			ToolName:     nodeData.ToolName,
			Parameters:   nodeData.Parameters,
			Dependencies: nodeData.Dependencies,
			State:        orchestrationpkg.NodeStatePending,
			RetryCount:   0,
		}

		plan.Nodes = append(plan.Nodes, node)
	}

	if len(plan.Nodes) == 0 {
		return nil, fmt.Errorf("plan has no nodes")
	}

	return plan, nil
}

func (p *Planner) callLLM(ctx context.Context, prompt string) (string, error) {
	provider, err := p.pluginManager.GetProvider(p.providerName)
	if err != nil {
		return "", fmt.Errorf("failed to get provider: %w", err)
	}

	models, err := provider.ListModels(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		return "", fmt.Errorf("no models available")
	}

	modelID := p.modelID
	if modelID == "" {
		modelID = models[0].ID
	}

	req := &providerpkg.CompletionRequest{
		Model: modelID,
		Messages: []providerpkg.Message{
			{
				Role:    "system",
				Content: "You are an expert task planner that decomposes complex requests into structured execution plans.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
		MaxTokens:   4096,
		Stream:      false,
	}

	resp, err := provider.GenerateCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to generate completion: %w", err)
	}

	return resp.Content, nil
}

func (p *Planner) validateToolNames(plan *orchestrationpkg.Plan, availableTools []*tools.ToolDefinition) error {
	toolMap := make(map[string]bool)
	for _, tool := range availableTools {
		toolMap[tool.Name] = true
	}

	for _, node := range plan.Nodes {
		if !toolMap[node.ToolName] {
			return fmt.Errorf("node %s references unknown tool: %s", node.ID, node.ToolName)
		}
	}

	return nil
}
