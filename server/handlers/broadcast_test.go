package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/server/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupBroadcastRouter() *gin.Engine {
	r := gin.New()
	r.POST("/broadcast", HandleBroadcast)
	r.GET("/events", HandleSSE)
	return r
}

func TestHandleBroadcast_ValidRequest(t *testing.T) {
	clearClients()
	router := setupBroadcastRouter()

	clientID := "test-client-broadcast"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sseRecorder := httptest.NewRecorder()
	sseReq, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
	sseReq = sseReq.WithContext(ctx)

	done := make(chan bool)
	go func() {
		router.ServeHTTP(sseRecorder, sseReq)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 1, GetConnectedClients())

	broadcastReq := models.BroadcastRequest{
		ClientID: clientID,
		Event:    "test-event",
		Data:     `{"message":"hello world"}`,
	}

	body, err := json.Marshal(broadcastReq)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, clientID, response["client_id"])
	assert.Equal(t, true, response["delivered"])

	cancel()
	<-done

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, GetConnectedClients())
}

func TestHandleBroadcast_MissingClientID(t *testing.T) {
	clearClients()
	router := setupBroadcastRouter()

	broadcastReq := models.BroadcastRequest{
		Event: "test-event",
		Data:  `{"message":"hello"}`,
	}

	body, err := json.Marshal(broadcastReq)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "ClientID")
}

func TestHandleBroadcast_MissingEvent(t *testing.T) {
	clearClients()
	router := setupBroadcastRouter()

	broadcastReq := models.BroadcastRequest{
		ClientID: "test-client",
		Data:     `{"message":"hello"}`,
	}

	body, err := json.Marshal(broadcastReq)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Event")
}

func TestHandleBroadcast_MissingData(t *testing.T) {
	clearClients()
	router := setupBroadcastRouter()

	broadcastReq := models.BroadcastRequest{
		ClientID: "test-client",
		Event:    "test-event",
	}

	body, err := json.Marshal(broadcastReq)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Data")
}

func TestHandleBroadcast_InvalidJSON(t *testing.T) {
	clearClients()
	router := setupBroadcastRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request")
}

func TestHandleBroadcast_ClientNotConnected(t *testing.T) {
	clearClients()
	router := setupBroadcastRouter()

	broadcastReq := models.BroadcastRequest{
		ClientID: "non-existent-client",
		Event:    "test-event",
		Data:     `{"message":"hello"}`,
	}

	body, err := json.Marshal(broadcastReq)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, false, response["delivered"])
}

func TestHandleBroadcast_ChannelBufferFull(t *testing.T) {
	clearClients()

	clientID := "test-client-buffer-full"
	client := &models.Client{
		ID:      clientID,
		Channel: make(chan string, 1),
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

	router := setupBroadcastRouter()

	client.Channel <- "blocking message"

	broadcastReq := models.BroadcastRequest{
		ClientID: clientID,
		Event:    "test-event",
		Data:     `{"message":"should fail"}`,
	}

	body, err := json.Marshal(broadcastReq)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, false, response["delivered"])
}

func TestHandleBroadcast_MultipleEvents(t *testing.T) {
	clearClients()
	router := setupBroadcastRouter()

	clientID := "test-client-multiple"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sseRecorder := httptest.NewRecorder()
	sseReq, _ := http.NewRequest("GET", "/events?client_id="+clientID, nil)
	sseReq = sseReq.WithContext(ctx)

	done := make(chan bool)
	go func() {
		router.ServeHTTP(sseRecorder, sseReq)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 1, GetConnectedClients())

	events := []models.BroadcastRequest{
		{ClientID: clientID, Event: "event1", Data: `{"num":1}`},
		{ClientID: clientID, Event: "event2", Data: `{"num":2}`},
		{ClientID: clientID, Event: "event3", Data: `{"num":3}`},
	}

	for _, event := range events {
		body, err := json.Marshal(event)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/broadcast", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, true, response["delivered"])
	}

	cancel()
	<-done
}
