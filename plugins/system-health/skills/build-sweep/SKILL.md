---
name: build-sweep
model: sonnet
description: "Produces a verified build/test status report and auto-fixed modules by sweeping all Go modules in a workspace. Use when: \"run a sweep\", \"check all builds\", \"nightly sweep\", \"fix all failing modules\", \"workspace health check\"."
category: system-health
triggers:
  - "run a sweep"
  - "check all builds"
  - "nightly sweep"
  - "fix all failing modules"
  - "workspace health check"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: workspace_path
    type: string
    description: Root directory containing Go modules to sweep
    required: true
outputs:
  - name: sweep_report
    type: text
    format: markdown-table
    description: Verified status table of all modules with build/test results, fixes applied, and commit hashes
---
# Build Sweep Skill

## Philosophy

A healthy workspace is one where every module compiles and every test passes. The build sweep is not a debugging session — it is a systematic sweep that treats each module as an independent unit, diagnoses failures mechanically, and applies minimal fixes. The sweep must never trust self-reports: every fix is verified by re-running the exact build and test commands that originally failed.

## When to Use

- As a scheduled nightly task to catch drift across a multi-module workspace
- Before a release to verify all modules are green
- After a large refactor or dependency update that may have broken multiple modules
- When onboarding to a workspace to establish a health baseline

## Workflow

### Phase 1: Detect
1. Find all directories containing `go.mod` under the workspace path (skip `vendor/`, `.git/`, `node_modules/`, `.claude/worktrees/`)
2. For each module, run `go build ./...` and `go test ./...`
3. Collect results into a table: module | build status | test status | error summary

### Phase 2: Diagnose and Fix
For each failing module, dispatch a parallel agent (max 5 concurrent) to:
1. Read the failing build/test output
2. Read the relevant source files to diagnose the root cause
3. Implement the minimal fix — do not refactor, do not add features
4. Re-run that module's build and tests to confirm green

### Phase 3: Verify
After ALL agents complete, independently verify each fix:
1. Run the tests yourself — do NOT trust agent self-reports of completion
2. Check actual file diffs with `git diff`
3. Confirm no duplicate imports, wrong formats, or regressions
4. Run the full suite one final time to catch cross-module integration issues

### Phase 4: Commit
If all fixes pass verification:
1. Stage only the fixed files (not unrelated changes)
2. Commit each module separately: `fix(sweep): [module] — [one-line description]`

### Phase 5: Report
Output a verified summary table: module | failure type | fix applied | verification status | commit hash

## Output

The skill produces:
1. A markdown table summarizing all module statuses
2. Git commits for each fix applied (one per module)
3. A final pass/fail verdict for the entire workspace

## Examples

**Clean workspace:**
```
/sweep
> All 29 modules pass build and test. No action needed.
```

**Two failures detected and fixed:**
```
/sweep
> Phase 1: 27 pass, 2 fail (DIP: missing interface method, MCP: stale test count)
> Phase 2: Dispatched 2 agents
> Phase 3: Both fixes verified independently
> Phase 4: 2 commits created
> RESULT: 29/29 modules green
```

## Edge Cases

- **Module with no test files**: Report as SKIP, not FAIL — `[no test files]` is not an error
- **Flaky tests**: If a test fails then passes on rerun, report as FLAKY with the original error
- **Circular dependency between modules**: Run builds in topological order if dependency info is available
- **Module requires external service**: Skip with a note if `go test` fails due to missing database/service connection
- **Agent fix breaks another module**: Resolve the cross-module conflict before committing either fix

## Anti-Patterns

- **Trusting agent self-reports**: Agents may report "all tests pass" without actually running them. Always verify.
- **Committing untested fixes**: Never commit a fix without running the full build and test suite for that module.
- **Fixing unrelated issues**: The sweep fixes build/test failures only. Do not refactor, add features, or clean up code.
- **Running the full workspace test suite after each individual fix**: Too slow. Verify each module independently, then run one final integration pass.
- **Force-pushing sweep commits**: Sweep commits are always local. Never push without explicit user confirmation.
