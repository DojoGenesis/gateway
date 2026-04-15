---
name: gap-audit-then-fix
model: sonnet
description: "Spec-to-fix pipeline: audit a codebase against its specification, classify gaps by severity, fix in parallel, track in GAPS.md registry. Use when a spec exists but implementation completeness is unknown."
category: specification-driven-development
version: 1.0.0
tags: [audit, gaps, specification, fix, registry]
triggers:
  - "audit gaps against spec"
  - "spec to fix pipeline"
  - "implementation completeness unknown"
  - "fix gaps in parallel"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: spec_path
    type: string
    description: Path to the specification document (or inline spec text) to audit against
    required: true
  - name: repo_path
    type: string
    description: Path to the codebase to audit
    required: true
outputs:
  - name: gaps_registry
    type: string
    description: GAPS.md file written to the repo root, containing all findings with severity and status
---
# Gap Audit Then Fix

## Philosophy

Writing a spec reveals which gaps matter. Auditing the spec against the codebase reveals the gaps the spec missed. Fixing without both steps misses the second class — including security issues.

In one concrete session: spec writing surfaced 8 gaps. A subsequent deep audit of the same codebase found 12 more, including a critical unauthenticated admin endpoint. That security gap would never have appeared in a "fix as you go" workflow because no one thought to look.

The pipeline is linear for a reason. Specs come first because they define which code artifacts must exist. Audit comes second because only a complete requirements map exposes missing coverage. Classification comes third because not all gaps are equal — fixing P2 cosmetics while P0 security gaps are open is waste. The GAPS.md registry is the output that makes all future sessions cheaper: known issues are never rediscovered.

## When to Use

Activate this skill when:

- A specification exists (even partial) but you do not know how complete the implementation is
- Handing off to an implementation agent and you want to surface blockers first
- Preparing for a release and need a confidence gate
- Post-audit: a previous codebase-audit-grounding run produced measurements and now you need to act on them
- A codebase has grown beyond a single author's mental model and silent gaps are likely

Do not use when:

- No spec exists — write the spec first using the `release-specification` or `specification-writer` skill
- The codebase is greenfield with no existing implementation to audit
- The task is exploratory measurement only — use `codebase-audit-grounding` for that

## Pipeline Overview

```
Spec Review → Deep Audit → Classify → Parallel Fix → Registry
    (Step 1)    (Step 2)    (Step 3)    (Step 4)      (Step 5)
```

Step 2 (Deep Audit) overlaps with what `codebase-audit-grounding` produces. If you have already run that skill, use its output as the measurement baseline for Step 2 and skip re-running the counts.

---

## Step 1: Spec Review

Read the full specification. For every requirement, extract the implied code artifact.

For each requirement, record:
- What the spec says must exist
- What file or type of artifact satisfies it (endpoint, middleware, type, migration, test, config entry)
- Whether the artifact is testable/verifiable

A requirement like "GET /api/v1/documents returns paginated results" implies:
- A route registration
- A handler function
- Pagination parameters (`page`, `limit` or cursor)
- A response envelope with `total`, `items`, `next_cursor`
- At minimum one test

Write this extracted list before opening the codebase. The list is your audit checklist. Deriving it from the spec rather than from the code prevents confirmation bias.

## Step 2: Deep Audit

For each item on the checklist from Step 1, verify it exists and is correct. Use grep and file reads — do not infer from directory structure alone.

**Route verification:**
```bash
grep -r "router\.\|r\.GET\|r\.POST\|r\.PUT\|r\.DELETE\|http\.HandleFunc" --include="*.go" .
```

**Auth middleware on sensitive routes:**
```bash
grep -A 5 "admin\|internal\|private" --include="*.go" -r . | grep -E "middleware|auth|token"
```

**Pagination parameter handling:**
```bash
grep -n "page\|limit\|cursor\|offset" handler_file.go
```

**Type/struct existence:**
```bash
grep -rn "type DocumentResponse\|DocumentResponse struct" --include="*.go" .
```

**Test coverage:**
```bash
grep -rn "TestGet\|TestCreate\|TestUpdate\|TestDelete" --include="*_test.go" .
```

