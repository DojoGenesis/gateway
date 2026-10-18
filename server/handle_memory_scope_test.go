package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/DojoGenesis/gateway/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DGS-163: /v1/memory is scoped to the JWT subject. These tests run the real
// AuthMiddleware with real signed tokens, so the subject comes from the token
// exactly as it does in production.

func subjectToken(t *testing.T, subject, role string) string {
	t.Helper()
	now := time.Now()
	claims := middleware.GatewayClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
		Role: role,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(middleware.GetJWTSecret())
	require.NoError(t, err)
	return signed
}

func newAuthedMemoryTestServer(t *testing.T) (*memory.MemoryManager, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mm, err := memory.NewMemoryManager(filepath.Join(t.TempDir(), "memory-scope.db"))
	require.NoError(t, err)
	t.Cleanup(func() { mm.Close() })

	router := gin.New()
	s := &Server{router: router, memoryManager: mm}
	v1 := router.Group("/v1")
	v1.Use(middleware.AuthMiddleware())
	v1.POST("/memory", s.handleStoreMemory)
	v1.GET("/memory", s.handleListMemories)
	v1.GET("/memory/:id", s.handleGetMemory)
	v1.PUT("/memory/:id", s.handleUpdateMemory)
	v1.DELETE("/memory/:id", s.handleDeleteMemory)
	v1.POST("/memory/search", s.handleSearchMemory)
	return mm, router
}

func doAuthed(t *testing.T, router *gin.Engine, token, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestMemoryRoutes_UserCannotReachAnotherUsersMemories(t *testing.T) {
	mm, router := newAuthedMemoryTestServer(t)
	alice := subjectToken(t, "user-alice", "user")
	bob := subjectToken(t, "user-bob", "user")

	w := doAuthed(t, router, alice, http.MethodPost, "/v1/memory", MemoryStoreRequest{
		Type:    "note",
		Content: "alice private zebra note",
		// A client-supplied user_id must not choose the owner.
		UserID: "user-bob",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var stored MemoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stored))

	got, err := mm.GetMemory(t.Context(), stored.ID)
	require.NoError(t, err)
	assert.Equal(t, "user-alice", got.OwnerID, "owner is the token subject, not the body's user_id")

	t.Run("list", func(t *testing.T) {
		w := doAuthed(t, router, bob, http.MethodGet, "/v1/memory", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp MemoryListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Empty(t, resp.Memories)
		assert.Equal(t, 0, resp.TotalCount, "total must not count other owners' rows")

		w = doAuthed(t, router, bob, http.MethodGet, "/v1/memory?query=zebra", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Empty(t, resp.Memories)
	})

	t.Run("get", func(t *testing.T) {
		w := doAuthed(t, router, bob, http.MethodGet, "/v1/memory/"+stored.ID, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("update", func(t *testing.T) {
		w := doAuthed(t, router, bob, http.MethodPut, "/v1/memory/"+stored.ID, MemoryUpdateRequest{Content: "bob was here"})
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("delete", func(t *testing.T) {
		w := doAuthed(t, router, bob, http.MethodDelete, "/v1/memory/"+stored.ID, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("search", func(t *testing.T) {
		w := doAuthed(t, router, bob, http.MethodPost, "/v1/memory/search", MemorySearchRequest{Query: "zebra"})
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp MemorySearchResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Empty(t, resp.Results)
		assert.Equal(t, 0, resp.TotalCount)
	})

	t.Run("owner still has it, unchanged", func(t *testing.T) {
		w := doAuthed(t, router, alice, http.MethodGet, "/v1/memory/"+stored.ID, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp MemoryResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "alice private zebra note", resp.Content)

		w = doAuthed(t, router, alice, http.MethodPost, "/v1/memory/search", MemorySearchRequest{Query: "zebra"})
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var search MemorySearchResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &search))
		assert.Len(t, search.Results, 1)
	})
}

func TestMemoryRoutes_AllOwnersIsExplicitAndAdminOnly(t *testing.T) {
	mm, router := newAuthedMemoryTestServer(t)
	alice := subjectToken(t, "user-alice", "user")
	admin := subjectToken(t, "operator", "admin")

	require.NoError(t, mm.Store(t.Context(), memory.Memory{
		ID: "alice-1", Type: "note", Content: "alice row", OwnerID: "user-alice",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, mm.Store(t.Context(), memory.Memory{
		ID: "legacy-1", Type: "note", Content: "pre-DGS-163 row", OwnerID: memory.LegacyOwnerID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	t.Run("a user asking for all_owners is refused", func(t *testing.T) {
		w := doAuthed(t, router, alice, http.MethodGet, "/v1/memory?all_owners=true", nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
		w = doAuthed(t, router, alice, http.MethodPost, "/v1/memory/search?all_owners=true", MemorySearchRequest{Query: "row"})
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("an admin without all_owners sees only their own memories", func(t *testing.T) {
		w := doAuthed(t, router, admin, http.MethodGet, "/v1/memory", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp MemoryListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Empty(t, resp.Memories)

		w = doAuthed(t, router, admin, http.MethodGet, "/v1/memory/alice-1", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("an admin with all_owners sees every owner, legacy rows included", func(t *testing.T) {
		w := doAuthed(t, router, admin, http.MethodGet, "/v1/memory?all_owners=true", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp MemoryListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp.Memories, 2)
		assert.Equal(t, 2, resp.TotalCount)

		w = doAuthed(t, router, admin, http.MethodGet, "/v1/memory/alice-1?all_owners=true", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("a user never sees legacy rows", func(t *testing.T) {
		w := doAuthed(t, router, alice, http.MethodGet, "/v1/memory/legacy-1", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
