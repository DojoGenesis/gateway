package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	orchestrationpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

// handleGatewayListTools returns all registered tools with namespace info and MCP server origin.
// GET /v1/gateway/tools
func (s *Server) handleGatewayListTools(c *gin.Context) {
	if s.toolRegistry == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Tool registry not available")
		return
	}

	tools, err := s.toolRegistry.List(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", fmt.Sprintf("Failed to list tools: %v", err))
		return
	}

	// Transform to API response format
	toolsResp := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		toolsResp = append(toolsResp, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
			"namespace":   extractNamespace(tool.Name),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"tools": toolsResp,
		"count": len(toolsResp),
	})
}

// handleGatewayCreateAgent creates a new agent with the provided disposition configuration.
// POST /v1/gateway/agents
func (s *Server) handleGatewayCreateAgent(c *gin.Context) {
	var req struct {
		WorkspaceRoot string `json:"workspace_root" binding:"required"`
		ActiveMode    string `json:"active_mode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if s.agentInitializer == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Agent initializer not available")
		return
	}

	// Initialize agent configuration from disposition
	agentConfig, err := s.agentInitializer.Initialize(c.Request.Context(), req.WorkspaceRoot, req.ActiveMode)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "agent_init_failed", fmt.Sprintf("Failed to initialize agent: %v", err))
		return
	}

	// Generate agent ID
	agentID := uuid.New().String()

	// Store agent (in-memory for now - production would use persistence)
	s.agentMu.Lock()
	s.agents[agentID] = agentConfig
	s.agentMu.Unlock()

	// Note: Disposition is now set during engine construction in main.go
	// The standalone orchestration engine uses orchestrationpkg.WithDisposition option
	// No runtime pacing updates needed here

	c.JSON(http.StatusCreated, gin.H{
		"agent_id": agentID,
		"config":   agentConfig,
	})
}

// handleGatewayGetAgent retrieves agent status and current disposition.
// GET /v1/gateway/agents/:id
func (s *Server) handleGatewayGetAgent(c *gin.Context) {
	agentID := c.Param("id")

	s.agentMu.RLock()
	agentConfig, exists := s.agents[agentID]
	s.agentMu.RUnlock()

	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"config":   agentConfig,
		"status":   "active",
	})
}

// handleGatewayAgentChat handles chat interactions with a specific agent.
// POST /v1/gateway/agents/:id/chat
func (s *Server) handleGatewayAgentChat(c *gin.Context) {
	agentID := c.Param("id")

	var req struct {
		Message string `json:"message" binding:"required"`
		UserID  string `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	s.agentMu.RLock()
	agentConfig, exists := s.agents[agentID]
	s.agentMu.RUnlock()

	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

	if s.orchestrationEngine == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Orchestration engine not available")
		return
	}

	// Create a task from the chat message
	userID := req.UserID
	if userID == "" {
		userID = "anonymous"
	}
	task := orchestrationpkg.NewTask(userID, req.Message)

	// Generate plan from the task using the planner
	// Note: The planner needs to be accessible through the server
	// The orchestration engine should have access to it via its internal planner
	// For now, we'll create a simple manual plan for chat processing
	// In a production system, this would call the planner's GeneratePlan method
	plan := orchestrationpkg.NewPlan(task.ID)

	// Add a simple chat processing node
	// In a real implementation, the planner would decompose this into appropriate tool calls
	chatNode := orchestrationpkg.NewPlanNode(
		"process_chat_message",
		map[string]interface{}{
			"message":    req.Message,
			"agent_id":   agentID,
			"pacing":     agentConfig.Pacing,
			"depth":      agentConfig.Depth,
			"user_id":    userID,
		},
		[]string{}, // No dependencies for single-node plan
	)
	plan.Nodes = append(plan.Nodes, chatNode)

	// Execute the plan synchronously for chat (blocking for response)
	execErr := s.orchestrationEngine.Execute(c.Request.Context(), plan, task, userID)

	if execErr != nil {
		s.errorResponseWithDetails(c, http.StatusInternalServerError, "execution_failed",
			fmt.Sprintf("Chat processing failed: %v", execErr),
			gin.H{"agent_id": agentID})
		return
	}

	// Extract result from completed node
	var responseText string
	if chatNode.State == orchestrationpkg.NodeStateSuccess && chatNode.Result != nil {
		if response, ok := chatNode.Result["response"].(string); ok {
			responseText = response
		} else {
			responseText = fmt.Sprintf("Processed with %s pacing and %s depth", agentConfig.Pacing, agentConfig.Depth)
		}
	} else {
		responseText = "Chat processing completed"
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id":   agentID,
		"response":   responseText,
		"task_id":    task.ID,
		"plan_id":    plan.ID,
		"message":    req.Message,
		"disposition": map[string]interface{}{
			"pacing": agentConfig.Pacing,
			"depth":  agentConfig.Depth,
		},
	})
}

