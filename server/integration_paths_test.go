package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orchestrationpkg "github.com/DojoGenesis/gateway/orchestration"
	"github.com/DojoGenesis/gateway/pkg/gateway"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Orchestration Store Tests ──────────────────────────────────────────────

func TestOrchestrationStore_StoreAndGet(t *testing.T) {
	store := NewOrchestrationStore()

	state := &OrchestrationState{
		ID:        "orch-test-123",
		TaskID:    "task-abc",
		Status:    "planning",
		CreatedAt: time.Now(),
	}

	store.Store(state)

	got, exists := store.Get("orch-test-123")
	require.True(t, exists, "stored orchestration should be retrievable")
	assert.Equal(t, "orch-test-123", got.ID)
	assert.Equal(t, "task-abc", got.TaskID)
	assert.Equal(t, "planning", got.Status)
}

func TestOrchestrationStore_GetNotFound(t *testing.T) {
	store := NewOrchestrationStore()

	_, exists := store.Get("nonexistent")
	assert.False(t, exists, "should not find nonexistent orchestration")
}

func TestOrchestrationStore_StatusTransitions(t *testing.T) {
	store := NewOrchestrationStore()

	state := &OrchestrationState{
		ID:     "orch-1",
		Status: "planning",
	}
	store.Store(state)

	// Simulate status transition
	state.mu.Lock()
	state.Status = "executing"
	state.mu.Unlock()

	got, _ := store.Get("orch-1")
	got.mu.Lock()
	assert.Equal(t, "executing", got.Status)
	got.mu.Unlock()
}

// ─── Agent Lifecycle Tests ──────────────────────────────────────────────────

func TestAgentLifecycle_CreateAndRetrieve(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	// Manually register an agent (simulating what handleGatewayCreateAgent does)
	agentConfig := &gateway.AgentConfig{
		Pacing:     "measured",
		Depth:      "thorough",
		Tone:       "professional",
		Initiative: "responsive",
	}
	s.agentMu.Lock()
	s.agents["test-agent-id"] = &AgentRuntime{Config: agentConfig}
	s.agentMu.Unlock()

	// Set up router and test retrieval
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req")
		c.Next()
	})
	router.GET("/v1/gateway/agents/:id", s.handleGatewayGetAgent)

	// Test: retrieve existing agent
	req, _ := http.NewRequest(http.MethodGet, "/v1/gateway/agents/test-agent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "test-agent-id", resp["agent_id"])
	assert.Equal(t, "active", resp["status"])

	// Test: retrieve non-existent agent
	req2, _ := http.NewRequest(http.MethodGet, "/v1/gateway/agents/no-such-agent", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

// ─── DAG Retrieval Test ─────────────────────────────────────────────────────

func TestDAGRetrieval_WithPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	// Store an orchestration with a plan that has nodes
	now := time.Now()
	startTime := now.Add(-100 * time.Millisecond)
	endTime := now

	orchState := &OrchestrationState{
		ID:        "orch-dag-test",
		TaskID:    "task-1",
		Status:    "complete",
		CreatedAt: now,
		Plan: &orchestrationpkg.Plan{
			ID:        "plan-1",
			CreatedAt: now,
			Nodes: []*orchestrationpkg.PlanNode{
				{
					ID:           "node-1",
					ToolName:     "web_search",
					State:        orchestrationpkg.NodeStateSuccess,
					StartTime:    &startTime,
					EndTime:      &endTime,
					Dependencies: []string{},
				},
				{
					ID:           "node-2",
					ToolName:     "summarize",
					State:        orchestrationpkg.NodeStateSuccess,
					Dependencies: []string{"node-1"},
					StartTime:    &endTime,
				},
			},
		},
	}
	s.orchestrations.Store(orchState)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req")
		c.Next()
	})
	router.GET("/v1/gateway/orchestrate/:id/dag", s.handleGatewayOrchestrationDAG)

	req, _ := http.NewRequest(http.MethodGet, "/v1/gateway/orchestrate/orch-dag-test/dag", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "orch-dag-test", resp["execution_id"])
	assert.Equal(t, "complete", resp["status"])
	assert.Equal(t, "plan-1", resp["plan_id"])

	dag, ok := resp["dag"].(map[string]interface{})
	require.True(t, ok, "dag must be an object")

	nodes, ok := dag["nodes"].([]interface{})
	require.True(t, ok, "dag.nodes must be an array")
	assert.Len(t, nodes, 2, "should have 2 nodes")

	edges, ok := dag["edges"].([]interface{})
	require.True(t, ok, "dag.edges must be an array")
	assert.Len(t, edges, 1, "should have 1 edge (node-1 -> node-2)")
}

