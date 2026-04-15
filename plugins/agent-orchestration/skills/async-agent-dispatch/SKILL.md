---
name: async-agent-dispatch
model: sonnet
description: "Produces a structured background-dispatch handoff — task description, output contract, success criteria, and timeout — so long-running work completes without blocking the user. Use when: \"run this in the background\", \"dispatch agent for long task\", \"I need this to run async\", \"parallel agent dispatch\"."
category: agent-orchestration
triggers:
  - "run this in the background"
  - "dispatch agent for long task"
  - "I need this to run async"
  - "parallel agent dispatch"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: task_description
    type: string
    description: Description of the long-running task to dispatch
    required: true
  - name: expected_duration_minutes
    type: number
    description: Estimated task duration in minutes
    required: false
outputs:
  - name: dispatch_plan
    type: string
    description: Structured handoff block with task description, output contract, success criteria, and timeout
---
# Async Agent Dispatch

## I. Philosophy

The user's attention is the scarcest resource. Long tasks that block interaction waste
it. Background dispatch turns sequential bottlenecks into parallel throughput -- but
only if the handoff is clean enough that results integrate without re-explanation.

The key insight is the control plane / data plane separation: the foreground agent
manages intent and coordination (control plane), while background agents execute
work and produce artifacts (data plane). A clean boundary between them means neither
blocks the other.

This is not about making things faster. It is about making long work invisible to
the user until results are ready.

## II. When to Use

- Task will take more than 2 minutes (large codebase scans, multi-file generation, test suites)
- User needs to continue interacting while work proceeds
- Multiple independent tasks can run in parallel
- Task produces file artifacts that can be read after completion
- Work is well-defined enough to run without mid-task clarification

Do NOT use when:
- Task requires iterative user feedback during execution
- Task is under 2 minutes (overhead of dispatch exceeds benefit)
- Task modifies shared state that the foreground agent also reads
- Results need real-time streaming to the user (use foreground with progress updates)
- Task definition is ambiguous and likely to need clarification

## III. Workflow

### Step 1: Classify Task Duration

Before dispatching, estimate whether background execution is warranted:

| Duration | Mode | Rationale |
|----------|------|-----------|
| < 30 seconds | Foreground, inline | Dispatch overhead exceeds task time |
| 30s - 2 min | Foreground, with progress | User can wait if they see movement |
| 2 - 10 min | Background, single agent | Clear win for async dispatch |
| 10+ min | Background, consider splitting | May benefit from parallel tracks |

Heuristics for estimation:
- File count * 0.5s per file for read-heavy tasks
- Test count * 2s per test for test suites
- API calls * latency per call for network-bound work

### Step 2: Choose Dispatch Mode

**Option A: Bash with `run_in_background=true`**
Best for: shell commands, build processes, test suites, scripts.
The command runs in background and you receive a notification when complete.

**Option B: Agent tool (sub-agent)**
Best for: complex multi-step tasks requiring tool access and judgment.
Structure the agent prompt with explicit scope, constraints, and output location.

**Option C: Parallel dispatch (multiple background tasks)**
Best for: independent workstreams that don't share state.
Dispatch 2-4 tasks simultaneously. Track each by its output location.

### Step 3: Structure the Handoff

Every background dispatch must include these elements:

```
TASK DESCRIPTION:
  What: [Specific deliverable in 1-2 sentences]
  Scope: [Explicit boundaries -- what IS and IS NOT included]

OUTPUT CONTRACT:
  Location: [Exact file path where results will be written]
  Format: [Expected structure -- JSON, markdown, code files]

SUCCESS CRITERIA:
  - [Measurable criterion 1]
  - [Measurable criterion 2]

TIMEOUT:
  Maximum: [Duration in minutes]

CONSTRAINTS:
  - Do not modify files outside [scope]
  - Do not require user interaction
  - Write all output to [location] before exiting
```

Critical rule: the output location must be agreed before dispatch. If you don't know
where to look for results, the background work is wasted.

### Step 4: Monitor Without Polling

After dispatch, do NOT poll in a sleep loop. Instead:

- For Bash `run_in_background`: you will receive an automatic notification when the
  command completes. Continue other work in the foreground.
- For Agent sub-tasks: the agent will complete and results will be at the agreed
  output location. Check only when notified or when the user asks.
- For parallel dispatches: track each task's output path. Check them in order of
  expected completion time.

Tell the user what was dispatched and when to expect results:
"I've dispatched [task] to run in the background. Expected completion: ~[N] minutes.
I'll check results when it finishes. In the meantime, what else can I help with?"

### Step 5: Collect and Integrate Results

When the background task completes:

1. **Read the output** at the agreed location
2. **Validate against success criteria** -- did it produce what was expected?
3. **Check for errors** -- scan for error markers, incomplete output, or missing files
4. **Integrate into main workflow** -- summarize results for the user, apply outputs
5. **Clean up** -- remove temporary files if the task used scratch space

If results are invalid or incomplete:
- Report what succeeded and what failed
- Decide whether to retry (same parameters) or escalate (ask user)
- Never silently swallow a failed background task

## IV. Best Practices

