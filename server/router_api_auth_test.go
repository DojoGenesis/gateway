package server

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/runtime/cas"
	"github.com/DojoGenesis/gateway/server/middleware"
)

// DGS-100 regression tests: /api/* must not serve anonymous callers.
//
// Like the DGS-88 and DGS-100 /gc tests, these are WIRING tests and therefore
// drive the real setupMiddleware()+setupRoutes(). A test that builds its own
// gin.New() and attaches handlers by hand cannot observe a missing middleware
// at all — it is the registration that was wrong, never the handler.
//
// Two independent handler paths serve this prefix and share no middleware:
// gin routes on s.router, and a bare http.ServeMux mounted through gin.WrapH
// (workflow/api/handler.go). The table below deliberately covers both, because
// checking one proves nothing about the other — the repo's CLAUDE.md calls this
// out as the single biggest source of handler bugs here.

func newRoutedAPITestServer(t *testing.T) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// A real CAS store: the /api/cas and /api/workflows groups are registered
	// under `if s.workflowCAS != nil`, so a nil store would leave them
	// unregistered and every assertion below would pass vacuously against a 404.
	store, err := cas.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	s := &Server{
		router:      gin.New(),
		workflowCAS: store,
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

// gatedAPIRoutes is the full surface DGS-100 closes. Path-2 rows are the
// ServeMux-backed routes; they are the ones a fix applied only to gin would
// silently miss.
var gatedAPIRoutes = []struct {
	method, path, note string
}{
	// Path 2 — http.ServeMux via gin.WrapH
	{http.MethodGet, "/api/workflows", "path-2 list"},
	{http.MethodPost, "/api/workflows", "path-2 create — stored the DGS-108 payload"},
	{http.MethodGet, "/api/workflows/demo", "path-2 get"},
	{http.MethodGet, "/api/workflows/demo/canvas", "path-2 canvas read"},
	{http.MethodPut, "/api/workflows/demo/canvas", "path-2 canvas write"},
	{http.MethodPost, "/api/workflows/demo/validate", "path-2 validate"},
	{http.MethodGet, "/api/skills", "path-2 skills"},

	// Path 1 — gin
	{http.MethodPost, "/api/workflows/demo/execute", "reached a shell before DGS-108"},
	{http.MethodGet, "/api/workflows/demo/execution", "execution stream"},
	{http.MethodGet, "/api/ws/workflow", "websocket"},
	{http.MethodPost, "/api/ada/validate", "was bare, zero callers"},
	{http.MethodGet, "/events", "SSE broadcaster, zero callers"},

	// CAS — durable writes and reads
	{http.MethodGet, "/api/cas/tags", "cas read"},
	{http.MethodPost, "/api/cas/tags", "cas durable write"},
	{http.MethodPost, "/api/cas/content", "cas durable write"},
	{http.MethodGet, "/api/cas/refs", "cas read"},
	{http.MethodPost, "/api/cas/refs", "cas durable write"},
	{http.MethodPost, "/api/cas/import", "cas bulk import"},
	{http.MethodPut, "/api/cas/batch", "cas bulk write"},
	{http.MethodGet, "/api/cas/status", "cas sync status"},
	{http.MethodGet, "/api/cas/delta", "cas sync delta"},
	{http.MethodGet, "/api/cas/export", "cas export"},
	{http.MethodPost, "/api/cas/gc", "destructive — also admin-gated"},
}

// TestDGS100_RoutesAreRegisteredInThisFixture guards every assertion below. If
// a path moves or a registration condition changes, the 401 assertions would
// start passing against a 404 — technically "not anonymous", but proving
// nothing. A 404 here means the table is stale, not that the route is safe.
func TestDGS100_RoutesAreRegisteredInThisFixture(t *testing.T) {
	s := newRoutedAPITestServer(t)

	for _, r := range gatedAPIRoutes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := do(t, s, r.method, r.path, "")
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"%s %s is not registered in this fixture, so its auth assertion would be vacuous (%s)",
				r.method, r.path, r.note)
		})
	}
}

