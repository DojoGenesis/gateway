---
name: research-synthesis
model: sonnet
description: Produces a unified Synthesis Document organized by themes with cross-source evidence, contradictions surfaced, and concrete recommendations. Use when: 'synthesize these research files', 'find patterns across these sources', 'consolidate my notes into insights', 'create a literature review', 'what do these sources agree and disagree on'.
category: continuous-learning

inputs:
  - name: research_files
    type: string[]
    description: Paths to 3+ research files to synthesize
    required: true
outputs:
  - name: synthesis_document
    type: ref
    format: cas-ref
    description: Synthesis Document organized by themes with cross-source evidence, contradictions surfaced, and concrete recommendations
---

# Research Synthesis Engine

**Version:** 1.0  
**Created:** 2026-02-08  
**Author:** Manus AI  
**Purpose:** To turn raw research into a coherent understanding of a topic, identifying patterns, contradictions, and key recommendations.

---

## I. The Philosophy: From Information to Insight

This skill transforms the act of research from a process of information gathering to a process of insight generation. By systematically cataloging, cross-referencing, and synthesizing multiple sources, it uncovers patterns and contradictions that are not visible from any single source, leading to a deeper, more actionable understanding.

---

## II. When to Use This Skill

- **When you have 3 or more research files** on a single topic.
- **When you need to create a literature review** or a competitive analysis.
- **When you want to consolidate notes** from multiple meetings or interviews.
- **When the `planning-with-files` meta-skill routes to this mode.**

---

## III. The Workflow

This is a 5-step workflow for synthesizing multiple research files.

### Step 1: File Ingestion and Cataloging

**Goal:** Read all uploaded research files and create a structured catalog.

1.  **Read Files:** Use the `file` tool to read each uploaded file (use `view` for PDFs).
2.  **Extract Metadata:** Identify author, date, and key topics for each file.
3.  **Create Catalog:** Create an internal catalog document listing all files and their metadata.

### Step 2: Cross-Referencing and Pattern Identification

**Goal:** Identify patterns, contradictions, and connections across all files.

1.  **Identify Themes:** Read through the catalog to identify recurring themes.
2.  **Note Convergence:** Where do the sources agree?
3.  **Note Divergence:** Where do the sources disagree or contradict each other?
4.  **Identify Gaps:** What topics are mentioned in one source but not others?
5.  **Create Cross-Reference Matrix:** Create an internal matrix to map themes to sources.

### Step 3: Synthesis Document Creation

**Goal:** Create a unified synthesis document that integrates findings from all files.

1.  **Organize by Themes:** Structure the document by the themes identified in Step 2 (not by individual files).
2.  **Summarize Evidence:** For each theme, summarize the convergent evidence and highlight contradictions.
3.  **Provide Key Insights:** For each theme, provide a key insight that is not obvious from any single source.
4.  **Include Actionable Recommendations:** Provide a list of concrete next steps based on the synthesis.
5.  **Cite Sources:** Include citations to the specific files that support each claim.

### Step 4: Validation and Refinement

**Goal:** Ensure the synthesis is accurate, complete, and actionable.

1.  **Review for Accuracy:** Check that all claims are supported by the source files.
2.  **Check for Completeness:** Ensure all major themes and contradictions are covered.
3.  **Verify Actionability:** Ensure that recommendations are concrete and can be implemented.

### Step 5: Delivery

**Goal:** Deliver the synthesis to the user with clear next steps.

1.  **Send Synthesis:** Use the `message` tool to send the synthesis document as an attachment.
2.  **Summarize:** Briefly summarize the key insights and recommendations.
3.  **Offer Deeper Dive:** Offer to answer questions or explore specific themes in more detail.

---

## IV. Best Practices

- **Organize by Themes, Not by Files:** Synthesis requires integration, not just summarization.
- **Contradictions Are Valuable:** Highlight disagreements between sources—they often reveal the most interesting insights.
- **Actionable Recommendations Are the Goal:** A synthesis should lead to action.
- **Cite Your Sources:** Every claim in the synthesis should be traceable to a specific source file.

---

## V. Quality Checklist

- [ ] Is the synthesis organized by themes, not by individual files?
- [ ] Does the synthesis highlight both convergent evidence and contradictions?
- [ ] Does the synthesis provide actionable recommendations?
- [ ] Does the synthesis cite the source files for its claims?

---

## Output

- A Synthesis Document saved to the project's `docs/research/` or `scouts/` directory
- Named: `[date]_[topic]_synthesis.md`
- Structure: File Catalog, Theme Map, Convergence Evidence, Contradictions, Actionable Recommendations, Source Citations

---

## Examples

**Scenario 1:** User provides 5 research notes on MCP architecture patterns → Synthesis Document with 3 themes (lifecycle management, tool registration, error handling), convergence points, one major contradiction between sources surfaced, and 4 prioritized implementation recommendations

**Scenario 2:** "Consolidate my notes from these 4 competitive analysis files" → Synthesis Document organized by competitive dimensions (pricing, features, positioning), gaps identified across all four sources, and a recommended differentiation strategy

---

## Edge Cases

- **Fewer than 3 files:** Can still synthesize, but note that cross-referencing is limited with 1-2 sources; recommend gathering additional sources before drawing strong conclusions
- **Files in different formats (PDF, markdown, raw text):** Use appropriate tool per format; normalize to the same note structure before cross-referencing
- **Contradictions cannot be resolved from the files alone:** Surface them explicitly as open questions requiring a primary source or domain expert consultation

---

## Anti-Patterns

- Summarizing each file in sequence rather than organizing by theme — produces a summary list, not a synthesis
- Forcing consensus when sources genuinely disagree — contradictions are findings, not errors to be smoothed over
- Treating the file with the most detail as the most authoritative — recency and methodology matter more than length
- Producing recommendations before completing the cross-reference step — conclusions drawn from partial reading miss contradicting evidence