1. **Write output contracts, not task novels.** The background agent needs to know WHERE to put results and WHAT format, not a 500-word backstory.
2. **One task, one output location.** Never have two background tasks writing to the same file. Use separate output paths and merge in the foreground.
3. **Set realistic timeouts.** A task that "might take 5 minutes" should have a 10-minute timeout. A task with no timeout can hang forever.
4. **Prefer file artifacts over stdout.** Background stdout can be lost or truncated. Write results to a file that persists.
5. **Keep the foreground agent useful.** The whole point is freeing the user's attention. If you dispatch background work and then sit idle waiting, you've gained nothing.

## V. Quality Checklist

- [ ] Task duration estimated and classified (foreground vs background)
- [ ] Dispatch mode selected (Bash background, Agent sub-task, or parallel)
- [ ] Handoff includes: task description, output location, format, success criteria, timeout
- [ ] Output location is a specific file path, not "somewhere"
- [ ] User is informed of what was dispatched and expected completion time
- [ ] No polling loops -- using notification-based monitoring
- [ ] Results validated against success criteria after completion
- [ ] Errors surfaced to user, not silently swallowed
- [ ] Temporary files cleaned up

## VI. Common Pitfalls

- **Dispatching tasks that need clarification.** If the task definition is fuzzy, the background agent will either guess wrong or stall. Get clarity BEFORE dispatch.
- **Polling in a sleep loop.** `while true; do sleep 5; check; done` wastes cycles and context. Use `run_in_background` and wait for notification.
- **Shared state conflicts.** Two agents writing to the same file or modifying the same codebase region will corrupt each other's work. Partition clearly.
- **Forgetting to check results.** Dispatching is only half the job. If you never read the output, the background work was pointless.
- **Over-parallelizing.** 2-4 parallel tracks are manageable. 8+ create coordination overhead that exceeds the parallelism benefit. The Solo Operator Focus Theorem applies.
- **No timeout.** A background task without a timeout is a resource leak. Always set one.

## VII. Example

**Scenario:** User asks to run a full test suite (estimated 5 minutes) while continuing to work on a new feature.

**Step 1:** Classify -- 5 minutes, clearly background territory.

**Step 2:** Dispatch mode -- Bash with `run_in_background=true` (it's a shell command).

**Step 3:** Handoff:
```bash
# Dispatch: run test suite in background
cd /project && npm test > /tmp/test-results-2026-04-06.log 2>&1
# run_in_background=true, timeout=600000
```
Output contract: `/tmp/test-results-2026-04-06.log`, plain text test output.

**Step 4:** Tell user: "Test suite dispatched to background. Expected ~5 minutes. I'll report results when it finishes. What would you like to work on?"

**Step 5:** On completion notification, read log, report: "Tests complete: 408 passed, 2 failed. Failures in auth.test.ts lines 45 and 82 -- both related to the token refresh change from yesterday."

## VIII. Related Skills

- `parallel-tracks` -- Splits large tasks into independent tracks (complements dispatch)
- `handoff-protocol` -- Structures context transfer between agents
- `granular-visibility` -- Progress tracking patterns for long-running work
- `background-agent-parallelism` seed -- Long ops as background, short as foreground
- `async-sandbox-dispatch` seed -- Control plane / data plane separation pattern

## Output

- Dispatch mode decision (Bash background, Agent sub-task, or parallel tracks) with rationale
- Completed handoff block: task description, output contract (exact file path + format), success criteria, timeout
- User-facing status message confirming what was dispatched and expected completion time
- Result validation report after background task completes: success/failure against criteria, errors surfaced, temporary files cleaned up

## Examples

**Scenario 1:** "Run the full test suite while we keep working on the auth feature." → Task classified as 5-minute background (Bash mode); dispatched with `run_in_background=true`, output to `/tmp/test-results-2026-04-08.log`, timeout 600s; user informed; on completion, results read and failures reported with file and line numbers.

**Scenario 2:** "Normalize all 47 community skills while I plan the next phase." → Task classified as 10+ minutes (Agent sub-task mode); split into 3 parallel agents (groups of ~16 skills each) each writing to a separate output path; main thread plans Phase 4; results merged and validated after all 3 complete.

## Edge Cases

- Task definition is ambiguous at dispatch time: do not dispatch; get clarity first — a background agent that guesses wrong produces unusable output and cannot ask mid-task
- Background task completes but output file is missing or empty: do not silently accept; report the failure, check the exit code, and decide whether to retry with the same parameters or escalate
- User asks to check background task status before notification arrives: do not poll in a sleep loop; tell the user the task is still running and you will report when notified

## Anti-Patterns

- Dispatching a task under 30 seconds to background — dispatch overhead exceeds the task duration; just run it inline
- Not specifying an exact output file path before dispatch — "output it somewhere" means the foreground agent cannot find results
- Running a polling loop (`while true; do sleep 5; check; done`) instead of waiting for `run_in_background` notification — wastes context and cycles
- Dispatching more than 4 parallel agents at once — coordination overhead exceeds parallelism benefit past that threshold
- Silently swallowing a failed background task — always surface errors to the user with what succeeded and what failed
