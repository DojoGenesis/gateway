# Skill Adapter Development Guide

**Version:** 1.0 (Phase 4a)
**Last Updated:** 2026-02-13

---

## Overview

Adapters bridge the gap between gateway-native tools and skill requirements. When a skill depends on capabilities not directly available in the gateway (like web search or script execution), an adapter provides the implementation.

---

## When Do You Need an Adapter?

A skill needs an adapter when it depends on a tool not natively available in the gateway core.

### Decision Tree

```
Does the skill use file_system or bash only?
  ✅ NO ADAPTER NEEDED (Tier 1)

Does the skill use web_tools?
  ✅ NEEDS WebToolAdapter (Tier 2)

Does the skill use script_execution?
  ✅ NEEDS ScriptExecutor (Tier 2)

Does the skill invoke other skills?
  ⏭️ NEEDS DAG subtask binding (Tier 3, Phase 4b)

Does the skill require custom integrations?
  ⚠️  Requires refactoring (Tier 4, Future)
```

---

## Phase 4a Adapters

### 1. WebToolAdapter

**Purpose:** Provides web search and HTTP fetch capabilities for Tier 2 skills.

**Location:** `pkg/skill/adapters/web_tools.go`

**Capabilities:**
- `Search(ctx, query)` - Web search via Brave API or fallback
- `Fetch(ctx, url, mode)` - HTTP GET with parsing modes (raw, json, markdown)

**Configuration:**
```go
adapter := adapters.NewWebToolAdapter(&adapters.WebToolAdapterConfig{
    BraveAPIKey: os.Getenv("BRAVE_API_KEY"), // Optional
    MaxResults:  10,
    Timeout:     10 * time.Second,
})
```

**Fallback Behavior:**
- If `BraveAPIKey` is empty, adapter enters fallback mode
- Fallback returns placeholder results with warning
- Skills degrade gracefully without API key

**Example Usage in Skill:**
```markdown
## Skill: web-research

This skill uses web_tools to search for relevant information.

**Tool Dependencies:** web_tools

When invoked, the skill will:
1. Use WebToolAdapter.Search() for web queries
2. Use WebToolAdapter.Fetch() to retrieve page content
3. Parse and synthesize results
```

---

### 2. ScriptExecutor

**Purpose:** Safely executes allowed scripts for Tier 2 skills.

**Location:** `pkg/skill/script_executor.go`

**Security Model:**
- **Allowlist:** Only 8 scripts can execute
- **Validation:** Blocks shell metacharacters in arguments
- **Isolation:** Path traversal prevention
- **Timeout:** 30-second default limit

**Allowed Scripts:**
1. `init_skill.py`
2. `suggest_seeds.py`
3. `diff_tracker.py`
4. `context_mapper.py`
5. `smart_clone.sh`
6. `apply_seed.py`
7. `lychee`
8. `validate_skill.py`

**Configuration:**
```go
executor := skill.NewScriptExecutor(&skill.ScriptExecutorConfig{
    BaseDir: "/plugins",
    Timeout: 30 * time.Second,
})
```

**Example Usage in Skill:**
```markdown
## Skill: seed-extraction

This skill uses script_execution to run pattern extraction.

**Tool Dependencies:** script_execution
**Required Scripts:** suggest_seeds.py

When invoked, the skill will:
1. Validate seed extraction context
2. Execute suggest_seeds.py with validated arguments
3. Parse script output
4. Return extracted patterns
```

---

## How to Write an Adapter

### Step 1: Identify the Need

Analyze the skill's tool dependencies:

```yaml
tool_dependencies:
  - file_system  # ✅ Native
  - bash         # ✅ Native
  - custom_api   # ⚠️  Needs adapter
```

### Step 2: Define the Interface

Create an interface that skill execution can depend on:

```go
// pkg/skill/adapters/custom_api.go
package adapters

import "context"

type CustomAPIAdapter struct {
    apiKey string
    client *http.Client
}

func NewCustomAPIAdapter(apiKey string) *CustomAPIAdapter {
    return &CustomAPIAdapter{
        apiKey: apiKey,
        client: &http.Client{Timeout: 10 * time.Second},
    }
}

func (a *CustomAPIAdapter) Query(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
    // Implementation here
    return map[string]interface{}{
        "status": "success",
        "data":   "result",
    }, nil
}
```

