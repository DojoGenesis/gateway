package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// neutralizeEnv clears every environment variable that can override a value
// under test, so these tests assert what the file and defaults produce rather
// than what the developer's shell happens to export.
func neutralizeEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"CONFIG_PATH", "PORT", "ENVIRONMENT", "PLUGIN_DIR",
		"ALLOWED_ORIGINS", "ALLOWED_ORIGIN", "REGISTRATION_ENABLED",
	} {
		t.Setenv(k, "")
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// ─── The config file actually reaches the process ───────────────────────────

// TestLoadWithOptions_ExplicitPathWins is the core of the -config fix: a file
// named on the command line is read, and its values beat the built-in defaults.
// Before this, the systemd unit's `-config /etc/dojo/config.yaml` was discarded
// and the file was never opened.
func TestLoadWithOptions_ExplicitPathWins(t *testing.T) {
	neutralizeEnv(t)

	path := writeConfig(t, `
port: "9111"
environment: staging
plugin_dir: /opt/dojo/plugins
allowed_origins:
  - "https://portal.example.test"
budget:
  query_limit: 123
  session_limit: 456
  monthly_limit: 789
`)

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	assert.True(t, res.Loaded, "the file must be reported as loaded")
	assert.Equal(t, path, res.Path)
	assert.Equal(t, PathSourceFlag, res.PathSource)
	assert.Empty(t, res.Warnings)

	// Every one of these differs from loadDefaults(), so passing proves the
	// file won rather than the defaults coincidentally matching.
	assert.Equal(t, "9111", res.Config.Port)
	assert.Equal(t, "staging", res.Config.Environment)
	assert.Equal(t, "/opt/dojo/plugins", res.Config.PluginDir)
	assert.Equal(t, []string{"https://portal.example.test"}, res.Config.AllowedOrigins)
	assert.Equal(t, 123, res.Config.Budget.QueryLimit)
	assert.Equal(t, 456, res.Config.Budget.SessionLimit)
	assert.Equal(t, 789, res.Config.Budget.MonthlyLimit)
}

func TestLoadWithOptions_ConfigPathEnvIsExplicit(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, "port: \"9222\"\n")
	t.Setenv("CONFIG_PATH", path)

	res := LoadWithOptions(LoadOptions{})

	require.NoError(t, res.Err)
	assert.True(t, res.Loaded)
	assert.Equal(t, PathSourceEnv, res.PathSource)
	assert.Equal(t, "9222", res.Config.Port)
}

func TestLoadWithOptions_FlagBeatsConfigPathEnv(t *testing.T) {
	neutralizeEnv(t)
	t.Setenv("CONFIG_PATH", writeConfig(t, "port: \"9333\"\n"))
	flagPath := writeConfig(t, "port: \"9444\"\n")

	res := LoadWithOptions(LoadOptions{Path: flagPath})

	require.NoError(t, res.Err)
	assert.Equal(t, PathSourceFlag, res.PathSource)
	assert.Equal(t, "9444", res.Config.Port)
}

func TestLoadWithOptions_EnvironmentBeatsFile(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, "port: \"9555\"\nenvironment: staging\n")
	t.Setenv("PORT", "9666")

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	assert.Equal(t, "9666", res.Config.Port, "environment must stay the highest-precedence layer")
	assert.Equal(t, "staging", res.Config.Environment)
}

// ─── A config file can no longer silently do nothing ────────────────────────

// TestLoadWithOptions_MissingExplicitPathIsReported covers the failure that
// hid everything else: a path the operator named that does not exist. It used
// to be indistinguishable from "no config file configured".
func TestLoadWithOptions_MissingExplicitPathIsReported(t *testing.T) {
	neutralizeEnv(t)
	missing := filepath.Join(t.TempDir(), "nope", "config.yaml")

	t.Run("from -config flag", func(t *testing.T) {
		res := LoadWithOptions(LoadOptions{Path: missing})

		require.Error(t, res.Err, "a named-but-missing config file must not be silently ignored")
		assert.Contains(t, res.Err.Error(), missing, "the error must name the path")
		assert.Contains(t, res.Err.Error(), string(PathSourceFlag), "the error must name where the path came from")
		assert.False(t, res.Loaded)
		// Still usable, so the caller decides whether to continue.
		require.NotNil(t, res.Config)
		assert.Equal(t, "7340", res.Config.Port)
	})

	t.Run("from CONFIG_PATH", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", missing)
		res := LoadWithOptions(LoadOptions{})

		require.Error(t, res.Err)
		assert.Contains(t, res.Err.Error(), string(PathSourceEnv))
	})
}

