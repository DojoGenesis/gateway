---
name: memory-garden
description: >
  Three-tier memory hierarchy for cultivating institutional knowledge with
  growth semantics. Use when writing daily notes, compressing sessions into
  wisdom, tending the memory garden, promoting artifacts between tiers, or
  applying the 3-month rule to triage what matters. Also use when someone
  asks about memory structure, tier placement, or semantic compression.
triggers:
  - "plant a new memory in the garden"
  - "tend the memory garden and promote artifacts"
  - "write a structured memory entry for this learning"
metadata:
  version: "1.1"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - research-agent
---

# Memory Garden

**Version:** 1.1
**Author:** Tres Pies Design
**Purpose:** Write structured, semantically rich memory entries for efficient context management using a three-tier hierarchy and growth semantics.

---

## I. Philosophy: Memory is a Garden, Not a Landfill

Memory should be cultivated, not accumulated. A garden requires tending -- planting what matters, composting what doesn't, and harvesting wisdom when it's ripe. The three-tier hierarchy ensures that raw experiences are refined into lasting wisdom, and that context windows remain fertile ground for new thinking rather than overgrown thickets of old data.

Growth language is not decoration -- it's a thinking tool. When we say "plant" instead of "create," we prime ourselves to consider whether this seed will actually grow. When we say "compost" instead of "delete," we acknowledge that even released material contributed to the soil.

The 3-month rule is the gardener's pruning shears: if it won't matter in 3 months, it doesn't deserve permanent cultivation.

---

## II. When to Use This Skill

- **Writing a daily note** after a session or workday
- **Deciding where an artifact belongs** in the three-tier hierarchy
- **Compressing Tier A notes** into Tier B wisdom or Tier C archives
- **Running a garden health check** to identify stale or overgrown areas
- **Onboarding someone** to the memory garden structure
- **Choosing what to cultivate vs. compost** from a long session

**When NOT to use:** If the content is a one-time reference (API docs, config values) that doesn't evolve, it's documentation, not a garden artifact. If it's a reusable pattern, it's a seed -- use the seed-extraction skill.

---

## III. The Three-Tier Hierarchy

### Tier A: Raw Daily Notes

**Purpose:** Capture everything from today's session
**Lifespan:** 1-3 days before compression
**Location:** `notes/YYYY-MM-DD_daily.md`

**Template sections:**
1. Key Activities (2-3 bullets)
2. Decisions Made (with rationale)
3. Seeds Discovered (with trigger)
4. Open Questions (to carry forward)
5. Tomorrow's Growth (next steps)

**Metadata:** `type: daily-note`, `tier: A`

**Tending rule:** Compress or promote within 3-7 days. Never let Tier A grow beyond 7 days without attention.

**Full Template:**

```markdown
# Memory: YYYY-MM-DD

**Session:** [Morning Planning | Deep Work | Creative Session | Review]
**Context:** [What we were working on]
**Duration:** [Approximate time spent]

---

## Key Activities

### [HH:MM] [Activity Name]

**What:** [Brief description of what happened]
**Why:** [The goal or motivation]
**Outcome:** [What was produced or decided]

**Insights:**
- [Specific insight or learning]
- [Pattern or principle discovered]

**Related:**
- Links to: [file, artifact, or previous memory]
- Builds on: [previous work or decision]

---

## Decisions Made

### Decision: [Short title]

**Context:** [What led to this decision]
**Options Considered:**
1. [Option A] - [Pros/Cons]
2. [Option B] - [Pros/Cons]

**Chosen:** [Selected option]
**Rationale:** [Why this was the best choice]
**Trigger:** [When to revisit this decision]

---

## Seeds Extracted

### Seed: [Name]

**Pattern:** [The reusable insight or principle]
**Why It Matters:** [The value or application]
**Trigger:** [When to apply this seed]
**Example:** [Concrete example from this session]
**Related Seeds:** [Other seeds this connects to]

---

## Open Questions

- [ ] [Question that needs resolution]
- [ ] [Uncertainty or ambiguity to clarify]

---

## Next Steps

- [ ] [Actionable task]
- [ ] [Follow-up or continuation]

---

## Metadata

**Tags:** #[category] #[topic] #[type]
**Compression Status:** Raw (not yet compressed)
**Importance:** High | Medium | Low
**Retention:** [How long to keep in Tier A before compression]
```

---

### Tier B: Curated Wisdom

**Purpose:** Distilled insights, decisions, patterns, seeds
**Lifespan:** Permanent, but evolves
**Location:** `seeds/`, `artifacts/`

**Content types:**
- Seeds (reusable patterns with triggers)
- Decision records (choice, rationale, revisit trigger)
- Philosophical reflections (deep insights about how to work)
- Conversation summaries (key outcomes from significant discussions)

