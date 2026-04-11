package mcp

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// MCPHostConfig represents the parsed YAML configuration for the MCP host.
// This is the top-level structure per gateway-mcp-contract.md specification.
type MCPHostConfig struct {
	// Version of the config schema
	Version string `yaml:"version"`

	// MCP module configuration
	MCP MCPConfig `yaml:"mcp"`
}

// MCPConfig contains all MCP-specific configuration.
type MCPConfig struct {
	// Global MCP settings
	Global GlobalMCPConfig `yaml:"global"`

	// Individual server configurations
	Servers []MCPServerConfig `yaml:"servers"`

	// Observability settings
	Observability ObservabilityConfig `yaml:"observability"`
}

// GlobalMCPConfig defines default values and global behavior.
type GlobalMCPConfig struct {
	// DefaultToolTimeout is the default timeout for MCP tool calls (seconds)
	DefaultToolTimeout int `yaml:"default_tool_timeout"`

	// ReconnectPolicy governs reconnection behavior
	Reconnect ReconnectPolicy `yaml:"reconnect"`

	// HealthCheckInterval is how often to check server health (seconds)
	HealthCheckInterval int `yaml:"health_check_interval"`

	// ResponseBufferSize for streamed responses
	ResponseBufferSize int `yaml:"response_buffer_size"`
}

// ReconnectPolicy controls reconnection attempts.
type ReconnectPolicy struct {
	MaxRetries        int     `yaml:"max_retries"`
	InitialBackoffMs  int     `yaml:"initial_backoff_ms"`
	MaxBackoffMs      int     `yaml:"max_backoff_ms"`
	BackoffMultiplier float64 `yaml:"backoff_multiplier"`
}

// MCPServerConfig describes a single MCP server.
type MCPServerConfig struct {
	// ID is the unique identifier for this server (alphanumeric + underscore)
	ID string `yaml:"id"`

	// DisplayName is the human-readable name
	DisplayName string `yaml:"display_name"`

	// NamespacePrefix is prepended to all tool names from this server
	// Example: "mcp_by_dojo" → tools become "mcp_by_dojo:create_artifact"
	NamespacePrefix string `yaml:"namespace_prefix"`

	// Transport configuration (stdio, SSE, or HTTP)
	Transport TransportConfig `yaml:"transport"`

	// Tool filtering
	Tools ToolFilterConfig `yaml:"tools"`

	// Connection timeouts
	Timeouts TimeoutConfig `yaml:"timeouts"`

	// Health check settings
	HealthCheck HealthCheckConfig `yaml:"health_check"`

	// Retry policy for failed invocations
	RetryPolicy RetryPolicy `yaml:"retry_policy"`
}

// TransportConfig describes how to connect to the MCP server.
type TransportConfig struct {
	// Type: "stdio", "sse", or "streamable_http"
	Type string `yaml:"type"`

	// For stdio: absolute path to executable
	Command string `yaml:"command"`

	// For stdio: command-line arguments (subject to env var expansion)
	Args []string `yaml:"args"`

	// For stdio: environment variables (supports ${VAR} expansion)
	Env map[string]string `yaml:"env"`

	// For SSE/HTTP: server URL
	URL string `yaml:"url"`

	// For SSE/HTTP: HTTP headers (e.g., Authorization, supports ${VAR} expansion)
	Headers map[string]string `yaml:"headers"`
}

// ToolFilterConfig controls which tools are registered.
type ToolFilterConfig struct {
	// Allowlist: if non-empty, only these tools are registered
	Allowlist []string `yaml:"allowlist"`

	// Blocklist: tools to exclude from registration
	Blocklist []string `yaml:"blocklist"`
}

// TimeoutConfig specifies timeouts for various operations.
type TimeoutConfig struct {
	// Startup is the time to establish the initial connection (seconds)
	Startup int `yaml:"startup"`

	// ToolDefault is the default timeout per tool invocation (seconds)
	ToolDefault int `yaml:"tool_default"`

	// HealthCheck is the timeout for health checks (seconds)
	HealthCheck int `yaml:"health_check"`
}

// HealthCheckConfig controls health monitoring.
type HealthCheckConfig struct {
	// Enabled toggles health checks
	Enabled bool `yaml:"enabled"`

	// Path is the endpoint to check (if applicable)
	Path string `yaml:"path"`

	// IntervalSec is how often to run health checks
	IntervalSec int `yaml:"interval_sec"`
}

// RetryPolicy controls retry behavior for tool invocations.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of attempts (including initial)
	MaxAttempts int `yaml:"max_attempts"`

	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64 `yaml:"backoff_multiplier"`

	// MaxBackoffMs is the maximum backoff duration
	MaxBackoffMs int `yaml:"max_backoff_ms"`
}

// ObservabilityConfig controls tracing and logging.
type ObservabilityConfig struct {
	// Enabled toggles observability
	Enabled bool `yaml:"enabled"`

	// TraceProvider (e.g., "otel" for OpenTelemetry)
	TraceProvider string `yaml:"trace_provider"`

	// Attributes added to all spans
	Attributes map[string]string `yaml:"attributes"`

	// LogLevel for MCP events
	LogLevel string `yaml:"log_level"`

	// ToolSpanSampleRate is the fraction of tool invocations to trace (0.0-1.0)
	ToolSpanSampleRate float64 `yaml:"tool_span_sample_rate"`
}

