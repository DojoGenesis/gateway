# Implementation Commission: Pre-v1 TODO Hardening

**Objective:** Resolve the critical TODO items that must be addressed before tagging v1.0.0 — harden skill dependency validation, wire execution tracking, implement user tier lookup, and add a basic E2E smoke test to CI.

---

## 1. Context & Grounding

**Primary Specification:**
- This commission is self-contained. It targets 5 specific TODO comments across 4 files plus a CI workflow addition.

**Pattern Files (Follow these examples):**
- `skill/executor.go` lines 44-111 (`Execute` method): The existing skill execution flow. You will modify `validateDependencies` and its callers.
- `server/handlers/chat.go` lines 500-530: The user tier lookup area. Follow the existing handler patterns (Gin context, database adapter).
- `.github/workflows/ci.yml`: The existing CI workflow. Add a new job following the same structure as `build`, `security`, and `lint`.

**Files to Read:**
- `skill/executor.go` — Contains 3 of the 5 TODOs to resolve
- `server/orchestration/gateway_executor.go` — Contains execution tracking TODO
- `server/handlers/chat.go` — Contains user tier TODO
- `server/database/local_adapter.go` — Reference for database query patterns
- `.github/workflows/ci.yml` — CI workflow to extend

---

## 2. Detailed Requirements

### Phase 1: Harden Skill Dependency Validation (Steps 1-4)

1. In `skill/executor.go` line 55-57, change the `validateDependencies` call from warning-only to hard error:
   ```go
   // BEFORE (Phase 4a — warnings only):
   if err := e.validateDependencies(ctx, skill); err != nil {
       // TODO(Phase 4b): Make this a hard error
       _ = err
   }

   // AFTER (Phase 4b — hard errors):
   if err := e.validateDependencies(ctx, skill); err != nil {
       return nil, fmt.Errorf("skill '%s' has unmet dependencies: %w", skillName, err)
   }
   ```

2. In `skill/executor.go` line 118-121, replace the `web_tools` check TODO with a real check:
   ```go
   // TODO(Phase 4a): Check if web_tools adapter is loaded
   ```
   Replace with: Check if `e.toolInvoker` can resolve a web_tools function. If the skill's `RequiresWebTools()` returns true, call `e.toolInvoker.InvokeTool(ctx, "web_search", nil)` — if it returns an error containing "not found" or "not registered", return `fmt.Errorf("web_tools adapter required but not loaded")`.

3. In `skill/executor.go` line 124-127, replace the script executor check TODO:
   ```go
   // TODO(Phase 4a): Check if script executor is available
   ```
   Replace with: If `skill.RequiresScriptExecutor()` returns true (or if the skill tier >= 2 and has script triggers), check that `e.scriptExecutor` is non-nil. If nil, return `fmt.Errorf("script executor required but not available")`.

4. Remove the 3 TODO comments from `skill/executor.go` after implementing the checks. The code should be clean with no Phase 4a/4b references.

### Phase 2: Wire Execution Tracking (Steps 5-7)

5. In `server/orchestration/gateway_executor.go` line 84-85, replace the TODO with basic execution tracking:
   ```go
   // TODO: Implement execution tracking and cancellation
   // For now, return not implemented
   ```
   Replace with: Add a `context.WithCancel` wrapper around the execution context. Store the cancel function in a map (`map[string]context.CancelFunc`) keyed by execution ID. Return the execution ID to the caller. When the context is cancelled (client disconnect or explicit cancel), the cancel function propagates to all downstream operations.

6. Add a `Cancel(executionID string) error` method to the gateway executor that looks up the cancel function and calls it. Return `fmt.Errorf("execution not found: %s", executionID)` if the ID doesn't exist.

7. Add a `Status(executionID string) (string, error)` method that returns "running", "completed", "cancelled", or "not_found" by checking the context's `Err()` state.

### Phase 3: User Tier Lookup (Steps 8-9)

8. In `server/handlers/chat.go` line 522, replace the TODO with a database lookup:
   ```go
   // TODO: Implement proper user tier lookup from database
   ```
   Replace with: Call `h.db.GetSettings(ctx, userID)` to retrieve user settings. If the settings include a tier field, use it. If `GetSettings` returns `ErrCloudAdapterNotImplemented` or any error, fall back to `"free"` tier (current default behavior). This preserves backward compatibility while enabling proper tier resolution when the database has the data.

9. Add a brief comment explaining the fallback: `// Falls back to "free" tier when DB unavailable or user has no settings`.

### Phase 4: E2E Smoke Test in CI (Steps 10-12)

