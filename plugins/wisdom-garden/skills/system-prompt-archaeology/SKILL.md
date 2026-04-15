---
name: system-prompt-archaeology
model: sonnet
description: "Produces a named ADA disposition preset (YAML) and a layered behavioral analysis (structural decomposition + evidence-cited classifications) from a system prompt. Use when: \"analyze this system prompt\", \"extract disposition from prompt\", \"reverse-engineer agent behavior\", \"what patterns does this prompt use\", \"build a preset from this prompt\"."
category: wisdom-garden
triggers:
  - "analyze this system prompt"
  - "extract disposition from prompt"
  - "reverse-engineer agent behavior"
  - "build a preset from this prompt"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: system_prompt
    type: string
    description: The system prompt to decompose and classify into behavioral patterns
    required: true
outputs:
  - name: disposition_preset
    type: string
    description: Named ADA disposition preset (YAML) and layered behavioral analysis with structural decomposition and evidence-cited classifications
---
# System Prompt Archaeology

## I. Philosophy

System prompts are the DNA of agent behavior. Reading them carefully reveals what the
designers optimized for -- and what they sacrificed. Every constraint implies a past
failure; every encouragement implies a desired behavior that doesn't happen naturally.

Archaeology is the right metaphor: you are uncovering layers of intent deposited over
time. The earliest layers are identity and safety. Later layers add tool definitions,
behavioral nudges, output formatting. The strata tell a story about what went wrong
in production and how the team responded.

This skill does not judge prompts as good or bad. It classifies them structurally so
their patterns can be reused, adapted, or deliberately avoided.

## II. When to Use

- Analyzing a competitor or peer AI tool's system prompt (from Piebald-AI/claude-code-system-prompts, user paste, or extraction)
- Building ADA disposition presets from observed agent behavior
- Comparing behavioral philosophies across tools (Cursor vs Windsurf vs Copilot)
- Designing your own system prompt by studying what works elsewhere
- Auditing your own prompts for unintentional constraint patterns

Do NOT use when:
- You need to write a system prompt from scratch (use specification-writer instead)
- You are debugging runtime agent behavior (use debugging skill instead)
- The prompt is shorter than 200 words (too thin for meaningful decomposition)

## III. Workflow

### Step 1: Acquire the System Prompt

Source the raw text. Preferred sources in order of reliability:
- `Piebald-AI/claude-code-system-prompts` GitHub repo (versioned, dated)
- User-provided paste from tool extraction (jailbreak, prompt leak, official docs)
- Behavioral inference from tool observation (lowest fidelity -- note this clearly)

Record: tool name, version, date acquired, acquisition method, and confidence level.

### Step 2: Decompose into Structural Layers

Parse the prompt into these canonical layers (not all will be present):

| Layer | What to Look For |
|-------|-----------------|
| **Identity** | Name, role definition, persona framing ("You are...") |
| **Safety/Constraints** | Refusal patterns, content policy, boundary rules |
| **Tool Definitions** | Available tools, parameter schemas, usage instructions |
| **Behavioral Rules** | Pacing, initiative, verbosity, formatting preferences |
| **Output Formatting** | Response structure, markdown usage, citation style |
| **Context Injection** | System reminders, memory sections, dynamic context |
| **Meta-Instructions** | Instructions about how to handle instructions |

Tag each section with its layer. Note sections that span multiple layers.

### Step 3: Extract Behavioral Signals

For each behavioral rule, classify along these dimensions:

- **Pacing**: Deliberate (think before acting) / Responsive (act then adjust) / Rapid (minimize latency)
- **Depth**: Surface (answer directly) / Functional (explain reasoning) / Exhaustive (cover all angles)
- **Tone**: Formal / Professional / Conversational / Casual
- **Initiative**: Passive (wait for instructions) / Balanced (suggest when relevant) / Proactive (anticipate needs)
- **Validation**: None / Self-check / Ask-user / Multi-step verification
- **Error Handling**: Silent / Acknowledge / Explain-and-retry / Escalate

Document evidence for each classification -- quote the specific prompt text.

### Step 4: Map to ADA Disposition Preset

Translate behavioral signals into ADA-compatible disposition format:

```yaml
preset_name: "[Tool]-[Version]-[Archetype]"
source_tool: "[Tool name and version]"
extracted_date: "[ISO date]"
behavioral_cluster: "[Deep Investigator|Rapid Builder|Balanced Assistant|Minimal Responder]"
dimensions:
  pacing: "[deliberate|responsive|rapid]"
  depth: "[surface|functional|exhaustive]"
  tone: "[formal|professional|conversational|casual]"
  initiative: "[passive|balanced|proactive]"
  validation: "[none|self-check|ask-user|multi-step]"
  error_handling: "[silent|acknowledge|explain-retry|escalate]"
constraints_taxonomy:
  hard_refusals: ["list of absolute refusal categories"]
  soft_boundaries: ["list of conditional restrictions"]
  encouraged_behaviors: ["list of explicitly promoted patterns"]
notable_patterns:
  - "Pattern description with evidence"
```

### Step 5: Validate Against Known Clusters

Compare the extracted preset against the four behavioral cluster archetypes:

