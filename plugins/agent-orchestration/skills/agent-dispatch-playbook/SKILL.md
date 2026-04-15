---
name: agent-dispatch-playbook
model: sonnet
description: "Produces a dispatch plan — isolation model, agent count, sequencing, and model assignments — for multi-agent parallel work. Use when: \"dispatch agents in parallel\", \"run multiple tracks simultaneously\", \"plan parallel agent strategy\", \"should this be foreground or background\"."
triggers:
  - "dispatch agents in parallel"
  - "orchestrate parallel work across repos"
  - "plan agent delegation strategy"
metadata:
  version: "1.0"
  created: "2026-04-07"
  author: "Tres Pies Design"
  tool_dependencies: ["meta_skill", "bash", "file_system"]
  portable: true
  tier: 1
  agents: ["agent-orchestration"]
category: agent-orchestration
tier: 1
agents:
  - "primary"

inputs:
  - name: work_description
    type: string
    description: Description of the multi-agent parallel work to plan
    required: true
  - name: track_count
    type: number
    description: Target number of parallel tracks (default auto-selected)
    required: false
outputs:
  - name: dispatch_plan
    type: string
    description: Dispatch plan with isolation model, agent count, sequencing, and model assignments
---
# Agent Dispatch Playbook

## I. Philosophy

Agent orchestration is not about launching as many agents as possible — it's about matching the **isolation model** to the **dependency graph**. Today's session (Apr 7, 2026) dispatched 15 agents across 7 repos, completing in hours what would take days sequentially. The patterns below are extracted from that lived experience, not theory.

The core insight: **the main thread is for strategy, agents are for heavy lifting.** The main thread reads results, makes judgment calls, resolves conflicts, and sequences dependent work. Agents do the grunt work in parallel.

## II. When to Use

- Multi-repo changes that touch independent file sets
- Health audits, documentation refreshes, or test fixes across repos
- Any task that decomposes into 2+ independent tracks
- When you need to verify work before proceeding to the next phase

## III. The Five Dispatch Patterns

### Pattern 1: Parallel Execution Tracks (Worktree Isolation)

**When:** Multiple code changes in the SAME repo that touch DIFFERENT packages/directories.

**How:**
- Launch each track as an Agent with `isolation: "worktree"`
- Each gets its own git branch in `.claude/worktrees/`
- Tracks MUST touch non-overlapping files
- After completion, merge worktree branches into main

**Example from today:** Gateway Track 2 (workflow/executor.go, channel/nats_bus.go) and Track 3 (server/handle_websocket.go) ran in parallel worktrees — different packages, clean merge.

**Gotcha:** Worktree changes are uncommitted by default. You must copy files to main and manually apply shared-file changes (server.go, router.go) if both tracks modify them.

```
Agent(isolation="worktree", prompt="Track A: modify workflow/ and channel/")
Agent(isolation="worktree", prompt="Track B: modify server/ only")
# After both complete:
# 1. Copy new files from worktrees to main
# 2. Cherry-pick shared-file edits manually
# 3. Verify: go build ./... && go test ./...
```

### Pattern 2: Cross-Repo Parallel (Background Agents)

**When:** Changes in DIFFERENT repos with zero dependencies between them.

**How:**
- Launch all agents with `run_in_background: true`
- Each agent works in its own repo directory — no isolation needed
- Main thread continues with non-overlapping work
- Results arrive as task notifications

**Example from today:** HTMLCraft integration (TypeScript) ran alongside Gateway workflow wiring (Go) — completely independent repos.

```
Agent(prompt="Fix tests in repo A", run_in_background=true)
Agent(prompt="Fix tests in repo B", run_in_background=true)
Agent(prompt="Add feature in repo C", run_in_background=true)
# Main thread: update documentation or plan while waiting
```

### Pattern 3: Audit Swarm (Read-Only Parallel)

**When:** Gathering information across multiple repos without making changes.

**How:**
- Launch 2-4 audit agents in background, each with a different scope
- Agents READ and REPORT but don't modify files (or save reports to non-overlapping paths)
- Main thread compiles the unified report from all agent results

