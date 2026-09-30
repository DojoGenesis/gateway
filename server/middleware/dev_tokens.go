package middleware

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

// This file decides ONE thing: may this process accept the legacy development
// shim tokens — unsigned literal strings like "admin-alice" — as credentials?
//
// WHY IT REPLACED isDevelopment() (DGS-112):
// the gate used to be
//
//	func isDevelopment() bool { return os.Getenv("ENVIRONMENT") != "production" }
//
// an exact byte comparison, negated. Every value other than the literal
// lowercase "production" selected development: "Production", "PRODUCTION",
// "prod", "production " with a trailing space, and — the common case — unset.
// In that state `Authorization: Bearer admin-<anything>` both authenticated
// (validateTokenWithClaims) and returned isAdmin (validateAdminRole), with no
// signature and no secret involved, reaching /admin/*, all of /v1/*, and the
// destructive POST /api/cas/gc.
//
// The bug was not the string comparison. It was the DIRECTION of the default:
// absence of configuration granted credentials. Fixing only the comparison
// would leave a control that still fails open for any value nobody thought of.
// So the default is inverted here — the shims are off unless an operator has
// explicitly asked for them, on a host that is not production. Nothing an
// operator forgets to set can turn them on.
//
// The environment check is kept as a SECOND condition, not the first one, and
// that ordering is the point: if ENVIRONMENT is misspelled on a production
// host, the misspelling no longer opens anything, because GATEWAY_DEV_TOKENS
// is still unset. It can only ever refuse, never permit.

// EnvDevTokens is the explicit opt-in for the legacy development shim tokens.
//
// Unset means OFF. There is deliberately no value of ENVIRONMENT, and no
// combination of other settings, that turns these tokens on by itself.
const EnvDevTokens = "GATEWAY_DEV_TOKENS"

// envEnvironment names the deployment environment. It is read in exactly one
// place, the fallback in IsProductionEnvironment below; every other decision
// uses the resolved value main records with SetResolvedEnvironment.
const envEnvironment = "ENVIRONMENT"

// devTokensEnabled reports whether unsigned shim tokens may authenticate.
//
// Read live rather than cached: this runs per request-with-a-shim-token, which
// is a development-only path where the cost is irrelevant, and reading live
// lets a test flip it with t.Setenv without a process restart.
func devTokensEnabled() bool {
	if IsProductionEnvironment() {
		// Belt and braces. Even an explicit GATEWAY_DEV_TOKENS=true does not
		// hand out unsigned credentials on a production host.
		return false
	}

	raw, ok := lookupNonBlankEnv(EnvDevTokens)
	if !ok {
		return false
	}

	// An unparseable value ("yes", "on", "1.0") is treated as absent rather
	// than as true, so a fat-fingered opt-in fails closed.
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && v
}

// resolvedEnvironment is the process's environment as the config resolved it:
// config.yaml's `environment:`, overridden by the ENVIRONMENT env var (the
// documented config order). main sets it once at startup, before any request.
//
// It exists because "production" used to have two sources. The JWT startup
// gate read the resolved config; this file read the raw env var. On a host
// whose production setting lived in config.yaml alone, the JWT gate said
// production and the dev-token and run_command refusals did not, so an
// explicit GATEWAY_DEV_TOKENS=true handed out unsigned admin tokens.
var resolvedEnvironment atomic.Pointer[string]

// SetResolvedEnvironment records the process's resolved environment. Call it
// once at startup with the config's final Environment value. An empty string
// clears it (tests), restoring the env-var fallback below.
func SetResolvedEnvironment(env string) {
	if env == "" {
		resolvedEnvironment.Store(nil)
		return
	}
	resolvedEnvironment.Store(&env)
}

// IsProductionEnvironment reports whether this process considers itself
// production. Every production decision in the tree goes through here or
// through IsProductionString: the JWT startup gate, the dev-token and
// workflow run_command refusals, gin's mode and HSTS.
//
// It answers from the resolved environment when main has set one, and falls
// back to the ENVIRONMENT env var otherwise (unit tests, and any binary that
// never loads the gateway config).
func IsProductionEnvironment() bool {
	if p := resolvedEnvironment.Load(); p != nil {
		return isProductionString(*p)
	}
	return isProductionString(os.Getenv(envEnvironment))
}

// IsProductionString applies the single definition of "production" to a
// value the caller already holds (gin's mode, HSTS).
func IsProductionString(s string) bool {
	return isProductionString(s)
}

// isProductionString is the single definition of "production" in this codebase.
//
// Case- and whitespace-insensitive on purpose. "Production" and " production "
// are the same deployment to any operator, and a security control that
// disagreed with that reading is what DGS-112 was.
func isProductionString(s string) bool {
	return strings.EqualFold(strings.TrimSpace(s), "production")
}

// DevTokensStartupWarning returns a log line when this process will accept
// unsigned shim tokens, and "" otherwise.
//
// Enabling them is legitimate on a developer machine and should still be said
// out loud, because the same binary runs in both places and the failure mode is
// silent: nothing about a running gateway looks different until someone
// presents `Bearer admin-x` and is believed.
func DevTokensStartupWarning() string {
	if !devTokensEnabled() {
		return ""
	}
	return fmt.Sprintf(
		"%s=true: unsigned development tokens are accepted on this gateway. "+
			"`Authorization: Bearer admin-<anything>` is a full admin credential, with no "+
			"signature check. This is refused outright when ENVIRONMENT=production, but never "+
			"enable it on any host reachable by someone you would not give admin to (DGS-112)",
		EnvDevTokens)
}
