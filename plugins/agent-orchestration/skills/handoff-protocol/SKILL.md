---
name: handoff-protocol
description: A structured protocol for handing off work between autonomous agents, ensuring context is preserved, goals are clear, and the receiving agent is set up for success.
triggers:
  - "hand this off to another agent"
  - "prepare a handoff package for the next agent"
  - "pass the baton on this task"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - implementation-agent
    - research-agent
    - strategic-agent
---

# Agent Handoff Protocol Skill

**Version:** 1.0
**Created:** 2026-02-11
**Author:** Tres Pies Design
**Purpose:** To provide a clear, repeatable protocol for handing off work between agents, ensuring no loss of context and a high probability of success.

---

## I. The Philosophy: The Sacred Relay

In a multi-agent ecosystem, a handoff is a **sacred relay**. It is the moment where one agent passes the baton of responsibility to another. A fumbled handoff leads to dropped context, wasted effort, and a broken workflow. A clean handoff is a moment of trust and shared understanding, where the receiving agent is not just given a task, but is fully equipped to carry it forward.

This protocol transforms the handoff from a hopeful transfer into a rigorous, verifiable exchange. It is an act of respect for the receiving agent, acknowledging that their ability to succeed is entirely dependent on the quality of the context they are given.

---

## II. When to Use This Skill

- **Always** use this skill when one agent's work is complete and another agent must begin the next phase
- **Spec → Implementation:** When a specification is complete and ready for implementation
- **Research → Synthesis:** When research is complete and needs synthesis or application
- **Any Agent → Any Agent:** Whenever responsibility for a task is transferred

---

## III. The Handoff Workflow

### Step 1: Prepare the Handoff Package

The sending agent is responsible for preparing a complete and self-contained "handoff package." This is a markdown document that contains everything the receiving agent needs to know. Use the template from Section IV.

### Step 2: Verify the Package with the Checklist

Before initiating the handoff, the sending agent must verify the package against the Handoff Checklist in Section V. This is a self-assessment to ensure quality.

### Step 3: Initiate the Handoff

The sending agent initiates the handoff by creating a task for the receiving agent, with the handoff package as the primary input.

**Example Handoff Task:**
- **Agent:** Implementation Agent (Zenflow, Claude Code, etc.)
- **Task:** Implement the breadcrumb navigation feature
- **Input:** `handoffs/2026-02-11_breadcrumb_handoff.md`

### Step 4: The Receiving Agent's Acceptance

The receiving agent's first action is to read the handoff package and confirm that it has everything it needs to proceed. If the package is incomplete, the receiving agent must immediately return the task to the sender with a request for more information.

**Example Acceptance:** "Handoff package received and verified. I have all necessary context to proceed with the implementation. Work will now begin."

**Example Rejection:** "Handoff package is incomplete. The link to the specification is broken. Please correct and resubmit."

---

## IV. Handoff Package Template

```markdown
# Agent Handoff Package

**From:** [Sending Agent Name]
**To:** [Receiving Agent Name]
**Date:** [YYYY-MM-DD]
**Subject:** Handoff of [Task/Project Name]

---

## 1. Objective

> [A single, clear sentence describing the receiving agent's primary goal.]

---

## 2. Required Context

[A comprehensive list of all files, documents, and resources the receiving agent MUST read before starting work. Use full paths.]

**Core Documents:**
- **Specification:** `[path/to/specification.md]`
- **Status:** `[path/to/STATUS.md]`

**Key Conversation Summaries:**
- `[path/to/conversation_summary_1.md]`
- `[path/to/conversation_summary_2.md]`

**Relevant Seeds or Skills:**
- `[path/to/seed.md]`
- `[path/to/skill.md]`

**Pattern Files / Code Examples:**
- `[path/to/pattern_file_1.tsx]`
- `[path/to/pattern_file_2.go]`

---

## 3. Task Definition

[A clear and unambiguous definition of the task to be performed. If this is an implementation agent handoff, this section should contain the full implementation prompt.]

---

## 4. Definition of Done

[How will we know the receiving agent's work is complete? This must be a list of binary, testable success criteria.]

- [ ] [Success Criterion 1]
- [ ] [Success Criterion 2]
- [ ] [Success Criterion 3]

---

## 5. Constraints & Boundaries

[What are the explicit constraints on the receiving agent's work?]

- **DO NOT** modify files outside of the specified scope.
- **DO NOT** make architectural decisions without consulting [Sending Agent Name].
- **MUST** follow the patterns established in the provided pattern files.

---

## 6. Next Steps (After Completion)

[What happens after the receiving agent is done? Who takes the next handoff?]

- Upon completion, notify [Next Agent Name] and hand off the results for the next phase (e.g., QA, documentation).
```

