---
name: build-intelligence-map
model: opus
description: "Produces a cross-tool disposition matrix, behavioral cluster analysis, and a set of named YAML disposition presets by synthesizing analyses from 3+ ingested AI system prompts. Use when: \"build an intelligence map\", \"compare all agent behaviors\", \"generate disposition presets from external tools\", \"what can we learn from other tools\", \"synthesize agent analyses\"."
license: Complete terms in LICENSE.txt
category: wisdom-garden
triggers:
  - "build an intelligence map"
  - "compare all agent behaviors"
  - "generate disposition presets from external tools"
  - "synthesize agent analyses"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: agent_analyses
    type: string[]
    description: Paths to 3+ ingested AI system prompt analysis files to synthesize
    required: true
outputs:
  - name: intelligence_map
    type: ref
    format: cas-ref
    description: Cross-tool disposition matrix, behavioral cluster analysis, and named YAML disposition presets
---
# Build Intelligence Map

Meta-skill that synthesizes multiple agent behavioral analyses into strategic intelligence.

## I. Philosophy

Individual agent analysis reveals how one tool thinks. The intelligence map reveals the design space — the full range of behavioral choices the industry has explored, and where Dojo sits within that landscape. This is not competitive intelligence in the traditional sense; it is cartography of agent behavioral design.

The map serves three purposes: (1) validate that Dojo's defaults are well-positioned, (2) identify behavioral combinations no tool has tried that might be valuable, and (3) build a library of disposition presets that agents can switch between based on task context.

## II. When to Use

- After running `ingest-system-prompt` + `analyze-agent-behavior` on 3+ tools
- When evolving the ADA disposition system with new presets
- When a user asks "what are all the tools doing differently?"
- During strategic planning for agent behavior features

Do NOT use with fewer than 3 analyzed tools (insufficient data for meaningful comparison) or when the goal is to analyze a single tool (use `analyze-agent-behavior` directly).

## III. Workflow

### Step 1: Gather Analyses

Query MemoryStore for all entries with `EntryType: "system_prompt"`:
- Retrieve each stored analysis (disposition YAML + evidence catalog)
- Validate: at least 3 tools with completed analyses
- Sort by analysis date (most recent first)

### Step 2: Build the Disposition Matrix

Construct a tool x field matrix:

```
            | pacing     | depth      | tone         | initiative | validation  | error_handling |
------------|------------|------------|--------------|------------|-------------|----------------|
Cursor      | responsive | functional | professional | responsive | thorough    | retry          |
Windsurf    | rapid      | surface    | conversational| proactive | spot-check  | log-and-cont.  |
Copilot     | measured   | thorough   | professional | responsive | thorough    | escalate       |
Claude Code | measured   | thorough   | professional | responsive | thorough    | log-and-cont.  |
Dojo Default| measured   | thorough   | professional | responsive | thorough    | log-and-cont.  |
```

For each cell, include the confidence score. Color-code or annotate cells where confidence < 0.6.

### Step 3: Identify Clusters

Group tools by behavioral similarity:
- **Conservative cluster:** High validation, escalate errors, deliberate pacing
- **Velocity cluster:** Rapid pacing, surface depth, autonomous initiative
- **Balanced cluster:** Measured pacing, thorough depth, responsive initiative

For each cluster:
1. Name it descriptively
2. List member tools
3. Identify the defining characteristics (which fields cluster together)
4. Note the use case this cluster optimizes for

### Step 4: Gap Analysis

Compare the observed design space against Dojo's ADA enum values:

For each disposition field:
1. Which values do external tools use most?
2. Which values does no tool use? (unexplored territory)
3. Which combinations appear together? (natural affinities)
4. Which combinations never appear? (potential conflicts or opportunities)

Flag gaps where:
- A valid ADA value is used by 0 external tools (is it really useful?)
- An external behavioral pattern has no ADA equivalent (missing expressiveness)
- Two tools achieve similar outcomes with different disposition combinations (alternative strategies)

### Step 5: Generate Disposition Presets

From the clusters and gap analysis, propose new named presets:

