# MCP Server Configuration Guide

**Version:** v0.2.0 (Phase 3)
**Audience:** Developers, DevOps engineers
**Purpose:** How to configure and add custom MCP servers to the AgenticGateway

---

## Table of Contents

1. [Overview](#overview)
2. [Configuration File Structure](#configuration-file-structure)
3. [Adding a New MCP Server](#adding-a-new-mcp-server)
4. [Transport Types](#transport-types)
5. [Tool Filtering](#tool-filtering)
6. [Timeouts and Retry Policy](#timeouts-and-retry-policy)
7. [Health Checks](#health-checks)
8. [Environment Variables](#environment-variables)
9. [Observability](#observability)
10. [Troubleshooting](#troubleshooting)

---

## Overview

The AgenticGateway can connect to external MCP (Model Context Protocol) servers to dynamically extend its tool library. MCP servers expose tools via:

- **stdio transport** (local subprocess, binary communication via stdin/stdout)
- **SSE transport** (remote HTTP server with Server-Sent Events)
- **Streamable HTTP** (future — not yet implemented)

All MCP servers are configured via YAML (`gateway-config.yaml` by default) and registered at startup.

**Key Benefits:**
- **Zero code changes** to add new tools
- **Namespace isolation** prevents naming conflicts
- **Auto-discovery** of tools from MCP servers
- **OTEL tracing** for all MCP tool invocations

---

## Configuration File Structure

The MCP configuration follows the schema defined in `specs/v0.2.0/gateway-mcp-contract.md`.

**Top-level structure:**

```yaml
version: "1.0"

mcp:
  global:
    # Global MCP settings (defaults for all servers)
    default_tool_timeout: 30
    reconnect:
      max_retries: 3
      initial_backoff_ms: 1000
      max_backoff_ms: 30000
      backoff_multiplier: 2.0
    health_check_interval: 60
    response_buffer_size: 1048576

  servers:
    # Array of MCP server configurations
    - id: "server_name"
      display_name: "Human-Readable Name"
      namespace_prefix: "namespace"
      transport: { ... }
      tools: { ... }
      timeouts: { ... }
      health_check: { ... }
      retry_policy: { ... }

  observability:
    enabled: true
    trace_provider: "otel"
    attributes:
      service.name: "agentic-gateway"
    log_level: "info"
    tool_span_sample_rate: 1.0
```

---

## Adding a New MCP Server

### Step 1: Define Server Configuration

Add a new entry under `mcp.servers` in `gateway-config.yaml`:

```yaml
servers:
  - id: "my_custom_server"
    display_name: "My Custom MCP Server"
    namespace_prefix: "custom"

    transport:
      type: "stdio"
      command: "/path/to/my-mcp-server"
      args: ["--mode=production"]
      env:
        API_KEY: "${MY_SERVER_API_KEY}"
        LOG_LEVEL: "info"

    tools:
      allowlist: []  # empty = allow all
      blocklist: []

    timeouts:
      startup: 10
      tool_default: 30
      health_check: 5

    health_check:
      enabled: true
      interval_sec: 60

    retry_policy:
      max_attempts: 2
      backoff_multiplier: 2.0
      max_backoff_ms: 5000
```

### Step 2: Set Environment Variables (if needed)

```bash
export MY_SERVER_API_KEY="your-api-key-here"
```

### Step 3: Restart Gateway

```bash
# If using Docker Compose
docker-compose restart gateway

# If running locally
./agentic-gateway
```

### Step 4: Verify Registration

```bash
# Check MCP status
curl http://localhost:8080/admin/mcp/status

# List all tools (should include custom:* tools)
curl http://localhost:8080/v1/gateway/tools
```

---

## Transport Types

### stdio (Local Subprocess)

**Use case:** MCP server is a local binary (Go, Python, Node.js, etc.)

**Configuration:**

```yaml
transport:
  type: "stdio"
  command: "/opt/mcp-servers/my-server"
  args:
    - "--port=0"
    - "--debug=false"
  env:
    CONFIG_PATH: "/etc/my-server/config.json"
    LOG_LEVEL: "info"
```

**How it works:**
1. Gateway spawns the binary as a subprocess
2. Communicates via JSON-RPC on stdin/stdout
3. Process lifecycle managed by the gateway

**Best practices:**
- Use absolute paths for `command`
- Include health check to detect crashes
- Set reasonable `startup` timeout (binary startup time)

### SSE (Server-Sent Events)

**Use case:** MCP server is a remote HTTP service

**Configuration:**

```yaml
transport:
  type: "sse"
  url: "https://api.example.com/mcp/sse"
  headers:
    Authorization: "Bearer ${API_TOKEN}"
    X-Client-ID: "agentic-gateway-prod"
```

**How it works:**
1. Gateway connects to SSE endpoint
2. Receives tool discovery and results via SSE stream
3. Sends tool invocations via HTTP POST

**Best practices:**
- Use environment variables for API keys/tokens
- Set longer `startup` timeout for remote connections
- Configure `health_check` with appropriate interval for remote service

---

## Tool Filtering

### Allowlist (Whitelist)

Only register specific tools from the MCP server:

```yaml
tools:
  allowlist:
    - "search"
    - "fetch"
    - "summarize"
  blocklist: []
```

**Behavior:** If allowlist is **non-empty**, ONLY tools in the allowlist are registered.

### Blocklist (Blacklist)

Exclude specific tools from registration:

```yaml
tools:
  allowlist: []  # empty = allow all
  blocklist:
    - "dangerous_delete"
    - "admin_*"  # wildcard: blocks all tools starting with "admin_"
```

**Behavior:** Blocklist takes **precedence** over allowlist.

### Wildcards

Both allowlist and blocklist support `*` wildcards:

```yaml
tools:
  allowlist:
    - "user_*"    # Allow all user-related tools
  blocklist:
    - "user_delete_account"  # But block this specific one
```

---

## Timeouts and Retry Policy

### Timeouts

```yaml
timeouts:
  startup: 10         # Max time to establish connection (seconds)
  tool_default: 30    # Default per-tool invocation timeout (seconds)
  health_check: 5     # Health check timeout (seconds)
```

**Recommendations:**
- **stdio servers:** startup 5-10s, tool_default 30s
- **Remote servers:** startup 15-30s, tool_default 30-60s
- **Long-running operations:** configure per-tool timeouts (future feature)

### Retry Policy

```yaml
retry_policy:
  max_attempts: 2              # Total attempts (including initial)
  backoff_multiplier: 2.0      # Exponential backoff multiplier
  max_backoff_ms: 5000         # Max wait between retries
```

**How it works:**
1. First attempt fails → wait 500ms (base backoff)
2. Second attempt fails → wait 1000ms (500ms × 2.0)
3. Third attempt fails → wait 2000ms (1000ms × 2.0)
4. Max backoff capped at `max_backoff_ms`

**Recommendations:**
- **Transient failures** (network glitches): max_attempts 3
- **Deterministic failures** (invalid input): max_attempts 1
- **External APIs**: max_attempts 3, max_backoff_ms 10000

---

## Health Checks

### Configuration

```yaml
health_check:
  enabled: true
  path: "/health"      # HTTP path (if applicable)
  interval_sec: 60     # Check every 60 seconds
```

### Behavior

- **stdio:** Checks if subprocess is running and responsive
- **SSE:** Sends HTTP GET to health endpoint
- **Failure detection:** If health check fails, server is marked unhealthy
- **Auto-reconnect:** Gateway attempts to reconnect on next health check cycle

### Monitoring Health

```bash
# Check overall MCP health
curl http://localhost:8080/admin/mcp/status

# Response includes per-server health:
# {
#   "servers": {
#     "mcp_by_dojo": {
#       "state": "connected",
#       "tool_count": 14,
#       "last_health_check": "2026-02-13T11:30:00Z"
#     }
#   },
#   "healthy": true
# }
```

---

## Environment Variables

### Variable Expansion

Gateway config supports `${VAR_NAME}` syntax for environment variable expansion:

```yaml
transport:
  command: "${MCP_SERVER_BINARY:-/opt/mcp-servers/default}"
  env:
    API_KEY: "${MCP_API_KEY}"
    LOG_LEVEL: "${LOG_LEVEL:-info}"
```

**Syntax:**
- `${VAR}` — Required variable (fails if not set)
- `${VAR:-default}` — Optional variable with default value

### Common Variables

```bash
# MCP server binary paths
export MCP_BY_DOJO_BINARY=/opt/mcp-servers/mcp-by-dojo-genesis
export CUSTOM_MCP_BINARY=/usr/local/bin/custom-server

# API keys and tokens
export COMPOSIO_API_KEY=your-composio-key
export CUSTOM_SERVER_TOKEN=your-token

# Configuration paths
export MCP_CONFIG_PATH=/etc/gateway/gateway-config.yaml
```

---

## Observability

### OTEL Integration

All MCP tool invocations emit OTEL spans with these attributes:

```yaml
# Span attributes
mcp.server_id: "mcp_by_dojo"
mcp.server_display_name: "MCP By Dojo Genesis"
mcp.tool_name: "search_wisdom"  # Original tool name (no namespace)
mcp.tool_namespaced: "mcp_by_dojo:search_wisdom"  # Fully qualified name
mcp.tool_latency_ms: 1234
mcp.tool_input_size_bytes: 512
mcp.tool_output_size_bytes: 2048
```

### Viewing Traces

1. **Langfuse UI:** http://localhost:3000 (if running with docker-compose)
2. **OTEL Collector logs:** Check stdout for span exports
3. **Prometheus metrics:** http://localhost:8889/metrics

### Configuration

```yaml
observability:
  enabled: true
  trace_provider: "otel"
  attributes:
    service.name: "agentic-gateway"
    mcp.host.enabled: "true"
  log_level: "info"
  tool_span_sample_rate: 1.0  # 1.0 = trace all, 0.1 = trace 10%
```

---

## Troubleshooting

### Issue: MCP server not connecting

**Symptoms:** `GET /admin/mcp/status` shows `"state": "disconnected"`

**Solutions:**
1. Check binary exists and is executable:
   ```bash
   ls -la /path/to/mcp-server
   chmod +x /path/to/mcp-server
   ```

2. Test binary manually:
   ```bash
   /path/to/mcp-server
   # Should not exit immediately; should wait for stdin
   ```

3. Check logs for startup errors:
   ```bash
   docker-compose logs gateway | grep "MCP"
   ```

4. Verify environment variables are set:
   ```bash
   docker-compose exec gateway env | grep MCP
   ```

### Issue: Tools not registering

**Symptoms:** `tool_count: 0` in status response

**Solutions:**
1. Check allowlist/blocklist configuration
2. Verify MCP server implements `list_tools` correctly
3. Check for namespace collisions (duplicate tool names)
4. Review gateway logs for registration errors

### Issue: Tool invocations timing out

**Symptoms:** Tool calls fail with timeout error

**Solutions:**
1. Increase `tool_default` timeout
2. Check MCP server is responsive (not blocked)
3. Monitor OTEL spans for actual latency
4. Consider long-running operations need async pattern

### Issue: Health checks failing

**Symptoms:** Server repeatedly disconnects and reconnects

**Solutions:**
1. Increase `health_check.interval_sec`
2. Check server implements health check endpoint
3. Review server logs for health check errors
4. Disable health checks if not supported:
   ```yaml
   health_check:
     enabled: false
   ```

---

## Example Configurations

### Python-based MCP Server

```yaml
- id: "python_tools"
  display_name: "Python Tool Server"
  namespace_prefix: "python"

  transport:
    type: "stdio"
    command: "python"
    args: ["-m", "my_mcp_server"]
    env:
      PYTHONPATH: "/app/mcp-servers/python"
      LOG_LEVEL: "info"

  timeouts:
    startup: 15  # Python startup can be slow
    tool_default: 60
    health_check: 5

  health_check:
    enabled: true
    interval_sec: 120
```

### Node.js MCP Server

```yaml
- id: "node_tools"
  display_name: "Node.js Tool Server"
  namespace_prefix: "node"

  transport:
    type: "stdio"
    command: "node"
    args: ["/app/mcp-servers/node/server.js"]
    env:
      NODE_ENV: "production"

  timeouts:
    startup: 10
    tool_default: 30
    health_check: 5
```

### Remote HTTP MCP Server

```yaml
- id: "remote_api"
  display_name: "Remote API Tools"
  namespace_prefix: "api"

  transport:
    type: "sse"
    url: "https://mcp.example.com/sse"
    headers:
      Authorization: "Bearer ${REMOTE_API_TOKEN}"
      X-Gateway-ID: "agentic-gateway-prod"

  timeouts:
    startup: 30  # Remote connection takes longer
    tool_default: 60
    health_check: 10

  health_check:
    enabled: true
    interval_sec: 180  # Less frequent for remote
```

---

## Next Steps

- [Composio Setup Guide](./composio-setup.md) — Configure Composio integration
- [Architecture Overview](../ARCHITECTURE.md) — Understand MCP integration architecture
- [Gateway MCP Contract](../docs/v0.2.0/gateway-mcp-contract.md) — Full specification

---

**Questions or issues?** Open an issue on GitHub or consult the MCP specification.
