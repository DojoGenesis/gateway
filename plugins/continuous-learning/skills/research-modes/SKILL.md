# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: research-modes
description: Adaptive research strategies for different exploration contexts and goals
triggers:
  - "choose research approach"
  - "adapt research strategy"
  - "select exploration mode"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - file_system
    - bash
    - web_tools
  portable: true
  tier: 2
  agents:
    - research-agent
---

# Research Modes Skill

**Version:** 1.0  
**Created:** 2026-02-02  
**Author:** Manus  
**Purpose:** Structured approaches for deep and wide research tasks

---

## Overview

This skill encodes two complementary research modes: **Deep Research** (focused, comprehensive investigation of a specific topic) and **Wide Research** (broad scan across multiple topics to identify patterns and opportunities). Use this skill to conduct efficient, high-quality research that produces actionable insights.

**Philosophy:** Research is not about collecting information—it's about building understanding and making decisions.

---

## When to Use This Skill

- Planning a new feature or system architecture
- Investigating a technical problem or design challenge
- Exploring competitive landscape or market trends
- Synthesizing learnings from multiple sources
- Making informed decisions based on evidence

---

## Research Mode Selection

### Deep Research Mode

**Use when:**
- You need comprehensive understanding of a specific topic
- The decision depends on technical details
- You're evaluating a complex system or architecture
- You need to become an "expert" in a narrow domain

**Characteristics:**
- Focused scope (1-3 related topics)
- Multiple sources per topic (5-10+)
- Deep analysis and synthesis
- Produces detailed report or specification

**Timeline:** 2-8 hours

### Wide Research Mode

**Use when:**
- You're exploring a new problem space
- You need to identify patterns across multiple domains
- You're scouting for inspiration or best practices
- You want to understand the landscape before diving deep

**Characteristics:**
- Broad scope (10-50+ topics)
- Few sources per topic (1-3)
- Pattern recognition and clustering
- Produces landscape map or opportunity matrix

**Timeline:** 1-4 hours

---

## Deep Research Mode

### Phase 1: Define Scope (15-30 minutes)

**Questions to answer:**
1. What is the core question I'm trying to answer?
2. What decision will this research inform?
3. What level of detail do I need?
4. What are the boundaries (in scope vs. out of scope)?
5. What success criteria will I use?

**Output:** Research brief (1-2 paragraphs)

**Template:**

```markdown
## Research Brief

**Question:** [The core question]

**Decision:** [What this research will inform]

**Scope:**
- In scope: [Topics, domains, or questions to explore]
- Out of scope: [Topics to explicitly exclude]

**Success Criteria:**
- [ ] [Criterion 1]
- [ ] [Criterion 2]
- [ ] [Criterion 3]

**Timeline:** [Expected duration]
```

### Phase 2: Source Discovery (30-60 minutes)

**Methods:**
- Search for academic papers, technical documentation, blog posts
- Identify authoritative sources (official docs, research labs, industry leaders)
- Look for case studies, implementations, and real-world examples
- Check GitHub repositories, open-source projects, and code examples

**Quality Filters:**
- Recency (prefer sources from last 2-3 years unless historical context is needed)
- Authority (prefer official docs, peer-reviewed papers, recognized experts)
- Relevance (directly addresses the research question)
- Depth (provides technical details, not just overviews)

**Output:** Source list (10-20 sources)

**Template:**

```markdown
## Sources

### Primary Sources (Authoritative)
1. [Title] - [Author/Organization] - [Year] - [URL]
   - **Why:** [Relevance to research question]
   - **Key Claims:** [What this source argues]

### Secondary Sources (Supporting)
[Repeat structure]

### Code Examples / Implementations
[Repeat structure]
```

### Phase 3: Deep Reading & Note-Taking (1-3 hours)

**Process:**
1. Read each source with the research question in mind
2. Extract key insights, claims, and evidence
3. Note disagreements or contradictions between sources
4. Identify patterns and themes
5. Flag open questions or gaps

**Note-Taking Structure:**

