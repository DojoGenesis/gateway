package middleware

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// The gateway signs and verifies every bearer token — human sessions and
// machine service tokens alike — with one HMAC secret. This file is the ONLY
// place that decides where that secret comes from.
//
// WHY TWO VARIABLE NAMES:
// the code has always read JWT_SECRET, but DEPLOYMENT.md, deploy/README.md and
// deploy/gateway-config.yaml told operators to set DOJO_JWT_SECRET, which no Go
// code read. An operator who followed the docs and set only DOJO_JWT_SECRET got
// a gateway signing every session with defaultJWTSecret — a constant committed
// to a repository with public mirrors, so anyone who had read it could forge a
// token, including one carrying role "admin".
//
// The fix keeps JWT_SECRET canonical and accepts DOJO_JWT_SECRET as a
// deprecated alias:
//
//   - JWT_SECRET is what every running host already signs with, so a host that
//     sets BOTH keeps exactly the secret it has today. A restart invalidates
//     nothing and logs nobody out.
//   - A host that followed the old docs and set ONLY DOJO_JWT_SECRET stops
//     silently falling back to the development secret.
//   - A production host that sets neither refuses to start — see
//     EnsureJWTSecretConfigured — instead of coming up forgeable.
//
// The secret is never logged, not even truncated. JWTSecretStatus carries the
// NAME of the variable it came from and whether it is the built-in default, and
// nothing else.

const (
	// EnvJWTSecret is the canonical environment variable holding the signing
	// secret. This is the name every doc and provisioning step must use.
	EnvJWTSecret = "JWT_SECRET"

	// EnvJWTSecretLegacy is a deprecated alias, honoured only so that a host
	// provisioned from the older documentation is not left on the development
	// secret. EnvJWTSecret wins when both are set.
	EnvJWTSecretLegacy = "DOJO_JWT_SECRET"
)

// defaultJWTSecret is the development fallback used when no secret is
// configured. It is publicly known, so anything signed with it is forgeable by
// anyone — see UsingDefaultJWTSecret, which both the service-token minting CLI
// and the production startup gate check.
const defaultJWTSecret = "dev-secret-change-in-production"

// jwtSecretConfig is the resolved secret plus where it came from.
type jwtSecretConfig struct {
	secret []byte
	// source is the environment variable the secret came from, or "" when the
	// built-in development fallback is in use.
	source string
	// bothNamesSet records that the canonical name and the deprecated alias
	// were both present. Worth surfacing: the alias is ignored, so an operator
	// who believes it is in force would rotate the wrong variable.
	bothNamesSet bool
}

// jwtSecretHolder holds the configuration in force. It is behind an atomic
// pointer because the auth hot path reads it from every request goroutine while
// startup replaces it exactly once, in LoadJWTSecretFromEnv, before the server
// accepts anything.
var jwtSecretHolder = newJWTSecretHolder()

func newJWTSecretHolder() *atomic.Pointer[jwtSecretConfig] {
	h := new(atomic.Pointer[jwtSecretConfig])
	h.Store(resolveJWTSecretFromEnv())
	return h
}

// currentJWTSecret returns the HMAC key in force. Every sign and verify path in
// this package goes through it.
func currentJWTSecret() []byte {
	return jwtSecretHolder.Load().secret
}

// resolveJWTSecretFromEnv applies the precedence rule: canonical name, then the
// deprecated alias, then the development fallback.
func resolveJWTSecretFromEnv() *jwtSecretConfig {
	canonical, hasCanonical := lookupNonBlankEnv(EnvJWTSecret)
	legacy, hasLegacy := lookupNonBlankEnv(EnvJWTSecretLegacy)

	switch {
	case hasCanonical:
		return &jwtSecretConfig{
			secret:       []byte(canonical),
			source:       EnvJWTSecret,
			bothNamesSet: hasLegacy,
		}
	case hasLegacy:
		return &jwtSecretConfig{
			secret: []byte(legacy),
			source: EnvJWTSecretLegacy,
		}
	default:
		return &jwtSecretConfig{secret: []byte(defaultJWTSecret)}
	}
}

