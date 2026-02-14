# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: decision-propagation
description: Record architectural decisions at source and propagate their effects through all dependent documents. Use when decisions arrive that answer open questions or change scope.
triggers:
  - "propagate this decision through the system"
  - "trace where this decision echoes"
  - "update the status file last"
  - "this decision changes the scope—where else must it flow"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - file_system
    - bash
    - meta_skill
  portable: true
  tier: 3
  agents:
    - implementation-agent
    - strategic-agent
---

# Decision-Propagation Protocol

## I. The Philosophy

Architectural decisions are not isolated events. When a decision arrives—especially to an open question in an existing document—it creates ripples across an entire ecosystem of interdependent files. A decision like "auth bypass" doesn't just answer one scout's question; it changes scope in the master plan, triggers deferrals in other scouts, and reshapes the STATUS file's summary of the architecture.

Without a propagation protocol, decisions become stranded. A spec answers a question, but the master plan still lists it as open. A scout defers work, but the dependency graph hasn't updated. The system becomes incoherent—different documents contradict each other about what is decided and when.

This skill prevents that dissonance by treating decision-propagation as a deliberate, multi-document process. The human provides the decision once, in one place. Your job is to trace where that decision echoes and update each location.

## II. When to Use This Skill

Use the decision-propagation protocol when:

- A human provides answers to open questions in a scout or specification
- An architectural decision changes the scope of a release or work track
- A decision in one document affects the content, sequencing, or priorities of others
- You need to defer work from one release to a later one based on new information
- The STATUS file or master tracking document is out of sync with the decisions living in specs and scouts
- Any time a decision made in one place must be reflected in many places

## III. The Workflow: Five Steps

### Step 1: Record Decisions at Source

Locate the document where the decision was made (scout, spec, or master plan section).

Replace the "Open Questions" section with "Decisions ([Name], [Date])". Number each decision:

```
## Decisions (Cruz, 2026-02-11)

1. Entity Backend is v0.2.0 Priority
   - Decision: Build entity-centric backend first, before other subsystems.
   - Reasoning: Reduces scope of v0.2.1 and focuses team on core abstraction.
   - Implication: Defers non-entity tasks to v0.2.1; reshuffles parallel track allocation.

2. Auth Bypass
   - Decision: Skip production auth layer in v0.2.0; ship with debug token instead.
   - Reasoning: Unblocks frontend integration work; auth hardening moves to v0.2.2.
   - Implication: Changes v0.2.0 scope (removes auth), changes v0.2.2 scope (adds auth).
```

For each decision, capture the exact words, the reasoning (why the human chose this), and any stated implications. If the decision adds or removes scope, note it explicitly.

### Step 2: Trace Document Dependencies

Before editing anything, make the complete list of documents affected. Common dependency patterns:

- **Master plan** → typically affected in: scope blocks, dependency graph, constraints, parallel track allocation, next steps
- **Other scouts** → may be promoted (dependencies now met), deferred (dependencies removed or pushed later), or need content updates
- **Implementation specs or prompts** → may need regeneration if scope changed
- **STATUS.md** → always needs updating; it's the coherence checkpoint

Walk through each document and ask: "Does this reference the decision I just recorded?" If yes, it's dependent.

### Step 3: Propagate to Each Dependent Document

For each dependent document, make surgical edits. Do not rewrite entire sections unless necessary.

**In master plans:** Update scope tables, dependency graph (remove/add edges), constraint lists, parallel track allocation.

**In other scouts:** If a decision defers work, add a new section noting the deferral, the reasoning (reference the decision), and the proposed timeline. If a decision enables a scout, promote it in priority or scheduling.

**In STATUS.md:** Add a "Key Architecture Decisions" block and list all decisions with their implications. Update any summary statements about scope or sequencing.

**In implementation prompts:** If scope changed, the prompt may need regeneration to reflect new boundaries.

### Step 4: Update Master Tracking

Update STATUS.md last. It's the summary of everything else. Include:

- The new decisions block (attribution, date, full text)
- Any tables listing specs, scouts, or prompts (update affected rows)
- "Next Steps" section (reorder if sequencing changed)
- Summary statement about architecture or scope (refresh if affected)

### Step 5: Sync Copies

If documents exist in multiple locations (e.g., thinking/ at repo root AND docs/v0.2.x/thinking/), sync them. Compare file sizes or use `diff` to verify.

## IV. Document Dependency Patterns

Understanding these patterns helps you trace dependencies quickly:

- **Decisions flow upward:** A scout decision affects the master plan above it
- **Decisions cascade sideways:** A decision in one scout defers or enables tasks in other scouts
- **Scope is bidirectional:** Adding scope to v0.2.0 removes it from v0.2.1; changes must be reflected in both
- **Deferrals create cross-references:** When a scout says "deferred to v0.2.3," the v0.2.3 master plan must acknowledge the deferred item

## V. Best Practices

- **Record verbatim.** Capture the human's exact words, not your paraphrase. Paraphrasing introduces interpretation and drift.
- **Trace before editing.** Make the full list of dependent documents first. This prevents missed updates and duplicated effort.
- **Surgical over wholesale.** Change only the specific sections affected. Rewriting entire documents introduces risk and obscures what actually changed.
- **Always update STATUS.md last.** It's the final coherence checkpoint. If STATUS.md is current, the whole system is current.
- **Document deferrals clearly.** When work is pushed to a later release, add a note explaining why and when it should be reconsidered.

## VI. Quality Checklist

Before considering the decision propagated:

- [ ] Decision recorded at source with human name, date, and full reasoning
- [ ] All dependent documents identified (master plan, related scouts, STATUS.md, implementation prompts)
- [ ] Each dependent document updated in specific affected sections (not blanket rewrites)
- [ ] STATUS.md reflects all changes and serves as a coherence checkpoint
- [ ] Document copies synced across locations if they exist in multiple places
- [ ] No orphaned references to old scope or pre-decision state remain in any document
- [ ] Cross-references between documents are consistent (if one document says "deferred to v0.2.3," the v0.2.3 plan acknowledges it)