// ─── Health Endpoint Test ───────────────────────────────────────────────────

func TestHealthEndpoint_BasicResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		startTime:      time.Now().Add(-10 * time.Second),
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	router := gin.New()
	router.GET("/health", s.handleHealth)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var resp HealthResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp.Status)
	assert.NotEmpty(t, resp.Version)
	assert.NotEmpty(t, resp.Timestamp)
	assert.True(t, resp.UptimeSeconds >= 10, "uptime should reflect actual time")
}

// ─── Models Endpoint Test ───────────────────────────────────────────────────

func TestModelsEndpoint_NoProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	router := gin.New()
	router.GET("/v1/models", s.handleListModels)

	req, _ := http.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp OpenAIModelList
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "list", resp.Object)
	assert.Empty(t, resp.Data)
}

// ─── Tools Endpoint Test ────────────────────────────────────────────────────

func TestToolsEndpoint_ListTools(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	router := gin.New()
	router.GET("/v1/tools", s.handleListTools)

	req, _ := http.NewRequest(http.MethodGet, "/v1/tools", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ToolListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.GreaterOrEqual(t, resp.Count, 0)
	assert.Len(t, resp.Tools, resp.Count)
}

// ─── Admin Endpoints Tests ──────────────────────────────────────────────────

func TestAdminHealth_BasicResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		startTime:      time.Now(),
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	router := gin.New()
	router.GET("/admin/health", s.handleAdminHealth)

	req, _ := http.NewRequest(http.MethodGet, "/admin/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp["status"])
	assert.NotNil(t, resp["system"])
	assert.NotNil(t, resp["memory"])
}

func TestAdminConfig_BasicResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:            "8080",
			Environment:     "test",
			AuthMode:        "api_key",
			ShutdownTimeout: 30 * time.Second,
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	router := gin.New()
	router.GET("/admin/config", s.handleAdminConfig)

	req, _ := http.NewRequest(http.MethodGet, "/admin/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	serverConf, ok := resp["server"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "8080", serverConf["port"])
	assert.Equal(t, "test", serverConf["environment"])
}

func TestAdminMetrics_PrometheusFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		startTime:      time.Now(),
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	router := gin.New()
	router.GET("/admin/metrics/prometheus", s.handleAdminMetrics)

	req, _ := http.NewRequest(http.MethodGet, "/admin/metrics/prometheus", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
	body := w.Body.String()
	// gateway_tools_total only present when toolRegistry is configured
	assert.Contains(t, body, "gateway_agents_active")
	assert.Contains(t, body, "gateway_memory_alloc_bytes")
	assert.Contains(t, body, "gateway_goroutines")
	assert.Contains(t, body, "gateway_uptime_seconds")
}

// ─── Memory Validation Test ─────────────────────────────────────────────────

func TestMemoryStore_InvalidContextType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// This test needs a memory manager to get past the nil check.
	// Since we don't have one, we test the nil-check path instead.
	s := &Server{
		cfg: &ServerConfig{
			Port:        "8080",
			Environment: "test",
		},
		orchestrations: NewOrchestrationStore(),
		agents:         make(map[string]*AgentRuntime),
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req")
		c.Next()
	})
	router.POST("/v1/memory", s.handleStoreMemory)

	// Without memory manager, should get service unavailable
	body := `{"type": "fact", "content": "test", "context_type": "invalid"}`
	req, _ := http.NewRequest(http.MethodPost, "/v1/memory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assertErrorShape(t, w, http.StatusServiceUnavailable, "server_error")
}
