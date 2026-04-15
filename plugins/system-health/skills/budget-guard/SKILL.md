---
name: budget-guard
model: sonnet
description: "Produces a structured APPROVE / WARN / BLOCK decision by checking remaining token budget across query, session, and monthly tiers before an expensive operation runs. Use when: \"is this operation within budget\", \"pre-flight check before web search\", \"budget alert fired\", \"before a multi-step pipeline\", \"session above 70% utilization\"."
license: proprietary
category: system-health
triggers:
  - "is this operation within budget"
  - "pre-flight check before web search"
  - "budget alert fired"
  - "before a multi-step pipeline"
  - "session above 70% utilization"
tier: 1
agents:
  - "primary"
tool_dependencies:
  - "file_system"

inputs:
  - name: operation_description
    type: string
    description: Description of the operation to check against remaining budget
    required: true
  - name: estimated_tokens
    type: number
    description: Estimated token count for the operation
    required: false
outputs:
  - name: budget_decision
    type: string
    description: Structured APPROVE/WARN/BLOCK decision with remaining budget across query, session, and monthly tiers
---
## I. Philosophy

Token budgets are not bureaucratic limits — they are the financial physics of
agent operation. Ignoring them does not make the cost disappear; it makes the
cost invisible until it becomes a crisis. Budget Guard makes cost visible before
it is incurred, giving operators and users the information they need to make
deliberate decisions rather than discovering overruns after the fact.

The three-tier structure (query, session, monthly) mirrors how costs actually
accumulate: individual calls compound into sessions, sessions compound into
monthly spend. A healthy system enforces all three, not just the monthly cap.

## II. When to Use

Use this skill:

- Before any orchestration plan that includes web search, compute-heavy
  inference, file processing, or multi-step memory operations.
- When a user explicitly asks about remaining budget or cost of a planned
  action.
- When BudgetMiddleware response headers (`X-Budget-Remaining`,
  `X-Budget-Tier`) indicate the session is above 70% utilization.
- As a pre-flight check in any automated pipeline where token cost is variable
  and potentially unbounded.
- After an unexpected spike in usage to determine whether the monthly tier is
  at risk.

Do not use this skill as a post-hoc audit tool. It is a forward-looking gate,
not a ledger review. For historical cost analysis, use `agent-performance-report`.

## III. Workflow

**Step 1 — Receive proposed action and estimated cost.**

Accept two inputs from the caller:
- `action_description`: a plain-language description of the operation to be
  checked (e.g., "web search for competitive analysis across 10 URLs")
- `estimated_tokens`: the caller's raw token estimate before category
  adjustment (use `EstimatePlanCost(plan)` output if available from the
  orchestration engine)

If `estimated_tokens` is not provided, apply a conservative default based on
action type: web operations default to 2000 tokens, compute operations to 3000,
file operations to 1500, memory operations to 800.

**Step 2 — Query remaining budget across all three tiers.**

Call `BudgetTracker.GetRemaining(userID)` to retrieve:

```
query_remaining:   <tokens remaining in current query limit>
session_remaining: <tokens remaining in current session>
monthly_remaining: <tokens remaining this calendar month>
monthly_reset:     <ISO date of next monthly reset>
```

The binding limit is the minimum across all three tiers. A session approaching
its limit is just as blocking as a monthly cap.

**Step 3 — Apply category multiplier.**

Determine the operation category from `action_description` using keyword
matching:

| Category | Keywords | Multiplier |
|---|---|---|
| web | "search", "fetch", "browse", "crawl", "URL" | 1.4x |
| compute | "analyze", "summarize", "generate", "synthesize", "infer" | 1.5x |
| file | "read", "write", "parse", "process", "upload", "download" | 1.3x |
| memory | "store", "retrieve", "seed", "memory", "embed" | 1.2x |
| default | (no keyword match) | 1.0x |

If an operation spans multiple categories, apply the highest multiplier. Do not
stack multipliers.

Adjusted cost = `estimated_tokens * multiplier`

**Step 4 — Apply decision matrix.**

Calculate utilization for each tier as:

```
utilization = (tier_limit - tier_remaining) / tier_limit
```

