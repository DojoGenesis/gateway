package middleware

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These are obviously-fake placeholders. No real secret value belongs in a
// test, a fixture, or any other file in this repository.
const (
	fakeCanonicalSecret = "test-only-canonical-value-not-a-real-secret"
	fakeLegacySecret    = "test-only-legacy-value-not-a-real-secret"
)

// configureJWTSecretEnv points the package at the given environment and
// re-resolves the secret, restoring both the environment and the previously
// resolved secret when the test ends.
//
// An empty string means "variable not set" — lookupNonBlankEnv treats unset,
// empty and whitespace-only identically.
//
// t.Setenv makes the test non-parallel by construction, which is what keeps
// the swap of the package-level secret safe under -race.
func configureJWTSecretEnv(t *testing.T, canonical, legacy string) JWTSecretStatus {
	t.Helper()

	previous := jwtSecretHolder.Load()
	t.Cleanup(func() { jwtSecretHolder.Store(previous) })

	t.Setenv(EnvJWTSecret, canonical)
	t.Setenv(EnvJWTSecretLegacy, legacy)

	return LoadJWTSecretFromEnv()
}

// ---------------------------------------------------------------------------
// The startup gate
// ---------------------------------------------------------------------------

// A production gateway with no configured secret would sign and verify every
// session with a constant published in this repository. It must refuse to run.
func TestEnsureJWTSecretConfigured_ProductionWithDefaultSecretRefuses(t *testing.T) {
	status := configureJWTSecretEnv(t, "", "")
	require.True(t, status.UsingDefault, "precondition: no secret configured")

	err := EnsureJWTSecretConfigured("production")

	require.Error(t, err, "production must not start on the built-in development secret")
	assert.ErrorIs(t, err, ErrJWTSecretNotConfigured)

	// The error has to tell the operator exactly which variable to set.
	assert.Contains(t, err.Error(), EnvJWTSecret,
		"the error must name the variable that actually works")

	// ...and must never carry the secret itself, not even truncated.
	assert.NotContains(t, err.Error(), defaultJWTSecret,
		"the startup error must never print the signing secret")
}

// The running production host has a real secret; it must keep starting.
func TestEnsureJWTSecretConfigured_ProductionWithConfiguredSecretStarts(t *testing.T) {
	status := configureJWTSecretEnv(t, fakeCanonicalSecret, "")
	require.False(t, status.UsingDefault, "precondition: a secret is configured")

	assert.NoError(t, EnsureJWTSecretConfigured("production"),
		"a production host with a configured secret must start")
}

// Local development must keep working with zero configuration — the gate is a
// production-only constraint, not a new setup step for contributors.
func TestEnsureJWTSecretConfigured_DevelopmentWithDefaultSecretStarts(t *testing.T) {
	status := configureJWTSecretEnv(t, "", "")
	require.True(t, status.UsingDefault, "precondition: no secret configured")

	for _, environment := range []string{"development", "", "staging", "test"} {
		assert.NoError(t, EnsureJWTSecretConfigured(environment),
			"environment %q must not be gated", environment)
	}
}

// The deprecated alias satisfies the gate too: a host provisioned from the old
// documentation is configured, just under the wrong name.
func TestEnsureJWTSecretConfigured_ProductionWithLegacyNameStarts(t *testing.T) {
	status := configureJWTSecretEnv(t, "", fakeLegacySecret)
	require.False(t, status.UsingDefault)

	assert.NoError(t, EnsureJWTSecretConfigured("production"))
}

// "production" is matched case-insensitively and tolerates stray whitespace, so
// ENVIRONMENT=Production does not quietly skip the gate.
func TestEnsureJWTSecretConfigured_ProductionSpellingVariants(t *testing.T) {
	configureJWTSecretEnv(t, "", "")

	for _, environment := range []string{"production", "Production", "PRODUCTION", " production "} {
		err := EnsureJWTSecretConfigured(environment)
		assert.ErrorIs(t, err, ErrJWTSecretNotConfigured,
			"environment %q must be treated as production", environment)
	}
}

