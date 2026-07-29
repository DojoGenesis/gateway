package server

// Service-token route tests.
//
// These drive the REAL setupMiddleware() + setupRoutes() wiring via
// newRoutedTestServer (see router_auth_test.go), for the same reason DGS-88
// does: the property under test is "which middleware chain does this route
// actually get", and a hand-built gin.New() cannot observe that.
//
// POST /v1/chat is the route PDI proxies anonymous visitor chat to. It lives
// inside the /v1 group, which applies middleware.AuthMiddleware() — so before
// this change PDI had no way to authenticate at all, and after it the only
// accepted credential is still a signature-verified JWT.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/server/middleware"
)

// pdiChatRoute is the exact route PDI calls: v1.POST("/chat", chatHandler.Chat)
// at router.go, inside the AuthMiddleware-protected /v1 group.
const pdiChatRoute = "/v1/chat"

// adminRoute is an arbitrary route behind AdminAuthMiddleware.
const adminRoute = "/admin/health"

// signServiceClaims forges a service token with the gateway's own secret, so
// that malformed variants IssueServiceToken would refuse to mint can still be
// tested against the validation path.
func signServiceClaims(t *testing.T, claims middleware.GatewayClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(middleware.GetJWTSecret())
	require.NoError(t, err)
	return s
}

func newServiceClaims(name string, issuedAt, expiresAt time.Time) middleware.GatewayClaims {
	return middleware.GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   middleware.ServiceSubjectPrefix + name,
			ID:        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Role: middleware.ServiceRole,
	}
}

// assertNotRejectedByAuth asserts the request got past the auth middleware.
// It cannot assert 200: newRoutedTestServer leaves handler dependencies nil on
// purpose, so a request that reaches the chat handler fails inside it. Getting
// PAST auth is the whole property under test.
func assertNotRejectedByAuth(t *testing.T, code int, body string) {
	t.Helper()
	assert.NotEqual(t, http.StatusUnauthorized, code,
		"a valid service token must not be rejected by AuthMiddleware; body=%s", body)
	assert.NotContains(t, body, "Invalid or expired token")
	assert.NotContains(t, body, "Authorization header is required")
}

// ─── Acceptance ─────────────────────────────────────────────────────────────

func TestServiceToken_AcceptedOnV1Chat(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)

	s := newRoutedTestServer(t)

	token, jti, err := middleware.IssueServiceToken("pdi", 0)
	require.NoError(t, err)
	require.NotEmpty(t, jti)

	w := do(t, s, http.MethodPost, pdiChatRoute, token)
	assertNotRejectedByAuth(t, w.Code, w.Body.String())
}

// The gap being closed: PDI sending no Authorization header must still 401.
// This is the DGS-88 invariant — the service-token path must not have opened a
// hole for anonymous callers.
func TestServiceToken_UnauthenticatedV1ChatStill401(t *testing.T) {
	s := newRoutedTestServer(t)

	w := do(t, s, http.MethodPost, pdiChatRoute, "")

	require.Equal(t, http.StatusUnauthorized, w.Code,
		"anonymous POST /v1/chat must still be rejected; got %d body=%s", w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "Authorization header is required")
}

// ─── Least privilege on the real admin route ────────────────────────────────

func TestServiceToken_RejectedOnAdminRoute(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)

	s := newRoutedTestServer(t)

	token, _, err := middleware.IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	// Same credential, same server, one route apart.
	chat := do(t, s, http.MethodPost, pdiChatRoute, token)
	assertNotRejectedByAuth(t, chat.Code, chat.Body.String())

	admin := do(t, s, http.MethodGet, adminRoute, token)

	require.Equal(t, http.StatusForbidden, admin.Code,
		"a service token accepted on %s must be REJECTED on %s; got %d body=%s",
		pdiChatRoute, adminRoute, admin.Code, admin.Body.String())
	assert.Contains(t, admin.Body.String(), "Admin privileges required")
}

func TestServiceToken_RejectedAcrossEveryAdminRoute(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)

	s := newRoutedTestServer(t)

	token, _, err := middleware.IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	adminRoutes := []struct{ method, path string }{
		{http.MethodGet, "/admin/health"},
		{http.MethodGet, "/admin/config"},
		{http.MethodPost, "/admin/config/reload"},
		{http.MethodGet, "/admin/providers"},
		{http.MethodGet, "/admin/costs"},
		{http.MethodGet, "/admin/users"},
		{http.MethodPost, "/admin/mcp/tools/invoke"},
		{http.MethodPost, "/admin/routing/mode"},
	}

	for _, r := range adminRoutes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := do(t, s, r.method, r.path, token)
			assert.Equal(t, http.StatusForbidden, w.Code,
				"%s %s must reject a machine credential; got %d body=%s",
				r.method, r.path, w.Code, w.Body.String())
		})
	}
}

// ─── Rejection paths on the live route ──────────────────────────────────────

func TestServiceToken_ExpiredRejectedOnV1Chat(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)

	s := newRoutedTestServer(t)

	now := time.Now()
	expired := signServiceClaims(t, newServiceClaims("pdi", now.Add(-48*time.Hour), now.Add(-24*time.Hour)))

	w := do(t, s, http.MethodPost, pdiChatRoute, expired)

	require.Equal(t, http.StatusUnauthorized, w.Code,
		"an expired service token must be rejected; got %d body=%s", w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "Invalid or expired token")
}

