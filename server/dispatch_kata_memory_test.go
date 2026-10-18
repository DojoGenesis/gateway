package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/memory"
)

// DGS-150: kata-memory-write and kata-account-delete over the real /dispatch
// route, with real tokens and a real memory store.

func newKataMemoryDispatchServer(t *testing.T) (*Server, *memory.MemoryManager) {
	t.Helper()
	mm, err := memory.NewMemoryManager(filepath.Join(t.TempDir(), "kata-memory.db"))
	require.NoError(t, err)
	t.Cleanup(func() { mm.Close() })
	s := newDispatchTestServer(t, nil)
	s.memoryManager = mm
	return s, mm
}

func postKata(t *testing.T, s *Server, sub, idemKey, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/dispatch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, sub))
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func memoryWriteBody(user, rollID string) string {
	return `{"skill":"kata-memory-write","session":{"app":"kata"},"input":{"user_id":"` + user +
		`","roll":{"id":"` + rollID + `","user_id":"` + user + `","goal":"write the intro","state":"done"}}}`
}

func dispatchOutput(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Output     map[string]interface{} `json:"output"`
		Provenance map[string]interface{} `json:"provenance"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "1.0.0", resp.Provenance["skill_version"])
	return resp.Output
}

func ownerCount(t *testing.T, mm *memory.MemoryManager, owner string) int {
	t.Helper()
	n, err := mm.CountMemoriesScoped(t.Context(), memory.ForOwner(owner), memory.MemoryFilter{})
	require.NoError(t, err)
	return n
}

func TestKataMemoryWrite_StoresRollForTheSubject(t *testing.T) {
	s, mm := newKataMemoryDispatchServer(t)
	out := dispatchOutput(t, postKata(t, s, "u1", "key-1", memoryWriteBody("u1", "roll-1")))
	id, _ := out["memory_id"].(string)
	require.NotEmpty(t, id)

	got, err := mm.GetMemoryScoped(t.Context(), memory.ForOwner("u1"), id)
	require.NoError(t, err)
	assert.Equal(t, "u1", got.OwnerID)
	assert.Equal(t, kataRollMemoryType, got.Type)
	assert.Equal(t, "roll-1", got.Metadata["roll_id"])
	assert.JSONEq(t, `{"id":"roll-1","user_id":"u1","goal":"write the intro","state":"done"}`, got.Content)
}

func TestKataMemoryWrite_ReplayedKeyWritesOnce(t *testing.T) {
	s, mm := newKataMemoryDispatchServer(t)
	body := memoryWriteBody("u1", "roll-1")

	first := dispatchOutput(t, postKata(t, s, "u1", "queued-key", body))
	second := dispatchOutput(t, postKata(t, s, "u1", "queued-key", body))
	third := dispatchOutput(t, postKata(t, s, "u1", "queued-key", body))
	assert.Equal(t, first["memory_id"], second["memory_id"])
	assert.Equal(t, first["memory_id"], third["memory_id"])
	assert.Equal(t, 1, ownerCount(t, mm, "u1"), "a replayed key must not write twice")

	// A different key is a different write.
	other := dispatchOutput(t, postKata(t, s, "u1", "another-key", body))
	assert.NotEqual(t, first["memory_id"], other["memory_id"])
	assert.Equal(t, 2, ownerCount(t, mm, "u1"))
}

func TestKataMemoryWrite_SameKeyDifferentUsersDoNotCollide(t *testing.T) {
	s, mm := newKataMemoryDispatchServer(t)
	a := dispatchOutput(t, postKata(t, s, "u1", "shared-key", memoryWriteBody("u1", "r")))
	b := dispatchOutput(t, postKata(t, s, "u2", "shared-key", memoryWriteBody("u2", "r")))
	assert.NotEqual(t, a["memory_id"], b["memory_id"])
	assert.Equal(t, 1, ownerCount(t, mm, "u1"))
	assert.Equal(t, 1, ownerCount(t, mm, "u2"))
}

func TestKataMemoryWrite_Rejections(t *testing.T) {
	cases := []struct {
		name, body, code string
	}{
		{"input user_id is someone else", memoryWriteBody("victim", "r"), "subject_mismatch"},
		{"roll user_id is someone else", `{"skill":"kata-memory-write","input":{"user_id":"u1","roll":{"id":"r","user_id":"victim"}}}`, "subject_mismatch"},
		{"roll missing", `{"skill":"kata-memory-write","input":{"user_id":"u1"}}`, "invalid_input"},
		{"roll not an object", `{"skill":"kata-memory-write","input":{"user_id":"u1","roll":"x"}}`, "invalid_input"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, mm := newKataMemoryDispatchServer(t)
			w := postKata(t, s, "u1", "k", tc.body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Equal(t, "true", w.Header().Get("X-No-Retry"))
			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tc.code, resp["error"])
			n, err := mm.CountMemories(t.Context(), memory.MemoryFilter{})
			require.NoError(t, err)
			assert.Zero(t, n, "nothing may be written")
		})
	}
}

func TestKataAccountDelete_ErasesOnlyTheSubject(t *testing.T) {
	s, mm := newKataMemoryDispatchServer(t)
	dispatchOutput(t, postKata(t, s, "u1", "a", memoryWriteBody("u1", "r1")))
	dispatchOutput(t, postKata(t, s, "u1", "b", memoryWriteBody("u1", "r2")))
	dispatchOutput(t, postKata(t, s, "u2", "a", memoryWriteBody("u2", "r3")))

	out := dispatchOutput(t, postKata(t, s, "u1", "", `{"skill":"kata-account-delete","input":{"user_id":"u1"}}`))
	assert.EqualValues(t, 2, out["deleted_memories"])
	assert.Zero(t, ownerCount(t, mm, "u1"), "no memory may remain for the deleted user")
	assert.Equal(t, 1, ownerCount(t, mm, "u2"), "other users are untouched")

	again := dispatchOutput(t, postKata(t, s, "u1", "", `{"skill":"kata-account-delete","input":{"user_id":"u1"}}`))
	assert.EqualValues(t, 0, again["deleted_memories"], "repeating the delete is harmless")
}
