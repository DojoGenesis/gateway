package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DGS-163, second handler path: MemoryHandler's memory CRUD is not routed by
// setupRoutes today, but it must be just as owner-scoped if it ever is.
func TestMemoryHandler_UserCannotReachAnotherUsersMemories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mm, err := memory.NewMemoryManager(filepath.Join(t.TempDir(), "scope.db"))
	require.NoError(t, err)
	t.Cleanup(func() { mm.Close() })
	h := NewMemoryHandler(mm, nil, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		if sub := c.GetHeader("X-Test-Subject"); sub != "" {
			c.Set("user_id", sub)
		}
		c.Next()
	})
	router.POST("/m", h.StoreMemory)
	router.GET("/m/list", h.ListMemories)
	router.GET("/m/:id", h.RetrieveMemory)
	router.PUT("/m/:id", h.UpdateMemory)
	router.DELETE("/m/:id", h.DeleteMemory)
	router.POST("/m/search", h.SearchMemory)

	do := func(subject, method, path string, body interface{}) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			require.NoError(t, json.NewEncoder(&buf).Encode(body))
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("X-Test-Subject", subject)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	w := do("alice", http.MethodPost, "/m", StoreMemoryRequest{Type: "note", Content: "alice zebra"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created struct {
		Memory memory.Memory `json:"memory"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	id := created.Memory.ID

	assert.Equal(t, http.StatusUnauthorized, do("", http.MethodGet, "/m/"+id, nil).Code, "no subject, no access")
	assert.Equal(t, http.StatusNotFound, do("bob", http.MethodGet, "/m/"+id, nil).Code)
	assert.Equal(t, http.StatusNotFound, do("bob", http.MethodPut, "/m/"+id, UpdateMemoryRequest{Content: "bob"}).Code)
	assert.Equal(t, http.StatusNotFound, do("bob", http.MethodDelete, "/m/"+id, nil).Code)
	assert.Equal(t, http.StatusForbidden, do("bob", http.MethodGet, "/m/list?all_owners=true", nil).Code)

	var list struct {
		Count int `json:"count"`
	}
	require.NoError(t, json.Unmarshal(do("bob", http.MethodGet, "/m/list", nil).Body.Bytes(), &list))
	assert.Zero(t, list.Count)

	var search struct {
		Count int `json:"count"`
	}
	require.NoError(t, json.Unmarshal(do("bob", http.MethodPost, "/m/search", SearchMemoryRequest{Query: "zebra"}).Body.Bytes(), &search))
	assert.Zero(t, search.Count)

	w = do("alice", http.MethodGet, "/m/"+id, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "alice zebra")
}
