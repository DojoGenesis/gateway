# MCP Implementation vs Specification Gap Analysis

**Date:** 2026-02-12
**Specification:** `docs/v0.2.0/gateway-mcp-contract.md`
**Implementation:** `mcp/` module

## Critical Gaps

### 1. YAML Schema Structure ❌ MAJOR

**Specified:**
```yaml
version: "1.0"
mcp:
  global:
    default_tool_timeout: 30
    reconnect:
      max_retries: 3
      initial_backoff_ms: 1000
    health_check_interval: 60
  servers:
    - id: "mcp_by_dojo"
      display_name: "MCP By Dojo Genesis"
      namespace_prefix: "mcp_by_dojo"
      transport:
        type: "stdio"
        command: "/path/to/binary"
      timeouts:
        startup: 10
        tool_default: 30
      health_check:
        enabled: true
        interval_sec: 60
```

**Implemented:**
```yaml
servers:
  - name: composio
    transport: stdio
    command: python
    namespace_prefix: "composio."
    health_check_interval_sec: 30
    timeout_sec: 60
```

**Impact:** HIGH - Configuration structure incompatible
**Resolution:** Need to restructure config types to match spec

---

### 2. Namespace Format ❌ CRITICAL

**Specified:** `namespace_prefix:tool_name` (with colon separator)
**Implemented:** `namespace_prefix + tool_name` (simple concatenation)

**Example:**
- Spec: `"mcp_by_dojo:create_artifact"`
- Mine: `"mcp_by_dojo.create_artifact"` (if prefix is "mcp_by_dojo.")

**Impact:** HIGH - Tool names won't match expected format
**Location:** `mcp/bridge.go:57`

**Fix Required:**
```go
// Current:
fullName := namespacePrefix + mcpTool.Name

// Should be:
fullName := strings.TrimSuffix(namespacePrefix, ".") + ":" + mcpTool.Name
// Or better: enforce namespace_prefix WITHOUT trailing punctuation
fullName := namespacePrefix + ":" + mcpTool.Name
```

---

### 3. OTEL Span Naming ❌ CRITICAL

**Specified:** Span name = `"mcp.tool.invoke"`
**Implemented:** Span name = `"mcp.tool.call"`

**Impact:** MEDIUM - Breaks observability conventions
**Location:** `mcp/otel.go:14`

**Fix Required:**
```go
const SpanNameToolCall = "mcp.tool.invoke" // Not "mcp.tool.call"
```

---

### 4. OTEL Span Attributes ❌ HIGH

**Specified Attributes:**
```
mcp.server_id
mcp.server_display_name
mcp.tool_name
mcp.tool_namespaced
mcp.tool_latency_ms
mcp.tool_success (bool)
```

**Implemented Attributes:**
```
mcp.server_name
mcp.tool_name
mcp.latency_ms
mcp.error
```

**Missing:**
- `mcp.server_id` (vs `mcp.server_name`)
- `mcp.server_display_name`
- `mcp.tool_namespaced`
- `mcp.tool_success`

**Impact:** MEDIUM - Incomplete telemetry
**Location:** `mcp/otel.go` attribute constants

---

### 5. Type Structure ❌ MAJOR

**Specified Types (Section 3.3):**
- `MCPHostConfig` (with Version, Global, Servers, Observability)
- `GlobalMCPConfig`
- `TransportConfig` (nested object, not flat)
- `TimeoutConfig` (nested object)
- `HealthCheckConfig` (nested object with Path field)
- `RetryPolicy` (nested object)
- `ObservabilityConfig`

**Implemented Types:**
- `MCPConfig` (flat)
- `ServerConfig` (flat with embedded fields)

**Impact:** HIGH - Cannot parse spec-compliant YAML
**Location:** `mcp/config.go`

---

### 6. Server ID vs Name 🟡 MODERATE

**Specified:**
- `id` (string): unique identifier, alphanumeric+underscore
- `display_name` (string): human-readable name

**Implemented:**
- `name` (string): used for both purposes

**Impact:** MODERATE - Works functionally but doesn't match spec
**Location:** `mcp/config.go:23`, `mcp/connection.go`

---

### 7. Transport Configuration ❌ HIGH

**Specified:**
```go
type TransportConfig struct {
    Type    string            // "stdio", "sse", "streamable_http"
    Command string            // For stdio
    Args    []string          // For stdio
    Env     map[string]string // For stdio
    URL     string            // For SSE/HTTP
    Headers map[string]string // For SSE/HTTP
}
```

**Implemented:** Flat in ServerConfig
```go
type ServerConfig struct {
    Transport string   // Just the type
    Command   string
    Args      []string
    Env       map[string]string
}
```

**Impact:** HIGH - Structure mismatch
**Location:** `mcp/config.go`

---

## Minor Gaps

### 8. Connection State Machine 🟡 MODERATE

**Specified States:**
- `StateInitializing`
- `StateConnected`
- `StateUnhealthy`
- `StateReconnecting`
- `StateDisconnected`
- `StateTerminated`

**Implemented:** Simple bool `healthy` field

**Impact:** MODERATE - Less granular state tracking
**Location:** `mcp/connection.go`

---

### 9. Health Check Configuration 🟡 MODERATE

**Specified:**
```go
type HealthCheckConfig struct {
    Enabled     bool
    Path        string  // Endpoint to check
    IntervalSec int
}
```

**Implemented:** Just `health_check_interval_sec` as int

**Impact:** MODERATE - Missing Path field
**Location:** `mcp/config.go`

---

### 10. Retry Policy Structure 🟡 MODERATE

**Specified:**
```go
type RetryPolicy struct {
    MaxAttempts       int
    BackoffMultiplier float64
    MaxBackoffMs      int
}
```

**Implemented:** Flat fields in ServerConfig

**Impact:** LOW - Functionally works but structure differs
**Location:** `mcp/config.go`

---

## Alignment Recommendations

### Priority 1: CRITICAL (Required for spec compliance)

1. **Fix namespace format** - Use colon separator
2. **Fix OTEL span name** - "mcp.tool.invoke"
3. **Add missing OTEL attributes** - server_id, display_name, tool_namespaced, tool_success

### Priority 2: HIGH (Breaking changes to config)

4. **Restructure YAML schema** - Nested structure with global/servers/observability
5. **Create TransportConfig type** - Separate transport configuration
6. **Add TimeoutConfig type** - Nested timeout settings
7. **Add server ID vs display_name** - Separate identification from display

### Priority 3: MODERATE (Enhancements)

8. **Implement connection state machine** - Full state tracking
9. **Add HealthCheckConfig** - Include Path field
10. **Create RetryPolicy type** - Separate retry configuration
11. **Add ObservabilityConfig** - Observability settings

---

## Recommendation

Given this is **Prompt 2A** implementation and we're approaching the spec:

**Option A: Full Alignment (Recommended for Production)**
- Implement all Priority 1 & 2 items
- Ensures full spec compliance
- Allows seamless integration with other MCP tools
- ~4 hours work

**Option B: Minimal Viable (Current + Critical Fixes)**
- Fix Priority 1 items only (namespace, OTEL)
- Document deviations from spec
- Note in docs: "Simplified config schema"
- ~1 hour work

**Option C: Document As-Is**
- Current implementation works functionally
- Document as "v0.2.0 simplified implementation"
- Full spec compliance in v0.3.0
- No additional work

**My Recommendation:** **Option A** - The spec exists for good reasons (interoperability, tooling, monitoring). We should align now while the code is fresh.
