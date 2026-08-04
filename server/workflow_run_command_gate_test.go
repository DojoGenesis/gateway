package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DGS-108 regression tests.
//
// These assert on a SIDE EFFECT ON THE FILESYSTEM, not on a return value. That
// is deliberate. The defect is that a workflow step becomes a real process on
// this host, so a test that only checked RunSkill's error would still pass if
// the command ran and merely reported oddly. Each test below asks the shell to
// create a file and then checks whether the file exists — the presence or
// absence of that file is the bug being on or off.
//
// Scope: these drive CommandSkillRunner directly, which is the exact boundary
// the gate protects. The unauthenticated HTTP path in front of it
// (POST /api/workflows -> POST /api/workflows/:name/execute) is a separate,
// still-open concern tracked by DGS-100; it is what makes this reachable from
// the internet, but it is not what makes it dangerous.

// runTouch asks the skill runner to create a marker file via the shell and
// reports whether the file exists afterwards. Existence means a real shell ran.
func runTouch(t *testing.T) (marker string, err error) {
	t.Helper()

	marker = filepath.Join(t.TempDir(), "dgs108-marker")
	r := NewCommandSkillRunner()

	_, err = r.RunSkill(context.Background(), "run_command", map[string]string{
		// Plain, innocuous, and — importantly — not matched by any of the five
		// substring patterns in tools.validateCommand. That is the point: the
		// denylist stops a handful of catastrophic literals and nothing else,
		// which is why the capability gate, not the denylist, has to be the
		// boundary.
		"command": "touch " + marker,
	})
	return marker, err
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, statErr := os.Stat(path)
	return statErr == nil
}

// TestDGS108_RunCommandReachesARealShellWhenEnabled is the repro. With the gate
// open — the default outside production — a workflow step executes on the host.
// If this test ever fails, the primitive is gone entirely and the rest of this
// file is testing nothing.
func TestDGS108_RunCommandReachesARealShellWhenEnabled(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv(EnvWorkflowRunCommand, "")

	marker, err := runTouch(t)

	require.NoError(t, err, "run_command should execute normally on a development gateway")
	require.True(t, fileExists(t, marker),
		"the shell did not run: this test cannot prove the gate blocks anything if the ungated path does not work")
}

// TestDGS108_ProductionBlocksRunCommandByDefault is THE regression test. This
// is the exact configuration of gateway.trespies.dev, and this is the step that
// an unauthenticated POST to /api/workflows could previously reach.
func TestDGS108_ProductionBlocksRunCommandByDefault(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv(EnvWorkflowRunCommand, "")

	marker, err := runTouch(t)

	require.Error(t, err, "a production gateway must refuse to shell out for a workflow step")
	assert.Contains(t, err.Error(), EnvWorkflowRunCommand,
		"the refusal must name the variable so an operator can act on it without reading the source")
	assert.False(t, fileExists(t, marker),
		"the command still executed: the gate returned an error but did not prevent the process")
}

// TestDGS108_ProductionOptInIsHonoured pins that this is a capability gate and
// not a prohibition. A self-hosted operator whose /api surface is genuinely
// closed may still want pipeline steps that shell out, and saying so in the
// environment must work — otherwise the gate gets patched out downstream.
func TestDGS108_ProductionOptInIsHonoured(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv(EnvWorkflowRunCommand, "true")

	marker, err := runTouch(t)

	require.NoError(t, err, "an explicit production opt-in must be honoured")
	assert.True(t, fileExists(t, marker), "opt-in did not actually re-enable execution")
}

// TestDGS108_ExplicitFalseBlocksOutsideProduction covers the other direction:
// a developer or CI box that wants the primitive off can turn it off, without
// having to claim to be production.
func TestDGS108_ExplicitFalseBlocksOutsideProduction(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv(EnvWorkflowRunCommand, "false")

	marker, err := runTouch(t)

	require.Error(t, err)
	assert.False(t, fileExists(t, marker))
}

// TestDGS108_UnparseableValueFailsClosedInProduction is the fat-finger case.
// WORKFLOW_RUN_COMMAND_ENABLED=yes is not a bool. Treating an unparseable value
// as "absent" rather than as "true" is what makes a typo safe in production.
func TestDGS108_UnparseableValueFailsClosedInProduction(t *testing.T) {
	for _, garbage := range []string{"yes", "on", "enabled", "TRUE!", "1.0"} {
		t.Run(garbage, func(t *testing.T) {
			t.Setenv("ENVIRONMENT", "production")
			t.Setenv(EnvWorkflowRunCommand, garbage)

			marker, err := runTouch(t)

			require.Error(t, err, "%q is not a parseable bool and must not enable a shell in production", garbage)
			assert.False(t, fileExists(t, marker))
		})
	}
}

// TestDGS108_EnvironmentMatchIsForgiving guards the string comparison itself.
// "Production", " production" and "PRODUCTION" are the same deployment; a
// gateway that shells out because its ENVIRONMENT had a trailing space would
// be the worst possible failure mode for this gate.
func TestDGS108_EnvironmentMatchIsForgiving(t *testing.T) {
	for _, env := range []string{"production", "Production", "PRODUCTION", " production ", "\tproduction"} {
		t.Run(strings.TrimSpace(env), func(t *testing.T) {
			t.Setenv("ENVIRONMENT", env)
			t.Setenv(EnvWorkflowRunCommand, "")

			marker, err := runTouch(t)

			require.Error(t, err, "ENVIRONMENT=%q must be recognised as production", env)
			assert.False(t, fileExists(t, marker))
		})
	}
}

// TestDGS108_UnsupportedSkillsStillRejected confirms the gate did not swallow
// the pre-existing behaviour for every other skill name.
func TestDGS108_UnsupportedSkillsStillRejected(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")

	r := NewCommandSkillRunner()
	_, err := r.RunSkill(context.Background(), "some_other_skill", map[string]string{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported skill")
}

// TestDGS108_StartupWarningOnlyFiresOnTheDangerousCombination keeps the warning
// meaningful. A warning that also fired in development would be tuned out, and
// the one configuration that must never be quiet is production + enabled.
func TestDGS108_StartupWarningOnlyFiresOnTheDangerousCombination(t *testing.T) {
	cases := []struct {
		env, flag string
		wantWarn  bool
		why       string
	}{
		{"production", "true", true, "production + enabled is the DGS-108 chain and must be loud"},
		{"production", "", false, "production defaults to disabled — nothing to warn about"},
		{"production", "false", false, "explicitly disabled in production is the desired state"},
		{"development", "true", false, "a developer machine shelling out is the normal case"},
		{"development", "", false, "development default needs no warning"},
	}

	for _, tc := range cases {
		t.Run(tc.env+"/"+tc.flag, func(t *testing.T) {
			t.Setenv("ENVIRONMENT", tc.env)
			t.Setenv(EnvWorkflowRunCommand, tc.flag)

			warning := RunCommandStartupWarning()

			if tc.wantWarn {
				require.NotEmpty(t, warning, tc.why)
				assert.Contains(t, warning, EnvWorkflowRunCommand)
			} else {
				assert.Empty(t, warning, tc.why)
			}
		})
	}
}
