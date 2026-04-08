# Deployment Guide

**Version:** v0.2.0 (Phase 3: MCP Server Wiring)

---

## Quick Start

### Option 1: Docker Compose (Recommended)

```bash
# 1. Clone the repository
git clone https://github.com/DojoGenesis/gateway.git
cd AgenticGatewayByDojoGenesis

# 2. Set environment variables (optional)
export LANGFUSE_NEXTAUTH_SECRET="your-random-secret-min-32-chars"
export LANGFUSE_SALT="your-random-salt-min-32-chars"

# 3. Start the stack
docker-compose up -d

# 4. Verify all services are running
docker-compose ps

# 5. Check MCP server status
curl http://localhost:8080/admin/mcp/status

# 6. View observability in Langfuse
open http://localhost:3000
```

### Option 2: Local Development

```bash
# 1. Install MCPByDojoGenesis binary (if not already installed)
go install github.com/TresPies-source/MCPByDojoGenesis@latest

# 2. Create config directory and copy config
mkdir -p config
cp gateway-config.yaml config/mcp_servers.yaml

# 3. Set environment variables
export MCP_CONFIG_PATH=config/mcp_servers.yaml
export MCP_BY_DOJO_BINARY=$(which mcp-by-dojo-genesis)
export OTEL_ENABLED=false  # or configure OTEL endpoint

# 4. Run the gateway
go run main.go

# 5. Verify MCP status
curl http://localhost:8080/admin/mcp/status
```

---

## Configuration

### MCP Server Configuration

The gateway reads MCP server configuration from:
- **Docker:** `/etc/gateway/gateway-config.yaml` (mounted via volume)
- **Local:** `config/mcp_servers.yaml` (default) or path specified in `MCP_CONFIG_PATH`

See [docs/mcp-configuration.md](./docs/mcp-configuration.md) for detailed configuration options.

### Environment Variables

**Required for Docker Compose:**
- `MCP_BY_DOJO_BINARY` — Path to MCPByDojoGenesis binary (set automatically in docker-compose)
- `OTEL_EXPORTER_OTLP_ENDPOINT` — OTEL collector endpoint (default: `http://otel-collector:4318`)

**Optional:**
- `MCP_CONFIG_PATH` — Path to MCP configuration file (default: `config/mcp_servers.yaml`)
- `MCP_LOG_LEVEL` — Log level for MCP servers (default: `info`)
- `LANGFUSE_NEXTAUTH_SECRET` — Langfuse authentication secret (required for production)
- `LANGFUSE_SALT` — Langfuse encryption salt (required for production)
- `COMPOSIO_API_KEY` — Composio API key (if enabling Composio integration)

---

## Services

The docker-compose stack includes:

| Service | Port | Description |
|---------|------|-------------|
| **gateway** | 8080 | Main AgenticGateway API |
| **mcp-by-dojo** | N/A | Binary provider (exits after copying binary) |
| **otel-collector** | 4318 | OTLP HTTP receiver for traces |
| **langfuse** | 3000 | Observability UI and trace storage |
| **postgres** | 5432 | Langfuse database |

---

## Health Checks

### Gateway Health

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "version": "v0.2.0"
}
```

### MCP Server Status

```bash
curl http://localhost:8080/admin/mcp/status
```

Expected response:
```json
{
  "servers": {
    "mcp_by_dojo": {
      "server_id": "mcp_by_dojo",
      "display_name": "MCP By Dojo Genesis",
      "state": "connected",
      "tool_count": 14,
      "last_health_check": "2026-02-13T12:00:00Z"
    }
  },
  "total_servers": 1,
  "total_tools": 14,
  "healthy": true
}
```

### OTEL Collector Health

```bash
curl http://localhost:13133/
```

### Langfuse Health

```bash
curl http://localhost:3000/api/health
```

---

## Troubleshooting

### MCP Server Not Connecting

**Symptom:** `"state": "disconnected"` in `/admin/mcp/status`

**Solutions:**

1. **Check binary exists:**
   ```bash
   docker-compose exec gateway ls -la /opt/mcp-servers/
   ```

2. **Check logs:**
   ```bash
   docker-compose logs gateway | grep MCP
   ```

3. **Verify environment variables:**
   ```bash
   docker-compose exec gateway env | grep MCP
   ```

4. **Restart gateway:**
   ```bash
   docker-compose restart gateway
   ```

### No Tools Registered

**Symptom:** `"tool_count": 0`

**Solutions:**

1. **Check allowlist/blocklist** in `gateway-config.yaml`
2. **Verify MCP server implements `list_tools`** correctly
3. **Check gateway logs** for registration errors
4. **Increase startup timeout** in config

### OTEL Traces Not Appearing

**Symptom:** No traces in Langfuse UI

**Solutions:**

1. **Check OTEL collector is running:**
   ```bash
   docker-compose logs otel-collector
   ```

2. **Verify OTEL endpoint:**
   ```bash
   docker-compose exec gateway env | grep OTEL
   ```

3. **Check sampling rate** in `gateway-config.yaml`:
   ```yaml
   observability:
     tool_span_sample_rate: 1.0  # 1.0 = trace all
   ```

4. **Test OTEL endpoint:**
   ```bash
   curl http://localhost:4318/v1/traces -X POST \
     -H "Content-Type: application/json" \
     -d '{"resourceSpans": []}'
   ```

### Langfuse Not Starting

**Symptom:** Langfuse container exits or fails to start

**Solutions:**

1. **Set required environment variables:**
   ```bash
   export LANGFUSE_NEXTAUTH_SECRET="random-32-char-minimum-secret"
   export LANGFUSE_SALT="random-32-char-minimum-salt"
   docker-compose up -d
   ```

2. **Check PostgreSQL is healthy:**
   ```bash
   docker-compose logs postgres
   ```

3. **Verify database connection:**
   ```bash
   docker-compose exec langfuse env | grep DATABASE_URL
   ```

---

## Stopping the Stack

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (⚠️ deletes all data)
docker-compose down -v
```

---

## Production Considerations

### Security

1. **Set strong secrets:**
   ```bash
   export LANGFUSE_NEXTAUTH_SECRET=$(openssl rand -base64 32)
   export LANGFUSE_SALT=$(openssl rand -base64 32)
   ```

2. **Use environment-specific configs:**
   - Development: `gateway-config.dev.yaml`
   - Staging: `gateway-config.staging.yaml`
   - Production: `gateway-config.prod.yaml`

3. **Enable authentication** on admin endpoints (future enhancement)

4. **Use secrets management:**
   - Kubernetes: Use `kubectl create secret`
   - AWS: Use AWS Secrets Manager
   - Docker Swarm: Use `docker secret create`

### Scaling

1. **Gateway horizontal scaling:**
   ```yaml
   gateway:
     deploy:
       replicas: 3
   ```

2. **PostgreSQL replication** for Langfuse

3. **OTEL collector clustering** for high throughput

### Monitoring

1. **Prometheus metrics:**
   ```bash
   curl http://localhost:8080/admin/metrics/prometheus
   ```

2. **Langfuse analytics:**
   - Go to http://localhost:3000
   - Filter by `mcp.server_id`
   - Analyze tool latency, error rates

3. **Log aggregation:**
   - Ship logs to ELK, Loki, or CloudWatch
   - Configure structured logging

---

## Next Steps

- [MCP Configuration Guide](./docs/mcp-configuration.md) — Add custom MCP servers
- [Composio Setup](./docs/composio-setup.md) — Enable Composio integration
- [Architecture Overview](./ARCHITECTURE.md) — Understand system design

---

**Questions?** Open an issue on GitHub or consult the documentation.
