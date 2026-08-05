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
// Scope note, updated: the whole /api/cas group now carries AuthMiddleware,
// and /gc keeps AdminAuthMiddleware on top of it. This file originally gated
// only /gc and carried a guard test asserting the rest of the group stayed
// anonymous, because the `dojo` CLI calls /content and /tags with a token only
// when one is configured — which is not the default.
//
// That guard fired when the group was gated, which is exactly what it was for.
// The break is now a deliberate, recorded decision rather than an accident:
// leaving durable unauthenticated writes open on a public gateway is worse than
// requiring the CLI to carry the credential that already exists for it
// (svc:dojo-cli). The guard has been rewritten below to assert the new
// intent — and to keep the CLI fact visible, because it is the migration step.
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
	"github.com/DojoGenesis/gateway/server/middleware"
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
	// "test-token" is a development shim token, which since DGS-112 requires an
	// explicit opt-in rather than merely a non-production ENVIRONMENT.
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv(middleware.EnvDevTokens, "true")

	s := newRoutedCASTestServer(t)

	// "test-token" is the non-production shim that satisfies AuthMiddleware.
	// It must still be refused here, proving /gc sits on the admin tier.
	w := do(t, s, http.MethodPost, "/api/cas/gc", "test-token")

	assert.Equal(t, http.StatusForbidden, w.Code,
		"a credential good enough for AuthMiddleware must be refused on the admin tier with 403 (not 401, which would mean the admin gate was replaced by ordinary auth); got %d body=%s",
		w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "Admin privileges required")
}

// TestDGS100_CASCallersMustNowCarryAToken replaces the blast-radius guard that
// used to assert the opposite.
//
// The original test existed so that widening the gate to the whole group could
// not happen by accident — it demanded that whoever did it deal with the `dojo`
// CLI consciously instead of discovering the breakage in production. When the
// group was gated, it fired. This is that conscious dealing, recorded:
//
//   - The CLI (cli/internal/commands/cmd_skill.go, via internal/client) attaches
//     `Authorization: Bearer <token>` only when c.token is non-empty, and the
//     default from cli/internal/config is empty. So a CLI that has never been
//     configured now gets 401 on these routes.
//   - That is accepted. These are durable writes on an internet-reachable
//     gateway; leaving them anonymous is worse than requiring the credential
//     that already exists for this consumer (svc:dojo-cli).
//   - Migration: set `gateway.token` in ~/.dojo/settings.json or export
//     DOJO_GATEWAY_TOKEN before this reaches a host the CLI talks to.
//
// The list of paths is kept verbatim from the old guard so the diff shows
// exactly which callers changed contract.
func TestDGS100_CASCallersMustNowCarryAToken(t *testing.T) {
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

	token, _, err := middleware.IssueServiceToken("dojo-cli", 0)
	require.NoError(t, err)

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			anon := do(t, s, tc.method, tc.path, "")
			assert.Equal(t, http.StatusUnauthorized, anon.Code,
				"%s %s must refuse an anonymous caller (%s was the reason this stayed open; it now needs its token)",
				tc.method, tc.path, tc.caller)

			// ...and the configured caller must still get through, or this is a
			// wall rather than a gate.
			withToken := do(t, s, tc.method, tc.path, token)
			assert.NotEqual(t, http.StatusUnauthorized, withToken.Code,
				"%s %s rejected a valid svc:dojo-cli token — the documented migration would not actually work",
				tc.method, tc.path)
		})
	}
}

// TestDGS100_SPARoutesRequireAToken replaces a guard whose premise turned out
// to be false.
//
// The old test asserted these routes must stay anonymous "because the SPAs call
// them from the browser with no token" — the stated reason DGS-100 went unfixed
// from 2026-07-24. That consumer does not exist in any deployed gateway:
//
//   - server/workflowui/dist and server/chatui/dist contain only .gitkeep; the
//     SPA build is a separate `make build-spa` step,
//   - goreleaser's only before-hook is `go mod download` — no npm — so no
//     release has ever embedded them,
//   - and gateway.trespies.dev answers 503 on /workflow and /chat, confirming
//     the deployed binary has no SPA in it.
//
// So the SPA's auth story is a prerequisite for SHIPPING the workflow builder,
// not a reason to serve /api/workflows to the whole internet. When it does
// ship it needs a Secure/HttpOnly/SameSite cookie, because
// workflow-builder/src/lib/api.ts subscribes to execution over EventSource,
// which cannot send an Authorization header at all. That is recorded in
// docs/api-route-disposition.md.
func TestDGS100_SPARoutesRequireAToken(t *testing.T) {
	s := newRoutedCASTestServer(t)

	for _, path := range []string{"/api/workflows", "/api/skills"} {
		t.Run(path, func(t *testing.T) {
			w := do(t, s, http.MethodGet, path, "")
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s must require a token; the browser consumer that justified leaving it open has never shipped", path)
		})
	}
}
