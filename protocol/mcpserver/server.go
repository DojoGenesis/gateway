package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// tracer is the package-level OpenTelemetry tracer for MCP protocol operations.
var tracer = otel.Tracer("dojo.protocol.mcp")

// Config holds configuration for the MCP server.
type Config struct {
	// Enabled controls whether the MCP server is active.
	Enabled bool

	// Transport is the transport type ("streamable_http" or "stdio").
	Transport string

	// Listen is the address to listen on (e.g., ":9090").
	Listen string

	// Tools lists which Dojo tools to expose via MCP.
	Tools []string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:   false,
		Transport: "streamable_http",
		Listen:    ":9090",
		Tools: []string{
			"dojo.skill.list",
			"dojo.skill.invoke",
			"dojo.tool.list",
			"dojo.remember",
			"dojo.observe.trace",
		},
	}
}

// ToolHandler processes an MCP tool call.
type ToolHandler func(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)

// ResourceHandler processes an MCP resource read.
type ResourceHandler func(ctx context.Context, uri string) (interface{}, error)

// ToolRegistration describes an MCP tool to expose.
type ToolRegistration struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     ToolHandler
}

// ResourceRegistration describes an MCP resource to expose.
type ResourceRegistration struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Handler     ResourceHandler
}

// Server is the bidirectional MCP server interface.
type Server interface {
	// Start begins listening for MCP client connections.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the server.
	Stop(ctx context.Context) error

	// RegisterTool exposes a tool via MCP.
	RegisterTool(reg ToolRegistration) error

	// RegisterResource exposes a resource via MCP.
	RegisterResource(reg ResourceRegistration) error

	// HandleMessage processes a JSON-RPC message according to the MCP protocol. (#7)
	HandleMessage(ctx context.Context, method string, params json.RawMessage) (interface{}, error)

	// ListenAndServe starts the Streamable HTTP transport.
	ListenAndServe(addr string) error

	// ListenAndServeOnListener starts HTTP on a pre-bound listener.
	ListenAndServeOnListener(ln net.Listener) error

	// ServeStdio reads JSON-RPC from reader and writes responses to writer.
	ServeStdio(ctx context.Context, reader io.Reader, writer io.Writer) error

	// ServeHTTP implements http.Handler for the MCP endpoint.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// mcpServer implements the MCP server with JSON-RPC message handling.
type mcpServer struct {
	mu         sync.RWMutex
	config     Config
	tools      map[string]ToolRegistration
	resources  map[string]ResourceRegistration
	ctx        context.Context    // server-scoped context derived from Start (#22)
	cancel     context.CancelFunc // cancels ctx on Stop
	running    bool
	httpServer *http.Server // HTTP transport server
}

// NewServer creates a new MCP server with the given configuration.
func NewServer(config Config) (Server, error) {
	return &mcpServer{
		config:    config,
		tools:     make(map[string]ToolRegistration),
		resources: make(map[string]ResourceRegistration),
	}, nil
}

func (s *mcpServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("mcpserver: already running")
	}

	// Store the server-scoped context so HandleMessage can respect
	// cancellation when Stop() is called. (#22)
	serverCtx, cancel := context.WithCancel(ctx)
	s.ctx = serverCtx
	s.cancel = cancel
	s.running = true

	return nil
}

func (s *mcpServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// Shut down HTTP server if running.
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			_ = s.httpServer.Close() // force close on shutdown failure
		}
		s.httpServer = nil
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
	return nil
}