- **Deep Investigator**: High depth, deliberate pacing, proactive initiative, multi-step validation
- **Rapid Builder**: Surface-to-functional depth, rapid pacing, balanced initiative, self-check validation
- **Balanced Assistant**: Functional depth, responsive pacing, balanced initiative, ask-user validation
- **Minimal Responder**: Surface depth, rapid pacing, passive initiative, no validation

If the preset doesn't fit cleanly, document the hybrid pattern -- these are often the
most interesting findings because they reveal intentional design tradeoffs.

## IV. Best Practices

1. **Quote, don't paraphrase.** Every behavioral classification must cite the specific prompt text that supports it. Archaeology without evidence is speculation.
2. **Note what's absent.** Missing layers are as informative as present ones. A prompt with no error handling instructions has made a deliberate (or accidental) choice.
3. **Date everything.** System prompts change frequently. An undated analysis loses value within weeks.
4. **Compare across versions.** When multiple versions of the same tool's prompt are available, diff them. The changes reveal what broke in production.
5. **Separate observation from interpretation.** Report what the prompt says before analyzing what it means.

## V. Quality Checklist

- [ ] Source prompt is recorded with tool name, version, date, and acquisition method
- [ ] All applicable structural layers are identified and tagged
- [ ] Behavioral signals are classified with quoted evidence for each dimension
- [ ] ADA disposition preset is generated in valid YAML format
- [ ] Preset is compared against known behavioral clusters
- [ ] Hybrid patterns (if any) are documented with reasoning
- [ ] Notable design tradeoffs are called out explicitly
- [ ] Analysis is dated and versioned

## VI. Common Pitfalls

- **Inferring intent without evidence.** "They probably meant X" is not archaeology. Stick to what the prompt actually says.
- **Treating all constraints as safety.** Many constraints are UX decisions (formatting, verbosity) not safety decisions. Conflating them obscures both.
- **Ignoring tool definitions.** The tools available to an agent shape its behavior as much as its behavioral rules. A prompt that gives an agent file-write access is fundamentally different from one that doesn't.
- **Single-version analysis.** One snapshot tells you what the prompt IS. Multiple snapshots tell you what it's BECOMING. Always seek version history.
- **Over-fitting to clusters.** The four archetypes are starting points, not boxes. Most real prompts are hybrids. Force-fitting loses nuance.

## VII. Example

**Input:** Claude Code system prompt (v2026-03, acquired from Piebald-AI repo)

**Layer decomposition:** Identity (lines 1-5), Safety (lines 6-120), Tool Definitions (lines 121-800), Behavioral Rules (lines 801-950), Output Formatting (lines 951-1000), Context Injection (dynamic system-reminder blocks)

**Key behavioral signals:**
- Pacing: Deliberate (explicit "think before acting" patterns, planning steps)
- Depth: Exhaustive (multi-file analysis, broad search before narrow)
- Initiative: Proactive ("complete the task fully", anticipate needs)
- Validation: Multi-step (read before edit, verify after write)

**Disposition preset:** `Claude-Code-2026Q1-DeepInvestigator`
**Cluster match:** Deep Investigator (93% fit, slight Rapid Builder influence in tool-use patterns)

## VIII. Related Skills

- `analyze-agent-behavior` -- Uses ingested prompts to map behavioral patterns to ADA fields
- `ingest-system-prompt` -- Parses and stores system prompts into the Dojo memory layer
- `build-intelligence-map` -- Synthesizes multiple tool analyses into comparative maps
- `seed-extraction` -- Extracts reusable patterns (the output of archaeology becomes seeds)
- `disposition-from-prompts` seed -- The original seed that inspired this skill

## Output

- A named disposition preset in valid YAML format (`[Tool]-[Version]-[Archetype]`) ready to import into Dojo ADA
- A layered behavioral analysis document with each structural layer tagged, behavioral signals evidence-cited, cluster match noted, and notable design tradeoffs called out

## Examples

**Scenario 1:** User provides the Claude Code system prompt from Piebald-AI/claude-code-system-prompts → skill decomposes it into 7 layers, classifies pacing as deliberate and initiative as proactive (with quoted evidence for each), produces `Claude-Code-2026Q1-DeepInvestigator` preset at 93% cluster match.

**Scenario 2:** User says "what patterns does the Windsurf prompt use?" → skill acquires the prompt from the Piebald-AI repo, identifies rapid pacing and surface depth as dominant signals, notes absence of error-handling instructions, produces `Windsurf-VelocityBuilder` preset and flags the missing safety layer.

## Edge Cases

- If the prompt is shorter than 200 words, return a note explaining the minimum length requirement and decline analysis — there is insufficient text for meaningful layer decomposition.
- If the prompt is from behavioral inference (not a real text source), mark every classification as low-fidelity and require the user to confirm before saving the preset.

## Anti-Patterns

- Inferring behavioral intent without quoting the specific prompt text that supports it — every classification must cite evidence; "they probably meant X" is not archaeology.
- Force-fitting hybrid prompts to the four cluster archetypes — when a prompt spans clusters, document the hybrid pattern; losing nuance by forcing a fit obscures useful signal.
