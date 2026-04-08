package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	cfg := loadDefaults()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, []string{"http://localhost:3000", "http://localhost:3003"}, cfg.AllowedOrigins)
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, "plugins", cfg.PluginDir)
	assert.Equal(t, "auto", cfg.Routing.DefaultProvider)
	assert.Equal(t, "auto", cfg.Routing.GuestProvider)
	assert.Equal(t, "auto", cfg.Routing.AuthenticatedProvider)
	assert.Equal(t, 50000, cfg.Budget.QueryLimit)
	assert.Equal(t, 200000, cfg.Budget.SessionLimit)
	assert.Equal(t, 2000000, cfg.Budget.MonthlyLimit)
}

func TestLoadFromYAML(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
port: "9000"
environment: "test"
plugin_dir: "test_plugins"
allowed_origins:
  - "http://test.com"

providers:
  - name: test-provider
    enabled: true
    priority: 1
    plugin_path: plugins/test
    config:
      api_key: "test-key"

routing:
  default_provider: test-provider
  guest_provider: test-provider
  authenticated_provider: test-provider

budget:
  query_limit: 10000
  session_limit: 50000
  monthly_limit: 1000000
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg := loadDefaults()
	err = cfg.loadFromYAML(configPath)
	require.NoError(t, err)

	assert.Equal(t, "9000", cfg.Port)
	assert.Equal(t, "test", cfg.Environment)
	assert.Equal(t, "test_plugins", cfg.PluginDir)
	assert.Equal(t, []string{"http://test.com"}, cfg.AllowedOrigins)
	assert.Len(t, cfg.Providers, 1)
	assert.Equal(t, "test-provider", cfg.Providers[0].Name)
	assert.Equal(t, true, cfg.Providers[0].Enabled)
	assert.Equal(t, 1, cfg.Providers[0].Priority)
	assert.Equal(t, "plugins/test", cfg.Providers[0].PluginPath)
	assert.Equal(t, "test-provider", cfg.Routing.DefaultProvider)
	assert.Equal(t, 10000, cfg.Budget.QueryLimit)
}

func TestExpandEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "expand ${VAR}",
			input:    "api_key: ${API_KEY}",
			envVars:  map[string]string{"API_KEY": "secret123"},
			expected: "api_key: secret123",
		},
		{
			name:     "expand $VAR",
			input:    "path: $HOME/data",
			envVars:  map[string]string{"HOME": "/home/user"},
			expected: "path: /home/user/data",
		},
		{
			name:     "multiple variables",
			input:    "url: ${PROTOCOL}://${HOST}:${PORT}",
			envVars:  map[string]string{"PROTOCOL": "https", "HOST": "api.example.com", "PORT": "443"},
			expected: "url: https://api.example.com:443",
		},
		{
			name:     "missing variable becomes empty",
			input:    "key: ${MISSING_VAR}",
			envVars:  map[string]string{},
			expected: "key: ",
		},
		{
			name:     "no variables",
			input:    "plain text",
			envVars:  map[string]string{},
			expected: "plain text",
		},
		{
			name:     "default value used when var not set",
			input:    "threads: ${THREADS:-8}",
			envVars:  map[string]string{},
			expected: "threads: 8",
		},
		{
			name:     "default value ignored when var is set",
			input:    "threads: ${THREADS:-8}",
			envVars:  map[string]string{"THREADS": "16"},
			expected: "threads: 16",
		},
		{
			name:     "default value with url",
			input:    "url: ${BASE_URL:-http://localhost:11434}",
			envVars:  map[string]string{},
			expected: "url: http://localhost:11434",
		},
		{
			name:     "multiple vars with defaults",
			input:    "config: ${HOST:-localhost}:${PORT:-8080}",
			envVars:  map[string]string{"HOST": "example.com"},
			expected: "config: example.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			for k := range tt.envVars {
				os.Unsetenv(k)
			}

			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := expandEnvVars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadFromYAMLWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_API_KEY", "my-secret-key")
	os.Setenv("TEST_BASE_URL", "https://api.test.com")
	defer os.Unsetenv("TEST_API_KEY")
	defer os.Unsetenv("TEST_BASE_URL")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
providers:
  - name: test-api
    enabled: true
    priority: 1
    plugin_path: plugins/test-api
    config:
      api_key: ${TEST_API_KEY}
      base_url: ${TEST_BASE_URL}
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg := loadDefaults()
	err = cfg.loadFromYAML(configPath)
	require.NoError(t, err)

	assert.Len(t, cfg.Providers, 1)
	assert.Equal(t, "my-secret-key", cfg.Providers[0].Config["api_key"])
	assert.Equal(t, "https://api.test.com", cfg.Providers[0].Config["base_url"])
}

