---
name: decision-propagation
description: A structured protocol for recording architectural decisions and systematically propagating their effects across an interconnected document ecosystem. Use when decisions affect multiple documents, when scope changes impact releases, or when maintaining coherence across a living documentation system.
triggers:
  - "propagate this decision across the docs"
  - "a decision was made, update all affected documents"
  - "trace where this architectural change echoes"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - implementation-agent
    - strategic-agent
---

# Decision-Propagation Protocol

## I. The Philosophy

Architectural decisions are not isolated events. When a decision arrives—especially answering an open question in an existing document—it creates ripples across an entire ecosystem of interdependent files. A decision like "auth bypass" doesn't just answer one question; it changes scope in the master plan, triggers deferrals in other documents, and reshapes the status summary of the architecture.

Without a propagation protocol, decisions become stranded. A spec answers a question, but the master plan still lists it as open. A document defers work, but the dependency graph hasn't updated. The system becomes incoherent—different documents contradict each other about what is decided and when.

This skill prevents that dissonance by treating decision-propagation as a deliberate, multi-document process. The human provides the decision once, in one place. Your job is to trace where that decision echoes and update each location.

**Core Principle:** Decisions propagate or they die.

## II. When to Use This Skill

Use the decision-propagation protocol when:

- A human provides answers to open questions in a document or specification
- An architectural decision changes the scope of a release or work track
- A decision in one document affects the content, sequencing, or priorities of others
- You need to defer work from one release to a later one based on new information
- The STATUS file or master tracking document is out of sync with the decisions living in specs
- Any time a decision made in one place must be reflected in many places

## III. The Workflow: Five Steps

### Step 1: Record Decisions at Source

Locate the document where the decision was made (scout, spec, or master plan section).

Replace the "Open Questions" section with "Decisions ([Name], [Date])". Number each decision:

```markdown
## Decisions (Cruz, 2026-02-11)

### 1. Entity Backend Priority

**Decision:** Build entity-centric backend first, before other subsystems.

**Rationale:** Reduces scope and focuses team on core abstraction.

**Alternatives Considered:**
- Parallel development of all subsystems → Rejected: coordination overhead
- Start with API gateway → Rejected: needs entities first

**Impact Scope:**
- Defers non-entity tasks to later release
- Reshuffles parallel track allocation
- Affects integration timeline

**Effective Date:** 2026-02-11

---

### 2. Auth Bypass

**Decision:** Skip production auth layer; ship with debug token instead.

**Rationale:** Unblocks frontend integration work; auth hardening can move to later release.

**Alternatives Considered:**
- Build basic auth → Rejected: adds 2-3 weeks to timeline
- Use third-party auth → Rejected: external dependency, integration complexity

**Impact Scope:**
- Removes auth from current scope
- Adds auth to future scope
- Frontend work can proceed immediately
- Security posture (temporary degradation, acceptable for internal alpha)

**Effective Date:** 2026-02-11
```