---

## V. Handoff Checklist (for Sending Agent)

- [ ] **Is the Objective a single, clear sentence?**
- [ ] **Are all links in the Required Context section valid and pointing to the correct files?**
- [ ] **Is the Task Definition unambiguous and complete?**
- [ ] **Is the Definition of Done a list of testable, binary criteria?**
- [ ] **Are the Constraints clear and explicit?**
- [ ] **Is the next step after completion clearly defined?**

---

## VI. Best Practices

### No Implicit Context
If it's not in the handoff package, it doesn't exist. Never assume the receiving agent knows something.

### Over-communicate
It is better to provide too much context than too little.

### The Receiver is the Gatekeeper
Empower the receiving agent to reject incomplete handoffs. This maintains quality across the ecosystem.

### Standardize Handoff Locations
Create a `handoffs/` directory in each project to store these packages, creating a clear audit trail.

### Include Attribution
Always include who made the handoff and when. This enables future agents to trace decisions and context.

### Link, Don't Duplicate
Reference source documents rather than copying their entire contents. This prevents drift and keeps the handoff package focused.

---

## VII. Common Patterns

### Pattern 1: Implementation Handoff

**When:** Spec is complete, ready for code implementation

**Key Elements:**
- Link to full specification
- Pattern files showing established code conventions
- Exact file paths for files to create/modify
- Test coverage requirements in Definition of Done

**Example Objective:**
> Implement breadcrumb navigation component with state management integration

### Pattern 2: Research Handoff

**When:** Research question needs to be explored

**Key Elements:**
- Clear research question framing
- Suggested starting sources
- Scope boundaries (what's in/out of scope)
- Output format specification

**Example Objective:**
> Research and synthesize context compression techniques for LLM-based agents

### Pattern 3: Review Handoff

**When:** Work needs quality review before finalization

**Key Elements:**
- Link to work to be reviewed
- Review criteria (checklist)
- Known concerns to investigate
- Decision authority (who approves)

**Example Objective:**
> Review backend API specification for completeness and consistency with system architecture

### Pattern 4: Cross-Session Handoff

**When:** Continuing work across agent sessions or different agents

**Key Elements:**
- Compressed context summary of previous session
- Current state documentation
- Next steps clearly defined
- Memory artifacts attached

**Example Objective:**
> Continue implementation of entity backend from previous session, starting with schema definition

---

## VIII. Integration with Other Skills

### With `workspace-navigation`
- Store handoffs in `00_Active/handoffs/` directory
- Move to archive after completion
- Link from workspace README for discoverability

### With `decision-propagation`
- Include decision records in Required Context
- Reference decision IDs for traceability
- Update decision impact tracking after handoff completion

### With `agent-teaching`
- Attach teaching artifacts for knowledge transfer
- Include learning resources in Required Context
- Use peer-to-peer tone in task explanation

---

## IX. Quality Indicators

**Good Handoff Package:**
- Receiver can begin work immediately without questions
- All links work and point to current resources
- Success criteria are measurable and binary
- Context is comprehensive but not duplicative
- Constraints prevent common mistakes

**Bad Handoff Package:**
- Receiver needs to ask clarifying questions
- References "see above" or "as discussed"
- Broken links or missing resources
- Vague success criteria ("good quality")
- Unstated assumptions about receiver's knowledge

---

## X. Troubleshooting

### "Receiver rejected my handoff"
- Review the 6-point checklist — which criteria failed?
- Ask receiver specifically what's missing
- Update package and resubmit
- Consider if task definition is truly unambiguous

### "I'm not sure what to include"
- Ask: "Could the receiver complete this with zero questions?"
- If no, add more context
- Err on side of over-communication
- Include pattern files and examples

### "Handoff package is getting too long"
- Link to detailed docs rather than duplicating
- Use summaries in Required Context with links to full content
- Focus Task Definition on what to do, not how (unless implementation details are critical)

---

## XI. Skill Metadata

**Token Savings:** ~5,000-10,000 tokens per handoff (prevents back-and-forth clarification)
**Quality Impact:** Reduces failed handoffs by ~80% when checklist is followed
**Maintenance:** Review template quarterly; update based on handoff rejection patterns

**Related Skills:**
- `workspace-navigation` — For organizing handoffs in shared workspaces
- `agent-teaching` — For transferring knowledge to receiving agents
- `decision-propagation` — For including decision context in handoffs

---

**Last Updated:** 2026-02-11
**Maintained By:** Tres Pies Design
**Status:** Active
