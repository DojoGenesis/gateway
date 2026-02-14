package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/agent"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func setupAgentTestDB(t *testing.T) *sql.DB {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test_agents.db"

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL,
		type TEXT NOT NULL CHECK(type IN ('primary', 'specialist', 'utility')),
		status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'inactive', 'experimental')),
		model_name TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS agent_capabilities (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		capability_type TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL,
		UNIQUE(agent_id, capability_type, name),
		FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func TestHandleListAgents_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	if err := am.SeedDefaultAgents(context.Background()); err != nil {
		t.Fatalf("Failed to seed agents: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents", h.ListAgents)

	req, _ := http.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var agents []agent.Agent
	err = json.Unmarshal(w.Body.Bytes(), &agents)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, len(agents), 2, "should return at least two agents (primary and mini)")

	var primaryAgent *agent.Agent
	var miniAgent *agent.Agent

	for i := range agents {
		ag := &agents[i]
		assert.NotEmpty(t, ag.ID, "agent ID should not be empty")
		assert.NotEmpty(t, ag.Name, "agent name should not be empty")
		assert.NotEmpty(t, ag.Description, "agent description should not be empty")
		assert.NotEmpty(t, ag.Type, "agent type should not be empty")
		assert.NotEmpty(t, ag.Status, "agent status should not be empty")

		if ag.ID == "primary_agent" {
			primaryAgent = ag
		} else if ag.ID == "mini_delegation_agent" {
			miniAgent = ag
		}
	}

	assert.NotNil(t, primaryAgent, "should have primary_agent")
	assert.NotNil(t, miniAgent, "should have mini_delegation_agent")

	assert.Equal(t, "Primary Agent", primaryAgent.Name)
	assert.Equal(t, "primary", primaryAgent.Type)
	assert.Equal(t, "active", primaryAgent.Status)

	assert.Equal(t, "Mini Delegation Agent", miniAgent.Name)
	assert.Equal(t, "utility", miniAgent.Type)
	assert.Equal(t, "active", miniAgent.Status)
}

func TestHandleListAgents_NotInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewAgentHandler(nil)

	router := gin.New()
	router.GET("/api/agents", h.ListAgents)

	req, _ := http.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "agent manager not initialized", response["error"])
}

func TestHandleGetAgent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	if err := am.SeedDefaultAgents(context.Background()); err != nil {
		t.Fatalf("Failed to seed agents: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents/:id", h.GetAgent)

	req, _ := http.NewRequest("GET", "/api/agents/primary_agent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var ag agent.Agent
	err = json.Unmarshal(w.Body.Bytes(), &ag)
	assert.NoError(t, err)

	assert.Equal(t, "primary_agent", ag.ID)
	assert.Equal(t, "Primary Agent", ag.Name)
	assert.Equal(t, "primary", ag.Type)
}

func TestHandleGetAgent_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents/:id", h.GetAgent)

	req, _ := http.NewRequest("GET", "/api/agents/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "agent not found", response["error"])
}

func TestHandleGetAgentCapabilities_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	if err := am.SeedDefaultAgents(context.Background()); err != nil {
		t.Fatalf("Failed to seed agents: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents/:id/capabilities", h.GetAgentCapabilities)

	req, _ := http.NewRequest("GET", "/api/agents/primary_agent/capabilities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var capabilities []agent.AgentCapability
	err = json.Unmarshal(w.Body.Bytes(), &capabilities)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, len(capabilities), 1, "should have at least one capability")

	hasFileRead := false
	for _, cap := range capabilities {
		assert.NotEmpty(t, cap.ID)
		assert.Equal(t, "primary_agent", cap.AgentID)
		assert.NotEmpty(t, cap.CapabilityType)
		assert.NotEmpty(t, cap.Name)

		if cap.Name == "file_read" {
			hasFileRead = true
		}
	}

	assert.True(t, hasFileRead, "primary agent should have file_read capability")
}

func TestHandleGetAgentCapabilities_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents/:id/capabilities", h.GetAgentCapabilities)

	req, _ := http.NewRequest("GET", "/api/agents/nonexistent/capabilities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSeedAgents_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.POST("/api/agents/seed", h.SeedAgents)

	req, _ := http.NewRequest("POST", "/api/agents/seed", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "default agents seeded successfully", response["message"])

	agents, err := am.ListAgents(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(agents), 2, "should have at least two seeded agents")
}

func TestHandleSeedAgents_Idempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.POST("/api/agents/seed", h.SeedAgents)

	req1, _ := http.NewRequest("POST", "/api/agents/seed", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2, _ := http.NewRequest("POST", "/api/agents/seed", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	agents, err := am.ListAgents(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(agents), "should still have only 2 agents after seeding twice")
}

func TestHandleListAgents_Pagination_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	for i := 1; i <= 25; i++ {
		_, err := am.RegisterAgent(context.Background(), agent.RegisterAgentRequest{
			ID:          "agent_" + strconv.Itoa(i),
			Name:        "Test Agent " + strconv.Itoa(i),
			Description: "Test agent for pagination",
			Type:        "utility",
			Status:      "active",
		})
		if err != nil {
			t.Fatalf("Failed to register test agent: %v", err)
		}
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents", h.ListAgents)

	req, _ := http.NewRequest("GET", "/api/agents?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response agent.PaginatedAgentsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, 10, len(response.Agents))
	assert.Equal(t, 25, response.Pagination.Total)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 10, response.Pagination.Limit)
	assert.Equal(t, 3, response.Pagination.TotalPages)
}

func TestHandleListAgents_Pagination_SecondPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	for i := 1; i <= 25; i++ {
		_, err := am.RegisterAgent(context.Background(), agent.RegisterAgentRequest{
			ID:          "agent_" + strconv.Itoa(i),
			Name:        "Test Agent " + strconv.Itoa(i),
			Description: "Test agent for pagination",
			Type:        "utility",
			Status:      "active",
		})
		if err != nil {
			t.Fatalf("Failed to register test agent: %v", err)
		}
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents", h.ListAgents)

	req, _ := http.NewRequest("GET", "/api/agents?page=2&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response agent.PaginatedAgentsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, 10, len(response.Agents))
	assert.Equal(t, 2, response.Pagination.Page)
}

func TestHandleListAgents_Pagination_InvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents", h.ListAgents)

	req, _ := http.NewRequest("GET", "/api/agents?page=invalid&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid page parameter", response["error"])
}

func TestHandleListAgents_Pagination_InvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents", h.ListAgents)

	req, _ := http.NewRequest("GET", "/api/agents?page=1&limit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid limit parameter", response["error"])
}

func TestHandleListAgents_Pagination_MaxLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupAgentTestDB(t)
	defer db.Close()

	am, err := agent.NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	for i := 1; i <= 150; i++ {
		_, err := am.RegisterAgent(context.Background(), agent.RegisterAgentRequest{
			ID:          "agent_" + strconv.Itoa(i),
			Name:        "Test Agent " + strconv.Itoa(i),
			Description: "Test agent for pagination",
			Type:        "utility",
			Status:      "active",
		})
		if err != nil {
			t.Fatalf("Failed to register test agent: %v", err)
		}
	}

	h := NewAgentHandler(am)

	router := gin.New()
	router.GET("/api/agents", h.ListAgents)

	req, _ := http.NewRequest("GET", "/api/agents?page=1&limit=200", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response agent.PaginatedAgentsResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, 100, len(response.Agents), "should enforce max limit of 100")
	assert.Equal(t, 100, response.Pagination.Limit)
}
