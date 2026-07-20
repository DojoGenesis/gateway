package server

// Race-detector regression test for the agent handlers sharing agentMu:
// concurrent channel binds (write lock + slice append) against the live read
// paths handleGatewayGetAgent and handleGatewayListAgentChannels.
//
// Original root cause: readers took agentMu.RLock(), fetched the *AgentRuntime
// pointer, then released the lock before dereferencing runtime.Channels —
// racing on the slice header against handleGatewayBindAgentChannels.  Fix:
// copy needed fields while the read lock is held (see
// handleGatewayListAgentChannels).  The other affected reader,
// handleGatewayGetAgentDetail, was never routed and has been deleted; the
// GET /agents/:id leg now exercises the live handleGatewayGetAgent, and the
// POST /chat leg exercises handleGatewayAgentChat — both copy Config (and
// disposition fields) into locals under the read lock and must never
// dereference the runtime after releasing it.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/DojoGenesis/gateway/disposition"
	"github.com/DojoGenesis/gateway/pkg/gateway"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildAgentRaceTestServer returns a minimal *Server pre-loaded with one
// AgentRuntime that has a Config, Disposition, and an initial Channels slice.
func buildAgentRaceTestServer(t *testing.T, agentID string) *Server {
	t.Helper()
	s := &Server{
		cfg: &ServerConfig{
			Port:        "7340",
			Environment: "test",
		},
		agents: map[string]*AgentRuntime{
			agentID: {
				Config: &gateway.AgentConfig{
					Pacing:     "measured",
					Depth:      "thorough",
					Tone:       "professional",
					Initiative: "responsive",
				},
				Disposition: &disposition.DispositionConfig{
					Pacing:     "measured",
					Depth:      "thorough",
					Tone:       "professional",
					Initiative: "responsive",
				},
				Channels: []string{"channel-seed"},
			},
		},
	}
	return s
}

// TestHandleGatewayAgents_ConcurrentBindAndGet launches 50 goroutines: half
// POST to /bind (write lock + slice append), the other half spread across the
// three reader legs — GET /agents/:id, GET /channels, and POST /chat — all
// paths that must copy shared state under the read lock.
//
// Run with: go test -race ./server/... — any data race causes an instant FAIL.
func TestHandleGatewayAgents_ConcurrentBindAndGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const agentID = "agent-race-test"
	const goroutines = 50 // 25 binders + 25 readers

	s := buildAgentRaceTestServer(t, agentID)

	router := gin.New()
	router.POST("/agents/:id/channels", s.handleGatewayBindAgentChannels)
	router.GET("/agents/:id", s.handleGatewayGetAgent)
	router.GET("/agents/:id/channels", s.handleGatewayListAgentChannels)
	router.POST("/agents/:id/chat", s.handleGatewayAgentChat)

	var wg sync.WaitGroup
	// Collect the first error from any goroutine to surface it.
	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		i := i // capture loop variable
		go func() {
			defer wg.Done()

			if i%2 == 0 {
				// ── Writer half: bind a new channel ─────────────────────────
				body := map[string]interface{}{
					"channels": []string{fmt.Sprintf("ch-%d", i)},
				}
				b, _ := json.Marshal(body)
				req, err := http.NewRequest(
					http.MethodPost,
					"/agents/"+agentID+"/channels",
					bytes.NewReader(b),
				)
				if err != nil {
					errCh <- fmt.Errorf("goroutine %d: request creation failed: %w", i, err)
					return
				}
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					errCh <- fmt.Errorf("goroutine %d (bind): unexpected status %d — body: %s",
						i, w.Code, w.Body.String())
				}
			} else if i%4 == 1 {
				// ── Reader A: GET agent ─────────────────────────────────────
				req, err := http.NewRequest(
					http.MethodGet,
					"/agents/"+agentID,
					nil,
				)
				if err != nil {
					errCh <- fmt.Errorf("goroutine %d: request creation failed: %w", i, err)
					return
				}

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					errCh <- fmt.Errorf("goroutine %d (get-agent): unexpected status %d — body: %s",
						i, w.Code, w.Body.String())
				}
			} else if i%8 == 3 {
				// ── Reader C: POST chat ─────────────────────────────────────
				// Exercises the Config copy-under-lock in
				// handleGatewayAgentChat.  The copy happens before the plugin
				// manager check, so the expected 503 (no plugin manager on the
				// test server) still covers the locked read path.
				body := map[string]interface{}{"message": "race probe"}
				b, _ := json.Marshal(body)
				req, err := http.NewRequest(
					http.MethodPost,
					"/agents/"+agentID+"/chat",
					bytes.NewReader(b),
				)
				if err != nil {
					errCh <- fmt.Errorf("goroutine %d: request creation failed: %w", i, err)
					return
				}
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusServiceUnavailable {
					errCh <- fmt.Errorf("goroutine %d (chat): unexpected status %d — body: %s",
						i, w.Code, w.Body.String())
				}
			} else {
				// ── Reader B: GET agent channels ────────────────────────────
				req, err := http.NewRequest(
					http.MethodGet,
					"/agents/"+agentID+"/channels",
					nil,
				)
				if err != nil {
					errCh <- fmt.Errorf("goroutine %d: request creation failed: %w", i, err)
					return
				}

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					errCh <- fmt.Errorf("goroutine %d (list-channels): unexpected status %d — body: %s",
						i, w.Code, w.Body.String())
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	// Drain errors — fail on the first one found.
	for err := range errCh {
		require.NoError(t, err)
	}

	// Sanity-check: the agent still exists and channels list is non-empty.
	req, err := http.NewRequest(http.MethodGet, "/agents/"+agentID+"/channels", nil)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	count, ok := resp["count"].(float64)
	require.True(t, ok, "response should have numeric 'count' field")
	assert.GreaterOrEqual(t, int(count), 1, "at least the seed channel should remain")
}
