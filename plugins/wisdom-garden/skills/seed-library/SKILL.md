# Copyright 2024 Tres Pies Design
# Licensed under the Apache License, Version 2.0

---
name: seed-library
description: Maintain curated library of wisdom seeds for reuse and reference
triggers:
  - "manage seed library"
  - "organize wisdom seeds"
  - "curate knowledge library"
metadata:
  version: "1.0"
  created: "2026-02-04"
  author: "Tres Pies Design"
  tool_dependencies:
    - script_execution
    - file_system
    - bash
  portable: true
  tier: 2
  agents:
    - research-agent
---

# Seed Module Library

Manage and apply the **10 Dojo Seed Patches** plus **3 field seeds** as reusable thinking modules. The core seeds derive from Dataiku research; the field seeds emerged from direct practice. Make enterprise-grade agent patterns accessible, consistent, and evolvable.

## When to Use This Skill

Trigger this skill when:
- User asks "What seed applies here?" or "Which pattern should I use?"
- Keywords match seed triggers (governance, multi-agent, cost, context, routing, etc.)
- Architecting new features or systems
- Debugging complex issues
- Optimizing costs or performance
- Building trust through transparency

## The 10 Core Seeds

1. **Three-Tiered Governance** - Strategic/Tactical/Operational framework
2. **Harness Trace** - Nested spans + events for traceability
3. **Context Iceberg** - 6x token multiplier, hierarchical loading
4. **Agent Connect** - Routing-first, not swarm-first
5. **Go-Live Bundles** - Reusable artifacts (DojoPacket)
6. **Cost Guard** - Budget for full iceberg (5-10x multiplier)
7. **Safety Switch** - Fallback to conservative mode
8. **Implicit Perspective Extraction** - Extract from constraints/metaphors
9. **Mode-Based Complexity Gating** - 3-question test
10. **Shared Infrastructure** - Build once, reuse everywhere

**Meta-Seed:** Governance Multiplies Velocity

### Field Seeds (From Practice)
11. **Voice Before Structure** - Read design language before writing structural artifacts
12. **Pointer Directories** - Empty directories are references, not gaps
13. **Granular Visibility** - Progress tracking serves the user, not the agent

## Core Workflow

### 1. Suggest Relevant Seeds

Based on keywords or task context:

```bash
python3.11 /home/ubuntu/skills/seed-library/scripts/suggest_seeds.py <keywords...>
```

**Example:**
```bash
python3.11 suggest_seeds.py multi-agent architecture coordination
```

**Output:**
- Top 3 relevant seeds ranked by relevance score
- Brief description of each seed
- File path to full seed content

**Saved to:** `/home/ubuntu/seed-suggestions.md`

### 2. Apply a Seed

Load full seed content and application guide:

```bash
python3.11 /home/ubuntu/skills/seed-library/scripts/apply_seed.py <seed_id> [session_id]
```

**Example:**
```bash
python3.11 apply_seed.py 04_agent_connect session_123
```

**Output:**
- Full seed content (What It Is, Why It Matters, The Pattern, etc.)
- Application checklist
- Next steps
- Usage tracked automatically

**Saved to:** `/home/ubuntu/seed-<seed_id>-applied.md`

## Script Reference

### suggest_seeds.py

**Purpose:** Suggest relevant seeds based on keywords

**Usage:**
```bash
python3.11 suggest_seeds.py <keywords...>
```

**How it works:**
1. Matches keywords to seed triggers
2. Calculates relevance score
3. Returns top 3 suggestions

**Trigger Keywords by Seed:**

- **01 (Governance):** governance, capabilities, complexity, multi-agent, coordination, policy, standards
- **02 (Trace):** debugging, trace, transparency, performance, evaluation, logging, monitoring
- **03 (Context):** token, cost, context, window, limit, budget, pruning, memory, overhead
- **04 (Agent Connect):** multi-agent, routing, coordination, specialized, handoff, permission, swarm
- **05 (Bundles):** export, sharing, reuse, artifact, package, bundle, repeatability, trust
- **06 (Cost Guard):** cost, budget, estimation, planning, infrastructure, investment, pricing
- **07 (Safety Switch):** fallback, conservative, alert, drift, failure, recovery, validation, error
- **08 (Perspective):** perspective, constraint, metaphor, scope, extraction, implicit, natural
- **09 (Complexity):** mode, complexity, routing, simple, query, reasoning, adaptive
- **10 (Infrastructure):** infrastructure, reuse, duplication, foundation, shared, common, service
- **11 (Voice):** voice, philosophy, design-language, manifest, description, ecosystem, grounding, plugin
- **12 (Pointers):** empty, missing, pointer, provenance, registry, audit, gap, coverage, directory
- **13 (Visibility):** progress, tracking, visibility, todo, granular, steering, trust, delegation