// A blank assignment (JWT_SECRET=) is not a configured secret.
func TestEnsureJWTSecretConfigured_BlankSecretIsNotConfigured(t *testing.T) {
	status := configureJWTSecretEnv(t, "   ", "\t")

	assert.True(t, status.UsingDefault, "a whitespace-only value must not pass for a secret")
	assert.ErrorIs(t, EnsureJWTSecretConfigured("production"), ErrJWTSecretNotConfigured)
}

// An operator who pastes the published development constant into JWT_SECRET is
// no better off than one who set nothing.
func TestEnsureJWTSecretConfigured_ExplicitDefaultValueIsRejected(t *testing.T) {
	configureJWTSecretEnv(t, defaultJWTSecret, "")

	assert.ErrorIs(t, EnsureJWTSecretConfigured("production"), ErrJWTSecretNotConfigured,
		"setting the published default explicitly must not satisfy the gate")
}

// ---------------------------------------------------------------------------
// Precedence between the canonical name and the deprecated alias
// ---------------------------------------------------------------------------

// The canonical name wins. This is the property that makes the change safe for
// the already-running production host, which sets both: it keeps signing with
// exactly the secret it signs with today, so a restart logs nobody out.
func TestJWTSecretPrecedence_CanonicalWinsOverLegacy(t *testing.T) {
	status := configureJWTSecretEnv(t, fakeCanonicalSecret, fakeLegacySecret)

	assert.Equal(t, fakeCanonicalSecret, string(GetJWTSecret()),
		"%s must win over %s", EnvJWTSecret, EnvJWTSecretLegacy)
	assert.Equal(t, EnvJWTSecret, status.Source)
	assert.False(t, status.UsingLegacyName)
	assert.True(t, status.BothNamesSet, "both names set must be reported so it can be warned about")
	assert.False(t, status.UsingDefault)
}

// The alias alone is honoured. This is what closes the original trap: a host
// provisioned from the old docs is no longer silently on the default.
func TestJWTSecretPrecedence_LegacyUsedWhenCanonicalAbsent(t *testing.T) {
	status := configureJWTSecretEnv(t, "", fakeLegacySecret)

	assert.Equal(t, fakeLegacySecret, string(GetJWTSecret()),
		"%s alone must configure the gateway", EnvJWTSecretLegacy)
	assert.Equal(t, EnvJWTSecretLegacy, status.Source)
	assert.True(t, status.UsingLegacyName, "using the deprecated name must be reported so it can be warned about")
	assert.False(t, status.BothNamesSet)
	assert.False(t, status.UsingDefault)
}

// Neither set: the development fallback, reported as such.
func TestJWTSecretPrecedence_FallsBackToDevelopmentDefault(t *testing.T) {
	status := configureJWTSecretEnv(t, "", "")

	assert.True(t, status.UsingDefault)
	assert.Empty(t, status.Source)
	assert.False(t, status.UsingLegacyName)
	assert.Equal(t, "built-in development default", status.SourceLabel())
}

// A blank canonical value must fall through to the alias rather than shadowing
// it — otherwise `JWT_SECRET=` in a stray .env would re-arm the original trap
// on a host that had correctly set the alias.
func TestJWTSecretPrecedence_BlankCanonicalFallsThroughToLegacy(t *testing.T) {
	status := configureJWTSecretEnv(t, "", fakeLegacySecret)

	assert.Equal(t, fakeLegacySecret, string(GetJWTSecret()))
	assert.Equal(t, EnvJWTSecretLegacy, status.Source)
}

// The secret is used byte-for-byte: resolution must not trim, pad or otherwise
// rewrite an operator's value, or tokens minted before the change would stop
// verifying after it.
func TestJWTSecretResolution_UsesValueVerbatim(t *testing.T) {
	padded := "  " + fakeCanonicalSecret + "  "
	configureJWTSecretEnv(t, padded, "")

	assert.Equal(t, padded, string(GetJWTSecret()),
		"the configured value must be used exactly as set, never trimmed")
}

// ---------------------------------------------------------------------------
// Status reporting must stay log-safe
// ---------------------------------------------------------------------------

