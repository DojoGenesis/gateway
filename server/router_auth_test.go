package server

// DGS-88 regression tests.
//
// Incident (confirmed 2026-07-20): from residential internet with zero
// credentials, POST /v1/gateway/agents created an agent and
// POST /v1/gateway/agents/:id/chat streamed OpenRouter completions — real
// billed provider spend.
//
// Root cause was a WIRING bug, not a handler bug: `v1.Group("/gateway")` had
// no `.Use()`, so the only middleware reaching it was the global
// OptionalAuthMiddleware, which never rejects — it assigns a random guest UUID
// and calls c.Next().
//
// Because the bug lived in setupRoutes(), these tests MUST drive the real
// setupMiddleware() + setupRoutes() wiring. A test that builds its own
// gin.New() and attaches handlers by hand (as the other tests in this package
// legitimately do, since they target handler behaviour) cannot observe this
// class of bug at all.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/server/middleware"
)

// newRoutedTestServer builds a *Server carrying only what the routing layer
// needs, then runs the real middleware + route registration. Handler
// dependencies are left nil on purpose: every assertion here is about whether
// a request is REJECTED before it reaches a handler, so the handlers must
// never run on the passing path.
//
// AuthMode is "api_key" to reproduce the production configuration exactly —
// that is the branch in setupMiddleware() that installs the permissive global
// OptionalAuthMiddleware which made the incident possible.
func newRoutedTestServer(t *testing.T) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	s := &Server{
		router: gin.New(),
		cfg: &ServerConfig{
			Port:           "7340",
			Environment:    "test",
			AuthMode:       "api_key",
			AllowedOrigins: []string{"*"},
		},
		agents:         map[string]*AgentRuntime{},
		orchestrations: NewOrchestrationStore(),
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

// do issues a request against the fully-wired router. An empty token means no
// Authorization header at all — the anonymous-internet case from the incident.
func do(t *testing.T, s *Server, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

// TestDGS88_UnauthenticatedGatewayAgentCreateIsRejected is THE regression test.
// This exact request, unauthenticated, created a real agent in production.
func TestDGS88_UnauthenticatedGatewayAgentCreateIsRejected(t *testing.T) {
	s := newRoutedTestServer(t)

	w := do(t, s, http.MethodPost, "/v1/gateway/agents", "")

	require.Equal(t, http.StatusUnauthorized, w.Code,
		"POST /v1/gateway/agents with no Authorization header must be rejected with 401 before reaching the handler; got %d body=%s",
		w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "Authorization header is required")
}

// TestDGS88_UnauthenticatedMoneyAndStateRoutesAreRejected covers the rest of
// the /v1 surface that spends money or mutates durable state. The incident
// report also names unauthenticated memory writes, which protecting only
// /v1/gateway would not have closed — hence the group-level attachment.
func TestDGS88_UnauthenticatedMoneyAndStateRoutesAreRejected(t *testing.T) {
	s := newRoutedTestServer(t)

	cases := []struct {
		method, path, why string
	}{
		{http.MethodPost, "/v1/gateway/agents", "creates an agent"},
		{http.MethodPost, "/v1/gateway/agents/abc/chat", "streams billed completions"},
		{http.MethodPost, "/v1/gateway/orchestrate", "executes a billed DAG"},
		{http.MethodGet, "/v1/gateway/agents", "lists other tenants' agents"},
		{http.MethodPost, "/v1/chat/completions", "billed completion"},
		{http.MethodPost, "/v1/chat", "billed completion (legacy)"},
		{http.MethodPost, "/v1/orchestrate", "billed orchestration"},
		{http.MethodPost, "/v1/memory", "durable memory write"},
		{http.MethodGet, "/v1/memory", "reads private memory"},
		{http.MethodPost, "/v1/memory/search", "reads private memory"},
		{http.MethodPost, "/v1/seeds", "durable seed write"},
		{http.MethodPost, "/v1/snapshots", "durable snapshot write"},
		{http.MethodPost, "/v1/settings/providers", "writes provider API keys"},
		{http.MethodPost, "/v1/tools/invoke", "invokes tools"},
		{http.MethodGet, "/v1/models", "enumerates private model inventory"},
		// Already had their own AuthMiddleware before this fix — asserted so a
		// future refactor cannot silently regress them either.
		{http.MethodGet, "/v1/conversations", "reads private conversations"},
		{http.MethodGet, "/v1/templates", "reads private templates"},
		{http.MethodGet, "/v1/documents", "reads private documents"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := do(t, s, tc.method, tc.path, "")
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s %s (%s) must be 401 when unauthenticated; got %d body=%s",
				tc.method, tc.path, tc.why, w.Code, w.Body.String())
		})
	}
}

// TestDGS88_PublicRoutesStayReachable is the blast-radius guard. The fix is
// group-wide, so it must be proven NOT to have swallowed the routes that are
// legitimately anonymous — in particular /health, which deploy/provision.sh
// uses as an unauthenticated liveness probe.
func TestDGS88_PublicRoutesStayReachable(t *testing.T) {
	s := newRoutedTestServer(t)

	for _, path := range []string{"/health", "/metrics", "/.well-known/did.json"} {
		t.Run(path, func(t *testing.T) {
			w := do(t, s, http.MethodGet, path, "")
			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"%s is registered on s.router outside the /v1 group and must stay anonymous; got 401", path)
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"%s should still be routed", path)
		})
	}
}

// TestDGS88_DoubleAuthAttachmentIsHarmless resolves the question raised by the
// group-level attachment: /v1/conversations, /v1/templates and /v1/documents
// each already call .Use(AuthMiddleware()) on their own subgroup, so
// AuthMiddleware now runs TWICE on those routes.
//
// It is safe: AuthMiddleware only reads a header, parses a stateless JWT, and
// c.Set()s two keys to identical values. It consumes no nonce, increments no
// counter, and writes no duplicate response. The redundant attachments are
// kept deliberately as defence-in-depth against exactly the regression that
// caused DGS-88 — if the group-level Use() is ever moved or dropped, these
// three subgroups stay protected on their own.
//
// This test proves the second pass does not reject a request the first pass
// accepted, and that a rejection still emits exactly one 401 body.
func TestDGS88_DoubleAuthAttachmentIsHarmless(t *testing.T) {
	// validateToken() accepts the "test-token" shim only when explicitly opted
	// in on a non-production host (DGS-112 — "not production" alone used to be
	// enough, which made any unconfigured host hand out admin).
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv(middleware.EnvDevTokens, "true")

	s := newRoutedTestServer(t)

	t.Run("valid token survives both passes", func(t *testing.T) {
		w := do(t, s, http.MethodGet, "/v1/conversations", "test-token")
		assert.NotEqual(t, http.StatusUnauthorized, w.Code,
			"a token accepted by the first AuthMiddleware must not be rejected by the second; got 401 body=%s",
			w.Body.String())
	})

	t.Run("rejection emits exactly one 401 body", func(t *testing.T) {
		w := do(t, s, http.MethodGet, "/v1/conversations", "")
		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, 1, strings.Count(w.Body.String(), "Authorization header is required"),
			"double attachment must not write the error body twice; c.Abort() should stop the chain. body=%s",
			w.Body.String())
	})
}