// TestDGS100_UnauthenticatedAPIIsRejected is THE regression test. Every one of
// these requests, with no credential at all, was served before this change.
func TestDGS100_UnauthenticatedAPIIsRejected(t *testing.T) {
	s := newRoutedAPITestServer(t)

	for _, r := range gatedAPIRoutes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := do(t, s, r.method, r.path, "")
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s %s must reject an anonymous caller with 401 before reaching its handler (%s); got %d",
				r.method, r.path, r.note, w.Code)
		})
	}
}

// TestDGS100_GarbageTokenIsRejected closes the obvious hole in the test above:
// asserting only on "no header" would still pass if any non-empty string were
// accepted.
func TestDGS100_GarbageTokenIsRejected(t *testing.T) {
	s := newRoutedAPITestServer(t)

	for _, r := range gatedAPIRoutes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := do(t, s, r.method, r.path, "not-a-real-token")
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s %s must reject an unsigned garbage token; got %d", r.method, r.path, w.Code)
		})
	}
}

// TestDGS100_ValidTokenReachesTheHandler proves the gate is authentication and
// not a wall. Without this the tests above would pass just as happily against a
// route that returned 401 unconditionally.
//
// The assertion is deliberately "not 401" rather than a specific success code:
// these handlers get an empty JSON body and no real dependencies, so they may
// answer 400, 404, 500 or 501. What matters is that the request got PAST auth.
func TestDGS100_ValidTokenReachesTheHandler(t *testing.T) {
	s := newRoutedAPITestServer(t)

	token, _, err := middleware.IssueServiceToken("dojo-cli", 0)
	require.NoError(t, err, "a service token is the credential the CLI will carry")

	for _, r := range gatedAPIRoutes {
		if r.path == "/api/cas/gc" {
			continue // admin tier: a service token is refused there by design (403)
		}
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := do(t, s, r.method, r.path, token)
			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"%s %s rejected a valid service token — the gate is refusing everything, not authenticating (%s)",
				r.method, r.path, r.note)
		})
	}
}

// TestDGS100_ServiceTokenStillCannotGarbageCollect keeps the DGS-100 /gc tier
// intact now that the whole group carries AuthMiddleware. Both middlewares run
// on that route; the admin one must still have the final say.
func TestDGS100_ServiceTokenStillCannotGarbageCollect(t *testing.T) {
	s := newRoutedAPITestServer(t)

	token, _, err := middleware.IssueServiceToken("dojo-cli", 0)
	require.NoError(t, err)

	w := do(t, s, http.MethodPost, "/api/cas/gc", token)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"a service token satisfies AuthMiddleware but must still be refused on the admin tier with 403; got %d body=%s",
		w.Code, w.Body.String())
}

// TestDGS100_PublicRoutesStayPublic is the blast-radius guard. /health is hit
// unauthenticated by the deploy liveness probe in deploy/provision.sh, so
// gating it would fail every provision run. /auth/login must stay reachable or
// nobody can obtain a token in the first place.
func TestDGS100_PublicRoutesStayPublic(t *testing.T) {
	s := newRoutedAPITestServer(t)

	cases := []struct{ method, path, why string }{
		{http.MethodGet, "/health", "deploy/provision.sh liveness probe"},
		{http.MethodGet, "/metrics", "scraped without a gateway token"},
		{http.MethodPost, "/auth/login", "the only way to obtain a token"},
		{http.MethodPost, "/auth/register", "closed by config, not by auth (DGS-101)"},
		{http.MethodGet, "/.well-known/did.json", "federation discovery — DID auth, separate design"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := do(t, s, tc.method, tc.path, "")
			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"%s %s must stay reachable without a token (%s)", tc.method, tc.path, tc.why)
		})
	}
}