// Requirement: report presence/absence and default-ness, nothing more. Status
// is logged at startup, so it must never carry secret material.
func TestJWTSecretStatus_NeverCarriesTheSecret(t *testing.T) {
	status := configureJWTSecretEnv(t, fakeCanonicalSecret, fakeLegacySecret)

	assert.NotContains(t, status.Source, fakeCanonicalSecret)
	assert.NotContains(t, status.SourceLabel(), fakeCanonicalSecret)
	assert.Equal(t, EnvJWTSecret, status.SourceLabel(),
		"the label must be the variable NAME, not its value")
}

// JWTSecretConfigStatus reports what is in force without re-reading the
// environment, so a later env change cannot make the startup log lie.
func TestJWTSecretConfigStatus_ReflectsLoadedState(t *testing.T) {
	loaded := configureJWTSecretEnv(t, fakeCanonicalSecret, "")

	assert.Equal(t, loaded, JWTSecretConfigStatus())

	// Changing the environment without reloading must not change what is in
	// force — the gateway keeps signing with the secret it started with.
	t.Setenv(EnvJWTSecret, fakeLegacySecret)
	assert.Equal(t, loaded, JWTSecretConfigStatus())
	assert.Equal(t, fakeCanonicalSecret, string(GetJWTSecret()))
}

// ---------------------------------------------------------------------------
// Wiring: the resolved secret is the one that actually signs and verifies
// ---------------------------------------------------------------------------

// A configured secret must reach the real verification path, not just the
// status struct. A token minted under one secret must not verify under another.
func TestJWTSecret_IsTheKeyUsedForSigningAndVerifying(t *testing.T) {
	configureJWTSecretEnv(t, fakeCanonicalSecret, "")
	t.Setenv("ENVIRONMENT", "production") // keep the dev token shim out of the way
	ResetServiceTokenRevocations()

	token, _, err := IssueServiceToken("pdi", DefaultServiceTokenTTL)
	require.NoError(t, err)

	subject, claims, err := validateTokenWithClaims(token)
	require.NoError(t, err, "a token must verify under the secret that signed it")
	assert.Equal(t, ServiceSubjectPrefix+"pdi", subject)
	require.NotNil(t, claims)

	// Re-resolve to a different secret: the same token must now fail.
	configureJWTSecretEnv(t, fakeLegacySecret, "")
	_, _, err = validateTokenWithClaims(token)
	assert.Error(t, err, "a token signed under a different secret must not verify")
}

// UsingDefaultJWTSecret is what the offline minting CLI checks before it will
// issue a machine credential. It must track the resolved secret.
func TestUsingDefaultJWTSecret_TracksResolvedSecret(t *testing.T) {
	configureJWTSecretEnv(t, "", "")
	assert.True(t, UsingDefaultJWTSecret(), "no configuration means the development fallback")

	configureJWTSecretEnv(t, fakeCanonicalSecret, "")
	assert.False(t, UsingDefaultJWTSecret(), "a configured secret is not the default")

	configureJWTSecretEnv(t, "", fakeLegacySecret)
	assert.False(t, UsingDefaultJWTSecret(), "the deprecated alias also counts as configured")
}

// ---------------------------------------------------------------------------
// Regression guard on the trap itself
// ---------------------------------------------------------------------------

// The original bug: the docs named DOJO_JWT_SECRET, the code read only
// JWT_SECRET, and a host that set only the documented name came up signing
// every session with a publicly known constant — with no error at all.
func TestRegression_LegacyOnlyHostIsNoLongerOnThePublishedDefault(t *testing.T) {
	status := configureJWTSecretEnv(t, "", fakeLegacySecret)

	require.False(t, status.UsingDefault,
		"a host that set only %s must not be signing with the published development secret",
		EnvJWTSecretLegacy)
	require.NoError(t, EnsureJWTSecretConfigured("production"))

	assert.NotEqual(t, defaultJWTSecret, string(GetJWTSecret()))
}

// The gate must fail closed: any non-nil error from it wraps the sentinel, so a
// caller matching on ErrJWTSecretNotConfigured cannot miss a refusal.
func TestEnsureJWTSecretConfigured_ErrorAlwaysWrapsSentinel(t *testing.T) {
	configureJWTSecretEnv(t, "", "")

	err := EnsureJWTSecretConfigured("production")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJWTSecretNotConfigured))
	assert.True(t, strings.Contains(err.Error(), "Refusing to start"),
		"the message must state plainly that startup is being refused")
}
