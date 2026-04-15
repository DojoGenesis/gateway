---
name: batch-normalize-and-package
model: sonnet
description: "Produces a signed distribution manifest and CAS-packaged skill set by orchestrating the full community-to-Dojo pipeline: shallow-clone repos, scan for compatibility, normalize frontmatter in parallel batches of 10, package via dojo CLI, and emit manifest.json with source attribution for every entry. Use when: \"import community skills\", \"batch normalize skills\", \"package skills from repos\", \"onboard external skills\", \"run the skill supply chain\"."
license: Complete terms in LICENSE.txt
category: skill-forge
triggers:
  - "import community skills"
  - "batch normalize skills"
  - "package skills from repos"
  - "onboard external skills"
  - "run the skill supply chain"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: repo_urls
    type: string[]
    description: GitHub repository URLs or local paths containing community skills to import
    required: true
outputs:
  - name: distribution_manifest
    type: ref
    format: cas-ref
    description: Signed distribution manifest and CAS-packaged skill set with source attribution for every entry
---
# Batch Normalize and Package

## I. Workflow

This is a 6-step DAG with dependency structure:

```
Step 1: Acquire sources (parallel per repo)
    |
Step 2: Scan all sources (invokes scan-community-repos)
    |
Step 3: Filter catalog (select normalizable + ready skills)
    |
Step 4: Normalize (parallel per skill, invokes normalize-community-skill)
    |
Step 5: Package (invokes dojo skill package-all on normalized directory)
    |
Step 6: Generate distribution manifest
```

### Step 1: Acquire Sources

For each input repo:
1. If GitHub URL: shallow clone (`git clone --depth=1`) to temp directory
2. If local path: validate exists, use directly
3. If already cloned (cache hit): `git pull --ff-only` to update
4. Record: repo name, local path, acquisition timestamp

**Parallelization:** All repos can be acquired simultaneously. No dependencies.

### Step 2: Scan All Sources

Invoke `scan-community-repos` with the list of acquired paths.

**Input:** list of `{repo_name, local_path}` pairs
**Output:** JSON catalog with per-file classification (ready / normalizable / incompatible)

### Step 3: Filter Catalog

From the scan catalog, build the work queue:

| Classification | Action |
|---------------|--------|
| ready | Copy to staging directory as-is |
| normalizable | Add to normalization queue |
| incompatible | Log and skip — include in manifest as "skipped" |

**Decision criteria for "normalizable":**
- Has `name` field in frontmatter (required — cannot be inferred)
- Has `description` field (required — cannot be inferred)
- Has markdown body with at least one section heading
- File size > 100 bytes and < 50KB (too small = stub, too large = not a skill)

### Step 4: Normalize

For each skill in the normalization queue, invoke `normalize-community-skill`:

**Input:** path to the community SKILL.md
**Output:** enriched SKILL.md with Dojo-compatible frontmatter

**Parallelization:** All normalizations are independent. Run in parallel batches of 10.

**Error handling:**
- If normalization fails (malformed YAML, encoding issues): log error, mark as "failed" in manifest, continue
- If normalized skill still fails IsValid() simulation: log warning, mark as "needs-manual-review"

### Step 5: Package

Run `dojo skill package-all <staging-directory>` on the combined staging directory containing:
- Ready skills (copied as-is from Step 3)
- Normalized skills (output from Step 4)

This produces CAS entries: config blob (SkillManifest JSON) + content blob (tar archive) for each skill.

**Validation:** After packaging, verify each skill has both CAS tags:
- `skill/{name}:config` @ `{version}`
- `skill/{name}:content` @ `{version}`

### Step 6: Generate Distribution Manifest

Produce a manifest document with:

```markdown
# Skill Distribution Manifest
Generated: {timestamp}
Source repos: {count}

## Summary
- Scanned: {total_files}
- Ready (no normalization needed): {count}
- Normalized successfully: {count}
- Failed normalization: {count}
- Incompatible (skipped): {count}
- Packaged into CAS: {count}

## Packaged Skills
| Name | Source Repo | Tier | CAS Config Hash | CAS Content Hash |
|------|-----------|------|-----------------|------------------|
| ... | ... | ... | sha256:... | sha256:... |

## Skipped / Failed
| Name | Source Repo | Reason |
|------|-----------|--------|
| ... | ... | "missing name field" / "normalization error: ..." |
```