// lookupNonBlankEnv treats a variable that is unset, empty, or whitespace-only
// as absent, so `JWT_SECRET=` cannot pass for a configured secret. The value
// itself is returned verbatim — never trimmed — so a real secret is used
// byte-for-byte as the operator set it.
func lookupNonBlankEnv(key string) (string, bool) {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

// JWTSecretStatus describes HOW the signing secret was configured. It never
// contains the secret and is safe to log.
type JWTSecretStatus struct {
	// Source is the environment variable the secret came from, or "" when the
	// built-in development fallback is in use.
	Source string
	// UsingDefault reports that this process signs with the publicly known
	// development fallback.
	UsingDefault bool
	// UsingLegacyName reports that the deprecated alias supplied the secret.
	UsingLegacyName bool
	// BothNamesSet reports that both variables were set; the deprecated one was
	// ignored.
	BothNamesSet bool
}

// SourceLabel is a log-safe description of where the secret came from.
func (s JWTSecretStatus) SourceLabel() string {
	if s.Source == "" {
		return "built-in development default"
	}
	return s.Source
}

func (c *jwtSecretConfig) status() JWTSecretStatus {
	return JWTSecretStatus{
		Source:          c.source,
		UsingDefault:    subtle.ConstantTimeCompare(c.secret, []byte(defaultJWTSecret)) == 1,
		UsingLegacyName: c.source == EnvJWTSecretLegacy,
		BothNamesSet:    c.bothNamesSet,
	}
}

// LoadJWTSecretFromEnv re-reads the signing secret from the environment and
// reports how it resolved.
//
// The package-level initialiser runs at import time, before main() has loaded
// .env, so a secret that lives only in .env would otherwise never take effect —
// exactly the silent-no-op class of bug this file exists to remove. Startup
// calls this once, after .env is loaded and before the server accepts any
// request. It is not a runtime rotation mechanism.
func LoadJWTSecretFromEnv() JWTSecretStatus {
	cfg := resolveJWTSecretFromEnv()
	jwtSecretHolder.Store(cfg)
	return cfg.status()
}

// JWTSecretConfigStatus reports the configuration in force without re-reading
// the environment.
func JWTSecretConfigStatus() JWTSecretStatus {
	return jwtSecretHolder.Load().status()
}

// UsingDefaultJWTSecret reports whether this process is signing and verifying
// with the built-in development fallback rather than a configured secret.
// The offline minting CLI refuses to issue a token when this is true, and
// EnsureJWTSecretConfigured refuses to start a production server: a token
// signed with a publicly known secret is forgeable by anyone.
func UsingDefaultJWTSecret() bool {
	return subtle.ConstantTimeCompare(currentJWTSecret(), []byte(defaultJWTSecret)) == 1
}

// ErrJWTSecretNotConfigured is the sentinel returned by
// EnsureJWTSecretConfigured when a production process would run on the built-in
// development secret.
var ErrJWTSecretNotConfigured = errors.New("JWT signing secret is not configured")

// EnsureJWTSecretConfigured is the production startup gate: a production
// gateway must never sign a session with the publicly known development secret.
//
// It has to be called from an explicit point in startup rather than from
// package initialisation. The secret is resolved by a package-level variable
// initialiser that runs at import time, before main() can load .env and before
// there is any logger or exit policy — there is no clean way to log-and-exit
// from a var initialiser. Startup must call this before the server begins
// accepting requests.
//
// Development is deliberately unaffected: the built-in default keeps local runs
// and the test suite working with no configuration at all.
func EnsureJWTSecretConfigured(environment string) error {
	if !strings.EqualFold(strings.TrimSpace(environment), "production") {
		return nil
	}
	if !UsingDefaultJWTSecret() {
		return nil
	}

	return fmt.Errorf("%w: ENVIRONMENT=production, but no signing secret was found in %s "+
		"(or the deprecated %s), so the gateway would sign and verify every session and "+
		"service token with the built-in development secret. That value is committed to "+
		"this repository, so anyone who has read it could forge a token — including one "+
		"carrying role \"admin\". Refusing to start.\n"+
		"  Fix: set %s to a strong random value and restart, for example\n"+
		"      %s=$(openssl rand -hex 32)\n"+
		"  On the systemd host that belongs in /etc/dojo/env (chmod 640, root:dojo).\n"+
		"  Note: a jwt_secret: key in config.yaml is NOT read by the gateway.",
		ErrJWTSecretNotConfigured, EnvJWTSecret, EnvJWTSecretLegacy, EnvJWTSecret, EnvJWTSecret)
}
