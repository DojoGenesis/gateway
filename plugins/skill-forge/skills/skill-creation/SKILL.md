---
name: skill-creation
model: sonnet
description: Produces a complete SKILL.md packaged as a .skill file ready for CAS distribution. Use when: "create a new skill", "write a skill", "build a skill for this workflow", "turn this into a skill", "upgrade this to a skill".
license: Complete terms in LICENSE.txt
category: skill-forge

inputs:
  - name: skill_description
    type: string
    description: Description of the workflow or domain to formalize into a skill
    required: true
  - name: usage_examples
    type: string[]
    description: Concrete usage scenarios for the skill (1-3 examples)
    required: false
outputs:
  - name: skill_package
    type: ref
    format: cas-ref
    description: Complete SKILL.md packaged as a .skill file ready for CAS distribution
---

# Skill Creator

## Purpose

Use this skill to formalize a workflow or domain into a reusable, installable skill package. The output is a `.skill` file ready to be added to any agent that needs the capability.

Invoke when:
- A user asks to "create a skill", "write a skill", "build a skill for this workflow", or "turn this into a skill"
- A workflow is executed repeatedly and would benefit from formalization
- A domain requires knowledge the agent lacks by default (schemas, APIs, business rules)
- An existing skill needs iteration or quality improvement to meet A+ standards

## Inputs

Before starting, gather:

- **Concrete examples** — at least 1–3 real usage scenarios
- **Reusable assets** the skill might need: scripts, templates, reference docs, schemas
- **Target agent** — who will use this skill (same agent, different instance, a partner agent)
- **Skill name** — short, lowercase, hyphenated (e.g., `bigquery-queries`, `pdf-rotation`)

Ask one targeted question at a time. Do not front-load a list of questions.

## Skill Anatomy

Every skill consists of a required SKILL.md file and optional bundled resources:

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter metadata (required)
│   │   ├── name: (required)
│   │   └── description: (required)
│   └── Markdown instructions (required)
└── Bundled Resources (optional)
    ├── scripts/          - Executable code (Python/Bash/etc.)
    ├── references/       - Documentation intended to be loaded into context as needed
    └── templates/        - Files used in output (templates, icons, fonts, etc.)
```

### SKILL.md (required)

Every SKILL.md consists of:

- **Frontmatter** (YAML): Contains `name` and `description` fields. These are the only fields that the agent reads to determine when the skill gets used; make the description clear and specific about what the skill produces and when to use it.
- **Body** (Markdown): Instructions and guidance for using the skill. Only loaded AFTER the skill triggers (if at all).

### Bundled Resources (optional)

- **`scripts/`** - Executable code for repetitive or deterministic tasks (e.g., `rotate_pdf.py`). Token efficient, can run without loading into context.
- **`references/`** - Documentation loaded as needed (schemas, API docs, policies). Keeps SKILL.md lean. For large files (>10k words), include grep patterns in SKILL.md.
- **`templates/`** - Output assets not loaded into context (logos, fonts, boilerplate code).

**Avoid duplication**: Information lives in SKILL.md OR references, not both.

**Do NOT include**: README.md, CHANGELOG.md, or other auxiliary documentation. Skills are for AI agents, not users.

### Progressive Disclosure

Three-level loading system:
1. **Metadata** - Always in context (~100 words)
2. **SKILL.md body** - When skill triggers (<500 lines)
3. **Bundled resources** - As needed

Keep SKILL.md under 500 lines. When splitting content to references, clearly describe when to read them.

**Key principle:** Keep core workflow in SKILL.md; move variant-specific details to reference files.

Example structure for multi-domain skills:

```
bigquery-skill/
├── SKILL.md (overview + navigation)
└── references/
    ├── finance.md
    ├── sales.md
    └── product.md
