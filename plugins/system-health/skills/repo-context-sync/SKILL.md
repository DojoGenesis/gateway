---
name: repo-context-sync
model: sonnet
description: Produces grounding context summaries — directory structure, recent diffs, and file patterns — from local repository state, enabling more accurate architectural and implementation decisions. Use when: "extract context from this repo", "what changed since last session", "understand the repo structure before writing a prompt", "ground a refactor in actual code state".
category: system-health

inputs:
  - name: repo_path
    type: string
    description: Path to the repository to extract context from
    required: true
outputs:
  - name: context_summary
    type: string
    description: Grounding context summaries with directory structure, recent diffs, and file patterns from local repository state
---

# Repo Context Sync

Efficiently sync with relevant parts of local repositories to ground reasoning in actual codebase state. This skill enables context-aware architectural decisions, refactoring, and implementation prompt writing by providing tools to clone, track changes, and extract patterns from repos.

## I. The Philosophy: Grounding in Reality

Architectural decisions made in a vacuum are fragile. Implementation prompts written without understanding existing patterns create friction. The Repo Context Sync skill embodies grounding in reality — the practice of syncing with the actual codebase state before making decisions or writing specifications.

This is not about reading every file, but about surgical context extraction: identifying what matters for the task at hand, syncing only those parts, and integrating that context into reasoning. It transforms vague architectural discussions into grounded, actionable decisions.

## II. When to Use This Skill

Trigger this skill when:
- User mentions "repo", "repository", "codebase", "sync", "what changed"
- Starting a conversation about refactoring, architecture, or design
- Writing implementation prompts that need to follow existing patterns
- User asks about specific repo directories (e.g., `/00_Roadmap/`, `/02_Specs/`)
- Need to understand current state of a local codebase before a sprint

## III. Core Workflow

### 1. Identify Context Need

Parse the user's request to determine:
- Which repo(s) are relevant
- Which directories matter for the task
- What keywords indicate focus areas

**Common patterns:**
- "refactor the agent routing" → Need `/02_Specs/`, `/04_System/`
- "write implementation prompt for X" → Need existing code patterns
- "what changed in the backend" → Need diff since last sync

### 2. Sync Repo State

Use the appropriate script based on repo state:

#### If repo not yet cloned:
```bash
bash {SKILLS_ROOT}/repo-context-sync/scripts/smart_clone.sh \
  <repo_url> \
  <local_path> \
  [dir1] [dir2] ...
```

**Example:**
```bash
bash {SKILLS_ROOT}/repo-context-sync/scripts/smart_clone.sh \
  https://github.com/TresPies-source/dojo-genesis \
  <your-local-repos-path>/dojo-genesis \
  /00_Roadmap/ /02_Specs/ /04_System/
```

This uses sparse checkout to only pull specified directories, saving time and space.

#### If repo already cloned:
The script automatically detects existing repos and runs `git fetch && git pull` instead.

### 3. Track Changes

Generate a diff summary to see what changed since last sync:

```bash
python3 {SKILLS_ROOT}/repo-context-sync/scripts/diff_tracker.py \
  <repo_path> \
  [last_commit_hash]
```

**Example:**
```bash
python3 {SKILLS_ROOT}/repo-context-sync/scripts/diff_tracker.py \
  <your-local-repos-path>/dojo-genesis \
  abc123
```

This generates a markdown summary showing:
- Added, modified, deleted files
- Commit messages
- Summary statistics

The summary is both printed and saved to `.diff_summary.md` in the repo.

### 4. Generate Context Map

Create a comprehensive overview of the codebase:

```bash
python3 {SKILLS_ROOT}/repo-context-sync/scripts/context_mapper.py \
  <repo_path> \
  [focus_keywords...]
```

**Example:**
```bash
python3 {SKILLS_ROOT}/repo-context-sync/scripts/context_mapper.py \
  <your-local-repos-path>/dojo-genesis \
  agent routing supervisor
```

This generates:
- File tree structure
- Detected languages and frameworks
- Files matching focus keywords
- Summaries of relevant files
- Pattern extraction

The summary is saved to `.context_summary.md` in the repo.

### 5. Integrate Context

