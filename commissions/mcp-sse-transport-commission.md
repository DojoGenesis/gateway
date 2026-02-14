# Implementation Commission: MCP SSE & Streamable HTTP Transport Support

**Objective:** Implement SSE and streamable_http transport support in the MCP module so the Gateway can connect to remote MCP servers over HTTP, not just local stdio subprocesses.

---

## 1. Context & Grounding

**Primary Specification:**
- This commission is self-contained. The MCP module already has config, connection, and host manager wired — only the transport establishment in `Connect()` needs implementation.

**Pattern Files (Follow these examples):**
- `mcp/connection.go` lines 54-74 (`connectStdio`): Follow this exact pattern for the new `connectSSE` and `connectStreamableHTTP` methods. Same mutex usage, same `c.client =` / `c.healthy = true` pattern.
- `mcp/config.go` lines 86-104 (`TransportConfig`): The config struct already has `URL` and `Headers` fields for SSE/HTTP transports. These are already validated at lines 246-261.
- `mcp/host.go` lines 78-119 (`startServer`): No changes needed here — it already calls `conn.Connect(ctx)` generically regardless of transport type.

**Files to Read:**
- `mcp/connection.go` — The file you will modify. Read the full file.
- `mcp/connection_test.go` — Existing tests including the SSE "not implemented" test at line 64.
- `mcp/config.go` — Config types and validation. Already supports SSE/HTTP.
- `mcp/host.go` — Host manager that calls Connect(). No changes needed.
- `mcp/go.mod` — Confirm `github.com/mark3labs/mcp-go v0.43.2` is the dependency.

**Library Reference:**
The `mcp-go` library (v0.43.2) provides:
- `client.NewSSEMCPClient(baseURL string, options ...transport.ClientOption) (*Client, error)` — Creates SSE transport client
- `client.NewStreamableHttpClient(baseURL string, options ...transport.ClientOption) (*Client, error)` — Creates streamable HTTP transport client
- `transport.WithHeaders(headers map[string]string) transport.ClientOption` — Adds custom HTTP headers (for auth tokens, API keys)

---

## 2. Detailed Requirements

### Phase 1: Implement SSE Transport (Steps 1-5)

1. In `mcp/connection.go`, add a new method `connectSSE(ctx context.Context) error` following the exact pattern of `connectStdio`:
   - Build a `[]transport.ClientOption` slice from `c.config.Transport.Headers` using `transport.WithHeaders()`
   - Call `client.NewSSEMCPClient(c.config.Transport.URL, options...)` to create the client
   - If `NewSSEMCPClient` returns an error, wrap it: `fmt.Errorf("failed to connect to MCP server %s via SSE: %w", c.name, err)`
   - Set `c.client = mcpClient` and `c.healthy = true`
   - Return nil on success

2. In `mcp/connection.go`, add the import for `github.com/mark3labs/mcp-go/client/transport` (needed for `transport.WithHeaders`)

3. In `mcp/connection.go` method `Connect()`, replace the SSE error return at line 46-47:
   ```go
   } else if c.config.Transport.Type == "sse" {
       return c.connectSSE(ctx)
   }
   ```

### Phase 2: Implement Streamable HTTP Transport (Steps 4-5)

4. In `mcp/connection.go`, add method `connectStreamableHTTP(ctx context.Context) error` with the same pattern as `connectSSE` but calling `client.NewStreamableHttpClient(c.config.Transport.URL, options...)`.

5. In `mcp/connection.go` method `Connect()`, add a case for streamable_http after the SSE case:
   ```go
   } else if c.config.Transport.Type == "streamable_http" {
       return c.connectStreamableHTTP(ctx)
   }
   ```

### Phase 3: Update Tests (Steps 6-9)