func TestApplyEnvironmentOverrides(t *testing.T) {
	os.Setenv("PORT", "9999")
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("PLUGIN_DIR", "prod_plugins")
	os.Setenv("ALLOWED_ORIGINS", "http://prod1.com,http://prod2.com")
	defer os.Unsetenv("PORT")
	defer os.Unsetenv("ENVIRONMENT")
	defer os.Unsetenv("PLUGIN_DIR")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	cfg := loadDefaults()
	cfg.applyEnvironmentOverrides()

	assert.Equal(t, "9999", cfg.Port)
	assert.Equal(t, "production", cfg.Environment)
	assert.Equal(t, "prod_plugins", cfg.PluginDir)
	assert.Equal(t, []string{"http://prod1.com", "http://prod2.com"}, cfg.AllowedOrigins)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				Port:      "8080",
				PluginDir: "plugins",
				Providers: []ProviderConfig{
					{Name: "mock", PluginPath: "plugins/mock"},
				},
				Routing: RoutingConfig{
					DefaultProvider: "mock",
				},
				Budget: BudgetConfig{
					QueryLimit:   50000,
					SessionLimit: 200000,
					MonthlyLimit: 2000000,
				},
			},
			expectErr: false,
		},
		{
			name: "empty port",
			config: &Config{
				Port:      "",
				PluginDir: "plugins",
			},
			expectErr: true,
			errMsg:    "port cannot be empty",
		},
		{
			name: "empty plugin_dir",
			config: &Config{
				Port:      "8080",
				PluginDir: "",
			},
			expectErr: true,
			errMsg:    "plugin_dir cannot be empty",
		},
		{
			name: "origin missing scheme",
			config: &Config{
				Port:           "8080",
				PluginDir:      "plugins",
				AllowedOrigins: []string{"localhost:3000"},
			},
			expectErr: true,
			errMsg:    `invalid allowed_origin "localhost:3000"`,
		},
		{
			name: "wildcard origin is valid",
			config: &Config{
				Port:           "8080",
				PluginDir:      "plugins",
				AllowedOrigins: []string{"*"},
			},
			expectErr: false,
		},
		{
			name: "https origin is valid",
			config: &Config{
				Port:           "8080",
				PluginDir:      "plugins",
				AllowedOrigins: []string{"https://example.com"},
			},
			expectErr: false,
		},
		{
			name: "mcp_apps origin missing scheme",
			config: &Config{
				Port:      "8080",
				PluginDir: "plugins",
				MCPApps: MCPAppsConfig{
					AllowedOrigins: []string{"example.com"},
				},
			},
			expectErr: true,
			errMsg:    `invalid mcp_apps.allowed_origin "example.com"`,
		},
		{
			name: "duplicate provider names",
			config: &Config{
				Port:      "8080",
				PluginDir: "plugins",
				Providers: []ProviderConfig{
					{Name: "mock", PluginPath: "plugins/mock"},
					{Name: "mock", PluginPath: "plugins/mock2"},
				},
			},
			expectErr: true,
			errMsg:    "duplicate provider name: mock",
		},
		{
			name: "empty provider name",
			config: &Config{
				Port:      "8080",
				PluginDir: "plugins",
				Providers: []ProviderConfig{
					{Name: "", PluginPath: "plugins/mock"},
				},
			},
			expectErr: true,
			errMsg:    "provider name cannot be empty",
		},
		{
			name: "empty plugin path",
			config: &Config{
				Port:      "8080",
				PluginDir: "plugins",
				Providers: []ProviderConfig{
					{Name: "mock", PluginPath: ""},
				},
			},
			expectErr: true,
			errMsg:    "provider mock: plugin_path cannot be empty",
		},
		{
			name: "invalid default_provider",
			config: &Config{
				Port:      "8080",
				PluginDir: "plugins",
				Providers: []ProviderConfig{
					{Name: "mock", PluginPath: "plugins/mock"},
				},
				Routing: RoutingConfig{
					DefaultProvider: "nonexistent",
				},
			},
			expectErr: true,
			errMsg:    "default_provider 'nonexistent' does not exist in providers list",
		},
		{
			name: "invalid guest_provider",
			config: &Config{
				Port:      "8080",
				PluginDir: "plugins",
				Providers: []ProviderConfig{
					{Name: "mock", PluginPath: "plugins/mock"},
				},
				Routing: RoutingConfig{
					GuestProvider: "nonexistent",
				},
			},
			expectErr: true,
			errMsg:    "guest_provider 'nonexistent' does not exist in providers list",
		},
		{
			name: "negative query_limit",
			config: &Config{
				Port:      "8080",
				PluginDir: "plugins",
				Providers: []ProviderConfig{
					{Name: "mock", PluginPath: "plugins/mock"},
				},
				Budget: BudgetConfig{
					QueryLimit:   -1000,
					SessionLimit: 200000,
					MonthlyLimit: 2000000,
				},
			},
			expectErr: true,
			errMsg:    "query_limit must be positive, got -1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Test loading with no config file (should use defaults)
	os.Setenv("CONFIG_PATH", "/nonexistent/config.yaml")
	defer os.Unsetenv("CONFIG_PATH")

	cfg := Load()
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "plugins", cfg.PluginDir)
}

