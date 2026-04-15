---
name: session-compression
model: sonnet
description: "Produces updated memory artifacts by compressing a session's key decisions, changes, and context into the structured memory garden. Use when: \"compress this session\", \"save context\", \"end of session\", \"update memory\", \"wrap up\"."
category: wisdom-garden
triggers:
  - "compress this session"
  - "save context"
  - "end of session"
  - "update memory"
  - "wrap up"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: memory_path
    type: string
    description: Path to the memory directory containing MEMORY.md and memory files
    required: true
outputs:
  - name: compression_report
    type: text
    format: markdown
    description: Summary of memories updated, created, and open items carried forward
---
# Session Compression Skill

## Philosophy

Memory is not a log — it is a garden. Session compression distills what happened into what matters for future sessions. The goal is not to record every action but to preserve the decisions, insights, and context that would otherwise be lost when the conversation ends. Good compression makes the next session start faster; bad compression creates noise that slows it down.

## When to Use

- At the end of every substantial work session (3+ meaningful changes)
- After architectural decisions that affect future work
- When switching between major project areas
- Before a long break between sessions

## Workflow

### Phase 1: Gather Changes
1. Run `git diff` and `git status` across the parent repo and all active sub-repos
2. Identify files changed, tests run, and their outcomes
3. Note any tools or services configured or reconfigured

### Phase 2: Extract Decisions
1. Identify key decisions made during the session — architectural choices, convention changes, approach pivots
2. For each decision, capture the *why* (not just the *what*)
3. Identify any outstanding issues, blockers, or unresolved friction

### Phase 3: Update Memory
1. Check existing memory files for entries that need updating based on this session's work
2. For each key insight or decision:
   - If an existing memory covers this topic, update it in place
   - If it's genuinely new, create a new memory file with proper frontmatter (name, description, type)
3. Update MEMORY.md index if any files were added or removed
4. Update the "Open Items" section in MEMORY.md with current state

### Phase 4: Report
Output a summary: memories updated, memories created, open items carried forward

## Output

The skill produces:
1. Updated or new memory files with YAML frontmatter
2. An updated MEMORY.md index
3. A compression summary listing what was persisted

## Examples

**After a gateway refactor session:**
```
/compress
> Updated: project_gateway_consolidation_plan.md (17→16 modules)
> Created: seed_two_handler_trap.md (new pattern discovered)
> Open items: protocol merge (2→1) still pending
> MEMORY.md: 142 lines (under 200 limit)
```

**After a debugging session with no architectural changes:**
```
/compress
> Updated: project_dojochat.md (v0.4 syntax highlighting shipped)
> No new memories needed — fix was mechanical
> Open items unchanged
```

## Edge Cases

- **Session with no meaningful changes**: Report "no compression needed" rather than creating empty memories
- **Memory file conflicts**: If two sessions update the same memory, merge by keeping the more recent facts and preserving the *why*
- **MEMORY.md approaching 200 lines**: Consolidate related entries or inline short memories before adding new links
- **Relative dates in user messages**: Always convert to absolute dates when saving ("Thursday" becomes "2026-04-10")
- **Session that only read code**: Reading sessions rarely produce memories — only save if a non-obvious insight was discovered

## Anti-Patterns

- **Saving session logs as memory**: Memory files are not activity reports. "Read 5 files and ran 3 tests" is not a memory.
- **Duplicate memories**: Always check existing files before creating new ones. Update, don't duplicate.
- **Saving code patterns as memory**: If it's derivable from the code itself, it doesn't belong in memory.
- **Overly verbose descriptions**: Memory descriptions are used for relevance matching. Keep them specific and under 150 characters.
- **Forgetting to update Open Items**: Stale open items cause future sessions to chase resolved issues.
