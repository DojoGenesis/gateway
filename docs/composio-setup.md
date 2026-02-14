# Composio Integration Setup Guide

**Version:** v0.2.0 (Phase 3)
**Status:** Documented for future use (no live connection in Phase 3)
**Audience:** Developers integrating Composio tools
**Purpose:** How to enable and configure Composio as an MCP server

---

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Configuration](#configuration)
4. [Authentication](#authentication)
5. [Tool Filtering](#tool-filtering)
6. [Testing the Connection](#testing-the-connection)
7. [Troubleshooting](#troubleshooting)
8. [Security Considerations](#security-considerations)

---

## Overview

[Composio](https://composio.dev) is a platform that provides 100+ tool integrations (GitHub, Slack, Jira, Linear, etc.) via a unified API. The AgenticGateway can connect to Composio as an MCP server using **SSE (Server-Sent Events)** transport.

**Key Benefits:**
- Access 100+ pre-built tool integrations
- No need to implement individual API clients
- Unified authentication and error handling
- Tools auto-register with `composio:` namespace prefix

**Architecture:**

```
┌─────────────────┐
│ AgenticGateway  │
│                 │
│  ┌───────────┐  │    SSE/HTTP      ┌──────────────┐
│  │ MCP Host  ├──┼─────────────────►│   Composio   │
│  │ Manager   │  │                   │   Platform   │
│  └───────────┘  │                   └──────────────┘
│                 │                           │
│  Tools:         │                           │
│  - composio:    │                           ▼
│    github_create_issue               ┌─────────────┐
│  - composio:                         │  External   │
│    slack_send_message                │  Services   │
│  - composio:                         │ (GitHub,    │
│    linear_create_ticket              │  Slack,     │
└─────────────────┘                    │  Jira, ...) │
                                       └─────────────┘
```

---

## Prerequisites

### 1. Composio Account

Sign up for a Composio account:
- Visit [https://app.composio.dev](https://app.composio.dev)
- Create an account (free tier available)
- Navigate to **Settings** → **API Keys**

### 2. Composio API Key

Generate an API key:
1. Click **"Generate New API Key"**
2. Copy the key (starts with `compsk_...`)
3. Store securely (you won't see it again)

### 3. Connected Apps (Optional)

If you plan to use specific integrations (GitHub, Slack, etc.):
1. Go to **Integrations** in Composio dashboard
2. Click **Connect** for each service
3. Complete OAuth flow
4. Tools for connected apps will appear automatically

---

## Configuration

### Step 1: Environment Variables

Set the Composio API key as an environment variable:

```bash
# Production
export COMPOSIO_API_KEY="compsk_your_api_key_here"

# Docker Compose
echo "COMPOSIO_API_KEY=compsk_your_api_key_here" >> .env
```

### Step 2: Update gateway-config.yaml

Add Composio to the `servers` section:

```yaml
version: "1.0"

mcp:
  global:
    default_tool_timeout: 30
    reconnect:
      max_retries: 3
      initial_backoff_ms: 1000
      max_backoff_ms: 30000
      backoff_multiplier: 2.0
    health_check_interval: 60
    response_buffer_size: 1048576

  servers:
    # ... other servers (e.g., mcp_by_dojo) ...

    # Composio Integration
    - id: "composio"
      display_name: "Composio Integration"
      namespace_prefix: "composio"

      # SSE transport for Composio
      transport:
        type: "sse"
        # Composio MCP endpoint (verify URL with Composio docs)
        url: "${COMPOSIO_SSE_URL:-https://mcp.composio.dev/sse}"

        # Authentication headers
        headers:
          Authorization: "Bearer ${COMPOSIO_API_KEY}"
          X-Client-ID: "agentic-gateway-prod"
          Content-Type: "application/json"

      # Tool filtering (optional)
      tools:
        # Allowlist: only register specific tools
        allowlist: []  # Empty = allow all

        # Blocklist: exclude dangerous operations
        blocklist:
          # Block admin/destructive operations
          - "composio:admin:delete_org"
          - "composio:admin:modify_config"
          # Block specific integrations if needed
          # - "composio:github:delete_repo"

      # Timeouts (remote service, longer than stdio)
      timeouts:
        startup: 15           # SSE connection establishment
        tool_default: 30      # Per-tool timeout
        health_check: 10      # Health check timeout

      # Health checks (less frequent for remote)
      health_check:
        enabled: true
        path: "/health"       # Composio health endpoint
        interval_sec: 120     # Check every 2 minutes

      # Retry policy
      retry_policy:
        max_attempts: 3
        backoff_multiplier: 1.5
        max_backoff_ms: 10000

  observability:
    enabled: true
    trace_provider: "otel"
    attributes:
      service.name: "agentic-gateway"
      mcp.host.enabled: "true"
    log_level: "info"
    tool_span_sample_rate: 1.0
```

### Step 3: Restart Gateway

```bash
# Docker Compose
docker-compose restart gateway

# Local development
./agentic-gateway
```

---

## Authentication

### API Key Management

**Best practices:**
- **Never commit API keys** to version control
- Use environment variables or secrets management
- Rotate keys regularly (Composio dashboard → API Keys)
- Use different keys for dev/staging/prod

### Docker Compose

Add to `.env` file (gitignored):

```bash
COMPOSIO_API_KEY=compsk_your_api_key_here
COMPOSIO_SSE_URL=https://mcp.composio.dev/sse
```

Reference in `docker-compose.yaml`:

```yaml
services:
  gateway:
    environment:
      - COMPOSIO_API_KEY=${COMPOSIO_API_KEY}
      - COMPOSIO_SSE_URL=${COMPOSIO_SSE_URL}
```

### Kubernetes (Production)

Use Kubernetes secrets:

```bash
kubectl create secret generic composio-secret \
  --from-literal=api-key=compsk_your_api_key_here
```

Mount in deployment:

```yaml
env:
  - name: COMPOSIO_API_KEY
    valueFrom:
      secretKeyRef:
        name: composio-secret
        key: api-key
```

---

## Tool Filtering

### Allow Specific Integrations

Only register GitHub and Slack tools:

```yaml
tools:
  allowlist:
    - "composio:github:*"    # All GitHub tools
    - "composio:slack:*"     # All Slack tools
  blocklist: []
```

### Block Dangerous Operations

Allow all tools except destructive ones:

```yaml
tools:
  allowlist: []  # Allow all
  blocklist:
    # Block all delete operations
    - "composio:*:delete_*"
    # Block specific dangerous tools
    - "composio:github:delete_repo"
    - "composio:slack:delete_workspace"
    - "composio:linear:delete_project"
```

### Recommended Filters for Production

```yaml
tools:
  allowlist:
    # Read-only operations
    - "composio:github:get_*"
    - "composio:github:list_*"
    # Safe write operations
    - "composio:github:create_issue"
    - "composio:github:create_pr_comment"
    - "composio:slack:send_message"
    - "composio:linear:create_ticket"

  blocklist:
    # Admin operations
    - "composio:*:admin:*"
    # Destructive operations
    - "composio:*:delete_*"
    - "composio:*:remove_*"
    # Dangerous GitHub operations
    - "composio:github:delete_repo"
    - "composio:github:force_push"
```

---

## Testing the Connection

### 1. Verify Server Registration

```bash
curl http://localhost:8080/admin/mcp/status
```

Expected response:

```json
{
  "servers": {
    "composio": {
      "server_id": "composio",
      "display_name": "Composio Integration",
      "state": "connected",
      "tool_count": 25,  // or more, depending on connected apps
      "last_health_check": "2026-02-13T12:00:00Z"
    }
  },
  "total_servers": 1,
  "total_tools": 25,
  "healthy": true
}
```

### 2. List Composio Tools

```bash
curl http://localhost:8080/v1/gateway/tools | jq '.tools[] | select(.name | startswith("composio:"))'
```

Expected output:

```json
{
  "name": "composio:github:create_issue",
  "description": "Create a new GitHub issue",
  "parameters": { ... }
}
```

### 3. Invoke a Tool

```bash
curl -X POST http://localhost:8080/v1/tools/composio:github:list_repos/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "owner": "your-github-username"
  }'
```

### 4. Check OTEL Traces

Visit Langfuse UI (http://localhost:3000) and look for spans with:
- `mcp.server_id: "composio"`
- `mcp.tool_name: "github:create_issue"`

---

## Troubleshooting

### Issue: Connection Failed

**Symptoms:** `"state": "disconnected"` in status

**Solutions:**
1. **Verify API key:**
   ```bash
   echo $COMPOSIO_API_KEY
   # Should output: compsk_...
   ```

2. **Test API key manually:**
   ```bash
   curl -H "Authorization: Bearer $COMPOSIO_API_KEY" \
        https://api.composio.dev/v1/health
   ```

3. **Check SSE URL:**
   ```bash
   # Verify the URL is correct (check Composio docs)
   echo $COMPOSIO_SSE_URL
   ```

4. **Review gateway logs:**
   ```bash
   docker-compose logs gateway | grep composio
   ```

### Issue: No Tools Registered

**Symptoms:** `"tool_count": 0`

**Solutions:**
1. **Connect apps in Composio dashboard:**
   - Go to https://app.composio.dev/integrations
   - Click "Connect" for GitHub, Slack, etc.
   - Complete OAuth flow

2. **Check allowlist/blocklist:**
   - Ensure allowlist isn't too restrictive
   - Remove blocklist temporarily to test

3. **Verify Composio account status:**
   - Check if account is active
   - Verify API key hasn't been revoked

### Issue: Tool Invocations Failing

**Symptoms:** Tool calls return errors

**Solutions:**
1. **Check app authorization:**
   - Go to Composio dashboard → Integrations
   - Re-authorize apps if needed

2. **Verify tool parameters:**
   - Composio tools may have required parameters
   - Check tool schema in `/v1/gateway/tools`

3. **Increase timeout:**
   ```yaml
   timeouts:
     tool_default: 60  # Increase for slow external APIs
   ```

4. **Check OTEL traces for error details:**
   - Review span attributes for error messages
   - Look for HTTP status codes

### Issue: Rate Limiting

**Symptoms:** Tools failing with 429 errors

**Solutions:**
1. **Check Composio plan limits**
2. **Implement request throttling** (future feature)
3. **Upgrade Composio plan** if needed
4. **Add retry policy with longer backoff:**
   ```yaml
   retry_policy:
     max_attempts: 5
     backoff_multiplier: 2.0
     max_backoff_ms: 30000
   ```

---

## Security Considerations

### API Key Protection

✅ **Do:**
- Store API keys in environment variables
- Use secrets management (Vault, AWS Secrets Manager, etc.)
- Rotate keys regularly
- Use different keys per environment (dev/staging/prod)

❌ **Don't:**
- Commit API keys to Git
- Share keys across teams
- Use production keys in development
- Log API keys in plaintext

### Tool Filtering

**Recommended approach:**
1. **Start with allowlist** (default deny)
2. **Only add tools you need**
3. **Block all admin/delete operations**
4. **Review tool permissions regularly**

### Network Security

- **Use HTTPS only** for Composio connections
- **Verify SSL certificates** (don't disable cert validation)
- **Restrict network access** (firewall rules for gateway)
- **Monitor outbound connections**

### Audit Logging

All Composio tool invocations are traced via OTEL:

```bash
# Query Langfuse for Composio tool usage
# Filter by: mcp.server_id = "composio"
```

Monitor for:
- Unusual tool usage patterns
- Failed authentication attempts
- Destructive operations
- High-frequency calls (potential abuse)

---

## Example Use Cases

### GitHub PR Automation

```yaml
# Allow only GitHub PR operations
tools:
  allowlist:
    - "composio:github:create_pr"
    - "composio:github:create_pr_comment"
    - "composio:github:list_prs"
    - "composio:github:get_pr"
```

### Slack Notifications

```yaml
# Allow only Slack messaging
tools:
  allowlist:
    - "composio:slack:send_message"
    - "composio:slack:list_channels"
```

### Linear Issue Creation

```yaml
# Allow Linear ticket creation only
tools:
  allowlist:
    - "composio:linear:create_ticket"
    - "composio:linear:list_projects"
    - "composio:linear:list_teams"
```

---

## Next Steps

- [MCP Configuration Guide](./mcp-configuration.md) — General MCP server configuration
- [Architecture Overview](../ARCHITECTURE.md) — Understand MCP integration
- [Composio Documentation](https://docs.composio.dev) — Official Composio docs

---

**Note:** This setup guide is based on Composio's MCP integration (as of v0.2.0). If Composio's API changes, refer to their official documentation for the latest configuration.

**Questions or issues?** Open an issue on GitHub or contact Composio support.
