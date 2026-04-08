# Pre-Implementation Checklist — Consolidated Pre-Flight Report
## Agentic Gateway by Dojo Genesis v0.1.0

**Date:** 2026-02-12
**Author:** Cruz + Claude
**Status:** Cross-Spec Alignment Review

---

## Executive Summary

After reading all four track specifications and cross-referencing their type definitions, module paths, and assumptions, I identified **10 cross-spec inconsistencies** that must be resolved before commissioning implementation agents. Seven are mechanical fixes (dates, versions, paths). Three require architectural alignment decisions.

---

## Inconsistencies Found

### CRITICAL — Architectural Conflicts

#### 1. Module Path Divergence

The specs disagree on the canonical module path convention.

| Spec | Convention | Example |
|------|-----------|---------|
| Track 0 | `agentic-gateway/` prefix | `github.com/DojoGenesis/gateway/provider` |
| Track 1 | No prefix | `github.com/dojo-genesis/provider` |
| Track 2 | No prefix | `github.com/dojo-genesis/orchestration` |
| Track 3 | Old monolith paths | `github.com/TresPies-source/dojo-genesis/go_backend/...` |

**Resolution:** Track 0 is the foundation spec and defines the canonical convention. All tracks must use `github.com/DojoGenesis/gateway/*`. This also matches the Go workspace layout (`go.work` with subdirectory modules). Tracks 1, 2, and 3 must be updated.

#### 2. ModelProvider Interface Conflict

Track 0's shared types define a 3-method interface. Track 1 defines a 6-method interface that matches the actual monolith codebase.

| Spec | Methods | Signature Style |
|------|---------|----------------|
| Track 0 (shared) | Call, Stream, Config | Simplified — `Call(ctx, messages) (Message, error)` |
| Track 1 (provider) | GetInfo, ListModels, GenerateCompletion, GenerateCompletionStream, CallTool, GenerateEmbedding | Full — matches actual codebase |

**Resolution:** Track 0's shared types were written as thin abstractions. Track 1's interface is the real contract that matches the codebase. Two options:

- **Option A (Recommended):** Remove `ModelProvider` from shared types entirely. Let the `provider` module own its interface. Other modules that need a provider reference import `provider.ModelProvider` directly. This avoids a stale abstraction in shared/.
- **Option B:** Update shared types to match Track 1's 6-method interface exactly. Risk: shared/ becomes a bottleneck for provider changes.

#### 3. Tool Type Conflict

Track 0 defines `Tool` as an interface (4 methods). Track 1 defines `ToolDefinition` as a struct with fields.

| Spec | Type | Approach |
|------|------|----------|
| Track 0 (shared) | `Tool` interface | `Name() string`, `Description() string`, `Execute(ctx, input) (ToolResult, error)` |
| Track 1 (tools) | `ToolDefinition` struct | Fields: Name, Description, Parameters, Function, Timeout |

**Resolution:** Same pattern as the provider conflict. The tools module should own its types. Track 0's shared `Tool` interface should either be removed or thinned to a minimal `Executable` interface that orchestration can use without importing the full tools module. Recommend removing from shared/ and letting tools/ own `ToolDefinition`.

#### 4. Events Module Placement

Track 0 places events inside the server module. Track 1 defines events as a separate module with its own `go.mod`.

| Spec | Location | Rationale |
|------|----------|-----------|
| Track 0 | `server/events.go` | Events are a streaming concern |
| Track 1 | `events/` (standalone module) | Events are shared across provider, tools, and orchestration |

**Resolution:** Track 1 is correct. Events need to be emitted from provider, tools, AND orchestration — not just server. If events lived in server/, those modules would need to import server/, creating a circular dependency. Events should either be a standalone module (`agentic-gateway/events`) or live in `shared/`. Recommend: standalone `events/` module per Track 1.

This means Track 0's module count increases from 5+1 to 5+2 (shared + events).

---

### HIGH — Incorrect Values

#### 5. Date Errors in Track 1

