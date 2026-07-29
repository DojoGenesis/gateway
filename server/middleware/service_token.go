package middleware

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Service tokens are long-lived, revocable, least-privilege machine credentials.
//
// DESIGN: a service token is an ordinary HS256 JWT signed with the same
// JWT_SECRET as every human session, and it is verified by exactly the same
// jwt.ParseWithClaims call in validateToken(). There is no static-string
// comparison, no side channel, and no second trust surface — the service-role
// checks below run strictly AFTER signature verification and can only ever
// REJECT a token that the signature check already accepted. They can never
// admit one it rejected.
//
// What distinguishes a service token from a human session is three claims:
//
//	sub  "service:<name>"  — identifies the calling system in gateway logs
//	role "service"         — never "admin"; see validateAdminRole
//	jti  <uuid>            — the revocation handle
//
// REVOCATION (why a jti denylist and not JWT_SECRET rotation):
// rotating the shared secret invalidates every human session at the same time,
// so it is unusable as a routine revocation path. A jti denylist revokes one
// credential and touches nothing else. It is held in process memory and seeded
// from the environment at startup, so the auth hot path acquires no database
// dependency — a revocation check that could fail open on a DB blip would be
// worse than no check at all, and one that failed closed would take the
// gateway down. The list stays bounded because every service token has a
// mandatory expiry (see MaxServiceTokenTTL): once a revoked token's own expiry
// passes, its denylist entry can be dropped.
//
// Subject-level revocation (SERVICE_TOKEN_REVOKED_SUBJECTS) covers the case
// where the jti was not recorded: it kills every token ever issued to that
// service in one move, which is the per-service key-version idea expressed
// through the same mechanism.

const (
	// ServiceRole is the value carried in GatewayClaims.Role by a machine
	// credential. It is deliberately neither "admin" nor "user":
	// AdminAuthMiddleware grants only on Role == "admin", so a service token
	// can never satisfy it.
	ServiceRole = "service"

	// ServiceSubjectPrefix namespaces machine subjects so that gateway logs and
	// every downstream consumer of user_id can tell a machine from a human at a
	// glance — e.g. "service:pdi".
	ServiceSubjectPrefix = "service:"

	// DefaultServiceTokenTTL is the lifetime used when the operator does not
	// specify one. Long-lived, but never infinite.
	DefaultServiceTokenTTL = 90 * 24 * time.Hour

	// MaxServiceTokenTTL is the hard ceiling on a service token's lifetime.
	// It is enforced at BOTH issuance and validation, so a token minted outside
	// this code path (or by a future bug in it) still cannot outlive the cap.
	MaxServiceTokenTTL = 365 * 24 * time.Hour

	// EnvRevokedTokenIDs holds a comma-separated list of revoked jti values.
	EnvRevokedTokenIDs = "SERVICE_TOKEN_REVOKED_IDS"

	// EnvRevokedSubjects holds a comma-separated list of revoked subjects
	// (e.g. "service:pdi"), which revokes every token issued to that service.
	EnvRevokedSubjects = "SERVICE_TOKEN_REVOKED_SUBJECTS"
)

// Service-token rejection reasons. These are logged, never returned to the
// caller — AuthMiddleware answers every failure with the same generic message
// so an attacker cannot distinguish "revoked" from "expired" from "forged".
var (
	ErrServiceTokenBadSubject   = errors.New("service token subject must be \"service:<name>\"")
	ErrServiceTokenMissingID    = errors.New("service token is missing the jti claim required for revocation")
	ErrServiceTokenNoExpiry     = errors.New("service token is missing a mandatory expiry")
	ErrServiceTokenNoIssuedAt   = errors.New("service token is missing a mandatory issued-at claim")
	ErrServiceTokenLifetime     = errors.New("service token lifetime exceeds the maximum permitted")
	ErrServiceTokenRevoked      = errors.New("service token has been revoked")
	ErrServiceTokenInvalidName  = errors.New("service name must match [a-z0-9][a-z0-9_-]{0,63}")
	ErrServiceTokenTTLTooLong   = errors.New("requested TTL exceeds MaxServiceTokenTTL")
	errServiceTokenNilClaimsBug = errors.New("service token claims were nil")
)

// serviceNamePattern keeps service names log-safe and prevents a name from
// smuggling a ":" into the subject and impersonating another namespace.
var serviceNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// revocations is the in-process denylist. Seeded from the environment at
// package init and extendable at runtime via RevokeServiceTokenID /
// RevokeServiceSubject.
var revocations = struct {
	mu       sync.RWMutex
	ids      []string
	subjects []string
}{}

func init() {
	LoadServiceTokenRevocationsFromEnv()
}

// LoadServiceTokenRevocationsFromEnv replaces the denylist with the contents of
// EnvRevokedTokenIDs and EnvRevokedSubjects. Called at package init; exported
// so a configuration reload can re-read it without restarting the process.
func LoadServiceTokenRevocationsFromEnv() {
	ids := splitAndTrim(os.Getenv(EnvRevokedTokenIDs))
	subjects := splitAndTrim(os.Getenv(EnvRevokedSubjects))

	revocations.mu.Lock()
	defer revocations.mu.Unlock()
	revocations.ids = ids
	revocations.subjects = subjects
}

// ResetServiceTokenRevocations clears the denylist. Intended for tests.
func ResetServiceTokenRevocations() {
	revocations.mu.Lock()
	defer revocations.mu.Unlock()
	revocations.ids = nil
	revocations.subjects = nil
}

