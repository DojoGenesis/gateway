package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DojoGenesis/gateway/observation/audit"
	"github.com/DojoGenesis/gateway/pkg/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAgentInitializer struct{}

func (stubAgentInitializer) Initialize(ctx context.Context, workspaceRoot, activeMode string) (*gateway.AgentConfig, error) {
	return &gateway.AgentConfig{}, nil
}

func newAuditTestServer(t *testing.T) *Server {
	t.Helper()
	log, err := audit.NewSQLiteAuditLog(":memory:")
	require.NoError(t, err)
	rec, err := audit.NewRecorder(context.Background(), log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rec.Close() })

	s, _ := newTestServer()
	s.auditRecorder = rec
	s.agentInitializer = stubAgentInitializer{}
	return s
}

func TestAgentSpawnIsAudited(t *testing.T) {
	s := newAuditTestServer(t)
	_, router := newTestServer()
	router.POST("/v1/gateway/agents", s.handleGatewayCreateAgent)
	router.GET("/admin/audit/entries", s.handleAdminAuditEntries)
	router.GET("/admin/audit/verify", s.handleAdminAuditVerify)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/gateway/agents", strings.NewReader(`{"workspace_root":"."}`)))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created struct {
		AgentID string `json:"agent_id"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/audit/entries?action=agent_spawn", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var list struct {
		Entries []audit.AuditEntry `json:"entries"`
		HeadSeq int64              `json:"head_seq"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list.Entries, 1)
	assert.Equal(t, created.AgentID, list.Entries[0].AgentID)
	assert.Equal(t, int64(1), list.HeadSeq)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/audit/verify", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var rep audit.VerifyReport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rep))
	assert.True(t, rep.OK)
	assert.Equal(t, 1, rep.Chained)
}

func TestAuditRoutesWithoutRecorder(t *testing.T) {
	s, router := newTestServer()
	router.GET("/admin/audit/entries", s.handleAdminAuditEntries)
	router.GET("/admin/audit/verify", s.handleAdminAuditVerify)
	for _, path := range []string{"/admin/audit/entries", "/admin/audit/verify"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		assertErrorShape(t, w, http.StatusServiceUnavailable, "service_unavailable")
	}
}

func TestAuditEntriesRejectsBadLimit(t *testing.T) {
	s := newAuditTestServer(t)
	_, router := newTestServer()
	router.GET("/admin/audit/entries", s.handleAdminAuditEntries)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/audit/entries?limit=5000", nil))
	assertErrorShape(t, w, http.StatusBadRequest, "invalid_request")
}