For the binding tier (lowest remaining), apply:

| Utilization | Decision | Action |
|---|---|---|
| < 70% | APPROVE | Proceed. Log decision and adjusted cost. |
| 70% to 90% | WARN | Proceed with caution. Inform user of proximity to limit. |
| > 90% | BLOCK | Do not proceed. Require explicit user override. |

A BLOCK decision cannot be overridden automatically. It requires a human
confirmation step.

**Step 5 — Output decision.**

Return a structured decision to the caller:

```
Decision: APPROVE | WARN | BLOCK
Reason: <which tier is binding and why>

Adjusted cost estimate: <tokens> (base <raw_tokens> x <multiplier> for <category>)

Budget status:
  Query:   <remaining> remaining  [<utilization>% used]
  Session: <remaining> remaining  [<utilization>% used]
  Monthly: <remaining> remaining  [<utilization>% used] — resets <monthly_reset>

Binding tier: <query|session|monthly>
Recommendation: <brief human-readable guidance>
```

For WARN decisions, include the exact token count that would trigger a BLOCK if
the current operation proceeds.

For BLOCK decisions, include the token count required to resume and the
earliest reset date across all blocked tiers.

## IV. Best Practices

- Always check all three tiers. A system that only enforces monthly limits will
  allow runaway sessions to consume the month's budget in a single hour.
- Use `EstimatePlanCost(plan)` from the orchestration engine when available —
  it produces more accurate estimates than keyword-based defaults because it
  accounts for DAG structure and parallelism.
- When the category is ambiguous (e.g., "analyze a file"), apply the higher
  multiplier (compute at 1.5x rather than file at 1.3x). It is better to
  over-estimate and approve a cheaper operation than to under-estimate and
  approve an expensive one.
- Log every BLOCK decision with the full budget snapshot. This creates an audit
  trail that helps identify whether limits are calibrated correctly.
- Do not cache the `GetRemaining` result across multiple checks. Budget state
  changes between calls and a stale snapshot can lead to false approvals.
- The 90% threshold for BLOCK is a policy default, not a hard system constant.
  If an operator has configured a different threshold, respect their
  configuration over this default.

## V. Quality Checklist

Before completing this skill, verify:

- [ ] Both inputs received (action description and token estimate, or default
  applied with note)
- [ ] All three tiers queried — not just monthly
- [ ] Category multiplier applied and category logged in output
- [ ] Binding tier identified (minimum remaining across all tiers)
- [ ] Decision is one of APPROVE, WARN, or BLOCK — no ambiguous states
- [ ] BLOCK decisions do not auto-proceed — human override required
- [ ] Monthly reset date included in output for WARN and BLOCK decisions
- [ ] Full budget snapshot (all three tiers with utilization percentage)
  included in output

## Output
- Structured decision block: Decision (APPROVE / WARN / BLOCK), reason, adjusted cost estimate, per-tier budget status with utilization percentages, binding tier, and human-readable recommendation.
- For WARN: includes the exact token count that would trigger a BLOCK if the current operation proceeds.
- For BLOCK: includes token count required to resume and earliest reset date across blocked tiers.

## Examples
**Scenario 1:** "Check if a 10-URL web search is within budget" → APPROVE with adjusted cost (2000 tokens × 1.4x web multiplier = 2800), all three tiers shown at current utilization.
**Scenario 2:** "Session is at 88% — can I run this summarize task?" → WARN at session tier. Compute multiplier (1.5x) applied. Shows exactly how many tokens remain before BLOCK threshold.

## Edge Cases
- If estimated_tokens is not provided, apply conservative defaults by operation type (web 2000, compute 3000, file 1500, memory 800) and note the default was used.
- When an operation spans multiple categories, apply the highest multiplier — never stack multipliers.
- BLOCK decisions cannot auto-proceed regardless of downstream urgency; surface the decision and wait for human confirmation.

## Anti-Patterns
- Caching the GetRemaining result across multiple checks — budget state changes between calls and a stale snapshot can produce false approvals.
- Using this skill as a post-hoc ledger review — it is a forward-looking gate. For historical cost analysis, use `agent-performance-report`.
