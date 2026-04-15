package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	providerpkg "github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/trace"
	"github.com/DojoGenesis/gateway/tools"
)

// HandleQueryWithOrchestration processes a query using the orchestration engine for autonomous multi-step workflows.
// v0.0.30: Orchestration Engine integration
func (pa *PrimaryAgent) HandleQueryWithOrchestration(ctx context.Context, req QueryRequest) (*Response, error) {
	if pa.orchestrationPlanner == nil || pa.orchestrationEngine == nil {
		return nil, fmt.Errorf("orchestration components not initialized")
	}

	var rootSpan *trace.Span
	var traceID string

	if pa.traceLogger != nil {
		sessionID := req.UserID
		if sessionID == "" {
			sessionID = "default"
		}

		tid, err := pa.traceLogger.StartTrace(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to start trace: %w", err)
		}
		traceID = tid

		rootSpan, err = pa.traceLogger.StartSpan(ctx, traceID, "HandleQueryWithOrchestration", map[string]interface{}{
			"query":     req.Query,
			"user_id":   req.UserID,
			"user_tier": req.UserTier,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start root span: %w", err)
		}
		ctx = trace.WithSpan(ctx, rootSpan)

		defer func() {
			if rootSpan != nil {
				if err := pa.traceLogger.EndTrace(ctx, traceID, "completed"); err == nil {
					pa.traceLogger.EndSpan(ctx, rootSpan, nil)
				}
			}
		}()
	}

	task := orchestrationpkg.NewTask(req.UserID, req.Query)

	// Set project context before plan generation so planner can use it
	if req.ProjectID != "" {
		ctx = tools.WithProjectID(ctx, req.ProjectID)
	}

	var plan *orchestrationpkg.Plan
	var err error

	if pa.traceLogger != nil {
		planSpan, spanErr := pa.traceLogger.StartSpan(ctx, traceID, "plan_generation", map[string]interface{}{
			"task_id":    task.ID,
			"project_id": req.ProjectID,
		})
		if spanErr == nil {
			ctx = trace.WithSpan(ctx, planSpan)
			plan, err = pa.orchestrationPlanner.GeneratePlan(ctx, task)
			if err != nil {
				pa.traceLogger.FailSpan(ctx, planSpan, fmt.Sprintf("failed to generate plan: %v", err))
				ctx = trace.WithSpan(ctx, rootSpan)
				return nil, fmt.Errorf("failed to generate plan: %w", err)
			}
			pa.traceLogger.EndSpan(ctx, planSpan, map[string]interface{}{
				"plan_id":    plan.ID,
				"node_count": len(plan.Nodes),
			})
			ctx = trace.WithSpan(ctx, rootSpan)
		}
	} else {
		plan, err = pa.orchestrationPlanner.GeneratePlan(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("failed to generate plan: %w", err)
		}
	}

	// Create separate timeout context for execution (5x default timeout for complex workflows)
	executionCtx, executionCancel := context.WithTimeout(ctx, pa.timeout*5)
	defer executionCancel()

	if err := pa.orchestrationEngine.Execute(executionCtx, plan, task, req.UserID, nil); err != nil {
		if pa.traceLogger != nil && rootSpan != nil {
			pa.traceLogger.FailSpan(ctx, rootSpan, fmt.Sprintf("orchestration execution failed: %v", err))
		}
		return nil, fmt.Errorf("orchestration execution failed: %w", err)
	}

	response := pa.buildResponseFromPlan(plan, task)

	if req.UseMemory {
		// Create separate timeout context for memory storage (standard timeout)
		memoryCtx, memoryCancel := context.WithTimeout(ctx, pa.timeout)
		defer memoryCancel()

		if pa.traceLogger != nil {
			memorySpan, spanErr := pa.traceLogger.StartSpan(ctx, traceID, "memory_storage", map[string]interface{}{
				"user_id": req.UserID,
			})
			if spanErr == nil {
				ctx = trace.WithSpan(ctx, memorySpan)
				if err := pa.storeConversation(memoryCtx, req.UserID, req.Query, response); err != nil {
					// Log the error but don't fail the request since orchestration succeeded
					pa.traceLogger.FailSpan(ctx, memorySpan, fmt.Sprintf("failed to store conversation: %v", err))
					slog.Warn("orchestration succeeded but failed to store conversation", "user_id", req.UserID, "error", err)
					ctx = trace.WithSpan(ctx, rootSpan)
				} else {
					pa.traceLogger.EndSpan(ctx, memorySpan, nil)
					ctx = trace.WithSpan(ctx, rootSpan)
				}
			}
		} else {
			if err := pa.storeConversation(memoryCtx, req.UserID, req.Query, response); err != nil {
				// Log the error but don't fail the request since orchestration succeeded
				slog.Warn("orchestration succeeded but failed to store conversation", "user_id", req.UserID, "error", err)
			}
		}
	}

	if pa.traceLogger != nil && rootSpan != nil {
		rootSpan.AddMetadata("completed_nodes", len(plan.Nodes))
		rootSpan.AddMetadata("plan_version", plan.Version)
	}

	return response, nil
}

// synthesizePlanSummary creates a natural language summary from plan results using LLM.
// v0.0.30: Orchestration Engine integration - sophisticated response synthesis
func (pa *PrimaryAgent) synthesizePlanSummary(ctx context.Context, plan *orchestrationpkg.Plan, task *orchestrationpkg.Task) string {
	// Build context for LLM synthesis
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Original Task: %s\n\n", task.Description))
	contextBuilder.WriteString("Execution Results:\n")

	for i, node := range plan.Nodes {
		contextBuilder.WriteString(fmt.Sprintf("%d. Tool: %s (Status: %s)\n", i+1, node.ToolName, node.State))
		if node.State == orchestrationpkg.NodeStateSuccess && node.Result != nil {
			resultJSON, _ := json.Marshal(node.Result)
			contextBuilder.WriteString(fmt.Sprintf("   Result: %s\n", string(resultJSON)))
		} else if node.State == orchestrationpkg.NodeStateFailed {
			contextBuilder.WriteString(fmt.Sprintf("   Error: %s\n", node.Error))
		}
	}

	// Create synthesis prompt
	synthesisPrompt := fmt.Sprintf(`You are an AI assistant that synthesizes technical execution results into clear, user-friendly summaries.

Given the following task execution details, create a natural language summary that:
1. Directly addresses what the user asked for
2. Highlights key findings and results
3. Mentions any important caveats or failures
4. Is concise and easy to understand

%s

Provide only the summary, without preamble or meta-commentary.`, contextBuilder.String())

	// Attempt to synthesize using default provider
	provider, err := pa.pluginManager.GetProvider(pa.defaultProvider)
	if err != nil {
		return "" // Fallback to technical response
	}

	messages := []providerpkg.Message{
		{Role: "system", Content: "You are a helpful AI assistant that creates clear, concise summaries."},
		{Role: "user", Content: synthesisPrompt},
	}

	req := &providerpkg.CompletionRequest{
		Model:       "",
		Messages:    messages,
		Temperature: 0.3, // Lower temperature for more focused summaries
	}

	// Use a short timeout for synthesis to avoid delays
	synthesisCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := provider.GenerateCompletion(synthesisCtx, req)
	if err != nil {
		return "" // Fallback to technical response
	}

	return resp.Content
}

// buildResponseFromPlan aggregates results from a completed plan into a coherent response.
// v0.0.30: Orchestration Engine integration with LLM-powered synthesis
func (pa *PrimaryAgent) buildResponseFromPlan(plan *orchestrationpkg.Plan, task *orchestrationpkg.Task) *Response {
	var contentBuilder strings.Builder

	successfulNodes := 0
	failedNodes := 0
	for _, node := range plan.Nodes {
		if node.State == orchestrationpkg.NodeStateSuccess {
			successfulNodes++
		} else if node.State == orchestrationpkg.NodeStateFailed {
			failedNodes++
		}
	}

	// Try to generate natural language summary if synthesis is enabled
	enableSynthesis := true // Can be made configurable
	var naturalSummary string

	if enableSynthesis && pa.pluginManager != nil && successfulNodes > 0 {
		naturalSummary = pa.synthesizePlanSummary(context.Background(), plan, task)
	}

	// If we got a natural summary, use it as the primary content
	if naturalSummary != "" {
		contentBuilder.WriteString(naturalSummary)
		contentBuilder.WriteString("\n\n---\n\n")
		contentBuilder.WriteString("## Execution Details\n\n")
	}

	contentBuilder.WriteString(fmt.Sprintf("**Task**: %s\n\n", task.Description))
	contentBuilder.WriteString(fmt.Sprintf("**Plan Execution Summary**:\n"))
	contentBuilder.WriteString(fmt.Sprintf("- Total nodes: %d\n", len(plan.Nodes)))
	contentBuilder.WriteString(fmt.Sprintf("- Successful: %d\n", successfulNodes))
	contentBuilder.WriteString(fmt.Sprintf("- Failed: %d\n\n", failedNodes))

	if reasoning, ok := plan.Metadata["reasoning"].(string); ok && reasoning != "" {
		contentBuilder.WriteString(fmt.Sprintf("**Planning Strategy**:\n%s\n\n", reasoning))
	}

	contentBuilder.WriteString("**Node Results**:\n")
	for i, node := range plan.Nodes {
		contentBuilder.WriteString(fmt.Sprintf("\n%d. %s (%s)\n", i+1, node.ToolName, node.State))
		if node.State == orchestrationpkg.NodeStateSuccess && node.Result != nil {
			resultJSON, err := json.MarshalIndent(node.Result, "   ", "  ")
			if err == nil {
				contentBuilder.WriteString(fmt.Sprintf("   Result: %s\n", string(resultJSON)))
			}
		} else if node.State == orchestrationpkg.NodeStateFailed {
			contentBuilder.WriteString(fmt.Sprintf("   Error: %s\n", node.Error))
		}
		if node.StartTime != nil && node.EndTime != nil {
			duration := node.EndTime.Sub(*node.StartTime)
			contentBuilder.WriteString(fmt.Sprintf("   Duration: %v\n", duration))
		}
	}

	toolCalls := make([]providerpkg.ToolCall, len(plan.Nodes))
	for i, node := range plan.Nodes {
		toolCalls[i] = providerpkg.ToolCall{
			ID:        node.ID,
			Name:      node.ToolName,
			Arguments: node.Parameters,
		}
	}

	toolResults := make([]ToolExecutionResult, len(plan.Nodes))
	for i, node := range plan.Nodes {
		var err error
		if node.Error != "" {
			err = fmt.Errorf("%s", node.Error)
		}
		toolResults[i] = ToolExecutionResult{
			ToolCallID: node.ID,
			ToolName:   node.ToolName,
			Result:     node.Result,
			Error:      err,
		}
	}

	// Aggregate usage from plan metadata if available
	usage := providerpkg.Usage{}
	if planUsage, ok := plan.Metadata["total_usage"].(map[string]interface{}); ok {
		if inputTokens, ok := planUsage["input_tokens"].(float64); ok {
			usage.InputTokens = int(inputTokens)
		}
		if outputTokens, ok := planUsage["output_tokens"].(float64); ok {
			usage.OutputTokens = int(outputTokens)
		}
		if totalTokens, ok := planUsage["total_tokens"].(float64); ok {
			usage.TotalTokens = int(totalTokens)
		}
	}

	// If total_usage not found, try to extract from individual node metadata
	if usage.TotalTokens == 0 {
		for _, node := range plan.Nodes {
			if nodeUsage, ok := node.Result["usage"].(map[string]interface{}); ok {
				if inputTokens, ok := nodeUsage["input_tokens"].(float64); ok {
					usage.InputTokens += int(inputTokens)
				}
				if outputTokens, ok := nodeUsage["output_tokens"].(float64); ok {
					usage.OutputTokens += int(outputTokens)
				}
			}
		}
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}

	return &Response{
		ID:          plan.ID,
		Content:     contentBuilder.String(),
		Model:       "orchestration-engine",
		Provider:    "orchestration",
		Usage:       usage,
		ToolCalls:   toolCalls,
		ToolResults: toolResults,
		Timestamp:   time.Now(),
	}
}

// shouldUseOrchestration determines if a query should be routed to the orchestration engine.
// v0.0.30: Orchestration routing logic
func (pa *PrimaryAgent) shouldUseOrchestration(intent Intent, confidence float64, query string, userTier string) bool {
	// Don't use orchestration if components not initialized
	if pa.orchestrationPlanner == nil || pa.orchestrationEngine == nil {
		return false
	}

	// Check for multi-step indicators in the query
	queryLower := strings.ToLower(query)
	multiStepIndicators := []string{
		"and then",
		"then",
		"after that",
		"next",
		"finally",
		"first", "second", "third",
		"step 1", "step 2", "step 3",
		"1.", "2.", "3.",
	}

	hasMultiStep := false
	for _, indicator := range multiStepIndicators {
		if strings.Contains(queryLower, indicator) {
			hasMultiStep = true
			break
		}
	}

	// Check for research + creation pattern
	hasResearch := strings.Contains(queryLower, "research") ||
		strings.Contains(queryLower, "find") ||
		strings.Contains(queryLower, "search for")

	hasCreation := strings.Contains(queryLower, "create") ||
		strings.Contains(queryLower, "generate") ||
		strings.Contains(queryLower, "write") ||
		strings.Contains(queryLower, "make") ||
		strings.Contains(queryLower, "build")

	hasResearchAndCreation := hasResearch && hasCreation

	// Check for multiple tool-like actions
	actionWords := []string{"analyze", "summarize", "extract", "fetch", "download", "compare", "combine"}
	actionCount := 0
	for _, action := range actionWords {
		if strings.Contains(queryLower, action) {
			actionCount++
		}
	}
	hasMultipleActions := actionCount >= 2

	// Route to orchestration if:
	// 1. Query has multi-step indicators
	// 2. Query has research + creation pattern
	// 3. Query has multiple tool-like actions
	// 4. Intent is complex and confidence is high
	if hasMultiStep || hasResearchAndCreation || hasMultipleActions {
		return true
	}

	// For think/build intents with high confidence, use orchestration for long queries
	if (intent == IntentThink || intent == IntentBuild) && confidence >= 0.8 {
		// Check query length as proxy for complexity
		if len(strings.Split(query, " ")) >= 15 {
			return true
		}
	}

	return false
}

// getOrchestrationReason returns a human-readable reason for routing to orchestration.
// v0.0.30: Orchestration routing metadata
func (pa *PrimaryAgent) getOrchestrationReason(intent Intent, confidence float64, query string) string {
	queryLower := strings.ToLower(query)

	if strings.Contains(queryLower, "and then") || strings.Contains(queryLower, "then") {
		return "multi_step_sequence_detected"
	}

	hasResearch := strings.Contains(queryLower, "research") || strings.Contains(queryLower, "find")
	hasCreation := strings.Contains(queryLower, "create") || strings.Contains(queryLower, "generate")
	if hasResearch && hasCreation {
		return "research_and_creation_workflow"
	}

	if strings.Contains(queryLower, "1.") || strings.Contains(queryLower, "step 1") {
		return "numbered_steps_detected"
	}

	actionWords := []string{"analyze", "summarize", "extract", "fetch", "compare"}
	actionCount := 0
	for _, action := range actionWords {
		if strings.Contains(queryLower, action) {
			actionCount++
		}
	}
	if actionCount >= 2 {
		return "multiple_actions_detected"
	}

	if (intent == IntentThink || intent == IntentBuild) && confidence >= 0.8 {
		return "complex_intent_high_confidence"
	}

	return "complex_query_heuristic"
}