**CORS and header configuration:**
```bash
grep -rn "Access-Control\|cors\|CORS" --include="*.go" .
```

For each checklist item, record a gap entry immediately when the item is missing or incorrect:

```
{
  requirement: "GET /api/v1/documents returns paginated results",
  expected: "handler with page/limit params and pagination envelope",
  actual: "handler exists but returns raw slice with no pagination",
  severity: P1
}
```

Do not wait until all auditing is done to record gaps — record as you find them.

## Step 3: Classify

Assign every gap a priority before touching any code:

| Priority | Criteria | Examples |
|----------|----------|---------|
| P0 | Blocks release: missing endpoint, broken contract, security vulnerability, data loss risk | Unauthenticated admin endpoint; required field missing from response type; migration not applied |
| P1 | Degrades quality: missing tests, incomplete error handling, wrong HTTP status codes, missing pagination | Handler returns 500 for input validation failure; no test for the delete path; list endpoint returns full dataset |
| P2 | Cosmetic or low-impact: naming inconsistency, missing doc comment, log message formatting | Handler named `handleDocs` instead of `handleDocuments`; missing godoc on exported function |

Classification rules:
- Security gaps are always P0, regardless of exploitability likelihood
- Missing tests on critical paths (auth, delete, billing) are P0, not P1
- When uncertain between P0 and P1, choose P0 — over-severity is cheaper than under-severity

## Step 4: Parallel Fix

**P0 gaps: fix on main thread immediately.** Do not dispatch P0s to agents — they are blocking and require your full attention to verify correctly.

**P1 gaps: group by file independence, dispatch as parallel agents.** Two P1 gaps in the same file must be fixed sequentially or by the same agent; two P1 gaps in different files can be parallel. Each agent receives exactly the gap entry (requirement, expected, actual) plus the file path. Agents should not receive the full spec — only the specific gap they own.

Agent dispatch format:
```
Fix this specific gap in [file_path]:
  Requirement: [requirement text]
  Expected: [expected behavior]
  Actual: [current behavior]
  Constraint: Do not modify any file outside [file_path] and its *_test.go pair.
```

After each agent completes, verify independently with `git diff` and a targeted test run. Do not trust agent self-reports.

**P2 gaps: defer.** Record in GAPS.md with status DEFERRED. Do not fix P2 gaps in the same session as P0/P1 work — the context switch cost is higher than the value.

## Step 5: Registry

Write all findings to `GAPS.md` at the repo root. This file is the living audit trail.

Format:

```markdown
# GAPS.md — Gap Registry

Last audit: [date]
Spec: [spec file or version]

## P0 — Release Blockers

| ID | Requirement | Expected | Actual | Status | Fixed In |
|----|-------------|----------|--------|--------|---------|
| G-001 | Admin endpoints require auth | AuthMiddleware on all /admin/* routes | GET /admin/users has no middleware | FIXED | abc1234 |
| G-002 | Document type includes `owner_id` | `owner_id string` in DocumentResponse | Field absent from struct | FIXED | def5678 |

## P1 — Quality Gaps

| ID | Requirement | Expected | Actual | Status | Fixed In |
|----|-------------|----------|--------|--------|---------|
| G-003 | GET /documents returns paginated results | page/limit params, pagination envelope | Raw slice returned | FIXED | ghi9012 |
| G-004 | Delete handler returns 404 for missing doc | 404 with error body | Returns 200 with empty body | DEFERRED | — |

## P2 — Deferred

| ID | Requirement | Expected | Actual | Status |
|----|-------------|----------|--------|--------|
| G-005 | Handler naming convention | handleDocuments | handleDocs | DEFERRED |
```

Append to GAPS.md across sessions. Do not delete WONTFIX or DEFERRED rows — the registry's value comes from showing the history of what was found and decided, not just what is open.

---

## Examples

**Example 1 — Security gap discovery (Gateway audit session)**

Spec stated: "All /admin routes are protected by AdminAuth middleware."

Audit step: grep all route registrations and trace the middleware chain.

```bash
grep -n "admin" server/routes.go
```

