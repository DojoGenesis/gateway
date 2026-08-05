package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DGS-112 regression tests.
//
// The defect: the shim gate was `os.Getenv("ENVIRONMENT") != "production"`, an
// exact byte comparison, negated. Every other value selected development —
// including unset, which is the default everywhere — and in that state
// `Bearer admin-<anything>` authenticated AND returned isAdmin, with no
// signature and no secret. That reached /admin/*, all of /v1/*, and the
// destructive POST /api/cas/gc.
//
// The fix is not a better string comparison. It is the direction of the
// default: absence of configuration must not grant credentials. These tests are
// therefore written mostly as "with nothing configured, is admin refused?" —
// because "nothing configured" was the exploitable state.

// enableDevTokens opts this test into the unsigned development shim tokens.
// Tests that authenticate with "test-token", "user-…" or "admin-…" must call
// it, which keeps their dependency on a development-only backdoor explicit.
func enableDevTokens(t *testing.T) {
	t.Helper()
	t.Setenv(envEnvironment, "development")
	t.Setenv(EnvDevTokens, "true")
}

// TestDGS112_UnconfiguredHostRefusesAdminShim is THE regression test. This is
// the exact configuration the bug shipped in: ENVIRONMENT unset, nothing opted
// in. Before the fix, `admin-alice` was a full admin here.
func TestDGS112_UnconfiguredHostRefusesAdminShim(t *testing.T) {
	t.Setenv(envEnvironment, "")
	t.Setenv(EnvDevTokens, "")

	_, _, err := validateTokenWithClaims("admin-alice")
	require.Error(t, err, "an unsigned token must not authenticate on an unconfigured host")

	isAdmin, _ := validateAdminRole("admin-alice", "admin-alice")
	assert.False(t, isAdmin, "an unsigned token must never be admin on an unconfigured host")
}

// TestDGS112_MisspelledProductionNoLongerGrantsAdmin covers the specific
// scenario that made this High: an operator who wrote "Production" got a
// gateway that had loudly enforced its JWT secret at startup and was
// simultaneously handing out admin to anyone. Every spelling below was
// exploitable before the fix.
func TestDGS112_MisspelledProductionNoLongerGrantsAdmin(t *testing.T) {
	for _, env := range []string{"Production", "PRODUCTION", " production ", "prod", "prd", "", "staging", "development"} {
		t.Run("ENVIRONMENT="+env, func(t *testing.T) {
			t.Setenv(envEnvironment, env)
			t.Setenv(EnvDevTokens, "") // the default: nobody opted in

			_, _, err := validateTokenWithClaims("admin-alice")
			assert.Error(t, err, "ENVIRONMENT=%q must not accept an unsigned token without an explicit opt-in", env)

			isAdmin, _ := validateAdminRole("admin-alice", "admin-alice")
			assert.False(t, isAdmin, "ENVIRONMENT=%q must not grant admin to an unsigned token", env)
		})
	}
}

// TestDGS112_ProductionRefusesEvenWithExplicitOptIn pins the belt-and-braces
// second condition. An operator who sets the opt-in on a production host — by
// copying an env file, say — still gets nothing.
func TestDGS112_ProductionRefusesEvenWithExplicitOptIn(t *testing.T) {
	for _, env := range []string{"production", "Production", "PRODUCTION", " production "} {
		t.Run("ENVIRONMENT="+env, func(t *testing.T) {
			t.Setenv(envEnvironment, env)
			t.Setenv(EnvDevTokens, "true")

			_, _, err := validateTokenWithClaims("admin-alice")
			assert.Error(t, err, "production must refuse unsigned tokens even when opted in")

			isAdmin, _ := validateAdminRole("admin-alice", "admin-alice")
			assert.False(t, isAdmin)
		})
	}
}