```

### Specificity Levels

Match the level of specificity to the task's fragility and variability:

**High freedom (text-based instructions)**: Use when multiple approaches are valid, decisions depend on context, or heuristics guide the approach.

**Medium freedom (pseudocode or scripts with parameters)**: Use when a preferred pattern exists, some variation is acceptable, or configuration affects behavior.

**Low freedom (specific scripts, few parameters)**: Use when operations are fragile and error-prone, consistency is critical, or a specific sequence must be followed.

## Skill Creation Process

Skill creation involves these steps:

1. Understand the skill with concrete examples
2. Plan reusable skill contents (scripts, references, templates)
3. Initialize the skill (run init_skill.py)
4. Edit the skill (implement resources and write SKILL.md)
5. Deliver the skill (send SKILL.md path via notify_user)
6. Iterate based on real usage

Follow these steps in order, skipping only if there is a clear reason why they are not applicable.

### Step 1: Understanding the Skill with Concrete Examples

Skip this step only when the skill's usage patterns are already clearly understood.

Gather concrete examples of how the skill will be used. Ask questions like:
- "What functionality should this skill support?"
- "Can you give examples of how it would be used?"

Avoid asking too many questions at once. Conclude when you have a clear sense of the functionality.

### Step 2: Planning the Reusable Skill Contents

For each example, identify reusable resources:

| Resource Type | When to Use                     | Example                               |
| ------------- | ------------------------------- | ------------------------------------- |
| `scripts/`    | Code rewritten repeatedly       | `rotate_pdf.py` for PDF rotation      |
| `templates/`  | Same boilerplate each time      | HTML/React starter for webapp builder |
| `references/` | Documentation needed repeatedly | Database schemas for BigQuery skill   |

### Step 3: Initializing the Skill

At this point, it is time to actually create the skill.

Skip this step only if the skill being developed already exists, and iteration or packaging is needed. In this case, continue to the next step.

When creating a new skill from scratch, always run the `init_skill.py` script. The script conveniently generates a new template skill directory that automatically includes everything a skill requires, making the skill creation process much more efficient and reliable.

Usage:

```bash
python /home/ubuntu/skills/skill-creation/scripts/init_skill.py <skill-name>
```

The script:

- Creates the skill directory at `/home/ubuntu/skills/<skill-name>/`
- Generates a SKILL.md template with proper frontmatter and TODO placeholders
- Creates example resource directories: `scripts/`, `references/`, and `templates/`
- Adds example files in each directory that can be customized or deleted

After initialization, customize or remove the generated SKILL.md and example files as needed.

### Step 4: Edit the Skill

When editing the (newly-generated or existing) skill, remember that the skill is being created for another instance of Manus to use. Include information that would be beneficial and non-obvious to Manus. Consider what procedural knowledge, domain-specific details, or reusable assets would help another Manus instance execute these tasks more effectively.

#### Learn Proven Design Patterns

Consult these helpful guides based on your skill's needs:

- **Multi-step processes**: See `/home/ubuntu/skills/skill-creation/references/workflows.md` for sequential workflows and conditional logic
- **Output formats or quality standards**: See `/home/ubuntu/skills/skill-creation/references/output-patterns.md` for template and example patterns
- **Progressive Disclosure Patterns**: See `/home/ubuntu/skills/skill-creation/references/progressive-disclosure-patterns.md` for splitting content across files.

These files contain established best practices for effective skill design.

#### Start with Reusable Skill Contents

Begin with the `scripts/`, `references/`, and `templates/` files identified in Step 2. This may require user input (e.g., brand assets for `templates/`, documentation for `references/`).

Test added scripts by running them to ensure they work correctly. For many similar scripts, test a representative sample.

Delete any unused example files from initialization.

#### Update SKILL.md

**Writing Guidelines:** Always use imperative/infinitive form.

##### Frontmatter

Write the YAML frontmatter with `name` and `description`:

- `name`: The skill name
- `description`: Primary trigger mechanism. Must include what the skill does AND when to use it (body only loads after triggering).
  - Example: "Document creation and editing with tracked changes. Use for: creating .docx files, modifying content, working with tracked changes."

##### Body

Write instructions for using the skill and its bundled resources.

### Step 5: Delivering the Skill

Once development of the skill is complete, validate and deliver it to the user.

#### Validate the Skill

Run the validation script to ensure the skill meets all requirements:

```bash
python /home/ubuntu/skills/skill-creation/scripts/quick_validate.py <skill-name>
```

If validation fails, fix the errors and run validation again.

#### Deliver to User

Use `message` tool to send the SKILL.md file as attachment:

```
/home/ubuntu/skills/{skill-name}/SKILL.md
```

The system will automatically:

1. Detect the path pattern `/home/ubuntu/skills/*/SKILL.md`
2. Package the skill directory into a `.skill` file
3. Send to frontend as a special card with options:
   - Add to My Skills
   - Download
   - Preview

### Step 6: Iterate

After testing the skill, users may request improvements. Often this happens right after using the skill, with fresh context of how the skill performed.

**Iteration workflow:**

1. Use the skill on real tasks
2. Notice struggles or inefficiencies
3. Identify how SKILL.md or bundled resources should be updated
4. Implement changes and test again

## Output

A completed skill package containing:

- `SKILL.md` — valid YAML frontmatter + body with at least 8 `##` sections (A+ standard)
- `scripts/` — tested, executable code (if applicable)
- `references/` — domain documentation (if applicable)
- `templates/` — output assets (if applicable)
- No auxiliary files (no README.md, CHANGELOG.md, or human-facing docs)

The delivered `.skill` file is ready to install and triggers automatically when the agent encounters matching context.

## Quality Criteria

An A+ skill meets all of these:

- **Frontmatter** — `name` is short and hyphenated; `description` covers both what and when
- **8+ sections** — body structured with `##` headers covering purpose, trigger/inputs, process, output, quality, examples, and anti-patterns
- **Under 500 lines** — body stays lean; variant detail lives in `references/`
- **No redundancy** — information appears in SKILL.md OR references, not both
- **Imperative voice** — all instructions use action verbs
- **Appropriate freedom** — specificity matches task fragility (see Core Principles)
- **Tested scripts** — any bundled scripts run without errors
- **Self-demonstrating** — the skill itself models the format it prescribes

Run `quick_validate.py` before delivery to confirm requirements are met.

## Examples

**Example 1: Formalizing a recurring export workflow**

A user runs the same BigQuery export every week and asks: "Can you turn this into a skill?"

- Step 1: Confirm the exact queries, output format, and destination
- Step 2: Identify `scripts/run_export.py` as a reusable asset; schema docs as `references/schema.md`
- Step 3: Run `init_skill.py bigquery-weekly-export`
- Step 4: Write SKILL.md with trigger phrase "run weekly export", reference the script, note schema location
- Step 5: Validate and deliver

**Example 2: Upgrading a 3-section skill to A+**

An existing skill has only `## Overview`, `## Steps`, `## Notes` — below A+ standard.

- Add the missing sections: Purpose, Inputs, Output, Quality Criteria, Examples, Anti-patterns
- Move verbose prose from `## Steps` to `references/detailed-steps.md` if over 150 lines
- Rewrite `## Notes` as `## Anti-patterns` with concrete failure modes
- Validate and re-deliver

## Edge Cases

- Requested skill is really a seed or note, not a workflow — ask for 2–3 concrete usage scenarios before starting; if the user cannot provide them, suggest capturing a seed first
- Existing skill directory already has a SKILL.md — treat this as iteration (Step 4), not creation from scratch; read the current file before proposing any changes
- User wants a skill that wraps another skill — infer tier 3, add `meta_skill` to tool_dependencies, and link the referenced skill in the body's Related Skills section
- Skill body exceeds 400 lines during drafting — pause, identify which sections contain variant or example detail, and move them to `references/` before continuing

## Anti-Patterns

**Over-explaining what the agent already knows.** Don't include basic API documentation the agent has in training. Only add domain-specific, proprietary, or procedural knowledge the agent cannot infer.

**Vague description field.** A skill with a vague `description` will never fire. The description is the only field read before loading the skill body — be explicit about when to use it.

**Monolithic SKILL.md.** A 600-line body defeats progressive disclosure. When the body exceeds ~400 lines, split variant details into `references/`.

**README.md in the skill directory.** Skills are for agents, not humans. Do not include README.md, CHANGELOG.md, or any auxiliary documentation.

**Skipping validation.** `quick_validate.py` catches structural errors before delivery. Running it only when something feels wrong means shipping broken skills.

**Designing for one example.** A skill built for exactly one known scenario fails on edge cases. Gather multiple examples in Step 1 to expose variation and plan for it.

**High freedom where low freedom is needed.** If a script must run in a specific sequence (e.g., database migration), do not leave it as a narrative description — use a script with enforced order.
