package mcp

import (
	"context"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

// mockToolRegistry is a simple in-memory implementation of gateway.ToolRegistry for testing.
type mockToolRegistry struct {
	tools map[string]*tools.ToolDefinition
}

func newMockToolRegistry() *mockToolRegistry {
	return &mockToolRegistry{
		tools: make(map[string]*tools.ToolDefinition),
	}
}

func (m *mockToolRegistry) Register(ctx context.Context, def *tools.ToolDefinition) error {
	if def == nil || def.Name == "" {
		return tools.RegisterTool(def)
	}
	m.tools[def.Name] = def
	return nil
}

func (m *mockToolRegistry) Get(ctx context.Context, name string) (*tools.ToolDefinition, error) {
	if tool, exists := m.tools[name]; exists {
		return tool, nil
	}
	return tools.GetTool(name)
}

func (m *mockToolRegistry) List(ctx context.Context) ([]*tools.ToolDefinition, error) {
	result := make([]*tools.ToolDefinition, 0, len(m.tools))
	for _, tool := range m.tools {
		result = append(result, tool)
	}
	return result, nil
}

func (m *mockToolRegistry) ListByNamespace(ctx context.Context, prefix string) ([]*tools.ToolDefinition, error) {
	result := make([]*tools.ToolDefinition, 0)
	for name, tool := range m.tools {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			result = append(result, tool)
		}
	}
	return result, nil
}

func TestNewMCPHostManager(t *testing.T) {
	registry := newMockToolRegistry()

	tests := []struct {
		name     string
		config   *MCPConfig
		registry *mockToolRegistry
		wantErr  bool
	}{
		{
			name: "valid config",
			config: &MCPConfig{
				Servers: []MCPServerConfig{
					{
						ID:              "test",
						NamespacePrefix: "test",
						Transport: TransportConfig{
							Type:    "stdio",
							Command: "echo",
						},
					},
				},
			},
			registry: registry,
			wantErr:  false,
		},
		{
			name:     "nil config",
			config:   nil,
			registry: registry,
			wantErr:  true,
		},
		{
			name: "nil registry",
			config: &MCPConfig{
				Servers: []MCPServerConfig{
					{
						ID:              "test",
						NamespacePrefix: "test",
						Transport: TransportConfig{
							Type:    "stdio",
							Command: "echo",
						},
					},
				},
			},
			registry: nil,
			wantErr:  true,
		},
		{
			name: "invalid config",
			config: &MCPConfig{
				Servers: []MCPServerConfig{},
			},
			registry: registry,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reg gateway.ToolRegistry
			if tt.registry != nil {
				reg = tt.registry
			}
			manager, err := NewMCPHostManager(tt.config, reg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMCPHostManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && manager == nil {
				t.Error("NewMCPHostManager() returned nil without error")
			}
		})
	}
}

func TestMCPHostManager_Status(t *testing.T) {
	registry := newMockToolRegistry()
	config := &MCPConfig{
		Servers: []MCPServerConfig{
			{
				ID:              "test",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "echo",
				},
			},
		},
	}

	manager, err := NewMCPHostManager(config, registry)
	if err != nil {
		t.Fatalf("NewMCPHostManager() failed: %v", err)
	}

	// Get status before starting
	status := manager.Status()
	if status == nil {
		t.Error("Status() returned nil")
	}

	// Status should be empty since we haven't started
	if len(status) != 0 {
		t.Errorf("Status() returned %d servers, want 0", len(status))
	}
}

func TestMCPHostManager_StopWithoutStart(t *testing.T) {
	registry := newMockToolRegistry()
	config := &MCPConfig{
		Servers: []MCPServerConfig{
			{
				ID:              "test",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "echo",
				},
			},
		},
	}

	manager, err := NewMCPHostManager(config, registry)
	if err != nil {
		t.Fatalf("NewMCPHostManager() failed: %v", err)
	}

	// Stop without starting should not panic
	ctx := context.Background()
	err = manager.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

func TestServerStatus(t *testing.T) {
	status := ServerStatus{
		Name:      "test",
		Connected: true,
		ToolCount: 5,
		LastError: "",
	}

	if status.Name != "test" {
		t.Errorf("ServerStatus.Name = %v, want test", status.Name)
	}

	if !status.Connected {
		t.Error("ServerStatus.Connected = false, want true")
	}

	if status.ToolCount != 5 {
		t.Errorf("ServerStatus.ToolCount = %v, want 5", status.ToolCount)
	}
}

func TestMCPHostManager_GracefulDegradation(t *testing.T) {
	// This test verifies that the manager can handle server connection failures
	// gracefully without crashing
	registry := newMockToolRegistry()
	config := &MCPConfig{
		Servers: []MCPServerConfig{
			{
				ID:              "nonexistent",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "nonexistent_command_12345",
				},
			},
		},
	}

	manager, err := NewMCPHostManager(config, registry)
	if err != nil {
		t.Fatalf("NewMCPHostManager() failed: %v", err)
	}

	// Start should not panic even if server fails to connect
	ctx := context.Background()
	err = manager.Start(ctx)

	// Start returns nil even if individual servers fail (graceful degradation)
	if err != nil {
		t.Errorf("Start() returned error: %v (should handle failures gracefully)", err)
	}

	// Clean up
	manager.Stop(ctx)
}
