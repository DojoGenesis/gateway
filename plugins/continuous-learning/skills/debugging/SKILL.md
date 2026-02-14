# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: debugging
description: Systematic debugging protocol for analyzing and resolving issues
triggers:
  - "debug issue systematically"
  - "analyze bug root cause"
  - "troubleshoot problem"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - implementation-agent
---

# Debugging & Troubleshooting Skills

**Version:** 1.0
**Created:** 2026-02-02
**Author:** Cipher 🧭
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

**Don't:**
- Change 5 things at once
- Guess the cause without evidence
- Apply fixes blindly
- Skip testing after fixing

**Do:**
- Formulate hypothesis ("X causes Y")
- Test hypothesis (verify it causes problem)
- Fix based on confirmed cause
- Re-test to verify fix works

### 2. Observe, Don't Assume

**Observe:**
- Error messages (exact text, stack traces)
- Logs (timestamps, context, sequence)
- System state (what changed, what's running)
- User behavior (what they did, what they expected)

**Don't assume:**
- "It's probably a caching issue"
- "The user must have done X wrong"
- "This library is broken" (without evidence)

### 3. Isolate Variables

**When you have multiple factors, test them independently:**

| Factors | Test Each | Result |
|----------|-----------|--------|
| Database query | ✅ Works | ❌ Timeout (4s) |
| Network request | ✅ Works | ❌ Works |
| Frontend render | ✅ Works | ❌ Works |

**Conclusion:** Database query is bottleneck (4s timeout). Optimize query or increase timeout.

### 4. Reproduce Consistently

**Before debugging:**
1. Identify exact steps to reproduce
2. Run reproduction 3+ times
3. Note if it happens every time or intermittently
4. If intermittent, identify patterns (time of day, data size, user count)

**Why:** You can't fix what you can't reproduce.

---

## Debugging Workflow

### Step 1: Gather Information (5-10 minutes)

**Collect:**
- **Error messages:** Exact text, stack traces, error codes
- **Logs:** Application logs, database logs, browser console
- **Context:** What happened before error? What changed recently?
- **Expected vs. actual:** What should happen? What actually happened?
- **Environment:** OS, Node version, dependencies, database version

**Template:**

```markdown
## Debug: [Issue Title]

**Gathered:** [Date/Time]

**Error Message:**
[Exact error text]

**Stack Trace:**
[Copy relevant portion]

**Logs:**
[Key log lines with timestamps]

**Context:**
- What triggered the issue:
- Recent changes:
- Environment:
  - Node: [version]
  - Database: [version]
  - OS: [platform]

**Expected Behavior:**
[What should happen]

**Actual Behavior:**
[What actually happened]
```

### Step 2: Formulate Hypothesis (2-5 minutes)

**Ask:**
- What could cause this error?
- What's the most likely cause? (start simple)
- What evidence supports this hypothesis?

**Examples:**
- Hypothesis: "Database query timeout is caused by missing index"
- Hypothesis: "404 error is caused by wrong file path in env var"
- Hypothesis: "Sync fails because file encoding is wrong"

**Prioritize:**
1. Most likely cause (70% confidence)
2. Second most likely (30% confidence)
3. Edge cases (10% confidence)

### Step 3: Test Hypothesis (5-15 minutes)

**For each hypothesis:**

**Test 1: Verify hypothesis directly**
- Add logging to confirm cause
- Run minimal reproduction case
- Observe result

**Test 2: Isolate variables**
- Remove dependencies (does it still happen?)
- Simplify data (does it happen with 1 record?)
- Change environment (dev vs. prod)

**Test 3: Confirm fix**
- Apply targeted fix
- Re-run reproduction steps
- Verify issue is resolved

**Document results:**
```
Hypothesis: [Description]

**Test Method:**
[How tested]

**Result:**
✅ Confirmed / ❌ Rejected

**Evidence:**
[What proved/disproved hypothesis]
```

### Step 4: Implement Fix (Variable time)

**If hypothesis confirmed:**
- Implement minimal fix (don't refactor everything)
- Add error handling or validation
- Add monitoring or logging to catch future instances

**If hypothesis rejected:**
- Move to next hypothesis
- Re-gather information (may have missed something)
- Consider broader causes (systemic issue, not local)

### Step 5: Verify & Document (5-10 minutes)

**Verify:**
- Original reproduction steps no longer fail
- Edge cases also work
- No regressions introduced (other features still work)

**Document:**
```markdown
## Fix: [Issue Title]

**Root Cause:**
[What actually caused issue]

**Solution:**
[What was fixed]

**Code Changes:**
[File, function, or module changed]

**Testing:**
- Original issue: ✅ Resolved
- Edge case 1: ✅ Tested
- Edge case 2: ✅ Tested
- Regression check: ✅ No regressions

**Lessons Learned:**
[What to do differently in future]
```

---

## Common Debugging Patterns

### Pattern 1: Read Logs Strategically

**Don't:** Read entire log file into context
**Do:**
- Search for error codes or keywords
- Extract log lines around error timestamp (±5 minutes)
- Look for patterns (same error recurring)

**Search commands:**
```bash
# Search for specific error
grep -i "error" app.log | tail -20

# Search around timestamp
grep "2026-02-02T14:00" app.log -A 5 -B 5

# Extract stack traces
grep -A 10 "Error:" app.log
```

### Pattern 2: Use Logging to Verify

**Add logs at critical points:**

| Point | Log What | Why |
|--------|-----------|-----|
| Before database query | Query string, parameters | See what's being executed |
| After database query | Rows returned, time taken | Performance check |
| Before API call | Request payload | Verify what's sent |
| After API call | Response status, body | Verify what's received |
| Before file write | File path, content | Verify what's written |

**When in doubt, add a log. You can always remove later.**

### Pattern 3: Binary Search for Root Cause

**When many variables, narrow down:**

```
Example: Database query slow (4s)

Variables:
- Database connection ✅
- Query complexity ✅
- Table size (100K rows) ✅
- Missing index ❓
- Lock contention ❓

Test 1: Run query with 10 rows
Result: 0.1s (fast) → Table size not cause

Test 2: Run EXPLAIN on query
Result: "seq scan" → Missing index confirmed

Hypothesis: Missing index causes full table scan
Fix: Add index on queried columns
Result: 0.05s → Confirmed
```

### Pattern 4: Reproduce in Isolated Environment

**When issue only happens in production:**
- Create minimal reproduction in dev
- Simplify data (use synthetic data)
- Remove external dependencies (mock APIs)
- Test against same environment versions

**Goal:** Isolate whether issue is data, code, or environment.

### Pattern 5: Use Version Control for Time Travel

**When something breaks after changes:**

```bash
# What changed recently?
git log --oneline -10

# When did it start failing?
git log --oneline --since="2026-02-02T13:00"

# Bisect to find breaking commit
git bisect start HEAD git bisect bad
git bisect good <last-working-commit>
# Git will test commits between, find exact one that broke it
```

---

## Troubleshooting Categories

### 1. Code Errors (Runtime, Compile, Type)

**Common causes:**
- Null/undefined access
- Async timing issues
- Type mismatches
- Module not found

**Approach:**
- Read error message carefully
- Check stack trace for file/line
- Search error code online
- Isolate the function causing error

### 2. Performance Issues

**Common causes:**
- N+1 queries (database missing index)
- Slow renders (unnecessary re-renders)
- Memory leaks (unclosed connections, retained references)
- Network latency (unoptimized requests, distant servers)

**Approach:**
- Profile the slow operation
- Identify bottleneck (database, CPU, I/O, network)
- Optimize the bottleneck, not everything
- Measure before/after to verify improvement

### 3. Data Inconsistency

**Common causes:**
- Database out of sync with files
- Cache invalidation issues
- Race conditions (concurrent writes)
- Migration not applied

**Approach:**
- Compare database state vs. expected state
- Check last sync timestamp
- Verify data integrity (constraints, foreign keys)
- Re-run sync or migration if needed

### 4. Integration Failures

**Common causes:**
- API contract mismatch (expected vs. actual)
- Authentication/authorization failures
- Network connectivity
- Service downtime

**Approach:**
- Verify request format matches API docs
- Check credentials and permissions
- Test API directly (curl or Postman)
- Check service status page

### 5. Environment-Specific Issues

**Common causes:**
- Environment variables missing or wrong
- Path differences (Windows vs. Mac vs. Linux)
- Dependency version conflicts
- File permissions

**Approach:**
- Compare working vs. broken environment
- Check environment variables
- Verify dependencies match versions
- Check file permissions and paths

---

## Quality Checklist

Before considering debugging complete, verify:

### Understanding
- [ ] Root cause identified (not just symptom fixed)
- [ ] Hypothesis tested with evidence
- [ ] Fix is minimal (don't over-engineer)

### Verification
- [ ] Original issue reproduced before fix
- [ ] Fix resolves issue (not just hides it)
- [ ] Edge cases tested
- [ ] No regressions (other features still work)

### Documentation
- [ ] Root cause documented
- [ ] Solution documented
- [ ] Lessons learned captured
- [ ] Logs or examples saved for future reference

---

## Common Pitfalls to Avoid

❌ **Shotgun debugging** — Change everything → ✅ One change, test, observe  
❌ **Ignoring error messages** — "Probably network" → ✅ Read exact error, search it  
❌ **Fixing without testing** — Apply change, assume it works → ✅ Reproduce, fix, re-test  
❌ **Assuming environment** — "Works on my machine" → ✅ Verify in actual environment  
❌ **Blaming external deps** — "Library is broken" → ✅ Verify your usage first  

---

## Example Usage

**Scenario:** AROMA sync fails with "ENOENT: file not found"

### Step 1: Gather Information
```
Error: "ENOENT: /path/to/file.md not found"
Context: Running npm run sync-to-markdown
Recent changes: Git pull merged new files
Environment: Dev, Node v25.5.0, SQLite v3.45
```

### Step 2: Formulate Hypothesis
```
Hypothesis 1 (70%): Sync script expects files that don't exist yet
Hypothesis 2 (30%): Git pull didn't update worktree correctly
```

### Step 3: Test Hypothesis 1
```
Test: Check if files exist in repository
Command: ls /path/to/file.md
Result: ❌ File does NOT exist

Conclusion: Hypothesis 1 confirmed
```

### Step 4: Implement Fix
```typescript
// Check if file exists before reading
if (fs.existsSync(filePath)) {
  const content = fs.readFileSync(filePath, 'utf-8');
  // ... process content
} else {
  console.log(`File not found: ${filePath}`);
  // Skip or create placeholder
}
```

### Step 5: Verify & Document
```
Test: Run sync again
Result: ✅ Sync completes without error
Edge cases: ✅ Works when files exist, ✅ Handles missing files gracefully
Regression: ✅ Other sync operations still work

Fix: File existence check added before read
Root cause: Sync script didn't handle missing files
```

---

## Integration with Other Skills

**Use with:**
- `workspace-navigation` - When debugging collaborative workspaces
- `repo-context-sync` - When debugging git integration issues
- `web-research` - When searching error codes or similar issues online

**Pattern:**
```
Error occurs → debugging(gather, hypothesize, test) → 
  Fix → Verify → Document
```

---

## Skill Metadata

**Token Efficiency:** ~8,000-15,000 tokens per debugging session (systematic approach, targeted testing)  
**Quality Impact:** Ensures bugs are fixed at root cause, not symptoms; prevents regressions  
**Maintenance:** Update when new debugging patterns emerge

**Related Skills:**
- `research-modes` - Deep investigation techniques
- `web-research` - Error code lookup and similar issues
- `specification-writer` - Documenting root causes and fixes

---

**Last Updated:** 2026-02-02  
**Maintained By:** Cipher 🧭  
**Status:** Active — Ready for implementation