### Step 3: Implement Core Methods

**Required Methods:**
- `IsAvailable()` - Returns true if adapter is configured
- Error handling with context
- Timeout support via `context.Context`
- Graceful degradation

**Example:**
```go
func (a *CustomAPIAdapter) IsAvailable() bool {
    return a.apiKey != "" && a.client != nil
}

func (a *CustomAPIAdapter) Query(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
    if !a.IsAvailable() {
        return nil, fmt.Errorf("adapter not configured")
    }

    // Timeout handling
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    // Implementation...
}
```

### Step 4: Write Comprehensive Tests

**Test Coverage:**
- Happy path (normal operation)
- Error cases (invalid input, timeout, API failure)
- Fallback behavior (if applicable)
- Configuration edge cases

**Example Test:**
```go
func TestCustomAPIAdapter_Query(t *testing.T) {
    adapter := NewCustomAPIAdapter("test-key")
    ctx := context.Background()

    result, err := adapter.Query(ctx, map[string]interface{}{
        "query": "test",
    })

    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "success", result["status"])
}

func TestCustomAPIAdapter_Timeout(t *testing.T) {
    adapter := NewCustomAPIAdapter("test-key")
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
    defer cancel()

    _, err := adapter.Query(ctx, map[string]interface{}{})

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "timeout")
}
```

### Step 5: Register with SkillExecutor

Update skill executor to inject adapter dependencies:

```go
// In SkillExecutor initialization
func NewSkillExecutorWithAdapters(
    registry SkillRegistry,
    toolInvoker ToolInvoker,
    traceLogger TraceLogger,
    webAdapter *adapters.WebToolAdapter,
    scriptExecutor *ScriptExecutor,
    customAdapter *adapters.CustomAPIAdapter,
) *DefaultSkillExecutor {
    executor := NewSkillExecutor(registry, toolInvoker, traceLogger)
    // Register adapters
    executor.webAdapter = webAdapter
    executor.scriptExecutor = scriptExecutor
    executor.customAdapter = customAdapter
    return executor
}
```

---

## Adapter Best Practices

### 1. Graceful Degradation
```go
// ✅ GOOD: Fallback behavior
if !adapter.IsAvailable() {
    return adapter.fallbackMode(ctx, params)
}

// ❌ BAD: Hard failure
if !adapter.IsAvailable() {
    return nil, errors.New("adapter required")
}
```

### 2. Timeout Support
```go
// ✅ GOOD: Respect context timeout
func (a *Adapter) Fetch(ctx context.Context, url string) ([]byte, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := a.client.Do(req)
    // ...
}

// ❌ BAD: Ignore context
func (a *Adapter) Fetch(ctx context.Context, url string) ([]byte, error) {
    resp, err := http.Get(url) // No timeout!
    // ...
}
```

### 3. Error Context
```go
// ✅ GOOD: Wrap errors with context
return nil, fmt.Errorf("failed to fetch %s: %w", url, err)

// ❌ BAD: Generic errors
return nil, err
```

### 4. Resource Cleanup
```go
// ✅ GOOD: Always clean up
resp, err := client.Do(req)
if err != nil {
    return nil, err
}
defer resp.Body.Close()

// ❌ BAD: Resource leak
resp, err := client.Do(req)
body, _ := io.ReadAll(resp.Body) // No defer close!
```

### 5. Configuration Validation
```go
// ✅ GOOD: Validate on creation
func NewAdapter(config *Config) (*Adapter, error) {
    if config.APIKey == "" {
        return nil, errors.New("API key required")
    }
    return &Adapter{apiKey: config.APIKey}, nil
}

// ❌ BAD: Panic at runtime
func (a *Adapter) Query(ctx context.Context) {
    // Panics if apiKey is empty!
    resp := callAPI(a.apiKey, ...)
}
```

---

## Common Patterns

### Pattern 1: HTTP-Based Adapter

