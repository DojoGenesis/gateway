---
name: research-synthesis
description: Synthesize multiple research files into actionable insights. Turns raw research into coherent understanding by cross-referencing sources, identifying patterns and contradictions, and producing theme-based synthesis with actionable recommendations.
triggers:
  - "synthesize these research files into insights"
  - "cross-reference these sources and find patterns"
  - "turn this raw research into actionable recommendations"
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

# Research Synthesis Skill

**Version:** 1.1
**Author:** Tres Pies Design
**Purpose:** Turn raw research into coherent understanding — identify patterns, contradictions, and actionable insights across multiple sources.

---

## I. Philosophy: From Information to Insight

Synthesis is insight generation, not information gathering. By systematically cataloging, cross-referencing, and synthesizing multiple sources, it uncovers patterns and contradictions that are not visible from any single source, leading to deeper, more actionable understanding.

Synthesis complements the other research modes:
- **Deep Research** finds answers (focused, 5 phases)
- **Wide Research** finds questions (broad, 4 phases)
- **Research Synthesis** finds patterns across sources (multi-file, 5 phases)

Deep and wide research feed into synthesis. When you have 3+ research files on a topic, synthesis extracts what no single source can reveal.

---

## II. When to Use This Skill

- When you have **3 or more research files** on a single topic
- When you need to create a **literature review** or competitive analysis
- When you want to **consolidate notes** from multiple meetings or interviews
- When `/plan-from-files` (specification-driven-development plugin) routes 3+ research files here
- After completing deep or wide research that produced multiple source documents

---

## III. The 5-Step Synthesis Workflow

### Step 1: Ingest & Catalog

Read all uploaded files. For each:
1. **Extract metadata** — author, date, key topics
2. **Summarize key content** — what does this file contribute?
3. **Identify type** — paper, article, notes, transcript, report
4. **Create internal catalog** listing all sources

**Output:** Source catalog table

```markdown
### Source Catalog

| # | File | Type | Author | Date | Key Topics | Summary |
|---|------|------|--------|------|------------|---------|
| 1 | [filename] | [type] | [author] | [date] | [topics] | [one sentence] |
```

### Step 2: Cross-Reference & Find Patterns

Systematically compare across all files:

- **Convergence** — Where do sources agree? What claims have support from multiple files?
- **Divergence** — Where do sources contradict each other? These are often the most interesting insights.
- **Gaps** — What topics appear in one source but not others? What questions does no source answer?

Create a **cross-reference matrix** mapping themes to sources:

```markdown
### Cross-Reference Matrix

| Theme | Source 1 | Source 2 | Source 3 | Consensus? |
|-------|----------|----------|----------|------------|
| [Theme A] | Supports | Supports | Silent | Partial |
| [Theme B] | Supports | Contradicts | Supports | Divergent |
| [Theme C] | Silent | Supports | Silent | Single source |
```

### Step 3: Synthesize by Themes

Structure the synthesis by **themes, NOT by individual files**. This is the cardinal rule — synthesis requires integration, not summarization.

For each theme:
1. **Summarize convergent evidence** — what do sources agree on?
2. **Highlight contradictions with citations** — where do they disagree, and why?
3. **Provide a key insight** that is not obvious from any single source

**Critical:** Contradictions are valuable. Highlight them, don't bury them. They often reveal the most interesting insights.

### Step 4: Actionable Recommendations

Based on the synthesis, provide concrete next steps:
- Each recommendation must **cite the sources** that support it
- Recommendations should be **concrete and implementable**
- Flag recommendations where evidence is thin or contradictory

### Step 5: Validate & Deliver

Before delivery:
- [ ] All claims are supported by source files
- [ ] All major themes and contradictions are covered
- [ ] Recommendations are concrete and actionable
- [ ] Every claim cites a specific source file

Deliver:
- Save as `research/[date]_synthesis_[topic].md`
- Summarize key insights
- Offer to explore specific themes in more detail

---

## IV. Best Practices

- **Organize by themes, NOT by files** — This is the cardinal rule. Synthesis is integration, not sequential summarization.
- **Contradictions are valuable** — Highlight disagreements; they reveal the most interesting insights
- **Cite your sources** — Every claim must be traceable to a specific source file
- **Actionable recommendations are the goal** — Synthesis should lead to action, not just understanding
- **Accept handoffs seamlessly** — When routed from `/plan-from-files`, begin immediately

---

## V. Quality Checklist

- [ ] Organized by themes, never by individual files
- [ ] Highlights contradictions between sources — does not silently resolve them
- [ ] Cites source files for every claim
- [ ] Produces actionable recommendations, not just summaries
- [ ] Cross-reference matrix maps themes to sources
- [ ] Accepts handoff from `/plan-from-files` when 3+ research files detected

---

## VI. Example: Dojo Architecture Research Synthesis (February 2026)

**The Problem:** After 10 days of intensive research producing 18 research documents (scouts, deep dives, competitive analysis), the findings needed to be consolidated into a coherent architectural direction for the Dojo platform.

**The Process:**