// handleGatewayOrchestrate submits an orchestration plan and returns execution ID.
// POST /v1/gateway/orchestrate
func (s *Server) handleGatewayOrchestrate(c *gin.Context) {
	var req struct {
		Plan   gateway.ExecutionPlan `json:"plan" binding:"required"`
		UserID string                `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if s.orchestrationEngine == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Orchestration engine not available")
		return
	}

	// Convert gateway.ExecutionPlan to orchestrationpkg.Plan
	orchPlan := convertToOrchestrationPlan(&req.Plan)

	// Create task
	task := &orchestrationpkg.Task{
		ID:          uuid.New().String(),
		Description: req.Plan.Name,
		CreatedAt:   time.Now(),
	}

	// Execute orchestration (async)
	go func() {
		userID := req.UserID
		if userID == "" {
			userID = "anonymous"
		}
		_ = s.orchestrationEngine.Execute(c.Request.Context(), orchPlan, task, userID)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"execution_id": task.ID,
		"plan_id":      req.Plan.ID,
		"status":       "submitted",
	})
}

// handleGatewayOrchestrationDAG retrieves DAG structure and execution status.
// GET /v1/gateway/orchestrate/:id/dag
func (s *Server) handleGatewayOrchestrationDAG(c *gin.Context) {
	executionID := c.Param("id")

	// Retrieve orchestration state from store
	state, exists := s.orchestrations.Get(executionID)
	if !exists {
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Orchestration not found: %s", executionID))
		return
	}

	state.mu.Lock()
	status := state.Status
	plan := state.Plan
	state.mu.Unlock()

	if plan == nil {
		c.JSON(http.StatusOK, gin.H{
			"execution_id": executionID,
			"status":       status,
			"dag": map[string]interface{}{
				"nodes": []interface{}{},
				"edges": []interface{}{},
			},
		})
		return
	}

	// Build nodes array with execution status
	nodes := make([]map[string]interface{}, 0, len(plan.Nodes))
	for _, node := range plan.Nodes {
		nodeData := map[string]interface{}{
			"id":        node.ID,
			"tool_name": node.ToolName,
			"state":     string(node.State),
		}

		if node.StartTime != nil {
			nodeData["start_time"] = node.StartTime.Format(time.RFC3339)
		}
		if node.EndTime != nil {
			nodeData["end_time"] = node.EndTime.Format(time.RFC3339)
			nodeData["duration_ms"] = node.EndTime.Sub(*node.StartTime).Milliseconds()
		}
		if node.Error != "" {
			nodeData["error"] = node.Error
		}
		if node.RetryCount > 0 {
			nodeData["retry_count"] = node.RetryCount
		}

		nodes = append(nodes, nodeData)
	}

	// Build edges array from dependencies
	edges := make([]map[string]interface{}, 0)
	for _, node := range plan.Nodes {
		for _, depID := range node.Dependencies {
			edges = append(edges, map[string]interface{}{
				"from": depID,
				"to":   node.ID,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"execution_id": executionID,
		"status":       status,
		"plan_id":      plan.ID,
		"created_at":   plan.CreatedAt.Format(time.RFC3339),
		"dag": map[string]interface{}{
			"nodes": nodes,
			"edges": edges,
		},
	})
}

// handleGatewayGetTrace retrieves trace details if OTEL is enabled.
// GET /v1/gateway/traces/:id
func (s *Server) handleGatewayGetTrace(c *gin.Context) {
	traceID := c.Param("id")

	if s.traceLogger == nil {
		s.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "Trace logger not available")
		return
	}

	// Retrieve trace metadata
	trace, err := s.traceLogger.GetTrace(c.Request.Context(), traceID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "not_found", fmt.Sprintf("Trace not found: %v", err))
		return
	}

	// Retrieve all spans for this trace
	spans, err := s.traceLogger.GetTraceSpans(c.Request.Context(), traceID)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "server_error", fmt.Sprintf("Failed to retrieve spans: %v", err))
		return
	}

	// Build spans response with relevant details
	spansResp := make([]map[string]interface{}, 0, len(spans))
	for _, span := range spans {
		spanData := map[string]interface{}{
			"span_id":    span.SpanID,
			"parent_id":  span.ParentID,
			"name":       span.Name,
			"start_time": span.StartTime.Format(time.RFC3339),
			"status":     span.Status,
		}

		if span.EndTime != nil {
			spanData["end_time"] = span.EndTime.Format(time.RFC3339)
			spanData["duration_ms"] = span.EndTime.Sub(span.StartTime).Milliseconds()
		}

		if span.Inputs != nil {
			spanData["inputs"] = span.Inputs
		}
		if span.Outputs != nil {
			spanData["outputs"] = span.Outputs
		}
		if span.Metadata != nil {
			spanData["metadata"] = span.Metadata
		}

		spansResp = append(spansResp, spanData)
	}

	// Build response
	response := gin.H{
		"trace_id":   trace.TraceID,
		"session_id": trace.SessionID,
		"start_time": trace.StartTime.Format(time.RFC3339),
		"status":     trace.Status,
		"spans":      spansResp,
		"span_count": len(spansResp),
	}

	if trace.EndTime != nil {
		response["end_time"] = trace.EndTime.Format(time.RFC3339)
		response["duration_ms"] = trace.EndTime.Sub(trace.StartTime).Milliseconds()
	}

	if trace.RootSpanID != "" {
		response["root_span_id"] = trace.RootSpanID
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func extractNamespace(toolName string) string {
	// Extract namespace from tool name (e.g., "composio.create_task" → "composio")
	for i, char := range toolName {
		if char == '.' || char == ':' {
			return toolName[:i]
		}
	}
	return "builtin"
}

func convertToOrchestrationPlan(gatewayPlan *gateway.ExecutionPlan) *orchestrationpkg.Plan {
	nodes := make([]*orchestrationpkg.PlanNode, 0, len(gatewayPlan.DAG))

	for _, invocation := range gatewayPlan.DAG {
		node := &orchestrationpkg.PlanNode{
			ID:           invocation.ID,
			ToolName:     invocation.ToolName,
			Parameters:   invocation.Input,
			Dependencies: invocation.DependsOn,
			State:        orchestrationpkg.NodeStatePending,
		}
		nodes = append(nodes, node)
	}

	return &orchestrationpkg.Plan{
		ID:        gatewayPlan.ID,
		TaskID:    gatewayPlan.Name, // Store gateway plan name in TaskID field
		Nodes:     nodes,
		CreatedAt: time.Now(),
		Version:   1,
		Metadata:  map[string]interface{}{"gateway_plan_name": gatewayPlan.Name},
	}
}
