package mcp

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid config",
			path:    "testdata/mcp_servers.yaml",
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    "testdata/nonexistent.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg == nil {
				t.Error("LoadConfig() returned nil config without error")
			}
		})
	}
}

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "expand single var",
			input: "key: ${TEST_VAR}",
			want:  "key: test_value",
		},
		{
			name:  "expand multiple vars",
			input: "${TEST_VAR} and ${TEST_VAR}",
			want:  "test_value and test_value",
		},
		{
			name:  "no expansion",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "undefined var",
			input: "${UNDEFINED_VAR}",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMCPServerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPServerConfig
		wantErr bool
	}{
		{
			name: "valid stdio config",
			config: MCPServerConfig{
				ID:              "test",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "python",
					Args:    []string{"-m", "test"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			config: MCPServerConfig{
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "python",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ID format (with spaces)",
			config: MCPServerConfig{
				ID:              "test server",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "python",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid transport type",
			config: MCPServerConfig{
				ID:              "test",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "invalid",
					Command: "python",
				},
			},
			wantErr: true,
		},
		{
			name: "empty command for stdio",
			config: MCPServerConfig{
				ID:              "test",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type: "stdio",
				},
			},
			wantErr: true,
		},
		{
			name: "empty namespace prefix",
			config: MCPServerConfig{
				ID: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "python",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MCPServerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsToolAllowed(t *testing.T) {
	tests := []struct {
		name      string
		config    MCPServerConfig
		toolName  string
		wantAllow bool
	}{
		{
			name: "empty allowlist allows all",
			config: MCPServerConfig{
				Tools: ToolFilterConfig{
					Allowlist: []string{},
					Blocklist: []string{},
				},
			},
			toolName:  "any_tool",
			wantAllow: true,
		},
		{
			name: "blocklist blocks tool",
			config: MCPServerConfig{
				Tools: ToolFilterConfig{
					Allowlist: []string{},
					Blocklist: []string{"blocked_tool"},
				},
			},
			toolName:  "blocked_tool",
			wantAllow: false,
		},
		{
			name: "allowlist allows tool",
			config: MCPServerConfig{
				Tools: ToolFilterConfig{
					Allowlist: []string{"allowed_tool"},
					Blocklist: []string{},
				},
			},
			toolName:  "allowed_tool",
			wantAllow: true,
		},
		{
			name: "allowlist blocks unlisted tool",
			config: MCPServerConfig{
				Tools: ToolFilterConfig{
					Allowlist: []string{"allowed_tool"},
					Blocklist: []string{},
				},
			},
			toolName:  "other_tool",
			wantAllow: false,
		},
		{
			name: "blocklist takes precedence",
			config: MCPServerConfig{
				Tools: ToolFilterConfig{
					Allowlist: []string{"*"},
					Blocklist: []string{"blocked_tool"},
				},
			},
			toolName:  "blocked_tool",
			wantAllow: false,
		},
		{
			name: "wildcard in allowlist",
			config: MCPServerConfig{
				Tools: ToolFilterConfig{
					Allowlist: []string{"test_*"},
					Blocklist: []string{},
				},
			},
			toolName:  "test_something",
			wantAllow: true,
		},
		{
			name: "wildcard in blocklist",
			config: MCPServerConfig{
				Tools: ToolFilterConfig{
					Allowlist: []string{},
					Blocklist: []string{"dangerous_*"},
				},
			},
			toolName:  "dangerous_action",
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsToolAllowed(tt.toolName)
			if got != tt.wantAllow {
				t.Errorf("IsToolAllowed(%s) = %v, want %v", tt.toolName, got, tt.wantAllow)
			}
		})
	}
}

func TestMCPConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: MCPConfig{
				Servers: []MCPServerConfig{
					{
						ID:              "test",
						NamespacePrefix: "test",
						Transport: TransportConfig{
							Type:    "stdio",
							Command: "python",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty servers",
			config: MCPConfig{
				Servers: []MCPServerConfig{},
			},
			wantErr: true,
		},
		{
			name: "duplicate server IDs",
			config: MCPConfig{
				Servers: []MCPServerConfig{
					{
						ID:              "test",
						NamespacePrefix: "test",
						Transport: TransportConfig{
							Type:    "stdio",
							Command: "python",
						},
					},
					{
						ID:              "test",
						NamespacePrefix: "test2",
						Transport: TransportConfig{
							Type:    "stdio",
							Command: "node",
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MCPConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
