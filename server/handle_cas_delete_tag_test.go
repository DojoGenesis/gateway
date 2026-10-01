package server

// DGS-111: DELETE /api/cas/tags/... must address slash-containing tag names
// such as "workflow/<name>" (how workflow/api createWorkflow tags workflows).

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/runtime/cas"
	"github.com/DojoGenesis/gateway/server/middleware"
)

func tagFixture(t *testing.T, s *Server, tags ...[2]string) {
	t.Helper()
	ctx := context.Background()
	ref, err := s.workflowCAS.Put(ctx, []byte("payload"), cas.ContentMeta{Type: "workflow"})
	require.NoError(t, err)
	for _, tg := range tags {
		require.NoError(t, s.workflowCAS.Tag(ctx, tg[0], tg[1], ref))
	}
}

func resolves(s *Server, name, version string) bool {
	_, err := s.workflowCAS.Resolve(context.Background(), name, version)
	return err == nil
}

func TestDGS111_DeleteSlashContainingTag(t *testing.T) {
	s := newRoutedCASTestServer(t)
	token, _, err := middleware.IssueServiceToken("dojo-cli", 0)
	require.NoError(t, err)
	tagFixture(t, s, [2]string{"workflow/x", "1.0.0"}, [2]string{"workflow/x", "latest"})

	w := do(t, s, http.MethodDelete, "/api/cas/tags/workflow/x/1.0.0", token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, resolves(s, "workflow/x", "1.0.0"))
	assert.True(t, resolves(s, "workflow/x", "latest"))

	w = do(t, s, http.MethodDelete, "/api/cas/tags/workflow/x/latest", token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, resolves(s, "workflow/x", "latest"))
}

func TestDGS111_DeletePlainTagStillWorks(t *testing.T) {
	s := newRoutedCASTestServer(t)
	token, _, err := middleware.IssueServiceToken("dojo-cli", 0)
	require.NoError(t, err)
	tagFixture(t, s, [2]string{"plain", "1.0.0"})

	w := do(t, s, http.MethodDelete, "/api/cas/tags/plain/1.0.0", token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.False(t, resolves(s, "plain", "1.0.0"))
}

func TestDGS111_DeleteMissingVersionIs400(t *testing.T) {
	s := newRoutedCASTestServer(t)
	token, _, err := middleware.IssueServiceToken("dojo-cli", 0)
	require.NoError(t, err)
	tagFixture(t, s, [2]string{"workflow/x", "1.0.0"})

	for _, p := range []string{"/api/cas/tags/workflow", "/api/cas/tags/workflow/x/", "/api/cas/tags/"} {
		w := do(t, s, http.MethodDelete, p, token)
		assert.Equal(t, http.StatusBadRequest, w.Code, "%s -> %s", p, w.Body.String())
	}
	assert.True(t, resolves(s, "workflow/x", "1.0.0"), "nothing may be deleted on a 400")
}
