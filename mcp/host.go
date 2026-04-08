package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/pkg/gateway"
)

// ServerStatus represents the health and status of an MCP server connection.
type ServerStatus struct {
	Name        string    `json:"name"`
	Connected   bool      `json:"connected"`
	ToolCount   int       `json:"tool_count"`
	LastError   string    `json:"last_error,omitempty"`
	LastChecked time.Time `json:"last_checked"`
}

// MCPHostManager manages multiple MCP server connections and coordinates
// tool discovery, registration, and health monitoring.
type MCPHostManager struct {
	config       *MCPConfig
	toolRegistry gateway.ToolRegistry
	connections  map[string]*MCPServerConnection
	mu           sync.RWMutex
	cancelHealth context.CancelFunc
	healthWg     sync.WaitGroup
}

// NewMCPHostManager creates a new MCP host manager with the given configuration.
// The toolRegistry is used to register discovered tools from MCP servers.
func NewMCPHostManager(cfg *MCPConfig, toolRegistry gateway.ToolRegistry) (*MCPHostManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("MCP config cannot be nil")
	}

	if toolRegistry == nil {
		return nil, fmt.Errorf("tool registry cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MCP config: %w", err)
	}

	return &MCPHostManager{
		config:       cfg,
		toolRegistry: toolRegistry,
		connections:  make(map[string]*MCPServerConnection),
	}, nil
}

// Start connects to all configured MCP servers, discovers their tools,
// and registers them in the Gateway tool registry with namespace prefixes.
// Starts health check loops for each server.
func (m *MCPHostManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create health check context
	healthCtx, cancel := context.WithCancel(context.Background())
	m.cancelHealth = cancel

	// Connect to each server and discover tools
	for _, serverCfg := range m.config.Servers {
		if err := m.startServer(ctx, serverCfg, healthCtx); err != nil {
			// Log error but continue with other servers (graceful degradation)
			slog.Warn("failed to start MCP server", "server", serverCfg.ID, "error", err)
			continue
		}
	}

	return nil
}

// startServer connects to a single MCP server, discovers tools, and starts health monitoring.
func (m *MCPHostManager) startServer(ctx context.Context, serverCfg MCPServerConfig, healthCtx context.Context) error {
	// Create connection
	conn, err := NewMCPServerConnection(serverCfg.ID, serverCfg)
	if err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}

	// Connect to server
	if err := conn.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Discover tools
	tools, err := conn.ListTools(ctx)
	if err != nil {
		conn.Disconnect(ctx)
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// Register tools in Gateway registry
	for _, mcpTool := range tools {
		toolDef := CreateToolDefinition(mcpTool, conn, serverCfg.NamespacePrefix)
		if err := m.toolRegistry.Register(ctx, toolDef); err != nil {
			slog.Warn("failed to register tool", "tool", toolDef.Name, "server", serverCfg.ID, "error", err)
			// Continue registering other tools
		}
	}

	// Store connection (use ID as the key)
	m.connections[serverCfg.ID] = conn

	// Start health check loop if interval is configured
	if serverCfg.HealthCheck.IntervalSec > 0 {
		m.healthWg.Add(1)
		go m.healthCheckLoop(healthCtx, serverCfg.ID, serverCfg.HealthCheck.IntervalSec)
	}

	slog.Info("MCP server connected", "server", serverCfg.ID, "display_name", serverCfg.DisplayName, "tool_count", len(tools))

	return nil
}

// healthCheckLoop periodically checks the health of an MCP server connection
// and attempts to reconnect if the connection is lost.
func (m *MCPHostManager) healthCheckLoop(ctx context.Context, serverName string, intervalSec int) {
	defer m.healthWg.Done()

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkAndReconnect(ctx, serverName)
		}
	}
}

// checkAndReconnect checks if a server connection is healthy and reconnects if needed.
func (m *MCPHostManager) checkAndReconnect(ctx context.Context, serverName string) {
	m.mu.RLock()
	conn, exists := m.connections[serverName]
	m.mu.RUnlock()

	if !exists {
		return
	}

	// Check health
	if conn.IsHealthy() {
		return
	}

	// Connection is unhealthy - attempt reconnect
	slog.Warn("MCP server is unhealthy, attempting reconnect", "server", serverName)

	// Find server config
	var serverCfg *MCPServerConfig
	for _, cfg := range m.config.Servers {
		if cfg.ID == serverName {
			serverCfg = &cfg
			break
		}
	}

	if serverCfg == nil {
		slog.Error("server config not found", "server", serverName)
		return
	}

	// Disconnect old connection
	if err := conn.Disconnect(ctx); err != nil {
		slog.Warn("error disconnecting from server", "server", serverName, "error", err)
	}

	// Create new connection
	newConn, err := NewMCPServerConnection(serverCfg.ID, *serverCfg)
	if err != nil {
		slog.Error("failed to create new connection", "server", serverName, "error", err)
		return
	}

	// Attempt to connect
	if err := newConn.Connect(ctx); err != nil {
		slog.Error("failed to reconnect", "server", serverName, "error", err)
		return
	}

	// Update connection
	m.mu.Lock()
	m.connections[serverName] = newConn
	m.mu.Unlock()

	slog.Info("MCP server reconnected successfully", "server", serverName)
}

// Stop gracefully shuts down all MCP server connections and stops health checks.
func (m *MCPHostManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel health check loops
	if m.cancelHealth != nil {
		m.cancelHealth()
	}

	// Wait for health check goroutines to finish (with timeout)
	done := make(chan struct{})
	go func() {
		m.healthWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All health checks stopped
	case <-time.After(5 * time.Second):
		slog.Warn("timeout waiting for health checks to stop")
	}

	// Disconnect from all servers
	var lastErr error
	for name, conn := range m.connections {
		if err := conn.Disconnect(ctx); err != nil {
			slog.Error("error disconnecting from server", "server", name, "error", err)
			lastErr = err
		}
	}

	m.connections = make(map[string]*MCPServerConnection)

	return lastErr
}

// Status returns the current status of all MCP server connections.
func (m *MCPHostManager) Status() map[string]ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]ServerStatus)

	for name, conn := range m.connections {
		// Count tools for this server
		toolCount := 0
		if tools, err := m.toolRegistry.ListByNamespace(context.Background(), m.getNamespacePrefix(name)); err == nil {
			toolCount = len(tools)
		}

		status[name] = ServerStatus{
			Name:        name,
			Connected:   conn.IsHealthy(),
			ToolCount:   toolCount,
			LastChecked: time.Now(),
		}
	}

	return status
}

// getNamespacePrefix returns the namespace prefix for a given server ID.
func (m *MCPHostManager) getNamespacePrefix(serverID string) string {
	for _, cfg := range m.config.Servers {
		if cfg.ID == serverID {
			return cfg.NamespacePrefix
		}
	}
	return ""
}
