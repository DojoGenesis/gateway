package config

import (
	"errors"
	"fmt"
	"io"
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
	MCPApps        MCPAppsConfig    `yaml:"mcp_apps"`

	// RegistrationEnabled opens POST /auth/register to anonymous callers.
	//
	// The default is TRUE — open. That default is about upgrade safety, not
	// policy: it matches the behaviour the gateway has always had in practice
	// (the server used to hardcode it), so upgrading a host cannot close a
	// door the operator wants open.
	//
	// It is NOT what gateway.trespies.dev runs. That host closed registration
	// on 2026-08-04 via REGISTRATION_ENABLED=false in /etc/dojo/env (DGS-101):
	// a completed sign-up called issueToken(userID, "user", …), which is a
	// working /v1 credential — provider spend — and re-entry for anything
	// gated on authentication alone.
	//
	// Until this release the key did not exist on the struct at all, so a
	// config file setting it — either way — was silently discarded. It is now
	// parsed. See applyRegistrationFileValue for the one asymmetry: a config
	// *file* is not allowed to close registration on an upgrade, because
	// deployed files predate this key being read. REGISTRATION_ENABLED=false
	// in the environment closes it and always wins.
	RegistrationEnabled bool `yaml:"registration_enabled"`
}

// MCPAppsConfig configures MCP Apps host infrastructure (v1.1.0).
type MCPAppsConfig struct {
	Enabled        bool     `json:"enabled" yaml:"enabled"`
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowed_origins"`
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

// DefaultConfigPath is consulted when neither -config nor CONFIG_PATH names a
// file. It is relative to the process working directory, which is why a
// deployment that does not name a path explicitly usually finds nothing.
const DefaultConfigPath = "config/config.yaml"

// PathSource records how the config file path was chosen. A path the operator
// named (flag or environment) is "explicit": failing to read it is an error,
// because silently running on defaults is what let a whole config file sit
// inert in production.
type PathSource string

const (
	PathSourceFlag    PathSource = "-config flag"
	PathSourceEnv     PathSource = "CONFIG_PATH environment variable"
	PathSourceDefault PathSource = "built-in default path"
)

// Explicit reports whether an operator named this path.
func (s PathSource) Explicit() bool { return s != PathSourceDefault }

// LoadOptions carries command-line input into configuration loading.
type LoadOptions struct {
	// Path is the file named by -config. Empty when the flag was not given.
	Path string
}

// LoadResult is the outcome of a load: always a usable Config, plus enough
// diagnostics for the caller to say out loud where configuration came from.
type LoadResult struct {
	// Config is never nil — on any failure it holds defaults plus environment
	// overrides, so a caller that chooses to continue still has a valid config.
	Config *Config

	// Path is the file that was consulted, and PathSource how it was chosen.
	Path       string
	PathSource PathSource

	// Loaded is true when the file was found and decoded.
	Loaded bool

	// Warnings names every key the file contained that the gateway could not
	// apply. Each one is a setting the operator wrote that does nothing.
	Warnings []string

	// Err is a hard failure: an explicitly-named file that could not be read,
	// or a file that is not parseable YAML. Callers should refuse to start.
	Err error
}

// LoadWithOptions resolves the config file, loads it, and applies environment
// overrides on top.
//
// Precedence, lowest first: built-in defaults → config file → environment.
func LoadWithOptions(opts LoadOptions) *LoadResult {
	res := &LoadResult{Config: loadDefaults()}

	switch {
	case opts.Path != "":
		res.Path, res.PathSource = opts.Path, PathSourceFlag
	case os.Getenv("CONFIG_PATH") != "":
		res.Path, res.PathSource = os.Getenv("CONFIG_PATH"), PathSourceEnv
	default:
		res.Path, res.PathSource = DefaultConfigPath, PathSourceDefault
	}

	//nolint:gosec // G703 -- the path is operator input by design (-config / CONFIG_PATH);
	// pointing the gateway at its own config file is the whole purpose of the flag.
	if _, err := os.Stat(res.Path); err != nil {
		if res.PathSource.Explicit() {
			// An operator who names a config file expects it to be read. The
			// systemd unit has passed -config /etc/dojo/config.yaml since the
			// first deploy while the binary parsed no flags at all, so the file
			// was never opened and nothing said so.
			res.Err = fmt.Errorf("config file %q named by the %s could not be opened: %w",
				res.Path, res.PathSource, err)
		}
		res.Config.applyEnvironmentOverrides()
		return res
	}

	warnings, err := res.Config.loadFromYAML(res.Path)
	res.Warnings = warnings
	if err != nil {
		res.Err = fmt.Errorf("config file %q (from %s): %w", res.Path, res.PathSource, err)
	} else {
		res.Loaded = true
	}

	res.Config.applyEnvironmentOverrides()
	return res
}

// Load is the historical entry point: it never reports failure to its caller,
// so it writes diagnostics to stderr. Prefer LoadWithOptions.
func Load() *Config {
	res := LoadWithOptions(LoadOptions{})
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}
	if res.Err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", res.Err)
	}
	return res.Config
}

