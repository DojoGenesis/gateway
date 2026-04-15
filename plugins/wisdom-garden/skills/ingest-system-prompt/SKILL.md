---
name: ingest-system-prompt
model: sonnet
description: "Produces a stored MemoryEntry (structured sections + ADA disposition indicators) and one or more MemorySeeds by parsing a system prompt from an external AI tool into Dojo memory. Use when: \"ingest this system prompt\", \"store this prompt in memory\", \"parse this agent's rules\", \"import this Cursor prompt\", \"analyze what this prompt does\"."
license: proprietary
category: wisdom-garden
triggers:
  - "ingest this system prompt"
  - "store this prompt in memory"
  - "parse this agent's rules"
  - "analyze what this prompt does"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: system_prompt
    type: string
    description: The external AI system prompt to parse and store in Dojo memory
    required: true
outputs:
  - name: memory_entry
    type: string
    description: Stored MemoryEntry with structured sections and ADA disposition indicators, plus one or more MemorySeeds
---
## I. Philosophy

A system prompt is not just configuration — it is a behavioral contract. Every
rule, constraint, and instruction in a system prompt encodes assumptions about
pace, depth, tone, initiative, and trust. Ingesting it properly means
converting that implicit behavioral contract into explicit, queryable knowledge
that the Dojo agent can reason about and apply.

The goal is not to replicate the foreign prompt verbatim. It is to extract the
*shape* of the intended agent behavior and store it in a form the Dojo memory
system can cross-reference against current dispositions.

## II. When to Use

Use this skill when:

- A user imports a system prompt from Cursor, Windsurf, GitHub Copilot, Cline,
  Continue, or any other AI coding or writing tool.
- An organization wants to audit what behavioral rules are governing a
  third-party AI instance.
- A developer is migrating from one AI tool to Dojo and needs to preserve the
  intended agent behavior.
- A team wants to compare the behavioral profile of two different system prompts.
- A system prompt needs to be version-controlled in the Dojo memory store.

Do not use this skill to execute or simulate the foreign system prompt. Its
purpose is ingestion and analysis only.

## III. Workflow

**Step 1 — Read the system prompt file.**

Accept either a file path or raw text. If a file path is given, read its
contents. If the source tool is not stated explicitly, attempt to identify it
from the file header, filename convention (e.g., `.cursorrules`,
`copilot-instructions.md`), or ask the user before proceeding.

**Step 2 — Identify the source tool.**

Determine the originating tool. This becomes the `source_tool` metadata value.
If the version or date is available (from filename, header, or git history),
capture it as the `version` value. Use `"unknown"` if not determinable.

**Step 3 — Parse into structured sections.**

Scan the prompt for structural markers: markdown headings, numbered lists,
labeled blocks (e.g., `## Rules`, `## Constraints`, `<!-- TOOLS -->`), or
implicit paragraph groupings. Map content into the following canonical sections:

- `core_instructions` — the primary task or role definition
- `behavioral_rules` — explicit dos and don'ts
- `tool_definitions` — any tool names, function signatures, or MCP endpoints
  defined within the prompt
- `constraints` — hard limits (token caps, forbidden outputs, format rules)
- `safety_rules` — content policies, refusal conditions, escalation paths

If a section cannot be identified, assign content to `core_instructions` by
default. Record unmapped sections as `unclassified`.

**Step 4 — Extract behavioral indicators.**

Scan each section for language patterns that signal agent disposition. Map them
to Dojo ADA (Agent Disposition Architecture) dimensions:

| Pattern | Dimension | Indicator |
|---|---|---|
| "be concise", "brief responses" | pacing | rapid |
| "thorough analysis", "explore fully" | depth | exhaustive |
| "professional tone", "formal" | tone | professional |
| "casual", "conversational" | tone | casual |
| "ask before acting", "confirm first" | initiative | reactive |
| "proceed autonomously", "just do it" | initiative | proactive |
| "never modify", "read-only" | trust | constrained |
| "full access", "execute freely" | trust | elevated |

Capture all matched indicators in a structured map. If conflicting signals
appear in the same prompt, note the conflict — do not resolve it silently.

**Step 5 — Store as MemoryEntry.**

Call `gateway.MemoryStore.Store()` with the following structure:

