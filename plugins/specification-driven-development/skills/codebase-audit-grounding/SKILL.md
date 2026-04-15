---
name: codebase-audit-grounding
model: sonnet
description: "Produces a quantified current-state report (test count, aria markers, error boundaries, storage usage, dependencies) used to anchor specifications in measured reality rather than assumptions. Use when: 'audit the codebase before speccing', 'ground this spec in reality', 'what does the code actually look like', 'measure before I write the spec'."
category: specification-driven-development
triggers:
  - "audit the codebase before speccing"
  - "ground this spec in reality"
  - "what does the code actually look like"
  - "measure before I write the spec"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: repo_path
    type: string
    description: Path to the codebase to audit before specification
    required: true
outputs:
  - name: audit_report
    type: string
    description: Quantified current-state report with test count, aria markers, error boundaries, storage usage, and dependencies
---
# Codebase Audit Grounding

## Philosophy

Specifications should describe deltas from measured reality, not deltas from ideals. A spec grounded in quantified codebase state ("207 aria instances, 24 test files, 0 localStorage usage") produces more precise implementation guidance than a spec describing an imagined end state. This prevents the gap between assumption and reality that causes rework.

The pattern emerged from direct evidence: v0.2.4 specs that included measured codebase audits required zero Track 0 remediation, while v0.2.2 specs written without grounding required significant correction. Measured state → better spec → better implementation.

## When to Use

Activate this skill when:

- Writing any specification that describes changes to an existing codebase
- Before handing off work to an autonomous implementation agent
- Grounding a strategic scout or analysis in actual codebase reality
- Competing interpretations exist about "current" state
- A spec will guide architectural changes or dependency decisions
- Prevention of Track 0 remediation is a priority

## Workflow

### 1. Identify What to Measure

Determine which metrics matter for your project type:

- **Test coverage**: Test file count, test patterns, coverage gaps
- **Accessibility**: aria-* instances, semantic HTML patterns
- **Error handling**: ErrorBoundary components, try-catch blocks, error patterns
- **State management**: localStorage usage, context providers, state patterns
- **Dependencies**: Specific library usage counts, version patterns, plugin ecosystems
- **Architecture**: Component hierarchy, file structure, naming conventions

### 2. Run Quantified Audits

Use grep, glob, and language-specific tools to measure actual patterns. Record exact counts and file lists.

**Test count:**
```bash
find . -name "*.test.*" -o -name "*.spec.*" | wc -l
```

**Accessibility markers (React example):**
```bash
grep -r "aria-" --include="*.tsx" --include="*.jsx" | wc -l
```

**Error boundary usage:**
```bash
grep -r "ErrorBoundary" --include="*.tsx" | wc -l
```

**Storage usage:**
```bash
grep -r "localStorage" --include="*.ts" --include="*.tsx" | wc -l
```

**Specific library usage:**
```bash
grep -r "framer-motion\|react-spring" --include="*.tsx" | wc -l
```

**Dependency audit:**
```bash
cat package.json | jq '.dependencies | keys | length'
```

**File structure snapshot:**
```bash
find src -type f -name "*.tsx" | head -20
```

### 3. Record Results in "Current State" Section

Create a section in your spec with exact numbers and file samples:

```markdown
## Current State (Measured)

- Test files: 24
- Accessibility markers (aria-*): 207 instances across 18 files
- Error boundaries: 1 (at App root)
- localStorage usage: 0
- Framer Motion: 8 files
- Dependencies: 42 direct
- Key patterns: Redux for state, React Router for navigation
```

Include specific file examples where relevant.

### 4. Write Spec as Delta from Measured State

Frame all changes relative to measured baseline:

```markdown
## Proposed Changes

From 207 aria instances → add 12 new markers in Modal and Form components
From 1 ErrorBoundary → add boundaries around Card components
From 0 localStorage → introduce localStorage for preferences in 1 module
```

Avoid describing an ideal end state. Describe the delta.

### 5. Include Audit Files and Commands

