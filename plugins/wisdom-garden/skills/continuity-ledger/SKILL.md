---
name: continuity-ledger
model: sonnet
description: "Maintains a running decision log and context state across agent sessions, preventing context loss between clears and session boundaries. Use when: 'starting a new session on an ongoing project', 'context is getting heavy', 'preserve state before clearing', 'create a handoff document', 'pick up where I left off'."
category: wisdom-garden
triggers:
  - "starting a new session on an ongoing project"
  - "context is getting heavy"
  - "preserve state before clearing"
  - "create a handoff document"
  - "pick up where I left off"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: session_context
    type: string
    description: Description of the current session's work, decisions made, and state to preserve
    required: true
  - name: prior_ledger
    type: ref
    format: cas-ref
    description: Reference to a previous ledger entry to build upon (for multi-session chains)
    required: false
outputs:
  - name: ledger
    type: ref
    format: cas-ref
    description: YAML handoff document capturing session state, decisions, findings, and next steps
---
# Continuity Ledger

**Purpose:** Create structured YAML handoff documents that preserve session state -- decisions made, files modified, findings discovered, approaches tried, and concrete next steps -- so that a fresh session can resume work without context loss. Each clear/restart gets a clean context with full signal instead of degraded compacted context.

---

## I. When to Use

- Before running `/clear` when context usage is approaching 70%+
- At the end of a multi-day implementation session
- During complex refactors you expect to pick up and put down
- Any session expected to hit 85%+ context utilization
- When switching between projects and need to freeze state for later

### When NOT to Use

- Quick tasks under 30 minutes
- Simple single-file bug fixes
- Sessions that will complete without approaching context limits

---

## II. Why Clear + Ledger Instead of Compact

Each compaction is lossy compression. After several compactions, you are working with degraded context -- key decisions and file paths silently disappear. Clearing and loading the ledger gives a fresh context with full signal from the structured handoff document.

---

## III. Workflow

### Step 1: Determine Ledger Location

Check for existing handoff folders to maintain session continuity:

```bash
ls -td thoughts/shared/handoffs/*/ 2>/dev/null | head -1 | xargs basename
```

- If a handoff folder exists, use that session name
- If none exist, use `general` as the folder name
- File path: `thoughts/shared/handoffs/{session-name}/YYYY-MM-DD_HH-MM_description.yaml`

### Step 2: Capture Session State

Gather the following from the current session:

1. **Goal** -- what this session set out to accomplish (one line, shown in statusline)
2. **Now** -- what the next session should do first (one line, shown in statusline)
3. **Test command** -- a single command to verify the work (`go test ./...`, `npm test`, etc.)
4. **Done this session** -- completed tasks with file references
5. **Decisions** -- choices made and their rationale
6. **Findings** -- key learnings and discoveries
7. **Worked/Failed** -- approaches to repeat vs. avoid
8. **Next steps** -- concrete action items for the next session
9. **Files** -- created and modified file lists

### Step 3: Write YAML Handoff

Write the ledger using this exact format (field names are parsed by the statusline -- do not rename them):

```yaml
---
session: {session-name}
date: YYYY-MM-DD
status: complete|partial|blocked
outcome: SUCCEEDED|PARTIAL_PLUS|PARTIAL_MINUS|FAILED
---

goal: {What this session accomplished}
now: {What next session should do first}
test: {Verification command}

done_this_session:
  - task: {Completed task description}
    files: [file1.py, file2.py]

blockers: [{any blocking issues}]

questions: [{unresolved questions for next session}]

decisions:
  - {decision_name}: {rationale}

findings:
  - {key_finding}: {details}

worked: [{approaches that worked}]
failed: [{approaches that failed and why}]

next:
  - {First next step}
  - {Second next step}

files:
  created: [{new files}]
  modified: [{changed files}]
```

### Step 4: Confirm and Mark Outcome

Ask the user for their assessment of the session outcome:
- **SUCCEEDED** -- task completed successfully
- **PARTIAL_PLUS** -- mostly done, minor issues remain
- **PARTIAL_MINUS** -- some progress, major issues remain
- **FAILED** -- task abandoned or blocked

Record the outcome in the YAML frontmatter.

---

## IV. Resuming from a Ledger

To resume in a new session:

1. Load the most recent handoff YAML
2. Read the `goal:` and `now:` fields for immediate orientation
3. Run the `test:` command to verify the baseline still holds
4. Review `decisions:` and `findings:` to restore context
5. Start executing from `next:` list

---

## V. Output

- A YAML handoff document at `thoughts/shared/handoffs/{session-name}/YYYY-MM-DD_HH-MM_description.yaml`
- Fields: session metadata, goal, now, test command, completed tasks with file refs, decisions with rationale, findings, worked/failed approaches, next steps, file manifest
- Target size: ~400 tokens (compact enough to load without significant context cost)

---

## VI. Examples

**Scenario 1:** 3-hour refactor session approaching 75% context --> Ledger captures: goal="Migrate event system from callbacks to channels", now="Wire remaining 3 handlers in server/events.go", 5 completed tasks with 12 files, 2 architectural decisions (chose channels over callbacks because of backpressure support), 1 blocker (circular import in pkg/bus), 4 next steps.

**Scenario 2:** Multi-day feature across sessions --> First ledger: goal="Add WebSocket hub to gateway", now="Implement broadcast logic", outcome=PARTIAL_PLUS. Second session loads ledger, completes broadcast, writes new ledger: goal="WebSocket hub broadcast complete", now="Add integration tests for hub reconnection", outcome=SUCCEEDED.

**Scenario 3:** Session hit a dead end --> Ledger captures: outcome=PARTIAL_MINUS, failed=["Attempted to use SSE for bidirectional communication -- protocol is server-to-client only"], decisions=["Switch to WebSocket for bidirectional requirement"], now="Start fresh with WebSocket approach using the ws package".

---

## VII. Edge Cases

- No prior handoffs exist: create the `thoughts/shared/handoffs/general/` directory and start the first ledger
- Session was purely exploratory with no code changes: still write the ledger capturing findings and decisions; set status to `complete` with outcome `SUCCEEDED` and files sections empty
- Multiple active session chains: use distinct session-name folders to keep ledgers separated by workstream
- Prior ledger referenced but file is missing: note the gap in the new ledger's findings and reconstruct what is known from git log and current file state

---

## VIII. Anti-Patterns

- Writing the ledger after context is already degraded from multiple compactions -- write it early, at 70% context, not after the damage is done
- Using alternative field names (session_goal, objective, focus, current) -- the statusline parser expects exactly `goal:` and `now:`
- Putting implementation details in the ledger instead of references -- the ledger points to files, it does not duplicate their contents
- Skipping the outcome marking step -- the outcome field drives session analytics and helps identify patterns in what causes partial/failed sessions
