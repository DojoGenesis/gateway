---
name: normalize-community-skill
model: sonnet
description: "Produces an enriched SKILL.md with all six Dojo SkillRegistry.IsValid() fields populated — name, description, tier, agents, tool_dependencies, and trigger phrases — inferred from the existing body without rewriting it. Use when: \"normalize this skill\", \"make this skill dojo-compatible\", \"import a community skill\", \"enrich skill frontmatter\", \"fix skill registry validation\", \"prepare skill for dojo\", \"community skill is missing fields\"."
license: Complete terms in LICENSE.txt
category: skill-forge
triggers:
  - "normalize this skill"
  - "make this skill dojo-compatible"
  - "import a community skill"
  - "enrich skill frontmatter"
  - "fix skill registry validation"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: skill_path
    type: string
    description: Path to the community SKILL.md file to normalize
    required: true
outputs:
  - name: normalized_skill
    type: ref
    format: cas-ref
    description: Enriched SKILL.md with all six Dojo SkillRegistry.IsValid() fields populated
---
# Normalize Community Skill

## I. Workflow

### Step 1: Read and Parse the Input File

Read the SKILL.md file at the provided path. Extract:

1. The raw YAML frontmatter block (between the `---` delimiters)
2. The markdown body (everything after the second `---`)

Handle two common frontmatter layouts found in community skills:

**Flat layout** (most common):
```yaml
name: my-skill
description: Does X when Y.
```

**Nested metadata layout** (some repos use this):
```yaml
metadata:
  name: my-skill
  description: Does X when Y.
  version: 1.0
```

For the nested layout, hoist `name` and `description` to the top level. Discard vendor-specific fields (`version`, `author`, `created_at`) — they are not part of the Dojo schema.

### Step 2: Extract or Generate Trigger Phrases

Check the description field for an explicit trigger block using the pattern:

```
Trigger phrases?: (.+)
```

If found, parse the comma-separated or semicolons-separated list as the trigger array.

If not found, generate triggers from the description's first sentence:

1. Take the first sentence of the description (up to the first period or newline)
2. Extract the core verb-object pair: e.g., "Guides creation of release specifications" → `"create a release specification"`
3. Generate 2–3 variations using common user phrasings:
   - Direct imperative: `"create a release specification"`
   - Question form: `"how do I create a release specification"`
   - Contextual: `"writing a release spec for this feature"`

**Quality gate:** Triggers must be natural language phrases a human would type, not YAML keys or function names.

### Step 3: Infer Tier

Scan the markdown body for evidence of tool usage and skill invocations. Apply this decision table:

| Evidence found in body | Tier |
|---|---|
| No tool or dependency mentions | 1 |
| References `file_system`, `bash`, `web_tools`, or `script_execution` by name or by action (read/write files, run commands, fetch URLs) | 2 |
| Invokes another skill by name (e.g., "use the `skill-creation` skill") or delegates to a sub-agent | 3 |
| Uses 3+ tools, orchestrates multiple agents, or explicitly calls itself a meta-skill | 4 |

When evidence is ambiguous, default to the lower tier. Tier inflation makes the skill harder to route correctly.

### Step 4: Extract tool_dependencies

Scan the body for references to any of the following allowlist items:

- `file_system` — reading/writing files, directory operations
- `bash` — shell commands, scripts, terminal execution
- `web_tools` — HTTP requests, URL fetching, browser actions
- `script_execution` — running Python, Node, or other language scripts (distinct from bash)
- `meta_skill` — invokes other skills, orchestration

Map natural language evidence to allowlist items:

| Natural language signal | Maps to |
|---|---|
| "read the file", "write output to", "list directory" | `file_system` |
| "run the command", "execute in terminal", "shell script" | `bash` |
| "fetch the URL", "HTTP GET", "open the browser" | `web_tools` |
| "run the Python script", "execute the Node script" | `script_execution` |
| "invoke the X skill", "call the Y workflow" | `meta_skill` |

If no matches are found, set `tool_dependencies: []`. Do not invent dependencies.

### Step 5: Default Agents

If the original frontmatter does not specify an `agents` field, set:

```yaml
agents: ["primary"]
```

This is the Dojo Gateway's default routing target. Only override if the body explicitly names a specific agent (e.g., "This skill is used by the `forger` agent").

### Step 6: Validate Against IsValid()

Before writing, verify the enriched frontmatter satisfies all IsValid() conditions:

- [ ] `name` is non-empty string
- [ ] `description` is non-empty string
- [ ] At least one trigger phrase exists (inline in description or as a generated list)
- [ ] `tier` is an integer 1–4
- [ ] `agents` is a non-empty array
- [ ] `tool_dependencies` contains only allowlist values (or is empty array)

If any check fails, diagnose which field is missing and apply the relevant step above again.

### Step 7: Write Enriched Frontmatter