Read the generated summaries and integrate into reasoning:

1. **Read the context summary** to understand repo structure
2. **Read relevant files** identified by the mapper
3. **Check reference docs** for patterns:
   - `{SKILLS_ROOT}/repo-context-sync/references/file_hierarchy_patterns.md`
   - `{SKILLS_ROOT}/repo-context-sync/references/zenflow_repo_patterns.md`
4. **Ground architectural decisions** in actual code state

## IV. Script Reference

### smart_clone.sh

**Purpose:** Efficient sparse checkout of relevant directories

**Usage:**
```bash
bash smart_clone.sh <repo_url> <local_path> [dir1] [dir2] ...
```

**Behavior:**
- If repo doesn't exist: Clone with sparse checkout
- If repo exists: Fetch and pull latest changes
- Only downloads specified directories (if provided)

**Storage location:** A local directory of your choosing, e.g. `<your-local-repos-path>/{repo_name}/`

### diff_tracker.py

**Purpose:** Track and summarize changes since last sync

**Usage:**
```bash
python3 diff_tracker.py <repo_path> [last_commit_hash]
```

**Output:**
- Markdown summary of changes
- List of modified/added/deleted files
- Commit messages
- Saved to `<repo_path>/.diff_summary.md`

**Default behavior:** If no commit hash provided, compares last 10 commits

### context_mapper.py

**Purpose:** Generate codebase overview for task context

**Usage:**
```bash
python3 context_mapper.py <repo_path> [focus_keywords...]
```

**Output:**
- File tree structure (max depth 3)
- Detected languages and frameworks
- Files matching keywords (via git grep)
- Summaries of top 5 relevant files
- Saved to `<repo_path>/.context_summary.md`

**Pattern detection:**
- Languages by file extensions
- Frameworks by config files (package.json, requirements.txt, etc.)
- File hierarchy patterns (numbered directories like `/00_Roadmap/`)

## V. Reference Documents

### file_hierarchy_patterns.md

Describes the standard "Planning with Files" hierarchy used in Dojo Genesis and related repos:

- `/00_Roadmap/` - High-level goals and task_plan.md
- `/01_PRDs/` - Product Requirement Documents
- `/02_Specs/` - Technical specifications
- `/03_Prompts/` - Local prompt library
- `/04_System/` - AI Personas and system prompts
- `/05_Logs/` - Dev traces, JOURNAL.md, AUDIT_LOG.md

Read this when:
- Understanding repo structure
- Determining which directories to sync
- Writing files that follow conventions

### implementation_agent_patterns.md

Describes how to work with implementation agents' workflows:

- Task artifact structure (e.g., `.zenflow/tasks/{task_id}/` for Zenflow)
- Built-in workflow types (Quick Change, Fix Bug, Spec and Build, Full SDD)
- Best practices for reading existing codebases
- Integration with standard file hierarchy

Read this when:
- Writing implementation prompts
- Understanding task artifacts
- Following existing codebase patterns

## VI. Common Use Cases

### Use Case 1: Starting a Refactoring Conversation

**User says:** "Let's refactor the agent routing logic"

**Workflow:**
1. Identify relevant repo: dojo-genesis
2. Identify relevant dirs: `/02_Specs/`, `/04_System/`
3. Run smart_clone.sh with those dirs
4. Run context_mapper.py with keywords: "agent", "routing", "supervisor"
5. Read generated context summary
6. Read relevant spec files
7. Ground refactoring suggestions in actual code

### Use Case 2: Writing an Implementation Prompt

**User says:** "Write an implementation prompt to implement the Trail of Thought visualization"

**Workflow:**
1. Identify relevant repo: dojo-genesis
2. Run smart_clone.sh to get latest code
3. Run context_mapper.py with keywords: "visualization", "trace", "component"
4. Read implementation_agent_patterns.md for best practices
5. Extract existing patterns (naming, imports, structure)
6. Write prompt that says "Follow existing patterns in /src/components/"
7. Reference specific files and conventions

### Use Case 3: Tracking Evolution

**User says:** "What changed in the backend since last session?"

