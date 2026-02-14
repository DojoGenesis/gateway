# L1: Adopt respondError Helper Consistently

**Priority:** Low
**Effort:** Medium (~2-3 hours)
**Impact:** Code consistency, standardized error responses

## Context

The codebase has a `respondError()` and `respondValidationError()` helper defined in `server/handlers/types.go`, but they're only used in 2 files (`memory_seeds.go` and the types.go file itself). Most handlers use inline error responses with `c.JSON(statusCode, gin.H{"error": "..."})`.

**Current state:**
```go
// handlers/types.go - DEFINED BUT RARELY USED
func respondError(c *gin.Context, statusCode int, message string, details ...string) {
	resp := ErrorResponse{
		Error: message,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	c.JSON(statusCode, resp)
}

func respondValidationError(c *gin.Context, fieldErrors map[string]string) {
	c.JSON(400, ErrorResponse{
		Error:  "Validation failed",
		Code:   "VALIDATION_ERROR",
		Fields: fieldErrors,
	})
}
```

**Typical inline pattern (used in 16 files):**
```go
c.JSON(http.StatusInternalServerError, gin.H{
	"error": "failed to process request",
})
```

## Benefits of Migration

1. **Consistent error structure** - All error responses use `ErrorResponse` type with optional `Details`, `Code`, and `Fields`
2. **Type safety** - Compiler catches malformed error responses
3. **Easier testing** - Mock/assert against structured error types
4. **Better client parsing** - Clients can rely on consistent JSON structure
5. **Support for validation errors** - Built-in field-level error reporting

## Files to Migrate (16 handlers)

Based on grep results, these files have inline `c.JSON(http.Status...)` error responses:

```
server/handlers/user_preferences.go
server/handlers/agents.go
server/handlers/memory.go
server/handlers/models.go
server/handlers/trace.go
server/handlers/artifacts.go
server/handlers/tools.go
server/handlers/search_conversations.go
server/handlers/migration.go
server/handlers/api_keys.go
server/handlers/chat.go
server/handlers/projects.go
server/handlers/broadcast.go
server/handlers/health.go
server/handlers/metrics.go
```

**Already using helpers:**
- `memory_seeds.go` ✓
- `types.go` (defines them) ✓

## Migration Strategy

### Phase 1: Analyze Error Patterns

For each file, categorize error responses:

1. **Simple errors** (message only):
   ```go
   // BEFORE
   c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})

   // AFTER
   respondError(c, http.StatusBadRequest, "invalid request")
   ```

2. **Errors with details** (message + context):
   ```go
   // BEFORE
   c.JSON(http.StatusBadRequest, gin.H{
       "error": "invalid format",
       "details": fmt.Sprintf("expected JSON, got %s", contentType),
   })

   // AFTER
   respondError(c, http.StatusBadRequest, "invalid format",
       fmt.Sprintf("expected JSON, got %s", contentType))
   ```

3. **Validation errors** (field-level):
   ```go
   // BEFORE
   c.JSON(http.StatusBadRequest, gin.H{
       "error": "validation failed",
       "fields": map[string]string{
           "email": "invalid format",
           "age": "must be >= 18",
       },
   })

   // AFTER
   respondValidationError(c, map[string]string{
       "email": "invalid format",
       "age": "must be >= 18",
   })
   ```

4. **Errors with codes** (extend helper if needed):
   ```go
   // If code support is needed, extend respondError:
   func respondErrorWithCode(c *gin.Context, statusCode int, code, message string) {
       c.JSON(statusCode, ErrorResponse{
           Error: message,
           Code:  code,
       })
   }
   ```

### Phase 2: File-by-File Migration

**Order of migration (low-risk to high-risk):**

1. **Start with simple handlers** (fewer errors, less logic):
   - `health.go` (1-2 errors)
   - `metrics.go` (1-2 errors)
   - `broadcast.go` (2-3 errors)

2. **Mid-complexity handlers** (moderate error handling):
   - `user_preferences.go`
   - `search_conversations.go`
   - `tools.go`

3. **Complex handlers** (many errors, complex logic):
   - `agents.go`
   - `projects.go`
   - `artifacts.go`
   - `migration.go`
   - `api_keys.go`

4. **Critical path handlers** (touch last, test thoroughly):
   - `chat.go` (streaming + SSE errors)
   - `memory.go` (core functionality)
   - `trace.go` (observability)

### Phase 3: Extend Helpers (If Needed)

Based on patterns found during migration, you may need:

