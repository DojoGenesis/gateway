---
name: analyze-agent-behavior
model: opus
description: "Map an external AI tool's behavioral patterns to Dojo ADA disposition fields by analyzing its ingested system prompt. Produces a disposition YAML approximation and similarity score against Dojo defaults. Use when understanding how other agents think or when designing new disposition presets. Trigger phrases: \"analyze this agent's behavior\", \"map prompt to disposition\", \"compare agent behavior\", \"what disposition does this tool use\", \"reverse-engineer agent personality\"."
license: Complete terms in LICENSE.txt
category: wisdom-garden
triggers:
  - "analyze this agent's behavior"
  - "map prompt to disposition"
  - "compare agent behavior"
  - "reverse-engineer agent personality"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: system_prompt
    type: string
    description: The external AI system prompt to analyze and map to Dojo ADA disposition fields
    required: true
outputs:
  - name: disposition_approximation
    type: string
    description: Disposition YAML approximation and similarity score against Dojo defaults derived from the ingested system prompt
---
# Analyze Agent Behavior

Reverse-engineer an AI tool's behavioral model from its system prompt and express it as a Dojo ADA disposition.

## I. Philosophy

Every AI coding tool embeds a behavioral model in its system prompt — rules about when to be verbose vs. terse, when to ask vs. act, how to handle errors, what to prioritize. These behavioral models are rarely documented as such; they emerge from scattered instructions like "be concise," "always verify before executing," and "prefer simple solutions."

This skill makes the implicit explicit. By mapping these scattered instructions to Dojo's structured ADA disposition fields, we gain two things: (1) a precise vocabulary for comparing how different tools behave, and (2) a library of validated disposition presets we can import into our own agents.

The goal is not to copy other tools but to learn from the design space they have explored.

## II. When to Use

- After `ingest-system-prompt` has stored a system prompt as a MemorySeed
- When designing a new agent disposition and wanting to draw from proven patterns
- When comparing behavioral approaches across multiple AI tools
- When a user asks "how does Cursor/Copilot/Windsurf handle X?"

Do NOT use for analyzing Dojo's own system prompts (that is self-reflection, not external analysis) or for prompts that are not agent system prompts (marketing copy, documentation, etc.).

## III. Workflow

### Step 1: Retrieve the Ingested Prompt

Load the system prompt from MemoryStore using the entry ID or source tool name:
- Query: `Search(ctx, {Text: source_tool_name, EntryType: "system_prompt"}, 1)`
- Validate: entry exists and has parsed sections in metadata

### Step 2: Extract Behavioral Indicators

Scan the prompt text for patterns that map to ADA fields. Use this evidence table:

**Pacing indicators:**
| Signal | Disposition value | Confidence |
|--------|------------------|------------|
| "think step by step", "take your time", "be thorough" | deliberate | 0.8 |
| "be efficient", "avoid unnecessary output" | measured | 0.7 |
| "respond quickly", "be concise", "keep it short" | responsive | 0.8 |
| "minimize latency", "stream immediately", "no preamble" | rapid | 0.9 |

**Depth indicators:**
| Signal | Disposition value | Confidence |
|--------|------------------|------------|
| "brief overview", "high-level summary" | surface | 0.7 |
| "working solution", "practical answer" | functional | 0.7 |
| "comprehensive analysis", "consider alternatives" | thorough | 0.8 |
| "exhaustive review", "leave nothing unexamined" | exhaustive | 0.9 |

**Tone indicators:**
| Signal | Disposition value | Confidence |
|--------|------------------|------------|
| "formal language", "professional correspondence" | formal | 0.8 |
| "clear and professional", "business-appropriate" | professional | 0.7 |
| "natural conversation", "friendly tone" | conversational | 0.7 |
| "casual", "informal", "like talking to a friend" | casual | 0.8 |

**Initiative indicators:**
| Signal | Disposition value | Confidence |
|--------|------------------|------------|
| "only do what is asked", "wait for instructions" | reactive | 0.9 |
| "suggest when appropriate", "offer alternatives" | responsive | 0.7 |
| "anticipate needs", "proactively identify issues" | proactive | 0.8 |
| "act independently", "execute without asking" | autonomous | 0.9 |

**Validation strategy indicators:**
| Signal | Disposition value | Confidence |
|--------|------------------|------------|
| No validation mentioned | none | 0.5 |
| "verify key assumptions" | spot-check | 0.7 |
| "always verify", "check your work" | thorough | 0.8 |
| "test everything", "never skip validation" | exhaustive | 0.9 |

