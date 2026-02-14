package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port           string           `yaml:"port"`
	AllowedOrigins []string         `yaml:"allowed_origins"`
	Environment    string           `yaml:"environment"`
	PluginDir      string           `yaml:"plugin_dir"`
	Providers      []ProviderConfig `yaml:"providers"`
	Routing        RoutingConfig    `yaml:"routing"`
	Budget         BudgetConfig     `yaml:"budget"`
	OTEL           OTELConfig       `yaml:"otel"`
}

type ProviderConfig struct {
	Name       string                 `yaml:"name"`
	Enabled    bool                   `yaml:"enabled"`
	Priority   int                    `yaml:"priority"`
	PluginPath string                 `yaml:"plugin_path"`
	Config     map[string]interface{} `yaml:"config"`
}

type RoutingConfig struct {
	DefaultProvider       string            `yaml:"default_provider"`
	GuestProvider         string            `yaml:"guest_provider"`
	AuthenticatedProvider string            `yaml:"authenticated_provider"`
	HandlerMapping        map[string]string `yaml:"handler_mapping"` // Maps logical handler names to provider names
}

// ResolveHandler maps a logical intent handler name (e.g. "llm-fast", "llm-reasoning")
// to a real provider plugin name. Falls back to default_provider if no mapping exists.
func (rc *RoutingConfig) ResolveHandler(handler string) string {
	if rc.HandlerMapping != nil {
		if provider, ok := rc.HandlerMapping[handler]; ok {
			return provider
		}
	}
	return rc.DefaultProvider
}

type BudgetConfig struct {
	QueryLimit   int `yaml:"query_limit"`
	SessionLimit int `yaml:"session_limit"`
	MonthlyLimit int `yaml:"monthly_limit"`
}

// OTELConfig configures OpenTelemetry trace export
type OTELConfig struct {
	Enabled      bool    `json:"enabled" yaml:"enabled"`
	Endpoint     string  `json:"endpoint" yaml:"endpoint"`
	SamplingRate float64 `json:"sampling_rate" yaml:"sampling_rate"`
	ServiceName  string  `json:"service_name" yaml:"service_name"`
}

func Load() *Config {
	cfg := loadDefaults()

	// Try to load from YAML file if it exists
	configPath := getEnv("CONFIG_PATH", "config/config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		if err := cfg.loadFromYAML(configPath); err != nil {
			// Log error but continue with defaults
			fmt.Fprintf(os.Stderr, "Warning: failed to load config from %s: %v\n", configPath, err)
		}
	}

	// Override with environment variables
	cfg.applyEnvironmentOverrides()

	return cfg
}

func loadDefaults() *Config {
	return &Config{
		Port:           "8080",
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:3003"},
		Environment:    "development",
		PluginDir:      "plugins",
		Providers:      []ProviderConfig{},
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
		OTEL: OTELConfig{
			Enabled:      false,
			Endpoint:     "",
			SamplingRate: 1.0,
			ServiceName:  "agentic-gateway",
		},
	}
}

func (c *Config) loadFromYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in YAML content
	expandedData := expandEnvVars(string(data))

	if err := yaml.Unmarshal([]byte(expandedData), c); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return nil
}

func (c *Config) applyEnvironmentOverrides() {
	if port := os.Getenv("PORT"); port != "" {
		c.Port = port
	}
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		c.Environment = env
	}
	if pluginDir := os.Getenv("PLUGIN_DIR"); pluginDir != "" {
		c.PluginDir = pluginDir
	}

	// Handle allowed origins
	originsEnv := getEnv("ALLOWED_ORIGINS", getEnv("ALLOWED_ORIGIN", ""))
	if originsEnv != "" {
		var origins []string
		for _, origin := range splitAndTrim(originsEnv, ",") {
			if origin != "" {
				origins = append(origins, origin)
			}
		}
		if len(origins) > 0 {
			c.AllowedOrigins = origins
		}
	}

	// Handle OTEL configuration
	if otelEnabled := os.Getenv("OTEL_ENABLED"); otelEnabled != "" {
		c.OTEL.Enabled = otelEnabled == "true" || otelEnabled == "1"
	}
	if otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); otelEndpoint != "" {
		c.OTEL.Endpoint = otelEndpoint
	}
	if otelServiceName := os.Getenv("OTEL_SERVICE_NAME"); otelServiceName != "" {
		c.OTEL.ServiceName = otelServiceName
	}
	if otelSamplingRate := os.Getenv("OTEL_SAMPLING_RATE"); otelSamplingRate != "" {
		if rate, err := strconv.ParseFloat(otelSamplingRate, 64); err == nil {
			c.OTEL.SamplingRate = rate
		}
	}

}