**Workflow:**
1. Identify relevant repo: dojo-genesis
2. Load last sync commit hash from state (if available)
3. Run diff_tracker.py with that commit
4. Read generated diff summary
5. Highlight architectural implications
6. Summarize for user

## VII. State Management

Track sync state in a `.repo-sync-state.json` file at a stable location in your workspace (e.g., your project root or a dedicated config directory):

```json
{
  "repos": {
    "TresPies-source/dojo-genesis": {
      "last_sync": "2026-01-28T17:48:00Z",
      "commit_hash": "abc123",
      "local_path": "<your-local-repos-path>/dojo-genesis",
      "tracked_dirs": ["/00_Roadmap/", "/02_Specs/"]
    }
  }
}
```

Update this file after each sync to enable diff tracking. Store it at your project root or another stable path — not at a platform-specific location.

## VIII. Integration with Implementation Agents

When working with implementation agents (Zenflow, Claude Code, etc.):

1. **Read existing codebase first** - Use context_mapper.py before generating code
2. **Follow existing patterns** - Extract conventions from context summary
3. **Reference standard hierarchy** - Use file_hierarchy_patterns.md as guide
4. **Integrate cleanly** - Minimize refactoring, extend existing components

Implementation agents typically have full repo access, so prompts should leverage that:
- "Read /src/components/ to understand component patterns"
- "Follow the same naming conventions as existing files"
- "Integrate with existing state management in /src/store/"

## IX. Best Practices

### Efficiency
- Only sync directories relevant to the task
- Use sparse checkout for large repos
- Cache context summaries (invalidate on sync)

### Accuracy
- Always fetch latest changes before generating context
- Verify commit hashes in diff tracking
- Cross-reference multiple files for patterns

### Transparency
- Show user what's being synced and why
- Surface generated summaries for review
- Log sync operations to session

### Integration
- Read reference docs to understand patterns
- Follow existing conventions in repos
- Update state file after each sync

## X. Limitations

### What This Skill Does
- Efficiently clone/sync relevant repo parts locally
- Track changes and generate diffs
- Extract patterns and conventions
- Provide grounded context for reasoning

### What This Skill Doesn't Do
- Automatically modify code
- Make architectural decisions
- Execute implementation tasks
- Replace manual code review
- Store full repo history (only relevant parts)
- Push or sync state to remote services — all operations are local

## XI. Troubleshooting

### Issue: Sparse checkout not working
**Solution:** Ensure Git version is 2.25+ (sparse-checkout v2 feature)

### Issue: Context mapper finds too many files
**Solution:** Use more specific keywords or limit to specific directories

### Issue: Diff tracker shows no changes
**Solution:** Verify last_commit_hash is correct and repo has been fetched

### Issue: Scripts fail with permission errors
**Solution:** Ensure scripts are executable (`chmod +x`)

## Output
- `.diff_summary.md` in the repo root: markdown summary of changes since the last sync (added, modified, deleted files; commit messages; statistics).
- `.context_summary.md` in the repo root: file tree, detected stack, keyword-matched files, and summaries of top relevant files.
- Updated `.repo-sync-state.json` at your workspace root recording the commit hash and sync timestamp.

## Examples
**Scenario 1:** "Understand the Gateway repo structure before writing a spec" → sparse checkout of `/02_Specs/` and `/04_System/`, context_mapper run with keywords "gateway routing middleware", `.context_summary.md` produced and read to ground the spec.
**Scenario 2:** "What changed in the backend since last session?" → diff_tracker run against the stored commit hash, `.diff_summary.md` produced showing 3 modified files in the routing layer and 1 new migration, architectural implications surfaced.

## Edge Cases
- If no commit hash is stored in `.repo-sync-state.json`, diff_tracker defaults to comparing the last 10 commits — note this default explicitly in the output so the user knows the baseline.
- If sparse checkout pulls more files than expected (e.g., a directory contains deeply nested subdirectories), the context map may be large — apply keyword filtering to keep output focused.

## Anti-Patterns
- Running context_mapper without first fetching latest changes — generates context from a stale local clone, which can produce misleading architectural suggestions.
- Using this skill as a replacement for reading the actual code — context summaries are grounding artifacts, not substitutes for reading critical files directly.