```markdown
## Notes: [Source Title]

**Main Argument:** [1-2 sentences]

**Key Insights:**
- [Insight 1]
- [Insight 2]
- [Insight 3]

**Evidence:**
- [Data point, study, or example]

**Disagreements:**
- [How this contradicts other sources]

**Open Questions:**
- [What this source doesn't address]

**Quotes:**
> "[Exact quote]" (Page X)

**Relevance:** [How this informs the research question]
```

### Phase 4: Synthesis & Analysis (1-2 hours)

**Questions to answer:**
1. What are the major themes or patterns?
2. What do most sources agree on?
3. Where do sources disagree, and why?
4. What are the tradeoffs or tensions?
5. What gaps remain in the knowledge?

**Output:** Synthesis document

**Template:**

```markdown
## Research Synthesis: [Topic]

**Research Question:** [The core question]

---

### Executive Summary

[2-3 paragraphs summarizing the key findings and recommendations]

---

### Key Findings

#### Finding 1: [Theme or Pattern]

**Evidence:**
- [Source 1] claims [X]
- [Source 2] supports this with [Y]
- [Source 3] provides example: [Z]

**Confidence:** High | Medium | Low

**Implication:** [What this means for the decision]

#### Finding 2: [Theme or Pattern]

[Repeat structure]

---

### Tradeoffs & Tensions

| Dimension | Option A | Option B | Recommendation |
|-----------|----------|----------|----------------|
| [Criterion] | [Pro/Con] | [Pro/Con] | [Which and why] |

---

### Open Questions

- [ ] [Question that needs further research]
- [ ] [Uncertainty or ambiguity]

---

### Recommendations

1. **[Recommendation 1]:** [Action to take based on findings]
   - **Rationale:** [Why this is the best choice]
   - **Risk:** [What could go wrong]
   - **Mitigation:** [How to address the risk]

2. **[Recommendation 2]:** [Repeat structure]

---

### References

1. [Source 1]
2. [Source 2]
[...]
```

### Phase 5: Validation (30-60 minutes)

**Questions to ask:**
1. Did I answer the research question?
2. Are my recommendations actionable?
3. Did I consider counterarguments?
4. Are there gaps in my reasoning?
5. Would someone else reach the same conclusion?

**Validation Methods:**
- Review against success criteria
- Check for confirmation bias (did I only seek supporting evidence?)
- Test recommendations against edge cases
- Share with a peer for feedback (if available)

---

## Wide Research Mode

### Phase 1: Define Landscape (15-30 minutes)

**Questions to answer:**
1. What problem space am I exploring?
2. What are the boundaries of this landscape?
3. What am I looking for (patterns, tools, approaches)?
4. How will I know when I've covered enough ground?

**Output:** Landscape brief (1-2 paragraphs)

**Template:**

```markdown
## Landscape Brief

**Problem Space:** [The domain or challenge]

**Goal:** [What I'm trying to discover]

**Boundaries:**
- [Dimension 1: e.g., technical vs. non-technical]
- [Dimension 2: e.g., open-source vs. commercial]
- [Dimension 3: e.g., mature vs. emerging]

**Success Criteria:**
- [ ] [Covered X categories]
- [ ] [Identified Y patterns]
- [ ] [Found Z opportunities]
```

### Phase 2: Rapid Scanning (1-2 hours)

