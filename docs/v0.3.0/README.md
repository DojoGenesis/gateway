# v0.3.0 Refactoring Tasks

This directory contains comprehensive prompts for the L1, L2, and L3 refactoring tasks deferred from the architecture audit cleanup (v0.2.0).

## Quick Reference

| Task | Priority | Effort | Impact | Status |
|------|----------|--------|--------|--------|
| **L1: respondError Helper** | Low | ~4 hours | Code consistency | 📋 Documented |
| **L2: Options Struct** | Medium | ~3-4 hours | Extensibility | 📋 Documented |
| **L3: Handler Structs** | Medium-High | ~13-16 hours | Testability | 📋 Documented |

## Task Overview

### L1: Adopt respondError Helper Consistently
**File:** `L1-adopt-respondError-helper.md`

**Summary:** Migrate 16 handler files from inline `c.JSON(status, gin.H{"error": "..."})` to structured `respondError()` helpers for consistent error responses.

**Why defer?** Nice-to-have improvement, not critical. Current inline errors work fine.

**When to do?** Include in a general code cleanup sprint or when adding comprehensive API documentation.

---

### L2: Server Constructor → Options Struct
**File:** `L2-server-options-struct.md`

**Summary:** Replace 18-parameter `New()` constructor with a `ServerDeps` struct for better extensibility and testing.

**Why defer?** The current constructor works but is hard to extend. This becomes important as the codebase grows.

**When to do?** **Recommended for v0.3.0.** The 18-parameter constructor is a code smell that will make future development harder. Do this before adding more server dependencies.

---

### L3: Handler Struct Pattern
**File:** `L3-handler-struct-pattern.md`

**Summary:** Refactor handlers from package-level global variables to handler structs with explicit dependency injection.

**Why defer?** Significant refactor touching route registration and all handler files. High effort, but important for testability.

**When to do?** **Do this BEFORE writing comprehensive handler unit tests.** The global variable pattern makes testing fragile and unsafe for parallel execution. If you plan to add thorough handler test coverage, do L3 first.

## Recommended Execution Order

### Option A: Incremental (Safest)
Do tasks one at a time with full testing between each:

1. **v0.3.0:** L2 (Server options struct) — Medium effort, high value
2. **v0.3.1:** L3 (Handler structs) — High effort, critical for testing
3. **v0.3.2:** L1 (respondError) — Low effort, polish

### Option B: Focused Testing Sprint
If planning to add comprehensive test coverage:

1. **First:** L3 (Handler structs) — Enables proper unit testing
2. **Second:** L2 (Server options struct) — Makes test setup easier
3. **Last:** L1 (respondError) — Polish while writing tests

### Option C: Defer All
If not adding extensive tests or extending server deps:

- Defer L1, L2, L3 to v0.4.0+
- Current architecture is functional (just not optimal)
- Focus on features instead of refactoring

## Priority Assessment

### Critical Path: L3 > L2 > L1

**L3 is most important** because:
- Blocks proper handler unit testing
- Global variables cause race conditions in parallel tests
- Implicit dependencies make debugging harder
- Already partially migrated (`models.go` uses struct pattern)

**L2 is second priority** because:
- 18-parameter constructor gets worse as server grows
- Harder to add new dependencies without this
- Makes testing harder (must provide all 18 params)

**L1 is lowest priority** because:
- Purely cosmetic (consistency improvement)
- No functional benefit
- Can be done anytime

## Dependencies Between Tasks

```
L3 (Handler Structs)
  ↓
L2 (Server Options)  ← Makes handler initialization cleaner
  ↓
L1 (respondError)    ← Easier to migrate with better test coverage
```

**Note:** Tasks are independent but synergistic. Doing L3 first makes L2 more valuable (cleaner handler setup), and both make L1 easier (better tests catch regressions).

## Effort vs. Impact

```
High Impact
    │
    │   L3 ■
    │      (testability)
    │
    │         L2 ■
    │            (extensibility)
Medium
    │
    │
    │
    │                  L1 ■
Low │                     (consistency)
    └────────────────────────────────
      Low    Medium    High
              Effort
```

## What Was Already Completed (v0.2.0)

All medium-priority fixes from the architecture audit:

- ✅ **M1:** Fixed err.Error() leaks in apptools (15+ instances)
- ✅ **M2:** Fixed unsafe type assertions in chat.go and trace.go
- ✅ **M3:** Applied SQLite tuning to server database
- ✅ **M4:** Added security headers middleware
- ✅ **M5:** Orchestration module extraction (completed by another agent)

And simple low-priority items:

- ✅ **L4:** Added HSTS header in production
- ✅ **L5:** Renamed Message → ContextMessage in context_builder.go

## Current Status

**Build status:** ✅ Clean
```bash
go build ./...  # Success
go vet ./...    # No issues
go test ./...   # All passing
```

**Code quality:**
- No err.Error() leaks to clients
- No unsafe type assertions
- SQLite properly tuned
- Security headers in place
- Ready for production

**Technical debt:**
- L1-L3 deferred to v0.3.0
- Not blocking, but recommended for long-term maintainability

## Questions?

If you're unsure which task to prioritize:

1. **Planning to add handler tests?** → Do L3 first
2. **Adding more server dependencies?** → Do L2 first
3. **Just want cleaner code?** → Do L1 (lowest priority)
4. **Focused on features?** → Defer all to v0.4.0+

Each prompt file includes:
- Detailed context and rationale
- Step-by-step migration strategy
- Code examples (before/after)
- Testing strategy
- Rollback plans
- Validation checklists
- Estimated timelines

All tasks are **safe to execute** with proper testing at each step.