func loadDefaults() *Config {
	return &Config{
		Port:           "7340",
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
		// Open by design: anyone may register and use the chat. See the field
		// comment on Config.RegistrationEnabled.
		RegistrationEnabled: true,
	}
}

// loadFromYAML applies path to c and returns one warning per key it could not
// use. It returns an error only when the file is unreadable or is not valid
// YAML — cases where nothing at all could be applied.
//
// Unknown and unparseable keys are warnings rather than errors on purpose.
// Strict decoding (KnownFields) is used to *find* them, but promoting them to
// a startup failure would refuse to boot the current production host, whose
// config file carries several keys this struct has never had fields for.
func (c *Config) loadFromYAML(path string) ([]string, error) {
	//nolint:gosec // G304 -- path is the operator-named config file (see LoadWithOptions).
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in YAML content
	expandedData := expandEnvVars(string(data))

	// Decode into a copy: a file that fails to parse outright must not leave
	// the caller with a half-applied configuration.
	scratch := *c

	dec := yaml.NewDecoder(strings.NewReader(expandedData))
	dec.KnownFields(true)
	err = dec.Decode(&scratch)

	var typeErr *yaml.TypeError
	var warnings []string

	switch {
	case errors.Is(err, io.EOF):
		// Empty or comment-only file. Nothing to apply, nothing wrong — but
		// say so, because "my settings do nothing" is the bug being fixed.
		return []string{fmt.Sprintf(
			"config file %s contains no settings — every value is coming from defaults and environment variables", path,
		)}, nil

	case err == nil:
		// Whole document understood.

	case errors.As(err, &typeErr):
		// yaml decoded everything it understood and is reporting the rest:
		// unknown keys, and values whose shape does not fit the struct. Keep
		// what decoded, name what did not.
		warnings = describeTypeErrors(path, expandedData, typeErr)

	default:
		// A syntax error: the document could not be parsed, so none of it was
		// applied. Do not pretend the file was honoured.
		return nil, fmt.Errorf("failed to parse YAML config: %w", errors.New(redactYAMLScalars(err.Error())))
	}

	prior := c.RegistrationEnabled
	*c = scratch
	return append(warnings, c.applyRegistrationFileValue(path, expandedData, prior)...), nil
}

// applyRegistrationFileValue enforces the one asymmetry in config loading: a
// config *file* may open registration but may not close it.
//
// The reason is a live upgrade hazard, not a policy preference. Deployed config
// files were written when this key was silently discarded, so the moment the
// file starts being honoured, a restart could close a service on the strength
// of a line nobody knew was live. Nobody wrote that `false` with the knowledge
// that it would take effect.
//
// REGISTRATION_ENABLED in the environment is applied later, unconditionally, in
// both directions: it is the supported way to close registration, and it cannot
// have been set by an older deploy that did not know about it.
//
// Removing this guard: the condition is that no deployed config file still
// carries a stale `registration_enabled: false`. Verified 2026-08-04 —
// gateway.trespies.dev's /etc/dojo/config.yaml carries `true`, so the guard is
// inert on that host and registration there is closed by the environment
// instead (DGS-101). Check the other deployed hosts before deleting it; this
// was checked on one.
func (c *Config) applyRegistrationFileValue(path, expandedData string, prior bool) []string {
	var probe struct {
		RegistrationEnabled *bool `yaml:"registration_enabled"`
	}
	// Non-strict on purpose: this decode only inspects one key.
	if err := yaml.Unmarshal([]byte(expandedData), &probe); err != nil {
		return nil
	}
	if probe.RegistrationEnabled == nil || *probe.RegistrationEnabled {
		return nil // absent, or opening registration — nothing to guard.
	}

	c.RegistrationEnabled = prior
	if !prior {
		return nil // already closed by a lower layer; nothing changed, nothing to say.
	}
	if v := os.Getenv("REGISTRATION_ENABLED"); v != "" && v != "true" && v != "1" {
		// The environment override is applied after this and will close
		// registration anyway. Announcing that the file is being overruled
		// would contradict the state the gateway actually ends up in.
		return nil
	}
	return []string{fmt.Sprintf(
		"%s sets registration_enabled: false, but user registration is staying OPEN. "+
			"That key was never read until this release, so a config file cannot be "+
			"trusted to mean it — a restart would have silently closed public sign-up. "+
			"To actually close registration set REGISTRATION_ENABLED=false in the "+
			"environment; to keep it open, change this file to registration_enabled: true", path,
	)}
}