func TestProviderExists(t *testing.T) {
	providers := []ProviderConfig{
		{Name: "mock"},
		{Name: "test"},
	}

	assert.True(t, providerExists(providers, "mock"))
	assert.True(t, providerExists(providers, "test"))
	assert.False(t, providerExists(providers, "nonexistent"))
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "normal split",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			input:    "a, b , c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with empty entries",
			input:    "a,,b,",
			sep:      ",",
			expected: []string{"a", "b"},
		},
		{
			name:     "empty string",
			input:    "",
			sep:      ",",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadFromYAMLWithDisabledProviders(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
providers:
  - name: enabled-provider
    enabled: true
    priority: 1
    plugin_path: plugins/enabled
    config: {}
  
  - name: disabled-provider
    enabled: false
    priority: 2
    plugin_path: plugins/disabled
    config: {}

routing:
  default_provider: enabled-provider
  guest_provider: enabled-provider
  authenticated_provider: enabled-provider
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg := loadDefaults()
	err = cfg.loadFromYAML(configPath)
	require.NoError(t, err)

	assert.Len(t, cfg.Providers, 2)

	// Verify both providers are loaded in config
	assert.Equal(t, "enabled-provider", cfg.Providers[0].Name)
	assert.True(t, cfg.Providers[0].Enabled)

	assert.Equal(t, "disabled-provider", cfg.Providers[1].Name)
	assert.False(t, cfg.Providers[1].Enabled)

	// Note: The actual filtering of disabled providers happens
	// in PluginManager.DiscoverPlugins(), not in config loading
}

func TestValidateAutoRoutingWithProviders(t *testing.T) {
	// "auto" is a special keyword that should always be valid, even when
	// providers are configured and "auto" isn't one of their names.
	cfg := &Config{
		Port:      "8080",
		PluginDir: "plugins",
		Providers: []ProviderConfig{
			{Name: "ollama", Enabled: true, PluginPath: "ollama/ollama"},
			{Name: "deepseek-api", Enabled: true, PluginPath: "deepseek-api/deepseek-api"},
		},
		Routing: RoutingConfig{
			DefaultProvider:       "auto",
			GuestProvider:         "auto",
			AuthenticatedProvider: "auto",
		},
		Budget: BudgetConfig{
			QueryLimit:   50000,
			SessionLimit: 200000,
			MonthlyLimit: 2000000,
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidationWithMixedEnabledProviders(t *testing.T) {
	cfg := &Config{
		Port:      "8080",
		PluginDir: "plugins",
		Providers: []ProviderConfig{
			{Name: "enabled", Enabled: true, PluginPath: "plugins/enabled"},
			{Name: "disabled", Enabled: false, PluginPath: "plugins/disabled"},
		},
		Routing: RoutingConfig{
			DefaultProvider: "enabled",
		},
		Budget: BudgetConfig{
			QueryLimit:   50000,
			SessionLimit: 200000,
			MonthlyLimit: 2000000,
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestLoadFromYAMLMalformed(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectErr   bool
	}{
		{
			name: "invalid YAML syntax",
			yamlContent: `
providers:
  - name: test
    enabled: true
    invalid yaml here
`,
			expectErr: true,
		},
		{
			name: "invalid type for enabled",
			yamlContent: `
providers:
  - name: test
    enabled: "not a boolean"
    priority: 1
    plugin_path: plugins/test
`,
			expectErr: true,
		},
		{
			name: "invalid type for priority",
			yamlContent: `
providers:
  - name: test
    enabled: true
    priority: "not a number"
    plugin_path: plugins/test
`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tt.yamlContent), 0644)
			require.NoError(t, err)

			cfg := loadDefaults()
			err = cfg.loadFromYAML(configPath)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