// TestDGS112_OptInStillWorksOutsideProduction proves the tests above are not
// passing vacuously because the shims were deleted. The escape hatch has to
// keep working, or contributors will patch the gate back out.
func TestDGS112_OptInStillWorksOutsideProduction(t *testing.T) {
	enableDevTokens(t)

	subject, claims, err := validateTokenWithClaims("admin-alice")
	require.NoError(t, err, "an explicit opt-in outside production must still work")
	assert.Equal(t, "admin-alice", subject)
	assert.Nil(t, claims, "shim tokens are not JWTs and carry no claims")

	isAdmin, err := validateAdminRole("admin-alice", "admin-alice")
	require.NoError(t, err)
	assert.True(t, isAdmin, "the admin shim must still work when explicitly opted in")
}

// TestDGS112_UnparseableOptInFailsClosed is the fat-finger case: an opt-in that
// is not a bool is treated as absent rather than as true.
func TestDGS112_UnparseableOptInFailsClosed(t *testing.T) {
	for _, garbage := range []string{"yes", "on", "enabled", "TRUE!", "1.0", "y"} {
		t.Run(garbage, func(t *testing.T) {
			t.Setenv(envEnvironment, "development")
			t.Setenv(EnvDevTokens, garbage)

			isAdmin, _ := validateAdminRole("admin-alice", "admin-alice")
			assert.False(t, isAdmin, "%q is not a parseable bool and must not grant admin", garbage)
		})
	}
}

// TestDGS112_OneDefinitionOfProduction is the cross-gate test. DGS-112's real
// shape was two functions in the same package disagreeing about what counted as
// production, so that one hardened the host while the other opened it. Both
// must now answer identically for every spelling.
func TestDGS112_OneDefinitionOfProduction(t *testing.T) {
	for _, env := range []string{"production", "Production", "PRODUCTION", " production "} {
		t.Run(env, func(t *testing.T) {
			status := configureJWTSecretEnv(t, "", "")
			require.True(t, status.UsingDefault, "precondition: running on the development secret")
			t.Setenv(envEnvironment, env)
			t.Setenv(EnvDevTokens, "true")

			// Gate 1: the JWT startup gate must call this production and refuse
			// to run on the built-in secret.
			assert.Error(t, EnsureJWTSecretConfigured(env),
				"EnsureJWTSecretConfigured must treat %q as production", env)

			// Gate 2: the auth shim must call it production too. Before the fix
			// this disagreed with gate 1 for every spelling except the first.
			assert.True(t, IsProductionEnvironment(),
				"IsProductionEnvironment must treat %q as production", env)
			isAdmin, _ := validateAdminRole("admin-alice", "admin-alice")
			assert.False(t, isAdmin,
				"the auth shim must treat %q as production too — this is the disagreement DGS-112 was", env)
		})
	}
}

// TestDGS112_StartupWarningOnlyFiresWhenTokensAreLive keeps the warning
// meaningful — it must be silent in the safe configurations, which are the
// overwhelming majority, or it will be tuned out.
func TestDGS112_StartupWarningOnlyFiresWhenTokensAreLive(t *testing.T) {
	cases := []struct {
		env, flag string
		wantWarn  bool
		why       string
	}{
		{"development", "true", true, "opted in outside production: tokens really are live"},
		{"development", "", false, "the default is off and needs no warning"},
		{"", "", false, "an unconfigured host is now safe and silent"},
		{"production", "true", false, "refused by the production condition — nothing to warn about"},
		{"production", "", false, "the normal production case"},
		{"development", "yes", false, "unparseable opt-in is not enabled, so not warned"},
	}

	for _, tc := range cases {
		t.Run(tc.env+"/"+tc.flag, func(t *testing.T) {
			t.Setenv(envEnvironment, tc.env)
			t.Setenv(EnvDevTokens, tc.flag)

			warning := DevTokensStartupWarning()

			if tc.wantWarn {
				require.NotEmpty(t, warning, tc.why)
				assert.Contains(t, warning, EnvDevTokens)
			} else {
				assert.Empty(t, warning, tc.why)
			}
		})
	}
}