Track 1 header states:
- "Release Date: Q1 2025" → should be **Q1 2026**
- "Last Updated: 2025-02-12" → should be **2026-02-12**

#### 6. Go Version Inconsistencies

| Spec | Go Version Stated | Correct |
|------|------------------|---------|
| Track 0 | Go 1.24 | ✅ |
| Track 1 | Dockerfile: `golang:1.21` | ❌ → `golang:1.24` |
| Track 2 | "Minimum: Go 1.19, Tested: 1.20-1.22" | ❌ → Go 1.24 |
| Track 3 | Dockerfile: `golang:1.21-alpine` | ❌ → `golang:1.24-alpine` |

The monolith uses Go 1.24. All modules must use 1.24.

#### 7. Track 3 Uses Old Monolith Import Paths

Track 3's server.go code examples (lines 1095-1100, 1440-1446) still reference:
```
github.com/TresPies-source/dojo-genesis/go_backend/...
```

These must be updated to:
```
github.com/DojoGenesis/gateway/...
```

---

### MEDIUM — Reference Errors

#### 8. Track 3 References Non-Existent Track 4

Track 3 header links to `[Track 4](./track_4_agentic_framework.md)` which does not exist.
Track 3 footer (line 1854-1855) says "Next Steps (Track 4)" — Track 4 is out of scope for v0.1.0.

**Fix:** Remove Track 4 link from header. Change footer to "Future Work" section instead of "Next Steps (Track 4)".

#### 9. Track 3 Has Wrong Filenames in Header Links

Line 7:
```
[Track 1](./track_1_provider_abstraction.md) | [Track 2](./track_2_orchestration.md)
```

Correct filenames:
```
[Track 1](./track_1_provider_tools_spec.md) | [Track 2](./track_2_orchestration_memory_spec.md)
```

#### 10. Track 1 Claims "v1.0+ Stability"

Track 1 header says `API Version: 1.0` and `Stability: Stable`. The overall project is v0.1.0. This is misleading — these are initial contracts, not stable v1.0 APIs.

**Fix:** Change to `API Version: 0.1.0` and `Stability: Pre-release`.

---

## Resolution Summary

| # | Issue | Fix | Tracks Affected |
|---|-------|-----|-----------------|
| 1 | Module paths | Standardize to `github.com/DojoGenesis/gateway/*` | 1, 2, 3 |
| 2 | ModelProvider | Remove from shared/; provider/ owns the interface | 0, 1 |
| 3 | Tool type | Remove from shared/; tools/ owns ToolDefinition | 0, 1 |
| 4 | Events module | Standalone module, not part of server/ | 0, 1 |
| 5 | Dates | Fix to 2026 | 1 |
| 6 | Go version | Standardize to 1.24 | 1, 2, 3 |
| 7 | Old imports in code | Update TresPies-source paths | 3 |
| 8 | Track 4 reference | Remove or change to "Future Work" | 3 |
| 9 | Wrong filenames | Fix header links | 3 |
| 10 | Version claim | Change to v0.1.0 pre-release | 1 |

---

## Architectural Decisions (for Track 0 shared/ update)

Based on the resolution of issues 2, 3, and 4, the `shared/` module's role changes from "define all core interfaces" to "define cross-cutting types that don't belong in any single module." Specifically:

**shared/ KEEPS:** Message, ToolResult, TaskStatus, error types, constants
**shared/ LOSES:** ModelProvider interface → owned by provider/; Tool interface → owned by tools/; Event interface → owned by events/

This makes shared/ lighter and reduces coupling. Each module owns its primary interface, and the shared module provides the common currency types (Message, ToolResult) that flow between modules.

The dependency graph updates to:

```
shared        → (standalone; only imports stdlib)
events        → shared (for Message type)
provider      → shared, events
tools         → shared, events
orchestration → shared, tools (for ToolDefinition), events
memory        → shared
server        → shared, provider, tools, orchestration, memory, events
```