// TestLoadWithOptions_MissingDefaultPathIsNotAnError keeps the running host
// bootable: it finds no file at the default relative path and must not care.
func TestLoadWithOptions_MissingDefaultPathIsNotAnError(t *testing.T) {
	neutralizeEnv(t)
	t.Chdir(t.TempDir()) // no config/config.yaml here

	res := LoadWithOptions(LoadOptions{})

	require.NoError(t, res.Err)
	assert.False(t, res.Loaded)
	assert.Equal(t, PathSourceDefault, res.PathSource)
	assert.Equal(t, "7340", res.Config.Port)
}

// TestLoadWithOptions_UnknownKeysAreSurfaced is the direct fix for the class of
// bug that made `registration_enabled: false` look effective: keys the struct
// cannot parse were accepted in silence.
func TestLoadWithOptions_UnknownKeysAreSurfaced(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, `
port: "9777"
data_dir: /var/lib/dojo
oauth:
  github:
    enabled: true
totally_made_up_key: 42
`)

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err, "unknown keys must not stop the gateway from booting")
	assert.True(t, res.Loaded)
	assert.Equal(t, "9777", res.Config.Port, "keys that DO parse are still applied")

	joined := strings.Join(res.Warnings, "\n")
	for _, key := range []string{"data_dir", "oauth", "totally_made_up_key"} {
		assert.Contains(t, joined, key, "every ignored key must be named")
	}
	assert.Contains(t, joined, "NO effect")
}

// TestLoadWithOptions_UnparsableValueIsSurfaced covers the other half: a key
// the struct knows, holding a shape it cannot decode. The production file's
// `providers:` map is exactly this case.
func TestLoadWithOptions_UnparsableValueIsSurfaced(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, `
port: "9888"
providers:
  anthropic:
    api_key: "not-a-list"
`)

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	assert.Equal(t, "9888", res.Config.Port)
	assert.Empty(t, res.Config.Providers, "an undecodable value leaves the previous value in place")

	joined := strings.Join(res.Warnings, "\n")
	assert.Contains(t, joined, "providers")
	assert.Contains(t, joined, "IGNORED")
}

// TestLoadFromYAML_WarningsNeverEchoValues guards the logs. Config files hold
// provider API keys, ${VAR} placeholders are expanded before parsing, and yaml
// quotes the offending value inside its type errors.
func TestLoadFromYAML_WarningsNeverEchoValues(t *testing.T) {
	neutralizeEnv(t)
	//nolint:gosec // G101 -- not a credential; a fake shaped like one, so the
	// assertion below proves real values never reach a log line.
	const fakeSecret = "sk-ant-notreal-0123456789abcdef"
	t.Setenv("A_TEST_SECRET", fakeSecret)

	path := writeConfig(t, `
budget:
  query_limit: "${A_TEST_SECRET}"
`)

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	require.NotEmpty(t, res.Warnings)
	joined := strings.Join(res.Warnings, "\n")
	assert.NotContains(t, joined, fakeSecret)
	// yaml truncates to 7 characters before quoting; that prefix must go too.
	assert.NotContains(t, joined, fakeSecret[:7])
	assert.Contains(t, joined, "<redacted>")
}

func TestLoadWithOptions_SyntaxErrorIsFatalAndAppliesNothing(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, "port: \"9999\"\n  bogus: [unclosed\n")

	res := LoadWithOptions(LoadOptions{Path: path})

	require.Error(t, res.Err, "an unparseable file must not be treated as honoured")
	assert.False(t, res.Loaded)
	assert.Equal(t, "7340", res.Config.Port, "a failed parse must not half-apply the file")
}

func TestLoadWithOptions_EmptyFileIsReported(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, "# nothing but a comment\n")

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	require.NotEmpty(t, res.Warnings)
	assert.Contains(t, strings.Join(res.Warnings, "\n"), "no settings")
}

// ─── registration_enabled ───────────────────────────────────────────────────

// TestRegistrationDefaultIsOpen pins the deliberate posture: anyone may sign up
// and use the chat. An upgrade must not close it.
func TestRegistrationDefaultIsOpen(t *testing.T) {
	neutralizeEnv(t)
	t.Chdir(t.TempDir())

	assert.True(t, loadDefaults().RegistrationEnabled)
	assert.True(t, LoadWithOptions(LoadOptions{}).Config.RegistrationEnabled)
}

func TestRegistrationEnabledTrueInFileIsHonoured(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, "registration_enabled: true\n")

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	assert.True(t, res.Config.RegistrationEnabled)
	assert.Empty(t, res.Warnings, "the key parses now — it must not be reported as unknown")
}