```
EntryType: "system_prompt"
Content: <full parsed sections as structured JSON string>
Metadata: {
  "context_type": "system_prompt",
  "source_tool": "<identified tool name>",
  "version": "<date or tag if known, else 'unknown'>",
  "section_count": "<number of parsed sections>",
  "has_tool_definitions": "<true|false>"
}
```

Capture the returned entry ID for output.

**Step 6 — Store key patterns as MemorySeeds.**

For each behavioral indicator identified in Step 4, create a MemorySeed using
`CreateUserSeed` (use this until `CreateSystemSeed` is available) with:

```
SeedType: "knowledge"
Content: "<dimension>: <indicator> — sourced from <source_tool>"
Metadata: {
  "dimension": "<ada dimension>",
  "indicator": "<indicator value>",
  "source_entry_id": "<entry ID from Step 5>",
  "source_tool": "<tool name>"
}
```

Group related indicators when they reinforce the same dimension rather than
creating one seed per pattern match.

**Step 7 — Output summary.**

Return to the user:

```
Stored entry ID: <entry_id>
Source tool: <source_tool>
Sections parsed: <list of section names>
Behavioral indicators:
  pacing: <value or "not detected">
  depth: <value or "not detected">
  tone: <value or "not detected">
  initiative: <value or "not detected">
  trust: <value or "not detected">
Conflicts detected: <list or "none">
Seeds stored: <count>
```

## IV. Best Practices

- Always confirm the source tool before storing. An incorrectly labeled entry
  pollutes behavioral cross-reference queries.
- Preserve the original prompt text in the `Content` field even after parsing.
  The structured sections are an overlay, not a replacement.
- When parsing sections, prefer over-categorization to under-categorization.
  A safety rule accidentally stored as a behavioral rule is less harmful than
  a safety rule stored as unclassified.
- Do not infer behavioral indicators that are not textually grounded. If the
  prompt is silent on tone, mark tone as "not detected" rather than defaulting
  to "professional".
- When storing MemorySeeds, use `CreateUserSeed` with the metadata workaround
  (`source_entry_id` linking back to the MemoryEntry) to maintain traceability
  until `CreateSystemSeed` is implemented.
- If the system prompt contains tool definitions (MCP server refs, function
  schemas), flag them separately. They may warrant a dedicated tool-registry
  ingestion pass.

## V. Quality Checklist

Before completing this skill, verify:

- [ ] Source tool identified and recorded (not defaulted to "unknown" if
  determinable)
- [ ] All five canonical sections attempted (even if some are empty)
- [ ] Behavioral indicators extracted using textual evidence, not inference
- [ ] Conflicting indicators surfaced explicitly in output
- [ ] MemoryEntry stored with complete metadata map (all four keys populated)
- [ ] At least one MemorySeed stored if any behavioral indicator was detected
- [ ] Entry ID included in final output
- [ ] No behavioral simulation of the foreign prompt performed

## Output

- A MemoryEntry stored via `gateway.MemoryStore.Store()` with `EntryType: "system_prompt"` and all five canonical sections in the Content field
- One or more MemorySeeds storing detected behavioral indicators with dimension metadata and `source_entry_id`
- A text summary returned to the user listing: stored entry ID, sections parsed, detected indicators per dimension, conflicts, and seed count

## Examples

**Scenario 1:** User pastes a Cursor `.cursorrules` file → skill identifies source as Cursor, parses into behavioral_rules and constraints sections, detects `pacing: rapid` and `initiative: responsive`, stores the MemoryEntry, creates 2 seeds, and returns the summary with entry ID.

**Scenario 2:** User provides a file path `~/Downloads/copilot-instructions.md` → skill reads the file, identifies Copilot as the source tool, parses 4 sections, detects a conflict between `initiative: reactive` and `initiative: proactive` in different sections, surfaces the conflict in the summary, and stores the entry with both signals documented.

## Edge Cases

- If the file is shorter than 200 words, still ingest it but flag the analysis as low-fidelity — there is insufficient text to detect reliable behavioral signals.
- If the source tool cannot be identified from the file name, header, or content, ask the user before storing. An unlabeled entry pollutes behavioral queries.

## Anti-Patterns

- Defaulting undetected dimensions to Dojo's standard values — if the prompt is silent on a dimension, the correct value is "not detected", not a borrowed default.
- Simulating or executing the foreign prompt's instructions — this skill ingests for analysis only; it does not adopt the foreign agent's behavior.
