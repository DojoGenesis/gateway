package agent

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	dbPath := "./test_registry.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
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
		updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS agent_capabilities (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		capability_type TEXT NOT NULL CHECK(capability_type IN ('tool', 'skill', 'model')),
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
		UNIQUE(agent_id, capability_type, name)
	);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		os.Remove(dbPath)
		t.Fatalf("Failed to create test schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func TestRegistryNewAgentManager(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	manager, err := NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager to be non-nil")
	}

	if manager.db != db {
		t.Fatal("Expected manager.db to match provided database")
	}
}

func TestRegistryRegisterAgent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	manager, err := NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		req := RegisterAgentRequest{
			ID:          "test_agent_1",
			Name:        "Test Agent",
			Description: "A test agent for unit testing",
			Type:        "primary",
			Status:      "active",
			ModelName:   "gpt-4o",
			Capabilities: []AgentCapability{
				{CapabilityType: "tool", Name: "file_read", Description: "Read files"},
				{CapabilityType: "skill", Name: "coding", Description: "Code generation"},
			},
		}

		agent, err := manager.RegisterAgent(ctx, req)
		if err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}

		if agent.ID != req.ID {
			t.Errorf("Expected ID %s, got %s", req.ID, agent.ID)
		}
		if agent.Name != req.Name {
			t.Errorf("Expected Name %s, got %s", req.Name, agent.Name)
		}
		if agent.Type != req.Type {
			t.Errorf("Expected Type %s, got %s", req.Type, agent.Type)
		}
		if agent.Status != req.Status {
			t.Errorf("Expected Status %s, got %s", req.Status, agent.Status)
		}
		if agent.ModelName != req.ModelName {
			t.Errorf("Expected ModelName %s, got %s", req.ModelName, agent.ModelName)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		req := RegisterAgentRequest{
			ID:          "test_agent_2",
			Name:        "",
			Description: "Test",
			Type:        "primary",
		}

		_, err := manager.RegisterAgent(ctx, req)
		if err == nil {
			t.Error("Expected error for missing name")
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		req := RegisterAgentRequest{
			ID:          "",
			Name:        "Test Agent",
			Description: "Test",
			Type:        "primary",
		}

		_, err := manager.RegisterAgent(ctx, req)
		if err == nil {
			t.Error("Expected error for missing ID")
		}
	})

	t.Run("missing description", func(t *testing.T) {
		req := RegisterAgentRequest{
			ID:          "test_agent_3",
			Name:        "Test Agent",
			Description: "",
			Type:        "primary",
		}

		_, err := manager.RegisterAgent(ctx, req)
		if err == nil {
			t.Error("Expected error for missing description")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		req := RegisterAgentRequest{
			ID:          "test_agent_4",
			Name:        "Test Agent",
			Description: "Test",
			Type:        "invalid",
		}

		_, err := manager.RegisterAgent(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid type")
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		req := RegisterAgentRequest{
			ID:          "test_agent_5",
			Name:        "Test Agent",
			Description: "Test",
			Type:        "primary",
			Status:      "invalid",
		}

		_, err := manager.RegisterAgent(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid status")
		}
	})

	t.Run("default status", func(t *testing.T) {
		req := RegisterAgentRequest{
			ID:          "test_agent_6",
			Name:        "Test Agent 6",
			Description: "Test with default status",
			Type:        "utility",
		}

		agent, err := manager.RegisterAgent(ctx, req)
		if err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}

		if agent.Status != "active" {
			t.Errorf("Expected default status 'active', got %s", agent.Status)
		}
	})

	t.Run("idempotent registration", func(t *testing.T) {
		req := RegisterAgentRequest{
			ID:          "test_agent_7",
			Name:        "Test Agent 7",
			Description: "Original description",
			Type:        "primary",
			Status:      "active",
		}

		agent1, err := manager.RegisterAgent(ctx, req)
		if err != nil {
			t.Fatalf("Failed first registration: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		req.Description = "Updated description"
		agent2, err := manager.RegisterAgent(ctx, req)
		if err != nil {
			t.Fatalf("Failed second registration: %v", err)
		}

		if agent1.ID != agent2.ID {
			t.Error("Expected same agent ID on re-registration")
		}

		retrieved, err := manager.GetAgent(ctx, agent2.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve agent: %v", err)
		}

		if retrieved.Description != "Updated description" {
			t.Errorf("Expected updated description, got %s", retrieved.Description)
		}
	})
}

func TestRegistryGetAgent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	manager, err := NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	ctx := context.Background()

	req := RegisterAgentRequest{
		ID:          "get_test_agent",
		Name:        "Get Test Agent",
		Description: "Agent for get testing",
		Type:        "specialist",
		Status:      "active",
		ModelName:   "gpt-4o-mini",
	}

	_, err = manager.RegisterAgent(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	t.Run("successful retrieval", func(t *testing.T) {
		agent, err := manager.GetAgent(ctx, "get_test_agent")
		if err != nil {
			t.Fatalf("Failed to get agent: %v", err)
		}

		if agent.ID != req.ID {
			t.Errorf("Expected ID %s, got %s", req.ID, agent.ID)
		}
		if agent.Name != req.Name {
			t.Errorf("Expected Name %s, got %s", req.Name, agent.Name)
		}
	})

	t.Run("agent not found", func(t *testing.T) {
		_, err := manager.GetAgent(ctx, "nonexistent_agent")
		if err == nil {
			t.Error("Expected error for nonexistent agent")
		}
	})
}

func TestRegistryListAgents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	manager, err := NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	ctx := context.Background()

	agents := []RegisterAgentRequest{
		{
			ID:          "list_agent_1",
			Name:        "List Agent 1",
			Description: "First test agent",
			Type:        "primary",
			Status:      "active",
		},
		{
			ID:          "list_agent_2",
			Name:        "List Agent 2",
			Description: "Second test agent",
			Type:        "utility",
			Status:      "inactive",
		},
		{
			ID:          "list_agent_3",
			Name:        "List Agent 3",
			Description: "Third test agent",
			Type:        "specialist",
			Status:      "experimental",
		},
	}

	for _, agentReq := range agents {
		_, err := manager.RegisterAgent(ctx, agentReq)
		if err != nil {
			t.Fatalf("Failed to register agent %s: %v", agentReq.ID, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("list all agents", func(t *testing.T) {
		retrieved, err := manager.ListAgents(ctx)
		if err != nil {
			t.Fatalf("Failed to list agents: %v", err)
		}

		if len(retrieved) < 3 {
			t.Errorf("Expected at least 3 agents, got %d", len(retrieved))
		}

		foundIDs := make(map[string]bool)
		for _, agent := range retrieved {
			foundIDs[agent.ID] = true
		}

		for _, agentReq := range agents {
			if !foundIDs[agentReq.ID] {
				t.Errorf("Expected to find agent %s in list", agentReq.ID)
			}
		}
	})
}

func TestRegistryGetAgentCapabilities(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	manager, err := NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	ctx := context.Background()

	req := RegisterAgentRequest{
		ID:          "cap_test_agent",
		Name:        "Capability Test Agent",
		Description: "Agent for testing capabilities",
		Type:        "primary",
		Status:      "active",
		Capabilities: []AgentCapability{
			{CapabilityType: "tool", Name: "file_read"},
			{CapabilityType: "tool", Name: "file_write"},
			{CapabilityType: "skill", Name: "coding"},
			{CapabilityType: "model", Name: "gpt-4o"},
		},
	}

	_, err = manager.RegisterAgent(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	t.Run("get capabilities", func(t *testing.T) {
		caps, err := manager.GetAgentCapabilities(ctx, "cap_test_agent")
		if err != nil {
			t.Fatalf("Failed to get capabilities: %v", err)
		}

		if len(caps) != 4 {
			t.Errorf("Expected 4 capabilities, got %d", len(caps))
		}

		capNames := make(map[string]bool)
		for _, cap := range caps {
			capNames[cap.Name] = true
			if cap.AgentID != "cap_test_agent" {
				t.Errorf("Expected AgentID 'cap_test_agent', got %s", cap.AgentID)
			}
		}

		expectedNames := []string{"file_read", "file_write", "coding", "gpt-4o"}
		for _, name := range expectedNames {
			if !capNames[name] {
				t.Errorf("Expected to find capability %s", name)
			}
		}
	})

	t.Run("agent not found", func(t *testing.T) {
		_, err := manager.GetAgentCapabilities(ctx, "nonexistent_agent")
		if err == nil {
			t.Error("Expected error for nonexistent agent")
		}
	})

	t.Run("agent with no capabilities", func(t *testing.T) {
		emptyReq := RegisterAgentRequest{
			ID:          "empty_agent",
			Name:        "Empty Agent",
			Description: "Agent with no capabilities",
			Type:        "utility",
			Status:      "active",
		}

		_, err := manager.RegisterAgent(ctx, emptyReq)
		if err != nil {
			t.Fatalf("Failed to register agent: %v", err)
		}

		caps, err := manager.GetAgentCapabilities(ctx, "empty_agent")
		if err != nil {
			t.Fatalf("Failed to get capabilities: %v", err)
		}

		if len(caps) != 0 {
			t.Errorf("Expected 0 capabilities, got %d", len(caps))
		}
	})
}

func TestRegistrySeedDefaultAgents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	manager, err := NewAgentManager(db)
	if err != nil {
		t.Fatalf("Failed to create agent manager: %v", err)
	}

	ctx := context.Background()

	t.Run("seed default agents", func(t *testing.T) {
		err := manager.SeedDefaultAgents(ctx)
		if err != nil {
			t.Fatalf("Failed to seed default agents: %v", err)
		}

		agents, err := manager.ListAgents(ctx)
		if err != nil {
			t.Fatalf("Failed to list agents: %v", err)
		}

		if len(agents) < 2 {
			t.Errorf("Expected at least 2 agents after seeding, got %d", len(agents))
		}

		foundPrimary := false
		foundMini := false
		for _, agent := range agents {
			if agent.ID == "primary_agent" {
				foundPrimary = true
				if agent.Type != "primary" {
					t.Error("Expected primary_agent to have type 'primary'")
				}
			}
			if agent.ID == "mini_delegation_agent" {
				foundMini = true
				if agent.Type != "utility" {
					t.Error("Expected mini_delegation_agent to have type 'utility'")
				}
			}
		}

		if !foundPrimary {
			t.Error("Expected to find primary_agent after seeding")
		}
		if !foundMini {
			t.Error("Expected to find mini_delegation_agent after seeding")
		}
	})

	t.Run("seeding is idempotent", func(t *testing.T) {
		err := manager.SeedDefaultAgents(ctx)
		if err != nil {
			t.Fatalf("Failed to seed default agents second time: %v", err)
		}

		agents, err := manager.ListAgents(ctx)
		if err != nil {
			t.Fatalf("Failed to list agents: %v", err)
		}

		primaryCount := 0
		miniCount := 0
		for _, agent := range agents {
			if agent.ID == "primary_agent" {
				primaryCount++
			}
			if agent.ID == "mini_delegation_agent" {
				miniCount++
			}
		}

		if primaryCount != 1 {
			t.Errorf("Expected exactly 1 primary_agent, got %d", primaryCount)
		}
		if miniCount != 1 {
			t.Errorf("Expected exactly 1 mini_delegation_agent, got %d", miniCount)
		}
	})

	t.Run("verify primary agent capabilities", func(t *testing.T) {
		caps, err := manager.GetAgentCapabilities(ctx, "primary_agent")
		if err != nil {
			t.Fatalf("Failed to get primary agent capabilities: %v", err)
		}

		if len(caps) == 0 {
			t.Error("Expected primary agent to have capabilities")
		}

		hasTools := false
		hasSkills := false
		hasModels := false
		for _, cap := range caps {
			switch cap.CapabilityType {
			case "tool":
				hasTools = true
			case "skill":
				hasSkills = true
			case "model":
				hasModels = true
			}
		}

		if !hasTools {
			t.Error("Expected primary agent to have tool capabilities")
		}
		if !hasSkills {
			t.Error("Expected primary agent to have skill capabilities")
		}
		if !hasModels {
			t.Error("Expected primary agent to have model capabilities")
		}
	})
}

func TestRegistryValidation(t *testing.T) {
	t.Run("validate agent type", func(t *testing.T) {
		tests := []struct {
			agentType string
			expectErr bool
		}{
			{"primary", false},
			{"specialist", false},
			{"utility", false},
			{"invalid", true},
			{"", true},
		}

		for _, tt := range tests {
			err := validateAgentType(tt.agentType)
			if (err != nil) != tt.expectErr {
				t.Errorf("validateAgentType(%s): expectErr=%v, got err=%v", tt.agentType, tt.expectErr, err)
			}
		}
	})

	t.Run("validate agent status", func(t *testing.T) {
		tests := []struct {
			status    string
			expectErr bool
		}{
			{"active", false},
			{"inactive", false},
			{"experimental", false},
			{"invalid", true},
			{"", true},
		}

		for _, tt := range tests {
			err := validateAgentStatus(tt.status)
			if (err != nil) != tt.expectErr {
				t.Errorf("validateAgentStatus(%s): expectErr=%v, got err=%v", tt.status, tt.expectErr, err)
			}
		}
	})

	t.Run("validate capability type", func(t *testing.T) {
		tests := []struct {
			capType   string
			expectErr bool
		}{
			{"tool", false},
			{"skill", false},
			{"model", false},
			{"invalid", true},
			{"", true},
		}

		for _, tt := range tests {
			err := validateCapabilityType(tt.capType)
			if (err != nil) != tt.expectErr {
				t.Errorf("validateCapabilityType(%s): expectErr=%v, got err=%v", tt.capType, tt.expectErr, err)
			}
		}
	})
}