**Example from today:** 3 health audit agents (Gateway, 4 smaller repos, HTMLCraft) ran simultaneously. Each produced findings; main thread compiled the unified scorecard.

**Scaling rule:** 3 concurrent audit agents is the sweet spot. More than 4 starts hitting diminishing returns from context overhead.

```
Agent(subagent_type="health-auditor", prompt="Audit repo A", run_in_background=true)
Agent(subagent_type="health-auditor", prompt="Audit repo B+C+D", run_in_background=true)
Agent(subagent_type="health-auditor", prompt="Audit repo E", run_in_background=true)
# Wait for all 3, then compile unified report
```

### Pattern 4: Sequential Fix Agent

**When:** Fixing issues that were discovered by an audit, where fixes have dependencies.

**How:**
- Run in foreground (not background) because you need the result before proceeding
- Give the agent the EXACT findings from the audit (file paths, line numbers, expected values)
- Verify the fix with build/test before moving to the next dependent task

**Example from today:** After audit found 19 SKILL.md files missing triggers, a single fix agent updated all 19 + verified `go test ./tests/skills/...` passed.

**Key principle:** Fix agents get specific instructions, not open-ended research tasks. The audit already did the research — the fix agent just executes.

```
# BAD: "Find and fix test failures" (too open-ended)
# GOOD: "Update loader_test.go line 163: change 15 to 31. Update line 357: change 'debugging' to 'debugging-troubleshooting'. Verify: go test ./internal/skills/..."
```

### Pattern 5: Phased Pipeline (Blocker → Parallel → Verify)

**When:** Some work must complete before parallel work can begin.

**How:**
1. **Phase A:** Fix blockers in foreground (build errors, missing deps)
2. **Phase B:** Launch parallel agents for independent tracks
3. **Phase C:** Merge results, verify combined build
4. **Phase D:** Launch dependent work (things that needed Phase B outputs)

**Example from today:**
- Phase A: Fix 4 duplicate method build errors (foreground, 5 min)
- Phase B: Launch 3 parallel agents (workflow wiring, WebSocket hub, HTMLCraft integration)
- Phase C: Copy worktree files, verify `go build && go test` (foreground)
- Phase D: Push to DojoGenesis, run health audits

```
# Phase A — foreground, blocks everything
fix_build_errors()
verify_build()

# Phase B — parallel, independent
Agent("Track 2: workflow", isolation="worktree", run_in_background=true)
Agent("Track 3: websocket", isolation="worktree", run_in_background=true)
Agent("Track 4: htmlcraft", run_in_background=true)

# Phase C — after all Phase B agents complete
merge_worktrees()
verify_combined_build()

# Phase D — depends on Phase C
push_to_remotes()
run_health_audits()
```

## IV. Decision Matrix

| Situation | Pattern | Isolation | Background? |
|-----------|---------|-----------|-------------|
| 2+ changes in same repo, different dirs | Worktree Tracks | worktree | Yes |
| Changes across different repos | Cross-Repo Parallel | none | Yes |
| Read-only audits across repos | Audit Swarm | none | Yes |
| Fixing specific known issues | Sequential Fix | none | No (foreground) |
| Mix of blocking + parallel work | Phased Pipeline | mixed | Mixed |

## V. Prompt Engineering for Agents

### What makes a good agent prompt:
1. **State the goal** in the first line
2. **List exact file paths** the agent should modify
3. **Specify the verification command** (`go test ./...`, `npm test`, etc.)
4. **Name what NOT to touch** (prevents overlap with other agents)
5. **Include the commit message** if the agent should commit

### What makes a bad agent prompt:
- "Based on your findings, fix the bug" — pushes synthesis onto the agent
- "Investigate and implement" — two different tasks crammed together
- Omitting file paths — agent wastes time exploring
- Not specifying verification — agent can't confirm its own work

## VI. Model Routing

### Frontmatter Convention

Every SKILL.md MUST declare `model: sonnet` or `model: opus` in YAML frontmatter. This is the authoritative routing signal — when the Dojo Gateway or a dispatch agent invokes a skill, it reads this field to select the model. Skills without a `model:` field default to the parent session's model, which wastes Opus tokens on mechanical work.

### Routing Decision Matrix

