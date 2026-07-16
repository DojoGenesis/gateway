package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/DojoGenesis/gateway/memory"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMemoryTestServer creates a minimal Server wired to a real, temp-file-backed
// memory.MemoryManager and the actual /v1/memory route handlers, mirroring
// newAuthTestServer's pattern in handle_auth_test.go. Exercising the real handlers
// (not just the memory package directly) is what proves the fix end-to-end over
// HTTP, matching how the defects were originally reproduced against the live
// Gateway.
func newMemoryTestServer(t *testing.T) (*Server, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "memory-test.db")
	mm, err := memory.NewMemoryManager(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { mm.Close() })

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request-id")
		c.Next()
	})

	s := &Server{
		router:        router,
		memoryManager: mm,
	}

	v1 := router.Group("/v1")
	v1.POST("/memory", s.handleStoreMemory)
	v1.GET("/memory", s.handleListMemories)
	v1.GET("/memory/:id", s.handleGetMemory)
	v1.PUT("/memory/:id", s.handleUpdateMemory)
	v1.DELETE("/memory/:id", s.handleDeleteMemory)
	v1.POST("/memory/search", s.handleSearchMemory)

	return s, router
}

func doJSON(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req, err := http.NewRequest(method, path, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func storeTestMemory(t *testing.T, router *gin.Engine, memType, content string) {
	t.Helper()
	w := doJSON(t, router, http.MethodPost, "/v1/memory", MemoryStoreRequest{
		Type:    memType,
		Content: content,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
}

// TestHandleListMemories_TotalCountIsHonest reproduces the live defect: GET
// /v1/memory reported TotalCount == len(page) instead of the store's true size,
// so a caller reading the default (limit=20) page had no way to tell "the store
// has exactly 20 entries" apart from "the store has more and I got truncated".
func TestHandleListMemories_TotalCountIsHonest(t *testing.T) {
	_, router := newMemoryTestServer(t)

	const storeSize = 25 // > the handler's default limit of 20
	for i := 0; i < storeSize; i++ {
		storeTestMemory(t, router, "general", fmt.Sprintf("memory entry number %d", i))
	}

	t.Run("default page truncates but total_count reports the true store size", func(t *testing.T) {
		w := doJSON(t, router, http.MethodGet, "/v1/memory", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var resp MemoryListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		assert.Len(t, resp.Memories, 20, "default page size should still be 20")
		assert.Equal(t, 20, resp.Limit)
		assert.Equal(t, 0, resp.Offset)
		assert.Equal(t, storeSize, resp.TotalCount,
			"total_count must report the true store size, not len(memories) (the reported defect)")
		assert.NotEqual(t, len(resp.Memories), resp.TotalCount,
			"this is exactly the truncation the defect made undetectable")
	})

	t.Run("limit=100 returns everything and total_count still matches", func(t *testing.T) {
		w := doJSON(t, router, http.MethodGet, "/v1/memory?limit=100", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var resp MemoryListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		assert.Len(t, resp.Memories, storeSize)
		assert.Equal(t, storeSize, resp.TotalCount)
	})

	t.Run("wire shape is unchanged: memories/total_count/limit/offset keys, no renames", func(t *testing.T) {
		w := doJSON(t, router, http.MethodGet, "/v1/memory?limit=1", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))

		assert.Contains(t, raw, "memories")
		assert.Contains(t, raw, "total_count")
		assert.Contains(t, raw, "limit")
		assert.Contains(t, raw, "offset")
		assert.Len(t, raw, 4, "no fields should have been added or removed from the envelope")
	})
}

// TestHandleListMemories_OffsetPaginates proves offset -- accepted and echoed by
// the handler before this fix, but silently never applied to the query -- now
// actually walks through the store instead of returning the same first page
// again. Left broken, this would have been a second, worse "silent truncation"
// trap sitting directly downstream of an honest total_count: a caller who
// correctly detects truncation and retries with ?offset=20 would silently get
// page 1 again instead of an error or the missing rows.
func TestHandleListMemories_OffsetPaginates(t *testing.T) {
	_, router := newMemoryTestServer(t)

	const storeSize = 6
	for i := 0; i < storeSize; i++ {
		storeTestMemory(t, router, "general", fmt.Sprintf("offset entry %d", i))
	}

	seen := map[string]bool{}
	for offset := 0; offset < storeSize; offset += 2 {
		w := doJSON(t, router, http.MethodGet, fmt.Sprintf("/v1/memory?limit=2&offset=%d", offset), nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var resp MemoryListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		require.Len(t, resp.Memories, 2)
		assert.Equal(t, storeSize, resp.TotalCount)
		for _, m := range resp.Memories {
			assert.False(t, seen[m.ID], "offset=%d returned an entry already seen at a prior offset", offset)
			seen[m.ID] = true
		}
	}
	assert.Len(t, seen, storeSize, "paging through with offset should visit every entry exactly once")
}

// TestHandleSearchMemory_TokenBased is the HTTP-level counterpart to
// TestSearchMemories_TokenBased in the memory package: it drives the actual
// POST /v1/memory/search route (not just the manager) with the same
// before/after queries used to reproduce the live defect.
func TestHandleSearchMemory_TokenBased(t *testing.T) {
	_, router := newMemoryTestServer(t)

	storeTestMemory(t, router, "general",
		"DECISION 2026-07-01 (IP strategy - LOCKED default): ALL TresPies software IP is kept "+
			"as a TRADE SECRET by default, until it is either deliberately open-sourced or patented.")
	storeTestMemory(t, router, "general", "Reminder: renew the domain registration before it expires.")

	search := func(t *testing.T, query string) MemorySearchResponse {
		t.Helper()
		w := doJSON(t, router, http.MethodPost, "/v1/memory/search", MemorySearchRequest{Query: query})
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp MemorySearchResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		return resp
	}

	t.Run("exact substring still matches and ranks first (regression guard)", func(t *testing.T) {
		resp := search(t, "trade secret")
		require.Len(t, resp.Results, 1)
		assert.Contains(t, resp.Results[0].Content, "TRADE SECRET")
		assert.Equal(t, 1, resp.TotalCount)
	})

	t.Run("case-insensitive", func(t *testing.T) {
		resp := search(t, "TRADE")
		require.Len(t, resp.Results, 1)
		assert.Contains(t, resp.Results[0].Content, "TRADE SECRET")
	})

	t.Run("natural language query now matches (was 0 hits live before the fix)", func(t *testing.T) {
		resp := search(t, "IP strategy trade secret locked default")
		require.NotEmpty(t, resp.Results, "this exact query returned 0 hits against the live Gateway before the fix")
		assert.Contains(t, resp.Results[0].Content, "TRADE SECRET")
		assert.Equal(t, 1, resp.TotalCount)
	})

	t.Run("relevance_score is populated, not hardcoded to zero", func(t *testing.T) {
		resp := search(t, "IP strategy trade secret locked default")
		require.NotEmpty(t, resp.Results)
		assert.Greater(t, resp.Results[0].RelevanceScore, 0.0)
	})

	t.Run("still no semantic matching (documents the known, deliberate gap)", func(t *testing.T) {
		resp := search(t, "intellectual property")
		assert.Empty(t, resp.Results)
		assert.Equal(t, 0, resp.TotalCount)
	})

	t.Run("wire shape is unchanged: results/total_count keys, result field names", func(t *testing.T) {
		w := doJSON(t, router, http.MethodPost, "/v1/memory/search", MemorySearchRequest{Query: "trade secret"})
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var raw map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
		assert.Contains(t, raw, "results")
		assert.Contains(t, raw, "total_count")
		assert.Len(t, raw, 2, "no fields should have been added or removed from the envelope")

		results, ok := raw["results"].([]interface{})
		require.True(t, ok)
		require.NotEmpty(t, results)
		first, ok := results[0].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, first, "id")
		assert.Contains(t, first, "type")
		assert.Contains(t, first, "content")
		assert.Contains(t, first, "relevance_score")
	})
}

// TestHandleSearchMemory_TotalCountIsHonest guards the same "total equals page
// size" bug class on the search endpoint's response, which declares TotalCount
// alongside RelevanceScore -- both were unpopulated/mispopulated dead fields.
func TestHandleSearchMemory_TotalCountIsHonest(t *testing.T) {
	_, router := newMemoryTestServer(t)

	for i := 0; i < 5; i++ {
		storeTestMemory(t, router, "general", "golang programming notes")
	}

	w := doJSON(t, router, http.MethodPost, "/v1/memory/search", MemorySearchRequest{Query: "golang", Limit: 2})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp MemorySearchResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Len(t, resp.Results, 2, "page should still respect the requested limit")
	assert.Equal(t, 5, resp.TotalCount, "total_count must reflect all matches, not just the returned page")
}