**Process:**
1. Search broadly across the problem space
2. Skim sources quickly (5-10 minutes per source)
3. Extract 1-3 key insights per source
4. Tag sources by category, theme, or approach
5. Move on quickly (don't get stuck in details)

**Output:** Tagged source list (20-50 sources)

**Template:**

```markdown
## Sources

### [Category 1]
1. [Title] - [URL]
   - **Key Insight:** [1 sentence]
   - **Tags:** #[tag1] #[tag2]

### [Category 2]
[Repeat structure]
```

### Phase 3: Pattern Recognition (30-60 minutes)

**Questions to answer:**
1. What categories or clusters emerge?
2. What approaches are most common?
3. What innovations or outliers stand out?
4. What gaps or opportunities exist?

**Output:** Landscape map

**Template:**

```markdown
## Landscape Map: [Problem Space]

### Categories Identified

1. **[Category 1]:** [Description]
   - **Examples:** [Source 1], [Source 2], [Source 3]
   - **Characteristics:** [Common traits]
   - **Maturity:** Emerging | Growing | Mature

2. **[Category 2]:** [Repeat structure]

---

### Patterns Observed

#### Pattern 1: [Name]

**Description:** [What this pattern is]

**Evidence:**
- [Source 1] does [X]
- [Source 2] does [Y]
- [Source 3] does [Z]

**Implication:** [What this suggests]

#### Pattern 2: [Repeat structure]

---

### Outliers & Innovations

- **[Source/Approach]:** [What makes this unique]

---

### Gaps & Opportunities

- **Gap 1:** [What's missing in the landscape]
  - **Opportunity:** [How this could be addressed]

- **Gap 2:** [Repeat structure]
```

### Phase 4: Opportunity Matrix (30-60 minutes)

**Process:**
1. Identify potential approaches or solutions
2. Evaluate each on key dimensions (effort, impact, risk, novelty)
3. Plot on a 2x2 matrix (e.g., effort vs. impact)
4. Prioritize based on goals

**Output:** Opportunity matrix

**Template:**

```markdown
## Opportunity Matrix

| Approach | Effort | Impact | Risk | Novelty | Priority |
|----------|--------|--------|------|---------|----------|
| [Approach 1] | Low/Med/High | Low/Med/High | Low/Med/High | Low/Med/High | 1-5 |
| [Approach 2] | [Repeat] | [Repeat] | [Repeat] | [Repeat] | [Repeat] |

---

### Top 3 Opportunities

1. **[Approach 1]:** [Why this is promising]
   - **Next Step:** [What to do to explore this further]

2. **[Approach 2]:** [Repeat structure]

3. **[Approach 3]:** [Repeat structure]
```

---

## Hybrid Research Mode

**Use when:**
- You need both breadth and depth
- The problem space is large and complex
- You're making a high-stakes decision

**Process:**
1. Start with **Wide Research** (2-4 hours)
2. Identify 2-3 promising areas
3. Conduct **Deep Research** on each area (2-4 hours per area)
4. Synthesize findings across all areas
5. Make recommendation

**Timeline:** 1-2 days

---

## Research Quality Checklist

Before finalizing research, verify:

### Scope & Focus
- [ ] Research question is clearly defined
- [ ] Scope boundaries are explicit
- [ ] Success criteria are measurable

### Source Quality
- [ ] Sources are authoritative and recent
- [ ] Multiple perspectives are represented
- [ ] Contradictions are acknowledged
- [ ] Bias is considered

### Analysis Depth
- [ ] Key findings are supported by evidence
- [ ] Tradeoffs and tensions are identified
- [ ] Open questions are flagged
- [ ] Recommendations are actionable

### Synthesis
- [ ] Patterns are clearly articulated
- [ ] Insights are connected to the research question
- [ ] Gaps and opportunities are identified
- [ ] Next steps are defined

---

## Common Pitfalls to Avoid

❌ **Scope Creep:** Starting focused, ending scattered → ✅ Define boundaries upfront  
❌ **Confirmation Bias:** Only seeking supporting evidence → ✅ Actively seek counterarguments  
❌ **Analysis Paralysis:** Reading forever, never synthesizing → ✅ Set time limits  
❌ **Surface Skimming:** Reading titles, not content → ✅ Take structured notes  
❌ **No Synthesis:** Collecting info, not building understanding → ✅ Answer the research question

---

## Usage Instructions

1. **Read this skill** before starting research
2. **Choose the right mode** (Deep, Wide, or Hybrid)
3. **Define scope** clearly (research brief or landscape brief)
4. **Follow the phase structure** for your chosen mode
5. **Take structured notes** using the templates
6. **Synthesize findings** into actionable insights
7. **Validate** against success criteria

---

## Skill Metadata

**Token Savings:** ~2,000-4,000 tokens per research session (structured approach prevents re-reading and wandering)  
**Quality Impact:** Ensures research is focused, comprehensive, and actionable  
**Maintenance:** Update when new research patterns emerge  

**Related Skills:**
- `specification-writer` - Research informs specifications
- `seed-extraction` - Extract seeds from research findings
- `memory-garden` - Document research in memory for future reference

---

**Last Updated:** 2026-02-02  
**Maintained By:** Manus  
**Status:** Active