func TestServiceToken_RevokedRejectedOnV1Chat(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)

	s := newRoutedTestServer(t)

	token, jti, err := middleware.IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	// Accepted before revocation.
	before := do(t, s, http.MethodPost, pdiChatRoute, token)
	assertNotRejectedByAuth(t, before.Code, before.Body.String())

	middleware.RevokeServiceTokenID(jti)

	after := do(t, s, http.MethodPost, pdiChatRoute, token)
	require.Equal(t, http.StatusUnauthorized, after.Code,
		"a revoked service token must be rejected without rotating JWT_SECRET; got %d body=%s",
		after.Code, after.Body.String())
	assert.Contains(t, after.Body.String(), "Invalid or expired token")
}

// Revoking a machine credential must not disturb human sessions — the whole
// reason revocation is a jti denylist and not a JWT_SECRET rotation.
func TestServiceTokenRevocation_DoesNotAffectHumanSessions(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)

	s := newRoutedTestServer(t)

	serviceToken, jti, err := middleware.IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	humanToken, err := issueToken("a-human-user", "user", time.Hour)
	require.NoError(t, err)

	middleware.RevokeServiceSubject("service:pdi")
	middleware.RevokeServiceTokenID(jti)

	svc := do(t, s, http.MethodPost, pdiChatRoute, serviceToken)
	require.Equal(t, http.StatusUnauthorized, svc.Code, "the revoked machine credential must be dead")

	human := do(t, s, http.MethodGet, "/v1/conversations", humanToken)
	assert.NotEqual(t, http.StatusUnauthorized, human.Code,
		"revoking a service token must leave every human session intact; body=%s", human.Body.String())
}

func TestServiceToken_TamperedOrForgedRejectedOnV1Chat(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)

	s := newRoutedTestServer(t)

	now := time.Now()
	valid, _, err := middleware.IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	// Mutate a byte of the signed payload.
	tampered := []byte(valid)
	dot := 0
	for i, ch := range tampered {
		if ch == '.' {
			dot = i
			break
		}
	}
	require.NotZero(t, dot)
	if tampered[dot+3] == 'A' {
		tampered[dot+3] = 'B'
	} else {
		tampered[dot+3] = 'A'
	}

	// Re-signed by someone who does not hold JWT_SECRET.
	foreign, err := jwt.NewWithClaims(jwt.SigningMethodHS256, newServiceClaims("pdi", now, now.Add(time.Hour))).
		SignedString([]byte("an-attacker-controlled-secret"))
	require.NoError(t, err)

	// Signature stripped entirely.
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, newServiceClaims("pdi", now, now.Add(time.Hour))).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	// No jti — no revocation handle, so unusable even though it is signed.
	noJTIClaims := newServiceClaims("pdi", now, now.Add(time.Hour))
	noJTIClaims.ID = ""
	noJTI := signServiceClaims(t, noJTIClaims)

	// Signed, but the subject does not name a client.
	anonClaims := newServiceClaims("pdi", now, now.Add(time.Hour))
	anonClaims.Subject = "not-namespaced"
	anonymous := signServiceClaims(t, anonClaims)

	// Lifetime beyond the hard ceiling.
	eternalClaims := newServiceClaims("pdi", now, now.Add(middleware.MaxServiceTokenTTL+48*time.Hour))
	eternal := signServiceClaims(t, eternalClaims)

	cases := map[string]string{
		"tampered payload":        string(tampered),
		"signed with wrong key":   foreign,
		"alg=none":                unsigned,
		"missing jti":             noJTI,
		"subject not namespaced":  anonymous,
		"lifetime over ceiling":   eternal,
		"guessed literal subject": "service:pdi",
		"guessed literal name":    "pdi",
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			w := do(t, s, http.MethodPost, pdiChatRoute, token)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s must be rejected at %s; got %d body=%s", name, pdiChatRoute, w.Code, w.Body.String())
		})
	}
}

// A service token must not be exchangeable for a human session. POST
// /auth/refresh is public (no AuthMiddleware), so it is the obvious place to
// try trading a machine credential for a user access token.
func TestServiceToken_CannotBeExchangedForAHumanSession(t *testing.T) {
	middleware.ResetServiceTokenRevocations()
	t.Cleanup(middleware.ResetServiceTokenRevocations)

	s := newRoutedTestServer(t)

	token, _, err := middleware.IssueServiceToken("pdi", 0)
	require.NoError(t, err)

	body := strings.NewReader(`{"refresh_token":"` + token + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code,
		"a service token must not be redeemable at /auth/refresh; got %d body=%s", w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), "access_token")
}

// ─── Blast radius ───────────────────────────────────────────────────────────

// The workflow API is the repo's SECOND handler path — an http.ServeMux wrapped
// with gin.WrapH, registered on s.router OUTSIDE the /v1 group. It shares no
// middleware with /v1, so the service-token change must not have reached it.
// The execution endpoints are always registered, so they are the observable
// part of that path in this fixture (workflowCAS is nil here, which is why the
// CRUD routes are absent).
func TestServiceToken_WorkflowServeMuxPathUnaffected(t *testing.T) {
	s := newRoutedTestServer(t)

	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/workflows/demo/execute"},
		{http.MethodGet, "/api/workflows/demo/execution"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := do(t, s, tc.method, tc.path, "")
			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"%s is registered outside the /v1 group and must keep its existing auth posture; got 401", tc.path)
		})
	}
}

// The public surface must stay public — the same guard DGS-88 installed.
func TestServiceToken_PublicRoutesStayReachable(t *testing.T) {
	s := newRoutedTestServer(t)

	for _, path := range []string{"/health", "/metrics", "/.well-known/did.json"} {
		t.Run(path, func(t *testing.T) {
			w := do(t, s, http.MethodGet, path, "")
			assert.NotEqual(t, http.StatusUnauthorized, w.Code, "%s must stay anonymous", path)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "%s should still be routed", path)
		})
	}
}