```go
type HTTPAdapter struct {
    baseURL string
    client  *http.Client
}

func (a *HTTPAdapter) Call(ctx context.Context, endpoint string, params map[string]interface{}) (map[string]interface{}, error) {
    url := a.baseURL + endpoint
    req, _ := http.NewRequestWithContext(ctx, "POST", url, toJSON(params))
    resp, err := a.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("API call failed: %w", err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result, nil
}
```

### Pattern 2: Script Execution Adapter

```go
type ScriptAdapter struct {
    scriptsDir string
    timeout    time.Duration
}

func (a *ScriptAdapter) Execute(ctx context.Context, scriptName string, args []string) (map[string]interface{}, error) {
    if !a.isAllowed(scriptName) {
        return nil, ErrScriptNotAllowed
    }

    ctx, cancel := context.WithTimeout(ctx, a.timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, scriptName, args...)
    output, err := cmd.CombinedOutput()

    return map[string]interface{}{
        "output": string(output),
        "error":  err,
    }, nil
}
```

### Pattern 3: Cached Adapter

```go
type CachedAdapter struct {
    adapter Adapter
    cache   map[string]CacheEntry
    ttl     time.Duration
}

func (a *CachedAdapter) Query(ctx context.Context, key string) (interface{}, error) {
    // Check cache
    if entry, exists := a.cache[key]; exists {
        if time.Since(entry.timestamp) < a.ttl {
            return entry.value, nil
        }
    }

    // Call underlying adapter
    result, err := a.adapter.Query(ctx, key)
    if err != nil {
        return nil, err
    }

    // Cache result
    a.cache[key] = CacheEntry{
        value:     result,
        timestamp: time.Now(),
    }

    return result, nil
}
```

---

## Testing Adapters

### Unit Tests

```go
func TestAdapter_HappyPath(t *testing.T) {
    adapter := NewAdapter(validConfig)
    result, err := adapter.Method(context.Background(), validInput)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}

func TestAdapter_InvalidInput(t *testing.T) {
    adapter := NewAdapter(validConfig)
    _, err := adapter.Method(context.Background(), invalidInput)
    assert.Error(t, err)
}

func TestAdapter_Timeout(t *testing.T) {
    adapter := NewAdapter(validConfig)
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
    defer cancel()
    _, err := adapter.Method(ctx, input)
    assert.Error(t, err)
}
```

### Integration Tests

```go
func TestAdapter_WithRealAPI(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    adapter := NewAdapter(&Config{
        APIKey: os.Getenv("API_KEY"),
    })

    result, err := adapter.Query(context.Background(), "test query")
    assert.NoError(t, err)
    assert.NotEmpty(t, result)
}
```

---

## Migration Checklist

When adding a new adapter:

- [ ] Create adapter file in `pkg/skill/adapters/`
- [ ] Implement core interface methods
- [ ] Add `IsAvailable()` method
- [ ] Support context timeouts
- [ ] Implement graceful degradation (if applicable)
- [ ] Write unit tests (happy path + error cases)
- [ ] Write integration tests (if applicable)
- [ ] Document configuration requirements
- [ ] Update MIGRATION.md with new adapter
- [ ] Add adapter to skill executor initialization
- [ ] Update skills to use new adapter

---

## FAQ

**Q: When should I create a new adapter vs. extending an existing one?**

A: Create a new adapter when the integration is substantially different (different API, different security model, different data model). Extend an existing adapter for minor variations (e.g., different endpoints on same API).

**Q: Should adapters be stateful or stateless?**

A: Prefer stateless adapters. If state is needed (e.g., caching, rate limiting), document it clearly and make it thread-safe.

**Q: How do I handle rate limiting?**

A: Implement rate limiting in the adapter using a token bucket or leaky bucket algorithm. Return clear errors when rate limit is exceeded.

**Q: What if the external API changes?**

A: Version your adapters. Create `WebToolAdapterV2` if breaking changes are needed, and migrate skills gradually.

---

## References

- **WebToolAdapter:** `pkg/skill/adapters/web_tools.go`
- **ScriptExecutor:** `pkg/skill/script_executor.go`
- **Contract:** `contracts/gateway-skills.md`
- **Testing:** `tests/skills/smoke_test.go`

---

**End of Guide**
