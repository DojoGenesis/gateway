---
name: audit-sweep-dispatch
description: Routes health-audit findings to the correct execution path — main thread, foreground fix, or parallel agent waves. Use when health-audit findings need to be sorted by effort and dispatched as agent waves.
model: sonnet
category: agent-orchestration
version: 1.0.0
tags: [audit, dispatch, sweep, triage, agents]
triggers:
  - "route health audit findings"
  - "dispatch audit findings as agent waves"
  - "sort audit by effort"
  - "triage health findings"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: findings
    type: array
    description: Health-audit findings with severity (RED/YELLOW/GREEN) and effort estimate
    required: true
  - name: build_commands
    type: string
    description: Build and test commands to use for verification (e.g. "go build ./... && go test ./...")
    required: false
outputs:
  - name: dispatch_report
    type: text
    format: markdown-table
    description: Summary of what was fixed in-thread, dispatched to agents, and deferred
---
# Audit-Sweep-Dispatch Skill

## Philosophy

A health audit is a pre-sorted dispatch plan. GREEN findings need no action. RED findings block all other work. YELLOW findings split cleanly: trivial ones are faster to fix in-thread than to delegate, substantial ones parallelize across packages. The value of this skill is recognizing which bucket each finding belongs in and routing it correctly — not treating every finding as equally deserving of an agent.

The hardest mistake to catch is dispatching an agent for a 4-line edit. The second hardest is fixing a RED item in a side thread while kicking off Wave 1 in parallel. This skill encodes both rules mechanically.

## When to Use

- After running `/health-audit` or any audit that produces RED/YELLOW/GREEN findings
- When a sweep produces more findings than you can hold in working memory
- When a codebase has both quick wins (trivial) and real work (substantial) mixed in the same audit output
- When you want parallel coverage raises without merge conflicts

## Workflow

### Phase 1: Triage

Receive the audit findings and sort into four buckets:

1. **GREEN** — Passing, no action. Acknowledge and move on. Do not dispatch agents for things that already pass.
2. **RED / P0** — Blocking. Fix in the main thread foreground before anything else is dispatched. These are the findings that break builds, fail security checks, or block Wave 1 from running at all.
3. **YELLOW / Trivial** — Doc changes, comment fixes, marker updates, `[DONE]` stamps, single-line tweaks. Fix directly in the main thread. Rule: if the fix is under 10 lines and touches only one file, it is trivial. Never launch an agent for a trivial fix.
4. **YELLOW / Substantial** — New test files, new config, multi-line code changes, coverage additions, handler rewrites. These go to agents.

Output a triage table before dispatching anything:

```
| Finding | Severity | Effort | Routing |
|---------|----------|--------|---------|
| Missing test coverage in internal/client | YELLOW | Substantial | Agent Wave 1 |
| Stale TODO marker in Makefile | YELLOW | Trivial | Main thread |
| Build fails: wrong import path | RED | Substantial | Foreground now |
```

### Phase 2: RED Fixes First

Fix all RED findings in the main thread before dispatching any agents. Verify each RED fix with the full build/test command before proceeding. Do not start Wave 1 until the build is green.

### Phase 3: Trivial Sweep (Main Thread)

Work through the trivial YELLOW items directly. These are faster to do than to explain to an agent. Keep a running checklist. When all trivials are done, commit or stage them if appropriate.

### Phase 4: Wave Planning

Group the substantial YELLOW findings into waves by file independence:

- Each agent in a wave must touch non-overlapping files. Test packages are the canonical case: `internal/client/`, `internal/commands/`, `internal/providers/` can all run in parallel because each writes to its own `*_test.go`.
- Cap at 3 concurrent agents per wave. Diminishing returns above that.
- If Wave 2 depends on Wave 1 output (e.g., a test agent that needs a new handler to exist), Wave 2 is blocked until Wave 1 verifies.

Wave plan format:
```
Wave 1 (3 agents, concurrent):
  - Agent A: internal/client — add missing test coverage
  - Agent B: internal/commands — add edge case tests for /init
  - Agent C: ci.yml + installer.go — fix security lint finding

Wave 2 (2 agents, after Wave 1 verified):
  - Agent D: internal/providers — add provider failover tests
  - Agent E: internal/telemetry — add span coverage
```

