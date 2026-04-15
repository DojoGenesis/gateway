---
name: scan-community-repos
model: sonnet
description: "Produces a compatibility catalog — skill-scan-report.md and skill-scan-catalog.json — classifying every SKILL.md in one or more GitHub repos or local paths as ready, normalizable, or incompatible, so you know exactly what normalization work lies ahead before committing to an import. Use when: \"scan this repo for skills\", \"check which skills are dojo-compatible\", \"audit community skills\", \"catalog skills from a github repo\", \"find skill files in this repository\", \"assess community skill compatibility\", \"build a skill import report\"."
license: Complete terms in LICENSE.txt
category: skill-forge
triggers:
  - "scan this repo for skills"
  - "check which skills are dojo-compatible"
  - "audit community skills"
  - "catalog skills from a github repo"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: repo_paths
    type: string[]
    description: GitHub repo URLs or local paths to scan for SKILL.md files
    required: true
outputs:
  - name: scan_report
    type: ref
    format: cas-ref
    description: Compatibility catalog — skill-scan-report.md and skill-scan-catalog.json — classifying each SKILL.md as ready, normalizable, or incompatible
---
# Scan Community Repos

## I. Workflow

### Step 1: Accept and Validate Inputs

Accept input as:

- One or more GitHub repository URLs: `https://github.com/owner/repo`
- One or more local directory paths: `/path/to/plugin-directory`
- A mixed list of both

For each GitHub URL, confirm the repository is accessible. If the URL requires authentication or returns 404, log it as inaccessible and skip — do not abort the full scan.

For each local path, confirm the directory exists. If it does not, log and skip.

**Output of this step:** A validated input list with accessibility status for each source.

### Step 2: Acquire Skill Files from Each Source

**For GitHub URLs:**

Clone each repository to a temporary directory:

```bash
git clone --depth=1 <repo-url> /tmp/skill-scan/<repo-name>
```

Use `--depth=1` to avoid fetching full history. If `git clone` fails (network issue, private repo), log the error and skip the source.

**For local paths:**

Walk the directory tree in place. No copy needed.

**Finding skill files:**

Within each acquired source, find candidate skill files using two passes:

1. **Exact match:** files named exactly `SKILL.md` (case-sensitive)
2. **Pattern match:** any `.md` file that contains a YAML frontmatter block (`---`) and whose path contains `skill` in the directory name or filename

For pattern-match candidates, apply a quick filter: the frontmatter must contain at least one of `name:` or `description:` to be considered a skill candidate. Discard files that pass the filename heuristic but have no frontmatter at all (these are likely documentation, not skills).

**Output of this step:** A flat list of candidate file paths per source repo.

### Step 3: Parse Each Candidate File

For each candidate file, parse:

1. **Frontmatter extraction:** Read lines between the first and second `---` delimiters. Parse as YAML.
2. **Body extraction:** All content after the second `---`.
3. **Handle parse errors:** If the YAML is malformed, classify the file as "incompatible" with reason "malformed frontmatter" and continue.

Extract these fields from the parsed frontmatter:

| Field | Check |
|---|---|
| `name` | Present and non-empty string |
| `description` | Present and non-empty string |
| Trigger phrases | Inline in description (`Trigger phrases?: (.+)`) or explicit `triggers:` key |
| `tier` | Present and integer 1–4 |
| `agents` | Present and non-empty array |
| `tool_dependencies` | Present (empty array is valid) |

Extract these structural signals from the body:

| Signal | Check |
|---|---|
| Philosophy section | Contains `## I.` or `## Philosophy` or `# Philosophy` |
| Workflow section | Contains `## III.` or `## Workflow` or `# Workflow` |
| Quality Checklist | Contains `## V.` or `Checklist` or `- [ ]` items |

### Step 4: Classify Each Skill

Apply the classification rules in order. Stop at the first matching rule.

**"ready"** — Passes SkillRegistry.IsValid():
- `name` present and non-empty
- `description` present and non-empty
- At least one trigger phrase (inline or extracted)
- `tier` present, integer 1–4
- `agents` non-empty array
- `tool_dependencies` present (can be empty array)

**"normalizable"** — Has the minimum required content but is missing Dojo-specific fields:
- `name` present and non-empty
- `description` present and non-empty
- Missing one or more of: triggers, tier, agents, tool_dependencies

**"incompatible"** — Cannot be normalized without manual repair:
- Missing `name` or `description`
- Frontmatter is malformed YAML
- File contains frontmatter delimiter but no actual YAML fields
- File matched the pattern heuristic but contains no frontmatter at all

### Step 5: Generate the Markdown Report

Write a markdown report to the output path (default: `./skill-scan-report.md`).

Report structure:

```markdown
# Skill Compatibility Scan Report

**Date:** <ISO 8601>
**Sources scanned:** <N>
**Total skills found:** <N>

## Summary

| Classification | Count | Percentage |
|---|---|---|
| ready | N | % |
| normalizable | N | % |
| incompatible | N | % |

## Skills Inventory

| Skill Name | Source Repo | Classification | Missing Fields | Body Structure |
|---|---|---|---|---|
| skill-name | owner/repo | ready | — | Philosophy, Workflow, Checklist |
| another-skill | owner/repo | normalizable | tier, agents, triggers | Philosophy, Workflow |
| broken-skill | owner/repo | incompatible | name, description | unknown |

## Recommendations

### Ready for Import (N skills)
These skills can be registered immediately with no changes required.
<list of skill names>

### Batch Normalization Candidates (N skills)
Run `batch-normalize-and-package` on these skills. Each requires frontmatter enrichment only — the knowledge content is intact.
<list of skill names with missing field counts>

### Manual Repair Required (N skills)
These skills cannot be automatically normalized. Review each individually.
<list of skill names with incompatibility reason>

## Source Accessibility Log
<any skipped sources and their error messages>
```