```go
// handlers/types.go additions

// respondErrorWithCode for errors that need semantic codes
func respondErrorWithCode(c *gin.Context, statusCode int, code, message string, details ...string) {
	resp := ErrorResponse{
		Error: message,
		Code:  code,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	c.JSON(statusCode, resp)
}

// respondUnauthorized - common 401 wrapper
func respondUnauthorized(c *gin.Context, message string) {
	respondErrorWithCode(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// respondNotFound - common 404 wrapper
func respondNotFound(c *gin.Context, resource string) {
	respondErrorWithCode(c, http.StatusNotFound, "NOT_FOUND",
		fmt.Sprintf("%s not found", resource))
}

// respondConflict - common 409 wrapper
func respondConflict(c *gin.Context, message string) {
	respondErrorWithCode(c, http.StatusConflict, "CONFLICT", message)
}
```

## Special Cases

### 1. SSE/Streaming Errors (chat.go)

SSE error events have a different format — **do not migrate these**:
```go
// Keep this pattern for SSE - NOT a regular HTTP response
c.SSEvent(string(streaming.Error), gin.H{
	"error": errorMsg,
})
```

### 2. Success Responses with Embedded Errors

Some handlers return 200 with `{"success": false, "error": "..."}`. Migration decision:

**Option A:** Keep current pattern (backwards compatibility)
```go
c.JSON(http.StatusOK, gin.H{
	"success": false,
	"error":   "operation failed",
})
```

**Option B:** Use proper HTTP status codes (breaking change)
```go
// Breaking change - clients expect 200 with success:false
respondError(c, http.StatusBadRequest, "operation failed")
```

**Recommendation:** Keep Option A for v0.3.0 to maintain backwards compatibility. Consider Option B for v1.0.0 with API versioning.

### 3. Custom Error Structures

Some handlers have rich error responses:
```go
c.JSON(http.StatusBadRequest, gin.H{
	"error":    "validation failed",
	"code":     "INVALID_INPUT",
	"field":    "email",
	"expected": "valid email format",
	"got":      input,
})
```

For these, extend `ErrorResponse` with optional fields or create handler-specific error types.

## Testing Strategy

### 1. Add Tests for Error Helpers

```go
// handlers/types_test.go
func TestRespondError(t *testing.T) {
	// Test basic error
	// Test error with details
	// Test different status codes
	// Test nil context (should not panic)
}

func TestRespondValidationError(t *testing.T) {
	// Test field errors
	// Test empty field map
	// Test multiple fields
}
```

### 2. Handler-Level Integration Tests

For each migrated handler, verify:
- Error responses have correct status codes
- Error structure matches `ErrorResponse` type
- Details field populated when provided
- Validation errors use correct format

### 3. Regression Testing

Before/after comparison:
```bash
# Capture current error responses
go test ./server/handlers/... -json > before.json

# After migration
go test ./server/handlers/... -json > after.json

# Diff (should show only structural changes, not behavior changes)
diff before.json after.json
```

## Verification Checklist

- [ ] All 16 handler files migrated
- [ ] SSE errors excluded (chat.go streaming)
- [ ] Success-with-error patterns preserved (if applicable)
- [ ] No new error patterns introduced
- [ ] All existing tests pass
- [ ] New tests added for error helpers
- [ ] Error response format documented in API docs
- [ ] `go vet ./...` passes
- [ ] `go build ./...` succeeds

## Rollback Plan

If migration causes issues:
1. Revert commits (git revert)
2. Keep helper functions (they're still useful)
3. Migrate incrementally file-by-file in smaller PRs

## Example Migration (agents.go)

**Before:**
```go
func HandleGetAgent(c *gin.Context) {
	if agentManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "agent manager not initialized",
		})
		return
	}

	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "agent_id is required",
		})
		return
	}

	agent, err := agentManager.GetAgent(c.Request.Context(), agentID)
	if errors.Is(err, agent.ErrAgentNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "agent not found",
		})
		return
	}
	if err != nil {
		slog.Error("failed to get agent", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, agent)
}
```

**After:**
```go
func HandleGetAgent(c *gin.Context) {
	if agentManager == nil {
		respondError(c, http.StatusInternalServerError, "agent manager not initialized")
		return
	}

	agentID := c.Param("id")
	if agentID == "" {
		respondError(c, http.StatusBadRequest, "agent_id is required")
		return
	}

	agent, err := agentManager.GetAgent(c.Request.Context(), agentID)
	if errors.Is(err, agent.ErrAgentNotFound) {
		respondNotFound(c, "agent")
		return
	}
	if err != nil {
		slog.Error("failed to get agent", "error", err)
		respondError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, agent)
}
```

## Estimated Timeline

- **Phase 1 (Analysis):** 30 minutes
- **Phase 2 (Migration):** 2-3 hours
- **Phase 3 (Testing):** 1 hour
- **Total:** ~4 hours

## Success Metrics

- All error responses use structured `ErrorResponse` type
- Zero new gin.H{"error": "..."} patterns introduced
- Error helper coverage: 100% of handlers
- Test coverage for error helpers: ≥90%