// TestRegistrationEnabledFalseInFileDoesNotCloseRegistration documents the one
// asymmetry in config loading, and why it exists: every deployed config file
// was written while this key was silently discarded, so a file saying `false`
// is not evidence that anyone chose to close public sign-up. Honouring it on
// the next restart would take a deliberately open service offline for new
// users. The discrepancy is reported instead of applied.
func TestRegistrationEnabledFalseInFileDoesNotCloseRegistration(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, "port: \"9100\"\nregistration_enabled: false\n")

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	assert.True(t, res.Config.RegistrationEnabled, "a config file must not close registration on upgrade")
	assert.Equal(t, "9100", res.Config.Port, "the rest of the file is still applied")

	joined := strings.Join(res.Warnings, "\n")
	assert.Contains(t, joined, path, "the warning must name the file")
	assert.Contains(t, joined, "registration_enabled: false")
	assert.Contains(t, joined, "REGISTRATION_ENABLED=false", "and must name the way to actually close it")
}

// TestRegistrationEnvOverride is the supported way to close registration, and
// it is honoured in both directions.
func TestRegistrationEnvOverride(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"false", false},
		{"0", false},
		{"true", true},
		{"1", true},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			neutralizeEnv(t)
			t.Chdir(t.TempDir())
			t.Setenv("REGISTRATION_ENABLED", tt.env)

			assert.Equal(t, tt.want, LoadWithOptions(LoadOptions{}).Config.RegistrationEnabled)
		})
	}
}

// TestRegistrationEnvBeatsFile: the environment is the highest-precedence
// layer, including against a file that opens registration.
func TestRegistrationEnvBeatsFile(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, "registration_enabled: true\n")
	t.Setenv("REGISTRATION_ENABLED", "false")

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	assert.False(t, res.Config.RegistrationEnabled)
}

// TestRegistrationEnvCloseSilencesTheFileWarning: when the operator has closed
// registration in the environment, the guard must not announce that the file's
// `false` is being overruled — the gateway ends up closed either way, and a
// warning saying otherwise would be simply wrong.
func TestRegistrationEnvCloseSilencesTheFileWarning(t *testing.T) {
	neutralizeEnv(t)
	path := writeConfig(t, "registration_enabled: false\n")
	t.Setenv("REGISTRATION_ENABLED", "false")

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err)
	assert.False(t, res.Config.RegistrationEnabled)
	assert.NotContains(t, strings.Join(res.Warnings, "\n"), "staying OPEN")
}

// ─── The artifact that is actually deployed ─────────────────────────────────

// TestDeployedConfigArtifactLoads runs the real file that provision.sh installs
// to /etc/dojo/config.yaml through the loader, and pins what it does and does
// not change. It is the regression guard for "the production config file is
// decorative".
func TestDeployedConfigArtifactLoads(t *testing.T) {
	neutralizeEnv(t)

	// Placeholders in the artifact are expanded before parsing. Feed known
	// values so this test never touches a real key, and so the redaction
	// assertion below has something concrete to look for.
	const fakeKey = "sk-test-notreal-abcdef0123456789"
	for _, k := range []string{
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY",
		"GITHUB_OAUTH_CLIENT_ID", "GITHUB_OAUTH_CLIENT_SECRET",
	} {
		t.Setenv(k, fakeKey)
	}

	path := filepath.Join("..", "..", "deploy", "gateway-config.yaml")
	require.FileExists(t, path)

	res := LoadWithOptions(LoadOptions{Path: path})

	require.NoError(t, res.Err, "the deployed artifact must never make the gateway refuse to boot")
	assert.True(t, res.Loaded)

	// What it changes: nothing that differs from the running defaults.
	assert.Equal(t, "7340", res.Config.Port)
	assert.Equal(t, "production", res.Config.Environment)
	assert.Empty(t, res.Config.Providers, "provider keys come from the environment, not this file")
	assert.Equal(t, []string{"http://localhost:3000", "http://localhost:3003"}, res.Config.AllowedOrigins,
		"the artifact sets no allowed_origins, so CORS is untouched")
	assert.True(t, res.Config.RegistrationEnabled, "the deployed file must not close public sign-up")

	// What it reports: every key in the file that does nothing.
	joined := strings.Join(res.Warnings, "\n")
	for _, inert := range []string{"data_dir", "providers", "oauth"} {
		assert.Contains(t, joined, inert, "an inert key in the deployed artifact must be named at startup")
	}
	assert.NotContains(t, joined, fakeKey, "warnings must never echo a value from the config file")
	assert.NotContains(t, joined, fakeKey[:7])
}