```yaml
presets:
  velocity-mode:
    description: "Optimized for rapid iteration with minimal verification"
    inspired_by: [windsurf, cursor]
    disposition:
      pacing: rapid
      depth: functional
      initiative: proactive
      validation: { strategy: spot-check }
      error_handling: { strategy: retry, retry_count: 1 }

  guardian-mode:
    description: "Maximum safety for production-critical operations"
    inspired_by: [copilot-enterprise]
    disposition:
      pacing: deliberate
      depth: exhaustive
      initiative: reactive
      validation: { strategy: exhaustive, require_tests: true }
      error_handling: { strategy: escalate }

  explorer-mode:
    description: "Balanced depth with proactive suggestions for discovery"
    inspired_by: [gap-analysis]
    disposition:
      pacing: measured
      depth: thorough
      initiative: proactive
      validation: { strategy: thorough }
      error_handling: { strategy: log-and-continue }
```

Each preset must have:
- A descriptive name (kebab-case)
- A one-line description of its optimization target
- The tools it draws inspiration from (or "gap-analysis" if novel)
- A complete, valid DispositionConfig

### Step 6: Generate the Intelligence Map

Produce the final document:

```markdown
# Agent Intelligence Map
Generated: {timestamp}
Tools analyzed: {count}

## Disposition Matrix
{table from Step 2}

## Behavioral Clusters
{clusters from Step 3, with member tools and defining characteristics}

## Design Space Coverage
{gap analysis from Step 4}

### Well-Covered Regions
- {field}: {values used by 3+ tools}

### Unexplored Regions
- {field}: {values used by 0 tools}

### Surprising Combinations
- {tool} combines {field1}={value1} with {field2}={value2}, which is unusual because...

## Recommended Presets
{presets from Step 5, with full YAML}

## Strategic Recommendations
1. {recommendation with rationale}
2. {recommendation with rationale}
3. {recommendation with rationale}
```

Store the map as a MemorySeed with `SeedType: "pattern"` for future reference.

## IV. Best Practices

1. **Minimum 3 tools.** Below this threshold, clusters are meaningless and gaps are noise. Delay the map until sufficient data exists.

2. **Weight by confidence.** Low-confidence disposition values should be flagged in the matrix, not silently treated as ground truth.

3. **Name presets for purpose, not tools.** "velocity-mode" is reusable; "cursor-mode" is not. The preset should describe the behavioral optimization, not its origin.

4. **Distinguish validated from speculative.** Presets drawn from tool clusters are validated (someone shipped this combination). Presets from gap analysis are speculative (no one has tried this). Label them differently.

5. **Update, do not replace.** When new tools are analyzed, re-run the map. The matrix grows; clusters may shift; presets may evolve. Append to the existing map rather than overwriting.

## V. Quality Checklist

- [ ] Disposition matrix includes all analyzed tools + Dojo defaults
- [ ] Every cell has a confidence score
- [ ] At least 2 clusters identified with 2+ member tools each
- [ ] Gap analysis covers all 7 core ADA fields
- [ ] At least 3 presets generated (1 from clusters, 1 from gaps, 1 hybrid)
- [ ] All presets produce valid DispositionConfig YAML
- [ ] Strategic recommendations are specific and actionable
- [ ] Intelligence map stored as MemorySeed with type "pattern"

## Output

- A markdown intelligence map document (in-session or saved to `memory/intelligence-maps/YYYY-MM-DD_intelligence_map.md`) with the full disposition matrix, behavioral clusters, gap analysis, and recommended presets
- A set of named YAML disposition presets ready to import into Dojo ADA
- A MemorySeed stored with `SeedType: "pattern"` for future retrieval

## Examples

**Scenario 1:** User says "build an intelligence map from the 4 tools we've analyzed" → skill queries MemoryStore, constructs a 4-tool x 7-field disposition matrix, identifies 2 clusters, runs gap analysis, proposes 3 new presets (velocity-mode, guardian-mode, explorer-mode), and saves the map.

**Scenario 2:** User says "what behavioral combinations haven't any tools tried?" → skill runs gap analysis step only, returns unexplored disposition field combinations with rationale for why they might be valuable.

## Edge Cases

- If fewer than 3 tools are analyzed, return an error explaining the minimum threshold and list which tools are currently available in MemoryStore.
- If two tools produce identical disposition YAML, note the duplication in the matrix and treat them as a single data point for cluster analysis.

## Anti-Patterns

- Running the map with only 2 tools — with 2 data points there are no clusters, only a comparison. Use `analyze-agent-behavior` instead.
- Naming presets after source tools (e.g., "cursor-mode") — preset names must describe the behavioral optimization target, not the tool that inspired it.