**Metadata:** `type: [seed|decision|reflection|summary]`, `tier: B`

**Tending rule:** Review during `/harvest`. Update when context changes. Merge duplicates. Compost when superseded.

**Full Template:**

```markdown
# Memory (Curated Wisdom)

**Last Updated:** YYYY-MM-DD
**Maintenance Cycle:** Every 3-7 days
**Purpose:** Distilled insights, decisions, and patterns that matter beyond a single session

---

## Core Principles

### [Principle Name]

**Statement:** [Clear, concise principle]
**Origin:** [Where this came from - date, context, or experience]
**Application:** [When and how to apply this]

**Examples:**
- [Concrete example 1]
- [Concrete example 2]

**Trigger:** [Keywords or contexts that should surface this principle]

---

## Key Decisions

### [Decision Title]

**Date:** YYYY-MM-DD
**Context:** [What led to this decision]
**Decision:** [What was decided]
**Rationale:** [Why this was chosen]
**Status:** Active | Revisit on [date] | Deprecated

**Trigger:** [When to recall this decision]

---

## Patterns & Insights

### [Pattern Name]

**Observation:** [What we've noticed repeatedly]

**Evidence:**
- [Instance 1: date, context]
- [Instance 2: date, context]
- [Instance 3: date, context]

**Implication:** [What this means for future work]
**Trigger:** [When to apply this pattern]

---

## Seeds (Reusable Knowledge)

### Seed: [Name]

**Pattern:** [The reusable insight]
**Why It Matters:** [The value]
**Trigger:** [When to apply]
**Origin:** [Where this came from]
**Last Used:** YYYY-MM-DD
**Usage Count:** [Number of times applied]

---

## Compression History

| Date | Compressed From | Summary | Retained |
|------|-----------------|---------|----------|
| YYYY-MM-DD | memory/YYYY-MM-DD.md | [Brief summary] | [What was kept] |

---

## Metadata

**Total Seeds:** [Number]
**Total Decisions:** [Number]
**Total Patterns:** [Number]
**Last Maintenance:** YYYY-MM-DD
**Next Maintenance:** YYYY-MM-DD
```

---

### Tier C: Compressed Archive

**Purpose:** Historical record, rarely accessed
**Lifespan:** Permanent, read-only
**Location:** `compressions/`

**Content:** Monthly summaries, compression logs, archived decisions

**Metadata:** `type: archive`, `tier: C`, compression ratio

**Tending rule:** Rarely touched. Only surface back to Tier B if insight becomes relevant to current work.

**Full Template:**

```markdown
# Memory Archive: YYYY-MM

**Compressed:** YYYY-MM-DD
**Source:** [List of daily files compressed]
**Compression Ratio:** [X]%
**Method:** Semantic compression (3-month rule)

---

## Summary

[2-3 paragraph summary of the month's activities, focusing on decisions, lessons, and patterns]

---

## Significant Events

### [Event Name]

**Date:** YYYY-MM-DD
**What:** [Brief description]
**Impact:** [Why this mattered]
**Outcome:** [Result or consequence]

---

## Lessons Learned

1. **[Lesson Title]:** [What we learned and why it matters]
2. **[Lesson Title]:** [What we learned and why it matters]

---

## Decisions Made

| Date | Decision | Rationale | Status |
|------|----------|-----------|--------|
| YYYY-MM-DD | [Brief decision] | [Why] | [Active/Deprecated] |

---

## Seeds Extracted

| Seed Name | Pattern | Trigger |
|-----------|---------|---------|
| [Name] | [Brief pattern] | [When to apply] |

---

## Metadata

**Compression Method:** Semantic (3-month rule)
**Original Size:** [X] lines
**Compressed Size:** [Y] lines
**Compression Ratio:** [Z]%
**Retention:** Permanent (read-only)
```

---

## IV. Semantic Compression Rules

### The 3-Month Rule

**Rule:** If it wouldn't matter in 3 months, compress or compost.

### What to Cultivate (preserve verbatim or with light editing)

1. **Decisions** -- The choice, rationale, and context
2. **Insights** -- Novel patterns or principles
3. **Seeds** -- Reusable knowledge with triggers
4. **Failures** -- What didn't work and why
5. **Breakthroughs** -- Moments of clarity or innovation

### What to Summarize (reduce to 1-2 sentences)

1. **Activities** -- "Worked on X, Y, Z" > "Shipped feature X"
2. **Discussions** -- Long back-and-forth > Key points and outcome
3. **Research** -- Detailed findings > Summary and conclusion
4. **Iterations** -- Multiple attempts > Final approach and why