// describeTypeErrors turns yaml's strict-decoding complaints into operator-
// facing warnings. Every line here is a setting somebody wrote that does not
// do what they think it does.
func describeTypeErrors(path, doc string, typeErr *yaml.TypeError) []string {
	keys := yamlKeyPaths(doc)
	warnings := make([]string, 0, len(typeErr.Errors))
	for _, msg := range typeErr.Errors {
		clean := redactYAMLScalars(msg)
		if strings.Contains(msg, "not found in type") {
			// These already name the offending key.
			warnings = append(warnings, fmt.Sprintf(
				"%s: %s — this key is not a gateway setting and has NO effect", path, clean))
			continue
		}
		// These identify the problem by line only ("line 14: cannot unmarshal
		// !!map into []config.ProviderConfig"), which is useless in a journal
		// without the file open. Name the key.
		if key, ok := keys[yamlErrorLine(msg)]; ok {
			clean += fmt.Sprintf(" (key %q)", key)
		}
		warnings = append(warnings, fmt.Sprintf(
			"%s: %s — this value could not be parsed and was IGNORED; the previous value is still in effect", path, clean))
	}
	return warnings
}

// yamlErrorLineRE matches the "line N:" prefix yaml puts on each type error.
var yamlErrorLineRE = regexp.MustCompile(`^line (\d+):`)

func yamlErrorLine(msg string) int {
	m := yamlErrorLineRE.FindStringSubmatch(msg)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

// yamlKeyPaths maps each line that declares a mapping key to that key's dotted
// path, so a line number from a decode error can be reported as a key name.
func yamlKeyPaths(doc string) map[int]string {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return nil
	}

	paths := make(map[int]string)
	var walk func(n *yaml.Node, prefix string)
	walk = func(n *yaml.Node, prefix string) {
		switch n.Kind {
		case yaml.DocumentNode:
			for _, child := range n.Content {
				walk(child, prefix)
			}
		case yaml.MappingNode:
			for i := 0; i+1 < len(n.Content); i += 2 {
				key, value := n.Content[i], n.Content[i+1]
				path := key.Value
				if prefix != "" {
					path = prefix + "." + key.Value
				}
				paths[key.Line] = path
				walk(value, path)
			}
		case yaml.SequenceNode:
			for i, child := range n.Content {
				walk(child, fmt.Sprintf("%s[%d]", prefix, i))
			}
		}
	}
	walk(&root, "")
	return paths
}

// yamlQuotedScalar matches the offending value yaml embeds in type errors, e.g.
// "cannot unmarshal !!str `sk-ant-...` into int".
var yamlQuotedScalar = regexp.MustCompile("`[^`]*`")

// redactYAMLScalars strips values out of yaml error text before it is logged.
// Config files hold provider API keys — and ${VAR} placeholders are expanded
// before parsing, so the real secret is what would be quoted. Key names and
// line numbers survive; values do not.
func redactYAMLScalars(msg string) string {
	return yamlQuotedScalar.ReplaceAllString(msg, "`<redacted>`")
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

	// Handle MCP Apps configuration
	if mcpAppsEnabled := os.Getenv("MCP_APPS_ENABLED"); mcpAppsEnabled != "" {
		c.MCPApps.Enabled = mcpAppsEnabled == "true" || mcpAppsEnabled == "1"
	}

	// Registration. Unlike the config file this is honoured in both
	// directions — it is the supported way to close public sign-up, and no
	// older deployment can have set it by accident.
	if registration := os.Getenv("REGISTRATION_ENABLED"); registration != "" {
		c.RegistrationEnabled = registration == "true" || registration == "1"
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

	// Validate allowed origins have scheme required by gin-contrib/cors
	for _, origin := range c.AllowedOrigins {
		if origin != "*" && !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			return fmt.Errorf("invalid allowed_origin %q: must be \"*\" or start with http:// or https://", origin)
		}
	}
	for _, origin := range c.MCPApps.AllowedOrigins {
		if origin != "*" && !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			return fmt.Errorf("invalid mcp_apps.allowed_origin %q: must be \"*\" or start with http:// or https://", origin)
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