Output showed `r.GET("/admin/users", s.handleAdminUsers)` registered directly on the base router, not on the admin middleware group. The handler existed; the protection did not.

Gap entry:
```
requirement: "All /admin routes protected by AdminAuth middleware"
expected: AdminAuth middleware applied to /admin/users
actual: Route registered on base router; no middleware
severity: P0
```

Fixed in main thread before any other work continued. Verified by reading the final route registration and running the auth test suite. Appended to GAPS.md as G-001 with status FIXED and commit SHA.

**Example 2 — Normal gap audit (Document API spec)**

Spec stated: "GET /api/v1/documents returns paginated results with `page`, `limit`, `total`, and `items` fields."

Audit steps:
1. Grep for the route — found, handler exists.
2. Read the handler — returns `[]Document` directly, no envelope struct.
3. Grep for `DocumentListResponse` — not found.
4. Check test file — test asserts on array length, not envelope fields.

Four gap entries created (missing envelope struct, missing query param parsing, wrong response shape, incomplete test assertion), all classified P1.

Grouped into one agent dispatch since all four touch the same handler file and its test pair. Agent produced the envelope struct, updated handler, updated test. Verified with `go build ./...` and `go test ./server/...`. All four marked FIXED in GAPS.md.

---

## Anti-Patterns

**Fixing P2 before P0.** If the registry has an unauthenticated admin endpoint open and you are renaming a handler for naming consistency, you have inverted the priority order. P0 gaps block release; P2 gaps do not. Fixing P2 first is not wrong by accident — it is the natural pull of easy wins. Resist it.

**Auditing without a spec.** Grep-and-read without a requirements checklist produces a list of observations, not a list of gaps. A gap is defined relative to a requirement. Without the spec, you cannot distinguish "this is incomplete" from "this was intentionally omitted." You will fix things that should not be fixed and miss things that must be fixed.

**Skipping the registry.** Without GAPS.md, every new session starts cold. Gaps found in session N are rediscovered in session N+3. Security gaps from two weeks ago get reopened by agents who were not told they existed. The registry is not overhead — it is what makes the pipeline terminate rather than loop.

**Dispatching P0s to agents.** P0 gaps are blocking by definition. Delegating them introduces verification latency (you must check the agent's work before proceeding) and risk (the agent may not have full context on why the gap is critical). Fix P0s yourself, immediately.

**Marking a gap FIXED without running tests.** A code change that satisfies the grep check but breaks an integration test is not fixed. The status FIXED in the registry means: code change applied, build passes, relevant test suite passes, change verified in git diff.

**Running the audit from a feature branch.** The audit reflects the branch state, not main. Gaps you "fix" on the branch that already exist in main will create false confidence. Always audit from main or the target release branch.

---

## Quality Checklist

- [ ] Spec was read fully before opening the codebase
- [ ] Every requirement extracted as a verifiable checklist item
- [ ] Audit used grep/read on actual source — no inference from directory structure
- [ ] Every gap recorded with: requirement, expected, actual, severity
- [ ] Security gaps classified P0 regardless of perceived exploitability
- [ ] P0 gaps fixed on main thread before any other work
- [ ] P1 gaps grouped by file independence before agent dispatch
- [ ] Each agent given exactly one gap and one file scope
- [ ] Agent work verified independently with `git diff` and test run
- [ ] GAPS.md written to repo root with all findings, statuses, and commit SHAs

---

## Related Skills

- **codebase-audit-grounding**: Step 2 of this pipeline in isolation. Use it when the goal is measurement only (before writing the spec). This skill picks up where grounding leaves off.
- **release-specification**: Produces the spec this skill audits against. Run it before this skill if no spec exists.
- **implementation-prompt**: Converts individual gaps into high-quality agent dispatch prompts when the fix is complex enough to warrant more than a one-line task.
- **parallel-tracks**: When P1 fix volume is high (10+ gaps across many files), use parallel-tracks to structure the dispatch wave formally rather than ad-hoc.
- **pre-commission-alignment**: Reactive counterpart — used when implementation is already in progress and alignment gaps are found mid-build. Gap Audit Then Fix is proactive; pre-commission-alignment is corrective.