### Phase 5: Dispatch Waves

Dispatch Wave 1 agents using parallel-dispatch conventions:
- Give each agent an explicit file manifest (only the files in their scope)
- Include the build/test command they must run before reporting done
- Include the finding text so the agent understands what they are fixing

### Phase 6: Verify Each Wave

After all agents in a wave complete, verify each independently before launching the next wave:

1. `git status` — confirm the expected files were changed
2. `grep` for expected patterns (new test function names, fixed import paths)
3. Run the build/test command scoped to the changed package
4. Confirm the coverage number or pass count matches the agent's claim

Do not trust agent self-reports. An agent that reports "complete" with no files changed in `git status` has not completed.

### Phase 7: Report

Produce a final dispatch report:

```
| Item | Routing | Status | Verification |
|------|---------|--------|-------------|
| Build fix: import path | RED → main thread | DONE | go build green |
| Makefile TODO marker | Trivial → main thread | DONE | in-thread |
| internal/client coverage | Wave 1 Agent A | DONE | go test: 78%→91% |
| internal/commands coverage | Wave 1 Agent B | DONE | go test: 65%→83% |
| ci.yml security lint | Wave 1 Agent C | DONE | lint clean |
| internal/providers tests | Wave 2 Agent D | DONE | go test: 4 new cases |
| internal/telemetry spans | Wave 2 Agent E | FAILED | no files changed — re-dispatch |
```

## Examples

**Example 1: Post-audit dispatch for dojo-cli sweep (Apr 10)**

Audit produced: 1 RED (import path), 4 YELLOW substantial (test coverage across 4 packages), 3 YELLOW trivial (Makefile comment, TODO.md, improvement-plan markers).

Routing:
- RED: Fixed import path in main thread. Build confirmed green.
- Trivials: Updated 3 files directly. 8 lines total. No agents.
- Wave 1 (3 agents): security+CI | client tests | commands tests — all concurrent, non-overlapping files
- Wave 2 (2 agents, after Wave 1 verified): providers tests | telemetry tests

Result: 2-wave dispatch, 5 agents total, zero file conflicts, all verified green.

**Example 2: Gateway audit with architectural RED**

Audit found: 1 RED (missing D1Syncer injection causing nil panic at startup), 2 YELLOW substantial (new handler stubs in handle_documents.go), 1 YELLOW trivial (stale port reference in README comment).

Routing:
- RED: D1Syncer injection was architectural — fixed in foreground with Opus before anything else. Took 45 minutes. Build confirmed clean.
- Trivial: README port reference updated in-thread (1 line).
- Wave 1 (1 agent): handle_documents.go handler stubs — single agent, single file, no conflicts.

Note: Even though Wave 1 was just 1 agent, the wave structure is still correct — the D1Syncer RED had to be resolved first or the agent's new handlers would have panicked at test time.

## Edge Cases

- **RED finding is architectural**: Do not rush it. Architectural RED items may need Opus-level reasoning. Fix carefully in the foreground before any wave begins.
- **Two substantial findings touch the same file**: Serialize them — fix one in-thread or in Wave 1 alone, then the second in Wave 2. Never let two agents write the same file.
- **Agent returns no changes**: Treat as failure. Check `git status`. Re-dispatch with more explicit instructions including the exact line numbers to change.
- **Wave 2 reveals a new RED**: Stop. Fix the new RED before continuing. Wave 2 agents may have uncovered a dependency the audit missed.
- **More than 9 substantial findings**: Break into 3+ waves. Never exceed 3 concurrent agents. Verify each wave before the next.

## Anti-Patterns

- **Dispatching an agent for a trivial fix**: If the change is under 10 lines in one file, do it yourself. The agent briefing overhead exceeds the work.
- **Starting Wave 1 before RED is resolved**: The most common cascade failure. RED items break builds, and broken builds make all agent test results meaningless.
- **Trusting agent self-reports**: Always verify with `git status` and the build command. Agents report "done" on partial work more often than you expect.
- **Skipping the triage table**: Dispatching without first categorizing all findings leads to missed items, duplicate coverage, and agents touching overlapping files.
- **Re-dispatching without reading the failure output**: When an agent fails, read what it actually did before re-dispatching. The fix is often a one-line correction to the instructions, not a full re-run.
