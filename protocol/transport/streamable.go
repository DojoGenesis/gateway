package transport

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// Config holds configuration for the Streamable HTTP transport.
type Config struct {
	// Listen is the address to listen on (e.g., ":9090").
	Listen string

	// ReadTimeout is the HTTP read timeout in seconds.
	ReadTimeout int

	// WriteTimeout is the HTTP write timeout in seconds for non-SSE endpoints.
	WriteTimeout int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Listen:       ":9090",
		ReadTimeout:  15,
		WriteTimeout: 30,
	}
}

// Message represents a JSON-RPC message.
type Message struct {
	// JSONRPC is the protocol version ("2.0").
	JSONRPC string `json:"jsonrpc"`

	// ID is the request identifier. May be nil for notifications.
	ID interface{} `json:"id,omitempty"`

	// Method is the RPC method name.
	Method string `json:"method,omitempty"`

	// Params are the method parameters.
	Params interface{} `json:"params,omitempty"`

	// Result is the response result. Set for responses.
	Result interface{} `json:"result,omitempty"`

	// Error is the response error. Set for error responses.
	Error *RPCError `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Handler processes incoming JSON-RPC messages and writes responses.
type Handler func(ctx context.Context, msg Message) (Message, error)

// Transport manages the Streamable HTTP transport layer.
type Transport interface {
	// Serve starts the HTTP server and routes requests to the handler.
	Serve(ctx context.Context, handler Handler) error

	// ServeHTTP implements http.Handler for embedding in existing servers.
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// SendSSE sends a server-sent event to connected clients.
	SendSSE(ctx context.Context, sessionID string, data io.Reader) error

	// Close shuts down the transport.
	Close() error
}

// streamableHTTP implements Streamable HTTP transport for JSON-RPC with SSE support.
type streamableHTTP struct {
	config   Config
	handler  Handler
	server   *http.Server
	listener net.Listener

	mu       sync.RWMutex
	sessions map[string]*sseSession
	closed   bool
}

type sseSession struct {
	id      string
	mu      sync.Mutex // protects writes to w/flusher after session lookup (#8)
	w       http.ResponseWriter
	flusher http.Flusher
	done    chan struct{}
}

// NewStreamableHTTP creates a new Streamable HTTP transport.
func NewStreamableHTTP(config Config) (Transport, error) {
	return &streamableHTTP{
		config:   config,
		sessions: make(map[string]*sseSession),
	}, nil
}

func (t *streamableHTTP) Serve(ctx context.Context, handler Handler) error {
	if handler == nil {
		return fmt.Errorf("transport: handler is required")
	}

	// Set handler under lock to avoid data race with ServeHTTP/handleMCP. (#1)
	t.mu.Lock()
	t.handler = handler
	t.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", t.handleMCP)
	mux.HandleFunc("/mcp/sse", t.handleSSE)

	t.server = &http.Server{
		Addr:        t.config.Listen,
		Handler:     mux,
		ReadTimeout: time.Duration(t.config.ReadTimeout) * time.Second,
		// WriteTimeout is NOT set on the server because SSE requires no write
		// deadline. Per-request write deadlines are enforced in handleMCP. (#12)
	}

	ln, err := net.Listen("tcp", t.config.Listen)
	if err != nil {
		return fmt.Errorf("transport: listen: %w", err)
	}
	t.listener = ln

	errCh := make(chan error, 1)
	go func() {
		errCh <- t.server.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		// Use a bounded shutdown context instead of context.Background(). (#23)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = t.server.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (t *streamableHTTP) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Apply per-request write deadline for non-SSE endpoints. (#12)
	if t.config.WriteTimeout > 0 {
		rc := http.NewResponseController(w)
		_ = rc.SetWriteDeadline(time.Now().Add(time.Duration(t.config.WriteTimeout) * time.Second))
	}

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error")
		return
	}

	// Read handler under lock to avoid data race with Serve(). (#1)
	t.mu.RLock()
	handler := t.handler
	t.mu.RUnlock()

	if handler == nil {
		writeJSONRPCError(w, msg.ID, -32603, "no handler registered")
		return
	}

	resp, err := handler(r.Context(), msg)
	if err != nil {
		writeJSONRPCError(w, msg.ID, -32603, err.Error())
		return
	}

	resp.JSONRPC = "2.0"
	if resp.ID == nil {
		resp.ID = msg.ID
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (t *streamableHTTP) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		// Use cryptographically random session IDs. (#27)
		sessionID = generateSessionID()
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	session := &sseSession{
		id:      sessionID,
		w:       w,
		flusher: flusher,
		done:    make(chan struct{}),
	}

	t.mu.Lock()
	t.sessions[sessionID] = session
	t.mu.Unlock()

	// Send initial connection event.
	_, _ = fmt.Fprintf(w, "event: endpoint\ndata: /mcp\n\n")
	flusher.Flush()

	// Keep connection alive until closed.
	select {
	case <-r.Context().Done():
	case <-session.done:
	}

	t.mu.Lock()
	delete(t.sessions, sessionID)
	t.mu.Unlock()
}

func (t *streamableHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/mcp":
		t.handleMCP(w, r)
	case "/mcp/sse":
		t.handleSSE(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (t *streamableHTTP) SendSSE(_ context.Context, sessionID string, data io.Reader) error {
	t.mu.RLock()
	session, ok := t.sessions[sessionID]
	t.mu.RUnlock()

	if !ok {
		return fmt.Errorf("transport: session %q not found", sessionID)
	}

	// Hold the session's own mutex during write to prevent concurrent
	// writes and writes after the HTTP handler has returned. (#8)
	session.mu.Lock()
	defer session.mu.Unlock()

	select {
	case <-session.done:
		return fmt.Errorf("transport: session %q is closed", sessionID)
	default:
	}

	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		_, _ = fmt.Fprintf(session.w, "data: %s\n", scanner.Text())
	}
	_, _ = fmt.Fprintf(session.w, "\n")
	session.flusher.Flush()

	return scanner.Err()
}

func (t *streamableHTTP) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	// Close all SSE sessions.
	for _, session := range t.sessions {
		close(session.done)
	}
	t.sessions = nil

	if t.server != nil {
		return t.server.Close()
	}
	return nil
}

func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := Message{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors still use 200
	_ = json.NewEncoder(w).Encode(resp)
}

// generateSessionID returns a cryptographically random hex session ID. (#27)
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
