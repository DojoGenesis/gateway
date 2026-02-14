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
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "tool registry not available",
		})
		return
	}

	tools, err := s.toolRegistry.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to list tools: %v", err),
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	if s.agentInitializer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "agent initializer not available",
		})
		return
	}

	// Initialize agent configuration from disposition
	agentConfig, err := s.agentInitializer.Initialize(c.Request.Context(), req.WorkspaceRoot, req.ActiveMode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to initialize agent: %v", err),
		})
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
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("agent not found: %s", agentID),
		})
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	s.agentMu.RLock()
	agentConfig, exists := s.agents[agentID]
	s.agentMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("agent not found: %s", agentID),
		})
		return
	}

	// TODO: Integrate with orchestration engine for actual chat processing
	// For now, return a placeholder response with agent disposition info

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"response": fmt.Sprintf("Agent responding with %s pacing and %s depth", agentConfig.Pacing, agentConfig.Depth),
		"message":  req.Message,
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
		return
	}

	if s.orchestrationEngine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "orchestration engine not available",
		})
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

	// TODO: Implement actual DAG retrieval from orchestration store
	// For now, return a placeholder response

	c.JSON(http.StatusOK, gin.H{
		"execution_id": executionID,
		"status":       "running",
		"dag": map[string]interface{}{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		},
	})
}

// handleGatewayGetTrace retrieves trace details if OTEL is enabled.
// GET /v1/gateway/traces/:id
func (s *Server) handleGatewayGetTrace(c *gin.Context) {
	traceID := c.Param("id")

	if s.traceLogger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "trace logger not available",
		})
		return
	}

	// TODO: Implement actual trace retrieval
	// For now, return a placeholder response

	c.JSON(http.StatusOK, gin.H{
		"trace_id": traceID,
		"spans":    []interface{}{},
	})
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
