package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/server/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter() *gin.Engine {
	r := gin.New()
	r.GET("/events", HandleSSE)
	return r
}

func clearClients() {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for id, client := range clients {
		close(client.Channel)
		delete(clients, id)
	}
}

func TestHandleSSE_MissingClientID(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "client_id")
}

func TestHandleSSE_ValidConnection(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	clientID := "test-client-123"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan bool)
	go func() {
		router.ServeHTTP(w, req)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, GetConnectedClients())
	assert.Contains(t, GetClientIDs(), clientID)

	cancel()
	<-done

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, GetConnectedClients())
	assert.NotContains(t, GetClientIDs(), clientID)
}

func TestHandleSSE_Headers(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	clientID := "test-client-headers"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)

	ctx, cancel := context.WithTimeout(req.Context(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	router.ServeHTTP(w, req)

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
	assert.Equal(t, "no", w.Header().Get("X-Accel-Buffering"))
}

func TestSendToClient_Success(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	clientID := "test-send-client"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan bool)
	go func() {
		router.ServeHTTP(w, req)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	err := SendToClient(clientID, "test-event", `{"message":"hello"}`)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := w.Body.String()
	assert.Contains(t, body, "event: test-event")
	assert.Contains(t, body, `data: {"message":"hello"}`)
}

func TestSendToClient_ClientNotFound(t *testing.T) {
	clearClients()

	err := SendToClient("nonexistent-client", "test", "data")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSendToClient_ChannelFull(t *testing.T) {
	clearClients()

	clientID := "test-buffer-full"
	client := &models.Client{
		ID:      clientID,
		Channel: make(chan string, channelBufferSize),
		Created: time.Now(),
	}

	clientsMutex.Lock()
	clients[clientID] = client
	clientsMutex.Unlock()

	defer func() {
		clientsMutex.Lock()
		delete(clients, clientID)
		close(client.Channel)
		clientsMutex.Unlock()
	}()

	for i := 0; i < channelBufferSize; i++ {
		err := SendToClient(clientID, "test", fmt.Sprintf("message-%d", i))
		assert.NoError(t, err)
	}

	err := SendToClient(clientID, "test", "overflow-message")
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "buffer full")
	}
}

func TestHandleSSE_ConcurrentClients(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	numClients := 100
	var wg sync.WaitGroup
	wg.Add(numClients)

	responses := make([]*httptest.ResponseRecorder, numClients)
	cancels := make([]context.CancelFunc, numClients)

	for i := 0; i < numClients; i++ {
		i := i
		go func() {
			defer wg.Done()

			clientID := fmt.Sprintf("concurrent-client-%d", i)
			w := httptest.NewRecorder()
			responses[i] = w

			req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
			ctx, cancel := context.WithCancel(req.Context())
			cancels[i] = cancel
			req = req.WithContext(ctx)

			router.ServeHTTP(w, req)
		}()
	}

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, numClients, GetConnectedClients())

	for i := 0; i < numClients; i++ {
		clientID := fmt.Sprintf("concurrent-client-%d", i)
		err := SendToClient(clientID, "test", fmt.Sprintf("message-%d", i))
		assert.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	for i := 0; i < numClients; i++ {
		cancels[i]()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, GetConnectedClients())

	for i := 0; i < numClients; i++ {
		body := responses[i].Body.String()
		assert.Contains(t, body, fmt.Sprintf("message-%d", i))
	}
}

func TestHandleSSE_ReconnectSameClientID(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	clientID := "reconnect-client"

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
	ctx1, cancel1 := context.WithCancel(req1.Context())
	req1 = req1.WithContext(ctx1)

	done1 := make(chan bool)
	go func() {
		router.ServeHTTP(w1, req1)
		done1 <- true
	}()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, GetConnectedClients())

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
	ctx2, cancel2 := context.WithCancel(req2.Context())
	defer cancel2()
	req2 = req2.WithContext(ctx2)

	done2 := make(chan bool)
	go func() {
		router.ServeHTTP(w2, req2)
		done2 <- true
	}()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, GetConnectedClients())

	cancel1()
	<-done1

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, GetConnectedClients())

	err := SendToClient(clientID, "test", "message-to-new-connection")
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	cancel2()
	<-done2

	body2 := w2.Body.String()
	assert.Contains(t, body2, "message-to-new-connection")
}

