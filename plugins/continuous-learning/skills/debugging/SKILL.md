---
name: debugging
model: sonnet
description: Produces a Debug Report documenting root cause, fix applied, and lessons learned through systematic hypothesis testing. Use when: 'debug this error', 'why is this failing', 'investigate the issue', 'find the root cause', 'fix this bug systematically'.
category: continuous-learning

inputs:
  - name: error_description
    type: string
    description: Description of the error or failing behavior to debug
    required: true
  - name: context
    type: string
    description: Additional context — stack traces, logs, reproduction steps
    required: false
outputs:
  - name: debug_report
    type: string
    description: Debug Report documenting root cause, fix applied, and lessons learned
---

# Debugging & Troubleshooting Skills

**Version:** 1.0
**Created:** 2026-02-02
**Author:** Cipher
**Purpose:** Systematic debugging and troubleshooting for code, systems, and workflows

---

## Overview

This skill encodes best practices for **debugging and troubleshooting**—isolating problems, identifying root causes, and implementing fixes methodically. It provides patterns for reading logs, reproducing issues, testing hypotheses, and verifying solutions.

**Philosophy:** Debugging is systematic investigation, not random guessing. One change at a time, test, observe, learn.

---

## When to Use This Skill

- Code behaves unexpectedly (errors, crashes, wrong output)
- Performance degradation (slow queries, long build times)
- Data inconsistency (database state out of sync)
- Integration failure (API returns errors, sync breaks)
- Feature not working as expected (UI bug, logic error)

---

## Core Principles

### 1. Debugging is Systematic, Not Random

**Don't:** Change 5 things at once, guess without evidence, apply fixes blindly, skip testing after fixing.

**Do:** Formulate hypothesis ("X causes Y"), test hypothesis, fix based on confirmed cause, re-test to verify.

### 2. Observe, Don't Assume

**Observe:** Error messages (exact text, stack traces), logs (timestamps, context, sequence), system state, user behavior.

**Don't assume:** "It's probably a caching issue" / "The user must have done X wrong" / "This library is broken" (without evidence).

### 3. Isolate Variables

When you have multiple factors, test them independently:

| Factor | Result |
|--------|--------|
| Database query | FAIL — Timeout (4s) |
| Network request | PASS — Works |
| Frontend render | PASS — Works |

**Conclusion:** Database query is bottleneck. Optimize query or increase timeout.

### 4. Reproduce Consistently

Before debugging: identify exact steps to reproduce, run 3+ times, note if intermittent, identify patterns if so.

**Why:** You can't fix what you can't reproduce.

---

## Debugging Workflow

### Step 1: Gather Information (5-10 minutes)

**Collect:** Error messages (exact text, stack traces, error codes), logs, context (what changed recently?), expected vs. actual behavior, environment (OS, versions, dependencies).

**Template:**

```markdown
## Debug: [Issue Title]

**Error Message:** [Exact error text]
**Stack Trace:** [Copy relevant portion]
**Logs:** [Key log lines with timestamps]
**Context:** What triggered the issue / Recent changes / Environment (Node, DB, OS)
**Expected Behavior:** [What should happen]
**Actual Behavior:** [What actually happened]
```

### Step 2: Formulate Hypothesis (2-5 minutes)

Ask: What could cause this? What's most likely? What evidence supports it?

**Examples:**
- "Database query timeout caused by missing index"
- "404 error caused by wrong file path in env var"
- "Sync fails because file encoding is wrong"

**Prioritize:** Most likely (70%) → Second most likely (30%) → Edge cases (10%)

### Step 3: Test Hypothesis (5-15 minutes)

1. **Verify directly** — Add logging to confirm cause; run minimal reproduction case
2. **Isolate variables** — Remove dependencies, simplify data, change environment
3. **Confirm fix** — Apply targeted fix, re-run reproduction steps, verify resolved

**Document results:**
```
Hypothesis: [Description]
Test Method: [How tested]
Result: Confirmed / Rejected
Evidence: [What proved/disproved hypothesis]
```

### Step 4: Implement Fix

**If confirmed:** Implement minimal fix, add error handling, add logging for future detection.  
**If rejected:** Move to next hypothesis, re-gather information, consider broader systemic causes.

### Step 5: Verify & Document (5-10 minutes)

**Verify:** Original reproduction steps no longer fail, edge cases work, no regressions.

**Document:**
```markdown
## Fix: [Issue Title]

**Root Cause:** [What actually caused issue]
**Solution:** [What was fixed]
**Code Changes:** [File, function, or module changed]
**Testing:**
- Original issue: Resolved
- Edge cases: Tested
- Regression check: No regressions
**Lessons Learned:** [What to do differently in future]
```

---

## Common Debugging Patterns

### Pattern 1: Read Logs Strategically

Don't read entire log files. Search for error codes or keywords, extract lines around error timestamp (±5 min), look for recurring patterns.

