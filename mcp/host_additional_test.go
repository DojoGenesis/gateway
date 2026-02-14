package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestMCPHostManager_getNamespacePrefix(t *testing.T) {
	registry := newMockToolRegistry()
	config := &MCPConfig{
		Servers: []MCPServerConfig{
			{
				ID:              "test1",
				NamespacePrefix: "test1",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "echo",
				},
			},
			{
				ID:              "test2",
				NamespacePrefix: "test2",
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

	tests := []struct {
		name       string
		serverName string
		want       string
	}{
		{
			name:       "existing server test1",
			serverName: "test1",
			want:       "test1",
		},
		{
			name:       "existing server test2",
			serverName: "test2",
			want:       "test2",
		},
		{
			name:       "non-existent server",
			serverName: "nonexistent",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.getNamespacePrefix(tt.serverName)
			if got != tt.want {
				t.Errorf("getNamespacePrefix(%s) = %v, want %v", tt.serverName, got, tt.want)
			}
		})
	}
}

func TestMCPHostManager_Status_WithMultipleServers(t *testing.T) {
	registry := newMockToolRegistry()
	config := &MCPConfig{
		Servers: []MCPServerConfig{
			{
				ID:              "server1",
				NamespacePrefix: "s1",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "echo",
				},
			},
			{
				ID:              "server2",
				NamespacePrefix: "s2",
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

	status := manager.Status()

	// Status should be empty since no servers are connected
	if len(status) != 0 {
		t.Errorf("Status() returned %d servers, want 0 (no servers started)", len(status))
	}
}

func TestMCPServerConnection_GetName(t *testing.T) {
	config := MCPServerConfig{
		ID:              "test_server",
		NamespacePrefix: "test",
		Transport: TransportConfig{
			Type:    "stdio",
			Command: "echo",
		},
	}

	conn, err := NewMCPServerConnection("test_server", config)
	if err != nil {
		t.Fatalf("NewMCPServerConnection() failed: %v", err)
	}

	if name := conn.GetName(); name != "test_server" {
		t.Errorf("GetName() = %v, want test_server", name)
	}
}

func TestMCPServerConfig_IsToolAllowed_EmptyAllowlist(t *testing.T) {
	config := MCPServerConfig{
		Tools: ToolFilterConfig{
			Allowlist: []string{},
			Blocklist: []string{},
		},
	}

	// Empty allowlist should allow all tools
	if !config.IsToolAllowed("any_tool") {
		t.Error("IsToolAllowed() should return true for empty allowlist")
	}
}

func TestMCPServerConfig_IsToolAllowed_AllowlistWithBlocklist(t *testing.T) {
	config := MCPServerConfig{
		Tools: ToolFilterConfig{
			Allowlist: []string{"tool1", "tool2"},
			Blocklist: []string{"tool1"},
		},
	}

	// Blocklist takes precedence
	if config.IsToolAllowed("tool1") {
		t.Error("IsToolAllowed() should return false for blocked tool even if in allowlist")
	}

	// tool2 is in allowlist and not blocked
	if !config.IsToolAllowed("tool2") {
		t.Error("IsToolAllowed() should return true for tool2")
	}

	// tool3 is not in allowlist
	if config.IsToolAllowed("tool3") {
		t.Error("IsToolAllowed() should return false for tool not in allowlist")
	}
}

func TestMCPServerConfig_IsToolAllowed_WildcardCombinations(t *testing.T) {
	tests := []struct {
		name      string
		allowlist []string
		blocklist []string
		toolName  string
		want      bool
	}{
		{
			name:      "wildcard allowlist",
			allowlist: []string{"*"},
			blocklist: []string{},
			toolName:  "anything",
			want:      true,
		},
		{
			name:      "wildcard blocklist",
			allowlist: []string{},
			blocklist: []string{"*"},
			toolName:  "anything",
			want:      false,
		},
		{
			name:      "prefix wildcard in allowlist",
			allowlist: []string{"admin_*"},
			blocklist: []string{},
			toolName:  "admin_delete",
			want:      true,
		},
		{
			name:      "prefix wildcard mismatch",
			allowlist: []string{"admin_*"},
			blocklist: []string{},
			toolName:  "user_delete",
			want:      false,
		},
		{
			name:      "suffix wildcard in blocklist",
			allowlist: []string{},
			blocklist: []string{"*_delete"},
			toolName:  "user_delete",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := MCPServerConfig{
				Tools: ToolFilterConfig{
					Allowlist: tt.allowlist,
					Blocklist: tt.blocklist,
				},
			}

			got := config.IsToolAllowed(tt.toolName)
			if got != tt.want {
				t.Errorf("IsToolAllowed(%s) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestConvertToolResult_WithContent(t *testing.T) {
	tests := []struct {
		name   string
		result *mcp.CallToolResult
		want   int // number of expected keys
	}{
		{
			name:   "nil result",
			result: nil,
			want:   0,
		},
		{
			name: "with empty content",
			result: &mcp.CallToolResult{
				Content: []mcp.Content{},
				IsError: false,
			},
			want: 1, // isError only (empty content not added)
		},
		{
			name: "with error",
			result: &mcp.CallToolResult{
				IsError: true,
			},
			want: 1, // isError only
		},
		{
			name: "with structured content",
			result: &mcp.CallToolResult{
				StructuredContent: map[string]interface{}{
					"key": "value",
				},
				IsError: false,
			},
			want: 2, // isError and structuredContent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToolResult(tt.result)
			if len(got) != tt.want {
				t.Errorf("convertToolResult() returned %d keys, want %d", len(got), tt.want)
			}
		})
	}
}
