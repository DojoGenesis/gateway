---
name: mcp-cloudflare-builder
model: sonnet
description: "Builds production-ready remote MCP servers on Cloudflare Workers with typed tools, OAuth authentication, and Wrangler deployment. Use when: 'build MCP server on Cloudflare', 'remote MCP', 'deploy MCP to Workers', 'MCP OAuth authentication', 'expose tools via MCP'."
category: skill-forge
triggers:
  - "build MCP server on Cloudflare"
  - "remote MCP"
  - "deploy MCP to Workers"
  - "MCP OAuth authentication"
  - "expose tools via MCP"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: server_name
    type: string
    description: Name for the MCP server (used in wrangler.toml and package.json)
    required: true
  - name: auth_mode
    type: string
    description: Authentication mode — "public" (no auth) or "oauth" (GitHub, Google, etc.)
    required: false
outputs:
  - name: mcp_server
    type: ref
    format: directory
    description: Cloudflare Worker project with typed tool definitions, entry point, and wrangler.toml ready for deployment
---

# MCP Cloudflare Builder

Scaffolds and deploys production-ready Model Context Protocol servers on Cloudflare Workers.

## Description

Creates remote MCP servers using the `McpAgent` Durable Object pattern on Cloudflare Workers. Covers tool definition with Zod validation, public vs. OAuth-protected modes, local testing with MCP Inspector, Wrangler deployment, and client connection. Distinct from `mcp-builder` (which covers general MCP server concepts) and `mcp-server-builder` (which scaffolds from OpenAPI specs) — this skill is Cloudflare-specific and deployment-complete.

## When to Use

- User wants a remote (not stdio) MCP server
- User needs to expose tools to Claude Desktop or other MCP clients over HTTPS
- User wants MCP authentication via OAuth (GitHub, Google, Auth0, etc.)
- User is deploying on Cloudflare Workers specifically

## Workflow

### Step 1: Scaffold the Project

**Public server (no auth):**
```bash
npm create cloudflare@latest -- my-mcp-server \
  --template=cloudflare/ai/demos/remote-mcp-authless
cd my-mcp-server && npm start
```
Server runs at `http://localhost:8788/mcp`.

**OAuth-protected server:**
```bash
npm create cloudflare@latest -- my-mcp-server \
  --template=cloudflare/ai/demos/remote-mcp-github-oauth
cd my-mcp-server
```

### Step 2: Define Tools

Tools are functions MCP clients can call. Define them in `McpAgent.init()`:

```typescript
import { McpAgent } from "agents/mcp";
import { z } from "zod";

export class MyMCP extends McpAgent {
  server = new Server({ name: "my-mcp", version: "1.0.0" });

  async init() {
    this.server.tool(
      "add",
      { a: z.number(), b: z.number() },
      async ({ a, b }) => ({
        content: [{ type: "text", text: String(a + b) }],
      })
    );

    this.server.tool(
      "get_weather",
      { city: z.string() },
      async ({ city }) => {
        const response = await fetch(`https://api.weather.com/${city}`);
        const data = await response.json();
        return { content: [{ type: "text", text: JSON.stringify(data) }] };
      }
    );
  }
}
```

### Step 3: Configure Entry Point

```typescript
// src/index.ts
import { MyMCP } from "./mcp";

export default {
  fetch(request: Request, env: Env, ctx: ExecutionContext) {
    const url = new URL(request.url);
    if (url.pathname === "/mcp") {
      return MyMCP.serveSSE("/mcp").fetch(request, env, ctx);
    }
    return new Response("MCP Server", { status: 200 });
  },
};

export { MyMCP };
```

### Step 4: Test Locally

```bash
npm start
# In another terminal:
npx @modelcontextprotocol/inspector@latest
# Open http://localhost:5173, connect to http://localhost:8788/mcp
```

### Step 5: Deploy

```bash
npx wrangler deploy
# Server accessible at https://[worker-name].[account].workers.dev/mcp
```

### Step 6: Connect Clients

**Claude Desktop** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "my-server": {
      "command": "npx",
      "args": ["mcp-remote", "https://my-mcp.workers.dev/mcp"]
    }
  }
}
```
Restart Claude Desktop after updating config.

## Tool Patterns

### Return Types
```typescript
// Text response
return { content: [{ type: "text", text: "result" }] };

// Multiple content items
return {
  content: [
    { type: "text", text: "Here's the data:" },
    { type: "text", text: JSON.stringify(data, null, 2) },
  ],
};
```

### Input Validation with Zod
```typescript
this.server.tool(
  "create_user",
  {
    email: z.string().email(),
    name: z.string().min(1).max(100),
    role: z.enum(["admin", "user", "guest"]),
    age: z.number().int().min(0).optional(),
  },
  async (params) => { /* params are fully typed and validated */ }
);
```

### Accessing Cloudflare Bindings
```typescript
export class MyMCP extends McpAgent<Env> {
  async init() {
    this.server.tool("query_db", { sql: z.string() }, async ({ sql }) => {
      const result = await this.env.DB.prepare(sql).all();
      return { content: [{ type: "text", text: JSON.stringify(result) }] };
    });
  }
}
```

## Wrangler Configuration

```toml
name = "my-mcp-server"
main = "src/index.ts"
compatibility_date = "2024-12-01"

[durable_objects]
bindings = [{ name = "MCP", class_name = "MyMCP" }]

[[migrations]]
tag = "v1"
new_classes = ["MyMCP"]

# Optional: D1, KV bindings
[[d1_databases]]
binding = "DB"
database_name = "my-db"
database_id = "xxx"
```

## OAuth Setup

Supported providers: GitHub, Google, Auth0, Stytch, WorkOS, any OAuth 2.0 provider.

Set secrets via Wrangler:
```bash
wrangler secret put GITHUB_CLIENT_ID
wrangler secret put GITHUB_CLIENT_SECRET
```

Callback URL must match OAuth app config exactly. For local dev: `http://localhost:8788/callback`.

## Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| "Tool not found" | Name mismatch | Verify exact case; ensure `init()` registers before connections |
| Connection fails | Wrong path | Confirm endpoint is `/mcp`; check CORS for browser clients |
| OAuth redirect error | Callback URL mismatch | Match OAuth app config; check secrets are set |

## Output

A Cloudflare Workers project directory containing:
- `src/index.ts` — entry point with SSE routing
- `src/mcp.ts` — `McpAgent` class with tool definitions
- `wrangler.toml` — deployment configuration
- `package.json` — dependencies including `agents`, `zod`, `@modelcontextprotocol/sdk`
