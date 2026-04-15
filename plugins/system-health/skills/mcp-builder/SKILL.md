---
name: mcp-builder
model: sonnet
description: "Scaffolds new MCP servers with tool definitions, transport wiring, handler structure, and test harness for Python (FastMCP) or TypeScript (MCP SDK). Use when: 'build a new MCP server', 'scaffold an MCP integration', 'create tools for an external API', 'set up MCP transport and handlers'."
category: system-health
triggers:
  - "build a new MCP server"
  - "scaffold an MCP integration"
  - "create tools for an external API"
  - "set up MCP transport and handlers"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: server_name
    type: string
    description: Name of the MCP server to scaffold (used for directory and package naming)
    required: true
  - name: tools
    type: string[]
    description: List of tool names to generate stubs for (e.g., list_items, create_item, search)
    required: false
  - name: transport
    type: string
    description: "Transport protocol: 'stdio' (default, for CLI integration) or 'sse' (for HTTP/server deployment)"
    required: false
outputs:
  - name: scaffold
    type: ref
    format: cas-ref
    description: Complete MCP server scaffold with project structure, tool handlers, transport config, and test harness
---
# MCP Builder

**Purpose:** Scaffold production-quality MCP (Model Context Protocol) servers that enable LLMs to interact with external services through well-designed tools. Produces a complete project structure with tool definitions, transport wiring, handler implementations, input validation, error handling, and an evaluation test harness.

---

## I. When to Use

- Starting a new MCP server to integrate an external API or service
- Adding MCP tool access to an existing service
- Need a working scaffold with transport, validation, and tests from the start
- Building for either Python (FastMCP) or TypeScript (MCP SDK) targets

---

## II. Agent-Centric Design Principles

MCP tools are used by AI agents, not humans. Design accordingly:

1. **Build for workflows, not API endpoints** -- consolidate related operations (e.g., `schedule_event` that checks availability AND creates the event)
2. **Optimize for limited context** -- return high-signal information, not exhaustive data dumps; offer concise vs. detailed response formats
3. **Actionable error messages** -- errors should guide agents toward correct usage ("Try filter='active_only' to reduce results")
4. **Natural task subdivisions** -- tool names reflect how humans think about tasks, with consistent prefixes for discoverability
5. **Evaluation-driven development** -- create realistic eval scenarios early; let agent feedback drive tool improvements

---

## III. Workflow

### Phase 1: Research and Planning

1. **Study the target API** -- read all available documentation (endpoints, auth, rate limits, pagination, error codes, data models)
2. **Select high-value tools** -- prioritize operations that enable complete workflows, not just individual API calls
3. **Plan shared utilities** -- identify common patterns (API request helpers, pagination, error formatting, auth token management)
4. **Design input/output schemas** -- Pydantic models for Python, Zod schemas for TypeScript; include constraints and descriptive field docs

### Phase 2: Scaffold Generation

For each target language, generate the project structure:

**Python (FastMCP):**
```
{server_name}/
  server.py          # MCP server with @mcp.tool registrations
  models.py          # Pydantic input validation models
  utils.py           # Shared API request helpers, error formatting
  requirements.txt   # Dependencies (mcp, pydantic, httpx)
  tests/
    test_tools.py    # Unit tests for each tool handler
    eval.xml         # Evaluation questions for agent testing
```

**TypeScript (MCP SDK):**
```
{server_name}/
  src/
    index.ts         # MCP server with registerTool calls
    schemas.ts       # Zod input validation schemas
    utils.ts         # Shared helpers
  package.json
  tsconfig.json
  tests/
    tools.test.ts    # Unit tests
    eval.xml         # Evaluation questions
```

### Phase 3: Tool Implementation

For each tool in the tools list:

1. **Define input schema** with proper constraints (min/max length, regex, ranges) and descriptive field docs with examples
2. **Write comprehensive docstring** -- one-line summary, detailed purpose, parameter types with examples, return schema, usage examples
3. **Implement handler** using shared utilities, async/await for I/O, proper error handling, response format options (JSON/Markdown)
4. **Add tool annotations** -- `readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`

### Phase 4: Transport Wiring

Configure the selected transport:

- **stdio** (default): reads from stdin, writes to stdout; suitable for CLI tool integration and local development
- **sse**: HTTP server with Server-Sent Events; suitable for remote deployment and multi-client scenarios

### Phase 5: Test Harness

1. **Unit tests** for each tool handler with mocked API responses
2. **Build verification** -- `python -m py_compile server.py` or `npm run build`
3. **Evaluation harness** -- 10 realistic questions that test whether an agent can accomplish real tasks using the tools

---

## IV. Quality Checklist

- [ ] Every tool has a comprehensive docstring with examples
- [ ] Input validation catches malformed requests with actionable error messages
- [ ] No duplicated code between tool handlers (shared utilities extracted)
- [ ] Pagination is handled (not left to the caller)
- [ ] Character/token limits respected (truncation at 25,000 characters)
- [ ] All external calls have error handling with retry for transient failures
- [ ] Tool annotations set correctly (readOnly, destructive, idempotent, openWorld)
- [ ] Transport configured and tested (stdio or sse)
- [ ] Build passes without errors
- [ ] Evaluation harness with 10 questions created

---

## V. Output

- Complete MCP server project scaffold at the specified location
- Contents: project structure, tool handler implementations, input validation schemas, shared utilities, transport configuration, test suite, evaluation harness
- Ready to run: `python server.py` (Python) or `npm run build && node dist/index.js` (TypeScript)

---

## VI. Examples

**Scenario 1:** "Build an MCP server for the GitHub API" with tools=[list_repos, search_code, get_pull_request, create_issue] and transport=stdio --> Python scaffold with 4 tool handlers, Pydantic models for each input, shared GitHub API client with auth token management, pagination helper, Markdown response formatter, 10 eval questions testing real GitHub workflows.

**Scenario 2:** "Scaffold an MCP server for our internal inventory API" with server_name=inventory-mcp and transport=sse --> TypeScript scaffold with SSE transport, Zod schemas, placeholder tool stubs ready for implementation, HTTP server configuration, CORS setup for remote access.

**Scenario 3:** "Create an MCP server that wraps our Postgres database" with tools=[query, list_tables, describe_table] --> Python scaffold with read-only tool annotations, SQL injection prevention in input validation, result truncation at 25K characters, connection pooling in shared utilities.

---

## VII. Edge Cases

- No tools specified: generate a minimal scaffold with one example tool (`hello_world`) as a template; include comments showing how to add more
- Target API requires OAuth flow: scaffold includes token refresh logic in shared utilities; document the manual auth setup step in README
- Server needs both stdio and SSE: scaffold both transport configurations; use environment variable to select at runtime
- Very large API surface (50+ endpoints): do not scaffold all 50; select the 8-12 highest-value workflow tools and document the rest as future additions

---

## VIII. Anti-Patterns

- Wrapping every API endpoint as a separate tool -- consolidate into workflow-level tools that complete tasks, not individual HTTP calls
- Returning raw API responses without formatting -- agents have limited context; parse and summarize responses
- Skipping input validation -- unvalidated inputs produce cryptic API errors that agents cannot recover from
- Testing by running the server directly in the main process -- MCP servers block on stdin; use the evaluation harness or run in tmux
- Hardcoding API keys in the scaffold -- use environment variables; never commit credentials