func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("port cannot be empty")
	}

	if c.PluginDir == "" {
		return fmt.Errorf("plugin_dir cannot be empty")
	}

	// Check if at least one enabled provider exists (warning, not error)
	hasEnabledProvider := false
	for _, provider := range c.Providers {
		if provider.Enabled {
			hasEnabledProvider = true
			break
		}
	}
	if len(c.Providers) > 0 && !hasEnabledProvider {
		fmt.Fprintf(os.Stderr, "Warning: No enabled providers in configuration\n")
	}

	// Validate provider names are unique
	providerNames := make(map[string]bool)
	for _, provider := range c.Providers {
		if provider.Name == "" {
			return fmt.Errorf("provider name cannot be empty")
		}
		if providerNames[provider.Name] {
			return fmt.Errorf("duplicate provider name: %s", provider.Name)
		}
		providerNames[provider.Name] = true

		if provider.PluginPath == "" {
			return fmt.Errorf("provider %s: plugin_path cannot be empty", provider.Name)
		}
	}

	// Validate routing references exist (only when providers are configured).
	// "auto" is a special keyword that resolves dynamically at runtime — always valid.
	if len(c.Providers) > 0 {
		if c.Routing.DefaultProvider != "" && c.Routing.DefaultProvider != "auto" {
			if !providerExists(c.Providers, c.Routing.DefaultProvider) {
				return fmt.Errorf("default_provider '%s' does not exist in providers list", c.Routing.DefaultProvider)
			}
		}

		if c.Routing.GuestProvider != "" && c.Routing.GuestProvider != "auto" {
			if !providerExists(c.Providers, c.Routing.GuestProvider) {
				return fmt.Errorf("guest_provider '%s' does not exist in providers list", c.Routing.GuestProvider)
			}
		}

		if c.Routing.AuthenticatedProvider != "" && c.Routing.AuthenticatedProvider != "auto" {
			if !providerExists(c.Providers, c.Routing.AuthenticatedProvider) {
				return fmt.Errorf("authenticated_provider '%s' does not exist in providers list", c.Routing.AuthenticatedProvider)
			}
		}

		// Validate handler_mapping references exist
		for handler, provider := range c.Routing.HandlerMapping {
			if !providerExists(c.Providers, provider) {
				fmt.Fprintf(os.Stderr, "Warning: handler_mapping '%s' → '%s' references a provider not in providers list\n", handler, provider)
			}
		}
	}

	// Validate budget limits are positive
	if c.Budget.QueryLimit < 0 {
		return fmt.Errorf("query_limit must be positive, got %d", c.Budget.QueryLimit)
	}
	if c.Budget.SessionLimit < 0 {
		return fmt.Errorf("session_limit must be positive, got %d", c.Budget.SessionLimit)
	}
	if c.Budget.MonthlyLimit < 0 {
		return fmt.Errorf("monthly_limit must be positive, got %d", c.Budget.MonthlyLimit)
	}

	return nil
}

func providerExists(providers []ProviderConfig, name string) bool {
	for _, p := range providers {
		if p.Name == name {
			return true
		}
	}
	return false
}

// expandEnvVars expands environment variables in the format ${VAR}, ${VAR:-default}, or $VAR
func expandEnvVars(s string) string {
	// Pattern to match ${VAR:-default}, ${VAR}, or $VAR
	re := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-([^}]*))?\}|\$([a-zA-Z_][a-zA-Z0-9_]*)`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		var varName, defaultValue string

		if strings.HasPrefix(match, "${") {
			// Handle ${VAR} or ${VAR:-default}
			inner := match[2 : len(match)-1]
			if idx := strings.Index(inner, ":-"); idx != -1 {
				// ${VAR:-default} format
				varName = inner[:idx]
				defaultValue = inner[idx+2:]
			} else {
				// ${VAR} format
				varName = inner
			}
		} else {
			// Handle $VAR format
			varName = match[1:]
		}

		// Return environment variable value, or default value if not set
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return defaultValue
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
