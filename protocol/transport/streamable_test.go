package transport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/protocol/transport"
)

func TestNewStreamableHTTP(t *testing.T) {
	tr, err := transport.NewStreamableHTTP(transport.DefaultConfig())
	if err != nil {
		t.Fatalf("NewStreamableHTTP returned error: %v", err)
	}
	if tr == nil {
		t.Fatal("NewStreamableHTTP returned nil")
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestServeNilHandler(t *testing.T) {
	tr, err := transport.NewStreamableHTTP(transport.DefaultConfig())
	if err != nil {
		t.Fatalf("NewStreamableHTTP: %v", err)
	}
	defer tr.Close()

	err = tr.Serve(context.Background(), nil)
	if err == nil {
		t.Error("Serve with nil handler: expected error")
	}
}

func TestServeHTTPPost(t *testing.T) {
	tr, err := transport.NewStreamableHTTP(transport.DefaultConfig())
	if err != nil {
		t.Fatalf("NewStreamableHTTP: %v", err)
	}
	defer tr.Close()

	// Create test request.
	reqMsg := transport.Message{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test/echo",
		Params:  "hello",
	}
	body, _ := json.Marshal(reqMsg)

	// Start a context-cancelable serve in background.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(_ context.Context, msg transport.Message) (transport.Message, error) {
		return transport.Message{
			Result: msg.Params,
		}, nil
	}

	// Use ServeHTTP directly for testing.
	go func() {
		tr.Serve(ctx, handler)
	}()
	time.Sleep(50 * time.Millisecond)

	// Test via ServeHTTP.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	tr.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP status: got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp transport.Message
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC: got %q, want %q", resp.JSONRPC, "2.0")
	}
}

func TestServeHTTPMethodNotAllowed(t *testing.T) {
	tr, err := transport.NewStreamableHTTP(transport.DefaultConfig())
	if err != nil {
		t.Fatalf("NewStreamableHTTP: %v", err)
	}
	defer tr.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	tr.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("HTTP status: got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestServeHTTPInvalidJSON(t *testing.T) {
	tr, err := transport.NewStreamableHTTP(transport.DefaultConfig())
	if err != nil {
		t.Fatalf("NewStreamableHTTP: %v", err)
	}
	defer tr.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("not json"))
	tr.ServeHTTP(rec, req)

	var resp transport.Message
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode error response: %v", err)
	}
	if resp.Error == nil {
		t.Error("Expected error response for invalid JSON")
	}
	if resp.Error != nil && resp.Error.Code != -32700 {
		t.Errorf("Error code: got %d, want %d", resp.Error.Code, -32700)
	}
}

func TestServeHTTPNotFound(t *testing.T) {
	tr, err := transport.NewStreamableHTTP(transport.DefaultConfig())
	if err != nil {
		t.Fatalf("NewStreamableHTTP: %v", err)
	}
	defer tr.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	tr.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("HTTP status: got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSendSSENoSession(t *testing.T) {
	tr, err := transport.NewStreamableHTTP(transport.DefaultConfig())
	if err != nil {
		t.Fatalf("NewStreamableHTTP: %v", err)
	}
	defer tr.Close()

	err = tr.SendSSE(context.Background(), "nonexistent", strings.NewReader("data"))
	if err == nil {
		t.Error("SendSSE to nonexistent session: expected error")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := transport.DefaultConfig()
	if cfg.Listen != ":9090" {
		t.Errorf("Listen: got %q, want %q", cfg.Listen, ":9090")
	}
	if cfg.ReadTimeout != 15 {
		t.Errorf("ReadTimeout: got %d, want 15", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 30 {
		t.Errorf("WriteTimeout: got %d, want 30", cfg.WriteTimeout)
	}
}
