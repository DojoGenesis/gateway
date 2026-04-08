---
name: compression-ritual
description: >
  Six-step ritual for distilling long conversations and sessions into potent
  wisdom artifacts. Use when compressing a transcript, ending a long session,
  performing context hygiene, handing off a project, or when the context feels
  heavy and noisy. Also use when someone asks how to compress, distill, or
  summarize a session into lasting artifacts.
triggers:
  - "compress this session into wisdom artifacts"
  - "distill this long conversation into lasting memory"
  - "run the compression ritual on this transcript"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - research-agent
---

# Compression Ritual

## I. Philosophy: The Art of Letting Go

Compression is not destruction -- it is transformation. We transform the raw material of conversation into refined artifacts of wisdom. The goal is to lose volume while preserving meaning. Every token in a context window has a cost; compression ensures that cost buys wisdom, not noise.

The ritual framing matters. This is not "cleanup" or "archiving" -- it's a conscious, creative act of choosing what deserves to grow and what can be respectfully composted. The Art of Letting Go is the recognition that not everything deserves permanent cultivation, and that releasing context is an act of respect for future clarity.

Distillation follows a principle: **compress the experience, not just the text.** A good compression captures the *meaning* of a 3-hour session in 200 words, not just the first paragraph of each exchange.

## II. When to Use This Skill

- **After a long conversation** (20+ turns or 30+ minutes)
- **At the end of a major work session** or sprint
- **When the context window feels heavy or noisy** -- too much raw material
- **Before handing off a project** to another agent or team member
- **As a regular practice** (end of day) to maintain cognitive hygiene
- **When Tier A daily notes are piling up** and need compression

**When NOT to use:** For short conversations with 1-2 key points, a simple daily note via `/tend` is sufficient. Don't ritualize what doesn't need it.

## III. The Six-Step Compression Workflow

### Step 1: Signal Intent

Announce the ritual to frame it as deliberate practice:

> "We're entering a compression ritual. The goal is to distill this session into its essential wisdom."

**Why this matters:** Signaling shifts the mindset from "working" to "reflecting." It creates a boundary between the session and its compression.

### Step 2: Review the Transcript

Read through the material looking for five types of significant moments:

1. **Key Decisions** -- Choices that altered direction
2. **Profound Insights** -- New understandings or "aha!" moments
3. **Actionable Learnings** -- Lessons that inform future behavior
4. **Reusable Patterns** -- Ideas generalizable into seeds
5. **Unresolved Questions** -- Important items still open

**Key Insight:** Read for significance, not completeness. Most of the transcript is machinery -- the process of getting to insights. The insights themselves are what deserve preservation.

### Step 3: Choose the Right Vessel

Each significant moment needs an appropriate container:

| Vessel | Purpose | When to Use |
|---|---|---|
| Philosophical Reflection | Capture the "why" behind work | Deep insights about thinking or approach |
| Conversation Summary | Document decisions and outcomes | Significant discussions with multiple decisions |
| Seed | Preserve a reusable pattern | Generalizable insight with a trigger |
| Documentation Update | Integrate into existing docs | Key decision that changes how something works |
| Decision Record | Formalize a choice with rationale | Consequential decisions with alternatives considered |

**Decision point:** If a moment fits multiple vessels, choose the most specific one. A reusable pattern is a seed, not a reflection. A specific decision is a record, not a summary.

### Step 4: Write the Artifacts

Write each artifact with distillation as the goal:

- **Capture essence, not transcript** -- Don't copy-paste; rewrite in compressed form
- **Link between artifacts** -- A reflection might reference a decision record
- **Include metadata** -- Type, tier, date on every artifact
- **Use growth language** -- Planted, cultivated, distilled -- never stored or created

### Step 5: Apply the 3-Month Rule

For every potential artifact, explicitly categorize:

| Category | Action | What Belongs Here |
|---|---|---|
| **Cultivate** | Preserve as Tier B | Decisions, insights, seeds, failures, breakthroughs |
| **Summarize** | Condense to 1-2 sentences | Activities, discussions, exploration, intermediate steps |
| **Compost** | Release entirely | Pleasantries, dead-ends, repetitive iterations, duplicate context |

**Present the triage visibly.** The user should see what was cultivated, summarized, and composted. This transparency builds trust in the compression process.

