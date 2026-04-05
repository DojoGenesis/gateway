# Agentic Gateway Integration Guide

How to integrate the Agentic Gateway into a real-world frontend application. Based on the HTMLCraft Studio integration (Tauri + Svelte), but the patterns apply to any frontend framework.

## Architecture Overview

```
┌─────────────────────┐     HTTP/SSE      ┌──────────────────────┐
│   Frontend App      │ ◄──────────────► │   Agentic Gateway    │
│   (any framework)   │                   │   (Go, port 8080)    │
│                     │                   │                      │
│  - Agent creation   │   POST /agents    │  - LLM routing       │
│  - SSE streaming    │   POST /chat      │  - Tool execution    │
│  - Health polling   │   GET  /health    │  - MCP servers       │
│  - Tool indicators  │   GET  /providers │  - Memory/Garden     │
└─────────────────────┘                   └──────────────────────┘
```

## 1. Starting the Gateway

### As a sidecar (desktop apps)

Ship the `agentic-gateway` binary with your app. On launch:

```rust
// Key: enrich PATH so MCP subprocesses can find node, python, etc.
let enriched_path = format!("/opt/homebrew/bin:/usr/local/bin:{}", inherited_path);

// Key: set ALLOWED_ORIGINS for CORS
let cors_origins = "http://localhost:5173,https://tauri.localhost";

Command::new(&gateway_binary)
    .env("PATH", &enriched_path)
    .env("PORT", "8080")
    .env("ALLOWED_ORIGINS", &cors_origins)
    .env("MCP_CONFIG_PATH", &config_path)
    .env("MEMORY_DB_PATH", &db_path)
    .spawn()
```

### As a standalone service

```bash
PORT=8080 \
ALLOWED_ORIGINS="http://localhost:3000,https://your-app.com" \
MCP_CONFIG_PATH=./gateway-config.yaml \
./agentic-gateway
```

### Essential environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No (default 8080) | HTTP listen port |
| `ALLOWED_ORIGINS` | Yes (for browser clients) | Comma-separated CORS origins |
| `MCP_CONFIG_PATH` | No | Path to MCP server config YAML |
| `MEMORY_DB_PATH` | No | SQLite path for persistent memory |
| `AUTH_DB_DIR` | No | Directory for auth database |

## 2. CORS Configuration

**Critical for browser-based clients.** The gateway uses `gin-contrib/cors` and reads `ALLOWED_ORIGINS` as a comma-separated list.

Rules:
- Only `http://` and `https://` origins are valid (no `tauri://`)
- For Tauri v2 apps, use `https://tauri.localhost`
- For development, include your Vite/webpack dev server origin
- Never use `*` in production

```bash
ALLOWED_ORIGINS="http://localhost:5173,https://tauri.localhost,https://your-app.com"
```

## 3. Health Monitoring

### Direct HTTP poll (recommended for initial status)

```typescript
async function checkGateway(): Promise<boolean> {
  try {
    const resp = await fetch('http://127.0.0.1:8080/health', {
      signal: AbortSignal.timeout(3000)
    });
    if (resp.ok) {
      const data = await resp.json();
      return data.status === 'healthy';
    }
  } catch {}
  return false;
}
```

Response shape:
```json
{
  "status": "healthy",
  "version": "1.1.0",
  "providers": { "ollama": "healthy" },
  "dependencies": {
    "memory_store": "healthy",
    "orchestration_engine": "healthy",
    "tool_registry": "healthy"
  },
  "uptime_seconds": 42,
  "requests_processed": 15
}
```

### MCP server status

```
GET /admin/mcp/status
```

Returns a map of connected MCP servers with tool counts.

## 4. Agent Lifecycle

### Create an agent

```typescript
const resp = await fetch('http://127.0.0.1:8080/v1/gateway/agents', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    workspace_root: '/path/to/project'  // required
  })
});
const { agent_id } = await resp.json();
```

The gateway initializes the agent with a disposition (personality) from the workspace. Extra fields like `name`, `provider`, `system_prompt` are accepted but the gateway primarily uses `workspace_root` for disposition resolution.

### Verify an agent is alive