10. In `.github/workflows/ci.yml`, add a new job `e2e` after the `build` job:
    ```yaml
    e2e:
      runs-on: ubuntu-latest
      needs: build
      steps:
        - uses: actions/checkout@v4

        - name: Set up Go
          uses: actions/setup-go@v5
          with:
            go-version-file: go.mod

        - name: Build binary
          run: go build -o agentic-gateway main.go

        - name: Start gateway
          run: |
            ./agentic-gateway &
            sleep 3

        - name: Health check
          run: |
            curl -sf http://localhost:8080/health || exit 1

        - name: Verify API endpoints respond
          run: |
            # Check that /v1/chat/completions returns 401 (auth required) or 200
            STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/v1/chat/completions -X POST -H "Content-Type: application/json" -d '{"messages":[{"role":"user","content":"test"}]}')
            if [ "$STATUS" -eq "000" ]; then
              echo "Gateway not responding"
              exit 1
            fi
            echo "API responded with status $STATUS"

        - name: Verify admin endpoints
          run: |
            curl -sf http://localhost:8080/admin/metrics/prometheus || exit 1

        - name: Stop gateway
          if: always()
          run: pkill agentic-gateway || true
    ```

11. The E2E test must NOT depend on any external API keys or services. It validates that the binary starts, serves health, and responds on API endpoints — nothing more.

12. Remove the TODO comment about execution tracking from `gateway_executor.go` after implementing it.

---

## 3. File Manifest

**Modify:**
- `skill/executor.go` — Harden dependency validation (3 TODOs → real checks)
- `server/orchestration/gateway_executor.go` — Wire execution tracking + cancellation
- `server/handlers/chat.go` — User tier lookup from database
- `.github/workflows/ci.yml` — Add E2E smoke test job

**No files to create or delete.**

---

## 4. Success Criteria

- [ ] `skill/executor.go` returns an error (not a warning) when a Tier 2 skill loads without `web_tools` adapter
- [ ] `skill/executor.go` returns an error when a script-dependent skill loads without `scriptExecutor`
- [ ] Zero `TODO(Phase 4a)` or `TODO(Phase 4b)` comments remain in `skill/executor.go`
- [ ] `gateway_executor.go` supports `Cancel(executionID)` and `Status(executionID)`
- [ ] Zero `TODO: Implement execution tracking` comments remain in `gateway_executor.go`
- [ ] `chat.go` resolves user tier from database settings, falls back to `"free"` on error
- [ ] Zero `TODO: Implement proper user tier` comments remain in `chat.go`
- [ ] CI workflow includes an `e2e` job that starts the binary and validates health + API endpoints
- [ ] `go test ./skill/...` passes with updated validation logic
- [ ] `go test ./server/...` passes with updated handler and executor
- [ ] `go vet ./...` passes with zero warnings
- [ ] The build passes with zero errors

---

## 5. Constraints & Boundaries

- **DO NOT** modify `skill/registry.go` — only `executor.go` changes
- **DO NOT** change the skill tier definitions or the `RequiresWebTools()` / `RequiresScriptExecutor()` method signatures
- **DO NOT** add external service dependencies to the E2E test — it must work with zero API keys
- **DO NOT** modify the config reload TODO in `handle_admin.go` — that is deferred post-v1
- **DO NOT** modify the intent classifier config TODO — that is deferred post-v1
- **DO NOT** modify the per-model cost estimation TODO — that is deferred post-v1
- **DO NOT** introduce any new third-party dependencies
- **DO NOT** change the existing public API contract of any endpoint

---

## 6. Integration Points

- Skill executor changes affect all skill invocations. Tier 1 skills (portable, file_system only) should be unaffected since they don't require web_tools or script executor.
- Execution tracking integrates with `server/handle_gateway.go:handleGatewayOrchestrate` which starts async executions. The cancel/status methods enable future `/v1/gateway/orchestrate/:id/cancel` and `/v1/gateway/orchestrate/:id/status` endpoints (not in scope for this commission).
- User tier lookup uses the existing `database.Manager` interface. The `local_adapter` has `GetSettings` implemented. The `cloud_adapter` returns `ErrCloudAdapterNotImplemented` — the fallback handles this gracefully.

---

## 7. Testing Requirements

**Unit Tests:**
- Test `validateDependencies` returns error when web_tools missing for Tier 2 skill
- Test `validateDependencies` returns error when script executor missing for script-dependent skill
- Test `validateDependencies` passes for Tier 1 portable skill
- Test execution tracking: cancel, status, not-found
- Test user tier lookup with settings present, with settings absent, with DB error

**Edge Cases:**
- Skill with no dependencies (should always pass validation)
- Cancel an already-completed execution (should return "completed", not error)
- User with no settings record in database (should default to "free")
