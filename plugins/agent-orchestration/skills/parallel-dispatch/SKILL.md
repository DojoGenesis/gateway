---
name: parallel-dispatch
model: opus
description: "Produces a verified multi-track execution by dispatching parallel agents with file manifests, independent verification, and status tracking. Use when: \"dispatch agents\", \"parallel tracks\", \"multi-track work\", \"fan out\", \"divide and conquer\"."
category: agent-orchestration
triggers:
  - "dispatch agents in parallel"
  - "parallel tracks"
  - "multi-track work"
  - "fan out"
  - "divide and conquer"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: tracks
    type: array
    description: List of track descriptions, each with objective, scope, and expected output
    required: true
outputs:
  - name: dispatch_report
    type: text
    format: markdown-table
    description: Verified status table with per-track agent status, verification result, and files changed
---
# Parallel Dispatch Skill

## Philosophy

Parallel dispatch is not "fire and forget" — it is "fire, verify, and integrate." The value of parallelism comes from independent execution, but the risk comes from unverified output. Every agent is treated as an untrusted contractor: given a clear scope, expected to deliver, but verified independently before their work is accepted. The dispatch orchestrator's primary job is not delegation — it is verification.

## When to Use

- When 2+ independent tracks can execute concurrently without file conflicts
- For multi-module fixes, refactors, or feature additions across a workspace
- When time pressure demands parallel execution over sequential safety
- For commission-style work where each track has clear boundaries

## Workflow

### Phase 1: Plan
1. Parse tracks from user input
2. For each track, define:
   - Clear objective and scope
   - Specific file manifest (files the agent may touch)
   - Expected outputs and success criteria
   - Build/test commands to verify
3. Check for file conflicts — if two tracks touch the same file, use `isolation: "worktree"` or serialize them
4. Create a TodoWrite checklist with one item per track

### Phase 2: Dispatch
1. Dispatch agents in parallel — one per track, max 5 concurrent
2. Each agent prompt must include:
   - The full file manifest and success criteria
   - Instruction: "Run all tests before reporting completion"
   - The build/test commands that will be used to verify
3. Model routing: Sonnet for straightforward implementation, Opus for architectural/complex tracks

## File Manifest Protocol

Every agent prompt must include an explicit file manifest stating exactly which files the agent may create or modify. This is the zero-overlap guarantee that makes parallel execution safe.

**Agent Dispatch Template:**
```
"You own ONLY these files: [explicit list]. DO NOT read, modify, or create any files outside this list. If you need changes to files outside your manifest, report the need in your output — do not make the change."
```

**Ownership rules:**
- Agents that CREATE new files never conflict — new-file creation is inherently exclusive
- Agents that MODIFY existing files need exclusive ownership — one agent per file, no exceptions
- One agent per concern, not per file. Assign a complete feature slice (component + test + wiring), not a single file
- Main thread integrates: after all agents land, run `go build ./...`, `go test ./...`, wire cross-component imports, then one final build check

**Evidence:** validated across 20+ parallel dispatches in a single HTMLCraft Studio session — 3 phases, ~20 agents, 81 files committed, zero merge conflicts. The manifest is the mechanism: without explicit ownership, silent collisions are inevitable; with it, parallelism scales to any team size.

### Phase 3: Verify
After ALL agents complete, run independent verification for each track:
1. `git status` to confirm expected files were changed
2. `grep` for expected patterns in modified files
3. Run the build/test commands specified in Phase 1
4. Check for duplicate imports, wrong formats, or regressions

### Phase 4: Resolve
If any track fails verification:
1. Read the agent's actual output
2. Diagnose what went wrong
3. Fix directly or re-dispatch with corrected instructions
4. Re-verify after fix

### Phase 5: Report
Output a verified status table: track | agent status | verification status | files changed

## Output

The skill produces:
1. A per-track execution plan with file manifests
2. Parallel agent dispatches with scoped prompts
3. A verified status table with independent verification results
4. Commits per track (if requested)

## Examples

**3-track dispatch with clean results:**
```
/dispatch
Track A: Fix auth middleware in AgenticGateway
Track B: Add unit tests to dojo-cli /init command
Track C: Update STATUS.md across all DojoGenesis repos

> Phase 1: 3 tracks, no file conflicts
> Phase 2: 3 agents dispatched (A: sonnet, B: sonnet, C: sonnet)
> Phase 3: All 3 verified green
> RESULT: 3/3 tracks complete, 12 files changed
```

**Dispatch with one failure:**
```
/dispatch
Track A: Consolidate 2 protocol handlers into 1
Track B: Add JWT validation to bridge.go

> Phase 2: 2 agents dispatched (A: opus, B: sonnet)
> Phase 3: Track A verified green, Track B failed (wrong import path)
> Phase 4: Fixed Track B import, re-verified green
> RESULT: 2/2 tracks complete after 1 remediation
```

## Edge Cases

- **File conflict between tracks**: Serialize the conflicting tracks or use worktree isolation. Never let two agents write to the same file.
- **Agent reports completion but files unchanged**: Treat as failure. Re-dispatch with explicit instruction to persist changes.
- **Agent fix breaks another track's work**: Run integration tests after all tracks complete. Resolve conflicts before committing.
- **More than 5 tracks**: Batch into waves of 5. Complete and verify wave 1 before dispatching wave 2.
- **Track depends on another track's output**: Do not parallelize dependent tracks. Execute the dependency first, verify, then dispatch the dependent track.

## Anti-Patterns

- **Trusting agent self-reports**: The most common failure mode. Always verify with `git status`, `grep`, and build/test commands.
- **No file manifest**: Without explicit file boundaries, agents may conflict or touch unrelated code.
- **Dispatching without a plan**: Jumping straight to dispatch without defining objectives, manifests, and success criteria leads to vague agent output.
- **Using Opus for simple tasks**: Model routing matters for cost and speed. Straightforward implementation gets Sonnet; architectural decisions get Opus.
- **Committing before verification**: Every track must pass independent verification before any commit is made.
- **Re-dispatching without diagnosing**: When a track fails, read the output first. The fix may be a one-line correction, not a full re-dispatch.

## Model Enforcement

As of Apr 14, 2026, a PostToolUse hook (`agent-model-enforce.sh`) warns whenever an agent is dispatched without an explicit `model:` parameter. The rule is non-negotiable: always specify `model: "sonnet"` or `model: "opus"` in every agent call — never inherit the default.

**Routing split:**
- `model: "sonnet"` — parsing, bulk transforms, template generation, audits, straightforward implementation (~80% of dispatches)
- `model: "opus"` — architecture decisions, synthesis of competing constraints, strategic calls (~20% of dispatches)

Inheriting the default is an anti-pattern because the default may change, may differ by environment, and makes cost and quality unpredictable across a large swarm. Explicit model selection is part of the dispatch contract.
