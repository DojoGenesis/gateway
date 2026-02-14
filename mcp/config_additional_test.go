package mcp

import (
	"os"
	"testing"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		pattern string
		want    bool
	}{
		{
			name:    "exact match",
			s:       "tool_name",
			pattern: "tool_name",
			want:    true,
		},
		{
			name:    "exact mismatch",
			s:       "tool_name",
			pattern: "other_tool",
			want:    false,
		},
		{
			name:    "wildcard asterisk only",
			s:       "anything",
			pattern: "*",
			want:    true,
		},
		{
			name:    "wildcard prefix",
			s:       "admin_delete",
			pattern: "admin_*",
			want:    true,
		},
		{
			name:    "wildcard suffix",
			s:       "delete_user",
			pattern: "*_user",
			want:    true,
		},
		{
			name:    "wildcard middle",
			s:       "admin_delete_user",
			pattern: "admin_*_user",
			want:    true,
		},
		{
			name:    "wildcard no match",
			s:       "user_create",
			pattern: "admin_*",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.s, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMCPServerConfigValidate_AllFields(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid full config",
			config: MCPServerConfig{
				ID:              "test",
				DisplayName:     "Test Server",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "python",
					Args:    []string{"-m", "test"},
				},
				HealthCheck: HealthCheckConfig{
					Enabled:     true,
					IntervalSec: 30,
				},
				Timeouts: TimeoutConfig{
					ToolDefault: 60,
				},
				RetryPolicy: RetryPolicy{
					MaxAttempts: 3,
				},
			},
			wantErr: false,
		},
		{
			name: "zero values allowed for optional fields",
			config: MCPServerConfig{
				ID:              "test",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type:    "stdio",
					Command: "python",
				},
				HealthCheck: HealthCheckConfig{
					IntervalSec: 0,
				},
				Timeouts: TimeoutConfig{
					ToolDefault: 0,
				},
				RetryPolicy: RetryPolicy{
					MaxAttempts: 0,
				},
			},
			wantErr: false,
		},
		{
			name: "valid with SSE transport",
			config: MCPServerConfig{
				ID:              "test",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type: "sse",
					URL:  "https://example.com/sse",
				},
			},
			wantErr: false,
		},
		{
			name: "empty URL for SSE transport",
			config: MCPServerConfig{
				ID:              "test",
				NamespacePrefix: "test",
				Transport: TransportConfig{
					Type: "sse",
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

func TestExpandEnvVars_ComplexCases(t *testing.T) {
	// Set up test environment variables
	os.Setenv("VAR1", "value1")
	os.Setenv("VAR2", "value2")
	os.Setenv("PATH_VAR", "/usr/local/bin")
	defer func() {
		os.Unsetenv("VAR1")
		os.Unsetenv("VAR2")
		os.Unsetenv("PATH_VAR")
	}()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no variables",
			input: "plain text with no variables",
			want:  "plain text with no variables",
		},
		{
			name:  "single variable",
			input: "value is ${VAR1}",
			want:  "value is value1",
		},
		{
			name:  "multiple variables",
			input: "${VAR1} and ${VAR2}",
			want:  "value1 and value2",
		},
		{
			name:  "variable at start",
			input: "${VAR1} is first",
			want:  "value1 is first",
		},
		{
			name:  "variable at end",
			input: "last is ${VAR2}",
			want:  "last is value2",
		},
		{
			name:  "path variable",
			input: "PATH=${PATH_VAR}",
			want:  "PATH=/usr/local/bin",
		},
		{
			name:  "undefined variable becomes empty",
			input: "value is ${UNDEFINED}",
			want:  "value is ",
		},
		{
			name:  "mixed defined and undefined",
			input: "${VAR1} and ${UNDEFINED} and ${VAR2}",
			want:  "value1 and  and value2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create a temporary invalid YAML file
	tmpfile, err := os.CreateTemp("", "invalid-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	invalidYAML := `
version: "1.0"
mcp:
  servers:
    - id: test
      invalid: [this is not closed
`
	if _, err := tmpfile.Write([]byte(invalidYAML)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	_, err = LoadConfig(tmpfile.Name())
	if err == nil {
		t.Error("LoadConfig() should return error for invalid YAML")
	}
}

func TestLoadConfig_ValidationFails(t *testing.T) {
	// Create a temporary YAML file with invalid config
	tmpfile, err := os.CreateTemp("", "invalid-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	invalidConfig := `
version: "1.0"
mcp:
  servers:
    - id: ""
      namespace_prefix: "test"
      transport:
        type: stdio
        command: echo
`
	if _, err := tmpfile.Write([]byte(invalidConfig)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	_, err = LoadConfig(tmpfile.Name())
	if err == nil {
		t.Error("LoadConfig() should return error for invalid config (empty name)")
	}
}

func TestMCPConfigValidate_ServerValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  MCPConfig
		wantErr bool
	}{
		{
			name: "valid multiple servers",
			config: MCPConfig{
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
							Command: "cat",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "server with invalid config",
			config: MCPConfig{
				Servers: []MCPServerConfig{
					{
						ID:              "server1",
						NamespacePrefix: "s1",
						Transport: TransportConfig{
							Type:    "invalid_transport",
							Command: "echo",
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