```bash
grep -i "error" app.log | tail -20
grep "2026-02-02T14:00" app.log -A 5 -B 5
grep -A 10 "Error:" app.log
```

### Pattern 2: Use Logging to Verify

| Point | Log What | Why |
|--------|-----------|-----|
| Before database query | Query string, parameters | See what's being executed |
| After database query | Rows returned, time taken | Performance check |
| Before API call | Request payload | Verify what's sent |
| After API call | Response status, body | Verify what's received |
| Before file write | File path, content | Verify what's written |

**When in doubt, add a log. You can always remove later.**

### Pattern 3: Binary Search for Root Cause

Narrow down variables systematically. For a slow query: test with 10 rows (rules out table size), then run EXPLAIN (reveals missing index), then add index and confirm fix. One variable at a time.

### Pattern 4: Reproduce in Isolated Environment

When issue only happens in production: create minimal reproduction in dev, simplify data, mock external dependencies, test against same environment versions. Goal: isolate whether issue is data, code, or environment.

### Pattern 5: Use Version Control for Time Travel

```bash
git log --oneline -10
git log --oneline --since="2026-02-02T13:00"
git bisect start
git bisect bad HEAD
git bisect good <last-working-commit>
```

---

## Troubleshooting Categories

### 1. Code Errors (Runtime, Compile, Type)

**Causes:** Null/undefined access, async timing issues, type mismatches, module not found  
**Approach:** Read error carefully, check stack trace for file/line, search error code, isolate the function

### 2. Performance Issues

**Causes:** N+1 queries, slow renders, memory leaks, network latency  
**Approach:** Profile the slow operation, identify bottleneck (database/CPU/I/O/network), optimize the bottleneck, measure before/after

### 3. Data Inconsistency

**Causes:** Database out of sync, cache invalidation, race conditions, migration not applied  
**Approach:** Compare DB state vs. expected, check last sync timestamp, verify integrity, re-run sync or migration

### 4. Integration Failures

**Causes:** API contract mismatch, auth failures, network connectivity, service downtime  
**Approach:** Verify request format, check credentials, test API directly (curl/Postman), check service status

### 5. Environment-Specific Issues

**Causes:** Missing/wrong env vars, path differences (Win/Mac/Linux), dependency version conflicts, file permissions  
**Approach:** Compare working vs. broken env, check env vars, verify dependency versions, check paths and permissions

---

## Quality Checklist

Before considering debugging complete:

- [ ] Root cause identified (not just symptom fixed)
- [ ] Hypothesis tested with evidence
- [ ] Fix is minimal (don't over-engineer)
- [ ] Original issue reproduced before fix
- [ ] Fix resolves issue (not just hides it)
- [ ] Edge cases tested
- [ ] No regressions (other features still work)
- [ ] Root cause, solution, and lessons learned documented

---

## Common Pitfalls to Avoid

- **Shotgun debugging** — Change everything — one change, test, observe
- **Ignoring error messages** — "Probably network" — read exact error, search it
- **Fixing without testing** — Apply change, assume it works — reproduce, fix, re-test
- **Assuming environment** — "Works on my machine" — verify in actual environment
- **Blaming external deps** — "Library is broken" — verify your usage first

---

## Output

- A Debug Report markdown file saved to the project's working directory or `docs/debug/`
- Named: `[date]_[issue-slug]_debug.md`
- Sections: Error Message, Stack Trace, Root Cause, Solution, Code Changes, Test Results, Lessons Learned

---

## Examples

**Scenario 1:** "Debug this error: TypeError: Cannot read property 'id' of undefined" → Debug Report with hypothesis (null user object), test method (added logging), root cause confirmed, fix applied (null check), regression test documented

**Scenario 2:** "Why is this database query taking 8 seconds?" → Debug Report using binary search on query complexity, EXPLAIN output showing missing index, index added, before/after timing documented

---

## Edge Cases

- **Error is intermittent (not reproducible):** Add logging around the suspected area, capture the next occurrence, treat the log as the reproduction case
- **Issue only happens in production:** Create a minimal reproduction in dev using production-equivalent data and environment vars; document the environment delta
- **Multiple simultaneous errors:** Prioritize by severity — fix the error that is blocking other fixes first; do not debug all at once

---

## Anti-Patterns

- Changing multiple things at once before testing — makes it impossible to identify which change resolved the issue
- Searching for the error message online before reading the stack trace — the stack trace almost always locates the cause; search is for when the trace is unhelpful
- Marking a bug as fixed when tests pass but root cause was not confirmed — symptom masking produces the same bug later under different conditions
- Writing a fix that hides the error (empty catch, fallback default) instead of addressing the cause

---

**Related Skills:**
- `workspace-navigation` — When debugging collaborative workspaces
- `repo-context-sync` — When debugging git integration issues
- `web-research` — When searching error codes or similar issues online
- `research-modes` — Deep investigation techniques

---

**Last Updated:** 2026-04-08
**Maintained By:** Cipher
**Status:** Active