### Step 6: Generate the JSON Catalog

Write a machine-readable catalog to `./skill-scan-catalog.json` alongside the markdown report.

JSON schema:

```json
{
  "scan_date": "<ISO 8601>",
  "sources": [
    {
      "url": "https://github.com/owner/repo",
      "accessible": true,
      "skills_found": 12
    }
  ],
  "skills": [
    {
      "name": "skill-name",
      "source_repo": "owner/repo",
      "source_path": "skills/skill-name/SKILL.md",
      "classification": "ready | normalizable | incompatible",
      "missing_fields": ["tier", "agents"],
      "has_triggers": true,
      "body_has_philosophy": true,
      "body_has_workflow": true,
      "body_has_checklist": false,
      "incompatibility_reason": null
    }
  ],
  "summary": {
    "total": 0,
    "ready": 0,
    "normalizable": 0,
    "incompatible": 0
  }
}
```

The JSON catalog is the primary input for `batch-normalize-and-package`. Keep field names stable.

### Step 7: Deliver Results

Report to the user:

1. Path to the markdown report
2. Path to the JSON catalog
3. One-line summary: "Found N skills across M repos: X ready, Y normalizable, Z incompatible."
4. If any sources were inaccessible, name them explicitly
5. Suggested next action based on counts:
   - If normalizable > 0: "Run `batch-normalize-and-package` on the JSON catalog."
   - If incompatible > 0 and count is small: "Review incompatible skills manually — see the report for reasons."
   - If ready == total: "All skills are Dojo-compatible. Proceed to import."

Clean up temp directories created during the scan (`/tmp/skill-scan/`).

---

## II. Best Practices

**Scan before you normalize, always.** Running `normalize-community-skill` on every file without a prior scan wastes compute on incompatible files that cannot be fixed by normalization anyway.

**Use `--depth=1` for all clones.** Community skill repos can be large. Full history is irrelevant for a compatibility scan; shallow clones keep the operation fast and disk-efficient.

**Preserve the source path in the catalog.** The `source_path` field in the JSON catalog is what allows `batch-normalize-and-package` to locate the file without re-scanning. Never omit it.

**Pattern-match candidates are noisy.** Files that match the filename heuristic but have no frontmatter should be logged as "not a skill" rather than "incompatible." Incompatible implies the author intended a skill but missed required fields. "Not a skill" means the file was a false positive from the heuristic.

**Percentage breakdowns matter.** A repo with 80% normalizable skills is a good import candidate — one batch run cleans it up. A repo with 80% incompatible is a warning sign about the repo's overall quality or schema assumptions.

**Clean up temp directories.** Every shallow clone takes disk space. Always delete `/tmp/skill-scan/` at the end of the workflow, even on failure.

---

## III. Quality Checklist

Before delivering the scan results, confirm:

- [ ] All input sources have been processed (accessible ones scanned, inaccessible ones logged)
- [ ] Every candidate file has been classified as exactly one of: ready, normalizable, incompatible
- [ ] No file appears in more than one classification
- [ ] Markdown report exists at the output path and is readable
- [ ] JSON catalog exists at the output path and is valid JSON
- [ ] JSON catalog `summary` counts match the count of skills in the `skills` array
- [ ] `source_path` is populated for every skill entry in the catalog
- [ ] One-line summary has been delivered to the user
- [ ] Temp directories have been deleted
- [ ] If any sources were inaccessible, the user has been told which ones and why

---

## IV. Related Skills

- `normalize-community-skill` - The per-file normalization skill invoked after this scan identifies "normalizable" candidates
- `batch-normalize-and-package` - Consumes the JSON catalog produced by this skill to run normalization at scale
- `skill-maintenance` - Use after import to rename skills that don't follow verb-object naming conventions
- `health-audit` - Use to audit the Dojo skill ecosystem after a batch import completes

## Output

- `skill-scan-report.md` — human-readable markdown report with summary table, skills inventory, recommendations, and source accessibility log
- `skill-scan-catalog.json` — machine-readable catalog consumed by `batch-normalize-and-package`; includes per-skill classification, missing fields, and body structure signals
- One-line summary delivered to the user: "Found N skills across M repos: X ready, Y normalizable, Z incompatible."
- Temp directories cleaned up (`/tmp/skill-scan/`)

## Examples

**Scenario 1:** "Scan alirezarezvani/claude-skills and tell me which skills we can use" → Clone repo with `--depth=1`, find all SKILL.md files, parse frontmatter and body, classify each as ready/normalizable/incompatible, produce report and catalog, deliver one-line summary with next-step recommendation.

**Scenario 2:** "Before running batch normalization, audit our local plugin directory" → Walk the local path in place (no clone), apply the same two-pass file discovery and classification, write report and catalog to the current directory.

## Edge Cases

- Repository is private or returns 404 — log as inaccessible, skip, continue with other sources; name it explicitly in the accessibility log section of the report
- A file matches the pattern heuristic (contains `skill` in path, has `---` delimiter) but has no actual YAML fields — classify as "not a skill" (false positive), not "incompatible"; incompatible implies the author intended a skill
- Source list is a mix of GitHub URLs and local paths — process each independently and merge results into a single catalog; source type is tracked in the `url` field

## Anti-Patterns

- **Normalizing without scanning first:** Running `normalize-community-skill` on every file before classification wastes compute on files that cannot be salvaged by normalization
- **Treating "incompatible" as "bad knowledge":** The classification is structural, not qualitative — log the reason, preserve the name, and let the operator decide whether to invest in manual repair
- **Omitting `source_path` from the catalog:** Without the source path, `batch-normalize-and-package` cannot locate the file and must re-scan; always populate this field
