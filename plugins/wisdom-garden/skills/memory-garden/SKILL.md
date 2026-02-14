# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: memory-garden
description: Cultivate and maintain organizational memory through structured knowledge capture
triggers:
  - "maintain memory garden"
  - "cultivate knowledge"
  - "organize organizational memory"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - script_execution
    - file_system
    - bash
  portable: true
  tier: 2
  agents:
    - research-agent
---

# Memory Garden Writer Skill

**Version:** 1.0  
**Created:** 2026-02-02  
**Author:** Manus  
**Purpose:** Write structured, semantically rich memory entries for efficient context management

---

## Overview

This skill encodes the pattern for writing memory entries that follow the **4-Tier Context Iceberg** and **Hierarchical Memory** principles from Dojo Genesis v0.0.19. Use this skill to create memory entries that are easy to search, retrieve, and compress.

**Philosophy:** Memory should be a garden, not a landfill. Cultivate what matters, compost what doesn't.

---

## When to Use This Skill

- Creating daily memory notes after a session
- Writing compressed summaries of conversations
- Extracting "seeds" (reusable insights) from experiences
- Documenting decisions and their rationale
- Maintaining the memory hierarchy (Tier A → Tier B → Tier C)

---

## Memory Hierarchy (From v0.0.19)

### Tier A: Raw Daily Notes
- **Location:** `memory/YYYY-MM-DD.md`
- **Purpose:** Capture everything from today's session
- **Lifespan:** 1-3 days before compression
- **Format:** Timestamped entries with context

### Tier B: Curated Wisdom
- **Location:** `MEMORY.md` (root level)
- **Purpose:** Distilled insights, decisions, patterns
- **Lifespan:** Permanent, but evolves
- **Format:** Structured sections with triggers

### Tier C: Compressed Archive
- **Location:** `memory/archive/YYYY-MM.md`
- **Purpose:** Historical record, rarely accessed
- **Lifespan:** Permanent, read-only
- **Format:** Semantic summaries

---

## Daily Memory Entry Template (Tier A)

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
- [Context or situation]
- [Keywords or signals]

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

## Curated Memory Template (Tier B)

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

## Compressed Archive Template (Tier C)

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

## The "3-Month Rule" (From Cipher)

**Rule:** If it wouldn't matter in 3 months → compress or discard.

**Keep:**
- Decisions and their rationale
- Lessons learned and patterns discovered
- Seeds (reusable insights)
- Significant events and outcomes

**Compress:**
- Routine activities ("worked on X")
- Pleasantries and confirmations
- Detailed step-by-step logs (keep summary only)

**Discard:**
- Duplicate information
- Temporary notes that were resolved
- Irrelevant tangents

---

## Semantic Compression Guidelines

### What to Keep (Verbatim)

1. **Decisions:** The choice, rationale, and context
2. **Insights:** Novel patterns or principles
3. **Seeds:** Reusable knowledge with triggers
4. **Failures:** What didn't work and why
5. **Breakthroughs:** Moments of clarity or innovation

### What to Summarize

1. **Activities:** "Worked on X, Y, Z" → "Implemented feature X"
2. **Discussions:** Long back-and-forth → Key points and outcome
3. **Research:** Detailed findings → Summary and conclusion
4. **Iterations:** Multiple attempts → Final approach and why

### What to Discard

1. **Pleasantries:** "Great work!" "Thank you!" (unless significant)
2. **Confirmations:** "Got it" "Understood" "Proceeding"
3. **Redundant logs:** Repeated information
4. **Resolved questions:** Questions that were answered and no longer relevant

---

## Memory Maintenance Cycle

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
   - Delete raw daily notes older than 7 days (after compression)
   - Keep only what matters

---

## Quality Checklist

Before finalizing a memory entry, verify:

### Daily Notes (Tier A)
- [ ] Timestamped entries with context
- [ ] Clear "What, Why, Outcome" structure
- [ ] Insights and learnings captured
- [ ] Decisions documented with rationale
- [ ] Seeds extracted with triggers
- [ ] Tags and metadata included

### Curated Memory (Tier B)
- [ ] Principles are clear and actionable
- [ ] Decisions include context and rationale
- [ ] Patterns have evidence (3+ instances)
- [ ] Seeds have clear triggers
- [ ] Compression history is updated
- [ ] Maintenance date is set

### Compressed Archive (Tier C)
- [ ] Summary captures key events
- [ ] Decisions and lessons are preserved
- [ ] Seeds are extracted
- [ ] Compression ratio is calculated
- [ ] Original files are deleted after compression

---

## Examples

**From Dojo Genesis:**
- Daily notes from v0.0.17-v0.0.23 development
- Curated wisdom from Cipher collaboration
- Compressed archives from backend migration

**Study these for:**
- How to extract seeds from experiences
- How to document decisions with context
- How to apply the 3-month rule
- How to maintain the memory hierarchy

---

## Common Pitfalls to Avoid

❌ **Hoarding Everything:** Keeping every detail → ✅ Compress ruthlessly  
❌ **Vague Insights:** "This was useful" → ✅ "This pattern applies when X"  
❌ **Missing Triggers:** Seed without context → ✅ Seed with clear "when to apply"  
❌ **No Maintenance:** Let Tier A grow forever → ✅ Compress every 3-7 days  
❌ **Duplicate Information:** Same thing in multiple places → ✅ Single source of truth

---

## Usage Instructions

1. **Read this skill** before writing memory entries
2. **Choose the right template** (Tier A, B, or C)
3. **Fill in each section** with specific, structured content
4. **Apply the 3-month rule** when compressing
5. **Maintain the cycle** every 3-7 days
6. **Run the quality checklist** before finalizing

---

## Skill Metadata

**Token Savings:** ~5,000-8,000 tokens per session (structured format enables efficient retrieval)  
**Quality Impact:** Ensures consistent memory format across sessions  
**Maintenance:** Update when new memory patterns emerge  

**Related Skills:**
- `specification-writer` - For documenting technical decisions
- `seed-extraction` - For extracting reusable insights
- `workspace-navigation` - For managing memory files efficiently

---

**Last Updated:** 2026-02-02  
**Maintained By:** Manus  
**Status:** Active