6. In `mcp/connection_test.go`, update `TestMCPServerConnection_SSETransport` (line 64):
   - Remove the assertion that SSE returns an error
   - The test should now verify that `NewSSEMCPClient` is called (it will fail with a connection error since there's no real server, but the error should NOT be "SSE transport not yet implemented")
   - Assert the error message contains connection-related text (e.g., "failed to connect"), NOT "not yet implemented"

7. In `mcp/connection_test.go`, add `TestMCPServerConnection_StreamableHTTPTransport`:
   - Same pattern as the SSE test but with `Type: "streamable_http"`
   - Assert connection attempt is made (error is connection failure, not "not implemented")

8. In `mcp/connection_test.go`, add `TestMCPServerConnection_SSEWithHeaders`:
   - Config with `Headers: map[string]string{"Authorization": "Bearer test-token"}`
   - Verify the connection attempt includes headers (will fail at network level, but validates the code path)

9. In `mcp/connection_test.go`, add `TestMCPServerConnection_StreamableHTTPWithHeaders`:
   - Same as above but for streamable_http transport

### Phase 4: Config Example & Documentation (Steps 10-11)

10. In `gateway-config.yaml`, add a commented-out example SSE server configuration block after the existing stdio server:
    ```yaml
    # Example: Remote MCP server via SSE
    # - id: composio
    #   display_name: "Composio MCP"
    #   namespace_prefix: "composio"
    #   transport:
    #     type: sse
    #     url: "https://mcp.composio.dev/sse"
    #     headers:
    #       Authorization: "Bearer ${COMPOSIO_API_KEY}"
    #   health_check:
    #     enabled: true
    #     interval_sec: 30
    ```

11. Remove the comment "SSE transport not implemented in v0.2.0" from `mcp/connection.go` line 46.

---

## 3. File Manifest

**Modify:**
- `mcp/connection.go` — Add `connectSSE()`, `connectStreamableHTTP()`, update `Connect()` routing, add transport import
- `mcp/connection_test.go` — Update SSE test, add streamable_http test, add header tests
- `gateway-config.yaml` — Add commented SSE example

**No files to create or delete.**

---

## 4. Success Criteria

- [ ] `Connect()` with `type: "sse"` attempts a real SSE connection (no longer returns "not yet implemented")
- [ ] `Connect()` with `type: "streamable_http"` attempts a real streamable HTTP connection
- [ ] Custom headers from `TransportConfig.Headers` are passed to the mcp-go client
- [ ] All existing tests in `mcp/connection_test.go` pass
- [ ] New SSE and streamable_http tests pass
- [ ] `go vet ./mcp/...` passes with zero warnings
- [ ] `go build ./mcp/...` compiles cleanly
- [ ] No references to "SSE transport not yet implemented" remain in any `.go` file
- [ ] `gateway-config.yaml` contains a commented SSE example

---

## 5. Constraints & Boundaries

- **DO NOT** modify `mcp/host.go` — it already handles transport-agnostic connection via `conn.Connect(ctx)`
- **DO NOT** modify `mcp/config.go` — config structs and validation already support SSE/HTTP
- **DO NOT** add any new dependencies — `mcp-go v0.43.2` already provides all needed client constructors
- **DO NOT** implement a custom SSE client — use the library's `client.NewSSEMCPClient`
- **DO NOT** modify any files outside the `mcp/` module and `gateway-config.yaml`
- **DO NOT** add reconnection logic beyond what `mcp/host.go` already provides (it has `checkAndReconnect`)

---

## 6. Integration Points

- `mcp/host.go:startServer()` calls `conn.Connect(ctx)` — this is the sole consumer of the transport methods. No changes needed there.
- `mcp/config.go:TransportConfig` already carries `URL` and `Headers` fields — fully wired for SSE/HTTP.
- Health check loop in `host.go:healthCheckLoop()` already calls `checkAndReconnect()` which creates new connections — SSE reconnection comes for free.

---

## 7. Testing Requirements

**Unit Tests:**
- Test SSE connection attempt (verifies code path, will fail at network level)
- Test streamable_http connection attempt
- Test SSE with custom headers
- Test streamable_http with custom headers
- All existing tests continue to pass

**Edge Cases:**
- SSE with empty URL (should be caught by config validation, not connection code)
- SSE with invalid URL (connection error, not panic)
- Headers with `${VAR}` expansion (handled by config loader, not connection code)