### apply_seed.py

**Purpose:** Load and explain how to apply a seed

**Usage:**
```bash
python3.11 apply_seed.py <seed_id> [session_id]
```

**What it does:**
1. Loads full seed content from `/seeds/<seed_id>.md`
2. Generates application guide with checklist
3. Tracks usage in `/home/ubuntu/.seed-usage.json`
4. Saves guide to file

**Seed IDs:**
- `01_three_tiered_governance`
- `02_harness_trace`
- `03_context_iceberg`
- `04_agent_connect`
- `05_go_live_bundles`
- `06_cost_guard`
- `07_safety_switch`
- `08_implicit_perspective_extraction`
- `09_mode_based_complexity_gating`
- `10_shared_infrastructure`
- `meta_governance_multiplies_velocity`
- `11_voice_before_structure`
- `12_pointer_directories`
- `13_granular_visibility`

## Seed Files

All seeds are stored in `/home/ubuntu/skills/seed-library/seeds/` as markdown files.

### Seed File Structure

```markdown
---
seed_id: 01
name: Three-Tiered Governance
version: 1.0
created: 2026-01-12
source: Dataiku Research
status: active
---

# Seed Name

## What It Is
{description}

## Why It Matters
{rationale}

## The Pattern
{pattern description}

## Revisit Trigger
{when to apply this seed}

## Dojo Application
{how Dojo uses this pattern}

## What It Refuses
{what this pattern explicitly rejects}

## Checks
- [ ] {validation check 1}
- [ ] {validation check 2}

## Related Seeds
- {seed name}: {relationship}
```

## Usage Tracking

The skill automatically tracks seed usage in `/home/ubuntu/.seed-usage.json`:

```json
{
  "seeds": {
    "04_agent_connect": {
      "usage_count": 3,
      "last_used": "2026-01-29T12:00:00Z",
      "sessions": ["session_123", "session_456"]
    }
  },
  "session_seeds": {
    "session_123": ["04_agent_connect", "10_shared_infrastructure"]
  }
}
```

## Reference Documents

### seed_catalog.md

Quick reference to all 10 seeds plus meta-seed:
- One-line descriptions
- Triggers and refusals
- Seed relationships
- Usage patterns (5 common scenarios)
- Maintenance guidelines

**Read this when:**
- Getting overview of all seeds
- Understanding seed relationships
- Learning usage patterns

## Common Use Cases

### Use Case 1: Architecting Multi-Agent System

**User says:** "How should we architect the multi-agent system?"

**Workflow:**
1. Run `suggest_seeds.py multi-agent architecture coordination`
2. Top suggestions: Agent Connect, Shared Infrastructure, Three-Tiered Governance
3. Apply Seed 04 (Agent Connect)
4. Follow application checklist
5. Track usage

**Output:**
> Based on your query, I recommend **Agent Connect** (Seed 04). This seed provides a routing-first pattern where the Supervisor acts as a single entry point, routing to specialized agents (Dojo, Librarian, Debugger, Builder) based on context. This prevents agent sprawl and keeps coordination manageable.

### Use Case 2: Optimizing Token Usage

**User says:** "Token usage is spiking, how do we optimize?"

**Workflow:**
1. Run `suggest_seeds.py token cost context optimization`
2. Top suggestions: Context Iceberg, Cost Guard, Mode-Based Complexity Gating
3. Apply Seed 03 (Context Iceberg)
4. Implement hierarchical context loading (Tier 1-4)
5. Set pruning triggers (80%, 90%, 95%)

**Output:**
> The **Context Iceberg** pattern (Seed 03) explains the 6x token multiplier from demo to production. Implement hierarchical context loading with 4 tiers, and set pruning triggers at 80% (prune Tier 4), 90% (prune Tier 3), and 95% (alert user).

### Use Case 3: Debugging Unexpected Behavior

**User says:** "Why did Dojo choose Scout instead of Mirror?"

**Workflow:**
1. Run `suggest_seeds.py debugging trace transparency`
2. Top suggestion: Harness Trace
3. Apply Seed 02 (Harness Trace)
4. Review trace spans for `mode_selection`
5. Read `metadata.reasoning`

**Output:**
> Use the **Harness Trace** (Seed 02) to inspect the decision path. Check the `mode_selection` span in the trace JSON. The `metadata.reasoning` field will explain why Scout was chosen over Mirror.

### Use Case 4: Building Trust with Users

**User says:** "Users don't understand our recommendations"

**Workflow:**
1. Run `suggest_seeds.py trust transparency trace`
2. Top suggestions: Harness Trace, Go-Live Bundles, Governance Multiplies Velocity
3. Apply Seed 02 (Harness Trace) for transparency
4. Apply Seed 05 (Go-Live Bundles) for repeatability
5. Apply Meta-Seed for philosophy

