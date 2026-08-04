package server

// DGS-100 regression tests.
//
// POST /api/cas/gc calls workflowCAS.GC(), which DELETES content, and the
// /api/cas group carried no middleware at all — so on a publicly reachable
// gateway this was an unauthenticated destructive operation. DGS-88 closed
// /v1/**; this prefix was deliberately scoped out of that hotfix and stayed
// open.
//
// Like the DGS-88 tests, this is a WIRING bug, not a handler bug: the group is
// declared without .Use(), so nothing rejects the request before the handler
// runs. These tests therefore MUST drive the real setupMiddleware() +
// setupRoutes(). A test that builds its own gin.New() and attaches handlers by
// hand (as handle_cas_sync_test.go legitimately does, since it targets handler
// behaviour) cannot observe this class of bug at all.
//
// Scope note: only /gc is gated. The rest of /api/cas has live callers — the
// `dojo` CLI uses /content and /tags — so gating the group wholesale would
// break them. That broader design is still open on DGS-100; the guard test
// below is what makes the narrowness deliberate rather than accidental.
//
// AdminAuthMiddleware's own accept/reject behaviour is covered in
// server/middleware/service_token_test.go. What is proven here is that the
// route is ATTACHED to it.

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// newRoutedCASTestServer is newRoutedTestServer plus a real in-memory CAS.
// The CAS group is registered under `if s.workflowCAS != nil`, so a nil store
// would leave every route below unregistered and every assertion would pass
// vacuously against a 404.
func newRoutedCASTestServer(t *testing.T) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

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

// TestDGS100_CASRoutesAreRegisteredInThisFixture guards the whole file. If the
// registration condition or the paths ever change, the 401 assertion below
// would start passing for the wrong reason (404 != 401 would fail, but a route
// silently moving to a new path would not be caught here otherwise).
func TestDGS100_CASRoutesAreRegisteredInThisFixture(t *testing.T) {
	s := newRoutedCASTestServer(t)

	w := do(t, s, http.MethodGet, "/api/cas/tags", "")
	require.NotEqual(t, http.StatusNotFound, w.Code,
		"/api/cas/* must be registered in this fixture or every assertion in this file is vacuous; got 404")
}

// TestDGS100_UnauthenticatedCASGarbageCollectIsRejected is THE regression test.
// This exact request, unauthenticated, would delete content in production.
func TestDGS100_UnauthenticatedCASGarbageCollectIsRejected(t *testing.T) {
	s := newRoutedCASTestServer(t)

	w := do(t, s, http.MethodPost, "/api/cas/gc", "")

	require.Equal(t, http.StatusUnauthorized, w.Code,
		"POST /api/cas/gc with no Authorization header must be rejected with 401 before reaching workflowCAS.GC(); got %d body=%s",
		w.Code, w.Body.String())
}

// TestDGS100_ServiceTokenCannotGarbageCollect pins the tier choice. /gc is on
// AdminAuthMiddleware, not AuthMiddleware, so the ordinary per-consumer service
// tokens (svc:dojo-cli, svc:dojo-mcp, svc:ceniza) must NOT be able to trigger a
// destructive GC — only an admin credential may.
//
// The expected code is 403, not 401, and the difference is the point:
// AdminAuthMiddleware answers "no credential at all" with 401 and "valid
// credential, insufficient role" with 403. A token that satisfies
// AuthMiddleware therefore reaches validateAdminRole() and is refused there.
// Asserting 403 specifically is what proves the request was rejected on the
// ADMIN tier rather than merely on authentication — a 401 here would mean the
// route had fallen back to ordinary auth.
func TestDGS100_ServiceTokenCannotGarbageCollect(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")

	s := newRoutedCASTestServer(t)

	// "test-token" is the non-production shim that satisfies AuthMiddleware.
	// It must still be refused here, proving /gc sits on the admin tier.
	w := do(t, s, http.MethodPost, "/api/cas/gc", "test-token")

	assert.Equal(t, http.StatusForbidden, w.Code,
		"a credential good enough for AuthMiddleware must be refused on the admin tier with 403 (not 401, which would mean the admin gate was replaced by ordinary auth); got %d body=%s",
		w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "Admin privileges required")
}

// TestDGS100_LiveCASCallersAreNotBrokenByTheGCFix is the blast-radius guard.
//
// The fix is deliberately narrow — one route, not the group — because the rest
// of /api/cas has real callers today. Gating the group would break the `dojo`
// CLI silently. This test makes that narrowness a checked invariant, so a
// future "tidy-up" that promotes the middleware to the group has to fail here
// and consciously deal with the CLI rather than discover it in production.
func TestDGS100_LiveCASCallersAreNotBrokenByTheGCFix(t *testing.T) {
	s := newRoutedCASTestServer(t)

	cases := []struct {
		method, path, caller string
	}{
		{http.MethodGet, "/api/cas/tags", "dojo CLI"},
		{http.MethodPost, "/api/cas/tags", "dojo CLI"},
		{http.MethodPost, "/api/cas/content", "dojo CLI"},
		{http.MethodGet, "/api/cas/refs", "gateway-internal"},
		{http.MethodGet, "/api/cas/status", "D1 sync"},
		{http.MethodGet, "/api/cas/delta", "D1 sync"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := do(t, s, tc.method, tc.path, "")
			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"%s %s still has a live caller (%s) and is NOT part of the DGS-100 /gc fix; a 401 here means the gate was widened to the group without addressing that caller",
				tc.method, tc.path, tc.caller)
		})
	}
}

// TestDGS100_SPARoutesStayAnonymous protects the reason DGS-100 was not fixed
// wholesale in the first place: chat-ui and workflow-builder call these two
// prefixes from the browser with no token, so they must stay anonymous until
// the SPA auth story is designed.
func TestDGS100_SPARoutesStayAnonymous(t *testing.T) {
	s := newRoutedCASTestServer(t)

	for _, path := range []string{"/api/workflows", "/api/skills"} {
		t.Run(path, func(t *testing.T) {
			w := do(t, s, http.MethodGet, path, "")
			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"%s is called from the browser by the SPAs with no token; a 401 here breaks the workflow builder", path)
		})
	}
}
