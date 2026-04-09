---
name: mcp-server-builder
model: sonnet
description: Scaffolds production-ready MCP servers from OpenAPI specs with schema validation, named tool definitions, and versioning strategy. Use when: "build MCP server from API", "create tool server from OpenAPI", "expose REST API to LLM agent", "scaffold MCP from spec", "convert API contract to tools".
category: skill-forge

inputs:
  - name: openapi_spec
    type: string
    description: Path or URL to the OpenAPI spec (JSON or YAML) to convert into MCP tool definitions
    required: true
  - name: server_name
    type: string
    description: Canonical name for the MCP server (used in tool manifests and package naming)
    required: true
  - name: language
    type: string
    description: Implementation language — "python" or "typescript"
    required: false
outputs:
  - name: mcp_scaffold
    type: ref
    format: directory
    description: MCP server scaffold with tool manifest (JSON), starter server code (Python or TypeScript), and validation report
---

# MCP Server Builder

Scaffolds production-ready MCP servers from OpenAPI specs. Treats the API contract as the source of truth and generates tool manifests + starter code with quality gates before publication.

## Description

Converts OpenAPI paths/operations into named MCP tool definitions, generates server scaffolds in Python or TypeScript, enforces naming and description quality, validates manifests against common production failures, and applies a versioning strategy for safe evolution. Distinct from `mcp-cloudflare-builder` (Cloudflare-specific deployment) and `mcp-builder` (general MCP concepts) — this skill is spec-driven and language-agnostic.

## When to Use

- Exposing an internal or external REST API to LLM agents via MCP
- Replacing brittle browser automation with typed, validated tools
- Sharing one MCP server across multiple teams or assistants
- Bootstrapping from an existing OpenAPI spec rather than hand-writing tool definitions
- Running quality checks before publishing MCP tools to a registry

## Workflow

### Step 1: Parse the OpenAPI Spec

Read the spec and extract:
- Paths and HTTP methods
- `operationId` (use as canonical tool name when available)
- Request parameters and body schemas
- Response schemas
- Authentication requirements

### Step 2: Generate Tool Manifest

For each operation, create an MCP tool definition:

```json
{
  "name": "get_user",
  "description": "Retrieve a user by ID. Returns full profile including email and role.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "user_id": { "type": "string", "description": "The user's UUID" }
    },
    "required": ["user_id"]
  }
}
```

**Naming rules:**
- Use `operationId` when available (preferred)
- Otherwise derive from method + path: `GET /users/{id}` → `get_user`
- Use snake_case; avoid raw path segments (`get__v1__users___id` is wrong)
- One intent per tool; avoid mega-tools combining unrelated operations

**Description rules:**
- Lead with action verb + object
- Include what the tool returns, not just what it does
- Note any side effects for mutating operations

### Step 3: Scaffold Server Code

Generate starter implementation in the chosen language:

**Python (FastMCP):**
```python
from mcp.server.fastmcp import FastMCP

mcp = FastMCP("billing-mcp")

@mcp.tool()
def get_user(user_id: str) -> dict:
    """Retrieve a user by ID. Returns full profile."""
    response = requests.get(f"{BASE_URL}/users/{user_id}", headers=AUTH_HEADERS)
    response.raise_for_status()
    return response.json()
```

**TypeScript (MCP SDK):**
```typescript
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";

const server = new McpServer({ name: "billing-mcp", version: "1.0.0" });

server.tool("get_user", { user_id: z.string() }, async ({ user_id }) => {
  const res = await fetch(`${BASE_URL}/users/${user_id}`, { headers: AUTH_HEADERS });
  return { content: [{ type: "text", text: await res.text() }] };
});
```

### Step 4: Validate the Manifest

Before integration, check for:
- Duplicate tool names
- Missing or empty descriptions
- Parameters with no types
- Ambiguous schemas (no `required` fields on non-trivial inputs)
- Tool names that expose secret values
- Destructive operations without explicit confirmation parameters

Fix all validation errors before proceeding.

### Step 5: Apply Auth and Safety Controls

- Store secrets in environment variables, never in tool schemas
- Allowlist outbound hosts explicitly; do not proxy arbitrary user-provided URLs
- Return structured errors: `{ "code": "NOT_FOUND", "message": "...", "details": {...} }`
- Add explicit confirmation parameters to destructive operations
- Rate-limit high-cost tools and add request timeouts

### Step 6: Versioning Strategy

- Additive fields only for non-breaking updates
- Never rename a tool name in-place — introduce a new tool ID for breaking behavior changes
- Maintain a changelog of tool contract changes per release
- Validate backward compatibility in CI using strict mode

## Contract Quality Gates

Before publishing the manifest, confirm:

1. Every tool has a verb-first, clear name
2. Every description explains intent AND expected result
3. Every required field is explicitly typed
4. Destructive actions include a confirmation parameter
5. Error payload format is consistent across all tools
6. Validator returns zero errors in strict mode

## Runtime Selection

| Constraint | Choose |
|-----------|--------|
| Fast iteration, data-heavy backend | Python |
| Shared types with JS stack | TypeScript |
| Single server, simple ops | Python FastMCP |
| Split domain servers with strict contracts | TypeScript + Zod |

## Common Pitfalls

1. Tool names derived from raw paths: `get__v1__users___id` — fix with clean `operationId`
2. Missing operation descriptions — agents choose tools poorly without them
3. Ambiguous schemas with no `required` fields
4. Mixing transport errors and domain errors in one opaque message
5. Exposing secret values in tool contracts
6. Breaking clients by changing schema keys without versioning

## Output

An MCP server scaffold directory containing:
- `tool_manifest.json` — all tool definitions with names, descriptions, and schemas
- `server.py` or `server.ts` — starter implementation with stubs for each tool
- `CHANGELOG.md` — initial entry for v1.0.0 contract
- Validation report listing any issues found
