package mcpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// maxRequestBodySize limits incoming JSON-RPC request bodies (10MB).
const maxRequestBodySize = 10 * 1024 * 1024

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

// jsonRPCError is a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ServeHTTP handles Streamable HTTP MCP requests.
func (s *mcpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent memory exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error")
		return
	}

	if req.JSONRPC != "2.0" {
		writeJSONRPCError(w, req.ID, -32600, "invalid request: jsonrpc must be 2.0")
		return
	}

	result, err := s.HandleMessage(r.Context(), req.Method, req.Params)
	if err != nil {
		writeJSONRPCError(w, req.ID, -32603, err.Error())
		return
	}

	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("mcpserver: failed to encode response", "error", err)
	}
}

// ListenAndServe starts the HTTP transport on the configured address.
func (s *mcpServer) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/mcp", s)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return s.httpServer.ListenAndServe()
}

// ListenAndServeOnListener starts the HTTP transport on a pre-bound listener.
func (s *mcpServer) ListenAndServeOnListener(ln net.Listener) error {
	mux := http.NewServeMux()
	mux.Handle("/mcp", s)
	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return s.httpServer.Serve(ln)
}

func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors are 200 at HTTP level.
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("mcpserver: failed to encode error response", "error", err)
	}
}

// Addr returns the listener address, or empty if not serving.
func (s *mcpServer) Addr() string {
	if s.httpServer == nil {
		return ""
	}
	return fmt.Sprintf("http://%s/mcp", s.config.Listen)
}
