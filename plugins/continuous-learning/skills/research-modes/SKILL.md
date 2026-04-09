---
name: research-modes
model: opus
description: Produces a Research Brief (deep mode) or Landscape Map (wide mode) document synthesizing findings into actionable recommendations. Use when: 'research this topic deeply', 'map the landscape of X', 'investigate and synthesize', 'do a wide scan of the competitive space', 'I need to understand before deciding'.
category: continuous-learning

inputs:
  - name: topic
    type: string
    description: The research topic or question to investigate
    required: true
  - name: mode
    type: string
    description: Research mode — deep (Research Brief) or wide (Landscape Map)
    required: false
outputs:
  - name: research_document
    type: ref
    format: cas-ref
    description: Research Brief (deep mode) or Landscape Map (wide mode) with findings and actionable recommendations
---

# Research Modes Skill

**Version:** 1.1
**Created:** 2026-02-02
**Author:** Manus
**Purpose:** Structured approaches for deep and wide research tasks

---

## Overview

This skill encodes two complementary research modes: **Deep Research** (focused, comprehensive investigation of a specific topic) and **Wide Research** (broad scan across multiple topics to identify patterns and opportunities). Use this skill to conduct efficient, high-quality research that produces actionable insights.

Output templates for all phases are in `references/research-output-templates.md`.

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
- Produces Research Brief or Research Synthesis document

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
- Produces Landscape Map or Opportunity Matrix

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

**Output:** Research Brief — see `references/research-output-templates.md`

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

**Output:** Source List (10-20 sources) — see `references/research-output-templates.md`

### Phase 3: Deep Reading & Note-Taking (1-3 hours)

**Process:**
1. Read each source with the research question in mind
2. Extract key insights, claims, and evidence
3. Note disagreements or contradictions between sources
4. Identify patterns and themes
5. Flag open questions or gaps

**Note-taking structure per source:**
- Main argument (1-2 sentences)
- Key insights (bulleted)
- Evidence (data points, studies, examples)
- Disagreements with other sources
- Open questions raised
- Exact quotes with page or section reference
- Relevance to research question

### Phase 4: Synthesis & Analysis (1-2 hours)

**Questions to answer:**
1. What are the major themes or patterns?
2. What do most sources agree on?
3. Where do sources disagree, and why?
4. What are the tradeoffs or tensions?
5. What gaps remain in the knowledge?

**Output:** Research Synthesis document — see `references/research-output-templates.md`

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

**Output:** Landscape Brief — see `references/research-output-templates.md`

### Phase 2: Rapid Scanning (1-2 hours)

**Process:**
1. Search broadly across the problem space
2. Skim sources quickly (5-10 minutes per source)
3. Extract 1-3 key insights per source
4. Tag sources by category, theme, or approach
5. Move on quickly (don't get stuck in details)

**Output:** Tagged source list (20-50 sources) organized by category

### Phase 3: Pattern Recognition (30-60 minutes)

**Questions to answer:**
1. What categories or clusters emerge?
2. What approaches are most common?
3. What innovations or outliers stand out?
4. What gaps or opportunities exist?

**Output:** Landscape Map — see `references/research-output-templates.md`

### Phase 4: Opportunity Matrix (30-60 minutes)

**Process:**
1. Identify potential approaches or solutions
2. Evaluate each on key dimensions (effort, impact, risk, novelty)
3. Plot on a 2x2 matrix (e.g., effort vs. impact)
4. Prioritize based on goals

**Output:** Opportunity Matrix — see `references/research-output-templates.md`

---

## Hybrid Research Mode

**Use when:**
- You need both breadth and depth
- The problem space is large and complex
- You're making a high-stakes decision

**Process:**
1. Start with Wide Research (2-4 hours)
2. Identify 2-3 promising areas
3. Conduct Deep Research on each area (2-4 hours per area)
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

- **Scope Creep:** Starting focused, ending scattered — define boundaries upfront
- **Confirmation Bias:** Only seeking supporting evidence — actively seek counterarguments
- **Analysis Paralysis:** Reading forever, never synthesizing — set time limits
- **Surface Skimming:** Reading titles, not content — take structured notes
- **No Synthesis:** Collecting info, not building understanding — answer the research question

---

## Usage Instructions

1. Read this skill before starting research
2. Choose the right mode (Deep, Wide, or Hybrid)
3. Define scope clearly (research brief or landscape brief)
4. Follow the phase structure for your chosen mode
5. Take structured notes using the templates in `references/research-output-templates.md`
6. Synthesize findings into actionable insights
7. Validate against success criteria

---

## Output

- **Deep mode:** Research Brief (scope + question) + Research Synthesis document saved to the project's `docs/research/` or `scouts/` directory
- **Wide mode:** Landscape Map + Opportunity Matrix document saved to the same location
- Format: Markdown file named `[date]_[topic]_research.md` or `[date]_[topic]_landscape.md`

---

## Examples

**Scenario 1:** "Research AI agent memory compression techniques deeply" → Research Brief defining question + Research Synthesis with findings across 8-12 sources, confidence ratings, tradeoffs table, and ranked recommendations

**Scenario 2:** "Map the landscape of open-source design system tools" → Landscape Brief + Landscape Map categorizing 30+ tools by personality/density/use-case + Opportunity Matrix scoring the top options

---

## Edge Cases

- **Topic too broad for deep mode:** Split into 2-3 focused sub-questions, run Deep Research on each, then synthesize
- **Fewer than 3 credible sources found:** Note the evidence gap explicitly in the synthesis; do not inflate confidence
- **Conflicting authoritative sources:** Surface the contradiction as a finding, do not resolve it artificially
- **Time-boxed request (e.g., "quick scan"):** Default to Wide Research with Phase 2 limited to 10-15 sources; label output as a preliminary scan

---

## Anti-Patterns

- Running Wide Research when a specific decision needs Deep Research — the outputs are structurally different and not interchangeable
- Starting Phase 4 synthesis before completing source reading — partial data produces overconfident recommendations
- Reusing a prior Research Brief for a different question without redefining scope — old scope constraints silently distort new findings
- Writing the synthesis as a summary of each source in order — synthesis must be organized by themes, not by source

---

**Related Skills:**
- `specification-writer` — Research informs specifications
- `seed-extraction` — Extract seeds from research findings
- `memory-garden` — Document research in memory for future reference

---

**Last Updated:** 2026-04-08
**Maintained By:** Manus
**Status:** Active