```typescript
const resp = await fetch(`http://127.0.0.1:8080/v1/gateway/agents/${agentId}`);
const alive = resp.ok; // 200 = alive, 404 = dead
```

Agents are in-memory — they don't survive gateway restarts. Your app should:
1. Store the `agent_id` locally (SQLite, localStorage)
2. On app launch, verify the stored agent
3. If dead (404), create a new one

## 5. Chat — Non-Streaming

```typescript
const resp = await fetch(`http://127.0.0.1:8080/v1/gateway/agents/${agentId}/chat`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    message: 'Hello',
    stream: false
  })
});
const data = await resp.json();
console.log(data.response);    // The LLM's text response
console.log(data.tool_calls);  // Array of tools that were invoked
```

Good for simple request/response. Timeout should be generous (180s+) since Ollama inference can be slow.

## 6. Chat — SSE Streaming (Recommended)

SSE streaming gives real-time feedback: thinking indicators, tool invocations, and incremental text.

```typescript
async function streamChat(agentId: string, message: string): Promise<string> {
  const resp = await fetch(`${GW}/v1/gateway/agents/${agentId}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message, stream: true })
  });

  if (!resp.ok || !resp.headers.get('content-type')?.includes('text/event-stream')) {
    // Fall back to non-streaming
    return '';
  }

  const reader = resp.body!.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let fullText = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() ?? '';

    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      const payload = line.slice(6).trim();
      if (payload === '[DONE]') return fullText;

      const evt = JSON.parse(payload);
      switch (evt.type) {
        case 'thinking':
          showThinkingIndicator(evt.data.message);
          break;
        case 'tool_invoked':
          showToolBadge(evt.data.tool, 'running');
          break;
        case 'tool_completed':
          showToolBadge(evt.data.tool, 'done');
          break;
        case 'response_chunk':
          fullText += evt.data.content;
          updateUI(fullText);
          break;
        case 'complete':
          hideThinkingIndicator();
          break;
        case 'error':
          showError(evt.data.error);
          break;
      }
    }
  }
  return fullText;
}
```

### SSE Event Types

| Event | Data | When |
|-------|------|------|
| `thinking` | `{ message }` | LLM is processing |
| `tool_invoked` | `{ tool, arguments }` | Tool execution started |
| `tool_completed` | `{ tool, result, duration_ms }` | Tool execution finished |
| `response_chunk` | `{ content }` | Incremental text content |
| `patch_intent` | `{ operation, section_id, content }` | Document edit proposal |
| `complete` | `{ usage }` | Response finished |
| `error` | `{ error, error_code }` | Error occurred |

### Critical: rAF-Batched UI Updates

Streaming can fire 10+ `response_chunk` events per second. If each triggers a re-render, the UI freezes. **Batch updates with requestAnimationFrame:**

```typescript
let chunkBuffer = '';
let rafPending = false;

function onChunk(content: string) {
  chunkBuffer += content;
  if (!rafPending) {
    rafPending = true;
    requestAnimationFrame(() => {
      // Update UI with accumulated chunks
      appendToMessage(chunkBuffer);
      chunkBuffer = '';
      rafPending = false;
    });
  }
}
```

## 7. Pitfalls and Lessons Learned

### Never use `afterUpdate` + `tick()` together (Svelte)

```typescript
// BAD — infinite loop that freezes the UI
afterUpdate(async () => {
  await tick();
  scrollToBottom();
});

// GOOD — reactive block that only fires on actual changes
$: if (messages.length !== lastCount) {
  lastCount = messages.length;
  requestAnimationFrame(scrollToBottom);
}
```

### Lazy-load all Tauri IPC imports

`@tauri-apps/api/core` evaluates `window.__TAURI_INTERNALS__` at module load time. If imported statically in any module that's part of a barrel export, it will crash or hang in browser mode.

```typescript
// BAD — top-level import loads Tauri SDK immediately
import { invoke } from '@tauri-apps/api/core';

// GOOD — lazy import only when needed
const tauri = () => import('$lib/api/tauri.js');
async function doSomething() {
  const { invoke } = await tauri();
  return invoke('my_command');
}
```

### Avoid barrel re-exports for stores

```typescript
// BAD — importing anything from index.js loads ALL stores
import { gatewayStatus } from '$lib/stores/index.js';

// GOOD — import directly from the specific store
import { gatewayStatus } from '$lib/stores/gatewayStatus.js';
```

### Self-contained components over shared stores

For complex components that manage async state (chat panels, streaming), keep state local to the component instead of external stores. Cross-store reactive subscriptions can create update cascades that freeze the UI.

### Gateway CORS origins must be HTTP(S)

The `gin-contrib/cors` library rejects non-HTTP origins like `tauri://localhost`. For Tauri v2, use `https://tauri.localhost` instead.

### Sidecar PATH inheritance

When spawning the gateway as a sidecar, the child process may not inherit PATH entries for tools like `node` or `python3`. Explicitly prepend common tool directories:

```rust
let enriched_path = format!("/opt/homebrew/bin:/usr/local/bin:{}", std::env::var("PATH").unwrap_or_default());
cmd.env("PATH", &enriched_path);
```

## 8. MCP Configuration

The gateway loads MCP servers from a YAML config file:

```yaml
version: "1.0"
mcp:
  servers:
    - id: "my-tools"
      display_name: "My Tools"
      namespace_prefix: "myapp"
      transport:
        type: "stdio"
        command: "node"
        args: ["path/to/mcp-server/index.js"]
      tools:
        allowlist: []
        blocklist: []
```

Tools are automatically namespaced: `myapp:tool_name`. The gateway discovers tools at startup and makes them available to agents.

## 9. Provider Configuration

Available providers are detected at startup. Ollama is always available when running locally. Cloud providers require API keys:

```bash
# Set via environment
ANTHROPIC_API_KEY=sk-... ./agentic-gateway

# Or via the settings API
curl -X POST http://localhost:8080/v1/settings/providers \
  -H "Content-Type: application/json" \
  -d '{"provider": "anthropic", "api_key": "sk-..."}'
```

List available providers:
```
GET /v1/providers
```