// LoadConfig loads the MCP configuration from a YAML file.
// Environment variables in the format ${VAR_NAME} are expanded during parsing.
// Returns an error if the file cannot be read or parsed.
func LoadConfig(path string) (*MCPHostConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config file: %w", err)
	}

	// Expand environment variables
	expandedData := expandEnvVars(string(data))

	var cfg MCPHostConfig
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse MCP YAML config: %w", err)
	}

	// Validate the config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MCP config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that the MCP host configuration is valid.
// Returns an error if any validation rules are violated.
func (c *MCPHostConfig) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	return c.MCP.Validate()
}

// Validate checks that the MCP configuration is valid.
// Returns an error if any validation rules are violated.
func (c *MCPConfig) Validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("at least one MCP server must be configured")
	}

	serverIDs := make(map[string]bool)
	for i, server := range c.Servers {
		if err := server.Validate(); err != nil {
			return fmt.Errorf("server %d (%s): %w", i, server.ID, err)
		}

		// Check for duplicate server IDs
		if serverIDs[server.ID] {
			return fmt.Errorf("duplicate server ID: %s", server.ID)
		}
		serverIDs[server.ID] = true
	}

	return nil
}

// Validate checks that an MCPServerConfig is valid.
// Returns an error if any required fields are missing or invalid.
func (s *MCPServerConfig) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("server ID cannot be empty")
	}

	// Validate ID format (alphanumeric + underscore)
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(s.ID) {
		return fmt.Errorf("server ID must be alphanumeric with underscores only, got '%s'", s.ID)
	}

	if s.NamespacePrefix == "" {
		return fmt.Errorf("namespace_prefix cannot be empty")
	}

	return s.Transport.Validate()
}

// Validate checks that a TransportConfig is valid.
func (t *TransportConfig) Validate() error {
	if t.Type != "stdio" && t.Type != "sse" && t.Type != "streamable_http" {
		return fmt.Errorf("transport type must be 'stdio', 'sse', or 'streamable_http', got '%s'", t.Type)
	}

	if t.Type == "stdio" && t.Command == "" {
		return fmt.Errorf("command cannot be empty for stdio transport")
	}

	if (t.Type == "sse" || t.Type == "streamable_http") && t.URL == "" {
		return fmt.Errorf("URL cannot be empty for %s transport", t.Type)
	}

	return nil
}

// IsToolAllowed checks if a tool name is allowed based on allowlist and blocklist.
// Blocklist takes precedence over allowlist.
func (s *MCPServerConfig) IsToolAllowed(toolName string) bool {
	// Check blocklist first
	for _, blocked := range s.Tools.Blocklist {
		if matched := matchPattern(toolName, blocked); matched {
			return false
		}
	}

	// If allowlist is empty, allow all (unless blocked)
	if len(s.Tools.Allowlist) == 0 {
		return true
	}

	// Check allowlist
	for _, allowed := range s.Tools.Allowlist {
		if matched := matchPattern(toolName, allowed); matched {
			return true
		}
	}

	return false
}

// ValidateExpansions checks for empty-string expansions that indicate missing
// environment variables. This catches the common issue where ${ZEN_SCI_SERVERS_ROOT}
// or ${PYTHON_PATH} is not set, which silently produces broken subprocess args.
// Returns the number of warnings logged. Does not return an error — MCP startup
// is intentionally non-fatal so the gateway can still serve non-MCP traffic.
func (c *MCPConfig) ValidateExpansions() int {
	warnings := 0
	for _, srv := range c.Servers {
		if srv.Transport.Type != "stdio" {
			continue
		}
		// Check args for empty or bare-slash paths (indicates unexpanded var)
		for i, arg := range srv.Transport.Args {
			if arg == "" || strings.HasPrefix(arg, "/") && strings.Contains(arg, "//") {
				slog.Warn("MCP server has empty or suspicious arg — likely missing env var",
					"server", srv.ID, "arg_index", i, "arg_value", arg)
				warnings++
			}
		}
		// Check env values for empty strings that were likely ${VAR} expansions
		for k, v := range srv.Transport.Env {
			if v == "" && k != "LOG_LEVEL" {
				slog.Warn("MCP server env var expanded to empty — check that the variable is set",
					"server", srv.ID, "env_key", k)
				warnings++
			}
		}
	}
	return warnings
}

// expandEnvVars expands environment variables in the format ${VAR_NAME} or
// ${VAR_NAME:-default}. If the variable is not set (or is empty), the default
// value is used when provided; otherwise the empty string is substituted.
func expandEnvVars(s string) string {
	// Match ${VAR} or ${VAR:-default} where default may contain any non-} chars.
	re := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-([^}]*))?\}`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		varName := sub[1]
		defaultVal := sub[2] // empty string when no :- clause was present

		value := os.Getenv(varName)
		if value == "" {
			return defaultVal
		}
		return value
	})
}

// matchPattern performs simple wildcard matching (* matches any characters).
func matchPattern(s, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return s == pattern
	}

	// Simple wildcard matching: convert pattern to regex
	regexPattern := "^" + regexp.QuoteMeta(pattern) + "$"
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	re := regexp.MustCompile(regexPattern)
	return re.MatchString(s)
}