**Format Requirements:**
- Capture exact words (don't paraphrase)
- Include attribution (who decided) and date
- Number decisions for reference
- Document alternatives considered (not just chosen option)
- State impact scope explicitly

### Step 2: Trace Document Dependencies

Before editing anything, make the complete list of documents affected. Common dependency patterns:

**Master plan** → typically affected in:
- Scope blocks (what's in/out)
- Dependency graph (edges between tasks)
- Constraints (limitations acknowledged)
- Parallel track allocation
- Next steps (sequencing)

**Other specs/documents** → may be:
- Promoted (dependencies now met, can proceed)
- Deferred (dependencies removed or pushed later)
- Needing content updates (references to the decision area)

**Implementation prompts:**
- May need regeneration if scope changed
- Updated task lists
- Modified success criteria

**STATUS.md / Tracking files:**
- Always needs updating
- Decision log
- Scope summaries
- Architecture summaries
- Next steps

**Walk through each document systematically:**

```bash
# Find all documents that might reference the decision
grep -r "auth" docs/ --include="*.md" | grep -v ".git"

# Find all documents referencing the affected scope
grep -r "v0.2.0" docs/ --include="*.md"

# Find STATUS and tracking files
find docs/ -name "STATUS.md" -o -name "*_tracking.md"
```

**Create a propagation checklist:**
- [ ] Master plan: `docs/v0.2.0/master_plan.md`
- [ ] Related doc: `docs/v0.2.0/frontend_integration.md`
- [ ] Implementation prompt: `docs/v0.2.0/prompts/backend_implementation.md`
- [ ] STATUS file: `docs/v0.2.0/STATUS.md`
- [ ] Future scope: `docs/v0.2.2/master_plan.md`

### Step 3: Propagate to Each Dependent Document

For each dependent document, make **surgical edits** — change only the affected sections.

**In Master Plans:**
```markdown
## Scope

### In Scope (v0.2.0)
- Entity-centric backend ← UPDATED: promoted to priority
- Frontend breadcrumb navigation
- ~~Production auth layer~~ ← REMOVED: deferred to v0.2.2

### Out of Scope (Deferred)
- Production auth layer → v0.2.2 ← ADDED: decision on 2026-02-11
- OAuth integration → v0.2.2
```

**In Related Documents:**
```markdown
## Deferral Notice

This work has been deferred to v0.2.2 based on decision made 2026-02-11.

**Decision Reference:** Auth Bypass (Cruz, 2026-02-11)

**Reasoning:** Frontend integration work can proceed with debug token; full auth implementation moves to v0.2.2 without blocking current sprint.

**Proposed Timeline:** v0.2.2 (target: March 2026)

**Dependencies for Reactivation:**
- v0.2.0 completion
- v0.2.1 scope finalized
- Auth requirements review completed
```

**In STATUS.md:**
```markdown
## Key Architecture Decisions

### Auth Bypass (Cruz, 2026-02-11)
**Decision:** Skip production auth; ship with debug token
**Impact:** Unblocks frontend work immediately; auth moves to v0.2.2
**Status:** Active

### Entity Backend Priority (Cruz, 2026-02-11)
**Decision:** Build entity-centric backend first
**Impact:** Reshuffles scope allocation
**Status:** Active
```

### Step 4: Update Master Tracking

Update STATUS.md last. It's the summary of everything else. Include:

1. **Add Decisions Block** (if not present)
2. **Update Scope Tables** (what's in/out of each release)
3. **Update Dependency Graph** (if shown visually)
4. **Update Next Steps** (reorder if sequencing changed)
5. **Refresh Architecture Summary** (if affected)

**Example STATUS.md Update:**
```markdown
# Project Status

**Last Updated:** 2026-02-11 | **Status:** In Progress

## Key Architecture Decisions

### Auth Bypass (Cruz, 2026-02-11)
**Decision:** Skip production auth; ship with debug token instead
**Rationale:** Unblocks frontend integration immediately
**Impact:**
- Current scope reduced (removes auth)
- Future scope increased (adds auth)
- Frontend can proceed without waiting

### Entity Backend Priority (Cruz, 2026-02-11)
**Decision:** Build entity-centric backend first, before other subsystems
**Rationale:** Focuses effort on core abstraction
**Impact:**
- Defers non-entity work to later release
- Reshuffles parallel track allocation

## Scope by Release

| Release | In Scope | Deferred | Complete |
|---------|----------|----------|----------|
| v0.2.0 | Entity backend, Breadcrumb nav | ~~Auth layer~~ → v0.2.2 | 40% |
| v0.2.1 | API gateway, Non-entity tasks | TBD | 0% |
| v0.2.2 | **Auth layer** (deferred from v0.2.0), OAuth | TBD | 0% |

## Next Steps

1. ✅ Complete entity backend implementation
2. ✅ Integrate breadcrumb navigation
3. ⏳ Frontend integration (unblocked by auth bypass decision)
4. ⏳ QA and testing
5. 📅 Begin next release planning
```

### Step 5: Sync Copies

If documents exist in multiple locations (e.g., project root AND docs/v0.2.x/), sync them. Compare file sizes or use `diff` to verify.

```bash
# Compare files for consistency
diff docs/STATUS.md docs/v0.2.0/STATUS.md

# Sync if different
cp docs/v0.2.0/STATUS.md docs/STATUS.md
```

---

## IV. Document Dependency Patterns

Understanding these patterns helps you trace dependencies quickly:

- **Decisions flow upward:** A document decision affects the master plan above it
- **Decisions cascade sideways:** A decision in one document defers or enables tasks in other documents
- **Scope is bidirectional:** Adding scope to v0.2.0 removes it from v0.2.1; changes must be reflected in both
- **Deferrals create cross-references:** When a document says "deferred to v0.2.3," the v0.2.3 master plan must acknowledge the deferred item

---

## V. Best Practices

### Record Verbatim
Capture the human's exact words, not your paraphrase. Paraphrasing introduces interpretation and drift.

### Trace Before Editing
Make the full list of dependent documents first. This prevents missed updates and duplicated effort.

### Surgical Over Wholesale
Change only the specific sections affected. Rewriting entire documents introduces risk and obscures what actually changed.

### Always Update STATUS Last
It's the final coherence checkpoint. If STATUS is current, the whole system is current.

### Document Deferrals Clearly
When work is pushed to a later release, add a note explaining why and when it should be reconsidered.

### Verify Cross-References
If document A says "deferred to X," document X must acknowledge it. No orphaned references.

---

## VI. Quality Checklist

Before considering the decision propagated:

- [ ] Decision recorded at source with human name, date, and full reasoning
- [ ] All dependent documents identified (master plan, related docs, STATUS, implementation prompts)
- [ ] Each dependent document updated in specific affected sections (not blanket rewrites)
- [ ] STATUS.md reflects all changes and serves as a coherence checkpoint
- [ ] Document copies synced across locations if they exist in multiple places
- [ ] No orphaned references to old scope or pre-decision state remain in any document
- [ ] Cross-references between documents are consistent
- [ ] Alternatives considered are documented (not just the chosen option)

---

## VII. Integration with Connectors

### With ~~repository (GitHub)
- Commit decision as atomic change
- Link decision to related PRs or issues
- Use conventional commit message: `docs(decision): Record auth bypass for v0.2.0`
- Tag decision in commit for traceability

### With ~~project tracker (Linear, Asana)
- Create tickets for each touchpoint affected
- Update ticket dependencies
- Link tickets to decision document
- Adjust sprint planning based on scope changes

### With ~~chat (Slack)
- Announce decision in relevant channel
- Share decision document link
- Enable team discussion and questions
- Post impact summary for visibility

### With ~~knowledge base (Notion)
- Sync decision to centralized decision log
- Link to affected documentation
- Update architecture diagrams if needed
- Create searchable decision index

---

## VIII. Common Scenarios

### Scenario 1: Scope Reduction Decision

**Trigger:** "We're deferring feature X to next release"

**Propagation:**
1. Record decision in current document
2. Update current release scope (remove feature)
3. Update next release scope (add feature)
4. Update any documents dependent on feature X
5. Update STATUS with new scope allocation

### Scenario 2: Architectural Pattern Change

**Trigger:** "We're switching from pattern A to pattern B"

**Propagation:**
1. Record decision with rationale
2. Update all specs referencing pattern A
3. Update implementation prompts
4. Update status architecture summary
5. Create migration guide if needed

### Scenario 3: Timeline Impact Decision

**Trigger:** "Decision X delays release by 2 weeks"

**Propagation:**
1. Record decision and timeline impact
2. Update release roadmap
3. Update dependent release schedules
4. Update STATUS with new timelines
5. Notify stakeholders via ~~chat

---

## IX. Troubleshooting

### "I'm not sure what documents are affected"
- Start with obvious ones (master plan, STATUS)
- Use grep to search for keywords
- Check documents that reference the same scope or phase
- When in doubt, include it — better to check and find no changes needed

### "The decision affects too many documents"
- That's normal for architectural decisions
- Make a checklist, work through systematically
- Don't skip documents — propagation must be complete
- Consider if this indicates a need for better modularity

### "Some documents contradict each other after propagation"
- Review cross-references
- Ensure bidirectional consistency (if A references B, B should acknowledge A)
- Update STATUS last to catch inconsistencies
- Use grep to find orphaned references

---

## X. Skill Metadata

**Token Savings:** ~3,000-8,000 tokens (prevents reading all docs to find inconsistencies)
**Quality Impact:** Maintains document ecosystem coherence; prevents contradictions
**Maintenance:** Review propagation patterns quarterly; refine based on missed updates

**Related Skills:**
- `handoff-protocol` — For propagating decisions through handoffs
- `workspace-navigation` — For finding affected documents systematically
- `agent-teaching` — For teaching decision rationale to other agents

---

**Last Updated:** 2026-02-11
**Maintained By:** Tres Pies Design
**Status:** Active
