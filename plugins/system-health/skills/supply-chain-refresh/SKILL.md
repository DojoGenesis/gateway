---
name: supply-chain-refresh
model: sonnet
description: "Produces a Supply Chain Refresh Report and updated manifest.json by pulling the latest from all watched repos, scanning for new or changed skills, normalizing, and packaging them into CAS. Use when: \"refresh the supply chain\", \"update community skills\", \"weekly skill library maintenance\", \"a community repo announced a new release\"."
license: Complete terms in LICENSE.txt
category: system-health
triggers:
  - "refresh the supply chain"
  - "update community skills"
  - "weekly skill library maintenance"
  - "a community repo announced a new release"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: watched_repos
    type: string[]
    description: List of community repo URLs to pull and scan for new or changed skills
    required: false
outputs:
  - name: refresh_report
    type: ref
    format: cas-ref
    description: Supply Chain Refresh Report and updated manifest.json with newly normalized and CAS-packaged skills
---
# Supply Chain Refresh

Automated skill supply chain maintenance.

## I. Philosophy

A supply chain is only as good as its refresh rate. Community skills evolve — repos add new skills, fix bugs, and improve documentation. This skill runs the full scan-normalize-package pipeline on a scheduled or manual basis, producing a diff report that shows what changed since the last run.

## II. When to Use

- On a weekly schedule to keep the skill library current
- After adding new repos to the watched list
- When a community repo announces a new release
- Before publishing to OCI (Action 7) to ensure latest versions

## III. Workflow

### Step 1: Load Watched Repo List

Read the repo list from the supply chain manifest:
```
/AgenticStackOrchestration/specs/starred-repos-supply-chain/manifest.json
```

Extract `source_repos` array. Also check for any newly starred repos:
```bash
gh api user/starred --jq '.[].full_name'
```

### Step 2: Pull Latest

For each repo:
1. If already cloned: `git pull --ff-only`
2. If new: `git clone --depth=1`
3. Record: commit SHA before and after pull

### Step 3: Scan for Changes

Run `scan-community-repos` on all repo paths.
Compare new catalog against previous manifest.json:
- **New skills:** present in scan, absent in manifest
- **Updated skills:** present in both, content hash differs
- **Removed skills:** present in manifest, absent in scan

### Step 4: Normalize New/Updated Skills

For each new or updated skill:
1. Run `normalize-community-skill`
2. Copy to staging directory

### Step 5: Package

Run `dojo skill package-all` on staging directory.
Only new/updated skills will get new CAS entries (content-addressed = idempotent for unchanged content).

### Step 6: Generate Diff Report

```markdown
# Supply Chain Refresh Report
Date: {timestamp}
Previous run: {last_manifest_date}

## Changes
- New skills: {count} ({names})
- Updated skills: {count} ({names})
- Removed skills: {count} ({names})
- Unchanged: {count}

## New Skill Details
| Name | Source | Tier | CAS Hash |
|------|--------|------|----------|
```

### Step 7: Update Manifest

Overwrite `manifest.json` with new state.

## IV. Best Practices

1. **Always pull before scanning.** Stale clones produce false "unchanged" reports.
2. **Preserve the previous manifest.** Archive it as `manifest-{date}.json` before overwriting.
3. **Rate-limit git operations.** Space clones 1s apart to avoid GitHub rate limits.
4. **Log everything.** The diff report is the audit trail for what entered the skill library.

## V. Quality Checklist

- [ ] All watched repos pulled successfully (or failures logged)
- [ ] Scan catalog compared against previous manifest
- [ ] New/updated skills normalized and packaged
- [ ] Diff report generated with complete change summary
- [ ] Manifest.json updated with new state
- [ ] Previous manifest archived

## Output
- A Supply Chain Refresh Report (markdown) showing new, updated, removed, and unchanged skill counts with a details table for new skills (name, source repo, tier, CAS hash).
- An updated `manifest.json` at `/AgenticStackOrchestration/specs/starred-repos-supply-chain/manifest.json`.
- A archived copy of the previous manifest saved as `manifest-{date}.json` before overwriting.

## Examples
**Scenario 1:** "Weekly skill library maintenance" → 3 repos pulled, scan finds 2 new skills and 1 updated. Both new skills normalized and packaged into CAS. Report shows 495 → 497 community skills. Manifest updated.
**Scenario 2:** "A community repo announced a new release with 5 new skills" → Targeted pull of that repo, scan detects all 5 as new (absent in manifest). All 5 normalized and packaged. Diff report generated listing names, source, and CAS hashes.

## Edge Cases
- If a repo pull fails (rate limit, auth, network), log the failure but continue processing the remaining repos — a partial refresh with a complete failure log is better than a failed run.
- Content-addressed storage (CAS) is idempotent for unchanged content — re-running on the same state produces no new CAS entries, making the operation safe to re-run.

## Anti-Patterns
- Scanning before pulling — stale clones produce false "unchanged" reports that hide new skills from recently updated repos.
- Overwriting `manifest.json` without archiving the previous version — without the archive, diffs cannot be reconstructed and the audit trail is broken.