func TestGetConnectedClients(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	assert.Equal(t, 0, GetConnectedClients())

	clientID := "test-count-client"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)

	go router.ServeHTTP(w, req)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, GetConnectedClients())
}

func TestGetClientIDs(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	ids := GetClientIDs()
	assert.Empty(t, ids)

	clientIDs := []string{"client-a", "client-b", "client-c"}
	cancels := make([]context.CancelFunc, len(clientIDs))

	for i, clientID := range clientIDs {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancels[i] = cancel
		req = req.WithContext(ctx)

		go router.ServeHTTP(w, req)
	}

	time.Sleep(100 * time.Millisecond)

	ids = GetClientIDs()
	assert.Len(t, ids, len(clientIDs))
	for _, clientID := range clientIDs {
		assert.Contains(t, ids, clientID)
	}

	for _, cancel := range cancels {
		cancel()
	}
	time.Sleep(100 * time.Millisecond)
}

func TestDisconnectClient(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	clientID := "test-disconnect-client"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
	ctx := req.Context()
	req = req.WithContext(ctx)

	done := make(chan bool)
	go func() {
		router.ServeHTTP(w, req)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, GetConnectedClients())

	disconnected := DisconnectClient(clientID)
	assert.True(t, disconnected)

	<-done
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, GetConnectedClients())
}

func TestDisconnectClient_NotFound(t *testing.T) {
	clearClients()

	disconnected := DisconnectClient("nonexistent-client")
	assert.False(t, disconnected)
}

func TestFormatSSEMessage(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		data     string
		expected string
	}{
		{
			name:     "With event and data",
			event:    "test-event",
			data:     `{"message":"hello"}`,
			expected: "event: test-event\ndata: {\"message\":\"hello\"}\n\n",
		},
		{
			name:     "Data only",
			event:    "",
			data:     "simple message",
			expected: "data: simple message\n\n",
		},
		{
			name:     "Event only",
			event:    "ping",
			data:     "",
			expected: "event: ping\n\n",
		},
		{
			name:     "Empty",
			event:    "",
			data:     "",
			expected: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSSEMessage(tt.event, tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleSSE_MemoryLeakDetection(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	numIterations := 50
	for i := 0; i < numIterations; i++ {
		clientID := fmt.Sprintf("leak-test-client-%d", i)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
		ctx, cancel := context.WithTimeout(req.Context(), 10*time.Millisecond)
		req = req.WithContext(ctx)

		router.ServeHTTP(w, req)
		cancel()

		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, GetConnectedClients(), "Expected all clients to be cleaned up")
}

func TestHandleSSE_EventStreaming(t *testing.T) {
	clearClients()
	router := setupTestRouter()

	clientID := "stream-test-client"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan bool)
	go func() {
		router.ServeHTTP(w, req)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	events := []struct {
		event string
		data  string
	}{
		{"status", `{"type":"connected"}`},
		{"message", `{"text":"hello"}`},
		{"complete", `{"status":"done"}`},
	}

	for _, e := range events {
		err := SendToClient(clientID, e.event, e.data)
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	<-done

	body := w.Body.String()
	for _, e := range events {
		assert.Contains(t, body, fmt.Sprintf("event: %s", e.event))
		assert.Contains(t, body, fmt.Sprintf("data: %s", e.data))
	}

	lines := strings.Split(body, "\n")
	eventCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			eventCount++
		}
	}
	assert.Equal(t, len(events), eventCount)
}