func (s *mcpServer) RegisterTool(reg ToolRegistration) error {
	if reg.Name == "" {
		return fmt.Errorf("mcpserver: tool name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tools[reg.Name]; exists {
		return fmt.Errorf("mcpserver: tool %q already registered", reg.Name)
	}

	s.tools[reg.Name] = reg
	return nil
}

func (s *mcpServer) RegisterResource(reg ResourceRegistration) error {
	if reg.URI == "" {
		return fmt.Errorf("mcpserver: resource URI is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.resources[reg.URI]; exists {
		return fmt.Errorf("mcpserver: resource %q already registered", reg.URI)
	}

	s.resources[reg.URI] = reg
	return nil
}

// HandleMessage processes a JSON-RPC message according to the MCP protocol.
func (s *mcpServer) HandleMessage(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "initialize":
		return s.handleInitialize()
	case "tools/list":
		return s.handleToolsList()
	case "tools/call":
		return s.handleToolsCall(ctx, params)
	case "resources/list":
		return s.handleResourcesList()
	case "resources/read":
		return s.handleResourcesRead(ctx, params)
	default:
		return nil, fmt.Errorf("mcpserver: unknown method %q", method)
	}
}

func (s *mcpServer) handleInitialize() (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{"listChanged": true},
			"resources": map[string]interface{}{"subscribe": false, "listChanged": true},
		},
		"serverInfo": map[string]interface{}{
			"name":    "dojo-platform",
			"version": "2.0.0",
		},
	}, nil
}

func (s *mcpServer) handleToolsList() (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]map[string]interface{}, 0, len(s.tools))
	for _, t := range s.tools {
		tool := map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
		}
		if t.InputSchema != nil {
			tool["inputSchema"] = t.InputSchema
		}
		tools = append(tools, tool)
	}
	return map[string]interface{}{"tools": tools}, nil
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func (s *mcpServer) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p toolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("mcpserver: invalid tool call params: %w", err)
	}

	ctx, span := tracer.Start(ctx, "mcp.tool_call", trace.WithAttributes(
		attribute.String("tool.name", p.Name),
	))
	defer span.End()

	s.mu.RLock()
	tool, ok := s.tools[p.Name]
	s.mu.RUnlock()

	if !ok {
		err := fmt.Errorf("mcpserver: tool %q not found", p.Name)
		span.RecordError(err)
		return nil, err
	}
	if tool.Handler == nil {
		err := fmt.Errorf("mcpserver: tool %q has no handler", p.Name)
		span.RecordError(err)
		return nil, err
	}

	result, err := tool.Handler(ctx, p.Name, p.Arguments)
	if err != nil {
		span.RecordError(err)
		// Return domain errors as MCP isError responses, not Go errors.
		// Go errors are reserved for internal/transport failures. (#13)
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": err.Error()},
			},
			"isError": true,
		}, nil
	}

	// Use JSON marshaling to preserve type information. (#28)
	resultJSON, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return nil, fmt.Errorf("mcpserver: marshal tool result: %w", marshalErr)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(resultJSON)},
		},
	}, nil
}

func (s *mcpServer) handleResourcesList() (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]map[string]interface{}, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, map[string]interface{}{
			"uri":         r.URI,
			"name":        r.Name,
			"description": r.Description,
			"mimeType":    r.MimeType,
		})
	}
	return map[string]interface{}{"resources": resources}, nil
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

func (s *mcpServer) handleResourcesRead(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p resourceReadParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("mcpserver: invalid resource read params: %w", err)
	}

	ctx, span := tracer.Start(ctx, "mcp.resource_read", trace.WithAttributes(
		attribute.String("resource.uri", p.URI),
	))
	defer span.End()

	s.mu.RLock()
	resource, ok := s.resources[p.URI]
	s.mu.RUnlock()

	if !ok {
		err := fmt.Errorf("mcpserver: resource %q not found", p.URI)
		span.RecordError(err)
		return nil, err
	}
	if resource.Handler == nil {
		err := fmt.Errorf("mcpserver: resource %q has no handler", p.URI)
		span.RecordError(err)
		return nil, err
	}

	result, err := resource.Handler(ctx, p.URI)
	if err != nil {
		span.RecordError(err)
		// Return domain errors distinctly from internal errors. (#13)
		return nil, fmt.Errorf("mcpserver: resource %q handler: %w", p.URI, err)
	}

	// Use JSON marshaling to preserve type information. (#28)
	resultJSON, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return nil, fmt.Errorf("mcpserver: marshal resource result: %w", marshalErr)
	}

	return map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      p.URI,
				"mimeType": resource.MimeType,
				"text":     string(resultJSON),
			},
		},
	}, nil
}