1. **Ingested 18 files:** Cataloged scouts covering MCP architecture, gateway design, plugin systems, observability, WebSocket protocols, and competitive analysis of 7 platforms.
2. **Cross-referenced:** Built a matrix mapping 12 themes across all 18 sources. Found 3 major convergences (event-driven architecture, skill-as-unit, bidirectional MCP) and 2 critical divergences (monolith vs microservice gateway, WASM vs container sandboxing).
3. **Synthesized by theme:** Produced theme-based analysis. The divergence on sandboxing was the most valuable finding — 5 sources assumed containers, but 2 deep dives on WASM showed it was more aligned with the skill-as-unit philosophy.
4. **Recommendations:** 20 Architecture Decision Records (ADRs) produced directly from the synthesis.
5. **Validated:** Every ADR cited at least 2 source documents. Contradictions were explicitly documented as trade-off analyses.

**The Outcome:** 18 research docs produced 20 ADRs in 10 days. The synthesis revealed that the WASM sandboxing approach (which only 2 out of 18 sources advocated) was actually the strongest fit — a conclusion invisible from any single document.

**Key Insight:** The 2 divergent sources on WASM were the most junior scouts, which almost got dismissed. Synthesis surfaced their argument's strength by placing it against the full evidence base. Minority positions in research often carry the strongest signal.

---

## VII. Common Pitfalls

### Pitfall 1: Summarizing Files Sequentially

**Problem:** "File 1 says X, File 2 says Y, File 3 says Z" is a book report, not synthesis. It fails to reveal cross-cutting patterns.

**Solution:** Structure by themes, never by files. The cardinal rule exists because synthesis requires integration across sources.

### Pitfall 2: Silently Resolving Contradictions

**Problem:** When sources disagree, choosing one side without acknowledging the disagreement hides the most valuable insights.

**Solution:** Always surface contradictions explicitly. Name the sources, state the disagreement, and analyze why it exists. Contradictions often indicate where the most important decisions lie.

### Pitfall 3: Missing the Minority Position

**Problem:** When 8 out of 10 sources agree, the 2 dissenting sources get dismissed. But minority positions often carry crucial signal.

**Solution:** Give dissenting sources proportional analytical weight. Ask: "What do these 2 sources see that the majority misses?" The most important insight in the Dojo architecture synthesis came from the 2 junior scouts.

### Pitfall 4: Recommendations Without Citations

**Problem:** Recommendations that aren't traced to specific sources can't be validated or revisited.

**Solution:** Every recommendation must cite at least one source. "Based on [Source 3] and [Source 7], we recommend X" — not just "We recommend X."

### Pitfall 5: Synthesizing Too Few Sources

**Problem:** With fewer than 3 sources, there isn't enough evidence for meaningful cross-referencing.

**Solution:** Synthesis requires 3+ sources. With fewer, use deep research or wide research instead. The cross-reference matrix needs at least 3 columns to reveal patterns.

---

## VIII. Related Skills

- **`research-modes`** — Provides the deep and wide research that feeds into synthesis
- **`seed-extraction`** — Extract reusable patterns from synthesis insights into persistent seeds
- **`context-ingestion`** — Routes 3+ research files to synthesis via `/plan-from-files`
- **`strategic-scout`** — Often produces the research documents that synthesis consumes
- **`compression-ritual`** — Compress the full synthesis into a distilled memory artifact

---

## IX. Integration with Other Commands

### Accepting handoff from `/plan-from-files`

When the specification-driven-development plugin's `/plan-from-files` detects 3+ research files, it routes to `/synthesize`. Accept the handoff seamlessly — the files are already identified, begin with Step 1 (Ingest & Catalog).

### Feeding into `/deep-research`

Synthesis often reveals gaps that need focused investigation. When open questions are identified, suggest `/deep-research` on specific gaps.

### Cross-plugin: wisdom-garden

Particularly valuable synthesis insights can be preserved as seeds via `/plant`.

---

## X. Output Format

```markdown
## Synthesis: [Topic]

**Date:** [Date]
**Sources:** [Number] files analyzed
**Question:** [What topic or question these files address]

### Source Catalog

| # | File | Type | Author | Date | Key Topics |
|---|------|------|--------|------|------------|
| 1 | [filename] | [type] | [author] | [date] | [topics] |

### Cross-Reference Matrix

| Theme | Source 1 | Source 2 | Source 3 | Consensus? |
|-------|----------|----------|----------|------------|
| [Theme] | [Position] | [Position] | [Position] | [Assessment] |

### Theme-Based Synthesis

#### Theme 1: [Name]

**Convergence:** [What sources agree on]
Sources: [File 1], [File 3]

**Divergence:** [Where sources disagree]
[File 2] argues [X], while [File 4] argues [Y]. This tension suggests [insight].

**Key Insight:** [Insight not obvious from any single source]

### Actionable Recommendations

1. **[Recommendation 1]** — [Description]
   - Supported by: [File 1], [File 3]
   - Confidence: High/Medium/Low

### Open Questions

- [Questions no source answers]
- [Areas needing further research]
```

---

**Last Updated:** 2026-04-06
**Status:** Active