**Output:**
> Build trust through **Harness Trace** (transparency), **Go-Live Bundles** (repeatability), and the **Governance Multiplies Velocity** philosophy. Show users the decision path, make sessions exportable and inspectable, and emphasize that governance accelerates delivery.

### Use Case 5: Planning New Feature

**User says:** "Let's plan the Workbench feature"

**Workflow:**
1. Run `suggest_seeds.py planning feature governance`
2. Top suggestions: Three-Tiered Governance, Cost Guard, Shared Infrastructure
3. Apply Seed 01 (Three-Tiered Governance)
4. Define strategic (principles), tactical (standards), operational (tools)
5. Apply Seed 06 (Cost Guard) to budget for 5-10x multiplier

**Output:**
> Start with **Three-Tiered Governance** (Seed 01). Define strategic principles (does this align with "no autopilot"?), tactical standards (DojoPacket schema, routing logic), and operational tools (tests, traces). Then apply **Cost Guard** (Seed 06) to budget for the full iceberg, not just API costs.

## Integration with Dojo Agent

When Dojo is reasoning, it can reference seeds:

**Example:**
> "Considering **Agent Connect** (Seed 04) for this multi-agent architecture. Using Supervisor as router with transparent handoffs."

**Example:**
> "Applying **Context Iceberg** (Seed 03) to manage token usage. Loading Tier 1 + Tier 2 context for this query."

## Seed Relationships

### Foundational Seeds (Start Here)
- **01. Three-Tiered Governance** - Framework for all other seeds
- **Meta. Governance Multiplies Velocity** - Philosophy

### Operational Seeds (Day-to-Day)
- **02. Harness Trace** - Transparency and debugging
- **03. Context Iceberg** - Token and cost management
- **07. Safety Switch** - Error handling
- **09. Mode-Based Complexity Gating** - Query routing

### Architectural Seeds (System Design)
- **04. Agent Connect** - Multi-agent coordination
- **10. Shared Infrastructure** - Reusable services

### Delivery Seeds (Shipping)
- **05. Go-Live Bundles** - Packaging and reuse
- **06. Cost Guard** - Budgeting and planning

### UX Seeds (User Experience)
- **08. Implicit Perspective Extraction** - Reduce friction

### Field Seeds (From Practice)
- **11. Voice Before Structure** - Ground in philosophy before writing
- **12. Pointer Directories** - Understand provenance before filling gaps
- **13. Granular Visibility** - Track progress for the user, not the agent

## Seed Maintenance

### When to Update a Seed
- Pattern evolves based on new learnings
- Better approach discovered
- User feedback suggests improvement

### When to Archive a Seed
- Pattern no longer applies
- Superseded by better approach
- Context has fundamentally changed

### Versioning Convention
- **1.0** - Initial version from Dataiku research
- **1.1** - Minor update (clarification, example added)
- **2.0** - Major update (pattern changed)

### Update Process
1. Identify need for update
2. Draft new version
3. Test with real scenarios
4. Update seed file with "Revised:" section
5. Update seed_catalog.md
6. Notify users of change

## Best Practices

### Frequency
- **Proactive:** Suggest seeds during planning and design
- **Reactive:** Apply seeds when debugging or optimizing
- **Continuous:** Track usage to identify most valuable seeds

### Application
- **Read the full seed** before applying (don't just skim)
- **Follow the checklist** to validate application
- **Note what it refuses** to avoid anti-patterns
- **Track usage** to measure effectiveness

### Evolution
- **Update when patterns evolve** (not just for the sake of updating)
- **Archive when obsolete** (don't hoard outdated seeds)
- **Version carefully** (breaking changes = major version bump)

## Limitations

### What This Skill Does
- Suggest relevant seeds based on keywords
- Load and explain how to apply seeds
- Track usage over time
- Provide quick reference to all seeds

### What This Skill Doesn't Do
- Automatically apply seeds (you still need to implement)
- Make architectural decisions for you
- Guarantee success (seeds are patterns, not silver bullets)
- Replace deep thinking (seeds guide, they don't replace reasoning)

## Quick Reference

| Task | Command |
|------|---------|
| Suggest seeds | `python3.11 suggest_seeds.py <keywords...>` |
| Apply seed | `python3.11 apply_seed.py <seed_id>` |
| View catalog | `file read references/seed_catalog.md` |
| Check usage | `cat /home/ubuntu/.seed-usage.json` |
| List all seeds | `ls /home/ubuntu/skills/seed-library/seeds/` |

## Success Metrics

- **Accessibility:** Find relevant seed in < 10 seconds
- **Consistency:** Seeds applied correctly 90%+ of the time
- **Evolution:** Seeds updated when patterns improve
- **Reuse:** Seeds used across multiple sessions
- **Learning:** New seeds added as patterns emerge