// RevokeServiceTokenID adds a single jti to the denylist for the lifetime of
// this process. Persist it in EnvRevokedTokenIDs so it survives a restart.
func RevokeServiceTokenID(jti string) {
	if jti = strings.TrimSpace(jti); jti == "" {
		return
	}
	revocations.mu.Lock()
	defer revocations.mu.Unlock()
	revocations.ids = append(revocations.ids, jti)
}

// RevokeServiceSubject revokes every token issued to a subject (e.g.
// "service:pdi") for the lifetime of this process. Persist it in
// EnvRevokedSubjects so it survives a restart.
func RevokeServiceSubject(subject string) {
	if subject = strings.TrimSpace(subject); subject == "" {
		return
	}
	revocations.mu.Lock()
	defer revocations.mu.Unlock()
	revocations.subjects = append(revocations.subjects, subject)
}

// isServiceTokenRevoked reports whether the jti or the subject is denylisted.
//
// The jti is not secret material — it travels in the token payload and an
// attacker can already read their own. The comparison is nevertheless
// constant-time and scans the whole list without early exit, so neither the
// contents nor the size of the denylist can be inferred from response timing.
func isServiceTokenRevoked(jti, subject string) bool {
	revocations.mu.RLock()
	defer revocations.mu.RUnlock()

	return containsConstantTime(revocations.ids, jti) ||
		containsConstantTime(revocations.subjects, subject)
}

// containsConstantTime reports whether candidate appears in list, comparing
// every entry with crypto/subtle and never short-circuiting on a match.
func containsConstantTime(list []string, candidate string) bool {
	match := 0
	for _, entry := range list {
		match |= subtle.ConstantTimeCompare([]byte(entry), []byte(candidate))
	}
	return match == 1
}

func splitAndTrim(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// validateServiceClaims applies the machine-credential rules that sit ON TOP OF
// — never instead of — the JWT signature verification performed by
// validateTokenWithClaims. It is only ever reached once the signature has
// already been verified against jwtSecret, and it can only reject.
func validateServiceClaims(claims *GatewayClaims, subject string) error {
	if claims == nil {
		return errServiceTokenNilClaimsBug
	}

	// Identifiable: the subject must name the client.
	if !strings.HasPrefix(subject, ServiceSubjectPrefix) || len(subject) == len(ServiceSubjectPrefix) {
		return ErrServiceTokenBadSubject
	}

	// Revocable: no jti means no revocation handle, so the token is unusable.
	if claims.ID == "" {
		return ErrServiceTokenMissingID
	}

	// Bounded lifetime: long-lived is fine, infinite is not. Enforced here as
	// well as at issuance so the ceiling holds even for a token this code did
	// not mint.
	if claims.ExpiresAt == nil {
		return ErrServiceTokenNoExpiry
	}
	if claims.IssuedAt == nil {
		return ErrServiceTokenNoIssuedAt
	}
	if claims.ExpiresAt.Sub(claims.IssuedAt.Time) > MaxServiceTokenTTL {
		return ErrServiceTokenLifetime
	}

	if isServiceTokenRevoked(claims.ID, subject) {
		return ErrServiceTokenRevoked
	}

	return nil
}

// IssueServiceToken mints a signed service token for the named client.
//
// It returns the token and its jti. The jti is the revocation handle: record
// it, because revoking by jti is the only way to kill one token without
// killing every token issued to the service.
//
// The caller is responsible for gating issuance. There is no unauthenticated
// path to this function: it is reached only from the offline operator CLI
// (server/cmd/servicetoken), which runs on the gateway host with JWT_SECRET in
// its environment.
func IssueServiceToken(name string, ttl time.Duration) (token string, jti string, err error) {
	if !serviceNamePattern.MatchString(name) {
		return "", "", ErrServiceTokenInvalidName
	}
	if ttl <= 0 {
		ttl = DefaultServiceTokenTTL
	}
	if ttl > MaxServiceTokenTTL {
		return "", "", fmt.Errorf("%w: %s > %s", ErrServiceTokenTTLTooLong, ttl, MaxServiceTokenTTL)
	}

	now := time.Now()
	jti = uuid.NewString()

	claims := GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   ServiceSubjectPrefix + name,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Role: ServiceRole,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign service token: %w", err)
	}

	return signed, jti, nil
}

// IsServiceSubject reports whether a subject belongs to the machine namespace.
func IsServiceSubject(subject string) bool {
	return strings.HasPrefix(subject, ServiceSubjectPrefix) &&
		len(subject) > len(ServiceSubjectPrefix)
}

// ServiceNameFromSubject returns the client name from a service subject, or ""
// when the subject is not a service subject.
func ServiceNameFromSubject(subject string) string {
	if !IsServiceSubject(subject) {
		return ""
	}
	return strings.TrimPrefix(subject, ServiceSubjectPrefix)
}

// UsingDefaultJWTSecret reports whether this process is signing and verifying
// with the built-in development fallback rather than a configured JWT_SECRET.
// The offline minting CLI refuses to issue a token when this is true: a token
// signed with a publicly known secret is forgeable by anyone.
func UsingDefaultJWTSecret() bool {
	return subtle.ConstantTimeCompare(jwtSecret, []byte(defaultJWTSecret)) == 1
}
