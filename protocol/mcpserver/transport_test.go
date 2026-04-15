package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	mcpserver "github.com/DojoGenesis/gateway/protocol/mcpserver"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// --- HTTP Transport Tests ---

func setupHTTPServer(t *testing.T) (mcpserver.Server, string) {
	t.Helper()
	srv, err := mcpserver.NewServer(mcpserver.DefaultConfig())
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	_ = srv.Start(context.Background())

	_ = srv.RegisterTool(mcpserver.ToolRegistration{
		Name:        "echo",
		Description: "Echo tool",
		Handler: func(_ context.Context, _ string, args map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"echoed": args["msg"]}, nil
		},
	})

	// Start HTTP server on ephemeral port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := fmt.Sprintf("http://%s/mcp", ln.Addr().String())

	go func() { _ = srv.ListenAndServeOnListener(ln) }()
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })

	return srv, addr
}

func TestHTTPTransport_Initialize(t *testing.T) {
	_, addr := setupHTTPServer(t)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(addr, "application/json", bytes.NewReader(body)) //nolint:gosec // test server URL
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var rpcResp jsonRPCResponse
	_ = json.NewDecoder(resp.Body).Decode(&rpcResp)

	if rpcResp.JSONRPC != "2.0" {
		t.Errorf("jsonrpc: got %q", rpcResp.JSONRPC)
	}
	if rpcResp.Error != nil {
		t.Errorf("unexpected error: %v", rpcResp.Error)
	}

	result, ok := rpcResp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type: %T", rpcResp.Result)
	}
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion: got %v", result["protocolVersion"])
	}
}

func TestHTTPTransport_ToolCall(t *testing.T) {
	_, addr := setupHTTPServer(t)

	params, _ := json.Marshal(map[string]interface{}{
		"name":      "echo",
		"arguments": map[string]interface{}{"msg": "hello"},
	})
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params:  params,
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(addr, "application/json", bytes.NewReader(body)) //nolint:gosec // test server URL
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var rpcResp jsonRPCResponse
	_ = json.NewDecoder(resp.Body).Decode(&rpcResp)

	if rpcResp.Error != nil {
		t.Fatalf("unexpected error: %v", rpcResp.Error)
	}
}

func TestHTTPTransport_UnknownMethod(t *testing.T) {
	_, addr := setupHTTPServer(t)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "bogus/method",
	}
	body, _ := json.Marshal(req)

	resp, err := http.Post(addr, "application/json", bytes.NewReader(body)) //nolint:gosec // test server URL
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var rpcResp jsonRPCResponse
	_ = json.NewDecoder(resp.Body).Decode(&rpcResp)

	if rpcResp.Error == nil {
		t.Error("expected error for unknown method")
	}
}

func TestHTTPTransport_MethodNotAllowed(t *testing.T) {
	_, addr := setupHTTPServer(t)

	resp, err := http.Get(addr) //nolint:gosec // test server URL
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

// --- Stdio Transport Tests ---

func TestStdioTransport_Initialize(t *testing.T) {
	srv, _ := mcpserver.NewServer(mcpserver.DefaultConfig())
	_ = srv.Start(context.Background())
	defer func() { _ = srv.Stop(context.Background()) }()

	req := `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n"
	reader := strings.NewReader(req)
	var writer bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := srv.ServeStdio(ctx, reader, &writer)
	if err != nil {
		t.Fatalf("ServeStdio: %v", err)
	}

	var rpcResp jsonRPCResponse
	_ = json.NewDecoder(&writer).Decode(&rpcResp)

	if rpcResp.JSONRPC != "2.0" {
		t.Errorf("jsonrpc: got %q", rpcResp.JSONRPC)
	}

	result, ok := rpcResp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type: %T", rpcResp.Result)
	}
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion: got %v", result["protocolVersion"])
	}
}

func TestStdioTransport_ToolCall(t *testing.T) {
	srv, _ := mcpserver.NewServer(mcpserver.DefaultConfig())
	_ = srv.Start(context.Background())
	defer func() { _ = srv.Stop(context.Background()) }()

	_ = srv.RegisterTool(mcpserver.ToolRegistration{
		Name: "echo",
		Handler: func(_ context.Context, _ string, args map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{"echoed": args["msg"]}, nil
		},
	})

	req := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"msg":"stdio-test"}}}` + "\n"
	reader := strings.NewReader(req)
	var writer bytes.Buffer

	_ = srv.ServeStdio(context.Background(), reader, &writer)

	var rpcResp jsonRPCResponse
	_ = json.NewDecoder(&writer).Decode(&rpcResp)

	if rpcResp.Error != nil {
		t.Fatalf("unexpected error: %v", rpcResp.Error)
	}
}

func TestStdioTransport_ParseError(t *testing.T) {
	srv, _ := mcpserver.NewServer(mcpserver.DefaultConfig())
	_ = srv.Start(context.Background())
	defer func() { _ = srv.Stop(context.Background()) }()

	req := "not-json\n"
	reader := strings.NewReader(req)
	var writer bytes.Buffer

	_ = srv.ServeStdio(context.Background(), reader, &writer)

	var rpcResp jsonRPCResponse
	_ = json.NewDecoder(&writer).Decode(&rpcResp)

	if rpcResp.Error == nil {
		t.Error("expected parse error")
	}
}