Also generate machine-readable `manifest.json`:
```json
{
  "generated": "2026-04-05T...",
  "source_repos": [...],
  "skills": [
    {
      "name": "skill-name",
      "source_repo": "org/repo",
      "tier": 2,
      "cas_config_hash": "sha256:...",
      "cas_content_hash": "sha256:...",
      "status": "packaged"
    }
  ]
}
```

## II. Best Practices

1. **Run scan before normalize.** Never normalize blindly — the scan catalog prevents wasted effort on incompatible files.

2. **Preserve source attribution.** Every packaged skill must retain its source repo in the manifest. Provenance is non-negotiable for trust.

3. **Fail gracefully, report completely.** A single malformed skill should never abort the entire pipeline. Log it, skip it, include it in the manifest as failed, continue.

4. **Idempotent runs.** Running the pipeline twice on the same repos should produce the same CAS hashes (content-addressed storage guarantees this). Use this property for verification.

5. **Respect rate limits.** When cloning from GitHub, introduce a small delay between clones to avoid API rate limiting. Use shallow clones to minimize bandwidth.

6. **Stage before packaging.** Always copy/normalize into a clean staging directory. Never modify source repos in-place.

## III. Quality Checklist

- [ ] All source repos acquired successfully (or failures logged)
- [ ] Scan catalog generated with correct classifications
- [ ] No "normalizable" skills skipped without attempting normalization
- [ ] All normalized skills pass IsValid() simulation
- [ ] `dojo skill package-all` completes without errors
- [ ] Every packaged skill has both CAS tags (config + content)
- [ ] Distribution manifest includes ALL skills (packaged, failed, and skipped)
- [ ] manifest.json is valid JSON and parseable
- [ ] Source attribution preserved for every entry
- [ ] Pipeline is idempotent (re-run produces same hashes)

## Output

- Normalized skill files in a clean staging directory (source repos untouched)
- CAS entries for each packaged skill: `skill/{name}:config` and `skill/{name}:content` tags at the resolved version hash
- `skill-distribution-manifest.md` — human-readable summary of scanned, normalized, failed, and skipped counts
- `manifest.json` — machine-readable manifest with CAS hashes and source attribution for every entry
- Pipeline run logged with timestamps for each step

## Examples

**Scenario 1:** "Import community skills from alirezarezvani/claude-skills and slavingia/skills" → Shallow-clone both repos in parallel, invoke `scan-community-repos` on the combined paths, filter catalog, normalize all normalizable skills in parallel batches of 10, run `dojo skill package-all` on staging, generate manifest.

**Scenario 2:** "Run the weekly supply chain refresh on our watched repos" → Pull latest from cached clones (`git pull --ff-only`), re-scan, normalize only skills not already in CAS (idempotent check), package new additions, append to existing manifest.

## Edge Cases

- A single repo clone fails (network error, private repo) — log the failure, skip that source, continue with the rest; name the failed repo explicitly in the manifest's skipped section
- A skill fails normalization (malformed YAML, encoding issues) — mark as "failed" in the manifest with the error message, continue; do not abort the pipeline
- A normalized skill still fails IsValid() after normalization — mark as "needs-manual-review" in the manifest; do not attempt to package it
- Re-run on the same repos — content-addressed storage guarantees the same input produces the same hash; use this as a verification check, not a guard against re-running

## Anti-Patterns

- **Modifying source repos in place:** All normalization must happen in the staging directory — never edit the cloned source repos directly
- **Aborting on a single failure:** One malformed skill file should never stop the pipeline; log, skip, and continue
- **Omitting source attribution:** Every entry in the manifest must name its source repo — provenance is required for trust and for rollback decisions
- **Running without a prior scan:** Normalizing all files blindly wastes compute on incompatible files; always let `scan-community-repos` filter the work queue first