Reconstruct the SKILL.md file with:

1. New frontmatter containing all required fields
2. Original markdown body, **unchanged**

Output format:

```yaml
---
name: <original name>
description: <original description>. Trigger phrases: <comma-separated triggers>.
tier: <inferred integer>
agents: [<agent list>]
tool_dependencies: [<dep list>]
license: Complete terms in LICENSE.txt
---
```

Encode trigger phrases inline in the `description` field using the pattern `Trigger phrases: phrase one, phrase two, phrase three.` — this is how the Dojo SkillRegistry parses them from the description at load time.

Write the result back to the same file path, or to a new path if the user specified an output location.

### Step 8: Report the Normalization

Output a brief normalization summary:

```
Normalized: <skill name>
Source: <input file path>
Changes made:
  - Added tier: <N> (inferred from: <evidence>)
  - Added agents: ["primary"] (default)
  - Added tool_dependencies: [<list>] (inferred from: <evidence>)
  - Added triggers: <list> (extracted from: description / generated from: first sentence)
IsValid(): PASS
```

If IsValid() would still fail after normalization, report the specific failure and request user input.

---

## II. Best Practices

**Preserve the body exactly.** The markdown body is the intellectual content of the skill. Only the frontmatter changes. Do not reformat, reorder, or summarize the body.

**Prefer inference over defaults.** A defaulted `tool_dependencies: []` is valid but weak. Spend one pass scanning the body before accepting an empty list.

**One trigger phrase is the minimum, three is the target.** Single-phrase triggers create brittle routing. Provide phrasings that cover direct, question, and contextual invocation styles.

**Tier 2 is the most common correct answer.** Most community skills describe a workflow that touches files or runs commands. Tier 1 (pure reasoning) and Tier 4 (orchestration) are less common. When unsure between 1 and 2, check whether the workflow steps reference any tool actions.

**Nested metadata is a data loss risk.** When hoisting from a `metadata:` block, log which fields were discarded so the original author can audit the result.

---

## III. Quality Checklist

Before delivering the normalized skill, confirm:

- [ ] Frontmatter has exactly: `name`, `description`, `tier`, `agents`, `tool_dependencies`, `license`
- [ ] No vendor-specific fields remain (`version`, `author`, `created_at`)
- [ ] `description` ends with `Trigger phrases: ...` block
- [ ] Trigger phrases are natural language, not identifiers
- [ ] `tier` matches evidence found in the body (not assumed)
- [ ] `tool_dependencies` values are all from the allowlist
- [ ] Markdown body is byte-for-byte identical to the original
- [ ] IsValid() simulation passes all six checks
- [ ] Normalization summary has been reported to the user

---

## IV. Related Skills

- `scan-community-repos` - Identifies which skills in a repo need normalization before you run this skill
- `batch-normalize-and-package` - Calls this skill in a loop across an entire repo's skill set
- `skill-creation` - Use when a community skill is so incomplete it needs a full rewrite rather than normalization
- `skill-maintenance` - Use after normalization to rename or refactor skills that don't follow verb-object naming

## Output

- Enriched SKILL.md written back to the same path (or a specified output path) with all six required frontmatter fields populated
- Normalization summary reported to the user: which fields were added, what evidence supported each inference, and whether IsValid() passes
- Markdown body is byte-for-byte identical to the original — only frontmatter changes

## Examples

**Scenario 1:** "Import this skill from alirezarezvani/claude-skills — it only has name and description" → Read the file, extract trigger phrases from the description, scan the body for tool usage to infer tier and tool_dependencies, set agents to ["primary"], reconstruct frontmatter, write back, report normalization summary.

**Scenario 2:** "A skill is stuck in 'pending' state in the SkillRegistry" → Read the skill, run IsValid() simulation to identify the failing check, apply only the steps needed to fix the missing fields, write and re-validate.

## Edge Cases

- Skill has a `metadata:` nested block instead of flat frontmatter — hoist `name` and `description` to the top level, discard vendor-specific fields (`version`, `author`, `created_at`), log which fields were dropped so the original author can audit
- Skill body is empty or under 100 bytes — this is a stub, not a normalizable skill; report it as incompatible and request the full file from the original author
- Skill is missing `name` or `description` — this is an "incompatible" classification, not a normalization case; do not attempt to infer these fields, redirect to `skill-creation` if a rewrite is needed

## Anti-Patterns

- **Inventing fields instead of inferring:** Adding `tool_dependencies: ["bash"]` because it seems likely, without evidence in the body — every added field must be traceable to specific language in the skill
- **Rewriting the body:** Reformatting, summarizing, or reordering the markdown body violates the principle that normalization is structural, not editorial — only frontmatter changes
- **Defaulting tier to 2 without checking:** Tier 2 is the most common correct answer but not the automatic answer; scan the body for tool action evidence before assigning