Document how the measurements were taken:

- List specific grep/find commands used
- Record the execution context (repo root, branch, date)
- Include the measured output as reference

## Best Practices

- **Be specific**: "207 aria instances in 18 files" beats "many accessibility markers"
- **Name files**: "aria instances in Modal.tsx, Form.tsx, Button.tsx" not just a count
- **Measure before you speculate**: Run audits before writing any spec text
- **Preserve audit evidence**: Keep the grep output and commands in the spec for reproducibility
- **Update audits as you go**: If spec work takes days, re-run audits before finalizing
- **Distinguish assumptions from measurements**: Mark assumptions as such; keep audit results factual
- **Tailor metrics to your goals**: Different projects need different audits — choose what matters for implementation decisions

## Quality Checklist

- [ ] Identified 3-5 metrics relevant to this spec's scope
- [ ] Ran quantified audits using grep/glob/find for each metric
- [ ] Recorded exact counts and sample files
- [ ] Created "Current State" section in spec with measured data
- [ ] Framed all proposed changes as deltas from measured baseline
- [ ] Included specific grep commands and execution context
- [ ] Avoided describing ideals; stayed grounded in deltas
- [ ] Re-validated audits if spec work spans multiple days

## Common Pitfalls

- **Skipping the audit**: Assuming you know the current state. Always measure.
- **Measuring wrong metrics**: Auditing line count when component count matters. Choose metrics tied to your spec's goals.
- **High-level counts only**: Provide file names and examples, not just totals.
- **Not recording audit method**: Others can't validate or reproduce your measurements.
- **Measuring stale code**: Run audits from main/current branch in fresh checkout.
- **Forgetting to compare**: A good spec shows the delta, not just the current state. "From X → to Y" is stronger than "We have X."

## Related Skills

- **pre-commission-alignment**: Fixes gaps that audit prevents. Audit is proactive; Track 0 is reactive.
- **strategic-scout**: Can be grounded in codebase audit results for more precise analysis.
- **specification-writer**: Works best paired with audit grounding to prevent ungrounded specs.
- **implementation-prompt**: Receives higher-quality specs when grounding is done upstream.
- **gap-audit-then-fix** (specification-driven-development): This skill covers the measurement phase of a gap audit — producing quantified current state. For the full pipeline including gap classification by effort tier, parallel fix dispatch with file manifests, and a GAPS.md registry, see `gap-audit-then-fix` in specification-driven-development.

## Output

- A "Current State (Measured)" section ready to paste into a specification document, containing exact counts and file samples for each chosen metric
- The grep/find commands used, recorded in the spec for reproducibility
- A delta framing ("From X → to Y") for each measured dimension, ready to anchor proposed changes

## Examples

**Scenario 1:** "Audit the DojoChat codebase before writing the v0.4 accessibility spec." → Current State section listing 207 aria instances across 18 files, 1 ErrorBoundary at App root, 0 localStorage usage, 8 Framer Motion files — plus the exact grep commands that produced those counts.

**Scenario 2:** "Ground the new test strategy spec in what tests already exist." → A report showing 24 test files, the test framework (Vitest), current coverage gaps by directory, and a delta framing showing which modules have zero coverage.

## Edge Cases

- When the codebase is brand new with no existing files, skip the audit and note that all metrics start at zero — write the spec as greenfield.
- When running audits on a monorepo, scope each grep command to the specific package directory; repo-wide counts mislead when only one package is being specced.
- When the spec work spans multiple days, re-run all audits on the day of finalization — codebase changes invalidate counts taken earlier.

## Anti-Patterns

- Recording only totals without file names — a count of "207" without knowing which files contain the markers gives the implementing agent nothing to act on.
- Running audits from a feature branch instead of main — branch-specific state produces measurements that won't match what the agent sees when it checks out.
- Describing an ideal end state instead of a measured delta — "we should have full test coverage" is a wish, not a spec; "from 24 test files → add 8 more covering the Modal and Form modules" is a spec.
