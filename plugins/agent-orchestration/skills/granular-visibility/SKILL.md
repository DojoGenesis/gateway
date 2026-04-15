---
name: granular-visibility
model: sonnet
description: "Produces a granular todo list where each item maps to one discrete deliverable or decision point, maintained at real-time accuracy throughout execution. Use when: 'what is the status', 'too many items in the list', 'simplify the progress tracking', 'long todo list'."
category: agent-orchestration
triggers:
  - "what is the status"
  - "too many items in the list"
  - "simplify the progress tracking"
  - "long todo list"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: task_list
    type: string
    description: The current task list or work description to track
    required: true
outputs:
  - name: todo_list
    type: string
    description: Granular todo list where each item maps to one discrete deliverable or decision point
---
# Granular Visibility

## Philosophy

On complex work, the user steers. Visibility is the steering wheel. Compressed progress updates remove the user's ability to see what's happening, what's next, and what might need course correction. The instinct to simplify feels helpful to the agent but serves the agent's preference for tidiness, not the user's need for control.

The chain runs in one direction: granular visibility → user sees progress → user can steer → user trusts the process → user delegates more → agent moves faster → better outcomes.

The anti-pattern runs the other way: compressed visibility → user can't see progress → user interrupts to check status → agent stops to explain → both lose momentum → trust erodes.

## When to Use

- Any task with more than 10 discrete steps
- Multi-phase work spanning multiple tool calls or coordinated actions
- When the instinct arises to "simplify" or "clean up" progress tracking
- When working with a user who values steering and visibility over convenience
- When a todo list length triggers concern that it's "too long" — that's usually a signal the work is complex, not that the list is bloated

## Workflow

### Creating Granular Items

1. **Map discrete deliverables**: One todo item per concrete output or decision point, not one per phase
2. **Be specific**: "Read config file" is not granular; "Read config file, identify auth settings, determine if migration needed" is
3. **Avoid nesting**: Don't hide sub-steps under parent tasks; make them visible
4. **Start with the full list**: Granularity from the beginning creates accurate expectations

### Maintaining Visibility

1. **Update status in real-time**: Mark items `in_progress`, `completed`, or `pending` as they change
2. **Add blockers immediately**: If an item is blocked, note it — don't wait for the user to notice
3. **Communicate what changes**: If scope shifts, show how the list evolves
4. **Never pre-compress**: Let the user ask for a simpler view; don't offer it unprompted

### Responding to "Too Long"

If the user says the todo list feels long:

- Confirm that length reflects complexity, not bloat
- Offer to reorganize (by phase, by urgency) without removing items
- Ask if they want a compressed summary *in addition to* the granular list
- Never delete items to make the list shorter

## Best Practices

### What to Track Granularly

- Discrete deliverables (read file, analyze results, generate output)
- Decision points (determine eligibility, choose approach, validate direction)
- Quality gates (test results, review findings, check integrity)
- Coordination steps (wait for user input, get approval, resolve blocking issue)

### What to Consolidate

- Trivial sub-steps (opening a file is part of "read file," not a separate item)
- Steps that must happen together (if A always requires B immediately, they're one item)
- Repeated boilerplate (don't create 20 "read file X" items; group similar reads)

### Tone and Clarity

- Use imperative form: "Read config file" not "Reading the config file"
- Be specific about scope: "Parse JSON and extract auth settings" not "Process data"
- Make status clear: distinguish between "pending" (waiting for decision), "in_progress" (actively working), and "blocked" (waiting for external factor)

## Quality Checklist

- [ ] Each todo item represents one discrete deliverable or decision point
- [ ] Status is updated in real-time as work progresses
- [ ] The list length matches the actual complexity of the work
- [ ] Truly trivial sub-steps (e.g., "read file X") are not individually tracked
- [ ] User has not been asked to accept or approve a compressed view
- [ ] Blocking issues are flagged immediately with reason and impact
- [ ] If scope changes, the list evolves to reflect it
- [ ] Items are specific enough that their status is unambiguous

## Common Pitfalls

### Pre-Compression

**Pitfall**: "The list is getting long, I should simplify it before showing the user."
**Reality**: Length reflects work complexity. Compression removes steering ability.
**Fix**: Show the full list. If the user wants fewer items, they'll ask.

### Hiding Sub-Steps

**Pitfall**: Creating high-level items like "Phase 1: Analysis" to keep the list short.
**Reality**: The user can't see what needs to happen in Phase 1 until you show up with new items mid-stream.
**Fix**: Break Phase 1 into discrete items from the start. It's okay if that creates 8 items instead of 1.

### Assuming Simplicity Preference

**Pitfall**: Assuming all users prefer simplified views and offering compression unprompted.
**Reality**: Many users explicitly choose granular tracking and will ask if they want less detail.
**Fix**: Default to granular. Compression is opt-in, not default.

### Updating Retrospectively

**Pitfall**: Waiting until the phase is done to update the todo list.
**Reality**: The user sees stale status and loses confidence in the progress tracking.
**Fix**: Update as items change. Mark things `in_progress` before you start them, `completed` immediately after.

## Related Skills

- **handoff-protocol**: Clean handoffs require visible progress context; granular tracking provides it
- **workspace-navigation**: Navigating shared work benefits from detailed status visibility
- **decision-propagation**: Decisions surface faster when progress is visible and granular
- **agent-teaching**: Teaching another agent requires understanding where work stands — visibility enables this

## Output

- A structured todo list with one item per discrete deliverable or decision point, each in imperative form with unambiguous scope
- Real-time status markers (`pending`, `in_progress`, `completed`, `blocked`) updated as work changes state
- Blocker annotations on any item that cannot proceed, including the reason and what is needed to unblock

## Examples

**Scenario 1:** User asks to migrate a codebase across 14 files with 3 decision points → skill produces a 17-item list (14 file items + 3 decision items), each named specifically (e.g., "Parse auth settings in config.go and determine if migration required"), updated to `in_progress` before work starts and `completed` immediately after.

**Scenario 2:** User says "the todo list feels too long" on a 22-item task → skill confirms length reflects actual complexity, offers to reorganize items by phase without removing any, and asks whether the user wants a compressed summary alongside the full list — not instead of it.

## Edge Cases

- If the user explicitly requests compression and accepts the tradeoff of reduced steering visibility, provide a phased summary view but preserve the full granular list internally and offer to restore it at any time.
- If scope changes mid-task (new requirements added or items dropped), update the list visibly — add new items, mark removed items as `cancelled` with a reason — do not silently restructure.

## Anti-Patterns

- Grouping multiple deliverables under a single "Phase N" item to keep the count low — this hides complexity and breaks real-time status accuracy because the phase item cannot be marked complete until all sub-work finishes.
- Updating todo status in batch at the end of a work block rather than in real-time — the user sees stale data and cannot steer during execution.