| Task Type | Model | Why |
|-----------|-------|-----|
| Planning, judgment, architecture | Opus | Needs reasoning depth |
| Strategy, synthesis, pattern recognition | Opus | Ambiguity requires judgment |
| Parsing, sed, template application | Sonnet | Fast, mechanical work |
| Fix agents with exact instructions | Sonnet | Instructions are complete, just execute |
| Normalization, auditing, bulk edits | Sonnet | Deterministic operations |
| Health audits, code exploration | Sonnet | Read-heavy, report-focused |
| Deep research, retrospectives | Opus | Synthesis across sources |

### Target Split: 60% Sonnet / 40% Opus

The cost-optimal ratio for a solo operator running 200+ Agent calls per session. When in doubt, default to Sonnet — most agent work is mechanical once the main thread has made the judgment call.

### Enforcement

1. **New skills:** `model:` field is required in YAML frontmatter. Skill audit flags missing fields.
2. **Agent dispatch:** When calling the Agent tool, specify `model: "sonnet"` or `model: "opus"` explicitly. Do not rely on parent session default.
3. **Audit gate:** The nightly skill audit checks that all first-party skills declare a model. Missing `model:` is a quality defect.

**Rule of thumb:** If the prompt contains all file paths and exact changes, use Sonnet. If the agent needs to make judgment calls, use Opus.

## VII. Anti-Patterns

1. **Launching agents for trivial tasks** — If it's a single `sed` command or one file edit, just do it in the main thread.
2. **Duplicating work** — If you delegate research to an agent, don't also search the same files yourself.
3. **More than 4 parallel agents** — Diminishing returns. 3 is the sweet spot.
4. **Worktrees for shared files** — If two tracks both need to modify `server.go`, don't use worktrees. Do one track, then the other, or split the file changes.
5. **Background agents for blocking work** — If you need the result before proceeding, run foreground.

## VIII. Metrics from April 7, 2026 Session

| Metric | Value |
|--------|-------|
| Total agents dispatched | 15 |
| Max concurrent agents | 3 |
| Execution tracks (worktree) | 2 |
| Cross-repo parallel agents | 4 |
| Audit swarm agents | 3 |
| Fix agents | 4 |
| Documentation agents | 4 |
| Repos touched | 7 |
| Files modified | 500+ |
| New tests added | 78 |
| Time saved (estimated) | 3-4x vs sequential |

## Output

- Named dispatch pattern (Worktree Tracks, Cross-Repo Parallel, Audit Swarm, Sequential Fix, or Phased Pipeline)
- Agent prompt templates with: goal, file scope, verification command, files to avoid
- Model assignment for each agent (sonnet or opus) with rationale
- Sequencing diagram or phase list when the work has blockers
- Decision Matrix row populated for the current task

## Examples

**Scenario 1:** "Run health audits across 5 repos while I continue coding" → Audit Swarm pattern; 3 background agents (repos grouped by size), each writing findings to `/tmp/audit-<repo>.md`; main thread continues foreground work and compiles unified report on completion.

**Scenario 2:** "Wire the workflow executor and the WebSocket hub in the same repo" → Phased Pipeline; Phase A fixes any build errors foreground; Phase B launches two worktree agents (workflow/ and server/) in parallel; Phase C merges worktrees and runs `go build && go test`; shared file edits (router.go) applied manually after merge.

## Edge Cases

- Two planned tracks both modify a shared file (e.g., `server.go`): do not use worktrees; run one track foreground, then the second, applying shared-file changes sequentially
- Agent task definition is ambiguous at dispatch time: do not dispatch; resolve definition in the main thread first, then dispatch with exact file paths and verification command
- More than 4 parallel agents requested: cap at 3-4 active concurrently; queue remaining tracks to start as earlier agents complete

## Anti-Patterns

- Launching agents for tasks that are a single `sed` command or one-file edit — main thread is faster
- Omitting file paths from agent prompts — agent wastes tokens on exploration
- Using worktree isolation when two tracks both need to touch the same file — guarantees a merge conflict
- Dispatching background agents for blocking work — if you need the result before proceeding, run foreground
- Assigning Opus to a fix agent with complete exact instructions — mechanical work belongs on Sonnet
