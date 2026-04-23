package server

// Regression test for the structural data race in handleGatewayGetAgentDetail
// and handleGatewayListAgentChannels.
//
// Root cause: both handlers took agentMu.RLock(), fetched the *AgentRuntime
// pointer, then immediately released the lock — dereferencing runtime.Channels
// (and other fields) without protection.  Concurrent writers in
// handleGatewayBindAgentChannels hold agentMu.Lock() and do
// runtime.Channels = append(...), racing on the slice header.
//
// Fix: copy all needed fields into local variables while the read lock is held;
// release the lock before any JSON serialisation work.

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

// TestHandleGatewayAgents_ConcurrentBindAndGet is the race-detector regression
// test for Bug 2.  It launches 50 goroutines: half POST to /bind (write lock +
// slice append) and half GET /detail and /channels (read path that previously
// released the lock before dereferencing fields).
//
// Run with: go test -race ./server/... — any data race causes an instant FAIL.
func TestHandleGatewayAgents_ConcurrentBindAndGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const agentID = "agent-race-test"
	const goroutines = 50 // 25 binders + 25 getters

	s := buildAgentRaceTestServer(t, agentID)

	router := gin.New()
	router.POST("/agents/:id/channels", s.handleGatewayBindAgentChannels)
	router.GET("/agents/:id", s.handleGatewayGetAgentDetail)
	router.GET("/agents/:id/channels", s.handleGatewayListAgentChannels)

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
				// ── Reader half A: GET agent detail ─────────────────────────
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
					errCh <- fmt.Errorf("goroutine %d (get-detail): unexpected status %d — body: %s",
						i, w.Code, w.Body.String())
				}
			} else {
				// ── Reader half B: GET agent channels ───────────────────────
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