### What to Compost (release entirely)

1. **Pleasantries** -- "Great work!" "Thank you!"
2. **Confirmations** -- "Got it" "Understood" "Proceeding"
3. **Redundant content** -- Already captured elsewhere
4. **Resolved questions** -- Answered and no longer relevant
5. **Debugging dead-ends** -- Explored and abandoned

---

## V. Memory Maintenance Cycle

**Every 3-7 Days:**

1. **Review Tier A (Daily Notes):**
   - Identify seeds to extract
   - Identify decisions to document
   - Identify patterns to record

2. **Update Tier B (Curated Wisdom):**
   - Add new seeds, decisions, patterns
   - Update existing entries if needed
   - Remove deprecated information

3. **Compress to Tier C (Archive):**
   - Apply 3-month rule
   - Create semantic summary
   - Move to archive folder

4. **Prune:**
   - Compost raw daily notes older than 7 days (after compression)
   - Keep only what matters

---

## VI. Quality Checklist

### Daily Notes (Tier A)
- [ ] All 5 sections present (Activities, Decisions, Seeds, Questions, Tomorrow)
- [ ] Activities are concise (2-3 bullets, not a log)
- [ ] Decisions include rationale
- [ ] Seeds have pattern and trigger
- [ ] Metadata tags included (type, tier)

### Curated Wisdom (Tier B)
- [ ] Clear, specific title
- [ ] Content distilled (not raw transcript)
- [ ] Trigger or context for future recall
- [ ] Linked to related artifacts where appropriate
- [ ] Metadata complete (type, tier, date, domain)

### Compressed Archive (Tier C)
- [ ] Summary captures key events and decisions
- [ ] Seeds and lessons preserved
- [ ] Compression ratio calculated
- [ ] Original Tier A files composted after compression

---

## VII. Common Pitfalls

### Hoarding Everything in Tier A

**Problem:** Daily notes pile up beyond 7 days, creating an overgrown, unnavigable garden. Context windows fill with raw material instead of refined wisdom.

**Solution:** Apply the 3-7 day tending cycle. If a Tier A note is older than 3 days, it's time to promote, compress, or compost. Use `/harvest` to catch overdue items.

### Planting Without Triggers

**Problem:** Seeds and artifacts lack "when to recall" context. They're planted but can never be found when needed.

**Solution:** Every Tier B artifact must answer: "Under what conditions should this surface again?" Write explicit triggers -- keywords, situations, signals.

### Using Database Language

**Problem:** Writing "stored in the database" or "retrieved from memory" breaks the growth metaphor and subtly shifts thinking toward accumulation rather than cultivation.

**Solution:** Consistent garden vocabulary: plant, cultivate, harvest, compress, compost, tend, prune, surface, promote.

### Vague Insights

**Problem:** Entries like "This was useful" or "Good session" contain no retrievable knowledge.

**Solution:** Always include the specific pattern: "This pattern applies when X because Y."

### Duplicate Information

**Problem:** Same insight recorded in multiple tiers or files, leading to confusion about which is canonical.

**Solution:** Single source of truth per artifact. When promoting from Tier A to Tier B, compost the Tier A version.

---

## VIII. Example

**Scenario:** End of a workday where a team shipped a new API endpoint and discovered a useful error-handling pattern.

**Tier A daily note (planted same day):**
- Activities: Shipped `/api/v2/sessions` endpoint; refactored error middleware
- Decision: Use structured error codes instead of string messages (enables client-side handling)
- Seed discovered: "error-codes-over-strings" -- structured errors compound across services
- Open question: Should we backfill v1 endpoints with the new error format?
- Tomorrow: Backfill assessment + write integration tests

**3 days later (during `/harvest`):**
- Daily note promoted: Decision > Tier B decision record
- Seed formalized: `/plant error-codes-over-strings` > Tier B seed
- Activities composted (routine shipping, won't matter in 3 months)
- Open question carried forward to next daily note

---

## IX. Related Skills

- **compression-ritual** -- The 6-step process for distilling sessions into artifacts
- **seed-extraction** -- How to identify and formalize seeds from experiences
- **seed-library** -- Reference library of the 10 Dojo Seed Patches
- **workspace-navigation** -- For managing memory files efficiently
- **specification-writer** -- For documenting technical decisions

---

## X. Skill Metadata

**Token Savings:** ~5,000-8,000 tokens per session (structured format enables efficient retrieval)
**Quality Impact:** Ensures consistent memory format across sessions
**Maintenance:** Update when new memory patterns emerge

**When to Update This Skill:**
- When new tier patterns emerge from practice
- When compression ratios suggest template improvements
- When garden vocabulary evolves