**Error handling indicators:**
| Signal | Disposition value | Confidence |
|--------|------------------|------------|
| "stop on error", "fail immediately" | fail-fast | 0.8 |
| "log and continue", "note the error" | log-and-continue | 0.7 |
| "retry", "try again with different approach" | retry | 0.8 |
| "ask for help", "escalate to user" | escalate | 0.8 |

### Step 3: Resolve Conflicts

When multiple indicators point to different values for the same field:
1. Weight by confidence score
2. If tied, prefer the more conservative (less autonomous) value
3. If a field has no indicators, mark as "unspecified" with confidence 0.0
4. Document the conflicting signals in the analysis

### Step 4: Generate Disposition YAML

Produce a complete DispositionConfig YAML:

```yaml
# Generated from: {source_tool_name}
# Analysis date: {timestamp}
# Overall confidence: {weighted_average}

pacing: {value}           # confidence: {score}
depth: {value}            # confidence: {score}
tone: {value}             # confidence: {score}
initiative: {value}       # confidence: {score}

validation:
  strategy: {value}       # confidence: {score}
  require_tests: {bool}
  require_docs: {bool}

error_handling:
  strategy: {value}       # confidence: {score}
  retry_count: {n}

collaboration:
  style: {value}          # confidence: {score}
  check_in_frequency: {value}

reflection:
  frequency: {value}
  format: structured
  triggers: [{list}]
```

### Step 5: Score Similarity to Dojo Defaults

Compare the generated disposition against Dojo's default values:
- Default: pacing=measured, depth=thorough, tone=professional, initiative=responsive
- Default: validation=thorough, error_handling=log-and-continue, collaboration=consultative

Similarity score = (matching fields / total fields) weighted by confidence.

### Step 6: Generate Analysis Report

Produce a markdown report:

```markdown
# Behavioral Analysis: {source_tool_name}

## Disposition Summary
| Field | Value | Confidence | Dojo Default | Match? |
|-------|-------|-----------|-------------|--------|

## Key Behavioral Differences
- {field}: {tool_value} vs Dojo {default_value}. Evidence: "{quoted signal}"

## Recommended Disposition Preset
Name: `{tool_name}-inspired`
Rationale: {why this combination is interesting}

## Evidence Catalog
{list of all extracted signals with source quotes}
```

## IV. Best Practices

1. **Ground in evidence.** Every disposition value must cite a specific phrase from the prompt. No inference without textual support.

2. **Confidence is honest.** A 0.5 confidence means "weak signal" — do not round up to make the analysis look cleaner.

3. **Unspecified is valid.** If a prompt says nothing about error handling, the correct answer is "unspecified," not a guess.

4. **Compare, do not judge.** This skill describes behavioral models, not ranks them. "Rapid pacing" is not worse than "deliberate" — they serve different contexts.

5. **Preserve nuance.** Some prompts have conditional behavior ("be concise for simple questions, thorough for complex ones"). Document the condition, map to the dominant mode, note the conditional in the analysis.

## V. Quality Checklist

- [ ] Every disposition field has a value, confidence score, and evidence citation
- [ ] Conflicting signals are documented, not silently resolved
- [ ] Generated YAML is valid and parseable by DispositionConfig loader
- [ ] Similarity score computed against current Dojo defaults
- [ ] Analysis report includes all evidence with source quotes
- [ ] No disposition values assigned without textual evidence
- [ ] Unspecified fields explicitly marked as such

## Output

- A disposition YAML block (`[tool-name]-inspired` preset) suitable for import into Dojo ADA
- A markdown analysis report with the disposition matrix table, key behavioral differences, evidence catalog, and similarity score against Dojo defaults

## Examples

**Scenario 1:** User provides a Cursor `.cursorrules` file → skill reads it, maps "be concise" to `pacing: rapid`, "verify before applying" to `validation: thorough`, and produces a `cursor-inspired` YAML preset plus a one-page report with quoted evidence.

**Scenario 2:** User asks "how does Copilot handle errors compared to Dojo?" → loads the ingested Copilot prompt from MemoryStore, extracts error handling signals, compares to Dojo defaults, and produces a two-row comparison table with evidence quotes.

## Edge Cases

- If the ingested prompt has no text for a dimension, mark it `unspecified` with confidence 0.0 — do not default to Dojo values.
- If fewer than 3 behavioral indicators are found in total, return a partial analysis and flag it as insufficient for a reliable preset.

## Anti-Patterns

- Assigning a disposition value because it "seems right" for a well-known tool rather than because the prompt text supports it — every value requires a quoted source phrase.
- Silently resolving conflicting signals by picking the higher-confidence one without documenting the conflict — conflicts must appear in the report.