### Step 6: Write the Compression Log

Every compression produces a meta-record:

```markdown
# Compression Log: YYYY-MM-DD

**Source:** [What was compressed]
**Original volume:** ~[X] tokens
**Compressed volume:** ~[Y] tokens
**Compression ratio:** [Z]% reduction

## Artifacts Cultivated
| Type | Path | Description |
|---|---|---|

## Insights Preserved
- [Insight 1]
- [Insight 2]

## Context Composted
- [What was released]

## Seeds Planted
- [Seed]: [Pattern]
```

**Key Insight:** The compression log is itself a Tier B artifact. It documents your curation decisions, which is institutional knowledge about what your team considers important.

## IV. Best Practices

### Be Ruthless, But Respectful

The goal is to reduce noise while preserving signal. Challenge every potential artifact: "Would someone need this in 3 months?" If the answer is "probably not," compost it. But do so with acknowledgment -- even composted material contributed to the session's thinking.

### Favor Wisdom Over Data

Prioritize the "why" and the "how" over the raw "what." A compression that preserves "we chose X because of Y" is more valuable than one that lists every file changed.

### Link, Don't Repeat

If a concept is already captured in an existing seed or decision record, link to it rather than rewriting it. Duplication is the enemy of a clean garden.

### Compress the Experience, Not Just the Text

A 3-hour debugging session might compress to: "Root cause was X. We misdiagnosed as Y because of Z. Seed: check-assumptions-before-code." That's the wisdom. The debugging transcript is machinery.

### Perform the Ritual Regularly

The more frequently you compress, the less daunting it becomes. Daily compression (via `/tend`) handles routine days. Weekly or per-session compression (via `/compress`) handles significant sessions.

## V. Quality Checklist

- [ ] All 6 steps followed (signal, review, vessel, write, 3-month rule, log)
- [ ] 3-month rule explicitly applied with visible triage (cultivate/summarize/compost)
- [ ] Compression log produced with volume metrics
- [ ] Every artifact has metadata (type, tier, date)
- [ ] Artifacts are significantly shorter than source material
- [ ] Seeds identified and formalized (or flagged for `/plant`)
- [ ] Growth language used throughout (no "store," "retrieve," "delete")
- [ ] Links between related artifacts where appropriate

## VI. Common Pitfalls

### Compressing Too Late

**Problem:** Waiting weeks to compress means the context is cold. You've forgotten why decisions were made, and the compression becomes surface-level.

**Solution:** Compress within 24-48 hours of a significant session. The wisdom is freshest when the experience is recent.

### Preserving Machinery Instead of Wisdom

**Problem:** The compression reads like a summary of every exchange rather than a distillation of insights. "First we discussed X, then we explored Y, then we decided Z" is machinery.

**Solution:** Ask: "What would I tell someone who wasn't in this session?" The answer is the wisdom. The step-by-step narrative is the machinery.

### Skipping the Compression Log

**Problem:** No meta-record of what was compressed. Future you can't tell what was in the original session or why certain things were composted.

**Solution:** Always write the compression log. It takes 5 minutes and provides invaluable context for future harvests.

## VII. Example

**Scenario:** Compressing a 2-hour architecture session about migrating from monolith to microservices.

**Step 1 (Signal):** "Let's compress this architecture session into its essential wisdom."

**Step 2 (Review):** Key moments identified:
- Decision: Start with the auth service (lowest coupling)
- Insight: Shared database is the real constraint, not the code
- Pattern: "Strangle the database, not the service" -- decouple data first
- Dead-end: Spent 30 min on event sourcing before realizing it was premature

**Step 3 (Vessel):**
- Decision record: Auth-first migration order
- Seed: "strangle-the-database-first"
- Composted: Event sourcing tangent (interesting but premature)

**Step 5 (3-month rule):**
- Cultivated: Decision record + seed (will matter for months)
- Summarized: "Explored event sourcing, decided premature for current scale" (1 sentence)
- Composted: Specific code examples discussed, debugging of CI pipeline

**Step 6 (Log):** ~12,000 tokens > ~800 tokens (93% reduction)

## VIII. Related Skills

- **memory-garden** -- The three-tier hierarchy that compression artifacts flow into
- **seed-extraction** -- For formalizing seeds discovered during compression
- **seed-library** -- Reference patterns that may surface during review
